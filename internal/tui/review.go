package tui

import (
	"fmt"

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
func ReviewTransaction(tx model.Transaction, categories []string, existingTags []string) (*ReviewResult, error) {
	var (
		category     string
		newCategory  string
		selectedTags []string
		newTags      string
		memo         = tx.Memo
	)

	if tx.CategoryName != nil {
		category = *tx.CategoryName
	}
	for _, t := range tx.Tags {
		selectedTags = append(selectedTags, t.Name)
	}

	catOptions := []huh.Option[string]{
		huh.NewOption("Skip (Leave Uncategorized)", ""),
		huh.NewOption("[Add New Category...]", "__add_new__"),
	}
	for _, c := range categories {
		catOptions = append(catOptions, huh.NewOption(c, c))
	}

	tagOptions := []huh.Option[string]{
		huh.NewOption("[Add New Tags...]", "__add_new__"),
	}
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
		),
		huh.NewGroup(
			huh.NewInput().
				Title("New Tags (comma separated)").
				Value(&newTags),
		).WithHideFunc(func() bool {
			for _, t := range selectedTags {
				if t == "__add_new__" {
					return false
				}
			}
			return true
		}),
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
	for _, t := range selectedTags {
		if t != "__add_new__" {
			finalTags = append(finalTags, t)
		}
	}
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
