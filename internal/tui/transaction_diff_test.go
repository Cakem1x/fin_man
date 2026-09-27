package tui

import (
	"strings"
	"testing"

	"github.com/Cakem1x/fin_man/internal/model"
)

func TestBuildTransactionDiffShowsEmptyValuesAndIgnoresTagOrder(t *testing.T) {
	category := "Food"
	before := model.Transaction{
		CategoryName: &category,
		Memo:         "Old memo",
		Tags:         []model.Tag{{Name: "home"}, {Name: "work"}},
	}
	after := model.Transaction{
		Tags: []model.Tag{{Name: "work"}, {Name: "home"}},
	}
	diff := BuildTransactionDiff(before, after).String()
	for _, expected := range []string{"Category: Food → (empty)", "Memo: Old memo → (empty)"} {
		if !strings.Contains(diff, expected) {
			t.Errorf("diff missing %q", expected)
		}
	}
	if strings.Contains(diff, "Tags added") || strings.Contains(diff, "Tags removed") {
		t.Fatalf("tag order alone should not produce a diff: %s", diff)
	}
}
