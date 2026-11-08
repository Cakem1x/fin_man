package app

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/model"
)

// openTestDB opens a temporary database that contains txs.
func openTestDB(t *testing.T, txs []model.Transaction) *db.DB {
	t.Helper()
	ctx := context.Background()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.Migrate(database.DB); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	if _, _, err := database.InsertTransactions(ctx, txs); err != nil {
		t.Fatalf("InsertTransactions failed: %v", err)
	}
	return database
}

func newService(t *testing.T, database *db.DB) *Service {
	t.Helper()
	svc, err := New(context.Background(), database, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return svc
}

func day(d int) time.Time {
	return time.Date(2024, 1, d, 12, 0, 0, 0, time.UTC)
}

func findTx(t *testing.T, txs []model.Transaction, id string) model.Transaction {
	t.Helper()
	for _, tx := range txs {
		if tx.ID == id {
			return tx
		}
	}
	t.Fatalf("transaction %q not found", id)
	return model.Transaction{}
}

func tagNames(tx model.Transaction) []string {
	names := make([]string, len(tx.Tags))
	for i, tg := range tx.Tags {
		names[i] = tg.Name
	}
	slices.Sort(names)
	return names
}

func TestApplyReviewUpdatesTransactionsCategoriesAndTags(t *testing.T) {
	ctx := context.Background()
	database := openTestDB(t, []model.Transaction{
		{ID: "coffee", Date: day(1), Payee: "Coffee Shop", AmountCents: -450, Currency: "EUR"},
	})
	if err := database.EnrichTransaction(ctx, "coffee", "Food", []string{"old"}, "", false); err != nil {
		t.Fatalf("EnrichTransaction failed: %v", err)
	}
	svc := newService(t, database)

	if err := svc.ApplyReview(ctx, "coffee", "Food/Coffee", []string{"daily", "work"}, "latte", true); err != nil {
		t.Fatalf("ApplyReview failed: %v", err)
	}

	tx := findTx(t, svc.Transactions(), "coffee")
	if tx.CategoryName == nil || *tx.CategoryName != "Food/Coffee" {
		t.Errorf("category = %v, want Food/Coffee", tx.CategoryName)
	}
	if got := tagNames(tx); !slices.Equal(got, []string{"daily", "work"}) {
		t.Errorf("tags = %v, want [daily work]", got)
	}
	if tx.Memo != "latte" {
		t.Errorf("memo = %q, want latte", tx.Memo)
	}
	if !tx.IsReviewed {
		t.Error("transaction is not reviewed")
	}
	if !slices.Contains(svc.Categories(), "Food/Coffee") {
		t.Errorf("categories %v lack Food/Coffee", svc.Categories())
	}
	for _, tag := range []string{"daily", "work"} {
		if !slices.Contains(svc.Tags(), tag) {
			t.Errorf("tags %v lack %s", svc.Tags(), tag)
		}
	}
}

func TestApplyReviewReviewedTrainsSuggest(t *testing.T) {
	ctx := context.Background()
	database := openTestDB(t, []model.Transaction{
		{ID: "rewe", Date: day(1), Payee: "REWE Markt", AmountCents: -2000, Currency: "EUR"},
		{ID: "coffee", Date: day(2), Payee: "Bean Bar Espresso", AmountCents: -450, Currency: "EUR"},
	})
	if err := database.EnrichTransaction(ctx, "rewe", "Groceries", nil, "", true); err != nil {
		t.Fatalf("EnrichTransaction failed: %v", err)
	}
	svc := newService(t, database)

	if err := svc.ApplyReview(ctx, "coffee", "Coffee", nil, "", true); err != nil {
		t.Fatalf("ApplyReview failed: %v", err)
	}

	similar := model.Transaction{ID: "new", Date: day(3), Payee: "Bean Bar Espresso", AmountCents: -380, Currency: "EUR"}
	est := svc.Suggest(similar)
	if est == nil || len(est.CategoryScores) == 0 {
		t.Fatal("Suggest returned no category scores")
	}
	if got := est.CategoryScores[0].CategoryName; got != "Coffee" {
		t.Errorf("top category = %q, want Coffee", got)
	}
}

func TestSplitReplacesParentWithChildren(t *testing.T) {
	ctx := context.Background()
	database := openTestDB(t, []model.Transaction{
		{ID: "amazon", Date: day(1), Payee: "Amazon", AmountCents: -3000, Currency: "EUR"},
	})
	svc := newService(t, database)

	children := []model.Transaction{
		{ID: "amazon-a", Date: day(1), Payee: "Amazon", AmountCents: -1000, Currency: "EUR"},
		{ID: "amazon-b", Date: day(1), Payee: "Amazon", AmountCents: -2000, Currency: "EUR"},
	}
	if err := svc.Split(ctx, "amazon", children); err != nil {
		t.Fatalf("Split failed: %v", err)
	}

	var got []string
	for _, tx := range svc.Transactions() {
		got = append(got, tx.ID)
	}
	slices.Sort(got)
	if want := []string{"amazon-a", "amazon-b"}; !slices.Equal(got, want) {
		t.Errorf("transactions = %v, want %v", got, want)
	}
}
