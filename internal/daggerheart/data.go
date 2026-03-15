package daggerheart

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vcrini/lazyrpg/internal/common"
	"gopkg.in/yaml.v3"
)

const (
	defaultToken = 3
	minToken     = 0 // Il valore minimo
	maxToken     = 3 // Il valore massimo
)

var dataFile = "pngs.yml"
var namesFile = "config/names.yml"
var monstersFile = "config/mostri.yml"
var environmentsFile = "config/ambienti.yml"
var equipmentFile = "config/equipaggiamento.yml"
var cardsFile = "config/carte.yml"
var classesFile = "config/classi.yml"
var encounterFile = "encounter.yml"
var fearStateFile = "state.yml"
var notesFile = "notes.yml"
var appStateDir = ""

//go:embed config/names.yml config/mostri.yml config/ambienti.yml config/equipaggiamento.yml config/carte.yml config/classi.yml
var embeddedConfigFS embed.FS

// nameLists is an alias kept for backward compatibility within this package.
type nameLists = common.NameLists

var namesCache common.NameLists
var namesLoaded bool

func initStoragePaths() error {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return fmt.Errorf("impossibile risolvere HOME: %w", err)
	}
	appStateDir = filepath.Join(home, ".lazyrpg", "daggerheart")
	if err := os.MkdirAll(appStateDir, 0o755); err != nil {
		return fmt.Errorf("impossibile creare dir stato %s: %w", appStateDir, err)
	}
	dataFile = filepath.Join(appStateDir, "pngs.yml")
	encounterFile = filepath.Join(appStateDir, "encounter.yml")
	fearStateFile = filepath.Join(appStateDir, "state.yml")
	notesFile = filepath.Join(appStateDir, "notes.yml")
	return nil
}

func readData(path string) ([]byte, error) {
	return common.ReadData(path, embeddedConfigFS)
}

// Thresholds is re-exported from common.
type Thresholds = common.Thresholds

type Monster struct {
	Name               string     `yaml:"name"`
	Role               string     `yaml:"role"`
	Rank               int        `yaml:"rank"`
	Description        string     `yaml:"description"`
	MotivationsTactics string     `yaml:"motivations_tactics"`
	Difficulty         int        `yaml:"difficulty"`
	Thresholds         Thresholds `yaml:"thresholds"`
	PF                 int        `yaml:"pf"`
	Stress             int        `yaml:"stress"`
	Attack             struct {
		Bonus      string `yaml:"bonus"`
		Name       string `yaml:"name"`
		Range      string `yaml:"range"`
		Damage     string `yaml:"damage"`
		DamageType string `yaml:"damage_type"`
	} `yaml:"attack"`
	Traits []struct {
		Name string `yaml:"name"`
		Kind string `yaml:"kind"`
		Text string `yaml:"text"`
	} `yaml:"traits"`
	Source string `yaml:"source,omitempty"`
}

// Environment is re-exported from common.
type Environment = common.Environment

type EquipmentItem struct {
	Name           string `yaml:"name"`
	Category       string `yaml:"category"`
	Type           string `yaml:"type"`
	Rank           int    `yaml:"rank"`
	Levels         string `yaml:"levels"`
	Trait          string `yaml:"trait"`
	Range          string `yaml:"range"`
	Damage         string `yaml:"damage"`
	Grip           string `yaml:"grip"`
	Characteristic string `yaml:"characteristic"`
}

// CardItem is re-exported from common.
type CardItem = common.CardItem

// ClassItem is re-exported from common.
type ClassItem = common.ClassItem

// PNG rappresenta la struttura dati per un PNG con il suo token.
type PNG struct {
	Name              string `yaml:"name"`
	Token             int    `yaml:"token"`
	PF                int    `yaml:"pf,omitempty"`
	Stress            int    `yaml:"stress,omitempty"`
	ArmorScore        int    `yaml:"armor_score,omitempty"`
	ArmorMinThreshold int    `yaml:"armor_min_threshold,omitempty"`
	ArmorMaxThreshold int    `yaml:"armor_max_threshold,omitempty"`
	Hope              int    `yaml:"hope,omitempty"`
	Class             string `yaml:"class,omitempty"`
	Subclass          string `yaml:"subclass,omitempty"`
	Level             int    `yaml:"level,omitempty"`
	Rank              int    `yaml:"rank,omitempty"`
	CompBonus         int    `yaml:"comp_bonus,omitempty"`
	ExpBonus          int    `yaml:"exp_bonus,omitempty"`
	Description       string `yaml:"description,omitempty"`
	Traits            string `yaml:"traits,omitempty"`
	Primary           string `yaml:"primary,omitempty"`
	Secondary         string `yaml:"secondary,omitempty"`
	Armor             string `yaml:"armor,omitempty"`
	Look              string `yaml:"look,omitempty"`
	Inventory         string `yaml:"inventory,omitempty"`
}

func randomPNGName() string {
	return common.RandomPNGName(loadNameListsCached())
}

func capitalizeWord(s string) string {
	return common.CapitalizeWord(s)
}

func loadNameListsCached() common.NameLists {
	if namesLoaded {
		if len(namesCache.First) > 0 {
			return namesCache
		}
		return common.NameLists{First: []string{"Unknown"}}
	}
	namesLoaded = true
	lists, _ := common.LoadNameLists(readData, namesFile)
	namesCache = lists
	return namesCache
}

func defaultNameLists() common.NameLists {
	return common.DefaultNameLists()
}

func loadMonsters(path string) ([]Monster, error) {
	data, err := readData(path)
	if err != nil {
		return nil, err
	}

	var monsters []Monster
	if err := yaml.Unmarshal(data, &monsters); err != nil {
		return nil, err
	}
	for i := range monsters {
		monsters[i].Name = sanitizeMonsterText(monsters[i].Name)
		monsters[i].Role = sanitizeMonsterText(monsters[i].Role)
		monsters[i].Description = sanitizeMonsterText(monsters[i].Description)
		monsters[i].MotivationsTactics = sanitizeMonsterText(monsters[i].MotivationsTactics)
		monsters[i].Attack.Name = sanitizeMonsterText(monsters[i].Attack.Name)
		monsters[i].Attack.Range = sanitizeMonsterText(monsters[i].Attack.Range)
		monsters[i].Attack.Damage = sanitizeMonsterText(monsters[i].Attack.Damage)
		monsters[i].Attack.DamageType = sanitizeMonsterText(monsters[i].Attack.DamageType)
		for j := range monsters[i].Traits {
			monsters[i].Traits[j].Name = sanitizeMonsterText(monsters[i].Traits[j].Name)
			monsters[i].Traits[j].Kind = sanitizeMonsterText(monsters[i].Traits[j].Kind)
			monsters[i].Traits[j].Text = sanitizeMonsterText(monsters[i].Traits[j].Text)
		}
	}
	return monsters, nil
}

func sanitizeMonsterText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"Seguace(4)", "Seguace (4)",
		"AttaccoinMassa", "Attacco in Massa",
		"IlGrovigliovienesconfitto", "Il Groviglio viene sconfitto",
		"ilgrovigliovienesconfitto", "il Groviglio viene sconfitto",
		"SpendeteunaPaura", "Spendete una Paura",
		"spendeteunaPaura", "spendete una Paura",
		"MarcateunoStress", "Marcate uno Stress",
		"marcateunoStress", "marcate uno Stress",
		"dannifisici", "danni fisici",
		"dannimagici", "danni magici",
		" untiro ", " un tiro ",
	)
	s = replacer.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

type fearPersist struct {
	Paure int `yaml:"paure"`
}

func loadFearState(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var st fearPersist
	if err := yaml.Unmarshal(data, &st); err != nil {
		return 0, err
	}
	return clampFear(st.Paure), nil
}

func saveFearState(path string, paure int) error {
	payload := fearPersist{Paure: clampFear(paure)}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func clampFear(v int) int {
	if v < 0 {
		return 0
	}
	if v > 12 {
		return 12
	}
	return v
}

type notesPersist struct {
	Notes []string `yaml:"notes"`
}

func loadNotes(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload notesPersist
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Notes == nil {
		return []string{}, nil
	}
	return payload.Notes, nil
}

func saveNotes(path string, notes []string) error {
	payload := notesPersist{Notes: notes}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func loadEnvironments(path string) ([]Environment, error) {
	data, err := readData(path)
	if err != nil {
		return nil, err
	}

	var environments []Environment
	if err := yaml.Unmarshal(data, &environments); err != nil {
		return nil, err
	}
	return environments, nil
}

func loadEquipment(path string) ([]EquipmentItem, error) {
	data, err := readData(path)
	if err != nil {
		return nil, err
	}

	var items []EquipmentItem
	if err := yaml.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func loadCards(path string) ([]CardItem, error) {
	data, err := readData(path)
	if err != nil {
		return nil, err
	}

	var cards []CardItem
	if err := yaml.Unmarshal(data, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

func loadClasses(path string) ([]ClassItem, error) {
	data, err := readData(path)
	if err != nil {
		return nil, err
	}

	var classes []ClassItem
	if err := yaml.Unmarshal(data, &classes); err != nil {
		return nil, err
	}
	return classes, nil
}

func uniqueRandomPNGName(existing []PNG) string {
	names := make([]string, len(existing))
	for i, p := range existing {
		names[i] = p.Name
	}
	return common.UniqueRandomPNGName(names, loadNameListsCached())
}

func loadPNGList(path string) ([]PNG, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []PNG{}, "", nil
		}
		return nil, "", err
	}

	var wrapper struct {
		PNGs     []PNG  `yaml:"pngs"`
		Selected string `yaml:"selected"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, "", err
	}
	return wrapper.PNGs, wrapper.Selected, nil
}

func savePNGList(path string, pngs []PNG, selected string) error {
	payload := struct {
		PNGs     []PNG  `yaml:"pngs"`
		Selected string `yaml:"selected"`
	}{
		PNGs:     pngs,
		Selected: selected,
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func selectedPNGName(pngs []PNG, idx int) string {
	if idx < 0 || idx >= len(pngs) {
		return ""
	}
	return pngs[idx].Name
}
