package db

import (
	"context"
	"testing"

	"github.com/Cakem1x/fin_man/internal/model"
)

func insertEnrichFixture(t *testing.T, database *DB, ids ...string) {
	t.Helper()
	var txs []model.Transaction
	for i, id := range ids {
		txs = append(txs, model.Transaction{ID: id, Date: day(i + 1), Payee: id, AmountCents: -100, Currency: "EUR"})
	}
	if _, _, err := database.InsertTransactions(context.Background(), txs); err != nil {
		t.Fatalf("InsertTransactions failed: %v", err)
	}
}

func TestEnrichTransaction_NewCategoryIDDiffersFromName(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()
	insertEnrichFixture(t, database, "a")

	if err := database.EnrichTransaction(ctx, "a", "Food", nil, "", false); err != nil {
		t.Fatalf("EnrichTransaction failed: %v", err)
	}

	var id string
	if err := database.QueryRowContext(ctx, "SELECT id FROM categories WHERE name = 'Food'").Scan(&id); err != nil {
		t.Fatalf("category not found: %v", err)
	}
	if id == "Food" {
		t.Errorf("category ID equals its name %q", id)
	}
}

func TestEnrichTransaction_SameNewTagSharesID(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()
	insertEnrichFixture(t, database, "a", "b")

	for _, id := range []string{"a", "b"} {
		if err := database.EnrichTransaction(ctx, id, "", []string{"daily"}, "", false); err != nil {
			t.Fatalf("EnrichTransaction(%s) failed: %v", id, err)
		}
	}

	txs, err := database.ListTransactions(ctx, Filter{})
	if err != nil {
		t.Fatalf("ListTransactions failed: %v", err)
	}
	if len(txs) != 2 || len(txs[0].Tags) != 1 || len(txs[1].Tags) != 1 {
		t.Fatalf("expected two transactions with one tag each, got %+v", txs)
	}
	if txs[0].Tags[0].ID != txs[1].Tags[0].ID {
		t.Errorf("tag IDs differ: %q vs %q", txs[0].Tags[0].ID, txs[1].Tags[0].ID)
	}
	if txs[0].Tags[0].ID == "daily" {
		t.Errorf("tag ID equals its name %q", txs[0].Tags[0].ID)
	}
}

func TestEnrichTransaction_NewCategoryAfterRenameGetsOwnID(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()
	insertEnrichFixture(t, database, "a", "b")

	if err := database.EnrichTransaction(ctx, "a", "Food", nil, "", false); err != nil {
		t.Fatalf("EnrichTransaction(a) failed: %v", err)
	}
	if err := database.RenameCategory(ctx, "Food", "Groceries"); err != nil {
		t.Fatalf("RenameCategory failed: %v", err)
	}
	if err := database.EnrichTransaction(ctx, "b", "Food", nil, "", false); err != nil {
		t.Fatalf("EnrichTransaction(b) failed: %v", err)
	}

	var foodID, groceriesID string
	if err := database.QueryRowContext(ctx, "SELECT id FROM categories WHERE name = 'Food'").Scan(&foodID); err != nil {
		t.Fatalf("category Food not found: %v", err)
	}
	if err := database.QueryRowContext(ctx, "SELECT id FROM categories WHERE name = 'Groceries'").Scan(&groceriesID); err != nil {
		t.Fatalf("category Groceries not found: %v", err)
	}
	if foodID == groceriesID {
		t.Errorf("Food and Groceries share ID %q", foodID)
	}

	var aCategoryID string
	if err := database.QueryRowContext(ctx, "SELECT category_id FROM transactions WHERE id = 'a'").Scan(&aCategoryID); err != nil {
		t.Fatalf("transaction a not found: %v", err)
	}
	if aCategoryID != groceriesID {
		t.Errorf("transaction a has category ID %q, want Groceries ID %q", aCategoryID, groceriesID)
	}
}
