package db

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Cakem1x/fin_man/internal/model"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	database, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := Migrate(database.DB); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	return database
}

func day(d int) time.Time {
	return time.Date(2024, 1, d, 12, 0, 0, 0, time.UTC)
}

// seedListFixture inserts transactions that each filter test narrows down:
//
//	coffee  day 1  Food/Coffee  reviewed
//	bakery  day 2  Food         not reviewed
//	rent    day 3  Housing      reviewed
//	misc    day 4  uncategorized
//	cinema  day 5  Foodstuff    (shares a prefix with Food, but is not nested)
//	split   day 6  parent of split-a and split-b
func seedListFixture(t *testing.T, database *DB) {
	t.Helper()
	ctx := context.Background()
	txs := []model.Transaction{
		{ID: "coffee", Date: day(1), Payee: "Coffee Shop", AmountCents: -450, Currency: "EUR"},
		{ID: "bakery", Date: day(2), Payee: "Bakery", AmountCents: -320, Currency: "EUR"},
		{ID: "rent", Date: day(3), Payee: "Landlord", AmountCents: -90000, Currency: "EUR"},
		{ID: "misc", Date: day(4), Payee: "Unknown", AmountCents: -100, Currency: "EUR"},
		{ID: "cinema", Date: day(5), Payee: "Cinema", AmountCents: -1200, Currency: "EUR"},
		{ID: "split", Date: day(6), Payee: "Amazon", AmountCents: -3000, Currency: "EUR"},
	}
	if _, _, err := database.InsertTransactions(ctx, txs); err != nil {
		t.Fatalf("InsertTransactions failed: %v", err)
	}
	enrich := []struct {
		id, category string
		tags         []string
		reviewed     bool
	}{
		{"coffee", "Food/Coffee", []string{"daily"}, true},
		{"bakery", "Food", []string{"daily", "weekend"}, false},
		{"rent", "Housing", nil, true},
		{"cinema", "Foodstuff", nil, false},
		{"split", "", []string{"online"}, false},
	}
	for _, e := range enrich {
		if err := database.EnrichTransaction(ctx, e.id, e.category, e.tags, "", e.reviewed); err != nil {
			t.Fatalf("EnrichTransaction(%s) failed: %v", e.id, err)
		}
	}
	children := []model.Transaction{
		{ID: "split-a", Date: day(6), Payee: "Amazon", AmountCents: -1000, Currency: "EUR"},
		{ID: "split-b", Date: day(6), Payee: "Amazon", AmountCents: -2000, Currency: "EUR"},
	}
	if err := database.SplitTransaction(ctx, "split", children); err != nil {
		t.Fatalf("SplitTransaction failed: %v", err)
	}
}

func ids(txs []model.Transaction) []string {
	out := make([]string, len(txs))
	for i, tx := range txs {
		out[i] = tx.ID
	}
	return out
}

func TestListTransactions_Filter(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name   string
		filter Filter
		want   []string
	}{
		{"empty filter hides split parents", Filter{}, []string{"coffee", "bakery", "rent", "misc", "cinema", "split-a", "split-b"}},
		{"from is inclusive", Filter{From: day(5)}, []string{"cinema", "split-a", "split-b"}},
		{"to is exclusive", Filter{To: day(3)}, []string{"coffee", "bakery"}},
		{"from and to", Filter{From: day(2), To: day(4)}, []string{"bakery", "rent"}},
		{"category includes nested categories", Filter{Category: "Food"}, []string{"coffee", "bakery"}},
		{"nested category", Filter{Category: "Food/Coffee"}, []string{"coffee"}},
		{"category is case-sensitive", Filter{Category: "food"}, nil},
		{"reviewed", Filter{Reviewed: &yes}, []string{"coffee", "rent"}},
		{"not reviewed", Filter{Reviewed: &no}, []string{"bakery", "misc", "cinema", "split-a", "split-b"}},
		{"payee substring ignores case", Filter{PayeeContains: "shop"}, []string{"coffee"}},
		{"payee substring treats wildcards literally", Filter{PayeeContains: "%"}, nil},
		{"exclude uncategorized", Filter{ExcludeUncategorized: true}, []string{"coffee", "bakery", "rent", "cinema"}},
		{"combined fields", Filter{Category: "Food", Reviewed: &no}, []string{"bakery"}},
	}

	database := openTestDB(t)
	seedListFixture(t, database)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txs, err := database.ListTransactions(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("ListTransactions failed: %v", err)
			}
			if got := ids(txs); !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListTransactions_HydratesFields(t *testing.T) {
	database := openTestDB(t)
	seedListFixture(t, database)

	txs, err := database.ListTransactions(context.Background(), Filter{PayeeContains: "Bakery"})
	if err != nil {
		t.Fatalf("ListTransactions failed: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %v", ids(txs))
	}
	tx := txs[0]
	if tx.CategoryName == nil || *tx.CategoryName != "Food" {
		t.Errorf("expected category Food, got %v", tx.CategoryName)
	}
	var tagNames []string
	for _, tag := range tx.Tags {
		tagNames = append(tagNames, tag.Name)
	}
	slices.Sort(tagNames)
	if !slices.Equal(tagNames, []string{"daily", "weekend"}) {
		t.Errorf("expected tags [daily weekend], got %v", tagNames)
	}
}

func TestLoadTags_OnlyRequestedTransactions(t *testing.T) {
	database := openTestDB(t)
	seedListFixture(t, database)

	tags, err := database.loadTags(context.Background(), []string{"bakery", "rent"})
	if err != nil {
		t.Fatalf("loadTags failed: %v", err)
	}
	if len(tags) != 1 || len(tags["bakery"]) != 2 {
		t.Errorf("expected only the 2 tags of bakery, got %v", tags)
	}
}

func TestRenameCategory_Nested(t *testing.T) {
	database := openTestDB(t)
	seedListFixture(t, database)
	ctx := context.Background()
	if err := database.EnrichTransaction(ctx, "misc", "food/Other", nil, "", false); err != nil {
		t.Fatalf("EnrichTransaction failed: %v", err)
	}

	if err := database.RenameCategory(ctx, "Food", "Groceries"); err != nil {
		t.Fatalf("RenameCategory failed: %v", err)
	}
	cats, err := database.GetAllCategories(ctx)
	if err != nil {
		t.Fatalf("GetAllCategories failed: %v", err)
	}
	var names []string
	for _, c := range cats {
		names = append(names, c.Name)
	}
	want := []string{"Foodstuff", "Groceries", "Groceries/Coffee", "Housing", "food/Other"}
	if !slices.Equal(names, want) {
		t.Errorf("got %v, want %v", names, want)
	}
}

func TestDeleteEmptyCategory_RejectsNested(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `INSERT INTO categories (id, name) VALUES ('a', 'Food'), ('b', 'Food/Coffee'), ('c', 'Foodstuff')`); err != nil {
		t.Fatalf("insert categories failed: %v", err)
	}

	if err := database.DeleteEmptyCategory(ctx, "Food"); err == nil {
		t.Error("expected error for category with nested categories")
	}
	if err := database.DeleteEmptyCategory(ctx, "Foodstuff"); err != nil {
		t.Errorf("expected Foodstuff to be deleted, got %v", err)
	}
}
