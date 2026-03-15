package swade

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/vcrini/lazyrpg/internal/common"
	"gopkg.in/yaml.v3"
)

//go:embed config/names.yml config/mostri.yml config/equipaggiamento.yml config/classi.yml config/razze.yml config/svantaggi.yml config/vantaggi.yml config/tratti.yml config/regole_combattimento.yml
var embeddedConfigFS embed.FS

var dataFile = persistentPath("pngs.yml")
var namesFile = "config/names.yml"
var monstersFile = "config/mostri.yml"
var equipmentFile = "config/equipaggiamento.yml"
var classesFile = "config/classi.yml"
var swadeRulesFiles = []string{
	"config/razze.yml",
	"config/svantaggi.yml",
	"config/vantaggi.yml",
	"config/tratti.yml",
	"config/regole_combattimento.yml",
}
var encounterFile = persistentPath("encounter.yml")
var diceHistoryFile = persistentPath("dice_history.yml")

// nameLists is an alias kept for backward compatibility within this package.
type nameLists = common.NameLists

var namesCache common.NameLists
var namesLoaded bool

// Thresholds is re-exported from common.
type Thresholds = common.Thresholds

type Monster struct {
	Source             string     `yaml:"source,omitempty"`
	Name               string     `yaml:"name"`
	Role               string     `yaml:"role"`
	WildCard           bool       `yaml:"wild_card"`
	Size               int        `yaml:"size"`
	Pace               string     `yaml:"pace"`
	Skills             []string   `yaml:"skills"`
	Rank               int        `yaml:"rank"`
	Description        string     `yaml:"description"`
	MotivationsTactics string     `yaml:"motivations_tactics"`
	Parry              string     `yaml:"parry"`
	Toughness          string     `yaml:"toughness"`
	WoundsMax          int        `yaml:"wounds_max"`
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
}

// Environment is re-exported from common.
type Environment = common.Environment

type EquipmentItem struct {
	Source         string `yaml:"source,omitempty"`
	Name           string `yaml:"name"`
	Category       string `yaml:"category"`
	Type           string `yaml:"type"`
	Rank           int    `yaml:"rank"`
	Era            string `yaml:"era"`
	Cost           string `yaml:"cost"`
	Weight         string `yaml:"weight"`
	MinStrength    string `yaml:"min_strength"`
	AP             string `yaml:"ap"`
	ROF            string `yaml:"rof"`
	Shots          string `yaml:"shots"`
	Armor          string `yaml:"armor"`
	Parry          string `yaml:"parry"`
	Cover          string `yaml:"cover"`
	Note           string `yaml:"note"`
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

// PNG rappresenta la struttura dati per un PNG.
type PNG struct {
	Name        string `json:"Name" yaml:"name"`
	Class       string `json:"Class,omitempty" yaml:"class,omitempty"`
	Subclass    string `json:"Subclass,omitempty" yaml:"subclass,omitempty"`
	Level       int    `json:"Level,omitempty" yaml:"level,omitempty"`
	Rank        int    `json:"Rank,omitempty" yaml:"rank,omitempty"`
	CompBonus   int    `json:"CompBonus,omitempty" yaml:"comp_bonus,omitempty"`
	ExpBonus    int    `json:"ExpBonus,omitempty" yaml:"exp_bonus,omitempty"`
	Description string `json:"Description,omitempty" yaml:"description,omitempty"`
	Traits      string `json:"Traits,omitempty" yaml:"traits,omitempty"`
	Primary     string `json:"Primary,omitempty" yaml:"primary,omitempty"`
	Secondary   string `json:"Secondary,omitempty" yaml:"secondary,omitempty"`
	Armor       string `json:"Armor,omitempty" yaml:"armor,omitempty"`
	Look        string `json:"Look,omitempty" yaml:"look,omitempty"`
	Inventory   string `json:"Inventory,omitempty" yaml:"inventory,omitempty"`
	Token       int    `json:"Token,omitempty" yaml:"token,omitempty"`
}

func (p *PNG) UnmarshalJSON(data []byte) error {
	var aux struct {
		Name         string `json:"Name"`
		Token        *int   `json:"Token"`
		Counter      *int   `json:"Counter"`
		TokenLower   *int   `json:"token"`
		CounterLower *int   `json:"counter"`
		Class        string `json:"Class"`
		ClassLower   string `json:"class"`
		Subclass     string `json:"Subclass"`
		SubclassLow  string `json:"subclass"`
		Level        *int   `json:"Level"`
		LevelLower   *int   `json:"level"`
		Rank         *int   `json:"Rank"`
		RankLower    *int   `json:"rank"`
		CompBonus    *int   `json:"CompBonus"`
		CompBonusLow *int   `json:"comp_bonus"`
		ExpBonus     *int   `json:"ExpBonus"`
		ExpBonusLow  *int   `json:"exp_bonus"`
		Description  string `json:"Description"`
		DescLower    string `json:"description"`
		Traits       string `json:"Traits"`
		TraitsLower  string `json:"traits"`
		Primary      string `json:"Primary"`
		PrimaryLower string `json:"primary"`
		Secondary    string `json:"Secondary"`
		SecondLower  string `json:"secondary"`
		Armor        string `json:"Armor"`
		ArmorLower   string `json:"armor"`
		Look         string `json:"Look"`
		LookLower    string `json:"look"`
		Inventory    string `json:"Inventory"`
		InvLower     string `json:"inventory"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.Name = aux.Name
	if aux.Token != nil {
		p.Token = *aux.Token
	} else if aux.TokenLower != nil {
		p.Token = *aux.TokenLower
	} else if aux.Counter != nil {
		p.Token = *aux.Counter
	} else if aux.CounterLower != nil {
		p.Token = *aux.CounterLower
	}
	if strings.TrimSpace(aux.Class) != "" {
		p.Class = strings.TrimSpace(aux.Class)
	} else {
		p.Class = strings.TrimSpace(aux.ClassLower)
	}
	if strings.TrimSpace(aux.Subclass) != "" {
		p.Subclass = strings.TrimSpace(aux.Subclass)
	} else {
		p.Subclass = strings.TrimSpace(aux.SubclassLow)
	}
	if aux.Level != nil {
		p.Level = *aux.Level
	} else if aux.LevelLower != nil {
		p.Level = *aux.LevelLower
	}
	if aux.Rank != nil {
		p.Rank = *aux.Rank
	} else if aux.RankLower != nil {
		p.Rank = *aux.RankLower
	}
	if aux.CompBonus != nil {
		p.CompBonus = *aux.CompBonus
	} else if aux.CompBonusLow != nil {
		p.CompBonus = *aux.CompBonusLow
	}
	if aux.ExpBonus != nil {
		p.ExpBonus = *aux.ExpBonus
	} else if aux.ExpBonusLow != nil {
		p.ExpBonus = *aux.ExpBonusLow
	}
	if strings.TrimSpace(aux.Description) != "" {
		p.Description = strings.TrimSpace(aux.Description)
	} else {
		p.Description = strings.TrimSpace(aux.DescLower)
	}
	if strings.TrimSpace(aux.Traits) != "" {
		p.Traits = strings.TrimSpace(aux.Traits)
	} else {
		p.Traits = strings.TrimSpace(aux.TraitsLower)
	}
	if strings.TrimSpace(aux.Primary) != "" {
		p.Primary = strings.TrimSpace(aux.Primary)
	} else {
		p.Primary = strings.TrimSpace(aux.PrimaryLower)
	}
	if strings.TrimSpace(aux.Secondary) != "" {
		p.Secondary = strings.TrimSpace(aux.Secondary)
	} else {
		p.Secondary = strings.TrimSpace(aux.SecondLower)
	}
	if strings.TrimSpace(aux.Armor) != "" {
		p.Armor = strings.TrimSpace(aux.Armor)
	} else {
		p.Armor = strings.TrimSpace(aux.ArmorLower)
	}
	if strings.TrimSpace(aux.Look) != "" {
		p.Look = strings.TrimSpace(aux.Look)
	} else {
		p.Look = strings.TrimSpace(aux.LookLower)
	}
	if strings.TrimSpace(aux.Inventory) != "" {
		p.Inventory = strings.TrimSpace(aux.Inventory)
	} else {
		p.Inventory = strings.TrimSpace(aux.InvLower)
	}
	return nil
}

func (p PNG) MarshalJSON() ([]byte, error) {
	out := struct {
		Name        string `json:"Name"`
		Class       string `json:"Class,omitempty"`
		Subclass    string `json:"Subclass,omitempty"`
		Level       int    `json:"Level,omitempty"`
		Rank        int    `json:"Rank,omitempty"`
		CompBonus   int    `json:"CompBonus,omitempty"`
		ExpBonus    int    `json:"ExpBonus,omitempty"`
		Description string `json:"Description,omitempty"`
		Traits      string `json:"Traits,omitempty"`
		Primary     string `json:"Primary,omitempty"`
		Secondary   string `json:"Secondary,omitempty"`
		Armor       string `json:"Armor,omitempty"`
		Look        string `json:"Look,omitempty"`
		Inventory   string `json:"Inventory,omitempty"`
	}{
		Name:        p.Name,
		Class:       strings.TrimSpace(p.Class),
		Subclass:    strings.TrimSpace(p.Subclass),
		Level:       p.Level,
		Rank:        p.Rank,
		CompBonus:   p.CompBonus,
		ExpBonus:    p.ExpBonus,
		Description: strings.TrimSpace(p.Description),
		Traits:      strings.TrimSpace(p.Traits),
		Primary:     strings.TrimSpace(p.Primary),
		Secondary:   strings.TrimSpace(p.Secondary),
		Armor:       strings.TrimSpace(p.Armor),
		Look:        strings.TrimSpace(p.Look),
		Inventory:   strings.TrimSpace(p.Inventory),
	}
	return json.Marshal(out)
}

func randomPNGName() string {
	return common.RandomPNGName(loadNameListsCached())
}

func capitalizeWord(s string) string {
	return common.CapitalizeWord(s)
}

func readData(path string) ([]byte, error) {
	return common.ReadData(path, embeddedConfigFS)
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
	return monsters, nil
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

func loadClasses(path string) ([]ClassItem, error) {
	data, err := readData(path)
	if err != nil {
		return nil, err
	}

	var classes []ClassItem
	if err := yaml.Unmarshal(data, &classes); err != nil {
		return nil, err
	}
	for _, extra := range swadeRulesFiles {
		extraData, err := readData(extra)
		if err != nil {
			continue
		}
		var extraItems []ClassItem
		if err := yaml.Unmarshal(extraData, &extraItems); err != nil {
			continue
		}
		classes = append(classes, extraItems...)
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

func lazyswAppDir() string {
	if p := strings.TrimSpace(os.Getenv("LAZYSW_HOME")); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "."
	}
	return filepath.Join(home, ".lazyrpg", "swade")
}

func persistentPath(name string) string {
	return filepath.Join(lazyswAppDir(), name)
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func readPersistentFileWithFallback(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	legacyPath := filepath.Base(path)
	if legacyPath == "" || legacyPath == "." || legacyPath == path {
		return nil, err
	}
	legacyData, legacyErr := os.ReadFile(legacyPath)
	if legacyErr == nil {
		return legacyData, nil
	}
	return nil, err
}

func loadPNGList(path string) ([]PNG, string, error) {
	data, err := readPersistentFileWithFallback(path)
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
	if err := yaml.Unmarshal(data, &wrapper); err == nil && wrapper.PNGs != nil {
		return wrapper.PNGs, wrapper.Selected, nil
	}

	// Legacy JSON support
	var legacyWrapper struct {
		PNGs     []PNG  `json:"pngs"`
		Selected string `json:"selected"`
	}
	if err := json.Unmarshal(data, &legacyWrapper); err == nil && legacyWrapper.PNGs != nil {
		return legacyWrapper.PNGs, legacyWrapper.Selected, nil
	}

	var legacy []PNG
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, "", err
	}
	return legacy, "", nil
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
	if err := ensureParentDir(path); err != nil {
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

// campaignDir returns the folder for a named campaign under ~/.lazysw/.
func campaignDir(name string) string {
	return filepath.Join(lazyswAppDir(), name)
}

// listCampaigns returns all campaign directory names under ~/.lazysw/.
func listCampaigns() ([]string, error) {
	base := lazyswAppDir()
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// renameCampaign renames a campaign folder.
func renameCampaign(oldName, newName string) error {
	return os.Rename(campaignDir(oldName), campaignDir(newName))
}

// deleteCampaign removes a campaign folder and all its contents.
func deleteCampaign(name string) error {
	return os.RemoveAll(campaignDir(name))
}

func loadDiceHistory(path string) ([]common.DiceResult, error) {
	data, err := readPersistentFileWithFallback(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []common.DiceResult{}, nil
		}
		return nil, err
	}
	var payload struct {
		Entries []common.DiceResult `yaml:"entries"`
	}
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Entries == nil {
		return []common.DiceResult{}, nil
	}
	if len(payload.Entries) > 200 {
		payload.Entries = payload.Entries[len(payload.Entries)-200:]
	}
	return payload.Entries, nil
}

func saveDiceHistory(path string, entries []common.DiceResult) error {
	if len(entries) > 200 {
		entries = entries[len(entries)-200:]
	}
	payload := struct {
		Entries []common.DiceResult `yaml:"entries"`
	}{
		Entries: entries,
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	if err := ensureParentDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
