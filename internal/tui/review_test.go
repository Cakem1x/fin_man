package tui

import (
	"strings"
	"testing"

	"github.com/Cakem1x/fin_man/internal/categorize"
	"github.com/Cakem1x/fin_man/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestReviewEscClosesUnchangedDraftWithoutSaving(t *testing.T) {
	form := NewReviewModel(model.Transaction{Memo: "same"}, nil, nil, nil, 100, 40)
	updated, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.(*ReviewModel).focusedPane == PaneExit {
		t.Fatal("unchanged draft opened the save menu")
	}
	if cmd == nil {
		t.Fatal("unchanged draft did not emit a close event")
	}
	if _, ok := cmd().(ReviewClosedMsg); !ok {
		t.Fatal("unchanged draft emitted something other than a close event")
	}
}

func TestReviewEscOpensMenuForSuggestedChangesAndRevertedEdits(t *testing.T) {
	form := NewReviewModel(model.Transaction{}, nil, nil, &categorize.Estimation{
		CategoryScores: []categorize.CategoryEstimation{{CategoryName: "Food", Score: .99}},
	}, 100, 40)
	if !form.HasPendingChanges() {
		t.Fatal("suggestion should count as a pending change")
	}
	form.selectedCategory = ""
	form.catList.ResetFilter()
	form.tagList.ResetFilter()
	if form.HasPendingChanges() {
		t.Fatal("reverted suggestion should not count as a pending change")
	}
	form.selectedCategory = "Food"
	var cmd tea.Cmd
	for i := 0; i < 3 && form.focusedPane != PaneExit; i++ {
		var updated tea.Model
		updated, cmd = form.Update(tea.KeyMsg{Type: tea.KeyEsc})
		form = *updated.(*ReviewModel)
	}
	if form.focusedPane != PaneExit {
		t.Fatalf("changed draft should open save menu without closing (pane %v, has changes %t)", form.focusedPane, form.HasPendingChanges())
	}
	if cmd != nil {
		t.Fatal("opening save menu unexpectedly emitted a close command")
	}
}

func TestReviewExitMenuShowsTransactionDiff(t *testing.T) {
	category := "Food"
	tx := model.Transaction{Memo: "before", CategoryName: &category, IsReviewed: true, Tags: []model.Tag{{Name: "home"}}}
	form := NewReviewModel(tx, []string{"Food"}, []string{"home", "work"}, nil, 100, 40)
	form.selectedCategory = "Transport"
	form.memo = "after"
	form.selectedTags["home"] = false
	form.selectedTags["work"] = true
	form.exitList.Select(1)
	form.focusedPane = PaneExit
	view := form.View()
	for _, expected := range []string{"Category: Food → Transport", "Memo: before → after", "Tags added: work", "Tags removed: home", "Reviewed: yes → no"} {
		if !strings.Contains(view, expected) {
			t.Errorf("menu diff missing %q", expected)
		}
	}
}

func TestReviewExitMenuDiffIsEmptyWhenDiscardIsSelected(t *testing.T) {
	category := "Food"
	tx := model.Transaction{Memo: "before", CategoryName: &category, Tags: []model.Tag{{Name: "home"}}}
	form := NewReviewModel(tx, []string{"Food"}, []string{"home", "work"}, nil, 100, 40)
	form.selectedCategory = "Transport"
	form.memo = "after"
	form.selectedTags["home"] = false
	form.selectedTags["work"] = true
	form.exitList.Select(2)
	form.focusedPane = PaneExit
	if form.exitDiff().HasChanges() {
		t.Fatal("discard selection should show an empty diff")
	}
	view := form.View()
	for _, draftValue := range []string{"Transport", "after", "Tags added", "Tags removed"} {
		if strings.Contains(view, draftValue) {
			t.Errorf("discard menu displayed draft diff value %q", draftValue)
		}
	}
}

func TestReviewExitDialogWidthIsStableAcrossOptions(t *testing.T) {
	category := "Food"
	tx := model.Transaction{Memo: "old", CategoryName: &category, Tags: []model.Tag{{Name: "home"}}}
	form := NewReviewModel(tx, []string{"Food"}, []string{"home", "work"}, nil, 100, 40)
	form.selectedCategory = "A substantially longer category name"
	form.memo = strings.Repeat("a long updated memo ", 4)
	form.focusedPane = PaneExit
	width := -1
	for index := 0; index < 3; index++ {
		form.exitList.Select(index)
		currentWidth := lipgloss.Width(form.exitDialog())
		if width < 0 {
			width = currentWidth
		} else if currentWidth != width {
			t.Fatalf("dialog width changed for option %d: got %d, want %d", index, currentWidth, width)
		}
	}
}

func TestReviewClosedDoesNotPersist(t *testing.T) {
	model := OverviewModel{reviewingTx: true, reviewingID: "tx", transactions: []model.Transaction{{ID: "tx", Memo: "original"}}}
	updated, _ := model.Update(ReviewClosedMsg{})
	got := updated.(*OverviewModel)
	if got.reviewingTx || got.reviewingID != "" {
		t.Fatal("close event did not close the review view")
	}
	if got.transactions[0].Memo != "original" {
		t.Fatal("close event changed transaction state")
	}
}
