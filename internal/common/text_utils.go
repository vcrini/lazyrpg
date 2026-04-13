package common

import (
	"regexp"
	"strings"
)

// IndexToLetter converts a 1-based index to an uppercase letter label:
// 1→"A", 2→"B", …, 26→"Z", 27→"AA", 28→"AB", etc.
func IndexToLetter(n int) string {
	if n <= 0 {
		return "?"
	}
	result := ""
	for n > 0 {
		n--
		result = string(rune('A'+n%26)) + result
		n /= 26
	}
	return result
}

// CardDescriptionHead extracts the heading portion of a card description
// (everything before the first colon).
func CardDescriptionHead(desc string) string {
	s := strings.TrimSpace(desc)
	if s == "" || strings.EqualFold(s, "Da screenshot.") {
		return ""
	}
	if i := strings.Index(s, ":"); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// HighlightMatches wraps every occurrence of query inside text with tview
// gold-on-black colour tags.
func HighlightMatches(text, query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return text
	}
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(q))
	if err != nil {
		return text
	}
	return re.ReplaceAllStringFunc(text, func(m string) string {
		return "[black:gold]" + m + "[-:-]"
	})
}
