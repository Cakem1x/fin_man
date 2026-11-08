package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/dedup"
	"github.com/Cakem1x/fin_man/internal/importer/genericcsv"
	"github.com/Cakem1x/fin_man/internal/model"
	"github.com/Cakem1x/fin_man/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	configPath    string
	importVerbose bool
	importStrict  bool
)

var importCSVCmd = &cobra.Command{
	Use:   "import-csv <csv_file>",
	Short: "Import transactions from a CSV file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if configPath == "" {
			return fmt.Errorf("config path or builtin config name is required")
		}

		csvPath := resolvePath(args[0])
		mgr, err := findWorkspace(workDir)
		if err != nil {
			return err
		}
		sqliteDB, closeDB, err := openWorkspaceDB(mgr, true)
		if err != nil {
			return err
		}
		defer closeDB()

		var cfg genericcsv.Config
		if builtinCfg, ok := genericcsv.GetBuiltinConfig(configPath); ok {
			cfg = builtinCfg
		} else {
			cfgData, err := os.ReadFile(resolvePath(configPath))
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}
			if err := json.Unmarshal(cfgData, &cfg); err != nil {
				return fmt.Errorf("failed to unmarshal config: %w", err)
			}
		}

		f, err := os.Open(csvPath)
		if err != nil {
			return fmt.Errorf("failed to open csv: %w", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("failed to close file: %v", err)
			}
		}()

		imp := genericcsv.New(cfg)
		txs, err := imp.Import(f)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}

		for i := range txs {
			txs[i].ID = dedup.GenerateHash(txs[i])
		}

		cfgName := configPath
		if filepath.Ext(cfgName) == ".json" {
			cfgName = strings.TrimSuffix(filepath.Base(cfgName), ".json")
		}

		destDir := filepath.Join(mgr.WorkspaceDir, "store", "archives", fmt.Sprintf("csv_%s", cfgName))
		destFilename := time.Now().Format("20060102150405_") + filepath.Base(csvPath)
		destPath := filepath.Join(destDir, destFilename)

		for i := range txs {
			txs[i].ArchiveFilePath = &destPath
		}

		txs, identical := dedup.SplitIdentical(txs)

		ctx := context.Background()
		inserted, duplicates, err := sqliteDB.InsertTransactions(ctx, txs)
		if err != nil {
			return fmt.Errorf("failed to insert transactions: %w", err)
		}

		if inserted > 0 {
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return fmt.Errorf("failed to create archive directory: %w", err)
			}

			srcFile, err := os.Open(csvPath)
			if err != nil {
				return fmt.Errorf("failed to open input file for archiving: %w", err)
			}
			defer func() {
				if err := srcFile.Close(); err != nil {
					log.Printf("failed to close src file: %v", err)
				}
			}()

			dstFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create archive file: %w", err)
			}
			defer func() {
				if err := dstFile.Close(); err != nil {
					log.Printf("failed to close dst file: %v", err)
				}
			}()

			if _, err := io.Copy(dstFile, srcFile); err != nil {
				return fmt.Errorf("failed to copy to archive: %w", err)
			}
		}

		fmt.Printf("Imported %d new transactions\n", inserted)
		if inserted > 0 {
			embedAfterImport(ctx, mgr, sqliteDB)
		}
		printDuplicates(duplicates, identical)
		if importStrict && (len(duplicates) > 0 || len(identical) > 0) {
			return fmt.Errorf("--strict: import found duplicates")
		}

		return nil
	},
}

// printDuplicates reports duplicates of existing rows as info and identical
// rows within the file as a warning.
func printDuplicates(duplicates []db.DuplicatePair, identical [][]model.Transaction) {
	if len(duplicates) > 0 {
		fmt.Printf("Skipped %d transactions that already exist\n", len(duplicates))
		if importVerbose {
			for _, pair := range duplicates {
				fmt.Printf("- Existing: %s\n", formatTx(pair.Existing))
				fmt.Printf("  New:      %s\n", formatTx(pair.New))
			}
		}
	}
	for _, group := range identical {
		fmt.Fprintf(os.Stderr, "Warning: %d identical rows in file, stored only once: %s\n", len(group), formatTx(group[0]))
	}
}

func formatTx(tx model.Transaction) string {
	return fmt.Sprintf("%s | %s | %d %s | %s", tx.Date.Format("2006-01-02"), tx.Payee, tx.AmountCents, tx.Currency, tx.Memo)
}

// embedAfterImport runs embedTransactions if enabled in fin.toml.
// Failures only print a warning, because embeddings are optional.
func embedAfterImport(ctx context.Context, mgr *workspace.Manager, dbConn *db.DB) {
	cfg, err := mgr.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: skipped embedding, failed to read fin.toml: %v\n", err)
		return
	}
	if !cfg.LLM.Enabled {
		return
	}
	if err := embedTransactions(ctx, cfg.LLM, dbConn); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: embedding failed, run 'fin classify' later: %v\n", err)
	}
}

func init() {
	rootCmd.AddCommand(importCSVCmd)
	available := genericcsv.AvailableConfigs()
	configDesc := fmt.Sprintf("Path to config JSON file OR builtin config name %v", available)
	importCSVCmd.Flags().StringVarP(&configPath, "config", "c", "", configDesc)
	importCSVCmd.Flags().BoolVarP(&importVerbose, "verbose", "v", false, "List each skipped duplicate")
	importCSVCmd.Flags().BoolVar(&importStrict, "strict", false, "Exit 1 when any duplicate occurs")
}
