package categorize

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/Cakem1x/fin_man/internal/model"
)

// smoothingAlpha prevents scores from being unrealistically high when evidence
// is sparse. A small constant in the denominator keeps scores near zero when
// the total vote weight is negligible.
const smoothingAlpha = 0.1

// similarityExponent controls how aggressively low similarity scores are
// suppressed when computing vote weights (W = S^exponent). A value of 3 means
// a 0.2 similarity contributes only 0.008 weight while a 0.9 contributes 0.729.
const similarityExponent = 3.0

// cosineFloor is the cosine similarity that maps to an embedding similarity
// of 0. The rationale is in project_management/architecture/categorization.md.
const cosineFloor = 0.7

// CategoryEstimation holds the classification score for a single category.
type CategoryEstimation struct {
	CategoryName string
	CategoryID   string
	Score        float64 // 0.0 to 1.0
}

// TagEstimation holds the classification score for a single tag.
type TagEstimation struct {
	TagName string
	TagID   string
	Score   float64 // 0.0 to 1.0
}

// Estimation is the output of the distance-weighted k-NN classifier,
// containing ranked scores for every known category and tag.
type Estimation struct {
	CategoryScores []CategoryEstimation // sorted descending by Score
	TagScores      []TagEstimation      // sorted descending by Score
}

type trainedData struct {
	ID           string
	Tokens       map[string]struct{}
	CategoryID   string
	CategoryName string
	Tags         []model.Tag
}

// Estimator implements a distance-weighted k-NN classifier that learns from
// reviewed transactions and estimates category/tag scores for new ones.
type Estimator struct {
	memory          []trainedData
	knownCategories map[string]string    // name -> id
	knownTags       map[string]string    // name -> id
	embeddings      map[string][]float32 // transaction ID -> vector
}

// Option configures an Estimator.
type Option func(*Estimator)

// WithEmbeddings provides embedding vectors keyed by transaction ID.
func WithEmbeddings(vecs map[string][]float32) Option {
	return func(e *Estimator) {
		for id, v := range vecs {
			e.embeddings[id] = v
		}
	}
}

func NewEstimator(corpus []model.Transaction, opts ...Option) *Estimator {
	e := &Estimator{
		memory:          make([]trainedData, 0, len(corpus)),
		knownCategories: make(map[string]string),
		knownTags:       make(map[string]string),
		embeddings:      make(map[string][]float32),
	}
	for _, opt := range opts {
		opt(e)
	}
	for _, tx := range corpus {
		e.Learn(tx)
	}
	return e
}

// nonLetters matches numbers and punctuation.
var nonLetters = regexp.MustCompile(`[^a-z\s]+`)

// tokenize cleans the input string, removes non-alphabetical characters,
// and returns a set of unique lowercase words.
func tokenize(payee, memo string) map[string]struct{} {
	text := payee + " " + memo
	text = strings.ToLower(text)

	text = nonLetters.ReplaceAllString(text, " ")

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

// SetEmbedding sets the embedding vector of a transaction.
func (e *Estimator) SetEmbedding(txID string, vec []float32) {
	e.embeddings[txID] = vec
}

// embeddingSimilarity returns the cosine similarity of the transactions'
// vectors, linearly rescaled from [cosineFloor, 1] to [0, 1].
// It returns 0 if a vector is missing.
func (e *Estimator) embeddingSimilarity(idA, idB string) float64 {
	a, b := e.embeddings[idA], e.embeddings[idB]
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	cos := dot / math.Sqrt(normA*normB)
	return math.Max(0, (cos-cosineFloor)/(1-cosineFloor))
}

// Learn adds a newly reviewed transaction to the model's memory dynamically.
func (e *Estimator) Learn(tx model.Transaction) {
	// Only learn from transactions that actually have a category
	if tx.CategoryID == nil {
		return
	}

	catName := ""
	if tx.CategoryName != nil {
		catName = *tx.CategoryName
	}

	e.knownCategories[catName] = *tx.CategoryID

	for _, tag := range tx.Tags {
		e.knownTags[tag.Name] = tag.ID
	}

	e.memory = append(e.memory, trainedData{
		ID:           tx.ID,
		Tokens:       tokenize(tx.Payee, tx.Memo),
		CategoryID:   *tx.CategoryID,
		CategoryName: catName,
		Tags:         tx.Tags,
	})
}

// Estimate computes classification scores for all known categories and tags
// using distance-weighted k-NN voting.
//
// For categories (mutually exclusive):
//
//	Score(C) = W_pos(C) / (sum_all_categories(W_pos) + alpha)
//
// For tags (independent binary per tag):
//
//	Score(T) = W_pos(T) / (W_pos(T) + W_neg(T) + alpha)
//
// Vote weight W_i = S_i^3 where S_i is the larger of the Jaccard similarity
// and the embedding similarity.
func (e *Estimator) Estimate(tx model.Transaction) *Estimation {
	if len(e.memory) == 0 {
		return nil
	}

	txTokens := tokenize(tx.Payee, tx.Memo)

	// Accumulate weighted votes for categories and tags.
	catWeightPos := make(map[string]float64) // category name -> positive weight
	tagWeightPos := make(map[string]float64) // tag name -> positive weight
	tagWeightNeg := make(map[string]float64) // tag name -> negative weight

	for _, pastTx := range e.memory {
		sim := math.Max(jaccardSimilarity(txTokens, pastTx.Tokens), e.embeddingSimilarity(tx.ID, pastTx.ID))
		weight := math.Pow(sim, similarityExponent)

		// Category vote: this transaction votes for its own category.
		catWeightPos[pastTx.CategoryName] += weight

		// Tag votes: build a set of tags this past transaction has.
		pastTagSet := make(map[string]bool, len(pastTx.Tags))
		for _, tag := range pastTx.Tags {
			pastTagSet[tag.Name] = true
		}

		// Positive vote for tags this transaction has.
		for tagName := range pastTagSet {
			tagWeightPos[tagName] += weight
		}

		// Negative vote for known tags this transaction does NOT have.
		for knownTag := range e.knownTags {
			if !pastTagSet[knownTag] {
				tagWeightNeg[knownTag] += weight
			}
		}
	}

	// Calculate category scores.
	totalCatWeight := 0.0
	for _, w := range catWeightPos {
		totalCatWeight += w
	}

	var catScores []CategoryEstimation
	for catName, catID := range e.knownCategories {
		wPos := catWeightPos[catName]
		score := wPos / (totalCatWeight + smoothingAlpha)
		catScores = append(catScores, CategoryEstimation{
			CategoryName: catName,
			CategoryID:   catID,
			Score:        score,
		})
	}
	sort.Slice(catScores, func(i, j int) bool {
		if catScores[i].Score == catScores[j].Score {
			return catScores[i].CategoryName < catScores[j].CategoryName
		}
		return catScores[i].Score > catScores[j].Score
	})

	// Calculate tag scores.
	var tagScores []TagEstimation
	for tagName, tagID := range e.knownTags {
		wPos := tagWeightPos[tagName]
		wNeg := tagWeightNeg[tagName]
		score := wPos / (wPos + wNeg + smoothingAlpha)
		tagScores = append(tagScores, TagEstimation{
			TagName: tagName,
			TagID:   tagID,
			Score:   score,
		})
	}
	sort.Slice(tagScores, func(i, j int) bool {
		if tagScores[i].Score == tagScores[j].Score {
			return tagScores[i].TagName < tagScores[j].TagName
		}
		return tagScores[i].Score > tagScores[j].Score
	})

	return &Estimation{
		CategoryScores: catScores,
		TagScores:      tagScores,
	}
}
