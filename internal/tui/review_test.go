package tui

import (
	"testing"

	"github.com/Cakem1x/fin_man/internal/categorize"
	"github.com/Cakem1x/fin_man/internal/model"
)

func TestNewReviewModelPreservesExistingSelections(t *testing.T) {
	category := "Food"
	tx := model.Transaction{
		CategoryID:   &category,
		CategoryName: &category,
		Tags: []model.Tag{
			{ID: "home", Name: "home"},
		},
	}
	suggestion := &categorize.Estimation{
		CategoryScores: []categorize.CategoryEstimation{{CategoryName: "Transport", Score: 0.99}},
		TagScores:      []categorize.TagEstimation{{TagName: "work", Score: 0.99}},
	}

	form := NewReviewModel(tx, []string{"Food", "Transport"}, []string{"home", "work"}, suggestion, 100, 40)

	if form.selectedCategory != "Food" {
		t.Errorf("selected category = %q, want existing category %q", form.selectedCategory, "Food")
	}
	if !form.selectedTags["home"] {
		t.Error("existing tag was not selected")
	}
	if form.selectedTags["work"] {
		t.Error("suggested tag was added despite existing transaction tags")
	}
	for _, item := range form.catList.Items() {
		categoryItem, ok := item.(selectableItem)
		if ok && categoryItem.id == "Food" && !categoryItem.selected {
			t.Error("existing category is not marked selected in the category list")
		}
	}
}
