package genericcsv_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cakem1x/fin_man/internal/importer/genericcsv"
)

func TestGolden(t *testing.T) {
	// Look for all .csv files in testdata/dkb
	testDataDir := filepath.Join("testdata", "dkb")
	entries, err := os.ReadDir(testDataDir)
	if err != nil {
		if os.IsNotExist(err) {
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
			defer f.Close()

			imp := genericcsv.New(cfg)
			transactions, err := imp.Import(f)
			if err != nil {
				t.Fatalf("import failed: %v", err)
			}

			gotJSON, err := json.MarshalIndent(transactions, "", "  ")
			if err != nil {
				t.Fatal(err)
			}

			// If golden file doesn't exist, create it (bootstrap)
			if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
				if err := os.WriteFile(goldenPath, gotJSON, 0644); err != nil {
					t.Fatal(err)
				}
				t.Logf("created golden file: %s", goldenPath)
				return
			}

			wantJSON, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatal(err)
			}

			if string(gotJSON) != string(wantJSON) {
				t.Errorf("output mismatch for %s. Run with -update to update golden files.", entry.Name())
				// Simplified: you'd usually have a flag to update
			}
		})
	}
}
