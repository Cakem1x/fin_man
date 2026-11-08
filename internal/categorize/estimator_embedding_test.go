package categorize

import (
	"math"
	"reflect"
	"testing"

	"github.com/Cakem1x/fin_man/internal/model"
)

func embeddingCorpus() []model.Transaction {
	groceries, fuel := "Groceries", "Fuel"
	return []model.Transaction{
		{ID: "rewe", Payee: "REWE Markt", CategoryID: &groceries, CategoryName: &groceries,
			Tags: []model.Tag{{ID: "food", Name: "food"}}},
		{ID: "shell", Payee: "Shell Tankstelle", Memo: "Markt", CategoryID: &fuel, CategoryName: &fuel,
			Tags: []model.Tag{{ID: "car", Name: "car"}}},
	}
}

func unitVector(cos float64) []float32 {
	return []float32{float32(cos), float32(math.Sqrt(1 - cos*cos)), 0}
}

func TestEstimator_EmbeddingWithoutTokenOverlap(t *testing.T) {
	e := NewEstimator(embeddingCorpus(), WithEmbeddings(map[string][]float32{
		"rewe":  {1, 0, 0},
		"shell": {0, 0, 1},
	}))
	target := model.Transaction{ID: "edeka", Payee: "EDEKA Center"}

	if got := e.Estimate(target).CategoryScores[0].Score; got != 0 {
		t.Fatalf("score without target vector = %v, want 0", got)
	}

	e.SetEmbedding("edeka", unitVector(0.99))
	est := e.Estimate(target)
	if est.CategoryScores[0].CategoryName != "Groceries" || est.CategoryScores[0].Score <= est.CategoryScores[1].Score {
		t.Errorf("expected Groceries to win, got %+v", est.CategoryScores)
	}
	if est.TagScores[0].TagName != "food" {
		t.Errorf("expected tag food to win, got %+v", est.TagScores)
	}
}

func TestEstimator_EmbeddingBelowFloorIgnored(t *testing.T) {
	e := NewEstimator(embeddingCorpus(), WithEmbeddings(map[string][]float32{
		"rewe":  {1, 0, 0},
		"shell": {0, 0, 1},
		"edeka": unitVector(cosineFloor - 0.01),
	}))
	for _, s := range e.Estimate(model.Transaction{ID: "edeka", Payee: "EDEKA Center"}).CategoryScores {
		if s.Score != 0 {
			t.Errorf("score for %s = %v, want 0", s.CategoryName, s.Score)
		}
	}
}

func TestEstimator_MissingVectorMatchesJaccard(t *testing.T) {
	corpus := embeddingCorpus()
	jaccard := NewEstimator(corpus)
	targets := []model.Transaction{
		{ID: "t1", Payee: "REWE City", Memo: "Markt"},
		{ID: "t2", Payee: "Shell"},
	}
	cases := map[string]map[string][]float32{
		"only corpus vectors": {"rewe": {1, 0}, "shell": {0, 1}},
		"only target vectors": {"t1": {1, 0}, "t2": {0, 1}},
	}
	for name, vecs := range cases {
		t.Run(name, func(t *testing.T) {
			withEmb := NewEstimator(corpus, WithEmbeddings(vecs))
			for _, tx := range targets {
				if got, want := withEmb.Estimate(tx), jaccard.Estimate(tx); !reflect.DeepEqual(got, want) {
					t.Errorf("%s: Estimate = %+v, want %+v", tx.ID, got, want)
				}
			}
		})
	}
}
