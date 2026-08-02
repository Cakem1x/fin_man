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
	"github.com/Cakem1x/fin_man/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

var importCSVCmd = &cobra.Command{
	Use:   "import-csv <csv_file>",
	Short: "Import transactions from a CSV file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if configPath == "" {
			return fmt.Errorf("config path or builtin config name is required")
		}

		csvPath := args[0]
		cwd, _ := os.Getwd()
		wsDir, err := workspace.FindRoot(cwd)
		if err != nil {
			return fmt.Errorf("failed to find workspace: %w", err)
		}
		mgr := workspace.NewManager(wsDir)
		isOpen, err := mgr.IsOpen()
		if err != nil {
			return fmt.Errorf("failed to check workspace state: %w", err)
		}
		if !isOpen {
			fmt.Println("Workspace must be opened before importing transactions.")
			fmt.Println("Please run 'fin workspace open' to open it.")
			return fmt.Errorf("workspace not opened")
		}

		var cfg genericcsv.Config
		if builtinCfg, ok := genericcsv.GetBuiltinConfig(configPath); ok {
			cfg = builtinCfg
		} else {
			cfgData, err := os.ReadFile(configPath)
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

		destDir := filepath.Join(wsDir, "store", "archives", fmt.Sprintf("csv_%s", cfgName))
		destFilename := time.Now().Format("20060102150405_") + filepath.Base(csvPath)
		destPath := filepath.Join(destDir, destFilename)

		for i := range txs {
			txs[i].ArchiveFilePath = &destPath
		}

		dbPath := filepath.Join(wsDir, "store", "finance.db")
		sqliteDB, err := db.Open(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer func() {
			if err := sqliteDB.Close(); err != nil {
				log.Printf("failed to close database: %v", err)
			}
		}()

		ctx := context.Background()
		inserted, duplicates, err := sqliteDB.InsertTransactions(ctx, txs)

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
		if len(duplicates) > 0 {
			fmt.Printf("Skipped %d duplicate transactions:\n", len(duplicates))
			for _, pair := range duplicates {
				fmt.Printf("- Existing: %s | %s | %d %s | %s\n", pair.Existing.Date.Format("2006-01-02"), pair.Existing.Payee, pair.Existing.AmountCents, pair.Existing.Currency, pair.Existing.Memo)
				fmt.Printf("  New:      %s | %s | %d %s | %s\n", pair.New.Date.Format("2006-01-02"), pair.New.Payee, pair.New.AmountCents, pair.New.Currency, pair.New.Memo)
			}
			return fmt.Errorf("import finished with %d duplicate errors", len(duplicates))
		} else if err != nil {
			return fmt.Errorf("failed to insert transactions: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCSVCmd)
	available := genericcsv.AvailableConfigs()
	configDesc := fmt.Sprintf("Path to config JSON file OR builtin config name %v", available)
	importCSVCmd.Flags().StringVarP(&configPath, "config", "c", "", configDesc)
	// Archiving is now enabled by default.
}
