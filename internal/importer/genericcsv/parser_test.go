package genericcsv_test

import (
	"encoding/json"
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Cakem1x/fin_man/internal/importer/genericcsv"
	"github.com/Cakem1x/fin_man/internal/model"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden(t *testing.T) {
	// Look for all .csv files in testdata/dkb
	testDataDir := filepath.Join("testdata", "dkb")
	entries, err := os.ReadDir(testDataDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Skip("testdata not found")
		}
		t.Fatal(err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".csv") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			csvPath := filepath.Join(testDataDir, entry.Name())
			goldenPath := csvPath + ".golden.json"

			// We need a way to tell the importer what the config is for this specific file.
			// For now, let's assume a default config or look for a .config.json
			cfg := genericcsv.Config{
				HasHeader:  true,
				DateCol:    0,
				DateFormat: "2006-01-02", // placeholder
				PayeeCol:   1,
				AmountCol:  2,
				Comma:      ',',
			}

			// Try to load custom config if exists
			configPath := csvPath + ".config.json"
			if data, err := os.ReadFile(configPath); err == nil {
				if err := json.Unmarshal(data, &cfg); err != nil {
					t.Fatalf("failed to unmarshal config: %v", err)
				}
			}

			f, err := os.Open(csvPath)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()

			imp := genericcsv.New(cfg)
			gotTrans, err := imp.Import(f)
			if err != nil {
				t.Fatalf("import failed: %v", err)
			}

			// If golden file doesn't exist, create it (bootstrap)
			if _, err := os.Stat(goldenPath); errors.Is(err, fs.ErrNotExist) {
				writeGolden(t, goldenPath, gotTrans)
				return
			}

			wantJSON, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatal(err)
			}

			var wantTrans []model.Transaction
			if err := json.Unmarshal(wantJSON, &wantTrans); err != nil {
				t.Fatalf("failed to unmarshal golden file: %v", err)
			}

			if *update {
				writeGolden(t, goldenPath, gotTrans)
			} else if !reflect.DeepEqual(gotTrans, wantTrans) {
				t.Errorf("output mismatch for %s. Run with -update to update golden files.", entry.Name())
			}
		})
	}
}

func writeGolden(t *testing.T, path string, trans []model.Transaction) {
	data, err := json.MarshalIndent(trans, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	// Append newline to satisfy linter
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote golden file: %s", path)
}
