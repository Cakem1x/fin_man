package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// ReceiptMatcher defines an interface for providing itemized data for a transaction.
type ReceiptMatcher interface {
	// Match returns a list of items and their prices (in cents) for a given transaction memo/ID.
	// The boolean indicates if a match was found at all.
	Match(memo string, date time.Time) ([]Item, bool, error)
}

// Item represents a single line item in a matched receipt.
type Item struct {
	Title       string
	AmountCents int64
}

// AmazonPrivacyCSVMatcher parses Amazon's Privacy Central order history CSV.
type AmazonPrivacyCSVMatcher struct {
	// Order ID -> Items
	orders map[string][]Item
}

// NewAmazonPrivacyCSVMatcher creates a matcher from a CSV file path.
func NewAmazonPrivacyCSVMatcher(filePath string) (*AmazonPrivacyCSVMatcher, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open amazon csv: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	m := &AmazonPrivacyCSVMatcher{
		orders: make(map[string][]Item),
	}

	reader := csv.NewReader(f)
	// The privacy CSV has many columns. We typically look for:
	// Order ID, Product Name, Unit Price, etc.
	// Since we don't have the exact header layout of the 2026 privacy CSV, we will
	// do a dynamic column index lookup based on expected headers.
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read headers: %w", err)
	}

	orderIDIdx := -1
	productNameIdx := -1
	unitPriceIdx := -1

	for i, h := range headers {
		hLower := strings.ToLower(strings.TrimSpace(h))
		if strings.Contains(hLower, "order id") {
			orderIDIdx = i
		} else if strings.Contains(hLower, "product name") || strings.Contains(hLower, "item title") || strings.Contains(hLower, "title") {
			productNameIdx = i
		} else if strings.Contains(hLower, "unit price") || strings.Contains(hLower, "item total") || strings.Contains(hLower, "price") {
			unitPriceIdx = i
		}
	}

	if orderIDIdx == -1 || productNameIdx == -1 {
		return nil, fmt.Errorf("could not find required columns in amazon csv headers")
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read row: %w", err)
		}

		if len(record) <= orderIDIdx || len(record) <= productNameIdx {
			continue
		}

		orderID := strings.TrimSpace(record[orderIDIdx])
		title := strings.TrimSpace(record[productNameIdx])
		if orderID == "" || title == "" {
			continue
		}

		var amountCents int64
		if unitPriceIdx != -1 && len(record) > unitPriceIdx {
			priceStr := strings.TrimSpace(record[unitPriceIdx])
			// Strip currency symbols and commas
			priceStr = strings.ReplaceAll(priceStr, "$", "")
			priceStr = strings.ReplaceAll(priceStr, "€", "")
			priceStr = strings.ReplaceAll(priceStr, "EUR", "")
			priceStr = strings.ReplaceAll(priceStr, "USD", "")
			// Some locales use comma as decimal, some use period. We'll naively convert comma to period
			priceStr = strings.ReplaceAll(priceStr, ",", ".")
			priceStr = strings.TrimSpace(priceStr)

			if priceStr != "" {
				// Parse float and convert to cents
				if fPrice, err := strconv.ParseFloat(priceStr, 64); err == nil {
					amountCents = int64(fPrice * 100)
				}
			}
		}

		m.orders[orderID] = append(m.orders[orderID], Item{
			Title:       title,
			AmountCents: amountCents,
		})
	}

	return m, nil
}

// Match looks for an Amazon order ID in the transaction memo.
func (m *AmazonPrivacyCSVMatcher) Match(memo string, _ time.Time) ([]Item, bool, error) {
	// Amazon order IDs are typically XXX-XXXXXXX-XXXXXXX
	// We can just iterate through our keys or use a regex to extract it from the memo.
	// The memo often contains it directly.
	for orderID, items := range m.orders {
		if strings.Contains(memo, orderID) {
			return items, true, nil
		}
	}
	return nil, false, nil
}
