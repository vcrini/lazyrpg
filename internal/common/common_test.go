package common

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ── CapitalizeWord ────────────────────────────────────────────────────────────

func TestCapitalizeWord(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"WORLD", "WORLD"},
		{"a", "A"},
		{"123abc", "123abc"},
	}
	for _, tc := range tests {
		if got := CapitalizeWord(tc.in); got != tc.want {
			t.Errorf("CapitalizeWord(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── DefaultNameLists ──────────────────────────────────────────────────────────

func TestDefaultNameLists_NotEmpty(t *testing.T) {
	lists := DefaultNameLists()
	if len(lists.First) == 0 {
		t.Error("DefaultNameLists().First is empty")
	}
	if len(lists.Last) == 0 {
		t.Error("DefaultNameLists().Last is empty")
	}
}

// ── LoadNameLists ─────────────────────────────────────────────────────────────

func TestLoadNameLists_ValidYAML(t *testing.T) {
	data := "first:\n  - Alice\n  - Bob\nlast:\n  - Smith\n  - Jones\n"
	readFn := func(string) ([]byte, error) { return []byte(data), nil }

	lists, err := LoadNameLists(readFn, "names.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists.First) != 2 || lists.First[0] != "Alice" {
		t.Errorf("unexpected First list: %v", lists.First)
	}
	if len(lists.Last) != 2 || lists.Last[1] != "Jones" {
		t.Errorf("unexpected Last list: %v", lists.Last)
	}
}

func TestLoadNameLists_StripsWhitespace(t *testing.T) {
	data := "first:\n  - \"  Alice  \"\n  - \"  \"\nlast:\n  - Smith\n"
	readFn := func(string) ([]byte, error) { return []byte(data), nil }

	lists, err := LoadNameLists(readFn, "names.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists.First) != 1 || lists.First[0] != "Alice" {
		t.Errorf("expected [Alice], got %v", lists.First)
	}
}

func TestLoadNameLists_ReadError_FallsBackToDefault(t *testing.T) {
	readFn := func(string) ([]byte, error) { return nil, fmt.Errorf("not found") }

	lists, _ := LoadNameLists(readFn, "names.yml")
	def := DefaultNameLists()
	if len(lists.First) != len(def.First) {
		t.Errorf("expected default names, got %v", lists.First)
	}
}

func TestLoadNameLists_EmptyFirstList_FallsBackToDefault(t *testing.T) {
	data := "first:\nlast:\n  - Smith\n"
	readFn := func(string) ([]byte, error) { return []byte(data), nil }

	lists, _ := LoadNameLists(readFn, "names.yml")
	def := DefaultNameLists()
	if len(lists.First) != len(def.First) {
		t.Errorf("expected default names on empty first list, got %v", lists.First)
	}
}

// ── RandomPNGName ─────────────────────────────────────────────────────────────

func TestRandomPNGName_EmptyFirstReturnsUnknown(t *testing.T) {
	lists := NameLists{}
	if got := RandomPNGName(lists); got != "Unknown" {
		t.Errorf("expected Unknown, got %q", got)
	}
}

func TestRandomPNGName_NoLastReturnsFirstOnly(t *testing.T) {
	lists := NameLists{First: []string{"Aria"}}
	if got := RandomPNGName(lists); got != "Aria" {
		t.Errorf("expected Aria, got %q", got)
	}
}

func TestRandomPNGName_WithBothLists(t *testing.T) {
	lists := NameLists{
		First: []string{"Aria"},
		Last:  []string{"Storm"},
	}
	got := RandomPNGName(lists)
	if got != "Aria Storm" {
		t.Errorf("expected 'Aria Storm', got %q", got)
	}
}

func TestRandomPNGName_ReturnsFromLists(t *testing.T) {
	lists := DefaultNameLists()
	for i := 0; i < 20; i++ {
		name := RandomPNGName(lists)
		parts := strings.SplitN(name, " ", 2)
		if !contains(lists.First, parts[0]) {
			t.Errorf("first name %q not in First list", parts[0])
		}
		if len(parts) == 2 && !contains(lists.Last, parts[1]) {
			t.Errorf("last name %q not in Last list", parts[1])
		}
	}
}

// ── UniqueRandomPNGName ───────────────────────────────────────────────────────

func TestUniqueRandomPNGName_IsUnique(t *testing.T) {
	lists := NameLists{
		First: []string{"Aria", "Brom", "Cala"},
		Last:  []string{"Stone", "Wind"},
	}
	existing := []string{"Aria Stone", "Aria Wind", "Brom Stone"}
	seen := make(map[string]struct{}, len(existing))
	for _, n := range existing {
		seen[n] = struct{}{}
	}
	for i := 0; i < 30; i++ {
		name := UniqueRandomPNGName(existing, lists)
		if _, ok := seen[name]; ok {
			t.Errorf("UniqueRandomPNGName returned existing name %q", name)
		}
	}
}

func TestUniqueRandomPNGName_NoExisting(t *testing.T) {
	lists := DefaultNameLists()
	name := UniqueRandomPNGName(nil, lists)
	if name == "" || name == "Unknown" {
		t.Errorf("unexpected name: %q", name)
	}
}

// ── CardDescriptionHead ───────────────────────────────────────────────────────

func TestCardDescriptionHead(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"  ", ""},
		{"Da screenshot.", ""},
		{"DA SCREENSHOT.", ""},
		{"Danno: tiro +2", "Danno"},
		{"Cura: recupera 1d6 HP", "Cura"},
		{"Nessun colon qui", "Nessun colon qui"},
		{"  Titolo: dettaglio  ", "Titolo"},
		{":solo colon iniziale", ":solo colon iniziale"}, // colon at pos 0: no heading
	}
	for _, tc := range tests {
		got := CardDescriptionHead(tc.in)
		if got != tc.want {
			t.Errorf("CardDescriptionHead(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── HighlightMatches ──────────────────────────────────────────────────────────

func TestHighlightMatches_EmptyQuery(t *testing.T) {
	text := "hello world"
	if got := HighlightMatches(text, ""); got != text {
		t.Errorf("expected unchanged text, got %q", got)
	}
}

func TestHighlightMatches_WhitespaceQuery(t *testing.T) {
	text := "hello world"
	if got := HighlightMatches(text, "   "); got != text {
		t.Errorf("expected unchanged text, got %q", got)
	}
}

func TestHighlightMatches_CaseInsensitive(t *testing.T) {
	got := HighlightMatches("Hello World", "hello")
	if !strings.Contains(got, "[black:gold]Hello[-:-]") {
		t.Errorf("expected gold highlight, got %q", got)
	}
}

func TestHighlightMatches_MultipleOccurrences(t *testing.T) {
	got := HighlightMatches("abc abc abc", "abc")
	count := strings.Count(got, "[black:gold]")
	if count != 3 {
		t.Errorf("expected 3 highlights, got %d in %q", count, got)
	}
}

func TestHighlightMatches_RegexSpecialChars(t *testing.T) {
	// Should not panic on regex special characters in query
	got := HighlightMatches("cost (1d6+2)", "(1d6+2)")
	if !strings.Contains(got, "[black:gold]") {
		t.Errorf("expected highlight for special chars, got %q", got)
	}
}

func TestHighlightMatches_NoMatch(t *testing.T) {
	text := "hello world"
	got := HighlightMatches(text, "xyz")
	if got != text {
		t.Errorf("expected unchanged text on no match, got %q", got)
	}
}

// ── Thresholds UnmarshalYAML ──────────────────────────────────────────────────

func TestThresholds_UnmarshalSequence(t *testing.T) {
	data := "thresholds: [5, 11]\n"
	var v struct {
		Thresholds Thresholds `yaml:"thresholds"`
	}
	if err := yaml.Unmarshal([]byte(data), &v); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(v.Thresholds.Values) != 2 || v.Thresholds.Values[0] != 5 || v.Thresholds.Values[1] != 11 {
		t.Errorf("unexpected Values: %v", v.Thresholds.Values)
	}
	if v.Thresholds.Text != "" {
		t.Errorf("expected empty Text, got %q", v.Thresholds.Text)
	}
}

func TestThresholds_UnmarshalScalar(t *testing.T) {
	data := "thresholds: \"speciale\"\n"
	var v struct {
		Thresholds Thresholds `yaml:"thresholds"`
	}
	if err := yaml.Unmarshal([]byte(data), &v); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if v.Thresholds.Text != "speciale" {
		t.Errorf("expected Text=speciale, got %q", v.Thresholds.Text)
	}
	if len(v.Thresholds.Values) != 0 {
		t.Errorf("expected empty Values, got %v", v.Thresholds.Values)
	}
}

// ── LoadYAMLList ──────────────────────────────────────────────────────────────

func TestLoadYAMLList_ValidList(t *testing.T) {
	type item struct {
		Name string `yaml:"name"`
	}
	data := "- name: Goblin\n- name: Orc\n"
	readFn := func(string) ([]byte, error) { return []byte(data), nil }

	items, err := LoadYAMLList[item]("mostri.yml", readFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 || items[0].Name != "Goblin" || items[1].Name != "Orc" {
		t.Errorf("unexpected items: %v", items)
	}
}

func TestLoadYAMLList_ReadError(t *testing.T) {
	type item struct{ Name string }
	readFn := func(string) ([]byte, error) { return nil, fmt.Errorf("not found") }

	_, err := LoadYAMLList[item]("missing.yml", readFn)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoadYAMLList_InvalidYAML(t *testing.T) {
	type item struct{ Name string }
	readFn := func(string) ([]byte, error) { return []byte(":::invalid:::"), nil }

	_, err := LoadYAMLList[item]("bad.yml", readFn)
	if err == nil {
		t.Error("expected unmarshal error, got nil")
	}
}

func TestLoadYAMLList_EmptyFile(t *testing.T) {
	type item struct{ Name string }
	readFn := func(string) ([]byte, error) { return []byte(""), nil }

	items, err := LoadYAMLList[item]("empty.yml", readFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty slice, got %v", items)
	}
}

// ── ReadData (filesystem fallback) ───────────────────────────────────────────

func TestReadData_FilesystemFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yml")
	content := []byte("hello: world\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// path does not start with "config/" so ReadData uses os.ReadFile directly
	var emptyFS embed.FS
	got, err := ReadData(path, emptyFS)
	if err != nil {
		t.Fatalf("ReadData filesystem fallback error: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("ReadData = %q, want %q", got, content)
	}
}

func TestReadData_NotFound(t *testing.T) {
	var emptyFS embed.FS
	_, err := ReadData("/nonexistent/path/file.yml", emptyFS)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// ── ClassPresetFor ────────────────────────────────────────────────────────────

func TestClassPresetFor_KnownClasses(t *testing.T) {
	classes := []string{
		"bardo", "consacrato", "druido", "fuorilegge",
		"guardiano", "guerriero", "mago", "ranger", "stregone",
	}
	for _, c := range classes {
		preset := ClassPresetFor(c)
		if preset.Traits == "" {
			t.Errorf("ClassPresetFor(%q): empty Traits", c)
		}
		if preset.Primary == "" && c != "guardiano" && c != "guerriero" && c != "mago" && c != "ranger" && c != "stregone" {
			t.Errorf("ClassPresetFor(%q): empty Primary", c)
		}
	}
}

func TestClassPresetFor_CaseInsensitive(t *testing.T) {
	lower := ClassPresetFor("bardo")
	upper := ClassPresetFor("BARDO")
	mixed := ClassPresetFor("Bardo")
	if lower.Traits != upper.Traits || lower.Traits != mixed.Traits {
		t.Error("ClassPresetFor should be case-insensitive")
	}
}

func TestClassPresetFor_UnknownClass(t *testing.T) {
	preset := ClassPresetFor("classeinesistente")
	if preset.Traits != "" || preset.Primary != "" {
		t.Errorf("expected empty preset for unknown class, got %+v", preset)
	}
}

func TestClassPresetFor_HasAbiti(t *testing.T) {
	classes := []string{"bardo", "consacrato", "druido", "fuorilegge", "guardiano", "guerriero", "mago", "ranger", "stregone"}
	for _, c := range classes {
		preset := ClassPresetFor(c)
		if len(preset.Abiti) == 0 {
			t.Errorf("ClassPresetFor(%q): empty Abiti", c)
		}
		if len(preset.Attitude) == 0 {
			t.Errorf("ClassPresetFor(%q): empty Attitude", c)
		}
	}
}

// ── TimerBarText ──────────────────────────────────────────────────────────────

func TestTimerBarText_ColorsAndFill(t *testing.T) {
	// 0% → all empty, green
	s := TimerBarText(0, 20, 10)
	if !strings.Contains(s, "[green:black]") {
		t.Errorf("expected green at 0%%, got %q", s)
	}
	if strings.Contains(s, "█") {
		t.Errorf("expected no filled blocks at 0%%, got %q", s)
	}

	// 50% → half filled, switches to yellow/orange
	s = TimerBarText(0.5, 10, 10)
	if !strings.Contains(s, "[yellow:black]") {
		t.Errorf("expected yellow at 50%%, got %q", s)
	}

	// 80% → red
	s = TimerBarText(0.8, 4, 10)
	if !strings.Contains(s, "[red:black]") {
		t.Errorf("expected red at 80%%, got %q", s)
	}

	// 100% → all filled, red, clamps at barWidth
	s = TimerBarText(1.0, 0, 10)
	if strings.Contains(s, "░") {
		t.Errorf("expected no empty blocks at 100%%, got %q", s)
	}
}

func TestTimerBarText_ShowsRemaining(t *testing.T) {
	s := TimerBarText(0.3, 14, 20)
	if !strings.Contains(s, "14s") {
		t.Errorf("expected remaining seconds in output, got %q", s)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

