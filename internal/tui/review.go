package tui

import (
	"fmt"

	"github.com/Cakem1x/fin_man/internal/categorize"
	"github.com/Cakem1x/fin_man/internal/model"
	"github.com/charmbracelet/huh"
)

type ReviewResult struct {
	Category string
	Tags     []string
	Memo     string
	Skip     bool
}

// ReviewTransaction presents a form to the user to enrich a transaction.
func ReviewTransaction(tx model.Transaction, categories []string, existingTags []string, suggestion *categorize.Estimation) (*ReviewResult, error) {
	var (
		category     string
		newCategory  string
		selectedTags []string
		newTags      string
		memo         = tx.Memo
	)

	if suggestion != nil {
		category = suggestion.CategoryName
	}

	catOptions := []huh.Option[string]{
		huh.NewOption("Skip (Leave Uncategorized)", ""),
		huh.NewOption("[Add New Category...]", "__add_new__"),
	}
	for _, c := range categories {
		catOptions = append(catOptions, huh.NewOption(c, c))
	}

	var tagOptions []huh.Option[string]
	for _, t := range existingTags { // Assume existingTags is passed as []string
		tagOptions = append(tagOptions, huh.NewOption(t, t))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Transaction Review").
				Description(fmt.Sprintf(
					"Date:   %s\nPayee:  %s\nAmount: %.2f %s\nMemo:   %s",
					tx.Date.Format("2006-01-02"),
					tx.Payee,
					float64(tx.AmountCents)/100.0,
					tx.Currency,
					tx.Memo,
				)),
			huh.NewSelect[string]().
				Title("Category (Primary Budget Group)").
				Options(catOptions...).
				Value(&category).
				Description("Type to fuzzy filter. Choose one main category."),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("New Category Name").
				Value(&newCategory),
		).WithHideFunc(func() bool {
			return category != "__add_new__"
		}),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Tags (Cross-category labels)").
				Options(tagOptions...).
				Value(&selectedTags).
				Description("Type to fuzzy filter. Select space to toggle."),
			huh.NewInput().
				Title("New Tag (optional)").
				Value(&newTags),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Update Memo").
				Value(&memo).
				Description("Modify the existing memo if needed"),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	// Resolve the final category and tags list
	finalCategory := category
	if category == "__add_new__" {
		finalCategory = newCategory
	}

	finalTags := []string{}
	finalTags = append(finalTags, selectedTags...)
	if newTags != "" {
		finalTags = append(finalTags, newTags)
	}

	return &ReviewResult{
		Category: finalCategory,
		Tags:     finalTags,
		Memo:     memo,
		Skip:     finalCategory == "",
	}, nil
}
