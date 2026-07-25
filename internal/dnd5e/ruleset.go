package dnd5e

import "strings"

// Ruleset selects which D&D 5th edition ruleset's catalog is shown: the
// original 2014 rules or the 2024 revision (colloquially "5.5e").
type Ruleset int

const (
	Ruleset2014 Ruleset = iota
	Ruleset2024
)

// core2024Sources are the source-book codes that make up the 2024 revision
// (Player's Handbook, Dungeon Master's Guide, Monster Manual). Everything
// else — third-party/legacy sourcebooks never reprinted for 2024 — is
// treated as 2014-era content and shown under both rulesets unless a 2024
// entry with the same name supersedes it.
var core2024Sources = map[string]bool{
	"XPHB": true,
	"XDMG": true,
	"XMM":  true,
}

// filterByRuleset scopes a catalog (monsters, items, spells, classes,
// races or feats — anything modeled as []Monster) to the given ruleset:
//   - Ruleset2014 excludes anything sourced only from a 2024 core book.
//   - Ruleset2024 prefers the 2024 revision when one exists (matched by
//     name) and otherwise keeps the legacy/third-party entry, so content
//     never revised for 2024 still shows up.
func filterByRuleset(entries []Monster, ruleset Ruleset) []Monster {
	if ruleset == Ruleset2014 {
		out := make([]Monster, 0, len(entries))
		for _, e := range entries {
			if !core2024Sources[e.Source] {
				out = append(out, e)
			}
		}
		return out
	}

	revisedNames := map[string]bool{}
	for _, e := range entries {
		if core2024Sources[e.Source] {
			revisedNames[strings.ToLower(strings.TrimSpace(e.Name))] = true
		}
	}
	out := make([]Monster, 0, len(entries))
	for _, e := range entries {
		if core2024Sources[e.Source] || !revisedNames[strings.ToLower(strings.TrimSpace(e.Name))] {
			out = append(out, e)
		}
	}
	return out
}

// filterCatalogByRuleset filters a []Monster catalog and recomputes its
// three facet lists (environments/CRs/types, or the per-catalog equivalent
// as returned by the loadXFromBytes functions) from the filtered set.
func filterCatalogByRuleset(entries []Monster, ruleset Ruleset) ([]Monster, []string, []string, []string) {
	filtered := filterByRuleset(entries, ruleset)

	facetA := map[string]struct{}{}
	facetB := map[string]struct{}{}
	facetC := map[string]struct{}{}
	for _, e := range filtered {
		for _, env := range e.Environment {
			facetA[env] = struct{}{}
		}
		cr := e.CR
		if cr == "" {
			cr = "Unknown"
		}
		facetB[cr] = struct{}{}
		t := e.Type
		if t == "" {
			t = "Unknown"
		}
		facetC[t] = struct{}{}
	}
	return filtered, keysSorted(facetA), sortCR(keysSorted(facetB)), keysSorted(facetC)
}

// rulesetLabel returns a short human-readable name for the ruleset, used in
// UI branding (help bar, window titles).
func rulesetLabel(ruleset Ruleset) string {
	if ruleset == Ruleset2024 {
		return "D&D 5.5e (2024)"
	}
	return "D&D 5e (2014)"
}

// rulesetShortName returns the systemEntry short name / save-directory
// suffix for the ruleset.
func rulesetShortName(ruleset Ruleset) string {
	if ruleset == Ruleset2024 {
		return "dnd5.5e"
	}
	return "dnd5e"
}
