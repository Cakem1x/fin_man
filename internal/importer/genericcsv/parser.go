package genericcsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Cakem1x/fin_man/internal/model"
	"github.com/Cakem1x/fin_man/internal/normalize"
)

// Config defines the mapping from CSV columns to Transaction fields.
type Config struct {
	HasHeader   bool
	DateCol     int
	DateFormat  string
	PayeeCol    int
	AmountCol   int
	CurrencyCol int
	MemoCol     int
	Comma       rune
}

// Importer implements the TransactionImporter interface for generic CSV files.
type Importer struct {
	config Config
}

func New(cfg Config) *Importer {
	return &Importer{config: cfg}
}

func (imp *Importer) Import(r io.Reader) ([]model.Transaction, error) {
	reader := csv.NewReader(r)
	reader.Comma = imp.config.Comma
	if imp.config.Comma == 0 {
		reader.Comma = ','
	}

	var results []model.Transaction
	line := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading csv: %w", err)
		}

		line++
		if line == 1 && imp.config.HasHeader {
			continue
		}

		tx, err := imp.mapRecord(record)
		if err != nil {
			return nil, fmt.Errorf("mapping record on line %d: %w", line, err)
		}

		results = append(results, tx)
	}

	return results, nil
}

func (imp *Importer) mapRecord(record []string) (model.Transaction, error) {
	// Utility to get column safely
	getCol := func(idx int) string {
		if idx < 0 || idx >= len(record) {
			return ""
		}
		return record[idx]
	}

	// Parse Date
	dateStr := getCol(imp.config.DateCol)
	date, err := time.Parse(imp.config.DateFormat, dateStr)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("parsing date %q: %w", dateStr, err)
	}

	// Parse Amount (assuming simple decimal string for now, will need more robust parsing later)
	amountStr := getCol(imp.config.AmountCol)
	// Remove common currency separators if any (e.g. "," -> "." etc) - this is a simplification for now
	amountCents, err := parseAmountToCents(amountStr)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("parsing amount %q: %w", amountStr, err)
	}

	payee := normalize.TitleCasePayee(getCol(imp.config.PayeeCol))

	return model.Transaction{
		Date:        date,
		Payee:       payee,
		AmountCents: amountCents,
		Currency:    getCol(imp.config.CurrencyCol),
		Memo:        getCol(imp.config.MemoCol),
	}, nil
}

func parseAmountToCents(s string) (int64, error) {
	// Very basic implementation: remove dots/commas and parse
	// In reality we should handle locales properly.
	// For this generic one, let's assume standard float-like string.
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil {
		return 0, err
	}
	return int64(f * 100), nil
}
