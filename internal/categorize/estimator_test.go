package categorize

import (
	"testing"

	"github.com/Cakem1x/fin_man/internal/model"
)

func TestEstimator_CategoryScores(t *testing.T) {
	cat1ID := "cat1"
	cat1Name := "Food"
	cat2ID := "cat2"
	cat2Name := "Transport"

	corpus := []model.Transaction{
		{
			Payee:        "UBER EATS",
			Memo:         "Order 1234",
			CategoryID:   &cat1ID,
			CategoryName: &cat1Name,
		},
		{
			Payee:        "UBER TRIP",
			Memo:         "Ride to work",
			CategoryID:   &cat2ID,
			CategoryName: &cat2Name,
		},
		{
			Payee:        "STARBUCKS STORE",
			Memo:         "",
			CategoryID:   &cat1ID,
			CategoryName: &cat1Name,
		},
	}

	estimator := NewEstimator(corpus)

	tests := []struct {
		name        string
		tx          model.Transaction
		expectedCat string
		expectNil   bool
	}{
		{
			name: "Best match with UBER EATS",
			tx: model.Transaction{
				Payee: "UBER EATS *4567",
				Memo:  "",
			},
			expectedCat: "Food",
		},
		{
			name: "Best match with UBER TRIP",
			tx: model.Transaction{
				Payee: "Uber Trip",
				Memo:  "Ride home",
			},
			expectedCat: "Transport",
		},
		{
			name: "Match with STARBUCKS",
			tx: model.Transaction{
				Payee: "Starbucks",
				Memo:  "Coffee",
			},
			expectedCat: "Food",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimator.Estimate(tt.tx)
			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatalf("Expected category %s, got nil", tt.expectedCat)
			}
			if len(result.CategoryScores) == 0 {
				t.Fatalf("Expected category scores, got empty slice")
			}
			topCat := result.CategoryScores[0].CategoryName
			if topCat != tt.expectedCat {
				t.Errorf("Expected top category %s, got %s", tt.expectedCat, topCat)
			}
		})
	}
}

func TestEstimator_NoMemory(t *testing.T) {
	estimator := NewEstimator(nil)
	tx := model.Transaction{Payee: "NETFLIX"}
	if res := estimator.Estimate(tx); res != nil {
		t.Fatalf("Expected nil, got %v", res)
	}
}

func TestEstimator_OnlineLearning(t *testing.T) {
	estimator := NewEstimator(nil)

	// Initial estimation should return nil
	tx1 := model.Transaction{Payee: "AMAZON WEB SERVICES"}
	if res := estimator.Estimate(tx1); res != nil {
		t.Fatalf("Expected nil, got %v", res)
	}

	// Simulate user categorizing the transaction in the TUI
	catID := "cat-aws"
	catName := "Cloud Infrastructure"
	tx1.CategoryID = &catID
	tx1.CategoryName = &catName

	// Model learns from the review
	estimator.Learn(tx1)

	// Subsequent estimation for a similar transaction should now succeed
	tx2 := model.Transaction{Payee: "AMAZON WEB SRVCS", Memo: "AWS"}
	res := estimator.Estimate(tx2)

	if res == nil {
		t.Fatal("Expected an estimation, got nil")
	}
	if len(res.CategoryScores) == 0 {
		t.Fatal("Expected category scores, got empty slice")
	}
	if res.CategoryScores[0].CategoryID != catID {
		t.Errorf("Expected category %s, got %s", catID, res.CategoryScores[0].CategoryID)
	}
}

func TestEstimator_TagScores_HighCertainty(t *testing.T) {
	// One very similar transaction has the tag, no negative examples.
	// Expected: high tag score.
	catID := "cat1"
	catName := "Food"
	corpus := []model.Transaction{
		{
			Payee:        "UBER EATS",
			Memo:         "Order",
			CategoryID:   &catID,
			CategoryName: &catName,
			Tags:         []model.Tag{{ID: "t1", Name: "delivery"}},
		},
	}

	estimator := NewEstimator(corpus)
	res := estimator.Estimate(model.Transaction{Payee: "UBER EATS", Memo: "Order new"})

	if res == nil {
		t.Fatal("Expected estimation, got nil")
	}
	if len(res.TagScores) == 0 {
		t.Fatal("Expected tag scores, got empty")
	}
	if res.TagScores[0].TagName != "delivery" {
		t.Errorf("Expected tag 'delivery', got %s", res.TagScores[0].TagName)
	}
	if res.TagScores[0].Score < 0.5 {
		t.Errorf("Expected high tag score for 'delivery', got %.3f", res.TagScores[0].Score)
	}
}

func TestEstimator_TagScores_Ambiguity(t *testing.T) {
	// Two very similar transactions: one has the tag, one does not.
	// Expected: tag score near 0.5 (ambiguous).
	catID := "cat1"
	catName := "Food"
	corpus := []model.Transaction{
		{
			Payee:        "UBER EATS",
			Memo:         "Order",
			CategoryID:   &catID,
			CategoryName: &catName,
			Tags:         []model.Tag{{ID: "t1", Name: "delivery"}},
		},
		{
			Payee:        "UBER EATS",
			Memo:         "Order",
			CategoryID:   &catID,
			CategoryName: &catName,
			Tags:         []model.Tag{}, // no delivery tag
		},
	}

	estimator := NewEstimator(corpus)
	res := estimator.Estimate(model.Transaction{Payee: "UBER EATS", Memo: "Order new"})

	if res == nil {
		t.Fatal("Expected estimation, got nil")
	}
	if len(res.TagScores) == 0 {
		t.Fatal("Expected tag scores, got empty")
	}

	var deliveryScore float64
	for _, ts := range res.TagScores {
		if ts.TagName == "delivery" {
			deliveryScore = ts.Score
		}
	}

	// With equal positive and negative evidence, score should be around 0.47
	// (due to alpha smoothing: wPos / (wPos + wNeg + alpha))
	if deliveryScore > 0.55 {
		t.Errorf("Expected ambiguous (low) tag score for 'delivery', got %.3f", deliveryScore)
	}
}

func TestEstimator_TagScores_NegativeEvidence(t *testing.T) {
	// Multiple similar transactions without the tag, one weaker one with.
	// Expected: the tag score should be low.
	catID := "cat1"
	catName := "Food"
	corpus := []model.Transaction{
		{
			Payee:        "UBER EATS",
			Memo:         "Dinner",
			CategoryID:   &catID,
			CategoryName: &catName,
			Tags:         []model.Tag{},
		},
		{
			Payee:        "UBER EATS",
			Memo:         "Lunch",
			CategoryID:   &catID,
			CategoryName: &catName,
			Tags:         []model.Tag{},
		},
		{
			Payee:        "Delivery Service Extra",
			Memo:         "Special order",
			CategoryID:   &catID,
			CategoryName: &catName,
			Tags:         []model.Tag{{ID: "t1", Name: "delivery"}},
		},
	}

	estimator := NewEstimator(corpus)
	res := estimator.Estimate(model.Transaction{Payee: "UBER EATS", Memo: "Dinner order"})

	if res == nil {
		t.Fatal("Expected estimation, got nil")
	}

	var deliveryScore float64
	for _, ts := range res.TagScores {
		if ts.TagName == "delivery" {
			deliveryScore = ts.Score
		}
	}

	// The highly similar negative examples should dominate.
	if deliveryScore > 0.3 {
		t.Errorf("Expected low tag score for 'delivery' due to negative evidence, got %.3f", deliveryScore)
	}
}

func TestEstimator_MultipleTags(t *testing.T) {
	catID := "cat1"
	catName := "Food"
	corpus := []model.Transaction{
		{
			Payee:        "UBER EATS",
			Memo:         "Order",
			CategoryID:   &catID,
			CategoryName: &catName,
			Tags: []model.Tag{
				{ID: "t1", Name: "delivery"},
				{ID: "t2", Name: "dinner"},
			},
		},
	}

	estimator := NewEstimator(corpus)
	res := estimator.Estimate(model.Transaction{Payee: "UBER EATS", Memo: "Order new"})

	if res == nil {
		t.Fatal("Expected estimation, got nil")
	}
	if len(res.TagScores) != 2 {
		t.Fatalf("Expected 2 tag scores, got %d", len(res.TagScores))
	}

	// Both tags should have the same high score (same single positive evidence)
	for _, ts := range res.TagScores {
		if ts.Score < 0.5 {
			t.Errorf("Expected high score for tag %s, got %.3f", ts.TagName, ts.Score)
		}
	}
}

func TestEstimator_SkipsUncategorized(t *testing.T) {
	// Transactions without a CategoryID should be ignored.
	corpus := []model.Transaction{
		{
			Payee: "NETFLIX",
			Memo:  "Subscription",
			Tags:  []model.Tag{{ID: "t1", Name: "streaming"}},
			// No CategoryID — should be skipped
		},
	}

	estimator := NewEstimator(corpus)
	if len(estimator.memory) != 0 {
		t.Errorf("Expected 0 memory entries for uncategorized tx, got %d", len(estimator.memory))
	}
}

func TestEstimator_ScoresSortedDescending(t *testing.T) {
	cat1ID := "cat1"
	cat1Name := "Food"
	cat2ID := "cat2"
	cat2Name := "Transport"

	corpus := []model.Transaction{
		{
			Payee:        "UBER EATS",
			Memo:         "Order",
			CategoryID:   &cat1ID,
			CategoryName: &cat1Name,
			Tags:         []model.Tag{{ID: "t1", Name: "delivery"}},
		},
		{
			Payee:        "TAXI SERVICE",
			Memo:         "Airport",
			CategoryID:   &cat2ID,
			CategoryName: &cat2Name,
			Tags:         []model.Tag{{ID: "t2", Name: "travel"}},
		},
	}

	estimator := NewEstimator(corpus)
	res := estimator.Estimate(model.Transaction{Payee: "UBER EATS", Memo: "Order"})

	if res == nil {
		t.Fatal("Expected estimation, got nil")
	}

	// Category scores should be sorted descending
	for i := 1; i < len(res.CategoryScores); i++ {
		if res.CategoryScores[i].Score > res.CategoryScores[i-1].Score {
			t.Errorf("Category scores not sorted descending at index %d: %.3f > %.3f",
				i, res.CategoryScores[i].Score, res.CategoryScores[i-1].Score)
		}
	}

	// Tag scores should be sorted descending
	for i := 1; i < len(res.TagScores); i++ {
		if res.TagScores[i].Score > res.TagScores[i-1].Score {
			t.Errorf("Tag scores not sorted descending at index %d: %.3f > %.3f",
				i, res.TagScores[i].Score, res.TagScores[i-1].Score)
		}
	}

	// The "delivery" tag should rank higher because UBER EATS is highly similar
	if len(res.TagScores) >= 2 && res.TagScores[0].TagName != "delivery" {
		t.Errorf("Expected 'delivery' as top tag, got %s", res.TagScores[0].TagName)
	}
}
