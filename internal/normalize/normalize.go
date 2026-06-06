package normalize

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// CleanWhitespace removes leading/trailing spaces and replaces multiple internal
// whitespace characters with a single space.
func CleanWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// TitleCasePayee converts a payee string to title case (e.g., AMAZON -> Amazon).
func TitleCasePayee(s string) string {
	cleaned := CleanWhitespace(s)
	caser := cases.Title(language.English)
	return caser.String(cleaned)
}
