package common

import (
	"regexp"
	"sort"
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

// UniqueOptions collects unique non-empty strings from values, trims whitespace,
// deduplicates them, sorts them, and prepends allLabel (e.g. "Tutti" or "All").
// Used to populate dropdown filter options from data slices.
func UniqueOptions(values []string, allLabel string) []string {
	set := map[string]struct{}{}
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			set[v] = struct{}{}
		}
	}
	opts := make([]string, 0, len(set)+1)
	opts = append(opts, allLabel)
	for k := range set {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
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
