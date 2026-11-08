package dedup

import (
	"testing"
	"time"

	"github.com/Cakem1x/fin_man/internal/model"
)

func TestGenerateHash(t *testing.T) {
	date := time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC)

	tx1 := model.Transaction{
		Date:        date,
		Payee:       "Amazon",
		AmountCents: 1599,
		Currency:    "EUR",
		Memo:        "Order #123-456",
	}

	tx2 := model.Transaction{
		Date:        date,
		Payee:       "AMAZON",
		AmountCents: 1599,
		Currency:    "EUR",
		Memo:        "Order #123-456", // same text
	}

	tx3 := model.Transaction{
		Date:        date,
		Payee:       "Amazon",
		AmountCents: 1599,
		Currency:    "EUR",
		Memo:        "Order 123456", // punctuation stripped, so should be same
	}

	tx4 := model.Transaction{
		Date:        date,
		Payee:       "Amazon",
		AmountCents: 1599,
		Currency:    "EUR",
		Memo:        "ORDER 1 23 456", // different case and spaces, should be same
	}

	txDifferentAmount := model.Transaction{
		Date:        date,
		Payee:       "Amazon",
		AmountCents: 1600,
		Currency:    "EUR",
		Memo:        "Order #123-456",
	}

	hash1 := GenerateHash(tx1)
	hash2 := GenerateHash(tx2)
	hash3 := GenerateHash(tx3)
	hash4 := GenerateHash(tx4)
	hashDiff := GenerateHash(txDifferentAmount)

	if hash1 != hash2 {
		t.Errorf("Expected case differences in payee to yield same hash, got %s and %s", hash1, hash2)
	}

	if hash1 != hash3 {
		t.Errorf("Expected punctuation differences in memo to yield same hash, got %s and %s", hash1, hash3)
	}

	if hash1 != hash4 {
		t.Errorf("Expected space/case differences in memo to yield same hash, got %s and %s", hash1, hash4)
	}

	if hash1 == hashDiff {
		t.Errorf("Expected different amounts to yield different hashes")
	}

	if len(hash1) != 64 {
		t.Errorf("Expected hash to be 64 characters long (SHA-256 hex string), got %d", len(hash1))
	}
}

func TestSplitIdentical(t *testing.T) {
	date := time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC)
	coffee := model.Transaction{ID: "a", Date: date, Payee: "Cafe", AmountCents: -300, Currency: "EUR"}
	bread := model.Transaction{ID: "b", Date: date, Payee: "Bakery", AmountCents: -250, Currency: "EUR"}
	tea := model.Transaction{ID: "c", Date: date, Payee: "Tea", AmountCents: -200, Currency: "EUR"}

	unique, identical := SplitIdentical([]model.Transaction{coffee, bread, coffee, tea, coffee})

	wantUnique := []string{"a", "b", "c"}
	if len(unique) != len(wantUnique) {
		t.Fatalf("expected %d unique rows, got %d", len(wantUnique), len(unique))
	}
	for i, id := range wantUnique {
		if unique[i].ID != id {
			t.Errorf("unique[%d]: expected ID %s, got %s", i, id, unique[i].ID)
		}
	}

	if len(identical) != 1 {
		t.Fatalf("expected 1 group of identical rows, got %d", len(identical))
	}
	if len(identical[0]) != 3 {
		t.Errorf("expected 3 identical rows, got %d", len(identical[0]))
	}
	for _, tx := range identical[0] {
		if tx.ID != "a" {
			t.Errorf("expected identical row ID a, got %s", tx.ID)
		}
	}
}

func TestSplitIdentical_NoIdentical(t *testing.T) {
	txs := []model.Transaction{{ID: "a"}, {ID: "b"}}
	unique, identical := SplitIdentical(txs)
	if len(unique) != 2 {
		t.Errorf("expected 2 unique rows, got %d", len(unique))
	}
	if len(identical) != 0 {
		t.Errorf("expected no identical rows, got %d groups", len(identical))
	}
}
