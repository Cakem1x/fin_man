package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Cakem1x/fin_man/internal/db"
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

	if err := fs.Parse(args); err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	if *configPath == "" || fs.NArg() < 1 {
		fmt.Println("Usage: fin import-csv -config <cfg_or_name> <csv_file>")
		fs.PrintDefaults()
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

	// Output as JSON
	out, err := json.MarshalIndent(txs, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal results: %v", err)
	}

	fmt.Println(string(out))
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
