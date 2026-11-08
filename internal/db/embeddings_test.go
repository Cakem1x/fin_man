package db

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Cakem1x/fin_man/internal/model"
)

func TestUpsertAndGetEmbeddings(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = database.Close() }()
	if err := Migrate(database.DB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	ctx := context.Background()
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, _, err := database.InsertTransactions(ctx, []model.Transaction{
		{ID: "a", Date: date, Payee: "REWE", Currency: "EUR"},
		{ID: "b", Date: date, Payee: "EDEKA", Currency: "EUR"},
	}); err != nil {
		t.Fatalf("InsertTransactions: %v", err)
	}

	if err := database.UpsertEmbeddings(ctx, "m1", map[string]model.Embedding{
		"a": {TextHash: "h1", Vector: []float32{0.25, -1.5, 3}},
		"b": {TextHash: "h2", Vector: []float32{1}},
	}); err != nil {
		t.Fatalf("UpsertEmbeddings: %v", err)
	}
	if err := database.UpsertEmbeddings(ctx, "m1", map[string]model.Embedding{
		"a": {TextHash: "h3", Vector: []float32{2, 4}},
	}); err != nil {
		t.Fatalf("UpsertEmbeddings overwrite: %v", err)
	}
	if err := database.UpsertEmbeddings(ctx, "m2", map[string]model.Embedding{
		"a": {TextHash: "other", Vector: []float32{9}},
	}); err != nil {
		t.Fatalf("UpsertEmbeddings other model: %v", err)
	}

	got, err := database.GetEmbeddings(ctx, "m1")
	if err != nil {
		t.Fatalf("GetEmbeddings: %v", err)
	}
	want := map[string]model.Embedding{
		"a": {TextHash: "h3", Vector: []float32{2, 4}},
		"b": {TextHash: "h2", Vector: []float32{1}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetEmbeddings = %v, want %v", got, want)
	}
}
