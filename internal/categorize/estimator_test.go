package categorize

import (
	"testing"

	"github.com/Cakem1x/fin_man/internal/model"
)

func TestEstimator(t *testing.T) {
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
		name         string
		tx           model.Transaction
		expectedCat  string
		expectNil    bool
	}{
		{
			name: "Exact match with UBER EATS",
			tx: model.Transaction{
				Payee: "UBER EATS *4567",
				Memo:  "",
			},
			expectedCat: "Food",
		},
		{
			name: "Match with UBER TRIP",
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
		{
			name: "No match",
			tx: model.Transaction{
				Payee: "NETFLIX",
				Memo:  "",
			},
			expectedCat: "",
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimator.Estimate(tt.tx)
			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil result, got category %s", result.CategoryName)
				}
				return
			}
			if result == nil {
				t.Fatalf("Expected category %s, got nil", tt.expectedCat)
			}
			if result.CategoryName != tt.expectedCat {
				t.Errorf("Expected category %s, got %s", tt.expectedCat, result.CategoryName)
			}
		})
	}
}
