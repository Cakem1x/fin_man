package normalize_test

import (
	"testing"

	"github.com/Cakem1x/fin_man/internal/normalize"
)

func TestCleanWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already clean",
			input:    "Clean String",
			expected: "Clean String",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "multiple internal spaces",
			input:    "Multiple    Spaces   Here",
			expected: "Multiple Spaces Here",
		},
		{
			name:     "tabs and newlines",
			input:    "Tab\tAnd\nNewline",
			expected: "Tab And Newline",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only spaces",
			input:    "     ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalize.CleanWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("CleanWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTitleCasePayee(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "all caps",
			input:    "AMAZON MARKETPLACE",
			expected: "Amazon Marketplace",
		},
		{
			name:     "all lowercase",
			input:    "rewe markt gmbh",
			expected: "Rewe Markt Gmbh",
		},
		{
			name:     "already title case",
			input:    "Spotify Ab",
			expected: "Spotify Ab",
		},
		{
			name:     "leading whitespace and mixed casing",
			input:    "   uber BV  ",
			expected: "Uber Bv", // Expects whitespace cleanup to have happened or handled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalize.TitleCasePayee(tt.input)
			if result != tt.expected {
				t.Errorf("TitleCasePayee(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
