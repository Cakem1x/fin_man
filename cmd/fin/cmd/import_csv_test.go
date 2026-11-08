package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/model"
)

// fakeInserter records calls and inserts a fixed number of transactions.
type fakeInserter struct {
	calls    int
	inserted int
}

func (f *fakeInserter) InsertTransactions(_ context.Context, _ []model.Transaction) (int, []db.DuplicatePair, error) {
	f.calls++
	return f.inserted, nil, nil
}

func writeSourceCSV(t *testing.T) string {
	t.Helper()
	srcPath := filepath.Join(t.TempDir(), "input.csv")
	if err := os.WriteFile(srcPath, []byte("a,b\n"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}
	return srcPath
}

func TestArchiveAndInsert_CopyFails(t *testing.T) {
	srcPath := writeSourceCSV(t)
	// A regular file in place of the archive directory makes the copy fail.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, 0644); err != nil {
		t.Fatalf("failed to write blocker file: %v", err)
	}
	destPath := filepath.Join(blocker, "archive.csv")
	inserter := &fakeInserter{inserted: 1}

	_, _, err := archiveAndInsert(context.Background(), inserter, srcPath, destPath, []model.Transaction{{ID: "a"}})
	if err == nil {
		t.Fatal("expected error when copy fails")
	}
	if inserter.calls != 0 {
		t.Errorf("InsertTransactions called %d times, want 0", inserter.calls)
	}
}

func TestArchiveAndInsert_NothingNew(t *testing.T) {
	srcPath := writeSourceCSV(t)
	destPath := filepath.Join(t.TempDir(), "archives", "archive.csv")
	inserter := &fakeInserter{inserted: 0}

	inserted, _, err := archiveAndInsert(context.Background(), inserter, srcPath, destPath, []model.Transaction{{ID: "a"}})
	if err != nil {
		t.Fatalf("archiveAndInsert failed: %v", err)
	}
	if inserted != 0 {
		t.Errorf("inserted = %d, want 0", inserted)
	}
	if _, err := os.Stat(destPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("archive file exists after inserting nothing, stat err: %v", err)
	}
}
