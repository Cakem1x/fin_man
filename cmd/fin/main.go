package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Cakem1x/fin_man/internal/importer/genericcsv"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]
	switch subcommand {
	case "import-csv":
		handleImportCSV(os.Args[2:])
	default:
		fmt.Printf("unknown subcommand: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: fin <subcommand> [args]")
	fmt.Println("Subcommands:")
	fmt.Println("  import-csv    Import transactions from a CSV file")
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
