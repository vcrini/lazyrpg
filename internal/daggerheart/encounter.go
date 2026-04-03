package daggerheart

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type EncounterEntry struct {
	Monster      Monster
	Seq          int
	Wounds       int
	BasePF       int
	Stress       int
	BaseStress   int
	RankModified bool
	Conditions   map[string]int
}

type encounterConditionDef struct {
	Code        string
	Name        string
	Symbol      string
	Description string
}

var encounterConditionDefs = []encounterConditionDef{
	{Code: "VU", Name: "Vulnerabile", Symbol: "🙃", Description: "Gli attacchi contro questo personaggio hanno vantaggio (tira due volte, prendi il risultato più alto)."},
	{Code: "TR", Name: "Trattenuto", Symbol: "🪤", Description: "Non può muoversi. Può tentare di liberarsi spendendo un'azione (tiro Forza o Istinto a seconda del caso)."},
	{Code: "ST", Name: "Stordito", Symbol: "😵", Description: "Non può intraprendere azioni nel proprio turno."},
	{Code: "NA", Name: "Nascosto", Symbol: "🫥", Description: "Non può essere bersagliato dagli attacchi. Il primo attacco effettuato da Nascosto ha vantaggio."},
	{Code: "SP", Name: "Spaventato", Symbol: "😨", Description: "Svantaggio alle azioni che coinvolgono la fonte della paura."},
	{Code: "AV", Name: "Avvelenato", Symbol: "🤢", Description: "Segna Stress all'inizio di ogni turno."},
	{Code: "SE", Name: "Segnato", Symbol: "🎯", Description: "L'attaccante che ti ha Segnato ha vantaggio ai tiri contro di te."},
	{Code: "PR", Name: "Prono", Symbol: "⬇️", Description: "Svantaggio ai tiri in mischia; bersaglio con copertura contro attacchi a distanza."},
}

type encounterConditionState struct {
	Code   string
	Rounds int
}

func cloneStringIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func orderedEncounterConditions(conditions map[string]int) []encounterConditionState {
	if len(conditions) == 0 {
		return nil
	}
	out := make([]encounterConditionState, 0, len(conditions))
	seen := map[string]struct{}{}
	for _, d := range encounterConditionDefs {
		if n, ok := conditions[d.Code]; ok && n > 0 {
			out = append(out, encounterConditionState{Code: d.Code, Rounds: n})
			seen[d.Code] = struct{}{}
		}
	}
	extra := make([]string, 0)
	extraRounds := map[string]int{}
	for code, rounds := range conditions {
		norm := strings.ToUpper(strings.TrimSpace(code))
		if rounds <= 0 || norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		extra = append(extra, norm)
		extraRounds[norm] = rounds
	}
	sort.Strings(extra)
	for _, code := range extra {
		out = append(out, encounterConditionState{Code: code, Rounds: extraRounds[code]})
	}
	return out
}

func encounterConditionsBadge(entry EncounterEntry) string {
	if len(entry.Conditions) == 0 {
		return ""
	}
	parts := make([]string, 0, len(entry.Conditions))
	for _, d := range encounterConditionDefs {
		if n := entry.Conditions[d.Code]; n > 0 {
			sym := d.Symbol
			if sym == "" {
				sym = d.Code
			}
			parts = append(parts, sym+strconv.Itoa(n))
		}
	}
	if len(parts) == 0 {
		keys := make([]string, 0, len(entry.Conditions))
		for k := range entry.Conditions {
			keys = append(keys, strings.ToUpper(k))
		}
		sort.Strings(keys)
		for _, k := range keys {
			if n := entry.Conditions[k]; n > 0 {
				parts = append(parts, k+strconv.Itoa(n))
			}
		}
	}
	return strings.Join(parts, "")
}

func encounterConditionsLong(entry EncounterEntry) string {
	ordered := orderedEncounterConditions(entry.Conditions)
	if len(ordered) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ordered))
	for _, p := range ordered {
		sym := conditionSymbolByCode(p.Code)
		name := conditionNameByCode(p.Code)
		if sym == "" {
			sym = p.Code
		}
		parts = append(parts, sym+strconv.Itoa(p.Rounds)+" "+name)
	}
	return strings.Join(parts, ", ")
}

func conditionNameByCode(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	for _, d := range encounterConditionDefs {
		if d.Code == c {
			return d.Name
		}
	}
	return c
}

func conditionSymbolByCode(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	for _, d := range encounterConditionDefs {
		if d.Code == c {
			return d.Symbol
		}
	}
	return ""
}

func encounterConditionEffectsLong(entry EncounterEntry) string {
	ordered := orderedEncounterConditions(entry.Conditions)
	if len(ordered) == 0 {
		return ""
	}
	lines := make([]string, 0, len(ordered))
	seen := map[string]struct{}{}
	for _, p := range ordered {
		code := strings.ToUpper(strings.TrimSpace(p.Code))
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		var effect string
		for _, d := range encounterConditionDefs {
			if d.Code == code {
				effect = d.Name + ": " + d.Description
				break
			}
		}
		if effect == "" {
			effect = code + ": effetto non codificato."
		}
		lines = append(lines, "- "+effect)
	}
	return strings.Join(lines, "\n")
}

type encounterPersistEntry struct {
	Name           string         `yaml:"name"`
	Seq            int            `yaml:"seq,omitempty"`
	Wounds         int            `yaml:"wounds"`
	PF             int            `yaml:"pf"`
	Stress         int            `yaml:"stress,omitempty"`
	BaseStress     int            `yaml:"base_stress,omitempty"`
	Rank           int            `yaml:"rank,omitempty"`
	Difficulty     int            `yaml:"difficulty,omitempty"`
	ThresholdMajor int            `yaml:"threshold_major,omitempty"`
	ThresholdGrave int            `yaml:"threshold_grave,omitempty"`
	Damage         string         `yaml:"damage,omitempty"`
	AttackBonus    string         `yaml:"attack_bonus,omitempty"`
	Conditions     map[string]int `yaml:"conditions,omitempty"`
}

type encounterPersist struct {
	Entries []encounterPersistEntry `yaml:"entries"`
}

func nextEncounterSeq(entries []EncounterEntry, name string) int {
	maxSeq := 0
	fallbackCount := 0
	for _, e := range entries {
		if !strings.EqualFold(strings.TrimSpace(e.Monster.Name), strings.TrimSpace(name)) {
			continue
		}
		fallbackCount++
		if e.Seq > maxSeq {
			maxSeq = e.Seq
		}
	}
	if maxSeq > 0 {
		return maxSeq + 1
	}
	return fallbackCount + 1
}

func loadEncounter(path string, monsters []Monster) ([]EncounterEntry, error) {
	rawEntries, err := readEncounter(path)
	if err != nil {
		return nil, err
	}
	if len(rawEntries) == 0 {
		return []EncounterEntry{}, nil
	}

	byName := make(map[string]Monster, len(monsters))
	for _, m := range monsters {
		byName[m.Name] = m
	}

	var entries []EncounterEntry
	assigned := map[string]int{}
	for _, e := range rawEntries {
		name := e.Name
		stress := e.Stress
		baseStress := e.BaseStress
		seq := e.Seq
		if seq <= 0 {
			assigned[name]++
			seq = assigned[name]
		} else if seq > assigned[name] {
			assigned[name] = seq
		}
		var entry EncounterEntry
		if mon, ok := byName[name]; ok {
			if baseStress == 0 {
				baseStress = mon.Stress
			}
			if stress == 0 && baseStress > 0 && e.BaseStress == 0 {
				// Backward compatibility for old files without stress fields.
				stress = baseStress
			}
			entry = EncounterEntry{Monster: mon, Seq: seq, Wounds: e.Wounds, BasePF: e.PF, Stress: stress, BaseStress: baseStress, Conditions: cloneStringIntMap(e.Conditions)}
		} else {
			entry = EncounterEntry{Monster: Monster{Name: name, PF: e.PF, Stress: baseStress}, Seq: seq, Wounds: e.Wounds, BasePF: e.PF, Stress: stress, BaseStress: baseStress, Conditions: cloneStringIntMap(e.Conditions)}
		}
		// Apply rank overrides if persisted.
		if e.Rank > 0 {
			entry.Monster.Rank = e.Rank
			entry.RankModified = true
		}
		if e.Difficulty > 0 {
			entry.Monster.Difficulty = e.Difficulty
		}
		if e.ThresholdMajor > 0 || e.ThresholdGrave > 0 {
			entry.Monster.Thresholds.Values = []int{e.ThresholdMajor, e.ThresholdGrave}
		}
		if e.Damage != "" {
			entry.Monster.Attack.Damage = e.Damage
		}
		if e.AttackBonus != "" {
			entry.Monster.Attack.Bonus = e.AttackBonus
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func saveEncounter(path string, entries []encounterPersistEntry) error {
	payload := encounterPersist{Entries: entries}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readEncounter(path string) ([]encounterPersistEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []encounterPersistEntry{}, nil
		}
		return nil, err
	}
	var payload encounterPersist
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Entries == nil {
		return []encounterPersistEntry{}, nil
	}
	return payload.Entries, nil
}
