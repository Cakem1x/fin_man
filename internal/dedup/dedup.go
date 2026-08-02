package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	"github.com/Cakem1x/fin_man/internal/model"
)

// GenerateHash creates a stable deterministic SHA-256 hash for a transaction.
// This hash acts as the unique identifier for deduplication.
func GenerateHash(tx model.Transaction) string {
	dateStr := tx.Date.Format("2006-01-02")

	// Payee is already whitespace-cleaned and title-cased by the importer,
	// but we lowercase it here to guarantee case-insensitive hashing.
	payee := strings.ToLower(tx.Payee)

	// Strip all non-alphanumeric characters from the memo and lowercase it
	var memoBuilder strings.Builder
	for _, r := range tx.Memo {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			memoBuilder.WriteRune(unicode.ToLower(r))
		}
	}
	memo := memoBuilder.String()

	// Create a stable string representation
	// Format: Date|Payee|AmountCents|Currency|Memo
	payload := fmt.Sprintf("%s|%s|%d|%s|%s", dateStr, payee, tx.AmountCents, tx.Currency, memo)

	hasher := sha256.New()
	hasher.Write([]byte(payload))
	return hex.EncodeToString(hasher.Sum(nil))
}
