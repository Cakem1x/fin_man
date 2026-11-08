// Package app holds the application logic that the TUI uses. The TUI does
// not access the database directly.
package app

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Cakem1x/fin_man/internal/categorize"
	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/model"
)

// Service owns the database connection, the estimator and a cache of all
// transactions, categories and tags. Every mutation refreshes the cache.
type Service struct {
	db           *db.DB
	embeddings   map[string][]float32
	estimator    *categorize.Estimator
	transactions []model.Transaction
	categories   []string
	tags         []string
}

// New loads the cache and trains the estimator on all categorized
// transactions. embeddings holds cached transaction vectors for the
// estimator and may be nil.
func New(ctx context.Context, dbConn *db.DB, embeddings map[string][]float32) (*Service, error) {
	s := &Service{db: dbConn, embeddings: embeddings}
	if err := s.trainEstimator(ctx); err != nil {
		return nil, err
	}
	if err := s.refresh(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// Transactions returns all transactions except split parents, ordered by date.
func (s *Service) Transactions() []model.Transaction {
	return slices.Clone(s.transactions)
}

// Categories returns all category paths, sorted by name.
func (s *Service) Categories() []string {
	return slices.Clone(s.categories)
}

// Tags returns all tag names, sorted by name.
func (s *Service) Tags() []string {
	return slices.Clone(s.tags)
}

// Suggest returns category and tag scores for tx. It returns nil if the
// estimator has not learned any transaction yet.
func (s *Service) Suggest(tx model.Transaction) *categorize.Estimation {
	return s.estimator.Estimate(tx)
}

// ApplyReview sets the category, tags, note and review state of a
// transaction. It creates missing categories and tags. If reviewed is true,
// the estimator learns from the transaction.
func (s *Service) ApplyReview(ctx context.Context, txID, category string, tags []string, note string, reviewed bool) error {
	if err := s.db.EnrichTransaction(ctx, txID, category, tags, note, reviewed); err != nil {
		return err
	}
	if err := s.refresh(ctx); err != nil {
		return err
	}
	if reviewed {
		if i := slices.IndexFunc(s.transactions, func(tx model.Transaction) bool { return tx.ID == txID }); i >= 0 {
			s.estimator.Learn(s.transactions[i])
		}
	}
	return nil
}

// Split replaces a transaction with children.
func (s *Service) Split(ctx context.Context, txID string, children []model.Transaction) error {
	if err := s.db.SplitTransaction(ctx, txID, children); err != nil {
		return err
	}
	return s.refresh(ctx)
}

// RenameCategory renames the last path segment of oldPath to newName,
// including all nested categories. It returns the new path.
func (s *Service) RenameCategory(ctx context.Context, oldPath, newName string) (string, error) {
	if err := s.db.RenameCategory(ctx, oldPath, newName); err != nil {
		return "", err
	}
	if err := s.trainEstimator(ctx); err != nil {
		return "", err
	}
	if err := s.refresh(ctx); err != nil {
		return "", err
	}
	return newPath(oldPath, newName), nil
}

// DeleteCategory deletes a category without transactions and nested
// categories.
func (s *Service) DeleteCategory(ctx context.Context, path string) error {
	if err := s.db.DeleteEmptyCategory(ctx, path); err != nil {
		return err
	}
	return s.refresh(ctx)
}

func newPath(oldPath, newName string) string {
	return oldPath[:strings.LastIndex(oldPath, "/")+1] + strings.TrimSpace(newName)
}

// trainEstimator builds a new estimator from all categorized transactions.
func (s *Service) trainEstimator(ctx context.Context) error {
	corpus, err := s.db.ListTransactions(ctx, db.Filter{ExcludeUncategorized: true})
	if err != nil {
		return fmt.Errorf("failed to load categorized transactions: %w", err)
	}
	s.estimator = categorize.NewEstimator(corpus, categorize.WithEmbeddings(s.embeddings))
	return nil
}

// refresh reloads transactions, categories and tags from the database.
func (s *Service) refresh(ctx context.Context) error {
	txs, err := s.db.ListTransactions(ctx, db.Filter{})
	if err != nil {
		return fmt.Errorf("failed to load transactions: %w", err)
	}
	cats, err := s.db.GetAllCategories(ctx)
	if err != nil {
		return fmt.Errorf("failed to load categories: %w", err)
	}
	tags, err := s.db.GetAllTags(ctx)
	if err != nil {
		return fmt.Errorf("failed to load tags: %w", err)
	}
	s.transactions = txs
	s.categories = make([]string, len(cats))
	for i, c := range cats {
		s.categories[i] = c.Name
	}
	s.tags = make([]string, len(tags))
	for i, t := range tags {
		s.tags[i] = t.Name
	}
	return nil
}
