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
	testDataDir := "testdata"
	bankEntries, err := os.ReadDir(testDataDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Skip("testdata not found")
		}
		t.Fatal(err)
	}

	for _, bankEntry := range bankEntries {
		if !bankEntry.IsDir() {
			continue
		}

		bankName := bankEntry.Name()
		bankDir := filepath.Join(testDataDir, bankName)

		entries, err := os.ReadDir(bankDir)
		if err != nil {
			t.Fatalf("failed reading dir %s: %v", bankDir, err)
		}

		defaultCfg, defaultCfgFound := genericcsv.GetBuiltinConfig(bankName)

		t.Run(bankName, func(t *testing.T) {
			for _, entry := range entries {
				if !strings.HasSuffix(entry.Name(), ".csv") {
					continue
				}

				t.Run(entry.Name(), func(t *testing.T) {
					csvPath := filepath.Join(bankDir, entry.Name())
					goldenPath := csvPath + ".golden.json"

					cfg := defaultCfg
					configPath := csvPath + ".config.json"
					if data, err := os.ReadFile(configPath); err == nil {
						if err := json.Unmarshal(data, &cfg); err != nil {
							t.Fatalf("failed to unmarshal config %s: %v", configPath, err)
						}
					} else if !defaultCfgFound {
						t.Fatalf("no builtin config for bank %q and no custom config %s found", bankName, configPath)
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
						t.Errorf("output mismatch for %s/%s. Run with -update to update golden files.", bankName, entry.Name())
					}
				})
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
