package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Cakem1x/fin_man/internal/model"
)

// TransactionDiff describes enrichment changes for one transaction. Callers
// can compose rendered diffs without coupling this value to a particular view.
type TransactionDiff struct {
	Fields      []FieldDiff
	TagsAdded   []string
	TagsRemoved []string
	Reviewed    *BoolDiff
}

// FieldDiff is a before/after value for one named transaction field.
type FieldDiff struct {
	Field string
	From  string
	To    string
}

// BoolDiff is a before/after boolean value.
type BoolDiff struct {
	From bool
	To   bool
}

// HasChanges reports whether the diff contains any transaction changes.
func (d TransactionDiff) HasChanges() bool {
	return len(d.Fields) > 0 || len(d.TagsAdded) > 0 || len(d.TagsRemoved) > 0 || d.Reviewed != nil
}

// BuildTransactionDiff compares two transaction states. Tag ordering is
// ignored; tag names are treated as a set.
func BuildTransactionDiff(before, after model.Transaction) TransactionDiff {
	d := TransactionDiff{}
	if categoryName(before.CategoryName) != categoryName(after.CategoryName) {
		d.Fields = append(d.Fields, FieldDiff{"Category", displayValue(categoryName(before.CategoryName)), displayValue(categoryName(after.CategoryName))})
	}
	if before.Memo != after.Memo {
		d.Fields = append(d.Fields, FieldDiff{"Memo", displayValue(before.Memo), displayValue(after.Memo)})
	}
	beforeTags, afterTags := make(map[string]bool), make(map[string]bool)
	for _, name := range transactionTagNames(before.Tags) {
		beforeTags[name] = true
	}
	for _, name := range transactionTagNames(after.Tags) {
		afterTags[name] = true
	}
	for name := range afterTags {
		if !beforeTags[name] {
			d.TagsAdded = append(d.TagsAdded, name)
		}
	}
	for name := range beforeTags {
		if !afterTags[name] {
			d.TagsRemoved = append(d.TagsRemoved, name)
		}
	}
	sort.Strings(d.TagsAdded)
	sort.Strings(d.TagsRemoved)
	if before.IsReviewed != after.IsReviewed {
		d.Reviewed = &BoolDiff{From: before.IsReviewed, To: after.IsReviewed}
	}
	return d
}

// String renders a single transaction's diff as readable lines.
func (d TransactionDiff) String() string {
	var lines []string
	for _, field := range d.Fields {
		lines = append(lines, fmt.Sprintf("%s: %s → %s", field.Field, field.From, field.To))
	}
	if len(d.TagsAdded) > 0 {
		lines = append(lines, "Tags added: "+strings.Join(d.TagsAdded, ", "))
	}
	if len(d.TagsRemoved) > 0 {
		lines = append(lines, "Tags removed: "+strings.Join(d.TagsRemoved, ", "))
	}
	if d.Reviewed != nil {
		lines = append(lines, fmt.Sprintf("Reviewed: %s → %s", boolLabel(d.Reviewed.From), boolLabel(d.Reviewed.To)))
	}
	if !d.HasChanges() {
		return "No transaction changes"
	}
	return strings.Join(lines, "\n")
}

func categoryName(name *string) string {
	if name == nil {
		return ""
	}
	return *name
}

func transactionTagNames(tags []model.Tag) []string {
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag.Name != "" {
			names = append(names, tag.Name)
		}
	}
	return names
}

func selectedTagNames(selected map[string]bool) []string {
	names := make([]string, 0, len(selected))
	for name, yes := range selected {
		if yes && name != "" {
			names = append(names, name)
		}
	}
	return names
}

func displayValue(value string) string {
	if value == "" {
		return "(empty)"
	}
	return value
}

func boolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
