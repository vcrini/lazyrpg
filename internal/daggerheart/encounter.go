package daggerheart

import (
	"os"
	"path/filepath"
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
}

type encounterPersistEntry struct {
	Name           string `yaml:"name"`
	Seq            int    `yaml:"seq,omitempty"`
	Wounds         int    `yaml:"wounds"`
	PF             int    `yaml:"pf"`
	Stress         int    `yaml:"stress,omitempty"`
	BaseStress     int    `yaml:"base_stress,omitempty"`
	Rank           int    `yaml:"rank,omitempty"`
	Difficulty     int    `yaml:"difficulty,omitempty"`
	ThresholdMajor int    `yaml:"threshold_major,omitempty"`
	ThresholdGrave int    `yaml:"threshold_grave,omitempty"`
	Damage         string `yaml:"damage,omitempty"`
	AttackBonus    string `yaml:"attack_bonus,omitempty"`
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
			entry = EncounterEntry{Monster: mon, Seq: seq, Wounds: e.Wounds, BasePF: e.PF, Stress: stress, BaseStress: baseStress}
		} else {
			entry = EncounterEntry{Monster: Monster{Name: name, PF: e.PF, Stress: baseStress}, Seq: seq, Wounds: e.Wounds, BasePF: e.PF, Stress: stress, BaseStress: baseStress}
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
