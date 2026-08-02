package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/dedup"
	"github.com/Cakem1x/fin_man/internal/importer/genericcsv"
	"github.com/Cakem1x/fin_man/internal/prompt"
	"github.com/Cakem1x/fin_man/internal/workspace"
)

func main() {
	log.SetFlags(0) // Remove timestamps from log output for a cleaner CLI UX
	var workDir string
	flag.StringVar(&workDir, "C", ".", "Run as if fin was started in <path> instead of the current working directory")
	flag.Parse()

	if flag.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	if workDir != "." {
		if err := os.Chdir(workDir); err != nil {
			log.Fatalf("failed to change directory: %v", err)
		}
	}

	subcommand := flag.Arg(0)
	subargs := flag.Args()[1:]

	switch subcommand {
	case "import-csv":
		handleImportCSV(subargs)
	case "workspace":
		handleWorkspace(subargs)
	default:
		fmt.Printf("unknown subcommand: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: fin [-C <path>] <subcommand> [args]")
	fmt.Println("\nGlobal Flags:")
	flag.PrintDefaults()
	fmt.Println("\nSubcommands:")
	fmt.Println("  import-csv    Import transactions from a CSV file")
	fmt.Println("  workspace     Manage encrypted workspace (init, open, close, upgrade)")
}

func handleImportCSV(args []string) {
	fs := flag.NewFlagSet("import-csv", flag.ExitOnError)
	available := genericcsv.AvailableConfigs()
	configDesc := fmt.Sprintf("Path to config JSON file OR builtin config name %v", available)
	configPath := fs.String("config", "", configDesc)
	fs.StringVar(configPath, "c", "", configDesc)
	archiveData := fs.Bool("archive", false, "Archive the input CSV file into the workspace")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	if *configPath == "" || fs.NArg() < 1 {
		fmt.Println("Usage: fin import-csv -config <cfg_or_name> [-archive] <csv_file>")
		fs.PrintDefaults()
		os.Exit(1)
	}

	cwd, _ := os.Getwd()
	wsDir, err := workspace.FindRoot(cwd)
	if err != nil {
		log.Fatalf("failed to find workspace: %v", err)
	}
	mgr := workspace.NewManager(wsDir)
	isOpen, err := mgr.IsOpen()
	if err != nil {
		log.Fatalf("failed to check workspace state: %v", err)
	}
	if !isOpen {
		fmt.Println("Workspace must be opened before importing transactions.")
		fmt.Println("Please run 'fin workspace open' to open it.")
		os.Exit(1)
	}

	csvPath := fs.Arg(0)

	var cfg genericcsv.Config
	// Check if it's a builtin config
	if builtinCfg, ok := genericcsv.GetBuiltinConfig(*configPath); ok {
		cfg = builtinCfg
	} else {
		// Load config from file
		cfgData, err := os.ReadFile(*configPath)
		if err != nil {
			log.Fatalf("failed to read config file: %v", err)
		}
		if err := json.Unmarshal(cfgData, &cfg); err != nil {
			log.Fatalf("failed to unmarshal config: %v", err)
		}
	}

	// Open CSV
	f, err := os.Open(csvPath)
	if err != nil {
		log.Fatalf("failed to open csv: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close file: %v", err)
		}
	}()

	// Parse
	imp := genericcsv.New(cfg)
	txs, err := imp.Import(f)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	// Generate deterministic IDs for deduplication
	for i := range txs {
		txs[i].ID = dedup.GenerateHash(txs[i])
	}

	var archivedPath string
	if *archiveData {
		cfgName := *configPath
		if filepath.Ext(cfgName) == ".json" {
			cfgName = strings.TrimSuffix(filepath.Base(cfgName), ".json")
		}

		destDir := filepath.Join(wsDir, "store", "archives", fmt.Sprintf("csv_%s", cfgName))
		if err := os.MkdirAll(destDir, 0755); err != nil {
			log.Fatalf("failed to create archive directory: %v", err)
		}

		destFilename := time.Now().Format("20060102150405_") + filepath.Base(csvPath)
		destPath := filepath.Join(destDir, destFilename)

		srcFile, err := os.Open(csvPath)
		if err != nil {
			log.Fatalf("failed to open input file for archiving: %v", err)
		}
		defer func() {
			if err := srcFile.Close(); err != nil {
				log.Printf("failed to close src file: %v", err)
			}
		}()

		dstFile, err := os.Create(destPath)
		if err != nil {
			log.Fatalf("failed to create archive file: %v", err)
		}
		defer func() {
			if err := dstFile.Close(); err != nil {
				log.Printf("failed to close dst file: %v", err)
			}
		}()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			log.Fatalf("failed to copy to archive: %v", err)
		}
		archivedPath = destPath
	}

	if archivedPath != "" {
		for i := range txs {
			txs[i].ArchiveFilePath = &archivedPath
		}
	}

	dbPath := filepath.Join(wsDir, "store", "finance.db")
	sqliteDB, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		if err := sqliteDB.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	ctx := context.Background()
	inserted, duplicates, err := sqliteDB.InsertTransactions(ctx, txs)

	fmt.Printf("Imported %d new transactions\n", inserted)
	if len(duplicates) > 0 {
		fmt.Printf("Skipped %d duplicate transactions:\n", len(duplicates))
		for _, pair := range duplicates {
			fmt.Printf("- Existing: %s | %s | %d %s | %s\n", pair.Existing.Date.Format("2006-01-02"), pair.Existing.Payee, pair.Existing.AmountCents, pair.Existing.Currency, pair.Existing.Memo)
			fmt.Printf("  New:      %s | %s | %d %s | %s\n", pair.New.Date.Format("2006-01-02"), pair.New.Payee, pair.New.AmountCents, pair.New.Currency, pair.New.Memo)
		}
		log.Fatalf("import finished with duplicate errors: %v", err)
	} else if err != nil {
		log.Fatalf("failed to insert transactions: %v", err)
	}
}

func handleWorkspace(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: fin workspace <init|open|close|upgrade> [options]")
		os.Exit(1)
	}

	sub := args[0]
	fs := flag.NewFlagSet("workspace "+sub, flag.ExitOnError)

	var storePath string
	if sub == "init" {
		fs.StringVar(&storePath, "store", "", "Path to the encrypted store (required for init)")
	}
	if err := fs.Parse(args[1:]); err != nil {
		log.Fatalf("failed to parse workspace flags: %v", err)
	}

	cwd, _ := os.Getwd()
	var wsDir string
	if sub == "init" {
		wsDir = cwd
	} else {
		var err error
		wsDir, err = workspace.FindRoot(cwd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}

	mgr := workspace.NewManager(wsDir)

	switch sub {
	case "init":
		if err := mgr.CanInit(); err != nil {
			log.Fatalf("Init failed: %v", err)
		}

		if storePath == "" {
			fmt.Println("Error: -store flag is required for init")
			os.Exit(1)
		}

		exists, err := mgr.IsStoreInitialized(storePath)
		if err != nil {
			log.Fatalf("Failed to check store: %v", err)
		}

		var pwd string
		if exists {
			fmt.Printf("Initializing workspace from existing store at %s\n", storePath)
			var err error
			pwd, err = prompt.ReadPassword("Enter workspace password")
			if err != nil {
				log.Fatalf("Password prompt failed: %v", err)
			}
		} else {
			fmt.Printf("Initializing workspace and creating new store at %s\n", storePath)
			var err error
			pwd, err = prompt.ReadNewPassword("Enter new workspace password", "Confirm new workspace password")
			if err != nil {
				log.Fatalf("Password prompt failed: %v", err)
			}
		}

		if err := mgr.Init(storePath, pwd); err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		fmt.Printf("Workspace configured at %s\n", cwd)

		if err := mgr.Open(pwd); err != nil {
			log.Fatalf("Failed to open workspace: %v", err)
		}
		fmt.Printf("Workspace mounted at %s/store\n", cwd)

		dbPath := filepath.Join(cwd, "store", "finance.db")
		dbExists := true
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			dbExists = false
		}

		sqliteDB, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}

		if !dbExists {
			if err := db.Migrate(sqliteDB); err != nil {
				log.Fatalf("Database creation failed: %v", err)
			}
			fmt.Println("New database created successfully.")
		} else {
			if needsUpgrade, _ := db.NeedsUpgrade(sqliteDB); needsUpgrade {
				doMigrate, err := prompt.Confirm("Database is out of date. Trigger migration?")
				if err != nil {
					log.Fatalf("Prompt failed: %v", err)
				}
				if doMigrate {
					if err := db.Migrate(sqliteDB); err != nil {
						log.Fatalf("Migration failed: %v", err)
					}
					fmt.Println("Workspace database upgraded successfully.")
				} else {
					fmt.Println("Notice: Workspace database is out of date and not usable before running 'fin workspace upgrade'")
				}
			}
		}
		if err := sqliteDB.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	case "open":
		isOpen, err := mgr.IsOpen()
		if err != nil {
			log.Fatalf("Failed to check workspace state: %v", err)
		}
		if isOpen {
			fmt.Println("Workspace is already open.")
			os.Exit(0)
		}

		pwd, err := prompt.ReadPassword("Enter workspace password")
		if err != nil {
			log.Fatalf("Password prompt failed: %v", err)
		}
		if err := mgr.Open(pwd); err != nil {
			log.Fatalf("Open failed: %v", err)
		}
		fmt.Printf("Workspace mounted at %s/store\n", cwd)

		// Check if DB needs upgrade
		dbPath := filepath.Join(wsDir, "store", "finance.db")
		if sqliteDB, err := sql.Open("sqlite3", dbPath); err == nil {
			if needsUpgrade, _ := db.NeedsUpgrade(sqliteDB); needsUpgrade {
				fmt.Println("\nNotice: Your workspace database is outdated!")
				fmt.Println("Please run 'fin workspace upgrade' to update it.")
			}
			if err := sqliteDB.Close(); err != nil {
				log.Printf("failed to close database: %v", err)
			}
		}
	case "close":
		isOpen, err := mgr.IsOpen()
		if err != nil {
			log.Fatalf("Failed to check workspace state: %v", err)
		}
		if !isOpen {
			fmt.Println("Workspace is already closed.")
			os.Exit(0)
		}

		if err := mgr.Close(); err != nil {
			log.Fatalf("Close failed: %v", err)
		}
		fmt.Printf("Workspace %s/store closed successfully.\n", cwd)
	case "upgrade":
		isOpen, err := mgr.IsOpen()
		if err != nil {
			log.Fatalf("Failed to check workspace state: %v", err)
		}
		if !isOpen {
			fmt.Println("Workspace must be opened before upgrading the database.")
			os.Exit(1)
		}
		dbPath := filepath.Join(wsDir, "store", "finance.db")
		sqliteDB, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer func() {
			if err := sqliteDB.Close(); err != nil {
				log.Printf("failed to close database: %v", err)
			}
		}()
		if err := db.Migrate(sqliteDB); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Workspace database upgraded successfully.")
	default:
		fmt.Printf("Unknown workspace subcommand: %s\n", sub)
		os.Exit(1)
	}
}
