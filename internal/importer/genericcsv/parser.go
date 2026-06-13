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
	HasHeader          bool
	SkipLines          int
	DateCol            int
	DateFormat         string
	PayeeCol           int
	AmountCol          int
	CurrencyCol        int
	MemoCol            int
	Comma              rune
	DecimalSeparator   string
	ThousandsSeparator string
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
	// DKB and others sometimes have "wrong" number of fields in meta lines
	reader.FieldsPerRecord = -1

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
		if line <= imp.config.SkipLines {
			continue
		}

		if line == imp.config.SkipLines+1 && imp.config.HasHeader {
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

	// Parse Amount
	amountStr := getCol(imp.config.AmountCol)
	amountCents, err := parseAmountToCents(amountStr, imp.config.DecimalSeparator, imp.config.ThousandsSeparator)
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

func parseAmountToCents(s, decimalSep, thousandSep string) (int64, error) {
	if s == "" {
		return 0, nil
	}

	cleanStr := strings.TrimSpace(s)

	if thousandSep != "" {
		cleanStr = strings.ReplaceAll(cleanStr, thousandSep, "")
	}

	if decimalSep != "" && decimalSep != "." {
		cleanStr = strings.ReplaceAll(cleanStr, decimalSep, ".")
	}

	// Remove common generic separators if nothing specified (for backwards compatibility)
	if decimalSep == "" && thousandSep == "" {
		cleanStr = strings.ReplaceAll(cleanStr, ",", ".")
	}

	f, err := strconv.ParseFloat(cleanStr, 64)
	if err != nil {
		return 0, err
	}
	return int64(f * 100), nil
}
