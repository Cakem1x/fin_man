package categorize

import (
	"regexp"
	"strings"

	"github.com/Cakem1x/fin_man/internal/model"
)

type Estimation struct {
	CategoryName string
	CategoryID   string
	Confidence   float64 // 0.0 to 1.0
}

type Estimator struct {
	Corpus []model.Transaction
}

func NewEstimator(corpus []model.Transaction) *Estimator {
	return &Estimator{Corpus: corpus}
}

// tokenize cleans the input string, removes non-alphabetical characters,
// and returns a set of unique lowercase words.
func tokenize(payee, memo string) map[string]struct{} {
	text := payee + " " + memo
	text = strings.ToLower(text)

	// Strip out numbers and punctuation
	reg := regexp.MustCompile(`[^a-z\s]+`)
	text = reg.ReplaceAllString(text, " ")

	words := strings.Fields(text)
	tokens := make(map[string]struct{})
	for _, w := range words {
		tokens[w] = struct{}{}
	}
	return tokens
}

// jaccardSimilarity calculates |A ∩ B| / |A ∪ B|
func jaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0.0
	}
	intersection := 0
	for token := range a {
		if _, exists := b[token]; exists {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

func (e *Estimator) Estimate(tx model.Transaction) *Estimation {
	txTokens := tokenize(tx.Payee, tx.Memo)

	var bestCategory string
	var bestCategoryID string
	var maxScore = 0.0

	for _, pastTx := range e.Corpus {
		pastTokens := tokenize(pastTx.Payee, pastTx.Memo)
		score := jaccardSimilarity(txTokens, pastTokens)

		if score > maxScore {
			maxScore = score
			if pastTx.CategoryName != nil {
				bestCategory = *pastTx.CategoryName
			}
			if pastTx.CategoryID != nil {
				bestCategoryID = *pastTx.CategoryID
			}
		}
	}

	if maxScore > 0 {
		return &Estimation{
			CategoryName: bestCategory,
			CategoryID:   bestCategoryID,
			Confidence:   maxScore,
		}
	}

	return nil
}
