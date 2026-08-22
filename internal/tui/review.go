package tui

import (
	"fmt"
	"strings"

	"github.com/Cakem1x/fin_man/internal/categorize"
	"github.com/Cakem1x/fin_man/internal/model"
	"github.com/charmbracelet/huh"
)

type ReviewAction string

const (
	ActionDiscard      ReviewAction = "Discard"
	ActionSave         ReviewAction = "Save changes (keep unreviewed)"
	ActionSaveReviewed ReviewAction = "Save & Mark Reviewed"
)

type ReviewResult struct {
	Category string
	Tags     []string
	Memo     string
}

type ReviewFormState struct {
	Category     string
	SelectedTags []string
	Memo         string
	Action       string
}

// BuildReviewForms creates three huh forms for the given transaction.
func BuildReviewForms(tx model.Transaction, categories []string, existingTags []string, suggestion *categorize.Estimation, state *ReviewFormState) (*huh.Form, *huh.Form, *huh.Form) {
	var suggestedText string
	if suggestion != nil && suggestion.Confidence >= 0.5 {
		state.Category = suggestion.CategoryName
		suggestedText = fmt.Sprintf(" (Auto-suggested: %s, %.0f%% confidence)", suggestion.CategoryName, suggestion.Confidence*100)
	} else if suggestion != nil {
		suggestedText = fmt.Sprintf(" (Top guess: %s, %.0f%% confidence - below threshold)", suggestion.CategoryName, suggestion.Confidence*100)
	}
	state.Memo = tx.Memo

	catOptions := []huh.Option[string]{
		huh.NewOption("None", ""),
	}
	for _, c := range categories {
		catOptions = append(catOptions, huh.NewOption(c, c))
	}

	var tagOptions []huh.Option[string]
	for _, t := range existingTags {
		tagOptions = append(tagOptions, huh.NewOption(t, t))
	}

	state.Action = string(ActionSaveReviewed) // Default action

	memoForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Update Memo").
				Value(&state.Memo).
				Description("Modify the existing memo if needed"),
		),
	)

	catForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("category").
				Title("Category (Primary Budget Group)").
				Options(catOptions...).
				Value(&state.Category).
				Description(strings.TrimSpace(suggestedText)),
		),
	)

	tagForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Key("tags").
				Title("Tags (Cross-category labels)").
				Options(tagOptions...).
				Value(&state.SelectedTags).
				Description("Select space to toggle."),
		),
	)

	return memoForm, catForm, tagForm
}

// ExtractReviewResult resolves the final category and tags list from the form state.
func ExtractReviewResult(state *ReviewFormState) *ReviewResult {
	return &ReviewResult{
		Category: state.Category,
		Tags:     state.SelectedTags,
		Memo:     state.Memo,
	}
}
