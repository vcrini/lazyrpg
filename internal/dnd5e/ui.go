package dnd5e

import (
	crand "crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vcrini/diceroll"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/vcrini/lazyrpg/internal/common"
	"gopkg.in/yaml.v3"
)

const (
	helpTextBase          = " [black:gold]?[-:-] help "
	defaultAppDirName     = ".lazyrpg/dnd5e"
	defaultEncountersFile = "encounters.yaml"
	lastEncountersFile    = ".encounters_last_path"
	defaultDiceFile       = "dice.yaml"
	lastDiceFile          = ".dice_last_path"
	defaultRandomFile     = "random.yaml"
	lastRandomFile        = ".random_last_path"
	defaultBuildFile      = "character_build.yaml"
	lastBuildFile         = ".character_build_last_path"
	filtersStateFile      = ".filters_state.yaml"
	descScrollStateFile   = ".description_scroll.yaml"
	defaultNotesFile      = "notes.yml"
)

var (
	helpText       = helpTextBase
	appVersion     = "dev"
	randomNPCNames = []string{
		"Marwen Holt",
		"Ilyra Voss",
		"Bramm Tallow",
		"Sera Nym",
		"Tavik Rime",
		"Dorian Pike",
	}
)

//go:embed config/en/mostri.yml
var embeddedMonstersYAML []byte

//go:embed config/en/oggetti.yml
var embeddedItemsYAML []byte

//go:embed config/en/incantesimi.yml
var embeddedSpellsYAML []byte

//go:embed config/en/classi.yml
var embeddedClassesYAML []byte

//go:embed config/en/sottoclassi.yml
var embeddedSubclassesYAML []byte

//go:embed config/en/class_feature_details.yml
var embeddedClassFeatureDetailsYAML []byte

//go:embed config/en/razze.yml
var embeddedRacesYAML []byte

//go:embed config/en/talenti.yml
var embeddedFeatsYAML []byte

//go:embed config/en/libri.yml
var embeddedBooksYAML []byte

//go:embed config/en/avventure.yml
var embeddedAdventuresYAML []byte

type BrowseMode int

const (
	BrowseMonsters BrowseMode = iota
	BrowseItems
	BrowseSpells
	BrowseCharacters
	BrowseRaces
	BrowseFeats
	BrowseBooks
	BrowseAdventures
	BrowseRandom
	BrowseNotes
)

type Note struct {
	Title   string `yaml:"title"`
	Content string `yaml:"content"`
}

type Monster struct {
	ID          int
	Name        string
	CR          string
	Environment []string
	Source      string
	Type        string
	Raw         map[string]any
}

type dataset struct {
	Monsters []map[string]any `yaml:"monsters"`
}

type itemsDataset struct {
	Items []map[string]any `yaml:"items"`
}

type spellsDataset struct {
	Spells []map[string]any `yaml:"spells"`
}

type classesDataset struct {
	Classes []map[string]any `yaml:"classes"`
}

type subclassesDataset struct {
	Features []subclassFeatureRecord `yaml:"features"`
}

type subclassFeatureRecord struct {
	ClassName      string `yaml:"class_name"`
	ClassSource    string `yaml:"class_source"`
	SubclassName   string `yaml:"subclass_name"`
	SubclassSource string `yaml:"subclass_source"`
	Feature        string `yaml:"feature"`
	Level          int    `yaml:"level"`
}

type classFeatureDetailsDataset struct {
	ClassFeatures    []classFeatureDetailRecord    `yaml:"class_features"`
	SubclassFeatures []subclassFeatureDetailRecord `yaml:"subclass_features"`
}

type classFeatureDetailRecord struct {
	ClassName   string `yaml:"class_name"`
	ClassSource string `yaml:"class_source"`
	Feature     string `yaml:"feature"`
	Source      string `yaml:"source"`
	Level       int    `yaml:"level"`
	Entries     any    `yaml:"entries"`
}

type subclassFeatureDetailRecord struct {
	ClassName      string `yaml:"class_name"`
	ClassSource    string `yaml:"class_source"`
	SubclassName   string `yaml:"subclass_name"`
	SubclassSource string `yaml:"subclass_source"`
	Feature        string `yaml:"feature"`
	Source         string `yaml:"source"`
	Level          int    `yaml:"level"`
	Entries        any    `yaml:"entries"`
}

type racesDataset struct {
	Races []map[string]any `yaml:"races"`
}

type featsDataset struct {
	Feats []map[string]any `yaml:"feats"`
}

type booksDataset struct {
	Books []map[string]any `yaml:"books"`
}

type adventuresDataset struct {
	Adventures []map[string]any `yaml:"adventures"`
}

type EncounterEntry struct {
	MonsterIndex     int
	Ordinal          int
	Custom           bool
	CustomName       string
	CustomLevel      int
	CustomInit       int
	CustomAC         string
	CustomPassive    int
	HasCustomPassive bool
	CustomMeta       string
	CustomBody       string
	Conditions       map[string]int
	BaseHP           int
	CurrentHP        int
	TempHP           int
	HPFormula        string
	UseRolledHP      bool
	RolledHP         int
	HasInitRoll      bool
	InitRoll         int
	Character        *CharacterBuild
}

type CharacterBuild struct {
	Name       string                `yaml:"name,omitempty"`
	Race       string                `yaml:"race,omitempty"`
	Classes    []CharacterClassLevel `yaml:"classes,omitempty"`
	BaseScores []int                 `yaml:"base_scores,omitempty"`
	Feats      []string              `yaml:"feats,omitempty"`
	Spells     []string              `yaml:"spells,omitempty"`
}

type CharacterClassLevel struct {
	Name   string `yaml:"name,omitempty"`
	Levels int    `yaml:"levels,omitempty"`
}

type PersistedEncounters struct {
	Version   int                      `yaml:"version"`
	Items     []PersistedEncounterItem `yaml:"items"`
	TurnMode  bool                     `yaml:"turn_mode,omitempty"`
	TurnIndex int                      `yaml:"turn_index,omitempty"`
	TurnRound int                      `yaml:"turn_round,omitempty"`
}

type PersistedEncounterItem struct {
	MonsterID        int             `yaml:"monster_id"`
	Ordinal          int             `yaml:"ordinal"`
	Custom           bool            `yaml:"custom,omitempty"`
	CustomName       string          `yaml:"custom_name,omitempty"`
	CustomLevel      int             `yaml:"custom_level,omitempty"`
	CustomInit       int             `yaml:"custom_init,omitempty"`
	CustomAC         string          `yaml:"custom_ac,omitempty"`
	CustomPassive    int             `yaml:"custom_passive,omitempty"`
	HasCustomPassive bool            `yaml:"has_custom_passive,omitempty"`
	CustomMeta       string          `yaml:"custom_meta,omitempty"`
	CustomBody       string          `yaml:"custom_body,omitempty"`
	Conditions       map[string]int  `yaml:"conditions,omitempty"`
	BaseHP           int             `yaml:"base_hp"`
	CurrentHP        int             `yaml:"current_hp"`
	TempHP           int             `yaml:"temp_hp,omitempty"`
	HPFormula        string          `yaml:"hp_formula,omitempty"`
	UseRolled        bool            `yaml:"use_rolled,omitempty"`
	RolledHP         int             `yaml:"rolled_hp,omitempty"`
	InitRolled       bool            `yaml:"init_rolled,omitempty"`
	InitRoll         int             `yaml:"init_roll,omitempty"`
	Character        *CharacterBuild `yaml:"character,omitempty"`
}

type PersistedDice struct {
	Version int          `yaml:"version"`
	Items   []DiceResult `yaml:"items"`
}

type PersistedRandom struct {
	Version int                   `yaml:"version"`
	Items   []PersistedRandomItem `yaml:"items"`
}

type PersistedRandomItem struct {
	Name      string `yaml:"name"`
	Category  string `yaml:"category,omitempty"`
	Generated string `yaml:"generated,omitempty"`
	Content   string `yaml:"content,omitempty"`
}

type PersistedFilterMode struct {
	Name    string   `yaml:"name,omitempty"`
	Env     string   `yaml:"env,omitempty"`
	Sources []string `yaml:"sources,omitempty"`
	CR      string   `yaml:"cr,omitempty"`
	Type    string   `yaml:"type,omitempty"`
}

type PersistedFilters struct {
	Version  int                 `yaml:"version"`
	Active   string              `yaml:"active,omitempty"`
	Monsters PersistedFilterMode `yaml:"monsters,omitempty"`
	Items    PersistedFilterMode `yaml:"items,omitempty"`
	Spells   PersistedFilterMode `yaml:"spells,omitempty"`
	Chars    PersistedFilterMode `yaml:"characters,omitempty"`
	Races    PersistedFilterMode `yaml:"races,omitempty"`
	Feats    PersistedFilterMode `yaml:"feats,omitempty"`
	Books    PersistedFilterMode `yaml:"books,omitempty"`
	Advs     PersistedFilterMode `yaml:"adventures,omitempty"`
	Random   PersistedFilterMode `yaml:"random,omitempty"`
}

type PersistedDescriptionScroll struct {
	Version int            `yaml:"version"`
	Offsets map[string]int `yaml:"offsets"`
}

type EncounterUndoState struct {
	Items    []EncounterEntry
	Serial   map[int]int
	Selected int
}

type DiceResult struct {
	Expression string `yaml:"expression"`
	Output     string `yaml:"output"`
}

type DiceUndoState struct {
	Items    []DiceResult
	Selected int
}

type treasureOutcome struct {
	Kind      string
	Band      string
	D100      int
	Coins     map[string]int
	Breakdown []string
	Extras    []string
}

type crBenchmark struct {
	CR     string
	AC     int
	HPMin  int
	HPMax  int
	Atk    int
	DPRMin int
	DPRMax int
	SaveDC int
}

type monsterScalePreview struct {
	BaseCR     string
	TargetCR   string
	Step       int
	BaseAC     int
	TargetAC   int
	BaseHP     int
	TargetHP   int
	BaseDPRMin int
	BaseDPRMax int
	TargetAtk  int
	TargetDC   int
	DPRMin     int
	DPRMax     int
	DamageMul  float64
}

type encounterConditionDef struct {
	Code string
	Name string
}

type skillDef struct {
	Name    string
	Ability string
}

type saveDef struct {
	Name string
	Key  string
}

type encounterXPThreshold struct {
	Easy   int
	Medium int
	Hard   int
	Deadly int
}

type encounterGenerationPreview struct {
	PartySize   int
	PartyLevel  int
	Environment string
	Count       int
	Power       int
	BudgetXP    int
	TargetXP    int
	Multiplier  float64
	MonsterIDs  []int
}

var encounterConditionDefs = []encounterConditionDef{
	{Code: "B", Name: "Blinded"},
	{Code: "C", Name: "Charmed"},
	{Code: "D", Name: "Deafened"},
	{Code: "E", Name: "Exhausted"},
	{Code: "F", Name: "Frightened"},
	{Code: "G", Name: "Grappled"},
	{Code: "I", Name: "Incapacitated"},
	{Code: "V", Name: "Invisible"},
	{Code: "A", Name: "Paralyzed"},
	{Code: "T", Name: "Petrified"},
	{Code: "O", Name: "Poisoned"},
	{Code: "P", Name: "Prone"},
	{Code: "R", Name: "Restrained"},
	{Code: "S", Name: "Stunned"},
	{Code: "U", Name: "Unconscious"},
}

var skillDefs = []skillDef{
	{Name: "Acrobatics", Ability: "dex"},
	{Name: "Animal Handling", Ability: "wis"},
	{Name: "Arcana", Ability: "int"},
	{Name: "Athletics", Ability: "str"},
	{Name: "Deception", Ability: "cha"},
	{Name: "History", Ability: "int"},
	{Name: "Insight", Ability: "wis"},
	{Name: "Intimidation", Ability: "cha"},
	{Name: "Investigation", Ability: "int"},
	{Name: "Medicine", Ability: "wis"},
	{Name: "Nature", Ability: "int"},
	{Name: "Perception", Ability: "wis"},
	{Name: "Performance", Ability: "cha"},
	{Name: "Persuasion", Ability: "cha"},
	{Name: "Religion", Ability: "int"},
	{Name: "Sleight of Hand", Ability: "dex"},
	{Name: "Stealth", Ability: "dex"},
	{Name: "Survival", Ability: "wis"},
}

var saveDefs = []saveDef{
	{Name: "Strength", Key: "str"},
	{Name: "Dexterity", Key: "dex"},
	{Name: "Constitution", Key: "con"},
	{Name: "Intelligence", Key: "int"},
	{Name: "Wisdom", Key: "wis"},
	{Name: "Charisma", Key: "cha"},
}

var encounterXPThresholdByLevel = map[int]encounterXPThreshold{
	1:  {Easy: 25, Medium: 50, Hard: 75, Deadly: 100},
	2:  {Easy: 50, Medium: 100, Hard: 150, Deadly: 200},
	3:  {Easy: 75, Medium: 150, Hard: 225, Deadly: 400},
	4:  {Easy: 125, Medium: 250, Hard: 375, Deadly: 500},
	5:  {Easy: 250, Medium: 500, Hard: 750, Deadly: 1100},
	6:  {Easy: 300, Medium: 600, Hard: 900, Deadly: 1400},
	7:  {Easy: 350, Medium: 750, Hard: 1100, Deadly: 1700},
	8:  {Easy: 450, Medium: 900, Hard: 1400, Deadly: 2100},
	9:  {Easy: 550, Medium: 1100, Hard: 1600, Deadly: 2400},
	10: {Easy: 600, Medium: 1200, Hard: 1900, Deadly: 2800},
	11: {Easy: 800, Medium: 1600, Hard: 2400, Deadly: 3600},
	12: {Easy: 1000, Medium: 2000, Hard: 3000, Deadly: 4500},
	13: {Easy: 1100, Medium: 2200, Hard: 3400, Deadly: 5100},
	14: {Easy: 1250, Medium: 2500, Hard: 3800, Deadly: 5700},
	15: {Easy: 1400, Medium: 2800, Hard: 4300, Deadly: 6400},
	16: {Easy: 1600, Medium: 3200, Hard: 4800, Deadly: 7200},
	17: {Easy: 2000, Medium: 3900, Hard: 5900, Deadly: 8800},
	18: {Easy: 2100, Medium: 4200, Hard: 6300, Deadly: 9500},
	19: {Easy: 2400, Medium: 4900, Hard: 7300, Deadly: 10900},
	20: {Easy: 2800, Medium: 5700, Hard: 8500, Deadly: 12700},
}

var encounterXPByCR = map[string]int{
	"0":   10,
	"1/8": 25,
	"1/4": 50,
	"1/2": 100,
	"1":   200,
	"2":   450,
	"3":   700,
	"4":   1100,
	"5":   1800,
	"6":   2300,
	"7":   2900,
	"8":   3900,
	"9":   5000,
	"10":  5900,
	"11":  7200,
	"12":  8400,
	"13":  10000,
	"14":  11500,
	"15":  13000,
	"16":  15000,
	"17":  18000,
	"18":  20000,
	"19":  22000,
	"20":  25000,
	"21":  33000,
	"22":  41000,
	"23":  50000,
	"24":  62000,
	"25":  75000,
	"26":  90000,
	"27":  105000,
	"28":  120000,
	"29":  135000,
	"30":  155000,
}

var crBenchmarks = []crBenchmark{
	{CR: "0", AC: 13, HPMin: 1, HPMax: 6, Atk: 3, DPRMin: 0, DPRMax: 1, SaveDC: 13},
	{CR: "1/8", AC: 13, HPMin: 7, HPMax: 35, Atk: 3, DPRMin: 2, DPRMax: 3, SaveDC: 13},
	{CR: "1/4", AC: 13, HPMin: 36, HPMax: 49, Atk: 3, DPRMin: 4, DPRMax: 5, SaveDC: 13},
	{CR: "1/2", AC: 13, HPMin: 50, HPMax: 70, Atk: 3, DPRMin: 6, DPRMax: 8, SaveDC: 13},
	{CR: "1", AC: 13, HPMin: 71, HPMax: 85, Atk: 3, DPRMin: 9, DPRMax: 14, SaveDC: 13},
	{CR: "2", AC: 13, HPMin: 86, HPMax: 100, Atk: 3, DPRMin: 15, DPRMax: 20, SaveDC: 13},
	{CR: "3", AC: 13, HPMin: 101, HPMax: 115, Atk: 4, DPRMin: 21, DPRMax: 26, SaveDC: 13},
	{CR: "4", AC: 14, HPMin: 116, HPMax: 130, Atk: 5, DPRMin: 27, DPRMax: 32, SaveDC: 14},
	{CR: "5", AC: 15, HPMin: 131, HPMax: 145, Atk: 6, DPRMin: 33, DPRMax: 38, SaveDC: 15},
	{CR: "6", AC: 15, HPMin: 146, HPMax: 160, Atk: 6, DPRMin: 39, DPRMax: 44, SaveDC: 15},
	{CR: "7", AC: 15, HPMin: 161, HPMax: 175, Atk: 6, DPRMin: 45, DPRMax: 50, SaveDC: 15},
	{CR: "8", AC: 16, HPMin: 176, HPMax: 190, Atk: 7, DPRMin: 51, DPRMax: 56, SaveDC: 16},
	{CR: "9", AC: 16, HPMin: 191, HPMax: 205, Atk: 7, DPRMin: 57, DPRMax: 62, SaveDC: 16},
	{CR: "10", AC: 17, HPMin: 206, HPMax: 220, Atk: 7, DPRMin: 63, DPRMax: 68, SaveDC: 16},
	{CR: "11", AC: 17, HPMin: 221, HPMax: 235, Atk: 8, DPRMin: 69, DPRMax: 74, SaveDC: 17},
	{CR: "12", AC: 17, HPMin: 236, HPMax: 250, Atk: 8, DPRMin: 75, DPRMax: 80, SaveDC: 18},
	{CR: "13", AC: 18, HPMin: 251, HPMax: 265, Atk: 8, DPRMin: 81, DPRMax: 86, SaveDC: 18},
	{CR: "14", AC: 18, HPMin: 266, HPMax: 280, Atk: 8, DPRMin: 87, DPRMax: 92, SaveDC: 18},
	{CR: "15", AC: 18, HPMin: 281, HPMax: 295, Atk: 8, DPRMin: 93, DPRMax: 98, SaveDC: 18},
	{CR: "16", AC: 18, HPMin: 296, HPMax: 310, Atk: 9, DPRMin: 99, DPRMax: 104, SaveDC: 18},
	{CR: "17", AC: 19, HPMin: 311, HPMax: 325, Atk: 10, DPRMin: 105, DPRMax: 110, SaveDC: 19},
	{CR: "18", AC: 19, HPMin: 326, HPMax: 340, Atk: 10, DPRMin: 111, DPRMax: 116, SaveDC: 19},
	{CR: "19", AC: 19, HPMin: 341, HPMax: 355, Atk: 10, DPRMin: 117, DPRMax: 122, SaveDC: 19},
	{CR: "20", AC: 19, HPMin: 356, HPMax: 400, Atk: 10, DPRMin: 123, DPRMax: 140, SaveDC: 19},
	{CR: "21", AC: 19, HPMin: 401, HPMax: 445, Atk: 11, DPRMin: 141, DPRMax: 158, SaveDC: 20},
	{CR: "22", AC: 19, HPMin: 446, HPMax: 490, Atk: 11, DPRMin: 159, DPRMax: 176, SaveDC: 20},
	{CR: "23", AC: 19, HPMin: 491, HPMax: 535, Atk: 11, DPRMin: 177, DPRMax: 194, SaveDC: 20},
	{CR: "24", AC: 19, HPMin: 536, HPMax: 580, Atk: 12, DPRMin: 195, DPRMax: 212, SaveDC: 21},
	{CR: "25", AC: 19, HPMin: 581, HPMax: 625, Atk: 12, DPRMin: 213, DPRMax: 230, SaveDC: 21},
	{CR: "26", AC: 19, HPMin: 626, HPMax: 670, Atk: 12, DPRMin: 231, DPRMax: 248, SaveDC: 21},
	{CR: "27", AC: 19, HPMin: 671, HPMax: 715, Atk: 13, DPRMin: 249, DPRMax: 266, SaveDC: 22},
	{CR: "28", AC: 19, HPMin: 716, HPMax: 760, Atk: 13, DPRMin: 267, DPRMax: 284, SaveDC: 22},
	{CR: "29", AC: 19, HPMin: 761, HPMax: 805, Atk: 13, DPRMin: 285, DPRMax: 302, SaveDC: 22},
	{CR: "30", AC: 19, HPMin: 806, HPMax: 850, Atk: 14, DPRMin: 303, DPRMax: 320, SaveDC: 23},
}

type UI struct {
	app           *tview.Application
	monsters      []Monster
	items         []Monster
	spells        []Monster
	classes       []Monster
	races         []Monster
	feats         []Monster
	books         []Monster
	adventures    []Monster
	randoms       []Monster
	browseMode    BrowseMode
	filtered      []int
	envOptions    []string
	sourceOptions []string
	crOptions     []string
	typeOptions   []string

	nameFilter    string
	envFilter     string
	sourceFilters map[string]struct{}
	crFilter      string
	typeFilter    string

	nameInput      *tview.InputField
	envDrop        *tview.DropDown
	sourceDrop     *tview.DropDown
	crDrop         *tview.DropDown
	typeDrop       *tview.DropDown
	dice           *tview.List
	encounter      *tview.List
	list           *tview.List
	detailMeta     *tview.TextView
	detailTreasure *tview.TextView
	detailRaw      *tview.TextView
	detailBottom   *tview.Pages
	status         *tview.TextView
	pages          *tview.Pages
	leftPanel      *tview.Flex
	monstersPanel  *tview.Flex
	mainRow        *tview.Flex
	detailPanel    *tview.Flex
	filterHost     *tview.Pages



	focusOrder     []tview.Primitive
	rawText        string
	rawQuery       string
	rawMatchLine   int
	rawMatchOcc    int
	treasureText   string
	diceLog        []DiceResult
	diceRender     bool
	wideFilter     bool
	modeFilters    map[BrowseMode]PersistedFilterMode
	monsterScale   map[int]int
	descScroll     map[string]int
	currentDescKey string
	bookBodyCache  map[string]string
	advBodyCache   map[string]string

	encounterSerial map[int]int
	encounterItems  []EncounterEntry
	encountersPath  string
	dicePath        string
	randomPath      string
	buildPath       string
	notesPath       string
	currentCampaign string

	notes           []Note
	noteEditArea    *tview.TextArea
	encounterUndo   []EncounterUndoState
	encounterRedo   []EncounterUndoState
	encounterYank   *EncounterEntry
	diceUndo        []DiceUndoState
	diceRedo        []DiceUndoState
	turnMode        bool
	turnIndex       int
	turnRound       int

	timer    *common.TurnTimer
	mainFlex *tview.Flex

	helpVisible                 bool
	helpReturnFocus             tview.Primitive
	helpTextView                *tview.TextView
	helpBody                    string
	helpQuery                   string
	helpMatchLine               int
	helpMatchOcc                int
	addCustomVisible            bool
	charCreateVisible           bool
	encounterEditVisible        bool
	encounterGenVisible         bool
	fullscreenActive            bool
	fullscreenTarget            string
	spellShortcutAlt            bool
	updatingSourceDrop          bool
	activeBottomPanel           string
	itemTreasureVisible         bool
	spellTreasureVisible        bool
	skillCheckVisible           bool
	saveCheckVisible            bool
	randomEncounterTableVisible bool
	panelJumpVisible            bool
	panelJumpReturnFocus        tview.Primitive
	diceGotoPending             bool
	campaignLoadVisible         bool
}

// Run is the entry point for the D&D 5e system. It loads data, builds the UI
// and runs the tview application. It blocks until the user quits.
func Run() error {
	helpText = helpTextBase

	encountersPath := readLastEncountersPath()
	dicePath := readLastDicePath()
	randomPath := readLastRandomPath()
	buildPath := readLastBuildPath()

	var (
		monsters []Monster
		items    []Monster
		spells   []Monster
		classes  []Monster
		races    []Monster
		feats    []Monster
		books    []Monster
		advs     []Monster
		envs     []string
		crs      []string
		types    []string
		err      error
	)

	monsters, envs, crs, types, err = loadMonstersFromBytes(embeddedMonstersYAML)
	if err != nil {
		return fmt.Errorf("loading error YAML embedded: %w", err)
	}

	items, _, _, _, err = loadItemsFromBytes(embeddedItemsYAML)
	if err != nil {
		return fmt.Errorf("loading error item YAML embedded: %w", err)
	}
	spells, _, _, _, err = loadSpellsFromBytes(embeddedSpellsYAML)
	if err != nil {
		return fmt.Errorf("loading error spell YAML embedded: %w", err)
	}
	classes, _, _, _, err = loadClassesFromBytes(embeddedClassesYAML)
	if err != nil {
		return fmt.Errorf("loading error class YAML embedded: %w", err)
	}
	races, _, _, _, err = loadRacesFromBytes(embeddedRacesYAML)
	if err != nil {
		return fmt.Errorf("loading error race YAML embedded: %w", err)
	}
	feats, _, _, _, err = loadFeatsFromBytes(embeddedFeatsYAML)
	if err != nil {
		return fmt.Errorf("loading error feat YAML embedded: %w", err)
	}
	books, _, _, _, err = loadBooksFromBytes(embeddedBooksYAML)
	if err != nil {
		return fmt.Errorf("loading error book YAML embedded: %w", err)
	}
	advs, _, _, _, err = loadAdventuresFromBytes(embeddedAdventuresYAML)
	if err != nil {
		return fmt.Errorf("loading error adventure YAML embedded: %w", err)
	}

	ui := newUI(monsters, items, spells, classes, races, feats, books, advs, envs, crs, types, encountersPath, dicePath, randomPath)
	ui.buildPath = buildPath
	settings := common.LoadCampaignSettings(lazy5eAppDir())
	ui.timer = common.NewTurnTimer(settings.TurnTimerSeconds)
	switch settings.LastPanel {
	case "encounter":
		ui.app.SetFocus(ui.encounter)
	case "dice":
		ui.app.SetFocus(ui.dice)
	default:
		if m, ok := browseModeFromString(settings.LastPanel); ok {
			ui.setBrowseMode(m)
		}
	}
	if err := ui.run(); err != nil {
		return err
	}
	switch ui.app.GetFocus() {
	case ui.encounter:
		settings.LastPanel = "encounter"
	case ui.dice:
		settings.LastPanel = "dice"
	default:
		settings.LastPanel = browseModeToString(ui.browseMode)
	}
	_ = common.SaveCampaignSettings(lazy5eAppDir(), settings)
	if err := ui.saveEncounters(); err != nil {
		log.Printf("save error encounters (%s): %v", ui.encountersPath, err)
	}
	if err := ui.saveDiceResults(); err != nil {
		log.Printf("save error dice (%s): %v", ui.dicePath, err)
	}
	if err := ui.saveFilterStates(); err != nil {
		log.Printf("save error filters: %v", err)
	}
	if err := ui.saveDescriptionScrollStates(); err != nil {
		log.Printf("save error posizione description: %v", err)
	}
	return nil
}

func newUI(monsters, items, spells, classes, races, feats, books, advs []Monster, envs, crs, types []string, encountersPath string, dicePath string, randomPath string) *UI {
	setTheme()

	ui := &UI{
		app:               tview.NewApplication().EnableMouse(true),
		monsters:          monsters,
		items:             items,
		spells:            spells,
		classes:           classes,
		races:             races,
		feats:             feats,
		books:             books,
		adventures:        advs,
		randoms:           []Monster{},
		browseMode:        BrowseMonsters,
		sourceFilters:     map[string]struct{}{},
		envOptions:        append([]string{"All"}, envs...),
		sourceOptions:     []string{"All"},
		crOptions:         append([]string{"All"}, crs...),
		typeOptions:       append([]string{"All"}, types...),
		filtered:          make([]int, 0, len(monsters)),
		encounterSerial:   map[int]int{},
		encounterItems:    make([]EncounterEntry, 0, 16),
		encountersPath:    encountersPath,
		dicePath:          dicePath,
		randomPath:        randomPath,
		buildPath:         readLastBuildPath(),
		modeFilters:       map[BrowseMode]PersistedFilterMode{},
		monsterScale:      map[int]int{},
		descScroll:        map[string]int{},
		bookBodyCache:     map[string]string{},
		advBodyCache:      map[string]string{},
		activeBottomPanel: "description",
		rawMatchLine:      -1,
		rawMatchOcc:       -1,
	}

	ui.nameInput = tview.NewInputField().
		SetLabel(" (n) Name ").
		SetFieldWidth(26)
	ui.nameInput.SetLabelColor(tcell.ColorGold)
	ui.nameInput.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.nameInput.SetFieldTextColor(tcell.ColorWhite)
	ui.nameInput.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite))
	ui.nameInput.SetChangedFunc(func(text string) {
		ui.nameFilter = strings.TrimSpace(text)
		ui.applyFilters()
	})
	ui.nameInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter, tcell.KeyEscape:
			ui.app.SetFocus(ui.list)
		}
	})

	ui.envDrop = tview.NewDropDown().
		SetLabel(" (e) Env ")
	ui.envDrop.SetOptions(ui.envOptions, func(option string, _ int) {
		if option == "All" {
			ui.envFilter = ""
		} else {
			ui.envFilter = option
		}
		ui.applyFilters()
		ui.maybeReturnFocusToListFromFilter()
	})
	ui.envDrop.SetLabelColor(tcell.ColorGold)
	ui.envDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.envDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.envDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)
	ui.envDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.app.SetFocus(ui.list)
		}
	})

	ui.sourceDrop = tview.NewDropDown().
		SetLabel(" (s) Source ")
	ui.sourceDrop.SetLabelColor(tcell.ColorGold)
	ui.sourceDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.sourceDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.sourceDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)
	ui.sourceDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.app.SetFocus(ui.list)
		}
	})

	ui.crDrop = tview.NewDropDown().
		SetLabel(" (c) CR ").
		SetOptions(ui.crOptions, func(option string, _ int) {
			if option == "All" {
				ui.crFilter = ""
			} else {
				ui.crFilter = option
			}
			ui.applyFilters()
			ui.maybeReturnFocusToListFromFilter()
		})
	ui.crDrop.SetLabelColor(tcell.ColorGold)
	ui.crDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.crDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.crDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)
	ui.crDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.app.SetFocus(ui.list)
		}
	})

	ui.typeDrop = tview.NewDropDown().
		SetLabel(" (t) Type ").
		SetOptions(ui.typeOptions, func(option string, _ int) {
			if option == "All" {
				ui.typeFilter = ""
			} else {
				ui.typeFilter = option
			}
			ui.applyFilters()
			ui.maybeReturnFocusToListFromFilter()
		})
	ui.typeDrop.SetLabelColor(tcell.ColorGold)
	ui.typeDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.typeDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.typeDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)
	ui.typeDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.app.SetFocus(ui.list)
		}
	})

	ui.list = tview.NewList()
	ui.list.SetBorder(false)
	ui.list.SetMainTextColor(tcell.ColorWhite)
	ui.list.SetSecondaryTextColor(tcell.ColorLightGray)
	ui.list.SetSelectedTextColor(tcell.ColorBlack)
	ui.list.SetSelectedBackgroundColor(tcell.ColorGold)
	ui.list.ShowSecondaryText(false)
	ui.list.SetChangedFunc(func(index int, _, _ string, _ rune) {
		ui.renderDetailByListIndex(index)
	})
	ui.list.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		ui.renderDetailByListIndex(index)
	})

	ui.encounter = tview.NewList()
	ui.encounter.SetBorder(true)
	ui.encounter.SetTitle(" [1]-Encounters ")
	ui.encounter.SetTitleColor(tcell.ColorGold)
	ui.encounter.SetBorderColor(tcell.ColorGold)
	ui.encounter.SetMainTextColor(tcell.ColorWhite)
	ui.encounter.SetSelectedTextColor(tcell.ColorBlack)
	ui.encounter.SetSelectedBackgroundColor(tcell.ColorGold)
	ui.encounter.ShowSecondaryText(false)
	ui.encounter.SetChangedFunc(func(index int, _, _ string, _ rune) {
		ui.renderDetailByEncounterIndex(index)
	})
	ui.encounter.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		ui.renderDetailByEncounterIndex(index)
	})
	ui.encounter.AddItem("No monster in encounter", "", 0, nil)

	ui.dice = tview.NewList()
	ui.dice.SetBorder(true)
	ui.dice.SetTitle(" [0]-Dice ")
	ui.dice.SetTitleColor(tcell.ColorGold)
	ui.dice.SetBorderColor(tcell.ColorGold)
	ui.dice.SetUseStyleTags(true, false)
	ui.dice.SetMainTextColor(tcell.ColorWhite)
	ui.dice.SetSelectedTextColor(tcell.ColorWhite)
	ui.dice.SetSelectedBackgroundColor(tcell.ColorDefault)
	ui.dice.ShowSecondaryText(false)
	ui.dice.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if ui.diceRender {
			return
		}
		ui.renderDiceList()
	})

	ui.detailMeta = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	ui.detailMeta.SetBorder(true)
	ui.detailMeta.SetTitle(" Details ")
	ui.detailMeta.SetTitleColor(tcell.ColorGold)
	ui.detailMeta.SetBorderColor(tcell.ColorGold)
	ui.detailMeta.SetTextColor(tcell.ColorWhite)
	ui.detailMeta.SetWrap(true)

	ui.detailTreasure = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	ui.detailTreasure.SetBorder(true)
	ui.detailTreasure.SetTitle(" Treasure ")
	ui.detailTreasure.SetTitleColor(tcell.ColorGold)
	ui.detailTreasure.SetBorderColor(tcell.ColorGold)
	ui.detailTreasure.SetTextColor(tcell.ColorWhite)
	ui.detailTreasure.SetWrap(true)
	ui.treasureText = "No treasure generated."
	ui.detailTreasure.SetText(ui.treasureText)

	ui.detailRaw = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetRegions(true).
		SetWrap(true).
		SetWordWrap(true)
	ui.detailRaw.SetBorder(true)
	ui.detailRaw.SetTitle(" [3]-Description ")
	ui.detailRaw.SetTitleColor(tcell.ColorGold)
	ui.detailRaw.SetBorderColor(tcell.ColorGold)
	ui.detailRaw.SetTextColor(tcell.ColorWhite)

	ui.detailBottom = tview.NewPages().
		AddPage("description", ui.detailRaw, true, true).
		AddPage("treasure", ui.detailTreasure, true, false)

	ui.detailPanel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.detailMeta, 8, 0, false).
		AddItem(ui.detailBottom, 0, 1, false)

	ui.status = tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText)
	ui.status.SetBackgroundColor(tcell.ColorBlack)

	filterRowSingle := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.nameInput, 0, 4, true).
		AddItem(ui.envDrop, 0, 2, false).
		AddItem(ui.sourceDrop, 0, 2, false).
		AddItem(ui.crDrop, 0, 1, false).
		AddItem(ui.typeDrop, 0, 2, false)

	filterRowTop := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.nameInput, 0, 3, true).
		AddItem(ui.envDrop, 0, 1, false).
		AddItem(ui.sourceDrop, 0, 1, false)

	filterRowBottom := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.crDrop, 0, 1, false).
		AddItem(ui.typeDrop, 0, 2, false)

	filterRow := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(filterRowTop, 1, 0, false).
		AddItem(filterRowBottom, 1, 0, false)

	ui.filterHost = tview.NewPages().
		AddPage("single", filterRowSingle, true, false).
		AddPage("double", filterRow, true, true)

	ui.monstersPanel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.filterHost, 2, 0, true).
		AddItem(ui.list, 0, 1, false)
	ui.monstersPanel.SetBorder(true)
	ui.monstersPanel.SetTitle(" [2]-Monsters ")
	ui.monstersPanel.SetTitleColor(tcell.ColorGold)
	ui.monstersPanel.SetBorderColor(tcell.ColorGold)

	ui.leftPanel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.dice, 7, 0, false).
		AddItem(ui.encounter, 8, 0, false).
		AddItem(ui.monstersPanel, 0, 1, true)

	ui.mainRow = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.leftPanel, 0, 1, false).
		AddItem(ui.detailPanel, 0, 1, false)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.mainRow, 0, 1, false).
		AddItem(ui.status, 1, 0, false)
	ui.mainFlex = root

	ui.pages = tview.NewPages().AddPage("main", root, true, true)
	ui.buildTimerOverlay()
	ui.app.SetRoot(ui.pages, true)
	ui.focusOrder = []tview.Primitive{ui.dice, ui.encounter, ui.nameInput, ui.envDrop, ui.sourceDrop, ui.crDrop, ui.typeDrop, ui.list, ui.detailMeta, ui.detailTreasure, ui.detailRaw}
	ui.app.SetFocus(ui.list)
	ui.modeFilters[BrowseMonsters] = PersistedFilterMode{}
	ui.modeFilters[BrowseItems] = PersistedFilterMode{}
	ui.modeFilters[BrowseSpells] = PersistedFilterMode{}
	ui.modeFilters[BrowseCharacters] = PersistedFilterMode{}
	ui.modeFilters[BrowseRaces] = PersistedFilterMode{}
	ui.modeFilters[BrowseFeats] = PersistedFilterMode{}
	ui.modeFilters[BrowseBooks] = PersistedFilterMode{}
	ui.modeFilters[BrowseAdventures] = PersistedFilterMode{}
	ui.modeFilters[BrowseRandom] = PersistedFilterMode{}
	ui.modeFilters[BrowseNotes] = PersistedFilterMode{}
	if err := ui.loadFilterStates(); err != nil {
		ui.status.SetText(fmt.Sprintf(" [white:red] loading error filters[-:-] %v  %s", err, helpText))
	}
	_ = ui.loadDescriptionScrollStates()
	ui.applyModeFilters(ui.browseMode)
	ui.updateBrowsePanelTitle()
	ui.updateFilterLayout(0)

	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape && ui.timer.Running {
			ui.stopTurnTimer()
			return nil
		}

		focus := ui.app.GetFocus()
		_, focusIsInputField := focus.(*tview.InputField)
		if focus != ui.dice {
			ui.diceGotoPending = false
		}

		if focus == ui.dice && ui.diceGotoPending {
			switch event.Key() {
			case tcell.KeyEscape:
				ui.diceGotoPending = false
				ui.status.SetText(helpText)
				return nil
			case tcell.KeyRune:
				r := event.Rune()
				ui.diceGotoPending = false
				switch {
				case r >= '1' && r <= '9':
					ui.gotoDiceRow(int(r - '0'))
					return nil
				case r == '$':
					ui.gotoLastDiceRow()
					return nil
				case r == '^':
					ui.gotoDiceRow(1)
					return nil
				default:
					return event
				}
			default:
				ui.diceGotoPending = false
				return event
			}
		}

		if ui.noteEditArea != nil && focus == ui.noteEditArea {
			// Note editor: let the TextArea handle all keys itself.
			return event
		}
		if ui.addCustomVisible {
			// While add-custom modal is open, do not process global shortcuts.
			return event
		}
		if ui.charCreateVisible || ui.encounterEditVisible || ui.encounterGenVisible {
			// While character creation modal is open, avoid global hotkeys stealing focus.
			return event
		}

		if ui.helpVisible {
			if ui.pages.HasPage("help-search") {
				// Let the help-search input modal handle Enter/Esc and text.
				return event
			}
			if event.Key() == tcell.KeyEscape ||
				(event.Key() == tcell.KeyRune && (event.Rune() == '?' || event.Rune() == 'q')) {
				ui.closeHelpOverlay()
				return nil
			}
			if event.Key() == tcell.KeyRune && event.Rune() == '/' {
				ui.openHelpSearch()
				return nil
			}
			if ui.app.GetFocus() == ui.helpTextView && event.Key() == tcell.KeyRune && event.Rune() == 'n' {
				ui.repeatHelpSearch(true)
				return nil
			}
			if ui.app.GetFocus() == ui.helpTextView && event.Key() == tcell.KeyRune && event.Rune() == 'N' {
				ui.repeatHelpSearch(false)
				return nil
			}
			// Let the help TextView handle scrolling keys (j/k, arrows, PgUp/PgDn).
			return event
		}
		if ui.panelJumpVisible {
			if event.Key() == tcell.KeyEscape {
				ui.closePanelJumpModal(false)
				return nil
			}
			return event
		}

		if ui.itemTreasureVisible || ui.spellTreasureVisible {
			if event.Key() == tcell.KeyEscape {
				if ui.itemTreasureVisible {
					ui.closeItemTreasureModal()
				}
				if ui.spellTreasureVisible {
					ui.closeSpellTreasureModal()
				}
				return nil
			}
			// While modal is open, do not process global shortcuts (1/2/3/q/...).
			return event
		}
		if ui.campaignLoadVisible {
			if event.Key() == tcell.KeyEscape && !focusIsInputField {
				ui.pages.RemovePage("campaign-load")
				ui.campaignLoadVisible = false
				ui.app.SetFocus(ui.list)
				return nil
			}
			return event
		}
		if ui.skillCheckVisible || ui.saveCheckVisible {
			// While skill check modal is open, do not process global shortcuts (1/2/3/...).
			return event
		}
		if ui.randomEncounterTableVisible {
			// While random encounter table form is open, do not process global shortcuts.
			return event
		}

		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == '?':
			ui.openHelpOverlay(focus)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'G':
			ui.openPanelJumpModal(focus)
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'a':
			ui.openDiceRollInput()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'A':
			ui.rerollAllDiceResults()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyEnter:
			ui.rerollSelectedDiceResult()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'e':
			ui.openDiceReRollInput()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'd':
			ui.deleteSelectedDiceResult()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'D':
			ui.clearDiceResults()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 's':
			ui.openDiceSaveAsInput()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'l':
			ui.openDiceLoadInput()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'g':
			ui.diceGotoPending = true
			ui.status.SetText(fmt.Sprintf(" [black:gold]dice goto[-:-] g# row, g$ last, g^ first (g1 alias)  %s", helpText))
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'f':
			ui.toggleFullscreenForFocus(focus)
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'q':
			ui.saveNotes()
			ui.app.Stop()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == '/':
			if focusIsInputField {
				return event
			}
			if focus == ui.list {
				ui.openRawSearch(ui.list)
				return nil
			}
			if focus == ui.encounter {
				ui.openRawSearch(ui.encounter)
				return nil
			}
			if focus == ui.detailRaw {
				ui.openRawSearch(ui.detailRaw)
				return nil
			}
			ui.app.SetFocus(ui.nameInput)
			return nil
		case focus == ui.detailRaw && event.Key() == tcell.KeyRune && event.Rune() == 'n':
			ui.repeatRawSearch(true)
			return nil
		case focus == ui.detailRaw && event.Key() == tcell.KeyRune && event.Rune() == 'N':
			ui.repeatRawSearch(false)
			return nil
		case event.Key() == tcell.KeyTab:
			ui.focusNext()
			return nil
		case event.Key() == tcell.KeyBacktab:
			ui.focusPrev()
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'X':
			if ui.focusHasBrowseFilters(focus) {
				ui.clearCurrentBrowseFilters()
				return nil
			}
			return event
		case (focus == ui.list || focus == ui.nameInput || focus == ui.envDrop || focus == ui.sourceDrop || focus == ui.crDrop || focus == ui.typeDrop) &&
			event.Key() == tcell.KeyRune && event.Rune() == 'x':
			ui.clearCurrentBrowseFilters()
			return nil
		case focus == ui.sourceDrop && (event.Key() == tcell.KeyEnter || (event.Key() == tcell.KeyRune && event.Rune() == ' ')):
			ui.openSourceMultiSelectModal()
			return nil
		case (focus == ui.envDrop || focus == ui.crDrop || focus == ui.typeDrop) && event.Key() == tcell.KeyEnter:
			ui.app.SetFocus(ui.list)
			return nil
		case event.Key() == tcell.KeyEscape &&
			(focus == ui.list || focus == ui.nameInput || focus == ui.envDrop || focus == ui.sourceDrop || focus == ui.crDrop || focus == ui.typeDrop):
			ui.app.SetFocus(ui.list)
			return nil
		case focus == ui.nameInput && event.Key() == tcell.KeyEscape:
			ui.app.SetFocus(ui.list)
			return nil
		case focus == ui.list && event.Key() == tcell.KeyPgUp:
			if len(ui.filtered) > 0 {
				ui.scrollDetailByPage(-1)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyPgDn:
			if len(ui.filtered) > 0 {
				ui.scrollDetailByPage(1)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'a':
			if ui.browseMode == BrowseMonsters {
				ui.addSelectedMonsterToEncounter()
				return nil
			}
			if ui.browseMode == BrowseCharacters {
				ui.openCreateCharacterFromClassForm()
				return nil
			}
			if ui.browseMode == BrowseNotes {
				ui.openAddNoteModal()
				return nil
			}
			return nil
		case focus == ui.list && ui.browseMode == BrowseNotes && event.Key() == tcell.KeyRune && event.Rune() == 'd':
			ui.deleteNote(ui.list.GetCurrentItem())
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'g':
			ui.generateRandomDungeonRoom()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'y':
			ui.generateRandomDungeonLayout()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'n':
			ui.generateRandomNPC()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'p':
			ui.generateRandomPlace()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'o':
			ui.generateRandomSocialEvent()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 't':
			ui.generateRandomTreasureTheme()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'm':
			ui.generateRandomMagicItemTheme()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'u':
			ui.generateRandomCurrencyTheme()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'e':
			ui.generateRandomAdventureEvent()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'h':
			ui.generateRandomPlotHook()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'i':
			ui.openRandomMonsterEncounterTableForm()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'k':
			ui.generateRandomEquipmentShopTable()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'K':
			ui.generateRandomMagicShopTable()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'd':
			ui.deleteSelectedRandomEntry()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'D':
			ui.clearAllRandomEntries()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'S':
			ui.openRandomSaveAsInput()
			return nil
		case focus == ui.list && ui.browseMode == BrowseRandom && event.Key() == tcell.KeyRune && event.Rune() == 'L':
			ui.openRandomLoadInput()
			return nil
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'n':
			ui.app.SetFocus(ui.nameInput)
			return nil
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'e':
			if ui.browseMode == BrowseNotes {
				ui.openEditNoteModal(ui.list.GetCurrentItem())
				return nil
			}
			if ui.browseMode == BrowseMonsters || ui.browseMode == BrowseItems || ui.browseMode == BrowseSpells || ui.browseMode == BrowseCharacters || ui.browseMode == BrowseRaces || ui.browseMode == BrowseFeats || ui.browseMode == BrowseBooks || ui.browseMode == BrowseAdventures {
				ui.app.SetFocus(ui.envDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'c':
			if ui.browseMode == BrowseMonsters {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			if ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.typeDrop)
				return nil
			}
			if ui.browseMode == BrowseCharacters {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			if ui.browseMode == BrowseRaces || ui.browseMode == BrowseFeats {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			if ui.browseMode == BrowseBooks || ui.browseMode == BrowseAdventures {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 't':
			if ui.browseMode == BrowseMonsters || ui.browseMode == BrowseItems || ui.browseMode == BrowseCharacters || ui.browseMode == BrowseRaces || ui.browseMode == BrowseFeats || ui.browseMode == BrowseBooks || ui.browseMode == BrowseAdventures {
				ui.app.SetFocus(ui.typeDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 's':
			if ui.browseMode == BrowseMonsters {
				ui.app.SetFocus(ui.sourceDrop)
				return nil
			}
			if ui.browseMode == BrowseItems {
				ui.app.SetFocus(ui.sourceDrop)
				return nil
			}
			if ui.browseMode == BrowseCharacters {
				ui.app.SetFocus(ui.sourceDrop)
				return nil
			}
			if ui.browseMode == BrowseRaces || ui.browseMode == BrowseFeats {
				ui.app.SetFocus(ui.sourceDrop)
				return nil
			}
			if ui.browseMode == BrowseBooks || ui.browseMode == BrowseAdventures {
				ui.app.SetFocus(ui.sourceDrop)
				return nil
			}
			if ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.sourceDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'r':
			if ui.browseMode == BrowseItems {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			return event
		case (focus == ui.envDrop || focus == ui.sourceDrop || focus == ui.crDrop || focus == ui.typeDrop) &&
			event.Key() == tcell.KeyRune && event.Rune() == 'r':
			if ui.browseMode == BrowseItems {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			return event
		case (focus == ui.envDrop || focus == ui.sourceDrop || focus == ui.crDrop || focus == ui.typeDrop) &&
			event.Key() == tcell.KeyRune && event.Rune() == 'c':
			if ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.typeDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'm':
			if ui.browseMode == BrowseMonsters {
				ui.openTreasureByCRInput()
				return nil
			}
			if ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'l':
			if ui.browseMode == BrowseMonsters {
				ui.openLairTreasureByCRInput()
				return nil
			}
			if ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'g':
			if ui.browseMode == BrowseItems {
				ui.openItemTreasureInput()
				return nil
			}
			if ui.browseMode == BrowseSpells {
				ui.openSpellTreasureInput()
				return nil
			}
			return event
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'a':
			ui.openAddCustomEncounterForm()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'e':
			ui.openEncounterCharacterEditForm()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'g':
			ui.openEncounterAutoGenerateForm()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'w':
			ui.openCharacterBuildSaveInput()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'o':
			ui.openCharacterBuildLoadInput()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'c':
			ui.openEncounterConditionModal()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'x':
			ui.openEncounterConditionRemoveModal()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'y':
			ui.yankEncounterEntry()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'C':
			ui.clearEncounterConditions()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == '[':
			ui.adjustEncounterConditionRounds(-1)
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == ']':
			ui.adjustEncounterConditionRounds(1)
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyLeft:
			ui.openEncounterHPInput(-1)
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRight:
			ui.openEncounterHPInput(1)
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'h':
			ui.openEncounterHPInput(-1)
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'L':
			ui.openEncounterTempHPInput()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'H':
			ui.clearEncounterTempHP()
			return nil
		case focus == ui.list && ui.browseMode == BrowseMonsters && event.Key() == tcell.KeyLeft:
			ui.adjustSelectedMonsterScale(-1)
			return nil
		case focus == ui.list && ui.browseMode == BrowseMonsters && event.Key() == tcell.KeyRight:
			ui.adjustSelectedMonsterScale(1)
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'd':
			ui.deleteSelectedEncounterEntry()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'D':
			ui.deleteAllMonsterEncounterEntries()
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'd':
			if focus == ui.list || focus == ui.detailMeta || focus == ui.detailTreasure || focus == ui.detailRaw {
				ui.toggleDetailsTreasureFocus()
				return nil
			}
			return event
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 's':
			ui.openEncounterSaveAsInput()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'l':
			ui.openEncounterLoadInput()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'i':
			ui.rollEncounterInitiative()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'I':
			ui.rollAllEncounterInitiative()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'K':
			ui.openEncounterSkillCheckModal()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'V':
			ui.openEncounterSaveCheckModal()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'S':
			ui.sortEncounterByInitiative()
			return nil
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'S':
			if ui.browseMode == BrowseItems {
				ui.openTreasureSaveAsInput()
				return nil
			}
			return event
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == '*':
			ui.toggleEncounterTurnMode()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'n':
			ui.nextEncounterTurn()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'p':
			ui.pasteEncounterEntry()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == ' ':
			ui.toggleEncounterHPMode()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 't':
			idx := ui.encounter.GetCurrentItem()
			if idx >= 0 && idx < len(ui.encounterItems) {
				e := ui.encounterItems[idx]
				if e.Custom || e.Character != nil {
					ui.startTurnTimer()
				}
			}
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'u':
			if focus == ui.dice {
				ui.undoDiceCommand()
			} else {
				ui.undoEncounterCommand()
			}
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'r':
			if focus == ui.dice {
				ui.redoDiceCommand()
			} else {
				ui.redoEncounterCommand()
			}
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '1':
			ui.app.SetFocus(ui.encounter)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '2':
			ui.app.SetFocus(ui.list)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '3':
			if focus == ui.detailRaw {
				ui.activeBottomPanel = "treasure"
				ui.detailBottom.SwitchToPage("treasure")
				ui.app.SetFocus(ui.detailTreasure)
			} else if focus == ui.detailTreasure {
				ui.activeBottomPanel = "description"
				ui.detailBottom.SwitchToPage("description")
				ui.app.SetFocus(ui.detailRaw)
			} else {
				if ui.activeBottomPanel == "treasure" {
					ui.detailBottom.SwitchToPage("treasure")
					ui.app.SetFocus(ui.detailTreasure)
				} else {
					ui.detailBottom.SwitchToPage("description")
					ui.app.SetFocus(ui.detailRaw)
				}
			}
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '0':
			ui.app.SetFocus(ui.dice)
			return nil
		case event.Key() == tcell.KeyCtrlS && !focusIsInputField:
			ui.openCampaignSaveInput()
			return nil
		case event.Key() == tcell.KeyCtrlO && !focusIsInputField:
			ui.openCampaignLoadModal()
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '4':
			ui.setBrowseMode(BrowseMonsters)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '5':
			ui.setBrowseMode(BrowseItems)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '6':
			ui.setBrowseMode(BrowseSpells)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '7':
			ui.setBrowseMode(BrowseCharacters)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '8':
			ui.setBrowseMode(BrowseRaces)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '9':
			ui.setBrowseMode(BrowseFeats)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'b':
			ui.setBrowseMode(BrowseBooks)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'v':
			ui.setBrowseMode(BrowseAdventures)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'z':
			if focus == ui.encounter && ui.turnMode {
				ui.centerEncounterTurnItem()
				return nil
			}
			ui.setBrowseMode(BrowseRandom)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'N':
			ui.setBrowseMode(BrowseNotes)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '[':
			ui.cycleBrowseMode(-1)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == ']':
			ui.cycleBrowseMode(1)
			return nil
		case focus == ui.detailTreasure && event.Key() == tcell.KeyRune && event.Rune() == 'D':
			ui.treasureText = "No treasure generated."
			ui.detailTreasure.SetText(ui.treasureText)
			ui.detailTreasure.ScrollToBeginning()
			return nil
		case focus != ui.nameInput && event.Key() == tcell.KeyRune && event.Rune() == 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case focus != ui.nameInput && event.Key() == tcell.KeyRune && event.Rune() == 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		default:
			return event
		}
	})
	var dragEnabled bool
	ui.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		if !dragEnabled {
			screen.EnableMouse(tcell.MouseDragEvents)
			dragEnabled = true
		}
		w, _ := screen.Size()
		ui.updateFilterLayout(w)
		return false
	})

	ui.applyFilters()
	if err := ui.loadEncounters(); err != nil {
		ui.status.SetText(fmt.Sprintf(" [white:red] loading error encounters[-:-] %v  %s", err, helpText))
	}
	if err := ui.loadDiceResults(); err != nil {
		ui.status.SetText(fmt.Sprintf(" [white:red] loading error dice[-:-] %v  %s", err, helpText))
	}
	ui.loadNotes()
	ui.renderEncounterList()
	ui.setupDividerResize()
	return ui
}

func (ui *UI) setupDividerResize() {
	type vRow struct {
		flex  *tview.Flex
		items []tview.Primitive
	}
	vRows := []vRow{
		{ui.leftPanel, []tview.Primitive{ui.dice, ui.encounter, ui.monstersPanel}},
		{ui.detailPanel, []tview.Primitive{ui.detailMeta, ui.detailBottom}},
	}

	var hDragging bool
	var vFlex *tview.Flex
	var vTopItem tview.Primitive
	var vItems []tview.Primitive

	// Returning nil as the event sets consumed=true in tview and triggers a.draw().
	ui.app.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		col, row := event.Position()

		switch action {
		case tview.MouseLeftDown:
			lx, _, lw, _ := ui.leftPanel.GetRect()
			if col == lx+lw-1 || col == lx+lw {
				hDragging = true
				return nil, action
			}
			for _, vr := range vRows {
				for i := 0; i < len(vr.items)-1; i++ {
					_, iy, _, ih := vr.items[i].GetRect()
					b := iy + ih
					if row == b-1 || row == b {
						vFlex = vr.flex
						vTopItem = vr.items[i]
						vItems = vr.items
						return nil, action
					}
				}
			}

		case tview.MouseMove:
			if hDragging {
				lx, _, _, _ := ui.mainRow.GetRect()
				_, _, totalW, _ := ui.mainRow.GetRect()
				newW := col - lx
				if newW < 10 {
					newW = 10
				}
				if newW > totalW-10 {
					newW = totalW - 10
				}
				ui.mainRow.ResizeItem(ui.leftPanel, newW, 0)
				ui.mainRow.ResizeItem(ui.detailPanel, 0, 1)
				return nil, action
			}
			if vFlex != nil {
				_, topY, _, _ := vTopItem.GetRect()
				newH := row - topY
				if newH < 2 {
					newH = 2
				}
				topIdx := -1
				for i, item := range vItems {
					if item == vTopItem {
						topIdx = i
						break
					}
				}
				if topIdx >= 0 {
					for i := 0; i < topIdx; i++ {
						_, _, _, h := vItems[i].GetRect()
						vFlex.ResizeItem(vItems[i], h, 0)
					}
					vFlex.ResizeItem(vTopItem, newH, 0)
					for i := topIdx + 1; i < len(vItems); i++ {
						vFlex.ResizeItem(vItems[i], 0, 1)
					}
				}
				return nil, action
			}

		case tview.MouseLeftUp:
			if hDragging {
				hDragging = false
				return nil, action
			}
			if vFlex != nil {
				vFlex = nil
				vTopItem = nil
				vItems = nil
				return nil, action
			}
		}
		return event, action
	})
}

func (ui *UI) openHelpOverlay(focus tview.Primitive) {
	ui.helpReturnFocus = focus
	ui.helpVisible = true

	text := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)
	text.SetBorder(true)
	text.SetBorderColor(tcell.ColorGold)
	text.SetTitleColor(tcell.ColorGold)
	text.SetTitle(fmt.Sprintf(" Help - %s ", ui.panelNameForFocus(focus)))
	text.SetText(ui.helpForFocus(focus))

	helpBody := ui.helpForFocus(focus) + "\n[gray]Scroll: j/k, arrows, PgUp/PgDn   Search: / then n/N[-]"
	text.SetText(helpBody)
	ui.helpTextView = text
	ui.helpBody = helpBody
	ui.helpMatchLine = -1
	ui.helpMatchOcc = -1
	ui.renderHelpWithHighlight("", -1, -1)

	// Bigger modal so panel-specific shortcuts are not clipped on common terminal sizes.
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(text, 22, 0, true).
			AddItem(nil, 0, 1, false), 92, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("help-overlay", modal, true, true)
	ui.app.SetFocus(text)
}

func (ui *UI) closeHelpOverlay() {
	ui.pages.RemovePage("help-search")
	ui.pages.RemovePage("help-overlay")
	ui.helpVisible = false
	ui.helpTextView = nil
	ui.helpBody = ""
	ui.helpQuery = ""
	ui.helpMatchLine = -1
	ui.helpMatchOcc = -1
	if ui.helpReturnFocus != nil {
		ui.app.SetFocus(ui.helpReturnFocus)
	} else {
		ui.app.SetFocus(ui.list)
	}
}

func (ui *UI) closePanelJumpModal(apply bool) {
	ui.pages.RemovePage("panel-jump")
	ui.panelJumpVisible = false
	if !apply {
		if ui.panelJumpReturnFocus != nil {
			ui.app.SetFocus(ui.panelJumpReturnFocus)
		} else {
			ui.app.SetFocus(ui.list)
		}
	}
	ui.panelJumpReturnFocus = nil
}

func (ui *UI) renderHelpWithHighlight(query string, lineToHighlight int, occToHighlight int) {
	if ui.helpTextView == nil || ui.helpBody == "" {
		return
	}
	lines := strings.Split(ui.helpBody, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if query != "" && i == lineToHighlight {
			b.WriteString(highlightRawOccurrence(line, query, occToHighlight))
		} else {
			b.WriteString(line)
		}
	}
	ui.helpTextView.SetText(b.String())
}

func highlightRawOccurrence(line, query string, occToHighlight int) string {
	if query == "" {
		return line
	}
	lowerLine := strings.ToLower(line)
	lowerQuery := strings.ToLower(query)
	var b strings.Builder
	start := 0
	occ := 0
	for {
		idx := strings.Index(lowerLine[start:], lowerQuery)
		if idx < 0 {
			b.WriteString(line[start:])
			break
		}
		abs := start + idx
		end := abs + len(query)
		b.WriteString(line[start:abs])
		if occToHighlight < 0 || occ == occToHighlight {
			b.WriteString("[black:gold]")
			b.WriteString(line[abs:end])
			b.WriteString("[-:-]")
		} else {
			b.WriteString(line[abs:end])
		}
		start = end
		occ++
		if start >= len(line) {
			break
		}
	}
	return b.String()
}

func lineMatchCount(line, query string) int {
	if query == "" {
		return 0
	}
	lowerLine := strings.ToLower(line)
	lowerQuery := strings.ToLower(query)
	if lowerQuery == "" {
		return 0
	}
	count := 0
	start := 0
	for {
		idx := strings.Index(lowerLine[start:], lowerQuery)
		if idx < 0 {
			return count
		}
		count++
		start += idx + len(query)
		if start >= len(line) {
			return count
		}
	}
}

func findNextOccurrenceInText(text string, query string, startLine int, startOcc int, forward bool) (int, int, bool) {
	if strings.TrimSpace(query) == "" || text == "" {
		return 0, 0, false
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return 0, 0, false
	}
	if startLine < -1 {
		startLine = -1
	}
	if startLine > len(lines) {
		startLine = len(lines)
	}
	if startOcc < -1 {
		startOcc = -1
	}
	if forward {
		for l := max(0, startLine); l < len(lines); l++ {
			count := lineMatchCount(lines[l], query)
			if count == 0 {
				continue
			}
			first := 0
			if l == startLine {
				first = startOcc + 1
			}
			if first < 0 {
				first = 0
			}
			if first < count {
				return l, first, true
			}
		}
		for l := 0; l < len(lines); l++ {
			count := lineMatchCount(lines[l], query)
			if count > 0 {
				return l, 0, true
			}
		}
		return 0, 0, false
	}
	if startLine == len(lines) {
		startLine = len(lines) - 1
	}
	for l := min(startLine, len(lines)-1); l >= 0; l-- {
		count := lineMatchCount(lines[l], query)
		if count == 0 {
			continue
		}
		last := count - 1
		if l == startLine && startOcc >= 0 {
			last = startOcc - 1
		}
		if last >= 0 {
			return l, last, true
		}
	}
	for l := len(lines) - 1; l >= 0; l-- {
		count := lineMatchCount(lines[l], query)
		if count > 0 {
			return l, count - 1, true
		}
	}
	return 0, 0, false
}

func (ui *UI) openHelpSearch() {
	if !ui.helpVisible || ui.helpTextView == nil {
		return
	}
	input := tview.NewInputField().
		SetLabel("/ ").
		SetFieldWidth(40)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Find In Help ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	input.SetText(ui.helpQuery)

	closeModal := func() {
		ui.pages.RemovePage("help-search")
		ui.app.SetFocus(ui.helpTextView)
	}
	input.SetDoneFunc(func(key tcell.Key) {
		closeModal()
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		query := strings.TrimSpace(input.GetText())
		if query == "" {
			ui.helpQuery = ""
			ui.helpMatchLine = -1
			ui.helpMatchOcc = -1
			ui.renderHelpWithHighlight("", -1, -1)
			ui.status.SetText(helpText)
			return
		}
		start, _ := ui.helpTextView.GetScrollOffset()
		line, occ, ok := findNextOccurrenceInText(ui.helpBody, query, start, -1, true)
		if !ok {
			ui.helpQuery = query
			ui.helpMatchLine = -1
			ui.helpMatchOcc = -1
			ui.renderHelpWithHighlight(query, -1, -1)
			ui.status.SetText(fmt.Sprintf(" [white:red] no match in Help [-:-] \"%s\"  %s", query, helpText))
			return
		}
		ui.helpQuery = query
		ui.helpMatchLine = line
		ui.helpMatchOcc = occ
		ui.renderHelpWithHighlight(query, line, occ)
		ui.helpTextView.ScrollTo(line, 0)
		ui.status.SetText(fmt.Sprintf(" [black:gold] found in Help[-:-] \"%s\" (linea %d)  %s", query, line+1, helpText))
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 52, 0, true).
		AddItem(nil, 0, 1, false)
	ui.pages.AddPage("help-search", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) repeatHelpSearch(forward bool) {
	query := strings.TrimSpace(ui.helpQuery)
	if query == "" || ui.helpTextView == nil {
		ui.status.SetText(fmt.Sprintf(" [white:red] no active search in Help [-:-]  %s", helpText))
		return
	}
	startLine := ui.helpMatchLine
	startOcc := ui.helpMatchOcc
	if startLine < 0 {
		startLine, _ = ui.helpTextView.GetScrollOffset()
		startOcc = -1
		if !forward {
			startOcc = 0
		}
	}
	line, occ, ok := findNextOccurrenceInText(ui.helpBody, query, startLine, startOcc, forward)
	if !ok {
		ui.helpMatchLine = -1
		ui.helpMatchOcc = -1
		ui.renderHelpWithHighlight(query, -1, -1)
		ui.status.SetText(fmt.Sprintf(" [white:red] no match in Help [-:-] \"%s\"  %s", query, helpText))
		return
	}
	ui.helpMatchLine = line
	ui.helpMatchOcc = occ
	ui.renderHelpWithHighlight(query, line, occ)
	ui.helpTextView.ScrollTo(line, 0)
	ui.status.SetText(fmt.Sprintf(" [black:gold] found in Help[-:-] \"%s\" (linea %d)  %s", query, line+1, helpText))
}

func (ui *UI) openPanelJumpModal(returnFocus tview.Primitive) {
	type jumpTarget struct {
		Label string
		Key   rune
		Hint  string
		Go    func()
	}

	targets := []jumpTarget{
		{Label: "Dice", Key: '0', Hint: "accanto: 1 Encounters", Go: func() { ui.app.SetFocus(ui.dice) }},
		{Label: "Encounters", Key: '1', Hint: "accanto: 0 Dice, 2 Catalog", Go: func() { ui.app.SetFocus(ui.encounter) }},
		{Label: "Catalog", Key: '2', Hint: "accanto: 1 Encounters, 3 Description", Go: func() { ui.app.SetFocus(ui.list) }},
		{Label: "Description", Key: '3', Hint: "accanto: 2 Catalog", Go: func() { ui.app.SetFocus(ui.detailRaw) }},
		{Label: "Monsters", Key: '4', Hint: "accanto: z Random, 5 Items", Go: func() { ui.setBrowseMode(BrowseMonsters); ui.app.SetFocus(ui.list) }},
		{Label: "Items", Key: '5', Hint: "accanto: 4 Monsters, 6 Spells", Go: func() { ui.setBrowseMode(BrowseItems); ui.app.SetFocus(ui.list) }},
		{Label: "Spells", Key: '6', Hint: "accanto: 5 Items, 7 Characters", Go: func() { ui.setBrowseMode(BrowseSpells); ui.app.SetFocus(ui.list) }},
		{Label: "Characters", Key: '7', Hint: "accanto: 6 Spells, 8 Races", Go: func() { ui.setBrowseMode(BrowseCharacters); ui.app.SetFocus(ui.list) }},
		{Label: "Races", Key: '8', Hint: "accanto: 7 Characters, 9 Feats", Go: func() { ui.setBrowseMode(BrowseRaces); ui.app.SetFocus(ui.list) }},
		{Label: "Feats", Key: '9', Hint: "accanto: 8 Races, b Manuals", Go: func() { ui.setBrowseMode(BrowseFeats); ui.app.SetFocus(ui.list) }},
		{Label: "Manuals", Key: 'b', Hint: "accanto: 9 Feats, v Adventures", Go: func() { ui.setBrowseMode(BrowseBooks); ui.app.SetFocus(ui.list) }},
		{Label: "Adventures", Key: 'v', Hint: "accanto: b Manuals, z Random", Go: func() { ui.setBrowseMode(BrowseAdventures); ui.app.SetFocus(ui.list) }},
		{Label: "Random", Key: 'z', Hint: "accanto: v Adventures, 4 Monsters", Go: func() { ui.setBrowseMode(BrowseRandom); ui.app.SetFocus(ui.list) }},
		{Label: "Notes", Key: 'n', Hint: "accanto: z Random", Go: func() { ui.setBrowseMode(BrowseNotes); ui.app.SetFocus(ui.list) }},
	}

	list := tview.NewList().
		ShowSecondaryText(false)
	list.SetBorder(true)
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetTitle(" Panel Jump ")
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)

	detail := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true)
	detail.SetBorder(true)
	detail.SetBorderColor(tcell.ColorGold)
	detail.SetTitleColor(tcell.ColorGold)
	detail.SetTitle(" Description ")

	activate := func(idx int) {
		if idx < 0 || idx >= len(targets) {
			return
		}
		ui.closePanelJumpModal(true)
		targets[idx].Go()
		ui.status.SetText(fmt.Sprintf(" [black:gold]panel jump[-:-] %s (%c)  %s", targets[idx].Label, targets[idx].Key, helpText))
	}

	for i := range targets {
		t := targets[i]
		list.AddItem(fmt.Sprintf("%s [black:gold](%c)[-:-]", t.Label, t.Key), "", 0, nil)
	}
	list.SetChangedFunc(func(idx int, _, _ string, _ rune) {
		if idx < 0 || idx >= len(targets) {
			detail.SetText("")
			return
		}
		t := targets[idx]
		detail.SetText(fmt.Sprintf("[white]Panel:[-] %s [black:gold](%c)[-:-]\n[white]Scorciatoie adiacenti:[-] %s", t.Label, t.Key, t.Hint))
	})
	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		activate(idx)
	})
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.closePanelJumpModal(false)
			return nil
		}
		if event.Key() == tcell.KeyRune {
			r := event.Rune()
			if r >= 'A' && r <= 'Z' {
				r = r - 'A' + 'a'
			}
			for i := range targets {
				if r == targets[i].Key {
					activate(i)
					return nil
				}
			}
		}
		return event
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 16, 0, true).
			AddItem(detail, 5, 0, false).
			AddItem(nil, 0, 1, false), 90, 0, true).
		AddItem(nil, 0, 1, false)

	list.SetCurrentItem(0)
	if len(targets) > 0 {
		t := targets[0]
		detail.SetText(fmt.Sprintf("[white]Panel:[-] %s [black:gold](%c)[-:-]\n[white]Scorciatoie adiacenti:[-] %s", t.Label, t.Key, t.Hint))
	}

	ui.panelJumpReturnFocus = returnFocus
	ui.panelJumpVisible = true
	ui.pages.AddPage("panel-jump", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *UI) panelNameForFocus(focus tview.Primitive) string {
	switch focus {
	case ui.dice:
		return "Dice"
	case ui.encounter:
		return "Encounters"
	case ui.list:
		switch ui.browseMode {
		case BrowseItems:
			return "Items"
		case BrowseSpells:
			return "Spells"
		case BrowseCharacters:
			return "Characters"
		case BrowseRaces:
			return "Races"
		case BrowseFeats:
			return "Feats"
		case BrowseRandom:
			return "Random"
		default:
			return "Monsters"
		}
	case ui.detailRaw:
		return "Description"
	case ui.detailMeta:
		return "Details"
	case ui.detailTreasure:
		return "Treasure"
	case ui.nameInput:
		return "Name Filter"
	case ui.envDrop:
		return "Env Filter"
	case ui.sourceDrop:
		return "Source Filter"
	case ui.crDrop:
		return "CR Filter"
	case ui.typeDrop:
		return "Type Filter"
	default:
		return "Panel"
	}
}

func (ui *UI) helpForFocus(focus tview.Primitive) string {
	header := "[black:gold]Global[-:-]\n" +
		"  ? : open/close this help\n" +
		"  Esc : close help\n" +
		"  q : quit app\n" +
		"  f : fullscreen on/off current panel\n" +
		"  X : clear filters in current browse mode\n" +
		"  Tab / Shift+Tab : change focus\n" +
		"  0 / 1 / 2 / 3 : go to Dice / Encounters / Catalog / Description↔Treasure (toggle)\n" +
		"  G : open panel jump modal (panel + shortcut)\n" +
		"  [ / ] : previous/next browse panel\n" +
		"  4 / 5 / 6 / 7 / 8 / 9 : Monsters / Items / Spells / Characters / Races / Feats\n" +
		"  b / v / z : Manuals / Adventures / Random\n" +
		"  Ctrl+S : save campaign (encounters + dice + treasure)\n" +
		"  Ctrl+O : load campaign from folder\n\n"

	switch focus {
	case ui.dice:
		return header +
			"[black:gold]Dice[-:-]\n" +
			"  a : roll dice expression (e.g. 2d6+d20+1)\n" +
			"  Enter : re-roll selected row\n" +
			"  g# / g$ / g^ : goto row # / last row / first row (g1 alias)\n" +
			"  A : re-roll all rows in history\n" +
			"  e : edit + re-roll selected row\n" +
			"  d : delete selected row\n" +
			"  D : clear all rows\n" +
			"  s : save dice results (save as)\n" +
			"  l : load dice results (load)\n" +
			"  f : fullscreen on/off Dice panel\n" +
			"\n" +
			"[black:gold]Examples[-:-]\n" +
			"  2d6+d20+1\n" +
			"  d20v+5   (v = keep higher of 2 rolls)\n" +
			"  d20s+1   (s = keep lower of 2 rolls)\n" +
			"  d20a+5   (a alias of v)\n" +
			"  d20d+1   (d alias of s)\n" +
			"  (4d8+1:slash)+(3d6:acid)\n" +
			"  d2,d3,d4\n" +
			"  4d10+6d6+5\n" +
			"  1d6 x2\n" +
			"  1d6-1\n" +
			"  1d20+5 > 2\n" +
			"  2d6+d20-1 > 15\n" +
			"  1d20+5 > 10 x3\n"
	case ui.encounter:
		return header +
			"[black:gold]Encounters[-:-]\n" +
			"  j / k (or arrows) : select entry\n" +
			"  / : search in selected monster Description\n" +
			"  a : add custom entry\n" +
			"  e : edit custom character (name + level-up/multiclass)\n" +
			"  g : generate encounter from PCs (preview/edit before apply)\n" +
			"      Enter flow: Name -> Class -> Add Levels -> Apply\n" +
			"  w / o : save/load character build from separate file\n" +
			"  d : delete selected entry\n" +
			"  D : delete all monster entries (keep custom/characters)\n" +
			"  s : save encounter to file (save as)\n" +
			"  l : load encounter from file (load)\n" +
			"  i : roll initiative for selected entry\n" +
			"  I : roll initiative for all entries\n" +
			"  K : roll a skill check (editable bonus)\n" +
			"  V : roll a saving throw vs DC (save type + bonus + DC)\n" +
			"  S : sort entries by initiative roll\n" +
			"  * : toggle turn mode\n" +
			"  n : next turn\n" +
			"  y / p : yank / paste encounter entry\n" +
			"  u : undo last encounter operation\n" +
			"  r : redo undone encounter operation\n" +
			"  c : add/remove conditions (multi select)\n" +
			"  x : remove one condition from entry\n" +
			"  C : clear all conditions from entry\n" +
			"  [ / ] : decrease/increase condition rounds\n" +
			"  L : set/update Temp HP (max rule)\n" +
			"  H : clear Temp HP\n" +
			"  space : switch HP average/formula (roll)\n" +
			"  h / left arrow : subtract HP (Temp HP consumed first)\n" +
			"  right arrow : add HP\n"
	case ui.list:
		if ui.browseMode == BrowseMonsters {
			return header +
				"[black:gold]Monsters[-:-]\n" +
				"  j / k (or arrows) : navigate monsters\n" +
				"  / : search in selected monster Description\n" +
				"  a : add monster to Encounters\n" +
				"  m : generate treasure from CR (5e rules)\n" +
				"  l : generate lair treasure from CR (5e rules)\n" +
				"  x : clear all filters\n" +
				"  left/right arrow : scale monster CR (-/+) using 5e benchmark\n" +
				"  n / e / s / c / t : focus on Name / Env / Source(multi) / CR / Type\n" +
				"  [ / ] : switch Monsters/Items/Spells/Characters/Races/Feats panel\n" +
				"  PgUp / PgDn : scroll Description panel\n"
		}
		if ui.browseMode == BrowseItems {
			return header +
				"[black:gold]Items[-:-]\n" +
				"  j / k (or arrows) : navigate list\n" +
				"  / : search in selected entry Description\n" +
				"  g : generate item treasure (type + quantity)\n" +
				"  S : save Treasure to file\n" +
				"  x : clear all filters\n" +
				"  n / s / r / t : focus on Name / Source(multi) / Rarity / Type\n" +
				"  [ / ] : switch Monsters/Items/Spells/Characters/Races/Feats panel\n" +
				"  PgUp / PgDn : scroll Description panel\n"
		}
		if ui.browseMode == BrowseSpells {
			return header +
				"[black:gold]Spells[-:-]\n" +
				"  j / k (or arrows) : navigate list\n" +
				"  / : search in selected entry Description\n" +
				"  g : generate spells (level + quantity)\n" +
				"  x : clear all filters\n" +
				"  n / s / l / c : focus on Name / Source(multi) / Level / School\n" +
				"  [ / ] : switch Monsters/Items/Spells/Characters/Races/Feats panel\n" +
				"  PgUp / PgDn : scroll Description panel\n"
		}
		if ui.browseMode == BrowseCharacters {
			return header +
				"[black:gold]Characters[-:-]\n" +
				"  j / k (or arrows) : navigate classes\n" +
				"  / : search in selected class Description\n" +
				"  a : create character (level + race)\n" +
				"  x : clear all filters\n" +
				"  n / e / s / c / t : focus on Name / Primary / Source(multi) / Hit Die / Caster\n" +
				"  [ / ] : switch Monsters/Items/Spells/Characters/Races/Feats panel\n" +
				"  PgUp / PgDn : scroll Description panel\n"
		}
		if ui.browseMode == BrowseBooks {
			return header +
				"[black:gold]Manuals[-:-]\n" +
				"  j / k (or arrows) : navigate manuals\n" +
				"  / : search in selected manual Description\n" +
				"  x : clear all filters\n" +
				"  n / e / s / c / t : focus on Name / Group / Source(multi) / Year / Author\n" +
				"  [ / ] : switch Monsters/Items/Spells/Characters/Races/Feats/Manuals/Adventures panel\n" +
				"  PgUp / PgDn : scroll Description panel\n"
		}
		if ui.browseMode == BrowseAdventures {
			return header +
				"[black:gold]Adventures[-:-]\n" +
				"  j / k (or arrows) : navigate adventures\n" +
				"  / : search in selected adventure Description\n" +
				"  x : clear all filters\n" +
				"  n / e / s / c / t : focus on Name / Group / Source(multi) / Year / Author\n" +
				"  [ / ] : switch Monsters/Items/Spells/Characters/Races/Feats/Manuals/Adventures panel\n" +
				"  PgUp / PgDn : scroll Description panel\n"
		}
		if ui.browseMode == BrowseRandom {
			return header +
				"[black:gold]Random[-:-]\n" +
				"  g : dungeon room contents (traps/treasure/monsters/puzzles)\n" +
				"  y : dungeon layout (doors/corridors/hazards)\n" +
				"  n : NPC (name/traits/motivation/profession)\n" +
				"  p : place name (tavern/ship/fortress)\n" +
				"  o : social event / ethnic tension\n" +
				"  t : treasure cache (individual/hoard style)\n" +
				"  m : magic item (by rarity/category)\n" +
				"  u : random currency / trade bars / art objects\n" +
				"  e : adventure event (wilderness/chase/stronghold)\n" +
				"  h : divination / plot hook\n" +
				"  i : random monster encounter table (choose environment + tier)\n" +
				"  k : equipment shop table (from items dataset)\n" +
				"  K : magic shop table (from items dataset)\n" +
				"  d : delete selected random entry\n" +
				"  D : clear all random entries\n" +
				"  S : save random list as\n" +
				"  L : load random list\n" +
				"  x : clear all filters\n"
		}
		if ui.browseMode == BrowseRaces {
			return header +
				"[black:gold]Races[-:-]\n" +
				"  j / k (or arrows) : navigate races\n" +
				"  / : search in selected race Description\n" +
				"  x : clear all filters\n" +
				"  n / e / s / c / t : focus on Name / Ability / Source(multi) / Size / Lineage\n" +
				"  [ / ] : switch Monsters/Items/Spells/Characters/Races/Feats panel\n" +
				"  PgUp / PgDn : scroll Description panel\n"
		}
		if ui.browseMode == BrowseNotes {
			return header +
				"[black:gold]Notes[-:-]\n" +
				"  a : add new note\n" +
				"  e : edit selected note content (Ctrl+S save, Esc cancel)\n" +
				"  d : delete selected note\n" +
				"  j / k (or arrows) : navigate notes\n" +
				"  n : focus on Title filter\n"
		}
		return header +
			"[black:gold]Feats[-:-]\n" +
			"  j / k (or arrows) : navigate feats\n" +
			"  / : search in selected feat Description\n" +
			"  x : clear all filters\n" +
			"  n / e / s / c / t : focus on Name / Prereq / Source(multi) / Category / Ability\n" +
			"  [ / ] : switch Monsters/Items/Spells/Characters/Races/Feats panel\n" +
			"  PgUp / PgDn : scroll Description panel\n"
	case ui.detailRaw:
		return header +
			"[black:gold]Description[-:-]\n" +
			"  / : search text in current Description\n" +
			"  n / N : next / previous search match\n" +
			"  j / k (or arrows) : scroll content\n"
	case ui.detailMeta:
		return header +
			"[black:gold]Details[-:-]\n" +
			"  d : switch focus between Details and Treasure\n" +
			"  j / k (or arrows) : scroll content\n"
	case ui.detailTreasure:
		return header +
			"[black:gold]Treasure[-:-]\n" +
			"  d : switch focus between Details and Treasure\n" +
			"  D : clear treasure content\n" +
			"  j / k (or arrows) : scroll content\n"
	case ui.nameInput:
		return header +
			"[black:gold]Name Filter[-:-]\n" +
			"  type text : filter by name\n" +
			"  x : clear all filters\n" +
			"  Enter / Esc : return to Monsters\n"
	case ui.envDrop, ui.sourceDrop, ui.crDrop, ui.typeDrop:
		return header +
			"[black:gold]Filter Dropdown[-:-]\n" +
			"  arrows / Enter : change filter value\n" +
			"  x : clear all filters\n"
	default:
		return header + "[black:gold]Panel[-:-]\n  No panel-specific shortcut.\n"
	}
}

func (ui *UI) focusNext() {
	current := ui.app.GetFocus()
	for i, p := range ui.focusOrder {
		if p == current {
			ui.app.SetFocus(ui.focusOrder[(i+1)%len(ui.focusOrder)])
			return
		}
	}
	ui.app.SetFocus(ui.list)
}

func (ui *UI) focusPrev() {
	current := ui.app.GetFocus()
	for i, p := range ui.focusOrder {
		if p == current {
			prev := i - 1
			if prev < 0 {
				prev = len(ui.focusOrder) - 1
			}
			ui.app.SetFocus(ui.focusOrder[prev])
			return
		}
	}
	ui.app.SetFocus(ui.list)
}

func (ui *UI) toggleDetailsTreasureFocus() {
	if ui.detailBottom == nil {
		return
	}
	if ui.activeBottomPanel == "treasure" {
		ui.activeBottomPanel = "description"
		ui.detailBottom.SwitchToPage("description")
		ui.app.SetFocus(ui.detailRaw)
		return
	}
	ui.activeBottomPanel = "treasure"
	ui.detailBottom.SwitchToPage("treasure")
	ui.app.SetFocus(ui.detailTreasure)
}

func (ui *UI) scrollDetailByPage(direction int) {
	if direction == 0 {
		return
	}

	_, _, _, height := ui.detailRaw.GetInnerRect()
	if height <= 0 {
		height = 10
	}

	row, _ := ui.detailRaw.GetScrollOffset()
	step := max(height-1, 1)

	nextRow := max(row+(step*direction), 0)
	ui.detailRaw.ScrollTo(nextRow, 0)
}

func (ui *UI) openTreasureByCRInput() {
	input := tview.NewInputField().
		SetLabel("CR: ").
		SetFieldWidth(16)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Generate Treasure (5e Individual) ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	if cr := ui.currentMonsterCR(); cr != "" {
		input.SetText(cr)
	}

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 44, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("treasure-input")
		ui.app.SetFocus(ui.list)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		crText := strings.TrimSpace(input.GetText())
		outcome, err := generateIndividualTreasure(crText, rand.Intn)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid CR[-:-] \"%s\"  %s", crText, helpText))
			return
		}
		ui.renderTreasureOutcome(crText, outcome)
		ui.status.SetText(fmt.Sprintf(" [black:gold]treasure[-:-] generated for CR %s  %s", crText, helpText))
	})

	ui.pages.AddPage("treasure-input", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openLairTreasureByCRInput() {
	input := tview.NewInputField().
		SetLabel("CR: ").
		SetFieldWidth(16)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Generate Lair Treasure (5e Hoard) ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	if cr := ui.currentMonsterCR(); cr != "" {
		input.SetText(cr)
	}

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 46, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("lair-treasure-input")
		ui.app.SetFocus(ui.list)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		crText := strings.TrimSpace(input.GetText())
		outcome, err := generateLairTreasure(crText, rand.Intn)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid CR[-:-] \"%s\"  %s", crText, helpText))
			return
		}
		ui.renderTreasureOutcome(crText, outcome)
		ui.status.SetText(fmt.Sprintf(" [black:gold]lair treasure[-:-] generated for CR %s  %s", crText, helpText))
	})

	ui.pages.AddPage("lair-treasure-input", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) currentMonsterCR() string {
	if ui.browseMode != BrowseMonsters || len(ui.filtered) == 0 {
		return ""
	}
	cur := ui.list.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filtered) {
		return ""
	}
	idx := ui.filtered[cur]
	if idx < 0 || idx >= len(ui.monsters) {
		return ""
	}
	return strings.TrimSpace(ui.monsters[idx].CR)
}

func (ui *UI) openItemTreasureInput() {
	typeOptions := []string{"random", "potion", "scroll", "staff", "wand", "rod", "ring", "weapon", "armor", "wondrous"}
	selectedTypes := map[string]struct{}{"random": {}}

	typeList := tview.NewList()
	typeList.SetBorder(true)
	typeList.SetTitle(" Type (Space=toggle, Enter=Qty) ")
	typeList.SetBorderColor(tcell.ColorGold)
	typeList.SetTitleColor(tcell.ColorGold)
	typeList.SetMainTextColor(tcell.ColorWhite)
	typeList.SetSelectedTextColor(tcell.ColorBlack)
	typeList.SetSelectedBackgroundColor(tcell.ColorGold)
	typeList.ShowSecondaryText(false)

	renderTypes := func() {
		current := typeList.GetCurrentItem()
		typeList.Clear()
		for _, opt := range typeOptions {
			mark := "[ ]"
			if _, ok := selectedTypes[opt]; ok {
				mark = "[x]"
			}
			typeList.AddItem(fmt.Sprintf("%s %s", mark, opt), "", 0, nil)
		}
		if current < 0 {
			current = 0
		}
		if current >= len(typeOptions) {
			current = len(typeOptions) - 1
		}
		typeList.SetCurrentItem(current)
	}

	toggleAt := func(idx int) {
		if idx < 0 || idx >= len(typeOptions) {
			return
		}
		opt := typeOptions[idx]
		if opt == "random" {
			selectedTypes = map[string]struct{}{"random": {}}
			return
		}
		delete(selectedTypes, "random")
		if _, ok := selectedTypes[opt]; ok {
			delete(selectedTypes, opt)
		} else {
			selectedTypes[opt] = struct{}{}
		}
		if len(selectedTypes) == 0 {
			selectedTypes["random"] = struct{}{}
		}
	}

	closeModal := func() {
		ui.closeItemTreasureModal()
	}
	qtyInput := tview.NewInputField().SetLabel(" Qty ").SetFieldWidth(8).SetText("1")
	qtyInput.SetLabelColor(tcell.ColorGold)
	qtyInput.SetFieldBackgroundColor(tcell.ColorWhite)
	qtyInput.SetFieldTextColor(tcell.ColorBlack)
	qtyInput.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	qtyInput.SetBorder(true)
	qtyInput.SetBorderColor(tcell.ColorGold)

	runGenerate := func() {
		count, err := strconv.Atoi(strings.TrimSpace(qtyInput.GetText()))
		if err != nil || count <= 0 {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid quantity[-:-] \"%s\"  %s", qtyInput.GetText(), helpText))
			return
		}
		kinds := keysSorted(selectedTypes)
		items, err := ui.generateItemTreasureByKinds(kinds, count)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] %v[-:-]  %s", err, helpText))
			return
		}
		ui.renderGeneratedItemTreasure(strings.Join(kinds, ","), items)
		ui.status.SetText(fmt.Sprintf(" [black:gold]item treasure[-:-] generati %d item (%s)  %s", len(items), strings.Join(kinds, ","), helpText))
		closeModal()
	}

	typeList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == ' ':
			idx := typeList.GetCurrentItem()
			toggleAt(idx)
			renderTypes()
			return nil
		case event.Key() == tcell.KeyEnter:
			ui.app.SetFocus(qtyInput)
			return nil
		case event.Key() == tcell.KeyEscape:
			closeModal()
			return nil
		default:
			return event
		}
	})
	qtyInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			runGenerate()
		case tcell.KeyEscape:
			closeModal()
		}
	})

	renderTypes()

	modalBody := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(typeList, 11, 0, true).
		AddItem(qtyInput, 3, 0, false)
	modalBody.SetBorder(true)
	modalBody.SetTitle(" Generate Item Treasure ")
	modalBody.SetTitleColor(tcell.ColorGold)
	modalBody.SetBorderColor(tcell.ColorGold)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modalBody, 16, 0, true).
			AddItem(nil, 0, 1, false), 62, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("items-treasure-input", modal, true, true)
	ui.itemTreasureVisible = true
	ui.app.SetFocus(typeList)
}

func (ui *UI) openSpellTreasureInput() {
	levelOptions := []string{"random", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	schoolOptions := []string{"random"}
	schoolSet := map[string]struct{}{}
	for _, sp := range ui.spells {
		if s := strings.TrimSpace(sp.Type); s != "" {
			schoolSet[s] = struct{}{}
		}
	}
	schoolOptions = append(schoolOptions, keysSorted(schoolSet)...)

	level := "random"
	school := "random"
	qty := "1"

	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Generate Spell Treasure ")
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)
	form.SetFieldBackgroundColor(tcell.ColorWhite)
	form.SetFieldTextColor(tcell.ColorBlack)
	form.SetLabelColor(tcell.ColorGold)
	form.AddDropDown("Level", levelOptions, 0, func(option string, _ int) { level = option })
	form.AddDropDown("School", schoolOptions, 0, func(option string, _ int) { school = option })
	form.AddInputField("Qty", qty, 8, nil, func(text string) { qty = text })

	closeModal := func() {
		ui.closeSpellTreasureModal()
	}
	runGenerate := func() {
		count, err := strconv.Atoi(strings.TrimSpace(qty))
		if err != nil || count <= 0 {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid quantity[-:-] \"%s\"  %s", qty, helpText))
			return
		}
		filter := SpellTreasureFilter{
			Level:  level,
			School: school,
		}
		spells, err := ui.generateSpellTreasure(filter, count)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] %v[-:-]  %s", err, helpText))
			return
		}
		ui.renderGeneratedSpellTreasure(filter, spells)
		ui.status.SetText(fmt.Sprintf(" [black:gold]spell treasure[-:-] generate %d spells (level=%s school=%s)  %s", len(spells), filter.Level, filter.School, helpText))
		closeModal()
	}
	form.AddButton("Generate", runGenerate)
	form.AddButton("Cancel", closeModal)
	form.SetCancelFunc(closeModal)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			closeModal()
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			formIdx, btnIdx := form.GetFocusedItemIndex()
			if formIdx == 2 && btnIdx < 0 {
				runGenerate()
				return nil
			}
		}
		return event
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 13, 0, true).
			AddItem(nil, 0, 1, false), 64, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("spells-treasure-input", modal, true, true)
	ui.spellTreasureVisible = true
	ui.app.SetFocus(form)
}

func (ui *UI) openTreasureSaveAsInput() {
	content := strings.TrimSpace(ui.treasureText)
	if content == "" || strings.EqualFold(content, "No treasure generated.") {
		ui.status.SetText(fmt.Sprintf(" [white:red] no Treasure to save[-:-]  %s", helpText))
		return
	}
	defaultName := fmt.Sprintf("tesoro-%s.yaml", newShortUUID())
	input := tview.NewInputField().
		SetLabel("File: ").
		SetFieldWidth(60).
		SetText(defaultName)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBorder(true)
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	input.SetTitle(" Save Treasure As ")

	closeModal := func() {
		ui.pages.RemovePage("treasure-saveas")
		ui.app.SetFocus(ui.list)
	}
	trySave := func(path string, overwrite bool) {
		if err := ui.saveTreasureToPath(path, overwrite); err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] save error treasure[-:-] %v  %s", err, helpText))
			return
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold]treasure saved[-:-] %s  %s", path, helpText))
		closeModal()
	}

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			closeModal()
			return
		}
		path := strings.TrimSpace(input.GetText())
		if path == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid file name[-:-]  %s", helpText))
			return
		}
		if fileExists(path) {
			ui.openTreasureOverwriteConfirm(path, func(confirmed bool) {
				if !confirmed {
					ui.status.SetText(fmt.Sprintf(" [black:gold]save treasure[-:-] canceled (file exists)  %s", helpText))
					return
				}
				trySave(path, true)
			})
			return
		}
		trySave(path, false)
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 76, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("treasure-saveas", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openTreasureOverwriteConfirm(path string, done func(bool)) {
	msg := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true)
	msg.SetBorder(true)
	msg.SetBorderColor(tcell.ColorGold)
	msg.SetTitleColor(tcell.ColorGold)
	msg.SetTitle(" Overwrite Warning ")
	msg.SetText(fmt.Sprintf("Il file esiste gia:\n[white]%s[-]\n\nSovrascrivere? [black:gold]y[-:-]/[black:gold]n[-:-]", path))

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(msg, 8, 0, true).
			AddItem(nil, 0, 1, false), 76, 0, true).
		AddItem(nil, 0, 1, false)

	msg.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || (event.Key() == tcell.KeyRune && (event.Rune() == 'n' || event.Rune() == 'N')) {
			ui.pages.RemovePage("treasure-overwrite")
			done(false)
			return nil
		}
		if event.Key() == tcell.KeyRune && (event.Rune() == 'y' || event.Rune() == 'Y') {
			ui.pages.RemovePage("treasure-overwrite")
			done(true)
			return nil
		}
		return event
	})

	ui.pages.AddPage("treasure-overwrite", modal, true, true)
	ui.app.SetFocus(msg)
}

func (ui *UI) saveTreasureToPath(path string, overwrite bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	if !overwrite && fileExists(path) {
		return errors.New("file already exists")
	}
	content := strings.TrimSpace(ui.treasureText)
	if content == "" {
		return errors.New("empty treasure content")
	}
	return os.WriteFile(path, []byte(content+"\n"), 0o644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func appendTreasureText(existing, newText string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" || strings.EqualFold(existing, "no treasure generated.") {
		return newText
	}
	return existing + "\n\n─────────\n\n" + newText
}

func newShortUUID() string {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	hexv := hex.EncodeToString(b[:])
	if len(hexv) < 16 {
		return hexv
	}
	return hexv[:16]
}

type SpellTreasureFilter struct {
	Level  string
	School string
}

func (ui *UI) generateSpellTreasure(filter SpellTreasureFilter, count int) ([]Monster, error) {
	if count < 1 {
		return nil, errors.New("quantity must be >= 1")
	}
	candidates := filterSpellsByFilter(ui.spells, filter)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no spell found for level=%q school=%q", filter.Level, filter.School)
	}
	out := make([]Monster, 0, count)
	for range count {
		idx := rand.Intn(len(candidates))
		out = append(out, candidates[idx])
	}
	return out, nil
}

func filterSpellsByFilter(spells []Monster, filter SpellTreasureFilter) []Monster {
	level := strings.TrimSpace(strings.ToLower(filter.Level))
	school := strings.TrimSpace(strings.ToLower(filter.School))
	out := make([]Monster, 0, len(spells))
	for _, sp := range spells {
		if level != "" && level != "random" && !strings.EqualFold(strings.TrimSpace(sp.CR), level) {
			continue
		}
		if school != "" && school != "random" && !strings.EqualFold(strings.TrimSpace(sp.Type), school) {
			continue
		}
		out = append(out, sp)
	}
	return out
}

func filterSpellsByLevel(spells []Monster, level string) []Monster {
	return filterSpellsByFilter(spells, SpellTreasureFilter{Level: level})
}

func (ui *UI) renderGeneratedSpellTreasure(filter SpellTreasureFilter, spells []Monster) {
	meta := &strings.Builder{}
	fmt.Fprintf(meta, "[yellow]Spell Treasure[-]\n")
	fmt.Fprintf(meta, "[white]Level:[-] %s\n", blankIfEmpty(filter.Level, "random"))
	fmt.Fprintf(meta, "[white]School:[-] %s\n", blankIfEmpty(filter.School, "random"))
	fmt.Fprintf(meta, "[white]Count:[-] %d\n", len(spells))
	ui.detailMeta.SetText(meta.String())
	ui.detailMeta.ScrollToBeginning()

	lines := make([]string, 0, len(spells))
	for i, sp := range spells {
		lines = append(lines, fmt.Sprintf("%d. %s [%s] (Level %s, %s)", i+1, sp.Name, sp.Source, sp.CR, sp.Type))
	}
	newText := fmt.Sprintf("[yellow]Generated Spells[-]\n[white]Level:[-] %s  [white]School:[-] %s  [white]Qty:[-] %d\n\n%s", blankIfEmpty(filter.Level, "random"), blankIfEmpty(filter.School, "random"), len(spells), strings.Join(lines, "\n"))
	ui.treasureText = appendTreasureText(ui.treasureText, newText)
	ui.detailTreasure.SetText(ui.treasureText)
	ui.detailTreasure.ScrollToEnd()
	ui.activeBottomPanel = "treasure"
	if ui.detailBottom != nil {
		ui.detailBottom.SwitchToPage("treasure")
	}
}

func (ui *UI) generateItemTreasureByKinds(kinds []string, count int) ([]Monster, error) {
	if count < 1 {
		return nil, errors.New("quantity must be >= 1")
	}
	candidates := filterItemsByTreasureKinds(ui.items, kinds)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no item found for type \"%s\"", strings.Join(kinds, ","))
	}
	out := make([]Monster, 0, count)
	for range count {
		idx := rand.Intn(len(candidates))
		out = append(out, candidates[idx])
	}
	return out, nil
}

func filterItemsByTreasureKinds(items []Monster, kinds []string) []Monster {
	if len(kinds) == 0 {
		return append([]Monster(nil), items...)
	}
	set := map[string]struct{}{}
	for _, k := range kinds {
		k = strings.TrimSpace(k)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	if len(set) == 0 {
		return append([]Monster(nil), items...)
	}
	if _, ok := set["random"]; ok {
		return append([]Monster(nil), items...)
	}
	if _, ok := set["any"]; ok {
		return append([]Monster(nil), items...)
	}
	if _, ok := set["*"]; ok {
		return append([]Monster(nil), items...)
	}
	merged := make([]Monster, 0, len(items))
	seen := map[int]struct{}{}
	for k := range set {
		for _, it := range filterItemsByTreasureType(items, k) {
			if _, ok := seen[it.ID]; ok {
				continue
			}
			seen[it.ID] = struct{}{}
			merged = append(merged, it)
		}
	}
	return merged
}

func filterItemsByTreasureType(items []Monster, kind string) []Monster {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" || kind == "random" || kind == "*" || kind == "any" {
		return append([]Monster(nil), items...)
	}
	matches := make([]Monster, 0, len(items))
	for _, it := range items {
		raw := it.Raw
		typ := strings.ToLower(strings.TrimSpace(it.Type))
		name := strings.ToLower(strings.TrimSpace(it.Name))
		hasFlag := func(key string) bool {
			b, ok := raw[key].(bool)
			return ok && b
		}
		ok := false
		switch kind {
		case "potion":
			ok = hasFlag("potion") || strings.Contains(name, "potion")
		case "scroll", "spell":
			ok = hasFlag("scroll") || strings.Contains(name, "scroll")
		case "staff":
			ok = hasFlag("staff") || strings.Contains(typ, "staff")
		case "wand":
			ok = hasFlag("wand") || strings.Contains(typ, "wand")
		case "rod":
			ok = hasFlag("rod") || strings.Contains(typ, "rod")
		case "ring":
			ok = hasFlag("ring") || strings.Contains(typ, "ring")
		case "weapon":
			ok = hasFlag("weapon") || strings.Contains(typ, "weapon")
		case "armor", "armour":
			ok = hasFlag("armor") || strings.Contains(typ, "armor")
		case "wondrous":
			ok = hasFlag("wondrous") || strings.Contains(typ, "wondrous")
		default:
			ok = strings.Contains(typ, kind) || strings.Contains(name, kind)
		}
		if ok {
			matches = append(matches, it)
		}
	}
	return matches
}

func (ui *UI) renderGeneratedItemTreasure(kind string, items []Monster) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "random"
	}
	meta := &strings.Builder{}
	fmt.Fprintf(meta, "[yellow]Item Treasure[-]\n")
	fmt.Fprintf(meta, "[white]Type:[-] %s\n", kind)
	fmt.Fprintf(meta, "[white]Count:[-] %d\n", len(items))
	ui.detailMeta.SetText(meta.String())
	ui.detailMeta.ScrollToBeginning()

	lines := make([]string, 0, len(items))
	for i, it := range items {
		rarity := strings.TrimSpace(it.CR)
		if rarity == "" {
			rarity = "n/a"
		}
		price := formatItemBasePrice(it.Raw)
		if price == "" {
			price = "n/a"
		}
		lines = append(lines, fmt.Sprintf("%d. %s [%s] (%s) - %s", i+1, it.Name, it.Source, rarity, price))
	}

	newText := fmt.Sprintf("[yellow]Generated Items[-]\n[white]Type:[-] %s  [white]Qty:[-] %d\n\n%s", kind, len(items), strings.Join(lines, "\n"))
	ui.treasureText = appendTreasureText(ui.treasureText, newText)
	ui.detailTreasure.SetText(ui.treasureText)
	ui.detailTreasure.ScrollToEnd()
	ui.activeBottomPanel = "treasure"
	if ui.detailBottom != nil {
		ui.detailBottom.SwitchToPage("treasure")
	}
}

func (ui *UI) renderTreasureOutcome(crText string, out treasureOutcome) {
	order := []string{"cp", "sp", "ep", "gp", "pp"}
	coins := make([]string, 0, len(order))
	totalGP := 0.0
	values := map[string]float64{
		"cp": 0.01,
		"sp": 0.1,
		"ep": 0.5,
		"gp": 1.0,
		"pp": 10.0,
	}
	for _, c := range order {
		n := out.Coins[c]
		if n <= 0 {
			continue
		}
		coins = append(coins, fmt.Sprintf("%d %s", n, c))
		totalGP += float64(n) * values[c]
	}
	if len(coins) == 0 {
		coins = append(coins, "0 gp")
	}
	kind := strings.TrimSpace(out.Kind)
	if kind == "" {
		kind = "Individual Treasure"
	}
	meta := &strings.Builder{}
	fmt.Fprintf(meta, "[yellow]Treasure Generator[-]\n")
	fmt.Fprintf(meta, "[white]CR:[-] %s\n", crText)
	fmt.Fprintf(meta, "[white]Table:[-] %s (%s)\n", kind, out.Band)
	fmt.Fprintf(meta, "[white]d100:[-] %d\n", out.D100)
	fmt.Fprintf(meta, "[white]Coins:[-] %s\n", strings.Join(coins, ", "))
	if len(out.Extras) > 0 {
		fmt.Fprintf(meta, "[white]Extras:[-] %s\n", strings.Join(out.Extras, "; "))
	}
	fmt.Fprintf(meta, "[white]GP eq:[-] %.2f", totalGP)
	ui.detailMeta.SetText(meta.String())
	ui.detailMeta.ScrollToBeginning()

	tre := &strings.Builder{}
	fmt.Fprintf(tre, "[yellow]%s[-]\n", kind)
	fmt.Fprintf(tre, "[white]CR:[-] %s   [white]Band:[-] %s   [white]d100:[-] %d\n", crText, out.Band, out.D100)
	fmt.Fprintf(tre, "[white]Coins:[-] %s\n", strings.Join(coins, ", "))
	if len(out.Extras) > 0 {
		fmt.Fprintf(tre, "[white]Extras:[-]\n")
		for _, ex := range out.Extras {
			fmt.Fprintf(tre, "- %s\n", ex)
		}
	}
	fmt.Fprintf(tre, "[white]GP eq:[-] %.2f", totalGP)
	ui.treasureText = appendTreasureText(ui.treasureText, tre.String())
	ui.detailTreasure.SetText(ui.treasureText)
	ui.detailTreasure.ScrollToEnd()
	ui.activeBottomPanel = "treasure"
	if ui.detailBottom != nil {
		ui.detailBottom.SwitchToPage("treasure")
	}

	raw := &strings.Builder{}
	fmt.Fprintf(raw, "Treasure Generation (D&D 5e - %s)\n", kind)
	fmt.Fprintf(raw, "CR input: %s\n", crText)
	fmt.Fprintf(raw, "Band: %s\n", out.Band)
	fmt.Fprintf(raw, "d100 roll: %d\n", out.D100)
	fmt.Fprintf(raw, "\nRoll Breakdown\n")
	for _, line := range out.Breakdown {
		fmt.Fprintf(raw, "- %s\n", line)
	}
	if len(out.Extras) > 0 {
		fmt.Fprintf(raw, "\nExtra Loot\n")
		for _, line := range out.Extras {
			fmt.Fprintf(raw, "- %s\n", line)
		}
	}
	fmt.Fprintf(raw, "\nResult\n%s\n", strings.Join(coins, ", "))
	fmt.Fprintf(raw, "GP equivalent: %.2f\n", totalGP)

	ui.rawText = strings.TrimSpace(raw.String())
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) openDiceRollInput() {
	input := tview.NewInputField().
		SetLabel("Roll ").
		SetFieldWidth(40)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetTitle(" Dice Roll (e.g. 2d6+d20+1 or 1d6-1) ")
	input.SetBorder(true)
	input.SetTitleColor(tcell.ColorGold)
	input.SetBorderColor(tcell.ColorGold)

	closeModal := func() {
		ui.pages.RemovePage("dice-roll")
		ui.app.SetFocus(ui.dice)
	}

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			closeModal()
		case tcell.KeyEnter:
			exprInput := strings.TrimSpace(input.GetText())
			if exprInput == "" {
				closeModal()
				return
			}
			batchExprs, err := expandDiceRollInput(exprInput)
			if err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid dice expression[-:-] %v  %s", err, helpText))
				return
			}
			ui.pushDiceUndo()
			lastTotal := 0
			for _, expr := range batchExprs {
				total, breakdown, rollErr := rollDiceExpression(expr)
				if rollErr != nil {
					ui.status.SetText(fmt.Sprintf(" [white:red] invalid dice expression[-:-] %v  %s", rollErr, helpText))
					return
				}
				lastTotal = total
				ui.appendDiceLog(DiceResult{
					Expression: expr,
					Output:     breakdown,
				})
			}
			if len(batchExprs) > 1 {
				ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] creati %d lanci (ultimo=%d)  %s", len(batchExprs), lastTotal, helpText))
			} else {
				ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] %s = %d  %s", batchExprs[0], lastTotal, helpText))
			}
			closeModal()
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("dice-roll", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openDiceReRollInput() {
	if len(ui.diceLog) == 0 {
		ui.openDiceRollInput()
		return
	}
	index := ui.dice.GetCurrentItem()
	if index < 0 || index >= len(ui.diceLog) {
		index = len(ui.diceLog) - 1
	}

	input := tview.NewInputField().
		SetLabel("Roll ").
		SetFieldWidth(40)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetTitle(" Edit + Re-roll Dice ")
	input.SetBorder(true)
	input.SetTitleColor(tcell.ColorGold)
	input.SetBorderColor(tcell.ColorGold)
	input.SetText(ui.diceLog[index].Expression)

	closeModal := func() {
		ui.pages.RemovePage("dice-reroll")
		ui.app.SetFocus(ui.dice)
	}

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			closeModal()
		case tcell.KeyEnter:
			exprInput := strings.TrimSpace(input.GetText())
			if exprInput == "" {
				closeModal()
				return
			}
			batchExprs, err := expandDiceRollInput(exprInput)
			if err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid dice expression[-:-] %v  %s", err, helpText))
				return
			}
			ui.pushDiceUndo()
			total, breakdown, rollErr := rollDiceExpression(batchExprs[0])
			if rollErr != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid dice expression[-:-] %v  %s", rollErr, helpText))
				return
			}
			ui.diceLog[index] = DiceResult{Expression: batchExprs[0], Output: breakdown}
			insertAt := index + 1
			lastTotal := total
			for i := 1; i < len(batchExprs); i++ {
				t, b, e := rollDiceExpression(batchExprs[i])
				if e != nil {
					ui.status.SetText(fmt.Sprintf(" [white:red] invalid dice expression[-:-] %v  %s", e, helpText))
					return
				}
				lastTotal = t
				entry := DiceResult{Expression: batchExprs[i], Output: b}
				ui.diceLog = append(ui.diceLog, DiceResult{})
				copy(ui.diceLog[insertAt+1:], ui.diceLog[insertAt:])
				ui.diceLog[insertAt] = entry
				insertAt++
			}
			ui.renderDiceList()
			ui.dice.SetCurrentItem(index)
			if len(batchExprs) > 1 {
				ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] aggiornato in %d lanci (ultimo=%d)  %s", len(batchExprs), lastTotal, helpText))
			} else {
				ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] aggiornato %s = %d  %s", batchExprs[0], total, helpText))
			}
			closeModal()
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("dice-reroll", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) rerollSelectedDiceResult() {
	if len(ui.diceLog) == 0 {
		return
	}
	index := ui.dice.GetCurrentItem()
	if index < 0 || index >= len(ui.diceLog) {
		index = len(ui.diceLog) - 1
	}
	expr := strings.TrimSpace(ui.diceLog[index].Expression)
	if expr == "" {
		return
	}
	total, breakdown, err := rollDiceExpression(expr)
	if err != nil {
		ui.status.SetText(fmt.Sprintf(" [white:red] invalid dice expression[-:-] %v  %s", err, helpText))
		return
	}
	ui.pushDiceUndo()
	ui.diceLog[index] = DiceResult{
		Expression: expr,
		Output:     breakdown,
	}
	ui.renderDiceList()
	ui.dice.SetCurrentItem(index)
	ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] rilanciato %s = %d  %s", expr, total, helpText))
}

func (ui *UI) rerollAllDiceResults() {
	if len(ui.diceLog) == 0 {
		return
	}
	ui.pushDiceUndo()
	okCount := 0
	errCount := 0
	for i := range ui.diceLog {
		expr := strings.TrimSpace(ui.diceLog[i].Expression)
		if expr == "" {
			errCount++
			continue
		}
		_, breakdown, err := rollDiceExpression(expr)
		if err != nil {
			errCount++
			continue
		}
		ui.diceLog[i].Output = breakdown
		okCount++
	}
	ui.renderDiceList()
	ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] all rerolled (%d ok, %d errors)  %s", okCount, errCount, helpText))
}

func (ui *UI) gotoDiceRow(row1Based int) {
	if len(ui.diceLog) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no dice row available[-:-]  %s", helpText))
		return
	}
	if row1Based < 1 {
		row1Based = 1
	}
	if row1Based > len(ui.diceLog) {
		row1Based = len(ui.diceLog)
	}
	ui.dice.SetCurrentItem(row1Based - 1)
	ui.status.SetText(fmt.Sprintf(" [black:gold]dice goto[-:-] row %d/%d  %s", row1Based, len(ui.diceLog), helpText))
}

func (ui *UI) gotoLastDiceRow() {
	if len(ui.diceLog) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no dice row available[-:-]  %s", helpText))
		return
	}
	ui.gotoDiceRow(len(ui.diceLog))
}

func (ui *UI) appendDiceLog(entry DiceResult) {
	ui.diceLog = append(ui.diceLog, entry)
	if len(ui.diceLog) > 100 {
		ui.diceLog = ui.diceLog[len(ui.diceLog)-100:]
	}
	ui.renderDiceList()
	if len(ui.diceLog) > 0 {
		ui.dice.SetCurrentItem(len(ui.diceLog) - 1)
	}
}

func (ui *UI) renderDiceList() {
	ui.diceRender = true
	defer func() { ui.diceRender = false }()
	current := max(ui.dice.GetCurrentItem(), 0)
	ui.dice.Clear()
	for i, row := range ui.diceLog {
		expr := row.Expression
		out := row.Output
		if i == current {
			expr = "[black:gold]" + expr + "[-:-]"
			out = highlightDiceFinalResult(out)
		}
		ui.dice.AddItem(fmt.Sprintf("%d %s => %s", i+1, expr, out), "", 0, nil)
	}
	if len(ui.diceLog) == 0 {
		return
	}
	if current >= len(ui.diceLog) {
		current = len(ui.diceLog) - 1
	}
	ui.dice.SetCurrentItem(current)
}

func highlightDiceFinalResult(output string) string {
	locs := finalResultRe.FindAllStringIndex(output, -1)
	if len(locs) == 0 {
		return output
	}
	last := locs[len(locs)-1]
	return output[:last[0]] + "[black:gold]" + output[last[0]:last[1]] + "[-:-]" + output[last[1]:]
}

func (ui *UI) deleteSelectedDiceResult() {
	if len(ui.diceLog) == 0 {
		return
	}
	ui.pushDiceUndo()
	index := ui.dice.GetCurrentItem()
	if index < 0 || index >= len(ui.diceLog) {
		index = len(ui.diceLog) - 1
	}
	ui.diceLog = append(ui.diceLog[:index], ui.diceLog[index+1:]...)
	ui.renderDiceList()
	if len(ui.diceLog) == 0 {
		ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] lista vuota  %s", helpText))
		return
	}
	if index >= len(ui.diceLog) {
		index = len(ui.diceLog) - 1
	}
	ui.dice.SetCurrentItem(index)
	ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] row deleted  %s", helpText))
}

func (ui *UI) clearDiceResults() {
	if len(ui.diceLog) == 0 {
		return
	}
	ui.pushDiceUndo()
	ui.diceLog = nil
	ui.renderDiceList()
	ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] all rows cleared  %s", helpText))
}

func (ui *UI) pushDiceUndo() {
	snap := DiceUndoState{
		Items:    append([]DiceResult(nil), ui.diceLog...),
		Selected: ui.dice.GetCurrentItem(),
	}
	ui.diceUndo = append(ui.diceUndo, snap)
	ui.diceRedo = ui.diceRedo[:0]
}

func (ui *UI) captureDiceState() DiceUndoState {
	return DiceUndoState{
		Items:    append([]DiceResult(nil), ui.diceLog...),
		Selected: ui.dice.GetCurrentItem(),
	}
}

func (ui *UI) restoreDiceState(state DiceUndoState) {
	ui.diceLog = append([]DiceResult(nil), state.Items...)
	ui.renderDiceList()
	if len(ui.diceLog) == 0 {
		return
	}
	idx := max(state.Selected, 0)
	if idx >= len(ui.diceLog) {
		idx = len(ui.diceLog) - 1
	}
	ui.dice.SetCurrentItem(idx)
}

func (ui *UI) undoDiceCommand() {
	if len(ui.diceUndo) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no dice operation to undo[-:-]  %s", helpText))
		return
	}
	current := ui.captureDiceState()
	last := ui.diceUndo[len(ui.diceUndo)-1]
	ui.diceUndo = ui.diceUndo[:len(ui.diceUndo)-1]
	ui.diceRedo = append(ui.diceRedo, current)
	ui.restoreDiceState(last)
	ui.status.SetText(fmt.Sprintf(" [black:gold] undo[-:-] dice operation undone  %s", helpText))
}

func (ui *UI) redoDiceCommand() {
	if len(ui.diceRedo) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no dice operation to redo[-:-]  %s", helpText))
		return
	}
	current := ui.captureDiceState()
	last := ui.diceRedo[len(ui.diceRedo)-1]
	ui.diceRedo = ui.diceRedo[:len(ui.diceRedo)-1]
	ui.diceUndo = append(ui.diceUndo, current)
	ui.restoreDiceState(last)
	ui.status.SetText(fmt.Sprintf(" [black:gold] redo[-:-] dice operation redone  %s", helpText))
}

func (ui *UI) openDiceSaveAsInput() {
	input := tview.NewInputField().
		SetLabel("Dice file ").
		SetFieldWidth(56)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetTitle(" Save Dice Results As ")
	input.SetBorder(true)
	input.SetTitleColor(tcell.ColorGold)
	input.SetBorderColor(tcell.ColorGold)
	input.SetText(ui.dicePath)

	closeModal := func() {
		ui.pages.RemovePage("dice-save")
		ui.app.SetFocus(ui.dice)
	}

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			closeModal()
		case tcell.KeyEnter:
			path := strings.TrimSpace(input.GetText())
			if path == "" {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid file name[-:-]  %s", helpText))
				return
			}
			if err := ui.saveDiceResultsAs(path); err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] save error dice[-:-] %v  %s", err, helpText))
				return
			}
			ui.status.SetText(fmt.Sprintf(" [black:gold] dice saved[-:-] %s  %s", ui.dicePath, helpText))
			closeModal()
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("dice-save", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openDiceLoadInput() {
	input := tview.NewInputField().
		SetLabel("Dice file ").
		SetFieldWidth(56)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetTitle(" Load Dice Results ")
	input.SetBorder(true)
	input.SetTitleColor(tcell.ColorGold)
	input.SetBorderColor(tcell.ColorGold)
	input.SetText(ui.dicePath)

	closeModal := func() {
		ui.pages.RemovePage("dice-load")
		ui.app.SetFocus(ui.dice)
	}

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			closeModal()
		case tcell.KeyEnter:
			path := strings.TrimSpace(input.GetText())
			if path == "" {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid file name[-:-]  %s", helpText))
				return
			}
			prevState := ui.captureDiceState()
			prev := ui.dicePath
			ui.dicePath = path
			if err := ui.loadDiceResults(); err != nil {
				ui.dicePath = prev
				ui.status.SetText(fmt.Sprintf(" [white:red] loading error dice[-:-] %v  %s", err, helpText))
				return
			}
			ui.diceUndo = append(ui.diceUndo, prevState)
			ui.diceRedo = ui.diceRedo[:0]
			ui.status.SetText(fmt.Sprintf(" [black:gold] dice loaded[-:-] %s  %s", ui.dicePath, helpText))
			closeModal()
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("dice-load", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) updateFilterLayout(screenWidth int) {
	if ui.filterHost == nil || ui.leftPanel == nil || ui.monstersPanel == nil {
		return
	}
	wide := screenWidth >= 140
	if ui.wideFilter == wide {
		return
	}
	ui.wideFilter = wide
	if ui.fullscreenActive {
		return
	}
	if wide {
		ui.filterHost.SwitchToPage("single")
		ui.monstersPanel.ResizeItem(ui.filterHost, 1, 0)
	} else {
		ui.filterHost.SwitchToPage("double")
		ui.monstersPanel.ResizeItem(ui.filterHost, 2, 0)
	}
}

func (ui *UI) applyBaseLayout() {
	if ui.mainRow == nil || ui.leftPanel == nil || ui.detailPanel == nil || ui.filterHost == nil || ui.monstersPanel == nil || ui.detailBottom == nil {
		return
	}
	ui.mainRow.ResizeItem(ui.leftPanel, 0, 1)
	ui.mainRow.ResizeItem(ui.detailPanel, 0, 1)
	ui.leftPanel.ResizeItem(ui.dice, 7, 0)
	ui.leftPanel.ResizeItem(ui.encounter, 8, 0)
	ui.leftPanel.ResizeItem(ui.monstersPanel, 0, 1)
	if ui.wideFilter {
		ui.filterHost.SwitchToPage("single")
		ui.monstersPanel.ResizeItem(ui.filterHost, 1, 0)
	} else {
		ui.filterHost.SwitchToPage("double")
		ui.monstersPanel.ResizeItem(ui.filterHost, 2, 0)
	}
	ui.monstersPanel.ResizeItem(ui.list, 0, 1)
	ui.detailPanel.ResizeItem(ui.detailMeta, 8, 0)
	ui.detailPanel.ResizeItem(ui.detailBottom, 0, 1)
}

func (ui *UI) fullscreenTargetForFocus(focus tview.Primitive) string {
	switch focus {
	case ui.dice:
		return "dice"
	case ui.encounter:
		return "encounter"
	case ui.list:
		return "monsters"
	case ui.detailRaw, ui.detailTreasure, ui.detailMeta:
		return "description"
	case ui.nameInput, ui.envDrop, ui.sourceDrop, ui.crDrop, ui.typeDrop:
		return "filters"
	default:
		return ""
	}
}

func (ui *UI) toggleFullscreenForFocus(focus tview.Primitive) {
	if ui.fullscreenActive {
		ui.fullscreenActive = false
		ui.fullscreenTarget = ""
		ui.applyBaseLayout()
		ui.status.SetText(fmt.Sprintf(" [black:gold]fullscreen[-:-] disabled  %s", helpText))
		return
	}
	target := ui.fullscreenTargetForFocus(focus)
	if target == "" || ui.mainRow == nil || ui.leftPanel == nil || ui.detailPanel == nil || ui.filterHost == nil || ui.monstersPanel == nil {
		return
	}
	ui.applyBaseLayout()
	ui.fullscreenActive = true
	ui.fullscreenTarget = target

	switch target {
	case "dice":
		ui.mainRow.ResizeItem(ui.leftPanel, 0, 1)
		ui.mainRow.ResizeItem(ui.detailPanel, 0, 0)
		ui.leftPanel.ResizeItem(ui.dice, 0, 1)
		ui.leftPanel.ResizeItem(ui.encounter, 0, 0)
		ui.leftPanel.ResizeItem(ui.monstersPanel, 0, 0)
	case "encounter":
		ui.mainRow.ResizeItem(ui.leftPanel, 0, 1)
		ui.mainRow.ResizeItem(ui.detailPanel, 0, 0)
		ui.leftPanel.ResizeItem(ui.dice, 0, 0)
		ui.leftPanel.ResizeItem(ui.encounter, 0, 1)
		ui.leftPanel.ResizeItem(ui.monstersPanel, 0, 0)
	case "filters":
		ui.mainRow.ResizeItem(ui.leftPanel, 0, 1)
		ui.mainRow.ResizeItem(ui.detailPanel, 0, 0)
		ui.leftPanel.ResizeItem(ui.dice, 0, 0)
		ui.leftPanel.ResizeItem(ui.encounter, 0, 0)
		ui.leftPanel.ResizeItem(ui.monstersPanel, 0, 1)
		ui.monstersPanel.ResizeItem(ui.filterHost, 0, 1)
		ui.monstersPanel.ResizeItem(ui.list, 0, 0)
	case "monsters":
		ui.mainRow.ResizeItem(ui.leftPanel, 0, 1)
		ui.mainRow.ResizeItem(ui.detailPanel, 0, 0)
		ui.leftPanel.ResizeItem(ui.dice, 0, 0)
		ui.leftPanel.ResizeItem(ui.encounter, 0, 0)
		ui.leftPanel.ResizeItem(ui.monstersPanel, 0, 1)
		ui.monstersPanel.ResizeItem(ui.filterHost, 0, 0)
		ui.monstersPanel.ResizeItem(ui.list, 0, 1)
	case "description":
		ui.mainRow.ResizeItem(ui.leftPanel, 0, 0)
		ui.mainRow.ResizeItem(ui.detailPanel, 0, 1)
		ui.detailPanel.ResizeItem(ui.detailMeta, 0, 0)
		ui.detailPanel.ResizeItem(ui.detailBottom, 0, 1)
	}
	ui.status.SetText(fmt.Sprintf(" [black:gold]fullscreen[-:-] %s  %s", target, helpText))
}

func (ui *UI) run() error {
	return ui.app.Run()
}

func (ui *UI) closeItemTreasureModal() {
	ui.pages.RemovePage("items-treasure-input")
	ui.itemTreasureVisible = false
	ui.app.SetFocus(ui.list)
}

func (ui *UI) closeSpellTreasureModal() {
	ui.pages.RemovePage("spells-treasure-input")
	ui.spellTreasureVisible = false
	ui.app.SetFocus(ui.list)
}

func (ui *UI) browseModeName() string {
	switch ui.browseMode {
	case BrowseItems:
		return "Items"
	case BrowseSpells:
		return "Spells"
	case BrowseCharacters:
		return "Characters"
	case BrowseRaces:
		return "Races"
	case BrowseFeats:
		return "Feats"
	case BrowseBooks:
		return "Manuals"
	case BrowseAdventures:
		return "Adventures"
	case BrowseRandom:
		return "Random"
	case BrowseNotes:
		return "Notes"
	default:
		return "Monsters"
	}
}

func (ui *UI) activeEntries() []Monster {
	switch ui.browseMode {
	case BrowseItems:
		return ui.items
	case BrowseSpells:
		return ui.spells
	case BrowseCharacters:
		return ui.classes
	case BrowseRaces:
		return ui.races
	case BrowseFeats:
		return ui.feats
	case BrowseBooks:
		return ui.books
	case BrowseAdventures:
		return ui.adventures
	case BrowseRandom:
		return ui.randoms
	case BrowseNotes:
		return ui.notesToMonsters()
	default:
		return ui.monsters
	}
}

func (ui *UI) setFilterOptionsForMode() {
	switch ui.browseMode {
	case BrowseItems:
		ui.nameInput.SetLabel(" (n) Name ")
		ui.envDrop.SetLabel(" (e) Env ")
		ui.sourceDrop.SetLabel(" (s) Source ")
		ui.crDrop.SetLabel(" (r) Rarity ")
		ui.typeDrop.SetLabel(" (t) Type ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
		seenSource := map[string]struct{}{}
		seenCR := map[string]struct{}{}
		seenType := map[string]struct{}{}
		for _, it := range ui.items {
			if s := strings.TrimSpace(it.Source); s != "" {
				seenSource[s] = struct{}{}
			}
			if s := strings.TrimSpace(it.CR); s != "" {
				seenCR[s] = struct{}{}
			}
			if s := strings.TrimSpace(it.Type); s != "" {
				seenType[s] = struct{}{}
			}
		}
		ui.envOptions = append(ui.envOptions, keysSorted(seenSource)...)
		ui.sourceOptions = append(ui.sourceOptions, keysSorted(seenSource)...)
		ui.crOptions = append(ui.crOptions, keysSorted(seenCR)...)
		ui.typeOptions = append(ui.typeOptions, keysSorted(seenType)...)
	case BrowseSpells:
		ui.nameInput.SetLabel(" (n) Name ")
		ui.envDrop.SetLabel(" (e) Env ")
		ui.sourceDrop.SetLabel(" (s) Source ")
		ui.crDrop.SetLabel(" Level ")
		ui.typeDrop.SetLabel(" (c) School ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
		seenSource := map[string]struct{}{}
		seenCR := map[string]struct{}{}
		seenType := map[string]struct{}{}
		for _, sp := range ui.spells {
			if s := strings.TrimSpace(sp.Source); s != "" {
				seenSource[s] = struct{}{}
			}
			if s := strings.TrimSpace(sp.CR); s != "" {
				seenCR[s] = struct{}{}
			}
			if s := strings.TrimSpace(sp.Type); s != "" {
				seenType[s] = struct{}{}
			}
		}
		ui.envOptions = append(ui.envOptions, keysSorted(seenSource)...)
		ui.sourceOptions = append(ui.sourceOptions, keysSorted(seenSource)...)
		ui.crOptions = append(ui.crOptions, sortCR(keysSorted(seenCR))...)
		ui.typeOptions = append(ui.typeOptions, keysSorted(seenType)...)
	case BrowseCharacters:
		ui.nameInput.SetLabel(" (n) Name ")
		ui.envDrop.SetLabel(" (e) Primary ")
		ui.sourceDrop.SetLabel(" (s) Source ")
		ui.crDrop.SetLabel(" (c) Hit Die ")
		ui.typeDrop.SetLabel(" (t) Caster ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
		seenPrimary := map[string]struct{}{}
		seenSource := map[string]struct{}{}
		seenHD := map[string]struct{}{}
		seenCaster := map[string]struct{}{}
		for _, cl := range ui.classes {
			for _, p := range cl.Environment {
				if strings.TrimSpace(p) != "" {
					seenPrimary[p] = struct{}{}
				}
			}
			if s := strings.TrimSpace(cl.Source); s != "" {
				seenSource[s] = struct{}{}
			}
			if s := strings.TrimSpace(cl.CR); s != "" {
				seenHD[s] = struct{}{}
			}
			if s := strings.TrimSpace(cl.Type); s != "" {
				seenCaster[s] = struct{}{}
			}
		}
		ui.envOptions = append(ui.envOptions, keysSorted(seenPrimary)...)
		ui.sourceOptions = append(ui.sourceOptions, keysSorted(seenSource)...)
		ui.crOptions = append(ui.crOptions, keysSorted(seenHD)...)
		ui.typeOptions = append(ui.typeOptions, keysSorted(seenCaster)...)
	case BrowseRaces:
		ui.nameInput.SetLabel(" (n) Name ")
		ui.envDrop.SetLabel(" (e) Ability ")
		ui.sourceDrop.SetLabel(" (s) Source ")
		ui.crDrop.SetLabel(" (c) Size ")
		ui.typeDrop.SetLabel(" (t) Lineage ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
		seenAbility := map[string]struct{}{}
		seenSource := map[string]struct{}{}
		seenSize := map[string]struct{}{}
		seenLineage := map[string]struct{}{}
		for _, rc := range ui.races {
			for _, p := range rc.Environment {
				if strings.TrimSpace(p) != "" {
					seenAbility[p] = struct{}{}
				}
			}
			if s := strings.TrimSpace(rc.Source); s != "" {
				seenSource[s] = struct{}{}
			}
			if s := strings.TrimSpace(rc.CR); s != "" {
				seenSize[s] = struct{}{}
			}
			if s := strings.TrimSpace(rc.Type); s != "" {
				seenLineage[s] = struct{}{}
			}
		}
		ui.envOptions = append(ui.envOptions, keysSorted(seenAbility)...)
		ui.sourceOptions = append(ui.sourceOptions, keysSorted(seenSource)...)
		ui.crOptions = append(ui.crOptions, keysSorted(seenSize)...)
		ui.typeOptions = append(ui.typeOptions, keysSorted(seenLineage)...)
	case BrowseFeats:
		ui.nameInput.SetLabel(" (n) Name ")
		ui.envDrop.SetLabel(" (e) Prereq ")
		ui.sourceDrop.SetLabel(" (s) Source ")
		ui.crDrop.SetLabel(" (c) Category ")
		ui.typeDrop.SetLabel(" (t) Ability ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
		seenPrereq := map[string]struct{}{}
		seenSource := map[string]struct{}{}
		seenCategory := map[string]struct{}{}
		seenAbility := map[string]struct{}{}
		for _, ft := range ui.feats {
			for _, p := range ft.Environment {
				if strings.TrimSpace(p) != "" {
					seenPrereq[p] = struct{}{}
				}
			}
			if s := strings.TrimSpace(ft.Source); s != "" {
				seenSource[s] = struct{}{}
			}
			if s := strings.TrimSpace(ft.CR); s != "" {
				seenCategory[s] = struct{}{}
			}
			if s := strings.TrimSpace(ft.Type); s != "" {
				seenAbility[s] = struct{}{}
			}
		}
		ui.envOptions = append(ui.envOptions, keysSorted(seenPrereq)...)
		ui.sourceOptions = append(ui.sourceOptions, keysSorted(seenSource)...)
		ui.crOptions = append(ui.crOptions, keysSorted(seenCategory)...)
		ui.typeOptions = append(ui.typeOptions, keysSorted(seenAbility)...)
	case BrowseBooks:
		ui.nameInput.SetLabel(" (n) Name ")
		ui.envDrop.SetLabel(" (e) Group ")
		ui.sourceDrop.SetLabel(" (s) Source ")
		ui.crDrop.SetLabel(" (c) Year ")
		ui.typeDrop.SetLabel(" (t) Author ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
		seenGroup := map[string]struct{}{}
		seenSource := map[string]struct{}{}
		seenYear := map[string]struct{}{}
		seenAuthor := map[string]struct{}{}
		for _, bk := range ui.books {
			for _, p := range bk.Environment {
				if strings.TrimSpace(p) != "" {
					seenGroup[p] = struct{}{}
				}
			}
			if s := strings.TrimSpace(bk.Source); s != "" {
				seenSource[s] = struct{}{}
			}
			if s := strings.TrimSpace(bk.CR); s != "" {
				seenYear[s] = struct{}{}
			}
			if s := strings.TrimSpace(bk.Type); s != "" {
				seenAuthor[s] = struct{}{}
			}
		}
		ui.envOptions = append(ui.envOptions, keysSorted(seenGroup)...)
		ui.sourceOptions = append(ui.sourceOptions, keysSorted(seenSource)...)
		ui.crOptions = append(ui.crOptions, keysSorted(seenYear)...)
		ui.typeOptions = append(ui.typeOptions, keysSorted(seenAuthor)...)
	case BrowseAdventures:
		ui.nameInput.SetLabel(" (n) Name ")
		ui.envDrop.SetLabel(" (e) Group ")
		ui.sourceDrop.SetLabel(" (s) Source ")
		ui.crDrop.SetLabel(" (c) Year ")
		ui.typeDrop.SetLabel(" (t) Author ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
		seenGroup := map[string]struct{}{}
		seenSource := map[string]struct{}{}
		seenYear := map[string]struct{}{}
		seenAuthor := map[string]struct{}{}
		for _, ad := range ui.adventures {
			for _, p := range ad.Environment {
				if strings.TrimSpace(p) != "" {
					seenGroup[p] = struct{}{}
				}
			}
			if s := strings.TrimSpace(ad.Source); s != "" {
				seenSource[s] = struct{}{}
			}
			if s := strings.TrimSpace(ad.CR); s != "" {
				seenYear[s] = struct{}{}
			}
			if s := strings.TrimSpace(ad.Type); s != "" {
				seenAuthor[s] = struct{}{}
			}
		}
		ui.envOptions = append(ui.envOptions, keysSorted(seenGroup)...)
		ui.sourceOptions = append(ui.sourceOptions, keysSorted(seenSource)...)
		ui.crOptions = append(ui.crOptions, keysSorted(seenYear)...)
		ui.typeOptions = append(ui.typeOptions, keysSorted(seenAuthor)...)
	case BrowseNotes:
		ui.nameInput.SetLabel(" (n) Title ")
		ui.envDrop.SetLabel(" — ")
		ui.sourceDrop.SetLabel(" — ")
		ui.crDrop.SetLabel(" — ")
		ui.typeDrop.SetLabel(" — ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
	case BrowseRandom:
		ui.nameInput.SetLabel(" (n) Name ")
		ui.envDrop.SetLabel(" Category ")
		ui.sourceDrop.SetLabel(" Source ")
		ui.crDrop.SetLabel(" Group ")
		ui.typeDrop.SetLabel(" Type ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
		seenCat := map[string]struct{}{}
		for _, it := range ui.randoms {
			if s := strings.TrimSpace(it.CR); s != "" {
				seenCat[s] = struct{}{}
			}
		}
		ui.envOptions = append(ui.envOptions, keysSorted(seenCat)...)
		ui.sourceOptions = append(ui.sourceOptions, []string{}...)
		ui.crOptions = append(ui.crOptions, []string{}...)
		ui.typeOptions = append(ui.typeOptions, []string{}...)
	default:
		ui.nameInput.SetLabel(" (n) Name ")
		ui.envDrop.SetLabel(" (e) Env ")
		ui.sourceDrop.SetLabel(" (s) Source ")
		ui.crDrop.SetLabel(" (c) CR ")
		ui.typeDrop.SetLabel(" (t) Type ")
		ui.envOptions = append([]string{"All"}, ui.collectMonsterEnvOptions()...)
		ui.sourceOptions = append([]string{"All"}, ui.collectMonsterSourceOptions()...)
		ui.crOptions = append([]string{"All"}, ui.collectMonsterCROptions()...)
		ui.typeOptions = append([]string{"All"}, ui.collectMonsterTypeOptions()...)
	}
	ui.envDrop.SetOptions(ui.envOptions, func(option string, _ int) {
		if option == "All" {
			ui.envFilter = ""
		} else {
			ui.envFilter = option
		}
		ui.applyFilters()
		ui.maybeReturnFocusToListFromFilter()
	})
	ui.setDropDownByValue(ui.envDrop, ui.envOptions, ui.envFilter)
	ui.refreshSourceDropOptions(-1)
	ui.crDrop.SetOptions(ui.crOptions, func(option string, _ int) {
		if option == "All" {
			ui.crFilter = ""
		} else {
			ui.crFilter = option
		}
		ui.applyFilters()
		ui.maybeReturnFocusToListFromFilter()
	})
	ui.typeDrop.SetOptions(ui.typeOptions, func(option string, _ int) {
		if option == "All" {
			ui.typeFilter = ""
		} else {
			ui.typeFilter = option
		}
		ui.applyFilters()
		ui.maybeReturnFocusToListFromFilter()
	})
	if ui.browseMode == BrowseItems || ui.browseMode == BrowseSpells {
		ui.envFilter = ""
		ui.envDrop.SetCurrentOption(0)
	}
	ui.setDropDownByValue(ui.crDrop, ui.crOptions, ui.crFilter)
	ui.setDropDownByValue(ui.typeDrop, ui.typeOptions, ui.typeFilter)
}

func (ui *UI) refreshSourceDropOptions(preferredIdx int) {
	_ = preferredIdx
	label := "All"
	if n := len(ui.sourceFilters); n > 0 {
		label = fmt.Sprintf("%d selected", n)
	}
	ui.updatingSourceDrop = true
	ui.sourceDrop.SetOptions([]string{label}, nil)
	ui.sourceDrop.SetCurrentOption(0)
	ui.updatingSourceDrop = false
}

func (ui *UI) selectedSourcesSorted() []string {
	return keysSorted(ui.sourceFilters)
}

func (ui *UI) openSourceMultiSelectModal() {
	if len(ui.sourceOptions) <= 1 {
		return
	}
	temp := map[string]struct{}{}
	for k := range ui.sourceFilters {
		temp[k] = struct{}{}
	}

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Source Filter (Space=toggle, Enter=apply, Esc=cancel) ")
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(false)

	render := func() {
		cur := list.GetCurrentItem()
		list.Clear()
		if len(temp) == 0 {
			list.AddItem("[x] All", "", 0, nil)
		} else {
			list.AddItem("[ ] All", "", 0, nil)
		}
		for _, opt := range ui.sourceOptions[1:] {
			mark := "[ ]"
			if _, ok := temp[opt]; ok {
				mark = "[x]"
			}
			list.AddItem(fmt.Sprintf("%s %s", mark, opt), "", 0, nil)
		}
		if cur < 0 {
			cur = 0
		}
		if cur >= list.GetItemCount() {
			cur = list.GetItemCount() - 1
		}
		if cur < 0 {
			cur = 0
		}
		list.SetCurrentItem(cur)
	}

	closeModal := func(apply bool) {
		ui.pages.RemovePage("source-multi")
		if apply {
			ui.sourceFilters = temp
			ui.refreshSourceDropOptions(-1)
			ui.applyFilters()
			ui.app.SetFocus(ui.list)
			return
		}
		ui.app.SetFocus(ui.sourceDrop)
	}

	toggle := func() {
		idx := list.GetCurrentItem()
		if idx <= 0 {
			temp = map[string]struct{}{}
			render()
			return
		}
		if idx >= len(ui.sourceOptions) {
			return
		}
		opt := ui.sourceOptions[idx]
		if _, ok := temp[opt]; ok {
			delete(temp, opt)
		} else {
			temp[opt] = struct{}{}
		}
		render()
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == ' ':
			toggle()
			return nil
		case event.Key() == tcell.KeyEnter:
			closeModal(true)
			return nil
		case event.Key() == tcell.KeyEscape:
			closeModal(false)
			return nil
		default:
			return event
		}
	})

	render()
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 16, 0, true).
			AddItem(nil, 0, 1, false), 70, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("source-multi", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *UI) setSelectedSources(values []string) {
	ui.sourceFilters = map[string]struct{}{}
	allowed := map[string]struct{}{}
	for _, v := range ui.sourceOptions[1:] {
		allowed[v] = struct{}{}
	}
	for _, v := range values {
		if _, ok := allowed[v]; ok {
			ui.sourceFilters[v] = struct{}{}
		}
	}
}

func (ui *UI) setDropDownByValue(drop *tview.DropDown, options []string, value string) {
	if strings.TrimSpace(value) == "" {
		drop.SetCurrentOption(0)
		return
	}
	for i, opt := range options {
		if strings.EqualFold(opt, value) {
			drop.SetCurrentOption(i)
			return
		}
	}
	drop.SetCurrentOption(0)
}

func (ui *UI) maybeReturnFocusToListFromFilter() {
	focus := ui.app.GetFocus()
	if focus == ui.envDrop || focus == ui.crDrop || focus == ui.typeDrop {
		ui.app.SetFocus(ui.list)
	}
}

func (ui *UI) focusHasBrowseFilters(focus tview.Primitive) bool {
	switch focus {
	case ui.list, ui.detailMeta, ui.detailRaw, ui.detailTreasure, ui.nameInput, ui.envDrop, ui.sourceDrop, ui.crDrop, ui.typeDrop:
		return true
	default:
		return false
	}
}

func (ui *UI) clearCurrentBrowseFilters() {
	ui.nameFilter = ""
	ui.envFilter = ""
	ui.crFilter = ""
	ui.typeFilter = ""
	ui.sourceFilters = map[string]struct{}{}

	ui.nameInput.SetText("")
	ui.setDropDownByValue(ui.envDrop, ui.envOptions, "")
	ui.setDropDownByValue(ui.crDrop, ui.crOptions, "")
	ui.setDropDownByValue(ui.typeDrop, ui.typeOptions, "")
	ui.refreshSourceDropOptions(-1)
	ui.applyFilters()
	ui.saveCurrentModeFilters()
	ui.app.SetFocus(ui.list)
	ui.status.SetText(fmt.Sprintf(" [black:gold]filters[-:-] reset (%s)  %s", ui.browseModeName(), helpText))
}

func (ui *UI) descriptionKeyForMode(mode BrowseMode, idx int) string {
	switch mode {
	case BrowseItems:
		return fmt.Sprintf("items:%d", idx)
	case BrowseSpells:
		return fmt.Sprintf("spells:%d", idx)
	case BrowseCharacters:
		return fmt.Sprintf("classes:%d", idx)
	case BrowseRaces:
		return fmt.Sprintf("races:%d", idx)
	case BrowseFeats:
		return fmt.Sprintf("feats:%d", idx)
	case BrowseBooks:
		return fmt.Sprintf("books:%d", idx)
	case BrowseAdventures:
		return fmt.Sprintf("adventures:%d", idx)
	default:
		return fmt.Sprintf("monsters:%d", idx)
	}
}

func (ui *UI) descriptionKeyForEncounterEntry(entry EncounterEntry) string {
	if entry.Custom {
		return fmt.Sprintf("encounter:custom:%s:%d", strings.ToLower(strings.TrimSpace(entry.CustomName)), entry.Ordinal)
	}
	return fmt.Sprintf("monsters:%d", entry.MonsterIndex)
}

func (ui *UI) saveCurrentDescriptionScroll() {
	if ui.detailRaw == nil || ui.currentDescKey == "" {
		return
	}
	row, _ := ui.detailRaw.GetScrollOffset()
	if row < 0 {
		row = 0
	}
	ui.descScroll[ui.currentDescKey] = row
}

func (ui *UI) restoreDescriptionScrollForKey(key string) {
	if ui.detailRaw == nil {
		return
	}
	ui.currentDescKey = key
	row, ok := ui.descScroll[key]
	if !ok || row <= 0 {
		ui.detailRaw.ScrollToBeginning()
		return
	}
	ui.detailRaw.ScrollTo(row, 0)
}

func (ui *UI) loadDescriptionScrollStates() error {
	b, err := os.ReadFile(descScrollStatePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var data PersistedDescriptionScroll
	if err := yaml.Unmarshal(b, &data); err != nil {
		return err
	}
	ui.descScroll = map[string]int{}
	for k, v := range data.Offsets {
		if strings.TrimSpace(k) == "" || v < 0 {
			continue
		}
		ui.descScroll[k] = v
	}
	return nil
}

func (ui *UI) saveDescriptionScrollStates() error {
	ui.saveCurrentDescriptionScroll()
	data := PersistedDescriptionScroll{
		Version: 1,
		Offsets: ui.descScroll,
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	path := descScrollStatePath()
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, out, 0o644)
}

func (ui *UI) saveCurrentModeFilters() {
	ui.modeFilters[ui.browseMode] = PersistedFilterMode{
		Name:    strings.TrimSpace(ui.nameFilter),
		Env:     strings.TrimSpace(ui.envFilter),
		Sources: ui.selectedSourcesSorted(),
		CR:      strings.TrimSpace(ui.crFilter),
		Type:    strings.TrimSpace(ui.typeFilter),
	}
}

func (ui *UI) applyModeFilters(mode BrowseMode) {
	state, ok := ui.modeFilters[mode]
	if !ok {
		state = PersistedFilterMode{}
	}
	ui.nameFilter = strings.TrimSpace(state.Name)
	ui.envFilter = strings.TrimSpace(state.Env)
	ui.crFilter = strings.TrimSpace(state.CR)
	ui.typeFilter = strings.TrimSpace(state.Type)
	ui.setFilterOptionsForMode()
	ui.setSelectedSources(state.Sources)
	ui.refreshSourceDropOptions(-1)
	ui.nameInput.SetText(ui.nameFilter)
	ui.setDropDownByValue(ui.crDrop, ui.crOptions, ui.crFilter)
	ui.setDropDownByValue(ui.typeDrop, ui.typeOptions, ui.typeFilter)
}

func (ui *UI) collectMonsterEnvOptions() []string {
	set := map[string]struct{}{}
	for _, m := range ui.monsters {
		for _, env := range m.Environment {
			if strings.TrimSpace(env) != "" {
				set[env] = struct{}{}
			}
		}
	}
	return keysSorted(set)
}

func (ui *UI) collectMonsterSourceOptions() []string {
	set := map[string]struct{}{}
	for _, m := range ui.monsters {
		if s := strings.TrimSpace(m.Source); s != "" {
			set[s] = struct{}{}
		}
	}
	return keysSorted(set)
}

func (ui *UI) collectMonsterCROptions() []string {
	set := map[string]struct{}{}
	for _, m := range ui.monsters {
		if s := strings.TrimSpace(m.CR); s != "" {
			set[s] = struct{}{}
		}
	}
	return sortCR(keysSorted(set))
}

func (ui *UI) collectMonsterTypeOptions() []string {
	set := map[string]struct{}{}
	for _, m := range ui.monsters {
		if s := strings.TrimSpace(m.Type); s != "" {
			set[s] = struct{}{}
		}
	}
	return keysSorted(set)
}

func (ui *UI) updateBrowsePanelTitle() {
	count := 10
	prev := BrowseMode((int(ui.browseMode) - 1 + count) % count)
	next := BrowseMode((int(ui.browseMode) + 1) % count)
	ui.monstersPanel.SetTitle(fmt.Sprintf(" [2]-%s  [:%s  ]:%s ", ui.browseModeName(), browseModeLabel(prev), browseModeLabel(next)))
}

func (ui *UI) cycleBrowseMode(delta int) {
	if delta == 0 {
		return
	}
	count := 10
	next := (int(ui.browseMode) + delta) % count
	if next < 0 {
		next += count
	}
	ui.setBrowseMode(BrowseMode(next))
}

func browseModeLabel(mode BrowseMode) string {
	switch mode {
	case BrowseItems:
		return "Items"
	case BrowseSpells:
		return "Spells"
	case BrowseCharacters:
		return "Characters"
	case BrowseRaces:
		return "Races"
	case BrowseFeats:
		return "Feats"
	case BrowseBooks:
		return "Manuals"
	case BrowseAdventures:
		return "Adventures"
	case BrowseRandom:
		return "Random"
	case BrowseNotes:
		return "Notes"
	default:
		return "Monsters"
	}
}

func (ui *UI) setBrowseMode(mode BrowseMode) {
	if ui.browseMode == mode {
		return
	}
	ui.saveCurrentModeFilters()
	ui.browseMode = mode
	ui.spellShortcutAlt = false
	ui.applyModeFilters(ui.browseMode)
	ui.updateBrowsePanelTitle()
	ui.applyFilters()
	ui.status.SetText(fmt.Sprintf(" [black:gold]browse[-:-] %s  %s", ui.browseModeName(), helpText))
}

func browseModeToString(m BrowseMode) string {
	switch m {
	case BrowseMonsters:
		return "monsters"
	case BrowseItems:
		return "items"
	case BrowseSpells:
		return "spells"
	case BrowseCharacters:
		return "characters"
	case BrowseRaces:
		return "races"
	case BrowseFeats:
		return "feats"
	case BrowseBooks:
		return "books"
	case BrowseAdventures:
		return "adventures"
	case BrowseRandom:
		return "random"
	case BrowseNotes:
		return "notes"
	default:
		return ""
	}
}

func browseModeFromString(s string) (BrowseMode, bool) {
	switch s {
	case "monsters":
		return BrowseMonsters, true
	case "items":
		return BrowseItems, true
	case "spells":
		return BrowseSpells, true
	case "characters":
		return BrowseCharacters, true
	case "races":
		return BrowseRaces, true
	case "feats":
		return BrowseFeats, true
	case "books":
		return BrowseBooks, true
	case "adventures":
		return BrowseAdventures, true
	case "random":
		return BrowseRandom, true
	case "notes":
		return BrowseNotes, true
	}
	return BrowseMonsters, false
}

func (ui *UI) updateHelpText() {
	if ui.currentCampaign != "" {
		helpText = fmt.Sprintf("%s[gray]│ %s[-] ", helpTextBase, ui.currentCampaign)
	} else {
		helpText = helpTextBase
	}
	ui.status.SetText(fmt.Sprintf(" [black:gold]browse[-:-] %s  %s", ui.browseModeName(), helpText))
}

func (ui *UI) saveNotes() {
	path := ui.notesPath
	if path == "" {
		path = defaultNotesPath()
		ui.notesPath = path
	}
	type notesFile struct {
		Notes []Note `yaml:"notes"`
	}
	b, err := yaml.Marshal(notesFile{Notes: ui.notes})
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o644)
}

func (ui *UI) notesToMonsters() []Monster {
	out := make([]Monster, len(ui.notes))
	for i, n := range ui.notes {
		out[i] = Monster{ID: i, Name: n.Title}
	}
	return out
}

func (ui *UI) loadNotes() {
	path := ui.notesPath
	if path == "" {
		path = defaultNotesPath()
		ui.notesPath = path
	}
	if !fileExists(path) {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	type notesFile struct {
		Notes []Note `yaml:"notes"`
	}
	var nf notesFile
	if err := yaml.Unmarshal(b, &nf); err != nil {
		return
	}
	ui.notes = nf.Notes
}

func (ui *UI) rebuildNotesList() {
	ui.applyFilters()
	idx := ui.list.GetCurrentItem()
	if idx < 0 {
		idx = 0
	}
	if idx >= ui.list.GetItemCount() {
		idx = ui.list.GetItemCount() - 1
	}
	ui.list.SetCurrentItem(idx)
	ui.renderDetailByListIndex(idx)
}

func (ui *UI) openAddNoteModal() {
	titleInput := tview.NewInputField().
		SetLabel(" Title: ").
		SetFieldWidth(40)
	titleInput.SetBorder(true).
		SetTitle(" New Note ").
		SetTitleColor(tcell.ColorGold).
		SetBorderColor(tcell.ColorGold)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(titleInput, 3, 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)

	titleInput.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("note-add")
		ui.app.SetFocus(ui.list)
		if key != tcell.KeyEnter {
			return
		}
		title := strings.TrimSpace(titleInput.GetText())
		if title == "" {
			return
		}
		ui.notes = append(ui.notes, Note{Title: title})
		ui.saveNotes()
		ui.rebuildNotesList()
		ui.list.SetCurrentItem(ui.list.GetItemCount() - 1)
		ui.renderDetailByListIndex(ui.list.GetCurrentItem())
		// open editor immediately for the new note
		ui.openEditNoteModal(ui.list.GetCurrentItem())
	})

	ui.pages.AddPage("note-add", modal, true, true)
	ui.app.SetFocus(titleInput)
}

func (ui *UI) openEditNoteModal(listIdx int) {
	if listIdx < 0 || listIdx >= len(ui.filtered) {
		return
	}
	noteIdx := ui.filtered[listIdx]
	if noteIdx < 0 || noteIdx >= len(ui.notes) {
		return
	}

	area := tview.NewTextArea().
		SetWrap(true).
		SetWordWrap(true).
		SetText(ui.notes[noteIdx].Content, false)
	area.SetBorder(true).
		SetTitle(fmt.Sprintf(" Edit: %s  (Ctrl+S save · Esc cancel) ", tview.Escape(ui.notes[noteIdx].Title))).
		SetTitleColor(tcell.ColorGold).
		SetBorderColor(tcell.ColorGold)

	modal := tview.NewFlex().
		AddItem(nil, 2, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 1, 0, false).
			AddItem(area, 0, 1, true).
			AddItem(nil, 1, 0, false), 0, 1, true).
		AddItem(nil, 2, 0, false)

	ui.noteEditArea = area

	save := func() {
		ui.notes[noteIdx].Content = area.GetText()
		ui.saveNotes()
		ui.noteEditArea = nil
		ui.pages.RemovePage("note-edit")
		ui.app.SetFocus(ui.list)
		ui.renderDetailByListIndex(listIdx)
	}
	cancel := func() {
		ui.noteEditArea = nil
		ui.pages.RemovePage("note-edit")
		ui.app.SetFocus(ui.list)
	}

	area.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlS {
			save()
			return nil
		}
		if event.Key() == tcell.KeyEscape {
			cancel()
			return nil
		}
		return event
	})

	ui.pages.AddPage("note-edit", modal, true, true)
	ui.app.SetFocus(area)
}

func (ui *UI) deleteNote(listIdx int) {
	if listIdx < 0 || listIdx >= len(ui.filtered) {
		return
	}
	noteIdx := ui.filtered[listIdx]
	if noteIdx < 0 || noteIdx >= len(ui.notes) {
		return
	}
	ui.notes = append(ui.notes[:noteIdx], ui.notes[noteIdx+1:]...)
	ui.saveNotes()
	ui.rebuildNotesList()
}

func (ui *UI) applyFilters() {
	ui.filtered = ui.filtered[:0]

	for i, m := range ui.activeEntries() {
		if !ui.matchesNameFilterByMode(m) {
			continue
		}
		if !matchCR(m.CR, ui.crFilter) {
			continue
		}
		if !matchEnv(m.Environment, ui.envFilter) {
			continue
		}
		if !matchEnvMulti([]string{m.Source}, ui.sourceFilters) {
			continue
		}
		if !matchType(m.Type, ui.typeFilter) {
			continue
		}
		ui.filtered = append(ui.filtered, i)
	}

	ui.renderList()
}

func (ui *UI) matchesNameFilterByMode(entry Monster) bool {
	query := strings.TrimSpace(ui.nameFilter)
	if query == "" {
		return true
	}
	if matchName(entry.Name, query) {
		return true
	}
	if ui.browseMode != BrowseCharacters {
		return false
	}
	for _, feature := range extractClassFeatureNames(entry.Raw) {
		if matchName(feature, query) {
			return true
		}
	}
	for _, feature := range extractSubclassFeatureNames(entry.Raw) {
		if matchName(feature, query) {
			return true
		}
	}
	return false
}

func (ui *UI) renderList() {
	ui.list.Clear()

	entries := ui.activeEntries()
	for _, idx := range ui.filtered {
		if idx < 0 || idx >= len(entries) {
			continue
		}
		m := entries[idx]
		ui.list.AddItem(m.Name, "", 0, nil)
	}

	ui.status.SetText(fmt.Sprintf(" [black:gold] %d results [-:-] %s", len(ui.filtered), helpText))

	if len(ui.filtered) == 0 {
		ui.detailMeta.SetText(fmt.Sprintf("No results in %s with current filters.", ui.browseModeName()))
		ui.detailRaw.SetText("")
		ui.rawText = ""
		return
	}

	current := ui.list.GetCurrentItem()
	if current < 0 || current >= len(ui.filtered) {
		current = 0
		ui.list.SetCurrentItem(0)
	}
	ui.renderDetailByListIndex(current)
}

func (ui *UI) renderDetailByListIndex(listIndex int) {
	ui.saveCurrentDescriptionScroll()
	if listIndex < 0 || listIndex >= len(ui.filtered) {
		ui.detailMeta.SetText(fmt.Sprintf("Seleziona un elemento da %s.", ui.browseModeName()))
		ui.detailRaw.SetText("")
		ui.rawText = ""
		ui.currentDescKey = ""
		return
	}
	activeIndex := ui.filtered[listIndex]
	descKey := ui.descriptionKeyForMode(ui.browseMode, activeIndex)
	switch ui.browseMode {
	case BrowseItems:
		ui.renderDetailByItemIndex(activeIndex)
	case BrowseSpells:
		ui.renderDetailBySpellIndex(activeIndex)
	case BrowseCharacters:
		ui.renderDetailByClassIndex(activeIndex)
	case BrowseRaces:
		ui.renderDetailByRaceIndex(activeIndex)
	case BrowseFeats:
		ui.renderDetailByFeatIndex(activeIndex)
	case BrowseBooks:
		ui.renderDetailByBookIndex(activeIndex)
	case BrowseAdventures:
		ui.renderDetailByAdventureIndex(activeIndex)
	case BrowseRandom:
		ui.renderDetailByRandomIndex(activeIndex)
	case BrowseNotes:
		ui.renderDetailByNoteIndex(activeIndex)
	default:
		ui.renderDetailByMonsterIndex(activeIndex)
	}
	ui.restoreDescriptionScrollForKey(descKey)
}

func (ui *UI) renderDetailByNoteIndex(idx int) {
	if idx < 0 || idx >= len(ui.notes) {
		ui.detailMeta.SetText("No note selected.")
		ui.detailRaw.SetText("")
		ui.rawText = ""
		return
	}
	n := ui.notes[idx]
	ui.detailMeta.SetText(fmt.Sprintf("[gold]%s[-]", tview.Escape(n.Title)))
	ui.detailRaw.SetText(tview.Escape(n.Content))
	ui.rawText = n.Content
	ui.currentDescKey = fmt.Sprintf("note-%d", idx)
	ui.detailBottom.SwitchToPage("description")
}

func (ui *UI) renderDetailByEncounterIndex(encounterIndex int) {
	ui.saveCurrentDescriptionScroll()
	if encounterIndex < 0 || encounterIndex >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[encounterIndex]
	descKey := ui.descriptionKeyForEncounterEntry(entry)
	if entry.Custom {
		ui.renderDetailByCustomEntry(entry)
	} else {
		ui.renderDetailByMonsterIndex(entry.MonsterIndex)
	}
	ui.applyEncounterConditionsOverlay(entry)
	meta := ui.ensureEncounterPassivePerceptionLine(entry, ui.detailMeta.GetText(false))
	meta = ui.ensureEncounterTempHPLine(entry, meta)
	ui.detailMeta.SetText(meta)
	ui.restoreDescriptionScrollForKey(descKey)
}

func (ui *UI) applyEncounterConditionsOverlay(entry EncounterEntry) {
	title := " Details "
	cond := strings.TrimSpace(ui.encounterConditionsLong(entry))
	if cond == "" {
		ui.detailMeta.SetTitle(title)
		return
	}
	text := ui.detailMeta.GetText(false)
	ui.detailMeta.SetText(insertConditionsLine(text, cond))
	if badge := ui.encounterConditionsBadge(entry); badge != "" {
		title = fmt.Sprintf(" Details [%s] ", badge)
	}
	ui.detailMeta.SetTitle(title)
}

func (ui *UI) renderDetailByMonsterIndex(monsterIndex int) {
	if monsterIndex < 0 || monsterIndex >= len(ui.monsters) {
		return
	}

	m := ui.monsters[monsterIndex]
	scaleStep := ui.monsterScale[monsterIndex]
	scalePreview, hasScale := scaleMonsterByCR(m, scaleStep)

	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", m.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(m.Source, "n/a"))
	fmt.Fprintf(builder, "[white]Type:[-] %s\n", blankIfEmpty(m.Type, "n/a"))
	fmt.Fprintf(builder, "[white]Size:[-] %s\n", blankIfEmpty(extractMonsterSize(m.Raw["size"]), "n/a"))
	if hasScale {
		fmt.Fprintf(builder, "[white]CR:[-] %s -> %s (%+d)\n", blankIfEmpty(scalePreview.BaseCR, "n/a"), scalePreview.TargetCR, scalePreview.Step)
	} else {
		fmt.Fprintf(builder, "[white]CR:[-] %s\n", blankIfEmpty(m.CR, "n/a"))
	}
	if xp, ok := extractMonsterXP(m.Raw, m.CR); ok {
		fmt.Fprintf(builder, "[white]XP:[-] %d\n", xp)
	}
	if hasScale {
		fmt.Fprintf(builder, "[white]AC:[-] %d -> %d\n", scalePreview.BaseAC, scalePreview.TargetAC)
	} else if ac := extractAC(m.Raw); ac != "" {
		fmt.Fprintf(builder, "[white]AC:[-] %s\n", ac)
	}
	if speed := extractSpeed(m.Raw); speed != "" {
		fmt.Fprintf(builder, "[white]Speed:[-] %s\n", speed)
	}
	if ab := abilityInline(m.Raw); ab != "" {
		fmt.Fprintf(builder, "[white]Abilities:[-] %s\n", ab)
	}
	hpAverage, hpFormula := extractHP(m.Raw)
	if hasScale {
		if hpFormula != "" {
			fmt.Fprintf(builder, "[white]HP:[-] %d -> %d (%s)\n", scalePreview.BaseHP, scalePreview.TargetHP, hpFormula)
		} else {
			fmt.Fprintf(builder, "[white]HP:[-] %d -> %d\n", scalePreview.BaseHP, scalePreview.TargetHP)
		}
		fmt.Fprintf(builder, "[white]Offense Target:[-] hit %+d / DC %d / DPR %d-%d\n", scalePreview.TargetAtk, scalePreview.TargetDC, scalePreview.DPRMin, scalePreview.DPRMax)
	} else if hpAverage != "" || hpFormula != "" {
		switch {
		case hpAverage != "" && hpFormula != "":
			fmt.Fprintf(builder, "[white]HP:[-] %s (%s)\n", hpAverage, hpFormula)
		case hpAverage != "":
			fmt.Fprintf(builder, "[white]HP:[-] %s\n", hpAverage)
		default:
			fmt.Fprintf(builder, "[white]HP:[-] %s\n", hpFormula)
		}
	}
	if len(m.Environment) > 0 {
		fmt.Fprintf(builder, "[white]Environment:[-] %s\n", strings.Join(m.Environment, ", "))
	} else {
		fmt.Fprintf(builder, "[white]Environment:[-] n/a\n")
	}
	if passive, ok := extractPassivePerceptionFromMonster(m.Raw); ok {
		fmt.Fprintf(builder, "[white]Passive Perception:[-] %d\n", passive)
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildMonsterDescriptionTextScaled(m, scalePreview, hasScale)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailByItemIndex(itemIndex int) {
	if itemIndex < 0 || itemIndex >= len(ui.items) {
		return
	}
	it := ui.items[itemIndex]
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", it.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(it.Source, "n/a"))
	fmt.Fprintf(builder, "[white]Type:[-] %s\n", blankIfEmpty(it.Type, "n/a"))
	fmt.Fprintf(builder, "[white]Rarity:[-] %s\n", blankIfEmpty(it.CR, "n/a"))
	if price := formatItemBasePrice(it.Raw); price != "" {
		fmt.Fprintf(builder, "[white]Price:[-] %s\n", price)
	}
	if attune := strings.TrimSpace(asString(it.Raw["reqAttune"])); attune != "" {
		fmt.Fprintf(builder, "[white]Attunement:[-] %s\n", attune)
	}
	if ac := strings.TrimSpace(asString(it.Raw["ac"])); ac != "" {
		fmt.Fprintf(builder, "[white]AC:[-] %s\n", ac)
	}
	if econ, ok := magicItemEconomy(it.Raw, it.CR); ok {
		fmt.Fprintf(builder, "[white]Buy Cost:[-] %s\n", econ.BuyCost)
		fmt.Fprintf(builder, "[white]Find Time:[-] %s\n", econ.FindTime)
		fmt.Fprintf(builder, "[white]Craft Cost:[-] %s\n", econ.CraftCost)
		fmt.Fprintf(builder, "[white]Craft Time:[-] %s\n", econ.CraftTime)
		fmt.Fprintf(builder, "[white]Craft Procedure:[-] %s\n", strings.Join(econ.Procedure, " -> "))
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildItemDescriptionText(it)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailBySpellIndex(spellIndex int) {
	if spellIndex < 0 || spellIndex >= len(ui.spells) {
		return
	}
	sp := ui.spells[spellIndex]
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", sp.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(sp.Source, "n/a"))
	fmt.Fprintf(builder, "[white]School:[-] %s\n", blankIfEmpty(sp.Type, "n/a"))
	fmt.Fprintf(builder, "[white]Level:[-] %s\n", blankIfEmpty(sp.CR, "n/a"))
	if cast := extractSpellTime(sp.Raw); cast != "" {
		fmt.Fprintf(builder, "[white]Casting Time:[-] %s\n", cast)
	}
	if rng := extractSpellRange(sp.Raw); rng != "" {
		fmt.Fprintf(builder, "[white]Range:[-] %s\n", rng)
	}
	if dur := extractSpellDuration(sp.Raw); dur != "" {
		fmt.Fprintf(builder, "[white]Duration:[-] %s\n", dur)
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildSpellDescriptionText(sp)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailByClassIndex(classIndex int) {
	if classIndex < 0 || classIndex >= len(ui.classes) {
		return
	}
	cl := ui.classes[classIndex]
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", cl.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(cl.Source, "n/a"))
	fmt.Fprintf(builder, "[white]Hit Die:[-] %s\n", blankIfEmpty(cl.CR, "n/a"))
	fmt.Fprintf(builder, "[white]Caster:[-] %s\n", blankIfEmpty(cl.Type, "n/a"))
	if len(cl.Environment) > 0 {
		fmt.Fprintf(builder, "[white]Primary:[-] %s\n", strings.Join(cl.Environment, ", "))
	}
	if spell := strings.TrimSpace(asString(cl.Raw["spellcastingAbility"])); spell != "" {
		fmt.Fprintf(builder, "[white]Spellcasting Ability:[-] %s\n", strings.ToUpper(spell))
	}
	if saves := strings.TrimSpace(plainAny(cl.Raw["proficiency"])); saves != "" {
		fmt.Fprintf(builder, "[white]Save Proficiencies:[-] %s\n", strings.ToUpper(saves))
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildClassDescriptionText(cl)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailByRaceIndex(raceIndex int) {
	if raceIndex < 0 || raceIndex >= len(ui.races) {
		return
	}
	rc := ui.races[raceIndex]
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", rc.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(rc.Source, "n/a"))
	fmt.Fprintf(builder, "[white]Size:[-] %s\n", blankIfEmpty(rc.CR, "n/a"))
	fmt.Fprintf(builder, "[white]Lineage:[-] %s\n", blankIfEmpty(rc.Type, "n/a"))
	if len(rc.Environment) > 0 {
		fmt.Fprintf(builder, "[white]Ability:[-] %s\n", strings.Join(rc.Environment, ", "))
	}
	if speed := extractSpeed(rc.Raw); speed != "" {
		fmt.Fprintf(builder, "[white]Speed:[-] %s\n", speed)
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildRaceDescriptionText(rc)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailByFeatIndex(featIndex int) {
	if featIndex < 0 || featIndex >= len(ui.feats) {
		return
	}
	ft := ui.feats[featIndex]
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", ft.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(ft.Source, "n/a"))
	fmt.Fprintf(builder, "[white]Category:[-] %s\n", blankIfEmpty(ft.CR, "n/a"))
	fmt.Fprintf(builder, "[white]Ability:[-] %s\n", blankIfEmpty(ft.Type, "n/a"))
	if len(ft.Environment) > 0 {
		fmt.Fprintf(builder, "[white]Prereq:[-] %s\n", strings.Join(ft.Environment, ", "))
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildFeatDescriptionText(ft)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailByBookIndex(bookIndex int) {
	if bookIndex < 0 || bookIndex >= len(ui.books) {
		return
	}
	bk := ui.books[bookIndex]
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", bk.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(bk.Source, "n/a"))
	fmt.Fprintf(builder, "[white]Group:[-] %s\n", blankIfEmpty(strings.Join(bk.Environment, ", "), "n/a"))
	fmt.Fprintf(builder, "[white]Published:[-] %s\n", blankIfEmpty(bk.CR, "n/a"))
	fmt.Fprintf(builder, "[white]Author:[-] %s\n", blankIfEmpty(bk.Type, "n/a"))
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = ui.fullBookDescriptionText(bk)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailByAdventureIndex(adventureIndex int) {
	if adventureIndex < 0 || adventureIndex >= len(ui.adventures) {
		return
	}
	ad := ui.adventures[adventureIndex]
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", ad.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(ad.Source, "n/a"))
	fmt.Fprintf(builder, "[white]Group:[-] %s\n", blankIfEmpty(strings.Join(ad.Environment, ", "), "n/a"))
	fmt.Fprintf(builder, "[white]Published:[-] %s\n", blankIfEmpty(ad.CR, "n/a"))
	fmt.Fprintf(builder, "[white]Author:[-] %s\n", blankIfEmpty(ad.Type, "n/a"))
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = ui.fullAdventureDescriptionText(ad)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailByRandomIndex(randomIndex int) {
	if randomIndex < 0 || randomIndex >= len(ui.randoms) {
		return
	}
	it := ui.randoms[randomIndex]
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", it.Name)
	fmt.Fprintf(builder, "[white]Category:[-] %s\n", blankIfEmpty(it.CR, "n/a"))
	if gen := strings.TrimSpace(asString(it.Raw["generated"])); gen != "" {
		fmt.Fprintf(builder, "[white]Generated:[-] %s\n", gen)
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = strings.TrimSpace(asString(it.Raw["content"]))
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func chooseOne(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[rand.Intn(len(values))]
}

func randomEncounterCustomName() string {
	name := strings.TrimSpace(chooseOne(randomNPCNames))
	if name == "" {
		return "Custom"
	}
	return name
}

func baseRandomTitle(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	return strings.TrimSpace(numSuffixRe.ReplaceAllString(trimmed, ""))
}

func (ui *UI) nextRandomTitleOrdinal(title string) int {
	base := baseRandomTitle(title)
	if base == "" {
		return 1
	}
	count := 0
	for _, it := range ui.randoms {
		if strings.EqualFold(baseRandomTitle(it.Name), base) {
			count++
		}
	}
	return count + 1
}

func (ui *UI) addRandomEntry(category string, title string, body string) {
	category = strings.TrimSpace(category)
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" || body == "" {
		return
	}
	title = fmt.Sprintf("%s #%d", baseRandomTitle(title), ui.nextRandomTitleOrdinal(title))
	it := Monster{
		ID:          len(ui.randoms) + 1,
		Name:        title,
		CR:          category,
		Environment: []string{category},
		Source:      "random",
		Type:        "generated",
		Raw: map[string]any{
			"name":      title,
			"category":  category,
			"generated": time.Now().Format("2006-01-02 15:04:05"),
			"content":   body,
		},
	}
	ui.randoms = append(ui.randoms, it)
	ui.applyFilters()
	ui.status.SetText(fmt.Sprintf(" [black:gold]random[-:-] generated %s  %s", title, helpText))
	if ui.browseMode != BrowseRandom {
		return
	}
	target := len(ui.randoms) - 1
	for i, idx := range ui.filtered {
		if idx == target {
			ui.list.SetCurrentItem(i)
			ui.renderDetailByListIndex(i)
			break
		}
	}
}

func (ui *UI) generateRandomDungeonRoom() {
	roomKinds := []string{"Collapsed armory", "Flooded shrine", "Arcane vault", "Desecrated chapel", "Forgotten barracks", "Bone archive"}
	traps := []string{"pressure plate dart launcher", "swinging blade pendulum", "glyph of frost", "poison needle lock", "falling block trap"}
	treasures := []string{"sealed coffer of silver trade bars", "dusty chest with mixed coinage", "hidden compartment with gemstones", "ancient art object wrapped in oilcloth"}
	monsters := []string{"restless undead sentry", "ooze feeding on refuse", "ambush drakes", "cultist remnant patrol", "territorial giant spiders"}
	puzzles := []string{"rotating runes requiring elemental sequence", "weight-balance altar puzzle", "mirrored light-beam lock", "musical chime door mechanism"}
	title := "Dungeon Room Content"
	body := fmt.Sprintf("Room: %s\nTrap: %s\nTreasure: %s\nMonster: %s\nPuzzle: %s",
		chooseOne(roomKinds), chooseOne(traps), chooseOne(treasures), chooseOne(monsters), chooseOne(puzzles))
	ui.addRandomEntry("Dungeon", title, body)
}

func (ui *UI) generateRandomDungeonLayout() {
	entries := []string{
		"Entry hall with two locked iron doors; narrow corridor forks into three trapped branches.",
		"Spiral corridor around a central chasm, with collapsing bridges and murder holes.",
		"Grid of stone rooms linked by secret doors; one false corridor loops to a hazard chamber.",
		"Long gallery with portcullis checkpoints, side crypts, and a flooded lower passage.",
		"Broken fortress tunnels with barricaded junctions, dead-end ambush pockets, and sinkholes.",
	}
	ui.addRandomEntry("Dungeon", "Dungeon Layout", chooseOne(entries))
}

func (ui *UI) generateRandomNPC() {
	traits := []string{"meticulous and paranoid", "charming but evasive", "idealistic and stubborn", "coldly pragmatic", "superstitious yet brave"}
	motivations := []string{"redeem a family disgrace", "secure rare medicine", "take revenge on a rival faction", "protect a hidden heir", "recover a stolen relic"}
	professions := []string{"quartermaster", "street physician", "cartographer", "dockmaster", "arcane scribe", "mercenary captain"}
	body := fmt.Sprintf("Name: %s\nTrait: %s\nMotivation: %s\nProfession: %s",
		chooseOne(randomNPCNames), chooseOne(traits), chooseOne(motivations), chooseOne(professions))
	ui.addRandomEntry("NPC & World", "NPC Profile", body)
}

func (ui *UI) generateRandomPlace() {
	taverns := []string{"The Crooked Lantern", "Salt & Stag", "Gilded Anchor", "Rusted Griffin"}
	ships := []string{"Blackwake", "Dawn Cartographer", "Ivory Keel", "Storm Reliquary"}
	fortresses := []string{"Bastion Khar", "Northwatch Hold", "Sunfall Redoubt", "Mourning Gate"}
	placeType := chooseOne([]string{"Tavern", "Ship", "Fortress"})
	name := ""
	switch placeType {
	case "Tavern":
		name = chooseOne(taverns)
	case "Ship":
		name = chooseOne(ships)
	default:
		name = chooseOne(fortresses)
	}
	ui.addRandomEntry("NPC & World", "Place Name", fmt.Sprintf("%s: %s", placeType, name))
}

func (ui *UI) generateRandomSocialEvent() {
	events := []string{
		"A trade dispute escalates into a street boycott between two ethnic quarters.",
		"A wedding alliance is publicly challenged, reopening an old blood-feud.",
		"A festival procession is interrupted by accusations of cultural sacrilege.",
		"Dock workers strike after a noble decree favors one community over another.",
		"A refugee caravan arrival triggers panic, price spikes, and faction propaganda.",
	}
	ui.addRandomEntry("NPC & World", "Social Event & Tension", chooseOne(events))
}

func (ui *UI) generateRandomTreasureTheme() {
	coins := []string{"74 gp, 210 sp, 120 cp", "310 gp, 45 pp", "95 gp, 900 cp, 2 trade bars"}
	gems := []string{"3x bloodstone (50 gp each)", "1x black pearl (500 gp)", "6x agate (10 gp each)"}
	art := []string{"gold filigree chalice", "ivory war-mask", "miniature silver astrolabe", "enameled dragon brooch"}
	body := fmt.Sprintf("Coins: %s\nGems: %s\nArt Object: %s", chooseOne(coins), chooseOne(gems), chooseOne(art))
	ui.addRandomEntry("Treasure & Items", "Treasure Cache", body)
}

func (ui *UI) generateRandomMagicItemTheme() {
	rarities := []string{"Common", "Uncommon", "Rare", "Very Rare"}
	cats := []string{"Arcana", "Armaments", "Implements", "Relics"}
	examples := []string{
		"wand with utility transmutation effect",
		"weapon granting situational elemental burst",
		"focus that improves concentration resilience",
		"relic tied to oath-based activation",
	}
	body := fmt.Sprintf("Rarity: %s\nCategory: %s\nItem Theme: %s", chooseOne(rarities), chooseOne(cats), chooseOne(examples))
	ui.addRandomEntry("Treasure & Items", "Magic Item Theme", body)
}

func (ui *UI) generateRandomCurrencyTheme() {
	packs := []string{
		"4 iron trade bars + 32 gp + carved amber bead",
		"2 silver ingots + 140 sp + lacquer token set",
		"1 electrum chain + 65 gp + 3 stamped guild scrips",
		"mixed coin purse: 20 pp, 85 gp, 140 sp",
	}
	ui.addRandomEntry("Treasure & Items", "Currency Mix", chooseOne(packs))
}

func (ui *UI) generateRandomAdventureEvent() {
	events := []string{
		"Wilderness encounter: territorial wyvern shadow circles the convoy trail.",
		"Chase sequence: suspect flees through market rooftops as guards block streets.",
		"Stronghold event: sabotage in granary threatens a week-long siege defense.",
		"Wilderness encounter: fey crossing opens during a thunderstorm at dusk.",
		"Stronghold event: quartermaster reports forged seals on armory manifests.",
	}
	ui.addRandomEntry("Adventure", "Adventure Event", chooseOne(events))
}

func (ui *UI) generateRandomPlotHook() {
	hooks := []string{
		"Divination: 'Beneath the third bell, iron drinks moonlight.'",
		"Hook: recover a ledger that can trigger a citywide succession crisis.",
		"Divination: 'The heir walks masked among oathbreakers.'",
		"Hook: escort a defector who knows siege-engine weak points.",
		"Hook: investigate why every oracle in the district dreams the same fire.",
	}
	ui.addRandomEntry("Adventure", "Divination & Plot Hook", chooseOne(hooks))
}

func sampleRows(pool []Monster, count int) []Monster {
	if count <= 0 || len(pool) == 0 {
		return nil
	}
	out := make([]Monster, 0, count)
	if len(pool) >= count {
		perm := rand.Perm(len(pool))
		for i := 0; i < count; i++ {
			out = append(out, pool[perm[i]])
		}
		return out
	}
	out = append(out, pool...)
	for len(out) < count {
		out = append(out, pool[rand.Intn(len(pool))])
	}
	return out
}

func itemLooksMagical(it Monster) bool {
	rarity := strings.ToLower(strings.TrimSpace(it.CR))
	if rarity != "" && rarity != "unknown" && rarity != "none" && rarity != "mundane" {
		return true
	}
	typ := strings.ToLower(strings.TrimSpace(it.Type))
	if strings.Contains(typ, "wondrous") ||
		strings.Contains(typ, "potion") ||
		strings.Contains(typ, "scroll") ||
		strings.Contains(typ, "wand") ||
		strings.Contains(typ, "rod") ||
		strings.Contains(typ, "ring") ||
		strings.Contains(typ, "staff") {
		return true
	}
	for _, key := range []string{"potion", "scroll", "staff", "wand", "rod", "ring", "wondrous"} {
		if b, ok := it.Raw[key].(bool); ok && b {
			return true
		}
	}
	return false
}

func itemLooksEquipment(it Monster) bool {
	typ := strings.ToLower(strings.TrimSpace(it.Type))
	if strings.Contains(typ, "weapon") ||
		strings.Contains(typ, "armor") ||
		strings.Contains(typ, "shield") ||
		strings.Contains(typ, "gear") ||
		strings.Contains(typ, "tool") ||
		strings.Contains(typ, "ammo") ||
		strings.Contains(typ, "adventuring") {
		return true
	}
	return !itemLooksMagical(it)
}

func simpleTitleCase(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	parts := strings.Fields(s)
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func randomShopOwners() string {
	if len(randomNPCNames) == 0 {
		return "Unknown"
	}
	first := strings.TrimSpace(randomNPCNames[rand.Intn(len(randomNPCNames))])
	if len(randomNPCNames) < 2 || rand.Intn(100) < 60 {
		return first
	}
	second := first
	for second == first {
		second = strings.TrimSpace(randomNPCNames[rand.Intn(len(randomNPCNames))])
	}
	return first + ", " + second
}

func formatPriceWithOscillation(raw map[string]any, percent int) string {
	base := formatItemBasePrice(raw)
	if strings.TrimSpace(base) == "" {
		if percent == 0 {
			return "n/a"
		}
		return fmt.Sprintf("n/a (%+d%%)", percent)
	}
	baseCp, ok := anyToInt64(raw["value"])
	if !ok || baseCp <= 0 {
		if percent == 0 {
			return base
		}
		return fmt.Sprintf("%s (%+d%%)", base, percent)
	}
	finalCp := int64(math.Round(float64(baseCp) * (100.0 + float64(percent)) / 100.0))
	if finalCp < 1 {
		finalCp = 1
	}
	return fmt.Sprintf("%s -> %s (%+d%%)", base, formatCopperValue(finalCp), percent)
}

func (ui *UI) openRandomMonsterEncounterTableForm() {
	if len(ui.monsters) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] random[-:-] no monsters available  %s", helpText))
		return
	}
	envOptions := append([]string{"All"}, ui.collectMonsterEnvironmentOptions()...)
	if len(envOptions) == 0 {
		envOptions = []string{"All"}
	}
	tierOptions := []string{
		"Tier 1 (CR 0-4)",
		"Tier 2 (CR 5-10)",
		"Tier 3 (CR 11-16)",
		"Tier 4 (CR 17+)",
	}

	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Random Monster Encounter Table ")
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || (event.Key() == tcell.KeyRune && event.Rune() == 'q') {
			ui.pages.RemovePage("random-encounter-table")
			ui.randomEncounterTableVisible = false
			ui.app.SetFocus(ui.list)
			return nil
		}
		return event
	})
	ui.randomEncounterTableVisible = true

	envDrop := tview.NewDropDown().SetLabel("Environment: ")
	envDrop.SetOptions(envOptions, nil)
	envDrop.SetCurrentOption(0)
	tierDrop := tview.NewDropDown().SetLabel("Tier: ")
	tierDrop.SetOptions(tierOptions, nil)
	tierDrop.SetCurrentOption(0)
	for _, dd := range []*tview.DropDown{envDrop, tierDrop} {
		dd.SetLabelColor(tcell.ColorGold)
		dd.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
		dd.SetFieldTextColor(tcell.ColorWhite)
		dd.SetListStyles(
			tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
			tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
		)
	}

	closeModal := func() {
		ui.pages.RemovePage("random-encounter-table")
		ui.randomEncounterTableVisible = false
		ui.app.SetFocus(ui.list)
	}
	runGenerate := func() {
		_, env := envDrop.GetCurrentOption()
		if strings.TrimSpace(env) == "" {
			env = "All"
		}
		tierIdx, _ := tierDrop.GetCurrentOption()
		if err := ui.generateRandomMonsterEncounterTable(env, tierIdx+1); err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] random[-:-] %v  %s", err, helpText))
			return
		}
		closeModal()
	}
	envDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if isSubmitKey(key) {
			form.SetFocus(1)
		}
	})
	envDrop.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			form.SetFocus(1)
			return nil
		case tcell.KeyBacktab:
			form.SetFocus(3) // Cancel button.
			return nil
		case tcell.KeyEnter:
			if !envDrop.IsOpen() {
				form.SetFocus(1)
				return nil
			}
			return event
		case tcell.KeyEscape:
			if envDrop.IsOpen() {
				return event
			}
			closeModal()
			return nil
		default:
			return event
		}
	})
	tierDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if key == tcell.KeyTab || key == tcell.KeyEnter {
			form.SetFocus(2) // Generate button.
		}
	})
	tierDrop.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			form.SetFocus(2)
			return nil
		case tcell.KeyBacktab:
			form.SetFocus(0)
			return nil
		case tcell.KeyEnter:
			if !tierDrop.IsOpen() {
				form.SetFocus(2)
				return nil
			}
			return event
		case tcell.KeyEscape:
			if tierDrop.IsOpen() {
				return event
			}
			closeModal()
			return nil
		default:
			return event
		}
	})
	form.AddFormItem(envDrop)
	form.AddFormItem(tierDrop)
	form.AddButton("Generate", runGenerate)
	form.AddButton("Cancel", closeModal)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 10, 0, true).
			AddItem(nil, 0, 1, false), 76, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("random-encounter-table", modal, true, true)
	ui.app.SetFocus(form)
}

func (ui *UI) generateRandomMonsterEncounterTable(environment string, tier int) error {
	if len(ui.monsters) == 0 {
		return errors.New("no monsters available")
	}
	if tier < 1 || tier > 4 {
		return fmt.Errorf("invalid tier %d", tier)
	}
	environment = strings.TrimSpace(environment)
	if environment == "" {
		environment = "All"
	}
	candidates := make([]Monster, 0, len(ui.monsters))
	for _, m := range ui.monsters {
		if strings.TrimSpace(m.Name) == "" {
			continue
		}
		if !strings.EqualFold(environment, "All") {
			match := false
			for _, env := range m.Environment {
				if strings.EqualFold(strings.TrimSpace(env), environment) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		cr, ok := crToFloat(m.CR)
		if !ok {
			continue
		}
		inTier := false
		switch {
		case tier == 1:
			inTier = cr >= 0 && cr <= 4
		case tier == 2:
			inTier = cr >= 5 && cr <= 10
		case tier == 3:
			inTier = cr >= 11 && cr <= 16
		default:
			inTier = cr >= 17
		}
		if inTier {
			candidates = append(candidates, m)
		}
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no monsters for env=%s tier=%d", environment, tier)
	}

	avgQty := 1
	switch tier {
	case 1:
		avgQty = 4
	case 2:
		avgQty = 2
	case 3:
		avgQty = 2
	case 4:
		avgQty = 1
	}
	lines := []string{
		fmt.Sprintf("Roll 1d12 for a random encounter table built from loaded manuals data (env: %s, tier: %d).", environment, tier),
		"",
		"1d12 Monster Encounter Table",
	}
	for i := 1; i <= 12; i++ {
		qty := avgQty
		if tier == 1 {
			qty = rand.Intn(4) + 2 // 2..5
		} else if tier == 2 {
			qty = rand.Intn(3) + 1 // 1..3
		} else if tier == 3 {
			qty = rand.Intn(2) + 1 // 1..2
		}
		choice := candidates[rand.Intn(len(candidates))]
		src := blankIfEmpty(strings.TrimSpace(choice.Source), "n/a")
		cr := blankIfEmpty(strings.TrimSpace(choice.CR), "n/a")
		lines = append(lines, fmt.Sprintf("%2d. %dx %s (CR %s, %s)", i, qty, choice.Name, cr, src))
	}
	ui.addRandomEntry("Encounter Tables", "Monster Encounter Table", strings.Join(lines, "\n"))
	ui.status.SetText(fmt.Sprintf(" [black:gold]random[-:-] generated monster table env=%s tier=%d  %s", environment, tier, helpText))
	return nil
}

func (ui *UI) generateRandomEquipmentShopTable() {
	if len(ui.items) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] random[-:-] no items available  %s", helpText))
		return
	}
	shopAdj := []string{"Iron", "Stout", "Bronze", "Weathered", "Oak", "Granite", "Rugged", "Frontier", "Anvil", "Ranger", "Shield", "Wayfarer"}
	shopNoun := []string{"Outfitters", "Armory", "Provisioner", "Supply House", "Gearworks", "Market", "Emporium", "Stockpile"}

	equipmentPool := make([]Monster, 0, len(ui.items))
	for _, it := range ui.items {
		if itemLooksEquipment(it) {
			equipmentPool = append(equipmentPool, it)
		}
	}
	if len(equipmentPool) == 0 {
		equipmentPool = append([]Monster(nil), ui.items...)
	}

	equipmentRows := sampleRows(equipmentPool, 12)
	lines := []string{
		"Roll 1d12. Entries are assembled from loaded manuals datasets.",
		"",
		"1d12 Equipment Shop Table",
	}
	priceSwing := []int{-20, -10, 0, 10, 20, 30, 50}
	for i, it := range equipmentRows {
		src := blankIfEmpty(strings.TrimSpace(it.Source), "n/a")
		typ := blankIfEmpty(strings.TrimSpace(it.Type), "gear")
		shopName := fmt.Sprintf("%s %s", shopAdj[i%len(shopAdj)], shopNoun[rand.Intn(len(shopNoun))])
		owners := randomShopOwners()
		price := formatPriceWithOscillation(it.Raw, priceSwing[rand.Intn(len(priceSwing))])
		lines = append(lines, fmt.Sprintf("%2d. %s - owners: %s - featured: %s (%s, %s) - price: %s", i+1, shopName, owners, it.Name, typ, src, price))
	}
	ui.addRandomEntry("Settlement Tables", "Equipment Shop Table", strings.Join(lines, "\n"))
}

func (ui *UI) generateRandomMagicShopTable() {
	if len(ui.items) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] random[-:-] no items available  %s", helpText))
		return
	}
	shopAdj := []string{"Arcane", "Eldritch", "Astral", "Moon", "Rune", "Mystic", "Sigil", "Aether", "Enchanted", "Oracle", "Spellbound", "Starlit"}
	shopNoun := []string{"Vault", "Emporium", "Boutique", "Cabinet", "Repository", "Curiosity Shop", "Sanctum", "Bazaar"}
	magicPool := make([]Monster, 0, len(ui.items))
	for _, it := range ui.items {
		if itemLooksMagical(it) {
			magicPool = append(magicPool, it)
		}
	}
	if len(magicPool) == 0 {
		magicPool = append([]Monster(nil), ui.items...)
	}
	magicRows := sampleRows(magicPool, 12)
	lines := []string{
		"Roll 1d12. Entries are assembled from loaded manuals datasets.",
		"",
		"1d12 Magic Shop Table",
	}
	for i, it := range magicRows {
		src := blankIfEmpty(strings.TrimSpace(it.Source), "n/a")
		rarity := simpleTitleCase(blankIfEmpty(strings.TrimSpace(it.CR), "unknown"))
		shopName := fmt.Sprintf("%s %s", shopAdj[i%len(shopAdj)], shopNoun[rand.Intn(len(shopNoun))])
		lines = append(lines, fmt.Sprintf("%2d. %s - %s (%s, %s)", i+1, shopName, it.Name, rarity, src))
	}
	ui.addRandomEntry("Settlement Tables", "Magic Shop Table", strings.Join(lines, "\n"))
}

func (ui *UI) deleteSelectedRandomEntry() {
	if len(ui.randoms) == 0 {
		return
	}
	listIdx := ui.list.GetCurrentItem()
	if listIdx < 0 || listIdx >= len(ui.filtered) {
		return
	}
	idx := ui.filtered[listIdx]
	if idx < 0 || idx >= len(ui.randoms) {
		return
	}
	removed := ui.randoms[idx].Name
	ui.randoms = append(ui.randoms[:idx], ui.randoms[idx+1:]...)
	ui.applyFilters()
	if len(ui.filtered) > 0 {
		next := min(listIdx, len(ui.filtered)-1)
		ui.list.SetCurrentItem(next)
		ui.renderDetailByListIndex(next)
	} else {
		ui.detailMeta.SetText("No random entry.")
		ui.detailRaw.SetText("")
		ui.rawText = ""
	}
	ui.status.SetText(fmt.Sprintf(" [black:gold]random[-:-] deleted %s  %s", removed, helpText))
}

func (ui *UI) clearAllRandomEntries() {
	if len(ui.randoms) == 0 {
		return
	}
	count := len(ui.randoms)
	ui.randoms = nil
	ui.applyFilters()
	ui.detailMeta.SetText("No random entry.")
	ui.detailRaw.SetText("")
	ui.rawText = ""
	ui.status.SetText(fmt.Sprintf(" [black:gold]random[-:-] cleared %d entries  %s", count, helpText))
}

func (ui *UI) saveRandomListAs(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	data := PersistedRandom{
		Version: 1,
		Items:   make([]PersistedRandomItem, 0, len(ui.randoms)),
	}
	for _, it := range ui.randoms {
		data.Items = append(data.Items, PersistedRandomItem{
			Name:      strings.TrimSpace(it.Name),
			Category:  strings.TrimSpace(it.CR),
			Generated: strings.TrimSpace(asString(it.Raw["generated"])),
			Content:   strings.TrimSpace(asString(it.Raw["content"])),
		})
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return err
	}
	ui.randomPath = path
	_ = writeLastRandomPath(path)
	return nil
}

func (ui *UI) loadRandomList(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var data PersistedRandom
	if err := yaml.Unmarshal(b, &data); err != nil {
		return err
	}
	items := make([]Monster, 0, len(data.Items))
	for i, it := range data.Items {
		name := strings.TrimSpace(it.Name)
		if name == "" {
			continue
		}
		category := strings.TrimSpace(it.Category)
		if category == "" {
			category = "Random"
		}
		items = append(items, Monster{
			ID:          i + 1,
			Name:        name,
			CR:          category,
			Environment: []string{category},
			Source:      "random",
			Type:        "generated",
			Raw: map[string]any{
				"name":      name,
				"category":  category,
				"generated": strings.TrimSpace(it.Generated),
				"content":   strings.TrimSpace(it.Content),
			},
		})
	}
	ui.randoms = items
	ui.randomPath = path
	_ = writeLastRandomPath(path)
	ui.applyFilters()
	if len(ui.filtered) > 0 {
		ui.list.SetCurrentItem(0)
		ui.renderDetailByListIndex(0)
	} else {
		ui.detailMeta.SetText("No random entry.")
		ui.detailRaw.SetText("")
		ui.rawText = ""
	}
	return nil
}

func (ui *UI) openRandomSaveAsInput() {
	input := tview.NewInputField().
		SetLabel("Random file ").
		SetFieldWidth(56)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetTitle(" Save Random List As ")
	input.SetBorder(true)
	input.SetTitleColor(tcell.ColorGold)
	input.SetBorderColor(tcell.ColorGold)
	input.SetText(ui.randomPath)

	closeModal := func() {
		ui.pages.RemovePage("random-save")
		ui.app.SetFocus(ui.list)
	}
	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			closeModal()
		case tcell.KeyEnter:
			path := strings.TrimSpace(input.GetText())
			if path == "" {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid file name[-:-]  %s", helpText))
				return
			}
			if err := ui.saveRandomListAs(path); err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] save error random list[-:-] %v  %s", err, helpText))
				return
			}
			ui.status.SetText(fmt.Sprintf(" [black:gold]random[-:-] saved %s  %s", ui.randomPath, helpText))
			closeModal()
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)
	ui.pages.AddPage("random-save", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openRandomLoadInput() {
	input := tview.NewInputField().
		SetLabel("Random file ").
		SetFieldWidth(56)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetTitle(" Load Random List ")
	input.SetBorder(true)
	input.SetTitleColor(tcell.ColorGold)
	input.SetBorderColor(tcell.ColorGold)
	input.SetText(ui.randomPath)

	closeModal := func() {
		ui.pages.RemovePage("random-load")
		ui.app.SetFocus(ui.list)
	}
	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			closeModal()
		case tcell.KeyEnter:
			path := strings.TrimSpace(input.GetText())
			if path == "" {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid file name[-:-]  %s", helpText))
				return
			}
			if err := ui.loadRandomList(path); err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] loading error random list[-:-] %v  %s", err, helpText))
				return
			}
			ui.status.SetText(fmt.Sprintf(" [black:gold]random[-:-] loaded %s  %s", ui.randomPath, helpText))
			closeModal()
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)
	ui.pages.AddPage("random-load", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) renderDetailByCustomEntry(entry EncounterEntry) {
	maxHP := ui.encounterMaxHP(entry)
	meta := strings.TrimSpace(entry.CustomMeta)
	if meta == "" {
		builder := &strings.Builder{}
		fmt.Fprintf(builder, "[yellow]%s[-]\n", ui.encounterEntryDisplay(entry))
		if init, ok := ui.encounterInitBase(entry); ok {
			if entry.HasInitRoll {
				fmt.Fprintf(builder, "[white]Init:[-] %d/%d\n", entry.InitRoll, init)
			} else {
				fmt.Fprintf(builder, "[white]Init:[-] %d\n", init)
			}
		}
		if strings.TrimSpace(entry.CustomAC) != "" {
			fmt.Fprintf(builder, "[white]AC:[-] %s\n", entry.CustomAC)
		}
		if entry.CustomLevel > 0 {
			fmt.Fprintf(builder, "[white]Level:[-] %d\n", entry.CustomLevel)
		}
		if maxHP > 0 {
			fmt.Fprintf(builder, "[white]HP:[-] %d/%d\n", entry.CurrentHP, maxHP)
		} else {
			fmt.Fprintf(builder, "[white]HP:[-] ?\n")
		}
		if entry.TempHP > 0 {
			fmt.Fprintf(builder, "[white]Temp HP:[-] %d\n", entry.TempHP)
		}
		meta = builder.String()
	}
	meta = ui.ensureEncounterPassivePerceptionLine(entry, meta)
	ui.detailMeta.SetText(meta)
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildCustomDescriptionText(entry, maxHP)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func buildMonsterDescriptionText(m Monster) string {
	raw := m.Raw
	b := &strings.Builder{}

	fmt.Fprintf(b, "Name: %s\n", m.Name)
	if src := strings.TrimSpace(m.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if t := strings.TrimSpace(m.Type); t != "" {
		fmt.Fprintf(b, "Type: %s\n", t)
	}
	if size := strings.TrimSpace(extractMonsterSize(raw["size"])); size != "" {
		fmt.Fprintf(b, "Size: %s\n", size)
	}
	if cr := strings.TrimSpace(m.CR); cr != "" {
		fmt.Fprintf(b, "Challenge: %s\n", cr)
	}
	if xp, ok := extractMonsterXP(raw, m.CR); ok {
		fmt.Fprintf(b, "XP: %d\n", xp)
	}
	if align := plainAny(raw["alignment"]); align != "" {
		fmt.Fprintf(b, "Alignment: %s\n", align)
	}
	if ac := extractAC(raw); ac != "" {
		fmt.Fprintf(b, "Armor Class: %s\n", ac)
	}
	hpAverage, hpFormula := extractHP(raw)
	if hpAverage != "" || hpFormula != "" {
		if hpAverage != "" && hpFormula != "" {
			fmt.Fprintf(b, "Hit Points: %s (%s)\n", hpAverage, hpFormula)
		} else if hpAverage != "" {
			fmt.Fprintf(b, "Hit Points: %s\n", hpAverage)
		} else {
			fmt.Fprintf(b, "Hit Points: %s\n", hpFormula)
		}
	}
	if speed := extractSpeed(raw); speed != "" {
		fmt.Fprintf(b, "Speed: %s\n", speed)
	}
	if s := abilityBlock(raw); s != "" {
		fmt.Fprintf(b, "\n%s\n", s)
	}

	orderedFields := []struct {
		key   string
		label string
	}{
		{"save", "Saving Throws"},
		{"skill", "Skills"},
		{"vulnerable", "Damage Vulnerabilities"},
		{"resist", "Damage Resistances"},
		{"immune", "Damage Immunities"},
		{"conditionImmune", "Condition Immunities"},
		{"senses", "Senses"},
		{"languages", "Languages"},
	}
	for _, f := range orderedFields {
		if txt := plainAny(raw[f.key]); txt != "" {
			fmt.Fprintf(b, "%s: %s\n", f.label, txt)
		}
	}

	sectionOrder := []struct {
		key   string
		label string
	}{
		{"trait", "Traits"},
		{"action", "Actions"},
		{"bonus", "Bonus Actions"},
		{"reaction", "Reactions"},
		{"legendary", "Legendary Actions"},
		{"mythic", "Mythic Actions"},
	}
	for _, sec := range sectionOrder {
		if body := plainSection(raw[sec.key]); body != "" {
			fmt.Fprintf(b, "\n%s\n%s\n", sec.label, body)
		}
	}
	return strings.TrimSpace(b.String())
}

func buildMonsterDescriptionTextScaled(m Monster, preview monsterScalePreview, scaled bool) string {
	base := buildMonsterDescriptionText(m)
	if !scaled {
		return base
	}
	scaledText := scaleDamageInText(base, preview.DamageMul)
	b := &strings.Builder{}
	fmt.Fprintf(b, "Scaled by CR step %+d (DMG benchmark)\n", preview.Step)
	fmt.Fprintf(b, "CR: %s -> %s\n", preview.BaseCR, preview.TargetCR)
	fmt.Fprintf(b, "AC: %d -> %d\n", preview.BaseAC, preview.TargetAC)
	fmt.Fprintf(b, "HP: %d -> %d\n", preview.BaseHP, preview.TargetHP)
	fmt.Fprintf(b, "Offense target: hit %+d, save DC %d, DPR %d-%d (x%.2f dmg)\n", preview.TargetAtk, preview.TargetDC, preview.DPRMin, preview.DPRMax, preview.DamageMul)
	fmt.Fprintf(b, "\n%s", scaledText)
	return b.String()
}

func scaleDamageInText(text string, mul float64) string {
	if strings.TrimSpace(text) == "" || mul <= 0 {
		return text
	}
	if math.Abs(mul-1.0) < 0.01 {
		return text
	}
	// Scale the displayed average in patterns like "8 (1d8 + 4) slashing damage".
	reWithParens := regexp.MustCompile(`\b(\d+)(\s*\([^)]*\)\s*[^.\n]*?\bdamage\b)`)
	out := reWithParens.ReplaceAllStringFunc(text, func(match string) string {
		sub := reWithParens.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		base, err := strconv.Atoi(sub[1])
		if err != nil || base <= 0 {
			return match
		}
		scaled := max(1, int(math.Round(float64(base)*mul)))
		return strconv.Itoa(scaled) + sub[2]
	})
	// Scale simple patterns like "takes 7 fire damage".
	reSimple := regexp.MustCompile(`\b(\d+)(\s+[^.\n]*?\bdamage\b)`)
	out = reSimple.ReplaceAllStringFunc(out, func(match string) string {
		sub := reSimple.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		// Skip already-processed patterns with parenthesized formula.
		if strings.Contains(sub[2], "(") {
			return match
		}
		base, err := strconv.Atoi(sub[1])
		if err != nil || base <= 0 {
			return match
		}
		scaled := max(1, int(math.Round(float64(base)*mul)))
		return strconv.Itoa(scaled) + sub[2]
	})
	return out
}

func scaleMonsterByCR(m Monster, step int) (monsterScalePreview, bool) {
	if step == 0 {
		return monsterScalePreview{}, false
	}
	baseIdx, ok := crBenchmarkIndex(m.CR)
	if !ok {
		return monsterScalePreview{}, false
	}
	targetIdx := max(baseIdx+step, 0)
	if targetIdx >= len(crBenchmarks) {
		targetIdx = len(crBenchmarks) - 1
	}
	if targetIdx == baseIdx {
		return monsterScalePreview{}, false
	}
	base := crBenchmarks[baseIdx]
	target := crBenchmarks[targetIdx]

	baseHP, ok := extractHPAverageInt(m.Raw)
	if !ok || baseHP <= 0 {
		baseHP = (base.HPMin + base.HPMax) / 2
	}
	baseAC := extractACInt(m.Raw)
	if baseAC <= 0 {
		baseAC = base.AC
	}
	baseMid := max(1, (base.HPMin+base.HPMax)/2)
	targetMid := max(1, (target.HPMin+target.HPMax)/2)
	ratio := float64(targetMid) / float64(baseMid)
	targetHP := max(1, int(math.Round(float64(baseHP)*ratio)))
	targetAC := max(1, baseAC+(target.AC-base.AC))

	baseMidDPR := float64(max(1, (base.DPRMin+base.DPRMax)/2))
	targetMidDPR := float64(max(1, (target.DPRMin+target.DPRMax)/2))
	damageMul := targetMidDPR / baseMidDPR

	return monsterScalePreview{
		BaseCR:     base.CR,
		TargetCR:   target.CR,
		Step:       targetIdx - baseIdx,
		BaseAC:     baseAC,
		TargetAC:   targetAC,
		BaseHP:     baseHP,
		TargetHP:   targetHP,
		BaseDPRMin: base.DPRMin,
		BaseDPRMax: base.DPRMax,
		TargetAtk:  target.Atk,
		TargetDC:   target.SaveDC,
		DPRMin:     target.DPRMin,
		DPRMax:     target.DPRMax,
		DamageMul:  damageMul,
	}, true
}

func crBenchmarkIndex(cr string) (int, bool) {
	s := strings.TrimSpace(strings.ToLower(cr))
	if s == "" {
		return 0, false
	}
	s = strings.TrimPrefix(s, "cr ")
	s = strings.TrimSpace(s)
	for i, b := range crBenchmarks {
		if s == strings.ToLower(b.CR) {
			return i, true
		}
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		best := -1
		bestDist := 1e9
		for i, b := range crBenchmarks {
			bv, ok := crToFloat(b.CR)
			if !ok {
				continue
			}
			d := math.Abs(v - bv)
			if d < bestDist {
				bestDist = d
				best = i
			}
		}
		if best >= 0 {
			return best, true
		}
	}
	return 0, false
}

func extractACInt(raw map[string]any) int {
	ac := extractAC(raw)
	re := regexp.MustCompile(`\d+`)
	m := re.FindString(ac)
	if m == "" {
		return 0
	}
	n, _ := strconv.Atoi(m)
	return n
}

func buildItemDescriptionText(it Monster) string {
	raw := it.Raw
	b := &strings.Builder{}

	fmt.Fprintf(b, "Name: %s\n", it.Name)
	if src := strings.TrimSpace(it.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if t := strings.TrimSpace(it.Type); t != "" {
		fmt.Fprintf(b, "Type: %s\n", t)
	}
	if rarity := strings.TrimSpace(it.CR); rarity != "" {
		fmt.Fprintf(b, "Rarity: %s\n", rarity)
	}
	if price := formatItemBasePrice(raw); price != "" {
		fmt.Fprintf(b, "Price: %s\n", price)
	}
	if req := strings.TrimSpace(asString(raw["reqAttune"])); req != "" {
		fmt.Fprintf(b, "Attunement: %s\n", req)
	}
	if weight := strings.TrimSpace(asString(raw["weight"])); weight != "" {
		fmt.Fprintf(b, "Weight: %s\n", weight)
	}
	if value := strings.TrimSpace(asString(raw["value"])); value != "" {
		fmt.Fprintf(b, "Value: %s\n", value)
	}
	if econ, ok := magicItemEconomy(raw, it.CR); ok {
		fmt.Fprintf(b, "Buy Cost: %s\n", econ.BuyCost)
		fmt.Fprintf(b, "Find Time in Shop: %s\n", econ.FindTime)
		fmt.Fprintf(b, "Craft Cost: %s\n", econ.CraftCost)
		fmt.Fprintf(b, "Craft Time: %s\n", econ.CraftTime)
		fmt.Fprintf(b, "Craft Procedure: %s\n", strings.Join(econ.Procedure, " -> "))
	}
	if entries := plainAny(raw["entries"]); entries != "" {
		fmt.Fprintf(b, "\nDescription\n%s\n", entries)
	}
	return strings.TrimSpace(b.String())
}

func buildSpellDescriptionText(sp Monster) string {
	raw := sp.Raw
	b := &strings.Builder{}

	fmt.Fprintf(b, "Name: %s\n", sp.Name)
	if src := strings.TrimSpace(sp.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if level := strings.TrimSpace(sp.CR); level != "" {
		fmt.Fprintf(b, "Level: %s\n", level)
	}
	if school := strings.TrimSpace(sp.Type); school != "" {
		fmt.Fprintf(b, "School: %s\n", school)
	}
	if cast := extractSpellTime(raw); cast != "" {
		fmt.Fprintf(b, "Casting Time: %s\n", cast)
	}
	if rng := extractSpellRange(raw); rng != "" {
		fmt.Fprintf(b, "Range: %s\n", rng)
	}
	if dur := extractSpellDuration(raw); dur != "" {
		fmt.Fprintf(b, "Duration: %s\n", dur)
	}
	if comps := plainAny(raw["components"]); comps != "" {
		fmt.Fprintf(b, "Components: %s\n", comps)
	}
	if entries := plainAny(raw["entries"]); entries != "" {
		fmt.Fprintf(b, "\nDescription\n%s\n", entries)
	}
	if higher := plainAny(raw["entriesHigherLevel"]); higher != "" {
		fmt.Fprintf(b, "\nAt Higher Levels\n%s\n", higher)
	}
	return strings.TrimSpace(b.String())
}

func buildClassDescriptionText(cl Monster) string {
	raw := cl.Raw
	b := &strings.Builder{}

	fmt.Fprintf(b, "%s\n", cl.Name)
	if src := strings.TrimSpace(cl.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if hd := strings.TrimSpace(cl.CR); hd != "" {
		fmt.Fprintf(b, "Hit Die: %s\n", hd)
	}
	if caster := strings.TrimSpace(cl.Type); caster != "" {
		fmt.Fprintf(b, "Caster Progression: %s\n", caster)
	}
	if len(cl.Environment) > 0 {
		fmt.Fprintf(b, "Primary Saves: %s\n", strings.Join(cl.Environment, ", "))
	}
	if spell := strings.TrimSpace(asString(raw["spellcastingAbility"])); spell != "" {
		fmt.Fprintf(b, "Spellcasting Ability: %s\n", strings.ToUpper(spell))
	}
	if prof := manualText(raw["proficiency"]); strings.TrimSpace(prof) != "" {
		fmt.Fprintf(b, "Saving Throw Proficiencies: %s\n", strings.ToUpper(prof))
	}
	if sp, ok := raw["startingProficiencies"].(map[string]any); ok {
		if armor := manualText(sp["armor"]); armor != "" {
			fmt.Fprintf(b, "Armor: %s\n", armor)
		}
		if weap := manualText(sp["weapons"]); weap != "" {
			fmt.Fprintf(b, "Weapons: %s\n", weap)
		}
		if tools := manualText(sp["tools"]); tools != "" {
			fmt.Fprintf(b, "Tools: %s\n", tools)
		}
		if skills := manualText(sp["skills"]); skills != "" {
			fmt.Fprintf(b, "Skills: %s\n", skills)
		}
	}
	if se, ok := raw["startingEquipment"].(map[string]any); ok {
		if starts := manualText(se["default"]); strings.TrimSpace(starts) != "" {
			fmt.Fprintf(b, "\nStarting Equipment\n%s\n", starts)
		}
		if gold := manualText(se["goldAlternative"]); strings.TrimSpace(gold) != "" {
			fmt.Fprintf(b, "Starting Gold: %s\n", gold)
		}
	}
	if mc := manualText(raw["multiclassing"]); strings.TrimSpace(mc) != "" {
		fmt.Fprintf(b, "\nMulticlassing\n%s\n", mc)
	}
	if features := extractClassFeatureNames(raw); len(features) > 0 {
		fmt.Fprintf(b, "\nClass Features\n%s\n", strings.Join(features, "\n"))
	}
	if features := extractSubclassFeatureNames(raw); len(features) > 0 {
		fmt.Fprintf(b, "\nSubclass Features\n%s\n", strings.Join(features, "\n"))
	}
	if details := renderClassFeatureDetails(raw["__classFeatureDetails"], false); details != "" {
		fmt.Fprintf(b, "\nClass Feature Details\n%s\n", details)
	}
	if details := renderClassFeatureDetails(raw["__subclassFeatureDetails"], true); details != "" {
		fmt.Fprintf(b, "\nSubclass Feature Details\n%s\n", details)
	}
	return strings.TrimSpace(b.String())
}

func renderClassFeatureDetails(v any, includeSubclass bool) string {
	items, ok := v.([]map[string]any)
	if !ok || len(items) == 0 {
		return ""
	}
	lines := make([]string, 0, len(items)*3)
	for _, it := range items {
		name := strings.TrimSpace(asString(it["feature"]))
		if name == "" {
			continue
		}
		level, _ := anyToInt(it["level"])
		header := name
		if level > 0 {
			header = fmt.Sprintf("Lv %d - %s", level, name)
		}
		if includeSubclass {
			sub := strings.TrimSpace(asString(it["subclass_name"]))
			if sub != "" {
				header = fmt.Sprintf("%s [%s]", header, sub)
			}
		}
		lines = append(lines, header)
		if body := strings.TrimSpace(manualSection(it["entries"])); body != "" {
			lines = append(lines, body)
		}
		lines = append(lines, "")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func extractClassFeatureNames(raw map[string]any) []string {
	if raw == nil {
		return nil
	}
	val, ok := raw["classFeatures"]
	if !ok || val == nil {
		return nil
	}
	items, ok := val.([]any)
	if !ok || len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, it := range items {
		s := strings.TrimSpace(asString(it))
		if s == "" {
			continue
		}
		name := featureNameFromToken(s)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

func extractSubclassFeatureNames(raw map[string]any) []string {
	if raw == nil {
		return nil
	}
	val, ok := raw["__subclassFeatures"]
	if !ok || val == nil {
		return nil
	}
	var items []any
	switch x := val.(type) {
	case []any:
		items = x
	case []string:
		items = make([]any, 0, len(x))
		for _, s := range x {
			items = append(items, s)
		}
	default:
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, it := range items {
		name := featureNameFromToken(strings.TrimSpace(asString(it)))
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

func featureNameFromToken(s string) string {
	name := strings.TrimSpace(s)
	if name == "" {
		return ""
	}
	if idx := strings.Index(name, "|"); idx >= 0 {
		name = strings.TrimSpace(name[:idx])
	}
	name = strings.TrimSpace(strings.TrimPrefix(name, "classFeature:"))
	name = strings.TrimSpace(strings.TrimPrefix(name, "subclassFeature:"))
	return name
}

func buildRaceDescriptionText(rc Monster) string {
	raw := rc.Raw
	b := &strings.Builder{}
	fmt.Fprintf(b, "%s\n", rc.Name)
	if src := strings.TrimSpace(rc.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if size := strings.TrimSpace(rc.CR); size != "" {
		fmt.Fprintf(b, "Size: %s\n", size)
	}
	if lineage := strings.TrimSpace(rc.Type); lineage != "" {
		fmt.Fprintf(b, "Lineage: %s\n", lineage)
	}
	if len(rc.Environment) > 0 {
		fmt.Fprintf(b, "Ability: %s\n", strings.Join(rc.Environment, ", "))
	}
	if speed := extractSpeed(raw); speed != "" {
		fmt.Fprintf(b, "Speed: %s\n", speed)
	}
	if langs := manualText(raw["languageProficiencies"]); langs != "" {
		fmt.Fprintf(b, "Languages: %s\n", langs)
	}
	if entries := manualSection(raw["entries"]); entries != "" {
		fmt.Fprintf(b, "\nTraits\n%s\n", entries)
	}
	return strings.TrimSpace(b.String())
}

func buildFeatDescriptionText(ft Monster) string {
	raw := ft.Raw
	b := &strings.Builder{}
	fmt.Fprintf(b, "%s\n", ft.Name)
	if src := strings.TrimSpace(ft.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if cat := strings.TrimSpace(ft.CR); cat != "" {
		fmt.Fprintf(b, "Category: %s\n", cat)
	}
	if ability := strings.TrimSpace(ft.Type); ability != "" {
		fmt.Fprintf(b, "Ability: %s\n", ability)
	}
	if len(ft.Environment) > 0 {
		fmt.Fprintf(b, "Prerequisite: %s\n", strings.Join(ft.Environment, ", "))
	}
	if rep, ok := raw["repeatable"].(bool); ok && rep {
		fmt.Fprintf(b, "Repeatable: yes\n")
	}
	if entries := manualSection(raw["entries"]); entries != "" {
		fmt.Fprintf(b, "\nBenefit\n%s\n", entries)
	}
	return strings.TrimSpace(b.String())
}

func buildBookDescriptionText(bk Monster) string {
	raw := bk.Raw
	b := &strings.Builder{}
	fmt.Fprintf(b, "Name: %s\n", bk.Name)
	if src := strings.TrimSpace(bk.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if len(bk.Environment) > 0 {
		fmt.Fprintf(b, "Group: %s\n", strings.Join(bk.Environment, ", "))
	}
	if published := strings.TrimSpace(bk.CR); published != "" {
		fmt.Fprintf(b, "Published: %s\n", published)
	}
	if author := strings.TrimSpace(bk.Type); author != "" {
		fmt.Fprintf(b, "Author: %s\n", author)
	}
	if id := strings.TrimSpace(asString(raw["id"])); id != "" {
		fmt.Fprintf(b, "ID: %s\n", id)
	}
	if contents := plainAny(raw["contents"]); contents != "" {
		fmt.Fprintf(b, "\nContents\n%s\n", contents)
	}
	return strings.TrimSpace(b.String())
}

func buildAdventureDescriptionText(ad Monster) string {
	raw := ad.Raw
	b := &strings.Builder{}
	fmt.Fprintf(b, "Name: %s\n", ad.Name)
	if src := strings.TrimSpace(ad.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if len(ad.Environment) > 0 {
		fmt.Fprintf(b, "Group: %s\n", strings.Join(ad.Environment, ", "))
	}
	if published := strings.TrimSpace(ad.CR); published != "" {
		fmt.Fprintf(b, "Published: %s\n", published)
	}
	if author := strings.TrimSpace(ad.Type); author != "" {
		fmt.Fprintf(b, "Author: %s\n", author)
	}
	if level := strings.TrimSpace(plainAny(raw["level"])); level != "" {
		fmt.Fprintf(b, "Level: %s\n", level)
	}
	if storyline := strings.TrimSpace(plainAny(raw["storyline"])); storyline != "" {
		fmt.Fprintf(b, "Storyline: %s\n", storyline)
	}
	if id := strings.TrimSpace(asString(raw["id"])); id != "" {
		fmt.Fprintf(b, "ID: %s\n", id)
	}
	if contents := plainAny(raw["contents"]); contents != "" {
		fmt.Fprintf(b, "\nContents\n%s\n", contents)
	}
	return strings.TrimSpace(b.String())
}

func full5eDataRoot() string {
	if p := strings.TrimSpace(os.Getenv("FIVETOOLS_DATA_DIR")); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, "Downloads", "5etools-w-img-2.24.2", "data")
}

func (ui *UI) fullBookDescriptionText(bk Monster) string {
	base := buildBookDescriptionText(bk)
	id := strings.TrimSpace(asString(bk.Raw["id"]))
	if fd, ok := bk.Raw["full_data"]; ok {
		if txt := strings.TrimSpace(format5eStructuredText(fd, 0)); txt != "" {
			return base + "\n\nFull Text\n" + txt
		}
	}
	if id == "" {
		return base
	}
	key := strings.ToLower(id)
	if txt, ok := ui.bookBodyCache[key]; ok {
		if txt == "" {
			return base
		}
		return base + "\n\nFull Text\n" + txt
	}
	txt := load5eEntryBody("book", id)
	ui.bookBodyCache[key] = txt
	if txt == "" {
		return base
	}
	return base + "\n\nFull Text\n" + txt
}

func (ui *UI) fullAdventureDescriptionText(ad Monster) string {
	base := buildAdventureDescriptionText(ad)
	id := strings.TrimSpace(asString(ad.Raw["id"]))
	if fd, ok := ad.Raw["full_data"]; ok {
		if txt := strings.TrimSpace(format5eStructuredText(fd, 0)); txt != "" {
			return base + "\n\nFull Text\n" + txt
		}
	}
	if id == "" {
		return base
	}
	key := strings.ToLower(id)
	if txt, ok := ui.advBodyCache[key]; ok {
		if txt == "" {
			return base
		}
		return base + "\n\nFull Text\n" + txt
	}
	txt := load5eEntryBody("adventure", id)
	ui.advBodyCache[key] = txt
	if txt == "" {
		return base
	}
	return base + "\n\nFull Text\n" + txt
}

func load5eEntryBody(kind string, id string) string {
	root := full5eDataRoot()
	if root == "" {
		return ""
	}
	file := filepath.Join(root, kind, fmt.Sprintf("%s-%s.json", kind, strings.ToLower(strings.TrimSpace(id))))
	b, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	var payload struct {
		Data any `json:"data"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return ""
	}
	txt := format5eStructuredText(payload.Data, 0)
	if strings.TrimSpace(txt) == "" {
		txt = manualText(payload.Data)
	}
	if strings.TrimSpace(txt) == "" {
		txt = plainAny(payload.Data)
	}
	return strings.TrimSpace(txt)
}

func format5eStructuredText(v any, depth int) string {
	switch t := v.(type) {
	case []any:
		parts := make([]string, 0, len(t))
		for _, it := range t {
			if s := strings.TrimSpace(format5eStructuredText(it, depth)); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n\n")
	case map[string]any:
		return format5eStructuredMap(t, depth)
	case string:
		return clean5eTags(strings.TrimSpace(t))
	default:
		return clean5eTags(strings.TrimSpace(asString(t)))
	}
}

func format5eStructuredMap(m map[string]any, depth int) string {
	kind := strings.ToLower(strings.TrimSpace(asString(m["type"])))
	name := clean5eTags(strings.TrimSpace(asString(m["name"])))
	id := strings.TrimSpace(asString(m["id"]))
	page := strings.TrimSpace(asString(m["page"]))

	header := name
	if header == "" && id != "" {
		header = strings.ToUpper(id)
	}
	if page != "" {
		if header == "" {
			header = fmt.Sprintf("Page %s", page)
		} else {
			header = fmt.Sprintf("%s (p.%s)", header, page)
		}
	}
	if header != "" {
		header = strings.Repeat("#", min(6, depth+2)) + " " + header
	}

	switch kind {
	case "entries", "section":
		body := format5eStructuredText(m["entries"], depth+1)
		return joinHeaderBody(header, body)
	case "inset", "insetreadaloud":
		body := format5eStructuredText(m["entries"], depth+1)
		if body != "" {
			lines := strings.Split(body, "\n")
			for i := range lines {
				lines[i] = "> " + lines[i]
			}
			body = strings.Join(lines, "\n")
		}
		return joinHeaderBody(header, body)
	case "quote":
		body := format5eStructuredText(m["entries"], depth+1)
		if body == "" {
			body = clean5eTags(strings.TrimSpace(asString(m["by"])))
		}
		if body != "" {
			lines := strings.Split(body, "\n")
			for i := range lines {
				lines[i] = "> " + lines[i]
			}
			body = strings.Join(lines, "\n")
		}
		return joinHeaderBody(header, body)
	case "list":
		items, _ := m["items"].([]any)
		lines := make([]string, 0, len(items))
		for _, it := range items {
			txt := strings.TrimSpace(format5eStructuredText(it, depth+1))
			if txt == "" {
				continue
			}
			txt = strings.ReplaceAll(txt, "\n", " ")
			lines = append(lines, "- "+txt)
		}
		return joinHeaderBody(header, strings.Join(lines, "\n"))
	case "table":
		var lines []string
		if capn := clean5eTags(strings.TrimSpace(plainAny(m["caption"]))); capn != "" {
			lines = append(lines, capn)
		}
		if labels, ok := m["colLabels"].([]any); ok && len(labels) > 0 {
			cols := make([]string, 0, len(labels))
			for _, l := range labels {
				cols = append(cols, clean5eTags(strings.TrimSpace(plainAny(l))))
			}
			lines = append(lines, strings.Join(cols, " | "))
		}
		if rows, ok := m["rows"].([]any); ok {
			for _, r := range rows {
				arr, ok := r.([]any)
				if !ok {
					continue
				}
				cols := make([]string, 0, len(arr))
				for _, c := range arr {
					cols = append(cols, clean5eTags(strings.TrimSpace(plainAny(c))))
				}
				lines = append(lines, strings.Join(cols, " | "))
			}
		}
		return joinHeaderBody(header, strings.Join(lines, "\n"))
	default:
		// Generic fallback for unknown node types.
		body := format5eStructuredText(m["entries"], depth+1)
		if body == "" {
			body = clean5eTags(strings.TrimSpace(plainAny(m)))
		}
		return joinHeaderBody(header, body)
	}
}

func joinHeaderBody(header string, body string) string {
	header = strings.TrimSpace(header)
	body = strings.TrimSpace(body)
	if header == "" {
		return body
	}
	if body == "" {
		return header
	}
	return header + "\n" + body
}

type generatedCharacterSheet struct {
	Meta      string
	Body      string
	SpellPlan string
	HP        int
	AC        int
	Init      int
}

type backgroundProfile struct {
	Name      string
	Skills    []string
	Tools     []string
	Languages []string
	Equipment []string
}

func generateCharacterSheetFromScores(cl Monster, rc Monster, level int, base [6]int) (meta string, body string) {
	sheet := generateCharacterSheetDataFromScores(cl, rc, level, base)
	return sheet.Meta, sheet.Body
}

func generateCharacterSheetData(cl Monster, rc Monster, level int) generatedCharacterSheet {
	base := rollBaseAbilityScores()
	return generateCharacterSheetDataFromScores(cl, rc, level, base)
}

func rollBaseAbilityScores() [6]int {
	base := [6]int{}
	for i := range 6 {
		base[i] = rollAbilityScore4d6DropLowest()
	}
	return base
}

func generateCharacterSheetDataFromScores(cl Monster, rc Monster, level int, base [6]int) generatedCharacterSheet {
	// STR, DEX, CON, INT, WIS, CHA
	labels := []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}
	keys := []string{"str", "dex", "con", "int", "wis", "cha"}
	bonuses := extractRaceAbilityBonuses(rc.Raw["ability"])
	scores := make([]int, 6)
	for i := range base {
		scores[i] = base[i] + bonuses[keys[i]]
	}
	mods := make([]int, 6)
	for i := range scores {
		mods[i] = abilityMod(scores[i])
	}

	prof := proficiencyBonusForLevel(level)
	hitDieFaces := classHitDieFaces(cl.CR)
	if hitDieFaces <= 0 {
		hitDieFaces = 8
	}
	hp := max(1, hitDieFaces+mods[2])
	if level > 1 {
		perLevel := max(1, (hitDieFaces/2)+1+mods[2])
		hp += (level - 1) * perLevel
	}
	ac := 10 + mods[1]
	init := mods[1]
	passive := 10 + mods[4]
	speed := extractSpeed(rc.Raw)
	if speed == "" {
		speed = "30 ft."
	}
	spellAbility := strings.TrimSpace(asString(cl.Raw["spellcastingAbility"]))
	spellDC := 0
	spellAtk := 0
	if spellAbility != "" {
		idx := abilityIndex(spellAbility)
		if idx >= 0 {
			spellDC = 8 + prof + mods[idx]
			spellAtk = prof + mods[idx]
		}
	}
	saveProf := map[string]struct{}{}
	for _, s := range cl.Environment {
		saveProf[strings.ToUpper(strings.TrimSpace(s))] = struct{}{}
	}
	bg := randomBackgroundProfile()
	classSkillChoiceCount, classSkillChoices := extractClassSkillChoices(cl.Raw)
	classSkillsPicked := chooseRandomN(classSkillChoices, classSkillChoiceCount)
	raceSkills := extractRaceSkillProficiencies(rc.Raw)
	allSkills := uniqueSortedStrings(append(append([]string{}, classSkillsPicked...), append(bg.Skills, raceSkills...)...))
	tools := uniqueSortedStrings(append(extractClassToolProficiencies(cl.Raw), bg.Tools...))
	languages := uniqueSortedStrings(append(extractRaceLanguages(rc.Raw), bg.Languages...))
	equipment := append(extractStartingEquipment(cl.Raw), bg.Equipment...)
	subclassTitle := strings.TrimSpace(asString(cl.Raw["subclassTitle"]))
	subclassLevels := extractSubclassFeatureLevels(cl.Raw["classFeatures"], level)
	classFeatures := extractClassFeaturesUpToLevel(cl.Raw["classFeatures"], level, 8)
	spellPlan := buildSpellPlan(cl.Raw, level)

	metaB := &strings.Builder{}
	fmt.Fprintf(metaB, "[yellow]%s %s Lv%d[-]\n", cl.Name, rc.Name, level)
	fmt.Fprintf(metaB, "[white]Background:[-] %s\n", bg.Name)
	if subclassTitle != "" {
		fmt.Fprintf(metaB, "[white]Subclass:[-] %s\n", subclassTitle)
	}
	fmt.Fprintf(metaB, "[white]AC:[-] %d\n", ac)
	fmt.Fprintf(metaB, "[white]HP:[-] %d\n", hp)
	fmt.Fprintf(metaB, "[white]Init:[-] %+d\n", init)
	fmt.Fprintf(metaB, "[white]Speed:[-] %s\n", speed)
	fmt.Fprintf(metaB, "[white]PB:[-] %+d\n", prof)
	if spellAbility != "" {
		fmt.Fprintf(metaB, "[white]Spell:[-] %s (DC %d, ATK %+d)\n", strings.ToUpper(spellAbility), spellDC, spellAtk)
	}

	bodyB := &strings.Builder{}
	fmt.Fprintf(bodyB, "%s %s (Level %d)\n", cl.Name, rc.Name, level)
	fmt.Fprintf(bodyB, "Source: %s / %s\n", blankIfEmpty(cl.Source, "n/a"), blankIfEmpty(rc.Source, "n/a"))
	fmt.Fprintf(bodyB, "Hit Die: d%d\n", hitDieFaces)
	fmt.Fprintf(bodyB, "Proficiency Bonus: %+d\n", prof)
	fmt.Fprintf(bodyB, "Armor Class: %d\n", ac)
	fmt.Fprintf(bodyB, "Hit Points: %d\n", hp)
	fmt.Fprintf(bodyB, "Initiative: %+d\n", init)
	fmt.Fprintf(bodyB, "Speed: %s\n", speed)
	fmt.Fprintf(bodyB, "Passive Perception: %d\n", passive)
	fmt.Fprintf(bodyB, "Background: %s\n", bg.Name)
	if subclassTitle != "" {
		fmt.Fprintf(bodyB, "Subclass Track: %s\n", subclassTitle)
	}
	if spellAbility != "" {
		fmt.Fprintf(bodyB, "Spellcasting Ability: %s\n", strings.ToUpper(spellAbility))
		fmt.Fprintf(bodyB, "Spell Save DC: %d\n", spellDC)
		fmt.Fprintf(bodyB, "Spell Attack Bonus: %+d\n", spellAtk)
	}
	if len(subclassLevels) > 0 {
		fmt.Fprintf(bodyB, "Subclass Feature Levels: %s\n", strings.Join(intSliceToStrings(subclassLevels), ", "))
	}


	fmt.Fprintf(bodyB, "\nAbilities\n")
	for i := range labels {
		fmt.Fprintf(bodyB, "%s %d (%+d)", labels[i], scores[i], mods[i])
		if b := bonuses[keys[i]]; b != 0 {
			fmt.Fprintf(bodyB, " [race %+d]", b)
		}
		fmt.Fprintf(bodyB, "\n")
	}

	fmt.Fprintf(bodyB, "\nSaving Throws\n")
	for i := range labels {
		val := mods[i]
		if _, ok := saveProf[labels[i]]; ok {
			val += prof
			fmt.Fprintf(bodyB, "%s %+d (proficient)\n", labels[i], val)
		} else {
			fmt.Fprintf(bodyB, "%s %+d\n", labels[i], val)
		}
	}
	if len(allSkills) > 0 {
		fmt.Fprintf(bodyB, "\nSkills\n%s\n", strings.Join(allSkills, ", "))
	}
	if len(tools) > 0 {
		fmt.Fprintf(bodyB, "\nTools\n%s\n", strings.Join(tools, ", "))
	}
	if len(languages) > 0 {
		fmt.Fprintf(bodyB, "\nLanguages\n%s\n", strings.Join(languages, ", "))
	}
	if len(equipment) > 0 {
		fmt.Fprintf(bodyB, "\nStarting Equipment\n")
		for _, item := range equipment {
			fmt.Fprintf(bodyB, "- %s\n", item)
		}
	}
	if len(classFeatures) > 0 {
		fmt.Fprintf(bodyB, "\nClass Features up to Level %d\n", level)
		for _, f := range classFeatures {
			fmt.Fprintf(bodyB, "- %s\n", f)
		}
	}
	return generatedCharacterSheet{
		Meta:      metaB.String(),
		Body:      bodyB.String(),
		SpellPlan: spellPlan,
		HP:        hp,
		AC:        ac,
		Init:      init,
	}
}

func classLevelsTotal(classes []CharacterClassLevel) int {
	total := 0
	for _, c := range classes {
		if c.Levels > 0 {
			total += c.Levels
		}
	}
	if total <= 0 {
		return 1
	}
	return total
}

func normalizeClassLevels(classes []CharacterClassLevel) []CharacterClassLevel {
	out := make([]CharacterClassLevel, 0, len(classes))
	seen := map[string]int{}
	for _, c := range classes {
		name := strings.TrimSpace(c.Name)
		if name == "" || c.Levels <= 0 {
			continue
		}
		key := strings.ToLower(name)
		if idx, ok := seen[key]; ok {
			out[idx].Levels += c.Levels
			continue
		}
		seen[key] = len(out)
		out = append(out, CharacterClassLevel{Name: name, Levels: c.Levels})
	}
	if len(out) == 0 {
		out = append(out, CharacterClassLevel{Name: "Fighter", Levels: 1})
	}
	return out
}

func classLevelsSummary(classes []CharacterClassLevel) string {
	parts := make([]string, 0, len(classes))
	for _, c := range classes {
		if strings.TrimSpace(c.Name) == "" || c.Levels <= 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d", c.Name, c.Levels))
	}
	if len(parts) == 0 {
		return "n/a"
	}
	return strings.Join(parts, ", ")
}

func parseBaseScores(v []int) [6]int {
	base := [6]int{10, 10, 10, 10, 10, 10}
	for i := 0; i < len(base) && i < len(v); i++ {
		if v[i] > 0 {
			base[i] = v[i]
		}
	}
	return base
}

func (ui *UI) findClassByName(name string) (Monster, bool) {
	n := strings.TrimSpace(strings.ToLower(name))
	if n == "" {
		return Monster{}, false
	}
	for _, cl := range ui.classes {
		if strings.EqualFold(strings.TrimSpace(cl.Name), n) {
			return cl, true
		}
	}
	return Monster{}, false
}

func (ui *UI) findRaceByName(name string) (Monster, bool) {
	n := strings.TrimSpace(strings.ToLower(name))
	if n == "" {
		return Monster{}, false
	}
	for _, rc := range ui.races {
		if strings.EqualFold(strings.TrimSpace(rc.Name), n) {
			return rc, true
		}
	}
	return Monster{}, false
}

func (ui *UI) primaryClassFromBuild(build CharacterBuild) (Monster, bool) {
	classes := normalizeClassLevels(build.Classes)
	bestName := ""
	bestLevels := -1
	for _, c := range classes {
		if c.Levels > bestLevels {
			bestLevels = c.Levels
			bestName = c.Name
		}
	}
	return ui.findClassByName(bestName)
}

func computeMultiClassHP(classes []CharacterClassLevel, conMod int, classesData map[string]Monster) int {
	total := 0
	first := true
	for _, c := range classes {
		if c.Levels <= 0 {
			continue
		}
		cl, ok := classesData[strings.ToLower(strings.TrimSpace(c.Name))]
		hitDieFaces := 8
		if ok {
			if hd := classHitDieFaces(cl.CR); hd > 0 {
				hitDieFaces = hd
			}
		}
		for i := 0; i < c.Levels; i++ {
			if first {
				total += max(1, hitDieFaces+conMod)
				first = false
			} else {
				total += max(1, (hitDieFaces/2)+1+conMod)
			}
		}
	}
	if total <= 0 {
		return 1
	}
	return total
}

func (ui *UI) generateCharacterSheetFromBuild(build CharacterBuild) (generatedCharacterSheet, CharacterBuild, error) {
	outBuild := CharacterBuild{
		Name:       strings.TrimSpace(build.Name),
		Race:       strings.TrimSpace(build.Race),
		Classes:    normalizeClassLevels(build.Classes),
		BaseScores: append([]int(nil), build.BaseScores...),
		Feats:      uniqueSortedStrings(build.Feats),
		Spells:     uniqueSortedStrings(build.Spells),
	}
	for _, c := range outBuild.Classes {
		if strings.TrimSpace(c.Name) == "" || c.Levels <= 0 {
			continue
		}
		if _, ok := ui.findClassByName(c.Name); !ok {
			return generatedCharacterSheet{}, outBuild, fmt.Errorf("class not found: %s", c.Name)
		}
	}
	if outBuild.Name == "" {
		outBuild.Name = "Character"
	}
	if len(outBuild.BaseScores) < 6 {
		base := rollBaseAbilityScores()
		outBuild.BaseScores = []int{base[0], base[1], base[2], base[3], base[4], base[5]}
	}
	base := parseBaseScores(outBuild.BaseScores)

	primary, ok := ui.primaryClassFromBuild(outBuild)
	if !ok {
		return generatedCharacterSheet{}, outBuild, fmt.Errorf("invalid primary class: %s", classLevelsSummary(outBuild.Classes))
	}
	race, ok := ui.findRaceByName(outBuild.Race)
	if !ok {
		if len(ui.races) == 0 {
			return generatedCharacterSheet{}, outBuild, fmt.Errorf("no race available")
		}
		race = ui.races[0]
		outBuild.Race = race.Name
	}
	totalLevel := classLevelsTotal(outBuild.Classes)
	sheet := generateCharacterSheetDataFromScores(primary, race, totalLevel, base)

	if len(outBuild.Classes) > 1 {
		classesData := map[string]Monster{}
		for _, cl := range ui.classes {
			classesData[strings.ToLower(strings.TrimSpace(cl.Name))] = cl
		}
		scores := make([]int, 6)
		mods := make([]int, 6)
		keys := []string{"str", "dex", "con", "int", "wis", "cha"}
		bonuses := extractRaceAbilityBonuses(race.Raw["ability"])
		for i := range 6 {
			scores[i] = base[i] + bonuses[keys[i]]
			mods[i] = abilityMod(scores[i])
		}
		sheet.HP = computeMultiClassHP(outBuild.Classes, mods[2], classesData)
	}
	metaLines := strings.Split(strings.TrimSpace(sheet.Meta), "\n")
	if len(metaLines) == 0 {
		metaLines = []string{fmt.Sprintf("[yellow]%s[-]", outBuild.Name)}
	} else {
		metaLines[0] = fmt.Sprintf("[yellow]%s[-]", outBuild.Name)
	}
	metaLines = append(metaLines, "[white]Build:[-] "+classLevelsSummary(outBuild.Classes))
	if len(outBuild.Feats) > 0 {
		metaLines = append(metaLines, "[white]Feats:[-] "+strings.Join(outBuild.Feats, ", "))
	}
	if len(outBuild.Spells) > 0 {
		metaLines = append(metaLines, "[white]Custom Spells:[-] "+strings.Join(outBuild.Spells, ", "))
	}
	sheet.Meta = strings.Join(metaLines, "\n")

	body := &strings.Builder{}
	fmt.Fprintf(body, "%s\n", outBuild.Name)
	fmt.Fprintf(body, "Race: %s\n", outBuild.Race)
	fmt.Fprintf(body, "Classes: %s\n", classLevelsSummary(outBuild.Classes))
	fmt.Fprintf(body, "Total Level: %d\n\n", totalLevel)
	body.WriteString(sheet.Body)
	if spellText := generateCharacterSpellSelection(primary, totalLevel, ui.spells); spellText != "" {
		if sheet.SpellPlan != "" {
			fmt.Fprintf(body, "\n\nSpellcasting Plan: %s", sheet.SpellPlan)
		}
		fmt.Fprintf(body, "\n\nSpells Prepared/Known\n%s", spellText)
	}
	if len(outBuild.Spells) > 0 {
		fmt.Fprintf(body, "\n\nCustom Spells\n- %s", strings.Join(outBuild.Spells, "\n- "))
	}
	if len(outBuild.Feats) > 0 {
		fmt.Fprintf(body, "\n\nFeats\n- %s", strings.Join(outBuild.Feats, "\n- "))
	}
	sheet.Body = strings.TrimSpace(body.String())
	return sheet, outBuild, nil
}

func rollAbilityScore4d6DropLowest() int {
	rolls := []int{rand.Intn(6) + 1, rand.Intn(6) + 1, rand.Intn(6) + 1, rand.Intn(6) + 1}
	sort.Ints(rolls)
	return rolls[1] + rolls[2] + rolls[3]
}

func extractRaceAbilityBonuses(v any) map[string]int {
	out := map[string]int{}
	arr, ok := v.([]any)
	if !ok {
		return out
	}
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		for _, k := range []string{"str", "dex", "con", "int", "wis", "cha"} {
			if val, ok := anyToInt(m[k]); ok {
				out[k] += val
			}
		}
	}
	return out
}

func abilityMod(score int) int {
	return (score / 2) - 5
}

func proficiencyBonusForLevel(level int) int {
	if level < 1 {
		level = 1
	}
	if level > 20 {
		level = 20
	}
	return 2 + (level-1)/4
}

func classHitDieFaces(hitDie string) int {
	s := strings.TrimSpace(strings.ToLower(hitDie))
	if after, ok := strings.CutPrefix(s, "d"); ok {
		if v, err := strconv.Atoi(after); err == nil && v > 0 {
			return v
		}
	}
	return 0
}

func abilityIndex(k string) int {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "str":
		return 0
	case "dex":
		return 1
	case "con":
		return 2
	case "int":
		return 3
	case "wis":
		return 4
	case "cha":
		return 5
	default:
		return -1
	}
}

func randomBackgroundProfile() backgroundProfile {
	backgrounds := []backgroundProfile{
		{Name: "Acolyte", Skills: []string{"Insight", "Religion"}, Languages: []string{"Any", "Any"}, Equipment: []string{"holy symbol", "prayer book", "vestments"}},
		{Name: "Criminal", Skills: []string{"Deception", "Stealth"}, Tools: []string{"thieves' tools", "gaming set"}, Equipment: []string{"crowbar", "dark common clothes"}},
		{Name: "Sage", Skills: []string{"Arcana", "History"}, Languages: []string{"Any", "Any"}, Equipment: []string{"ink", "quill", "small knife"}},
		{Name: "Soldier", Skills: []string{"Athletics", "Intimidation"}, Tools: []string{"gaming set", "vehicles (land)"}, Equipment: []string{"insignia of rank", "trophy from enemy"}},
		{Name: "Hermit", Skills: []string{"Medicine", "Religion"}, Tools: []string{"herbalism kit"}, Languages: []string{"Any"}, Equipment: []string{"scroll case", "herbalism kit"}},
		{Name: "Noble", Skills: []string{"History", "Persuasion"}, Tools: []string{"gaming set"}, Languages: []string{"Any"}, Equipment: []string{"set of fine clothes", "signet ring"}},
		{Name: "Outlander", Skills: []string{"Athletics", "Survival"}, Tools: []string{"musical instrument"}, Languages: []string{"Any"}, Equipment: []string{"staff", "hunting trap"}},
		{Name: "Charlatan", Skills: []string{"Deception", "Sleight of Hand"}, Tools: []string{"disguise kit", "forgery kit"}, Equipment: []string{"set of fine clothes", "disguise kit"}},
		{Name: "Entertainer", Skills: []string{"Acrobatics", "Performance"}, Tools: []string{"disguise kit", "musical instrument"}, Equipment: []string{"musical instrument", "costume"}},
		{Name: "Folk Hero", Skills: []string{"Animal Handling", "Survival"}, Tools: []string{"artisan's tools", "vehicles (land)"}, Equipment: []string{"artisan's tools", "shovel"}},
	}
	if len(backgrounds) == 0 {
		return backgroundProfile{Name: "Custom"}
	}
	return backgrounds[rand.Intn(len(backgrounds))]
}

func extractClassSkillChoices(raw map[string]any) (int, []string) {
	sp, ok := raw["startingProficiencies"].(map[string]any)
	if !ok {
		return 0, nil
	}
	skills, ok := sp["skills"].([]any)
	if !ok {
		return 0, nil
	}
	count := 0
	opts := []string{}
	for _, it := range skills {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		ch, ok := m["choose"].(map[string]any)
		if !ok {
			continue
		}
		if c, ok := anyToInt(ch["count"]); ok && c > 0 {
			count = c
		}
		for _, from := range asStringSlice(ch["from"]) {
			opts = append(opts, titleCase(strings.TrimSpace(from)))
		}
	}
	return count, uniqueSortedStrings(opts)
}

func extractClassToolProficiencies(raw map[string]any) []string {
	sp, ok := raw["startingProficiencies"].(map[string]any)
	if !ok {
		return nil
	}
	out := []string{}
	for _, key := range []string{"tools", "toolProficiencies"} {
		v, ok := sp[key]
		if !ok {
			continue
		}
		switch vv := v.(type) {
		case []any:
			for _, it := range vv {
				t := strings.TrimSpace(clean5eTags(asString(it)))
				if t != "" {
					out = append(out, t)
				}
				if m, ok := it.(map[string]any); ok {
					for k, mv := range m {
						if b, ok := mv.(bool); ok && b {
							out = append(out, strings.TrimSpace(k))
						}
					}
				}
			}
		case map[string]any:
			for k, mv := range vv {
				if b, ok := mv.(bool); ok && b {
					out = append(out, strings.TrimSpace(k))
				}
			}
		}
	}
	return uniqueSortedStrings(out)
}

func extractRaceLanguages(raw map[string]any) []string {
	v, ok := raw["languageProficiencies"]
	if !ok {
		return nil
	}
	out := []string{}
	if arr, ok := v.([]any); ok {
		for _, it := range arr {
			if m, ok := it.(map[string]any); ok {
				for k, mv := range m {
					switch tv := mv.(type) {
					case bool:
						if tv {
							out = append(out, titleCase(strings.TrimSpace(k)))
						}
					case int:
						if tv > 0 {
							out = append(out, titleCase(strings.TrimSpace(k)))
						}
					}
				}
			}
		}
	}
	if len(out) == 0 {
		txt := manualText(v)
		if txt != "" {
			out = append(out, txt)
		}
	}
	return uniqueSortedStrings(out)
}

func extractRaceSkillProficiencies(raw map[string]any) []string {
	v, ok := raw["skillProficiencies"]
	if !ok {
		return nil
	}
	out := []string{}
	if arr, ok := v.([]any); ok {
		for _, it := range arr {
			if m, ok := it.(map[string]any); ok {
				for k, mv := range m {
					if _, reserved := map[string]struct{}{"choose": {}, "from": {}, "count": {}}[k]; reserved {
						continue
					}
					switch mv.(type) {
					case int, bool:
						out = append(out, titleCase(strings.TrimSpace(k)))
					}
				}
			}
		}
	}
	if len(out) == 0 {
		txt := manualText(v)
		if txt != "" {
			out = append(out, txt)
		}
	}
	return uniqueSortedStrings(out)
}

func extractStartingEquipment(raw map[string]any) []string {
	se, ok := raw["startingEquipment"].(map[string]any)
	if !ok {
		return nil
	}
	def, ok := se["default"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(def)+1)
	for _, it := range def {
		t := strings.TrimSpace(manualText(it))
		if t != "" {
			out = append(out, t)
		}
	}
	if gold := strings.TrimSpace(manualText(se["goldAlternative"])); gold != "" {
		out = append(out, "Alternative starting gold: "+gold)
	}
	return out
}

func extractSubclassFeatureLevels(v any, maxLevel int) []int {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := []int{}
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if g, ok := m["gainSubclassFeature"].(bool); !ok || !g {
			continue
		}
		cf := asString(m["classFeature"])
		if cf == "" {
			continue
		}
		parts := strings.Split(cf, "|")
		for i := len(parts) - 1; i >= 0; i-- {
			lv, err := strconv.Atoi(strings.TrimSpace(parts[i]))
			if err == nil && lv > 0 && lv <= maxLevel {
				out = append(out, lv)
				break
			}
		}
	}
	sort.Ints(out)
	uniq := out[:0]
	prev := -1
	for _, v := range out {
		if v == prev {
			continue
		}
		uniq = append(uniq, v)
		prev = v
	}
	return uniq
}

func extractClassFeaturesUpToLevel(v any, maxLevel int, limit int) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	type feature struct {
		level int
		name  string
	}
	out := []feature{}
	for _, it := range arr {
		switch vv := it.(type) {
		case string:
			parts := strings.Split(vv, "|")
			if len(parts) == 0 {
				continue
			}
			lv := 0
			if len(parts) >= 4 {
				lv, _ = strconv.Atoi(strings.TrimSpace(parts[3]))
			}
			if lv <= 0 || lv > maxLevel {
				continue
			}
			name := strings.TrimSpace(parts[0])
			if name == "" {
				continue
			}
			out = append(out, feature{level: lv, name: clean5eTags(name)})
		case map[string]any:
			cf := strings.TrimSpace(asString(vv["classFeature"]))
			if cf == "" {
				continue
			}
			parts := strings.Split(cf, "|")
			if len(parts) == 0 {
				continue
			}
			lv := 0
			for i := len(parts) - 1; i >= 0; i-- {
				if n, err := strconv.Atoi(strings.TrimSpace(parts[i])); err == nil {
					lv = n
					break
				}
			}
			if lv <= 0 || lv > maxLevel {
				continue
			}
			name := strings.TrimSpace(strings.TrimPrefix(parts[0], "classFeature:"))
			if name == "" {
				continue
			}
			out = append(out, feature{level: lv, name: clean5eTags(name)})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].level != out[j].level {
			return out[i].level < out[j].level
		}
		return out[i].name < out[j].name
	})
	names := make([]string, 0, len(out))
	seen := map[string]struct{}{}
	for _, f := range out {
		key := fmt.Sprintf("%d|%s", f.level, f.name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, fmt.Sprintf("Lv%d %s", f.level, f.name))
		if limit > 0 && len(names) >= limit {
			break
		}
	}
	return names
}

func intSliceToStrings(v []int) []string {
	out := make([]string, 0, len(v))
	for _, n := range v {
		out = append(out, strconv.Itoa(n))
	}
	return out
}

func buildSpellPlan(raw map[string]any, level int) string {
	caster := strings.ToLower(strings.TrimSpace(asString(raw["casterProgression"])))
	if caster == "" || caster == "none" {
		return ""
	}
	cantrips := progressionValueAt(raw["cantripProgression"], level)
	maxSpellLevel := spellMaxLevelForProgression(caster, level)
	if cantrips <= 0 && maxSpellLevel <= 0 {
		return ""
	}
	parts := []string{}
	if cantrips > 0 {
		parts = append(parts, fmt.Sprintf("%d cantrips", cantrips))
	}
	if maxSpellLevel > 0 {
		parts = append(parts, fmt.Sprintf("max spell level %d", maxSpellLevel))
	}
	if slots := spellSlotsSummary(raw, level); slots != "" {
		parts = append(parts, slots)
	}
	return strings.Join(parts, " | ")
}

func progressionValueAt(v any, level int) int {
	if level < 1 {
		return 0
	}
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return 0
	}
	idx := level - 1
	if idx >= len(arr) {
		idx = len(arr) - 1
	}
	val, _ := anyToInt(arr[idx])
	return val
}

func spellMaxLevelForProgression(caster string, level int) int {
	if level <= 0 {
		return 0
	}
	switch caster {
	case "full":
		if level <= 2 {
			return 1
		}
		if level >= 17 {
			return 9
		}
		return (level + 1) / 2
	case "half":
		if level < 2 {
			return 0
		}
		if level >= 17 {
			return 5
		}
		return (level + 1) / 4
	case "third":
		if level < 3 {
			return 0
		}
		if level >= 19 {
			return 4
		}
		return (level + 2) / 6
	case "artificer":
		if level < 1 {
			return 0
		}
		if level >= 17 {
			return 5
		}
		return (level + 3) / 4
	case "pact":
		switch {
		case level >= 9:
			return 5
		case level >= 7:
			return 4
		case level >= 5:
			return 3
		case level >= 3:
			return 2
		case level >= 1:
			return 1
		default:
			return 0
		}
	default:
		return 0
	}
}

func spellSlotsSummary(raw map[string]any, level int) string {
	groups, ok := raw["classTableGroups"].([]any)
	if !ok {
		return ""
	}
	for _, g := range groups {
		m, ok := g.(map[string]any)
		if !ok {
			continue
		}
		rows, ok := m["rowsSpellProgression"].([]any)
		if !ok || len(rows) == 0 {
			continue
		}
		idx := max(level-1, 0)
		if idx >= len(rows) {
			idx = len(rows) - 1
		}
		row, ok := rows[idx].([]any)
		if !ok {
			continue
		}
		parts := []string{}
		for i, slot := range row {
			n, ok := anyToInt(slot)
			if !ok || n <= 0 {
				continue
			}
			parts = append(parts, fmt.Sprintf("%d:%d", i+1, n))
		}
		if len(parts) > 0 {
			return "slots " + strings.Join(parts, " ")
		}
	}
	return ""
}

func chooseRandomN(values []string, n int) []string {
	vals := uniqueSortedStrings(values)
	if n <= 0 || len(vals) == 0 {
		return nil
	}
	if n >= len(vals) {
		return vals
	}
	perm := rand.Perm(len(vals))
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, vals[perm[i]])
	}
	sort.Strings(out)
	return out
}

func uniqueSortedStrings(v []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(v))
	for _, s := range v {
		t := strings.TrimSpace(s)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func titleCase(s string) string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(s)))
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func generateCharacterSpellSelection(cl Monster, level int, spells []Monster) string {
	if len(spells) == 0 {
		return ""
	}
	raw := cl.Raw
	caster := strings.ToLower(strings.TrimSpace(asString(raw["casterProgression"])))
	maxLvl := spellMaxLevelForProgression(caster, level)
	cantripN := progressionValueAt(raw["cantripProgression"], level)
	knownN := progressionValueAt(raw["spellsKnownProgression"], level)
	if knownN <= 0 {
		knownN = progressionValueAt(raw["spellsKnownProgressionFixed"], level)
	}
	if knownN <= 0 && maxLvl > 0 {
		knownN = max(1, level/2+1)
	}
	if cantripN <= 0 && maxLvl <= 0 {
		return ""
	}

	cantripPool := make([]Monster, 0, 64)
	levelledPool := make([]Monster, 0, 256)
	for _, sp := range spells {
		lv, ok := spellLevelNumber(sp)
		if !ok {
			continue
		}
		if lv == 0 {
			cantripPool = append(cantripPool, sp)
			continue
		}
		if maxLvl > 0 && lv <= maxLvl {
			levelledPool = append(levelledPool, sp)
		}
	}

	cantrips := pickRandomSpellNames(cantripPool, cantripN)
	if knownN > 0 {
		if knownN > 20 {
			knownN = 20
		}
	}
	levelledMonsters := pickRandomSpells(levelledPool, knownN)
	if len(cantrips) == 0 && len(levelledMonsters) == 0 {
		return ""
	}

	b := &strings.Builder{}
	if len(cantrips) > 0 {
		fmt.Fprintf(b, "Cantrips (%d): %s\n", len(cantrips), strings.Join(cantrips, ", "))
	}
	if len(levelledMonsters) > 0 {
		// Group by spell level
		byLevel := map[int][]string{}
		for _, sp := range levelledMonsters {
			lv, ok := spellLevelNumber(sp)
			if !ok {
				continue
			}
			byLevel[lv] = append(byLevel[lv], strings.TrimSpace(sp.Name))
		}
		levels := make([]int, 0, len(byLevel))
		for lv := range byLevel {
			levels = append(levels, lv)
		}
		sort.Ints(levels)
		for _, lv := range levels {
			names := byLevel[lv]
			sort.Strings(names)
			fmt.Fprintf(b, "Level %d: %s\n", lv, strings.Join(names, ", "))
		}
	}
	return strings.TrimSpace(b.String())
}

func spellLevelNumber(sp Monster) (int, bool) {
	s := strings.TrimSpace(strings.ToLower(sp.CR))
	if s == "" {
		return 0, false
	}
	switch s {
	case "0", "cantrip", "c":
		return 0, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	if n < 0 {
		return 0, false
	}
	return n, true
}

func pickRandomSpellNames(spells []Monster, count int) []string {
	if count <= 0 || len(spells) == 0 {
		return nil
	}
	if count > len(spells) {
		count = len(spells)
	}
	perm := rand.Perm(len(spells))
	out := make([]string, 0, count)
	seen := map[string]struct{}{}
	for _, idx := range perm {
		name := strings.TrimSpace(spells[idx].Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
		if len(out) >= count {
			break
		}
	}
	sort.Strings(out)
	return out
}

func pickRandomSpells(spells []Monster, count int) []Monster {
	if count <= 0 || len(spells) == 0 {
		return nil
	}
	if count > len(spells) {
		count = len(spells)
	}
	perm := rand.Perm(len(spells))
	out := make([]Monster, 0, count)
	seen := map[string]struct{}{}
	for _, idx := range perm {
		name := strings.ToLower(strings.TrimSpace(spells[idx].Name))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, spells[idx])
		if len(out) >= count {
			break
		}
	}
	return out
}

func manualSection(v any) string {
	if _, ok := v.([]any); ok {
		return manualText(plainSection(v))
	}
	return manualText(v)
}

func manualText(v any) string {
	txt := plainAny(v)
	if txt == "" {
		return ""
	}
	return clean5eTags(txt)
}

func clean5eTags(s string) string {
	tagRe := regexp.MustCompile(`\{@[^}]+\}`)
	out := tagRe.ReplaceAllStringFunc(s, func(tag string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(tag, "{@"), "}")
		inner = strings.TrimSpace(inner)
		if inner == "" {
			return ""
		}
		if inner == "h" {
			return "Hit:"
		}
		parts := strings.SplitN(inner, " ", 2)
		kind := strings.TrimSpace(parts[0])
		rest := ""
		if len(parts) > 1 {
			rest = strings.TrimSpace(parts[1])
		}
		if rest == "" {
			return ""
		}
		opts := strings.Split(rest, "|")
		display := strings.TrimSpace(opts[0])
		if len(opts) >= 3 && strings.TrimSpace(opts[2]) != "" {
			display = strings.TrimSpace(opts[2])
		}
		switch kind {
		case "dc":
			return "DC " + display
		case "hit":
			if strings.HasPrefix(display, "+") || strings.HasPrefix(display, "-") {
				return display
			}
			return "+" + display
		default:
			return display
		}
	})
	out = strings.ReplaceAll(out, "  ", " ")
	out = strings.ReplaceAll(out, " ,", ",")
	return strings.TrimSpace(out)
}

func buildCustomDescriptionText(entry EncounterEntry, maxHP int) string {
	if strings.TrimSpace(entry.CustomBody) != "" {
		return strings.TrimSpace(entry.CustomBody)
	}
	b := &strings.Builder{}
	fmt.Fprintf(b, "Name: %s\n", entry.CustomName)
	if entry.CustomLevel > 0 {
		fmt.Fprintf(b, "Level: %d\n", entry.CustomLevel)
	}
	fmt.Fprintf(b, "Initiative: %d\n", entry.CustomInit)
	if entry.HasInitRoll {
		fmt.Fprintf(b, "Initiative Roll: %d\n", entry.InitRoll)
	}
	if strings.TrimSpace(entry.CustomAC) != "" {
		fmt.Fprintf(b, "Armor Class: %s\n", entry.CustomAC)
	}
	if maxHP > 0 {
		fmt.Fprintf(b, "Hit Points: %d/%d\n", entry.CurrentHP, maxHP)
	} else {
		fmt.Fprintf(b, "Hit Points: ?\n")
	}
	if entry.TempHP > 0 {
		fmt.Fprintf(b, "Temp HP: %d\n", entry.TempHP)
	}
	if len(entry.Conditions) > 0 {
		parts := []string{}
		for _, d := range encounterConditionDefs {
			if r := entry.Conditions[d.Code]; r > 0 {
				parts = append(parts, fmt.Sprintf("%s %d", d.Name, r))
			}
		}
		if len(parts) > 0 {
			fmt.Fprintf(b, "Conditions: %s\n", strings.Join(parts, ", "))
		}
	}
	return strings.TrimSpace(b.String())
}

func abilityBlock(raw map[string]any) string {
	keys := []string{"str", "dex", "con", "int", "wis", "cha"}
	labels := []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}
	values := make([]int, len(keys))
	for i, k := range keys {
		v, ok := anyToInt(raw[k])
		if !ok {
			return ""
		}
		values[i] = v
	}
	b := &strings.Builder{}
	fmt.Fprintf(b, "%s  %s  %s  %s  %s  %s\n", labels[0], labels[1], labels[2], labels[3], labels[4], labels[5])
	for i, v := range values {
		if i > 0 {
			b.WriteString("  ")
		}
		mod := (v / 2) - 5
		fmt.Fprintf(b, "%2d (%+d)", v, mod)
	}
	return b.String()
}

func abilityInline(raw map[string]any) string {
	keys := []string{"str", "dex", "con", "int", "wis", "cha"}
	labels := []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}
	parts := make([]string, 0, len(keys))
	for i, k := range keys {
		v, ok := anyToInt(raw[k])
		if !ok {
			return ""
		}
		parts = append(parts, fmt.Sprintf("%s %d (%+d)", labels[i], v, abilityMod(v)))
	}
	return strings.Join(parts, " ")
}

func plainSection(v any) string {
	items, ok := v.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	lines := make([]string, 0, len(items))
	for _, it := range items {
		switch x := it.(type) {
		case map[string]any:
			if table := plainTable(x); table != "" {
				lines = append(lines, table)
				continue
			}
			name := strings.TrimSpace(asString(x["name"]))
			body := ""
			switch entries := x["entries"].(type) {
			case []any:
				body = strings.TrimSpace(plainSection(entries))
			default:
				body = strings.TrimSpace(plainAny(entries))
			}
			if name != "" && body != "" {
				lines = append(lines, fmt.Sprintf("%s. %s", name, body))
			} else if name != "" {
				lines = append(lines, name)
			} else if body != "" {
				lines = append(lines, body)
			}
		default:
			txt := strings.TrimSpace(plainAny(it))
			if txt != "" {
				lines = append(lines, txt)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func plainTable(m map[string]any) string {
	if m == nil {
		return ""
	}
	if _, hasRows := m["rows"]; !hasRows {
		if !strings.EqualFold(strings.TrimSpace(asString(m["type"])), "table") {
			return ""
		}
	}
	lines := make([]string, 0, 8)
	if capn := strings.TrimSpace(plainAny(m["caption"])); capn != "" {
		lines = append(lines, capn)
	}
	if labels, ok := m["colLabels"].([]any); ok && len(labels) > 0 {
		cols := make([]string, 0, len(labels))
		for _, l := range labels {
			col := strings.TrimSpace(plainAny(l))
			if col == "" {
				col = "-"
			}
			cols = append(cols, col)
		}
		lines = append(lines, strings.Join(cols, " | "))
	}
	if rows, ok := m["rows"].([]any); ok {
		for _, r := range rows {
			arr, ok := r.([]any)
			if !ok {
				txt := strings.TrimSpace(plainAny(r))
				if txt != "" {
					lines = append(lines, txt)
				}
				continue
			}
			cols := make([]string, 0, len(arr))
			for _, c := range arr {
				col := strings.TrimSpace(plainAny(c))
				cols = append(cols, col)
			}
			lines = append(lines, strings.Join(cols, " | "))
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var reToolsTag = regexp.MustCompile(`\{@(\w+)\s*([^}]*)\}`)

// stripTags converts 5etools inline tags (e.g. {@atkr m}, {@damage 1d4 + 2})
// to plain readable text matching the Monster Manual style.
func stripTags(s string) string {
	return reToolsTag.ReplaceAllStringFunc(s, func(match string) string {
		m := reToolsTag.FindStringSubmatch(match)
		if len(m) < 3 {
			return match
		}
		tag, content := m[1], strings.TrimSpace(m[2])
		switch tag {
		// ── Attack rolls ────────────────────────────────────────────────
		case "atkr":
			switch content {
			case "m":
				return "Melee Attack Roll:"
			case "r":
				return "Ranged Attack Roll:"
			case "m,r", "r,m":
				return "Melee or Ranged Attack Roll:"
			default:
				return "Attack Roll:"
			}
		case "atk":
			switch content {
			case "m":
				return "Melee Attack Roll:"
			case "r":
				return "Ranged Attack Roll:"
			case "mw":
				return "Melee Weapon Attack:"
			case "rw":
				return "Ranged Weapon Attack:"
			case "ms":
				return "Melee Spell Attack:"
			case "rs":
				return "Ranged Spell Attack:"
			case "mw,rw", "rw,mw":
				return "Melee or Ranged Weapon Attack:"
			case "ms,rs", "rs,ms":
				return "Melee or Ranged Spell Attack:"
			case "m,r", "r,m":
				return "Melee or Ranged Attack Roll:"
			default:
				return "Attack:"
			}
		// ── Hit / damage / DC ────────────────────────────────────────────
		case "hit":
			n, err := strconv.Atoi(content)
			if err != nil {
				return "+" + content
			}
			if n >= 0 {
				return fmt.Sprintf("+%d", n)
			}
			return fmt.Sprintf("%d", n)
		case "h":
			return "Hit: " // trailing space – tag is immediately followed by the number
		case "hom":
			return "Hit or Miss: "
		case "damage":
			d := strings.ReplaceAll(content, " + ", "+")
			d = strings.ReplaceAll(d, " - ", "-")
			return d
		case "dc":
			return "DC " + content
		// ── 2024 action-block tags ────────────────────────────────────────
		case "actSave":
			ability := map[string]string{
				"str": "Strength", "dex": "Dexterity", "con": "Constitution",
				"int": "Intelligence", "wis": "Wisdom", "cha": "Charisma",
			}
			if full, ok := ability[content]; ok {
				return full + " Saving Throw:"
			}
			return strings.ToUpper(content[:1]) + content[1:] + " Saving Throw:"
		case "actSaveFail":
			switch content {
			case "":
				return "Fail:"
			case "1":
				return "Fail (first save):"
			case "2":
				return "Fail (repeated save):"
			default:
				return "Fail:"
			}
		case "actSaveSuccess":
			return "Success:"
		case "actSaveSuccessOrFail":
			return "Success or Fail:"
		case "actTrigger":
			return "Trigger:"
		case "actResponse":
			return "Response:"
		case "hitYourSpellAttack":
			if content != "" {
				return content
			}
			return "Spell Attack Roll: your spell attack modifier"
		// ── Recharge ─────────────────────────────────────────────────────
		case "recharge":
			if content == "" {
				return "(Recharge 6)"
			}
			return fmt.Sprintf("(Recharge %s–6)", content)
		// ── Dice expressions ─────────────────────────────────────────────
		case "dice":
			parts := strings.SplitN(content, ";", 2)
			return strings.TrimSpace(parts[0])
		case "scaledice", "scaledamage":
			parts := strings.SplitN(content, "|", 2)
			return strings.TrimSpace(parts[0])
		case "chance":
			return content + " percent"
		case "skillCheck":
			parts := strings.Fields(content)
			if len(parts) >= 2 {
				skill := strings.Title(strings.ReplaceAll(parts[0], "_", " "))
				return fmt.Sprintf("DC %s %s check", parts[1], skill)
			}
			return strings.ReplaceAll(content, "_", " ")
		// ── Text-formatting tags ──────────────────────────────────────────
		case "b", "bold", "i", "italic", "s", "u", "sup", "sub", "color":
			// {@color text|color} – only the text part
			parts := strings.SplitN(content, "|", 2)
			return strings.TrimSpace(parts[0])
		case "note":
			return ""
		// ── Everything else: reference tags ──────────────────────────────
		// condition, spell, creature, item, skill, sense, feat, class,
		// variantrule, action, filter, table, hazard, status, quickref, …
		default:
			display := strings.TrimSpace(strings.SplitN(content, "|", 2)[0])
			// Strip area-of-effect suffixes: "Cone [Area of Effect]" → "Cone"
			if idx := strings.Index(display, " ["); idx >= 0 {
				display = display[:idx]
			}
			return display
		}
	})
}

func plainAny(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(stripTags(x))
	case int, int8, int16, int32, int64, float32, float64, bool:
		return strings.TrimSpace(fmt.Sprintf("%v", x))
	case []string:
		return strings.Join(x, ", ")
	case []any:
		out := make([]string, 0, len(x))
		for _, it := range x {
			txt := strings.TrimSpace(plainAny(it))
			if txt != "" {
				out = append(out, txt)
			}
		}
		return strings.Join(out, ", ")
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, 0, len(keys))
		for _, k := range keys {
			txt := strings.TrimSpace(plainAny(x[k]))
			if txt != "" {
				pairs = append(pairs, fmt.Sprintf("%s %s", k, txt))
			}
		}
		return strings.Join(pairs, ", ")
	case map[any]any:
		tmp := make(map[string]any, len(x))
		for k, vv := range x {
			tmp[asString(k)] = vv
		}
		return plainAny(tmp)
	default:
		return strings.TrimSpace(asString(v))
	}
}

func (ui *UI) openRawSearch(returnFocus tview.Primitive) {
	input := tview.NewInputField().
		SetLabel("/ ").
		SetFieldWidth(40)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Find In Description ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	input.SetText(ui.rawQuery)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 52, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("raw-search")
		ui.app.SetFocus(returnFocus)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		query := strings.TrimSpace(input.GetText())
		if query == "" {
			ui.rawQuery = ""
			ui.rawMatchLine = -1
			ui.rawMatchOcc = -1
			ui.renderRawWithHighlight("", -1)
			ui.status.SetText(helpText)
			return
		}
		line, occ, ok := ui.findNextRawOccurrence(query, -1, -1, true)
		if !ok {
			ui.rawQuery = query
			ui.rawMatchLine = -1
			ui.rawMatchOcc = -1
			ui.renderRawWithHighlight(query, -1)
			ui.status.SetText(fmt.Sprintf(" [white:red] no match in Description [-:-] \"%s\"  %s", query, helpText))
			return
		}
		ui.rawQuery = query
		ui.rawMatchLine = line
		ui.rawMatchOcc = occ
		ui.renderRawWithHighlightOccurrence(query, line, occ)
		ui.detailRaw.ScrollToHighlight()
		ui.status.SetText(ui.rawSearchFoundStatus(query, line, occ))
	})

	ui.pages.AddPage("raw-search", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) repeatRawSearch(forward bool) {
	query := strings.TrimSpace(ui.rawQuery)
	if query == "" {
		ui.status.SetText(fmt.Sprintf(" [white:red] no active search in Description [-:-]  %s", helpText))
		return
	}
	startLine := ui.rawMatchLine
	startOcc := ui.rawMatchOcc
	if startLine < 0 || !strings.EqualFold(ui.rawQuery, query) {
		startLine, _ = ui.detailRaw.GetScrollOffset()
		if forward {
			startOcc = -1
		} else {
			startOcc = 0
		}
	}
	line, occ, ok := ui.findNextRawOccurrence(query, startLine, startOcc, forward)
	if !ok {
		ui.rawMatchLine = -1
		ui.rawMatchOcc = -1
		ui.renderRawWithHighlight(query, -1)
		ui.status.SetText(fmt.Sprintf(" [white:red] no match in Description [-:-] \"%s\"  %s", query, helpText))
		return
	}
	ui.rawMatchLine = line
	ui.rawMatchOcc = occ
	ui.renderRawWithHighlightOccurrence(query, line, occ)
	ui.detailRaw.ScrollToHighlight()
	ui.status.SetText(ui.rawSearchFoundStatus(query, line, occ))
}

func (ui *UI) rawSearchCounter(query string, line, occ int) (int, int, bool) {
	if strings.TrimSpace(query) == "" || ui.rawText == "" {
		return 0, 0, false
	}
	lines := strings.Split(ui.rawText, "\n")
	total := 0
	current := 0
	for i, row := range lines {
		count := rawLineMatchCount(row, query)
		if count <= 0 {
			continue
		}
		if i == line && occ >= 0 && occ < count {
			current = total + occ + 1
		}
		total += count
	}
	if total <= 0 || current <= 0 {
		return 0, total, false
	}
	return current, total, true
}

func (ui *UI) rawSearchFoundStatus(query string, line, occ int) string {
	if cur, total, ok := ui.rawSearchCounter(query, line, occ); ok {
		return fmt.Sprintf(" [black:gold] trovato nella Description[-:-] \"%s\" (riga %d, match %d/%d)  %s", query, line+1, cur, total, helpText)
	}
	return fmt.Sprintf(" [black:gold] trovato nella Description[-:-] \"%s\" (riga %d)  %s", query, line+1, helpText)
}

func (ui *UI) openEncounterSaveAsInput() {
	input := tview.NewInputField().
		SetLabel("File: ").
		SetFieldWidth(52)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Save Encounters As ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	input.SetText(ui.encountersPath)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("encounter-saveas")
		ui.app.SetFocus(ui.encounter)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		path := strings.TrimSpace(input.GetText())
		if path == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid file name[-:-]  %s", helpText))
			return
		}
		if err := ui.saveEncountersAs(path); err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] save error encounters[-:-] %v  %s", err, helpText))
			return
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold] saved[-:-] %s  %s", ui.encountersPath, helpText))
	})

	ui.pages.AddPage("encounter-saveas", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openEncounterLoadInput() {
	input := tview.NewInputField().
		SetLabel("File: ").
		SetFieldWidth(52)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Load Encounters ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	input.SetText(ui.encountersPath)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("encounter-load")
		ui.app.SetFocus(ui.encounter)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		path := strings.TrimSpace(input.GetText())
		if path == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid file name[-:-]  %s", helpText))
			return
		}

		prev := ui.encountersPath
		ui.encountersPath = path
		if err := ui.loadEncounters(); err != nil {
			ui.encountersPath = prev
			ui.status.SetText(fmt.Sprintf(" [white:red] loading error encounters[-:-] %v  %s", err, helpText))
			return
		}
		ui.encounterUndo = ui.encounterUndo[:0]
		ui.encounterRedo = ui.encounterRedo[:0]
		ui.renderEncounterList()
		if len(ui.encounterItems) > 0 {
			idx := 0
			if ui.turnMode {
				idx = ui.turnIndex
			}
			if idx < 0 || idx >= len(ui.encounterItems) {
				idx = 0
			}
			ui.encounter.SetCurrentItem(idx)
			ui.renderDetailByEncounterIndex(idx)
		} else {
			ui.detailMeta.SetText("No monster in encounter.")
			ui.detailRaw.SetText("")
			ui.rawText = ""
		}
		_ = writeLastEncountersPath(ui.encountersPath)
		ui.status.SetText(fmt.Sprintf(" [black:gold] loaded[-:-] %s  %s", ui.encountersPath, helpText))
	})

	ui.pages.AddPage("encounter-load", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openCreateCharacterFromClassForm() {
	if ui.browseMode != BrowseCharacters || len(ui.filtered) == 0 {
		return
	}
	listIndex := ui.list.GetCurrentItem()
	if listIndex < 0 || listIndex >= len(ui.filtered) {
		return
	}
	classIndex := ui.filtered[listIndex]
	if classIndex < 0 || classIndex >= len(ui.classes) {
		return
	}
	cl := ui.classes[classIndex]

	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(fmt.Sprintf(" Create Character - %s ", cl.Name))
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)
	var raceDrop *tview.DropDown
	closeModal := func() {
		ui.pages.RemovePage("character-create")
		ui.charCreateVisible = false
		ui.app.SetFocus(ui.list)
	}
	var runGenerate func()
	form.SetCancelFunc(closeModal)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			closeModal()
			return nil
		}
		if isSubmitEvent(event) && runGenerate != nil && (raceDrop == nil || !raceDrop.IsOpen()) {
			formItem, button := form.GetFocusedItemIndex()
			switch resolveCreateCharacterSubmit(formItem, button, raceDrop != nil && raceDrop.IsOpen()) {
			case submitCancel:
				closeModal()
			case submitFocusRace:
				form.SetFocus(1)
			case submitGenerate:
				runGenerate()
			}
			return nil
		}
		return event
	})

	levelField := tview.NewInputField().SetLabel("Level (1-20): ").SetFieldWidth(8)
	levelField.SetText("1")
	levelField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if isSubmitKey(key) && runGenerate != nil {
			form.SetFocus(1)
		}
	})
	raceOptions := make([]string, 0, len(ui.races))
	raceIndexByOption := make([]int, 0, len(ui.races))
	for i, rc := range ui.races {
		label := rc.Name
		if strings.TrimSpace(rc.Source) != "" {
			label = fmt.Sprintf("%s [%s]", rc.Name, rc.Source)
		}
		raceOptions = append(raceOptions, label)
		raceIndexByOption = append(raceIndexByOption, i)
	}
	raceDrop = tview.NewDropDown().SetLabel("Race: ")
	if len(raceOptions) == 0 {
		raceOptions = []string{"(no races loaded)"}
		raceIndexByOption = []int{-1}
	}
	raceDrop.SetOptions(raceOptions, nil)
	raceDrop.SetCurrentOption(0)
	raceDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if isSubmitKey(key) && runGenerate != nil {
			runGenerate()
		}
	})
	raceDrop.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			closeModal()
			return nil
		}
		return event
	})

	setInput := func(f *tview.InputField) {
		f.SetLabelColor(tcell.ColorGold)
		f.SetFieldBackgroundColor(tcell.ColorWhite)
		f.SetFieldTextColor(tcell.ColorBlack)
		f.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	}
	setInput(levelField)
	raceDrop.SetLabelColor(tcell.ColorGold)
	raceDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	raceDrop.SetFieldTextColor(tcell.ColorWhite)
	raceDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)

	form.AddFormItem(levelField)
	form.AddFormItem(raceDrop)

	runGenerate = func() {
		level, err := strconv.Atoi(strings.TrimSpace(levelField.GetText()))
		if err != nil || level < 1 || level > 20 {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid level (1-20)[-:-]  %s", helpText))
			return
		}
		raceOpt, _ := raceDrop.GetCurrentOption()
		if raceOpt < 0 || raceOpt >= len(raceIndexByOption) || raceIndexByOption[raceOpt] < 0 || raceIndexByOption[raceOpt] >= len(ui.races) {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid race[-:-]  %s", helpText))
			return
		}
		rc := ui.races[raceIndexByOption[raceOpt]]
		base := rollBaseAbilityScores()
		build := CharacterBuild{
			Name: rc.Name + " " + cl.Name,
			Race: rc.Name,
			Classes: []CharacterClassLevel{
				{Name: cl.Name, Levels: level},
			},
			BaseScores: []int{base[0], base[1], base[2], base[3], base[4], base[5]},
		}
		sheet, build, err := ui.generateCharacterSheetFromBuild(build)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] character creation error[-:-] %v  %s", err, helpText))
			return
		}
		ui.detailMeta.SetText(sheet.Meta)
		ui.detailMeta.ScrollToBeginning()
		ui.rawText = sheet.Body
		ui.rawQuery = ""
		ui.renderRawWithHighlight("", -1)
		ui.detailRaw.ScrollToBeginning()
		charName := fmt.Sprintf("%s %s Lv%d", cl.Name, rc.Name, level)
		build.Name = charName
		ui.addGeneratedCharacterToEncounter(charName, sheet.Init, sheet.AC, sheet.HP, sheet.Meta, sheet.Body, &build)
		closeModal()
		ui.status.SetText(fmt.Sprintf(" [black:gold] character created[-:-] %s Lv%d (%s) + aggiunto a Encounters  %s", cl.Name, level, rc.Name, helpText))
	}
	form.AddButton("Generate", runGenerate)
	form.AddButton("Cancel", func() {
		closeModal()
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 10, 0, true).
			AddItem(nil, 0, 1, false), 70, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("character-create", modal, true, true)
	ui.charCreateVisible = true
	ui.app.SetFocus(form)
}

func (ui *UI) addGeneratedCharacterToEncounter(name string, init int, ac int, hp int, meta string, body string, build *CharacterBuild) {
	n := strings.TrimSpace(name)
	if n == "" {
		n = "Character"
	}
	if hp < 0 {
		hp = 0
	}
	entry := EncounterEntry{
		Custom:     true,
		CustomName: n,
		CustomInit: init,
		CustomMeta: strings.TrimSpace(meta),
		CustomBody: strings.TrimSpace(body),
		BaseHP:     hp,
		CurrentHP:  hp,
		Character:  cloneCharacterBuild(build),
	}
	if ac > 0 {
		entry.CustomAC = strconv.Itoa(ac)
	}
	ui.pushEncounterUndo()
	ui.encounterItems = append(ui.encounterItems, entry)
	ui.renderEncounterList()
	if len(ui.encounterItems) > 0 {
		ui.encounter.SetCurrentItem(len(ui.encounterItems) - 1)
	}
}

func (ui *UI) addSelectedMonsterToEncounter() {
	if ui.browseMode != BrowseMonsters {
		ui.status.SetText(fmt.Sprintf(" [white:red] aggiunta encounter disponibile solo da Monsters[-:-]  %s", helpText))
		return
	}
	if len(ui.filtered) == 0 {
		return
	}

	listIndex := ui.list.GetCurrentItem()
	if listIndex < 0 || listIndex >= len(ui.filtered) {
		return
	}

	monsterIndex := ui.filtered[listIndex]
	ui.pushEncounterUndo()
	ui.encounterSerial[monsterIndex]++
	ordinal := ui.encounterSerial[monsterIndex]
	baseHP, ok := extractHPAverageInt(ui.monsters[monsterIndex].Raw)
	if !ok {
		baseHP = 0
	}
	if scaled, ok := ui.scaledMonsterHP(monsterIndex); ok {
		baseHP = scaled
	}
	_, hpFormula := extractHP(ui.monsters[monsterIndex].Raw)
	ui.encounterItems = append(ui.encounterItems, EncounterEntry{
		MonsterIndex: monsterIndex,
		Ordinal:      ordinal,
		BaseHP:       baseHP,
		CurrentHP:    baseHP,
		HPFormula:    hpFormula,
		UseRolledHP:  false,
		RolledHP:     0,
		HasInitRoll:  false,
		InitRoll:     0,
	})
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(len(ui.encounterItems) - 1)

	m := ui.monsters[monsterIndex]
	ui.status.SetText(fmt.Sprintf(" [black:gold] aggiunto[-:-] %s #%d  %s", m.Name, ordinal, helpText))
}

func (ui *UI) adjustSelectedMonsterScale(delta int) {
	if delta == 0 || ui.browseMode != BrowseMonsters || len(ui.filtered) == 0 {
		return
	}
	listIndex := ui.list.GetCurrentItem()
	if listIndex < 0 || listIndex >= len(ui.filtered) {
		return
	}
	monsterIndex := ui.filtered[listIndex]
	step := max(min(ui.monsterScale[monsterIndex]+delta, 12), -12)
	if step == 0 {
		delete(ui.monsterScale, monsterIndex)
	} else {
		ui.monsterScale[monsterIndex] = step
	}
	ui.renderDetailByMonsterIndex(monsterIndex)
	if p, ok := scaleMonsterByCR(ui.monsters[monsterIndex], step); ok {
		ui.status.SetText(fmt.Sprintf(" [black:gold]monster scale[-:-] %s CR %s -> %s (%+d)  %s", ui.monsters[monsterIndex].Name, p.BaseCR, p.TargetCR, p.Step, helpText))
		return
	}
	ui.status.SetText(fmt.Sprintf(" [white:red] CR non scalabile per %s[-:-]  %s", ui.monsters[monsterIndex].Name, helpText))
}

func (ui *UI) scaledMonsterHP(monsterIndex int) (int, bool) {
	if monsterIndex < 0 || monsterIndex >= len(ui.monsters) {
		return 0, false
	}
	step := ui.monsterScale[monsterIndex]
	if step == 0 {
		return 0, false
	}
	p, ok := scaleMonsterByCR(ui.monsters[monsterIndex], step)
	if !ok {
		return 0, false
	}
	return p.TargetHP, true
}

func (ui *UI) encounterNPCLevels() []int {
	levels := make([]int, 0, len(ui.encounterItems))
	for _, it := range ui.encounterItems {
		if !it.Custom || it.Character == nil {
			continue
		}
		lv := classLevelsTotal(normalizeClassLevels(it.Character.Classes))
		if lv <= 0 {
			lv = 1
		}
		if lv > 20 {
			lv = 20
		}
		levels = append(levels, lv)
	}
	return levels
}

func (ui *UI) defaultEncounterPartyLevel(levels []int) int {
	if len(levels) == 0 {
		return 1
	}
	sum := 0
	for _, lv := range levels {
		sum += lv
	}
	avg := int(math.Round(float64(sum) / float64(len(levels))))
	if avg < 1 {
		return 1
	}
	if avg > 20 {
		return 20
	}
	return avg
}

func encounterMultiplierByCount(count int, partySize int) float64 {
	if count <= 1 {
		return 1.0
	}
	idx := 0
	switch {
	case count == 2:
		idx = 1
	case count >= 3 && count <= 6:
		idx = 2
	case count >= 7 && count <= 10:
		idx = 3
	case count >= 11 && count <= 14:
		idx = 4
	default:
		idx = 5
	}
	if partySize < 3 {
		idx++
	}
	if partySize > 5 {
		idx--
	}
	if idx < 0 {
		idx = 0
	}
	if idx > 5 {
		idx = 5
	}
	steps := []float64{1.0, 1.5, 2.0, 2.5, 3.0, 4.0}
	return steps[idx]
}

func partyMediumXPBudget(levels []int, forcedLevel int) int {
	if len(levels) == 0 {
		return 0
	}
	total := 0
	for _, baseLevel := range levels {
		lv := forcedLevel
		if lv <= 0 {
			lv = baseLevel
		}
		if lv < 1 {
			lv = 1
		}
		if lv > 20 {
			lv = 20
		}
		th, ok := encounterXPThresholdByLevel[lv]
		if !ok {
			continue
		}
		total += th.Medium
	}
	return total
}

func monsterXPFromCR(cr string) (int, bool) {
	xp, ok := encounterXPByCR[strings.TrimSpace(cr)]
	return xp, ok
}

func extractMonsterXP(raw map[string]any, cr string) (int, bool) {
	if raw != nil {
		if v, ok := anyToInt(raw["xp"]); ok && v >= 0 {
			return v, true
		}
		if m, ok := raw["xp"].(map[string]any); ok {
			if v, ok := anyToInt(m["value"]); ok && v >= 0 {
				return v, true
			}
		}
	}
	return monsterXPFromCR(cr)
}

func monsterMatchesEnvironment(m Monster, env string) bool {
	env = strings.TrimSpace(env)
	if env == "" || strings.EqualFold(env, "all") {
		return true
	}
	for _, e := range m.Environment {
		if strings.EqualFold(strings.TrimSpace(e), env) {
			return true
		}
	}
	return false
}

func (ui *UI) chooseAutoEncounterMonsters(targetXP int, count int, env string) ([]int, string, error) {
	if count <= 0 {
		return nil, "", errors.New("invalid monster count")
	}
	type candidate struct {
		index int
		delta int
	}
	buildCandidates := func(strictEnv bool) []candidate {
		out := make([]candidate, 0, len(ui.monsters))
		for i, m := range ui.monsters {
			xp, ok := monsterXPFromCR(m.CR)
			if !ok || xp <= 0 {
				continue
			}
			if strictEnv && !monsterMatchesEnvironment(m, env) {
				continue
			}
			delta := xp - targetXP
			if delta < 0 {
				delta = -delta
			}
			out = append(out, candidate{
				index: i,
				delta: delta,
			})
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].delta == out[j].delta {
				return strings.ToLower(ui.monsters[out[i].index].Name) < strings.ToLower(ui.monsters[out[j].index].Name)
			}
			return out[i].delta < out[j].delta
		})
		return out
	}

	usedEnv := strings.TrimSpace(env)
	cands := buildCandidates(!strings.EqualFold(env, "all") && strings.TrimSpace(env) != "")
	if len(cands) == 0 {
		cands = buildCandidates(false)
		usedEnv = "All"
	}
	if len(cands) == 0 {
		return nil, "", errors.New("no monster with valid CR/XP found")
	}

	limit := min(16, len(cands))
	selected := make([]int, 0, count)
	for range count {
		totalWeight := (limit * (limit + 1)) / 2
		r := rand.Intn(totalWeight)
		pick := 0
		for i := 0; i < limit; i++ {
			w := limit - i
			if r < w {
				pick = i
				break
			}
			r -= w
		}
		selected = append(selected, cands[pick].index)
	}
	return selected, usedEnv, nil
}

func (ui *UI) buildEncounterGenerationPreview(count int, power int, level int, env string) (encounterGenerationPreview, error) {
	levels := ui.encounterNPCLevels()
	if len(levels) == 0 {
		return encounterGenerationPreview{}, errors.New("no PCs in Encounters: add characters before generation")
	}
	if count < 1 || count > 50 {
		return encounterGenerationPreview{}, errors.New("invalid monster count (1-50)")
	}
	if power < -12 || power > 12 {
		return encounterGenerationPreview{}, errors.New("invalid power (-12..12)")
	}
	if level < 1 || level > 20 {
		return encounterGenerationPreview{}, errors.New("invalid level (1-20)")
	}

	budget := partyMediumXPBudget(levels, level)
	if budget <= 0 {
		return encounterGenerationPreview{}, errors.New("invalid XP budget")
	}
	multiplier := encounterMultiplierByCount(count, len(levels))
	targetXP := int(math.Round(float64(budget) / (float64(count) * multiplier)))
	if targetXP < 10 {
		targetXP = 10
	}

	ids, usedEnv, err := ui.chooseAutoEncounterMonsters(targetXP, count, env)
	if err != nil {
		return encounterGenerationPreview{}, err
	}

	return encounterGenerationPreview{
		PartySize:   len(levels),
		PartyLevel:  level,
		Environment: blankIfEmpty(usedEnv, "All"),
		Count:       count,
		Power:       power,
		BudgetXP:    budget,
		TargetXP:    targetXP,
		Multiplier:  multiplier,
		MonsterIDs:  ids,
	}, nil
}

func (ui *UI) renderEncounterGenerationPreview(p encounterGenerationPreview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[white]Party PNG:[-] %d   [white]Level:[-] %d   [white]Env:[-] %s\n", p.PartySize, p.PartyLevel, p.Environment)
	fmt.Fprintf(&b, "[white]Budget XP (Medium):[-] %d   [white]Multiplier:[-] %.1f   [white]Target XP/monster:[-] %d\n", p.BudgetXP, p.Multiplier, p.TargetXP)
	fmt.Fprintf(&b, "[white]Mostri:[-] %d   [white]Potenza:[-] %+d\n\n", p.Count, p.Power)

	type row struct {
		name string
		cr   string
		n    int
	}
	agg := map[string]*row{}
	order := make([]string, 0, len(p.MonsterIDs))
	for _, idx := range p.MonsterIDs {
		if idx < 0 || idx >= len(ui.monsters) {
			continue
		}
		m := ui.monsters[idx]
		key := m.Name + "|" + strings.TrimSpace(m.CR)
		if _, ok := agg[key]; !ok {
			agg[key] = &row{name: m.Name, cr: strings.TrimSpace(m.CR)}
			order = append(order, key)
		}
		agg[key].n++
	}
	if len(order) == 0 {
		return "[white:red]No monster generated[-:-]"
	}
	sort.Strings(order)
	for i, key := range order {
		r := agg[key]
		xp, _ := monsterXPFromCR(r.cr)
		line := fmt.Sprintf("%d. %s x%d [CR %s, XP %d]", i+1, r.name, r.n, blankIfEmpty(r.cr, "?"), xp)
		if p.Power != 0 {
			if preview, ok := scaleMonsterByCR(Monster{CR: r.cr}, p.Power); ok {
				line += fmt.Sprintf(" -> scale CR %s", preview.TargetCR)
			}
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (ui *UI) applyEncounterGenerationPreview(p encounterGenerationPreview) int {
	if len(p.MonsterIDs) == 0 {
		return 0
	}
	kept := make([]EncounterEntry, 0, len(ui.encounterItems))
	for _, it := range ui.encounterItems {
		if it.Custom {
			kept = append(kept, it)
		}
	}
	for _, idx := range p.MonsterIDs {
		if idx < 0 || idx >= len(ui.monsters) {
			continue
		}
		if p.Power == 0 {
			delete(ui.monsterScale, idx)
		} else {
			ui.monsterScale[idx] = p.Power
		}
		ui.encounterSerial[idx]++
		ordinal := ui.encounterSerial[idx]
		baseHP, ok := extractHPAverageInt(ui.monsters[idx].Raw)
		if !ok {
			baseHP = 0
		}
		if scaled, ok := ui.scaledMonsterHP(idx); ok {
			baseHP = scaled
		}
		_, hpFormula := extractHP(ui.monsters[idx].Raw)
		kept = append(kept, EncounterEntry{
			MonsterIndex: idx,
			Ordinal:      ordinal,
			BaseHP:       baseHP,
			CurrentHP:    baseHP,
			HPFormula:    hpFormula,
		})
	}

	ui.encounterItems = kept
	if len(ui.encounterItems) == 0 {
		ui.turnMode = false
		ui.turnIndex = 0
		ui.turnRound = 0
		ui.renderEncounterList()
		ui.detailMeta.SetText("No monster in encounter.")
		ui.detailRaw.SetText("")
		ui.rawText = ""
		return 0
	}
	if ui.turnMode {
		if ui.turnRound <= 0 {
			ui.turnRound = 1
		}
		ui.turnIndex = 0
	}
	ui.renderEncounterList()
	firstGenerated := max(0, len(ui.encounterItems)-len(p.MonsterIDs))
	if firstGenerated >= len(ui.encounterItems) {
		firstGenerated = len(ui.encounterItems) - 1
	}
	ui.encounter.SetCurrentItem(firstGenerated)
	ui.renderDetailByEncounterIndex(firstGenerated)
	return len(p.MonsterIDs)
}

func (ui *UI) openEncounterAutoGenerateForm() {
	levels := ui.encounterNPCLevels()
	if len(levels) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no PCs in Encounters: add characters first[-:-]  %s", helpText))
		return
	}

	defaultCount := max(1, len(levels))
	defaultLevel := ui.defaultEncounterPartyLevel(levels)
	defaultPower := 0
	envOptions := append([]string{"All"}, ui.collectMonsterEnvironmentOptions()...)

	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Generate Encounter from PNG ")
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)
	previewBox := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	previewBox.SetBorder(true)
	previewBox.SetTitle(" Preview ")
	previewBox.SetBorderColor(tcell.ColorGold)
	previewBox.SetTitleColor(tcell.ColorGold)
	previewBox.SetTextColor(tcell.ColorWhite)

	closeModal := func() {
		ui.pages.RemovePage("encounter-generate")
		ui.encounterGenVisible = false
		ui.app.SetFocus(ui.encounter)
	}
	ui.encounterGenVisible = true

	numberField := tview.NewInputField().SetLabel("Number Monsters: ").SetFieldWidth(8)
	numberField.SetText(strconv.Itoa(defaultCount))
	powerField := tview.NewInputField().SetLabel("Power (-12..12): ").SetFieldWidth(8)
	powerField.SetText(strconv.Itoa(defaultPower))
	levelField := tview.NewInputField().SetLabel("Party Level (1-20): ").SetFieldWidth(8)
	levelField.SetText(strconv.Itoa(defaultLevel))
	envDrop := tview.NewDropDown().SetLabel("Environment: ")
	envDrop.SetOptions(envOptions, nil)
	envDrop.SetCurrentOption(0)

	styleField := func(f *tview.InputField) {
		f.SetLabelColor(tcell.ColorGold)
		f.SetFieldBackgroundColor(tcell.ColorWhite)
		f.SetFieldTextColor(tcell.ColorBlack)
		f.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	}
	styleField(numberField)
	styleField(powerField)
	styleField(levelField)
	envDrop.SetLabelColor(tcell.ColorGold)
	envDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	envDrop.SetFieldTextColor(tcell.ColorWhite)
	envDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)

	form.AddFormItem(numberField)
	form.AddFormItem(powerField)
	form.AddFormItem(levelField)
	form.AddFormItem(envDrop)

	currentPreview := encounterGenerationPreview{}
	hasPreview := false
	buildPreview := func() error {
		count, err := strconv.Atoi(strings.TrimSpace(numberField.GetText()))
		if err != nil {
			return errors.New("invalid monster count")
		}
		power, err := strconv.Atoi(strings.TrimSpace(powerField.GetText()))
		if err != nil {
			return errors.New("invalid power")
		}
		level, err := strconv.Atoi(strings.TrimSpace(levelField.GetText()))
		if err != nil {
			return errors.New("invalid level")
		}
		_, env := envDrop.GetCurrentOption()
		prev, err := ui.buildEncounterGenerationPreview(count, power, level, env)
		if err != nil {
			return err
		}
		currentPreview = prev
		hasPreview = true
		previewBox.SetText(ui.renderEncounterGenerationPreview(prev))
		return nil
	}

	runPreview := func() {
		if err := buildPreview(); err != nil {
			hasPreview = false
			previewBox.SetText(fmt.Sprintf("[white:red]Input error:[-:-] %v", err))
			ui.status.SetText(fmt.Sprintf(" [white:red] %v[-:-]  %s", err, helpText))
			return
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold] encounter preview[-:-] %d monsters (lvl %d, env %s, power %+d)  %s",
			currentPreview.Count,
			currentPreview.PartyLevel,
			currentPreview.Environment,
			currentPreview.Power,
			helpText,
		))
	}
	runApply := func() {
		if !hasPreview {
			if err := buildPreview(); err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] %v[-:-]  %s", err, helpText))
				return
			}
		}
		ui.pushEncounterUndo()
		added := ui.applyEncounterGenerationPreview(currentPreview)
		closeModal()
		ui.status.SetText(fmt.Sprintf(" [black:gold] encounter generated[-:-] %d monsters added (custom/PC kept)  %s", added, helpText))
	}

	numberField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if isSubmitKey(key) {
			form.SetFocus(1)
		}
	})
	powerField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if isSubmitKey(key) {
			form.SetFocus(2)
		}
	})
	levelField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if isSubmitKey(key) {
			form.SetFocus(3)
		}
	})
	envDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if isSubmitKey(key) {
			form.SetFocus(4)
		}
	})
	envDrop.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			closeModal()
			return nil
		}
		return event
	})

	form.AddButton("Preview", runPreview)
	form.AddButton("Generate", runApply)
	form.AddButton("Cancel", closeModal)
	form.SetCancelFunc(closeModal)
	focusCount := func() int {
		return form.GetFormItemCount() + form.GetButtonCount()
	}
	currentFocusIndex := func() int {
		formItem, button := form.GetFocusedItemIndex()
		if button >= 0 {
			return form.GetFormItemCount() + button
		}
		if formItem >= 0 {
			return formItem
		}
		return 0
	}
	moveFocus := func(delta int) {
		n := focusCount()
		if n <= 0 {
			return
		}
		cur := currentFocusIndex()
		next := cur + delta
		for next < 0 {
			next += n
		}
		next = next % n
		form.SetFocus(next)
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			closeModal()
			return nil
		case tcell.KeyBacktab:
			if envDrop != nil && envDrop.IsOpen() {
				return event
			}
			moveFocus(-1)
			return nil
		case tcell.KeyTab:
			if envDrop != nil && envDrop.IsOpen() {
				return event
			}
			moveFocus(1)
			return nil
		default:
			if !isSubmitEvent(event) || envDrop.IsOpen() {
				return event
			}
			formItem, button := form.GetFocusedItemIndex()
			if button == 2 {
				closeModal()
				return nil
			}
			if button == 0 {
				runPreview()
				return nil
			}
			if button == 1 {
				runApply()
				return nil
			}
			if formItem >= 0 && formItem < 3 {
				form.SetFocus(formItem + 1)
			} else {
				form.SetFocus(4)
			}
			return nil
		}
	})

	runPreview()
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 0, 2, true).
			AddItem(previewBox, 0, 3, false).
			AddItem(nil, 0, 1, false), 124, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("encounter-generate", modal, true, true)
	ui.app.SetFocus(form)
}

func (ui *UI) collectMonsterEnvironmentOptions() []string {
	set := map[string]struct{}{}
	for _, m := range ui.monsters {
		for _, env := range m.Environment {
			env = strings.TrimSpace(env)
			if env == "" {
				continue
			}
			set[env] = struct{}{}
		}
	}
	return keysSorted(set)
}

func (ui *UI) openAddCustomEncounterForm() {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Add Custom Encounter ")
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || (event.Key() == tcell.KeyRune && event.Rune() == 'q') {
			ui.pages.RemovePage("encounter-add-custom")
			ui.addCustomVisible = false
			ui.app.SetFocus(ui.encounter)
			return nil
		}
		if event.Key() == tcell.KeyTab {
			return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		}
		return event
	})
	ui.addCustomVisible = true

	nameField := tview.NewInputField().SetLabel("Name: ").SetFieldWidth(28)
	levelField := tview.NewInputField().SetLabel("Level: ").SetFieldWidth(8)
	initField := tview.NewInputField().SetLabel("Init (x or x/x): ").SetFieldWidth(16)
	hpField := tview.NewInputField().SetLabel("HP (z or x/y): ").SetFieldWidth(16)
	acField := tview.NewInputField().SetLabel("AC (optional): ").SetFieldWidth(8)
	passiveField := tview.NewInputField().SetLabel("Passive Perception (optional): ").SetFieldWidth(8)
	nameField.SetText(randomEncounterCustomName())
	levelField.SetText("1")
	initField.SetText("0")
	hpField.SetText("5")
	acField.SetText("10")
	passiveField.SetText("0")

	setFieldStyle := func(f *tview.InputField) {
		f.SetLabelColor(tcell.ColorGold)
		f.SetFieldBackgroundColor(tcell.ColorWhite)
		f.SetFieldTextColor(tcell.ColorBlack)
		f.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	}
	setFieldStyle(nameField)
	setFieldStyle(levelField)
	setFieldStyle(initField)
	setFieldStyle(hpField)
	setFieldStyle(acField)
	setFieldStyle(passiveField)

	form.AddFormItem(nameField)
	form.AddFormItem(levelField)
	form.AddFormItem(initField)
	form.AddFormItem(hpField)
	form.AddFormItem(acField)
	form.AddFormItem(passiveField)

	form.AddButton("Save", func() {
		name := strings.TrimSpace(nameField.GetText())
		if name == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid name[-:-]  %s", helpText))
			ui.app.SetFocus(nameField)
			return
		}
		level, err := strconv.Atoi(strings.TrimSpace(levelField.GetText()))
		if err != nil || level < 1 {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid level (min 1)[-:-]  %s", helpText))
			ui.app.SetFocus(levelField)
			return
		}

		hasRoll, initRoll, initBase, ok := parseInitInput(initField.GetText())
		if !ok {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid init[-:-]  %s", helpText))
			ui.app.SetFocus(initField)
			return
		}

		currentHP, maxHP, ok := parseHPInput(hpField.GetText())
		if !ok {
			ui.status.SetText(fmt.Sprintf(" [white:red] HP non validi[-:-]  %s", helpText))
			ui.app.SetFocus(hpField)
			return
		}

		ac := strings.TrimSpace(acField.GetText())
		if ac != "" {
			if _, err := strconv.Atoi(ac); err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid AC[-:-]  %s", helpText))
				ui.app.SetFocus(acField)
				return
			}
		}
		hasPassive := false
		passive := 0
		passiveText := strings.TrimSpace(passiveField.GetText())
		if passiveText != "" {
			n, err := strconv.Atoi(passiveText)
			if err != nil || n < 0 {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid Passive Perception[-:-]  %s", helpText))
				ui.app.SetFocus(passiveField)
				return
			}
			hasPassive = true
			passive = n
		}

		ui.pushEncounterUndo()
		ordinal := ui.nextCustomOrdinal(name)
		ui.encounterItems = append(ui.encounterItems, EncounterEntry{
			MonsterIndex:     -1,
			Ordinal:          ordinal,
			Custom:           true,
			CustomName:       name,
			CustomLevel:      level,
			CustomInit:       initBase,
			CustomAC:         ac,
			CustomPassive:    passive,
			HasCustomPassive: hasPassive,
			BaseHP:           maxHP,
			CurrentHP:        currentHP,
			HasInitRoll:      hasRoll,
			InitRoll:         initRoll,
		})

		ui.pages.RemovePage("encounter-add-custom")
		ui.addCustomVisible = false
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(len(ui.encounterItems) - 1)
		ui.renderDetailByEncounterIndex(len(ui.encounterItems) - 1)
		ui.app.SetFocus(ui.encounter)
		ui.status.SetText(fmt.Sprintf(" [black:gold] aggiunta[-:-] entry custom %s  %s", name, helpText))
	})
	form.AddButton("Cancel", func() {
		ui.pages.RemovePage("encounter-add-custom")
		ui.addCustomVisible = false
		ui.app.SetFocus(ui.encounter)
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 16, 0, true).
			AddItem(nil, 0, 1, false), 74, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("encounter-add-custom", modal, true, true)
	ui.app.SetFocus(form)
}

func splitCSVValues(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return uniqueSortedStrings(out)
}

func buildDeltaLine(label string, before int, after int) string {
	delta := after - before
	if delta == 0 {
		return fmt.Sprintf("%s %d -> %d", label, before, after)
	}
	return fmt.Sprintf("%s %d -> %d (%+d)", label, before, after, delta)
}

func (ui *UI) previewCharacterBuildEdit(entry EncounterEntry, next CharacterBuild) string {
	sheet, normalized, err := ui.generateCharacterSheetFromBuild(next)
	if err != nil {
		return fmt.Sprintf("[white:red]Errore preview:[-:-] %v", err)
	}
	level := classLevelsTotal(normalized.Classes)
	lines := []string{
		fmt.Sprintf("[yellow]%s[-]", normalized.Name),
		fmt.Sprintf("Race: %s", normalized.Race),
		fmt.Sprintf("Classes: %s", classLevelsSummary(normalized.Classes)),
		fmt.Sprintf("Total Level: %d", level),
		buildDeltaLine("HP", entry.BaseHP, sheet.HP),
		buildDeltaLine("AC", atoiDefault(entry.CustomAC, 0), sheet.AC),
		buildDeltaLine("Init", entry.CustomInit, sheet.Init),
	}
	if level > 20 {
		lines = append(lines, "[white:red]Warning:[-:-] total level over 20")
	}
	return strings.Join(lines, "\n")
}

func atoiDefault(s string, def int) int {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return v
}

func (ui *UI) applyCharacterBuildToEncounter(index int, next CharacterBuild) error {
	if index < 0 || index >= len(ui.encounterItems) {
		return fmt.Errorf("invalid encounter index")
	}
	cur := &ui.encounterItems[index]
	if !cur.Custom {
		return fmt.Errorf("solo entry custom modificabili")
	}
	sheet, normalizedBuild, err := ui.generateCharacterSheetFromBuild(next)
	if err != nil {
		return err
	}

	oldMax := ui.encounterMaxHP(*cur)
	cur.CustomName = normalizedBuild.Name
	cur.Character = &normalizedBuild
	cur.CustomMeta = strings.TrimSpace(sheet.Meta)
	cur.CustomBody = strings.TrimSpace(sheet.Body)
	cur.CustomInit = sheet.Init
	cur.BaseHP = sheet.HP
	if sheet.AC > 0 {
		cur.CustomAC = strconv.Itoa(sheet.AC)
	}
	newMax := ui.encounterMaxHP(*cur)
	if oldMax > 0 && cur.CurrentHP >= oldMax {
		cur.CurrentHP = newMax
	} else if cur.CurrentHP > newMax {
		cur.CurrentHP = newMax
	}
	if cur.CurrentHP < 0 {
		cur.CurrentHP = 0
	}
	return nil
}

func (ui *UI) openEncounterCustomEntryEditForm(index int) {
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]
	if !entry.Custom {
		return
	}

	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Edit Custom Encounter ")
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)

	closeModal := func() {
		ui.pages.RemovePage("encounter-edit-custom")
		ui.encounterEditVisible = false
		ui.app.SetFocus(ui.encounter)
	}
	ui.encounterEditVisible = true

	nameField := tview.NewInputField().SetLabel("Name: ").SetFieldWidth(34)
	nameField.SetText(strings.TrimSpace(entry.CustomName))
	initField := tview.NewInputField().SetLabel("Init (x or x/x): ").SetFieldWidth(16)
	if entry.HasInitRoll {
		initField.SetText(fmt.Sprintf("%d/%d", entry.InitRoll, entry.CustomInit))
	} else {
		initField.SetText(strconv.Itoa(entry.CustomInit))
	}
	hpField := tview.NewInputField().SetLabel("HP (z or x/y): ").SetFieldWidth(16)
	maxHP := ui.encounterMaxHP(entry)
	if maxHP > 0 {
		hpField.SetText(fmt.Sprintf("%d/%d", entry.CurrentHP, maxHP))
	} else {
		hpField.SetText("0")
	}
	acField := tview.NewInputField().SetLabel("AC (optional): ").SetFieldWidth(8)
	acField.SetText(strings.TrimSpace(entry.CustomAC))
	passiveField := tview.NewInputField().SetLabel("Passive Perception (optional): ").SetFieldWidth(8)
	if entry.HasCustomPassive {
		passiveField.SetText(strconv.Itoa(max(0, entry.CustomPassive)))
	}

	setFieldStyle := func(f *tview.InputField) {
		f.SetLabelColor(tcell.ColorGold)
		f.SetFieldBackgroundColor(tcell.ColorWhite)
		f.SetFieldTextColor(tcell.ColorBlack)
		f.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	}
	setFieldStyle(nameField)
	setFieldStyle(initField)
	setFieldStyle(hpField)
	setFieldStyle(acField)
	setFieldStyle(passiveField)

	form.AddFormItem(nameField)
	form.AddFormItem(initField)
	form.AddFormItem(hpField)
	form.AddFormItem(acField)
	form.AddFormItem(passiveField)

	apply := func() {
		name := strings.TrimSpace(nameField.GetText())
		if name == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid name[-:-]  %s", helpText))
			return
		}

		hasRoll, initRoll, initBase, ok := parseInitInput(initField.GetText())
		if !ok {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid init[-:-]  %s", helpText))
			return
		}

		currentHP, baseHP, ok := parseHPInput(hpField.GetText())
		if !ok {
			ui.status.SetText(fmt.Sprintf(" [white:red] HP non validi[-:-]  %s", helpText))
			return
		}

		ac := strings.TrimSpace(acField.GetText())
		if ac != "" {
			if _, err := strconv.Atoi(ac); err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid AC[-:-]  %s", helpText))
				return
			}
		}

		hasPassive := false
		passive := 0
		passiveText := strings.TrimSpace(passiveField.GetText())
		if passiveText != "" {
			n, err := strconv.Atoi(passiveText)
			if err != nil || n < 0 {
				ui.status.SetText(fmt.Sprintf(" [white:red] invalid Passive Perception[-:-]  %s", helpText))
				return
			}
			hasPassive = true
			passive = n
		}

		ui.pushEncounterUndo()
		cur := &ui.encounterItems[index]
		cur.CustomName = name
		cur.CustomInit = initBase
		cur.HasInitRoll = hasRoll
		cur.InitRoll = initRoll
		cur.BaseHP = baseHP
		cur.CurrentHP = currentHP
		cur.CustomAC = ac
		cur.HasCustomPassive = hasPassive
		cur.CustomPassive = passive
		// Rebuild minimal meta/body for plain custom entries.
		if cur.Character == nil {
			cur.CustomMeta = ""
			cur.CustomBody = ""
		}

		ui.pages.RemovePage("encounter-edit-custom")
		ui.encounterEditVisible = false
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(index)
		ui.renderDetailByEncounterIndex(index)
		ui.app.SetFocus(ui.encounter)
		ui.status.SetText(fmt.Sprintf(" [black:gold] custom aggiornata[-:-] %s  %s", cur.CustomName, helpText))
	}

	form.AddButton("Apply", apply)
	form.AddButton("Cancel", closeModal)
	form.SetCancelFunc(closeModal)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			closeModal()
			return nil
		case tcell.KeyTab:
			return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		default:
			if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
				closeModal()
				return nil
			}
			if !isSubmitEvent(event) {
				return event
			}
			formItem, button := form.GetFocusedItemIndex()
			if button == 1 {
				closeModal()
				return nil
			}
			if button == 0 || formItem >= form.GetFormItemCount()-1 {
				apply()
			} else {
				form.SetFocus(formItem + 1)
			}
			return nil
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 14, 0, true).
			AddItem(nil, 0, 1, false), 84, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("encounter-edit-custom", modal, true, true)
	ui.app.SetFocus(form)
}

func (ui *UI) openEncounterCharacterEditForm() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]
	if !entry.Custom {
		ui.status.SetText(fmt.Sprintf(" [white:red] character edit available only for custom entry[-:-]  %s", helpText))
		return
	}
	if entry.Character == nil {
		ui.openEncounterCustomEntryEditForm(index)
		return
	}
	build := cloneCharacterBuild(entry.Character)
	if build == nil {
		ui.status.SetText(fmt.Sprintf(" [white:red] invalid character data[-:-]  %s", helpText))
		return
	}

	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Edit Encounter Character ")
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)
	preview := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	preview.SetBorder(true)
	preview.SetTitle(" Preview ")
	preview.SetBorderColor(tcell.ColorGold)
	preview.SetTitleColor(tcell.ColorGold)
	preview.SetTextColor(tcell.ColorWhite)
	var classDrop *tview.DropDown
	closeModal := func() {
		ui.pages.RemovePage("encounter-edit-character")
		ui.encounterEditVisible = false
		ui.app.SetFocus(ui.encounter)
	}
	var apply func()
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			closeModal()
			return nil
		case tcell.KeyTab:
			return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		default:
			if !isSubmitEvent(event) {
				return event
			}
			formItem, button := form.GetFocusedItemIndex()
			switch resolveEncounterEditSubmit(formItem, button, classDrop != nil && classDrop.IsOpen()) {
			case submitCancel:
				closeModal()
			case submitFocusRace:
				form.SetFocus(1)
			case submitFocusLevels:
				form.SetFocus(2)
			case submitApply:
				if apply != nil {
					apply()
				}
			}
			return nil
		}
	})

	nameField := tview.NewInputField().SetLabel("Name: ").SetFieldWidth(34)
	nameField.SetText(blankIfEmpty(build.Name, entry.CustomName))
	levelAddField := tview.NewInputField().SetLabel("Add Levels: ").SetFieldWidth(8)
	levelAddField.SetText("1")

	classOptions := make([]string, 0, len(ui.classes))
	classIdx := 0
	defaultClass := ""
	if len(build.Classes) > 0 {
		defaultClass = build.Classes[0].Name
	}
	for i, cl := range ui.classes {
		classOptions = append(classOptions, cl.Name)
		if defaultClass != "" && strings.EqualFold(strings.TrimSpace(cl.Name), strings.TrimSpace(defaultClass)) {
			classIdx = i
		}
	}
	if len(classOptions) == 0 {
		classOptions = []string{"Fighter"}
		classIdx = 0
	}
	classDrop = tview.NewDropDown().SetLabel("Class to Advance: ")
	classDrop.SetOptions(classOptions, nil)
	classDrop.SetCurrentOption(classIdx)

	nextBuild := func() (CharacterBuild, error) {
		addLevels, err := strconv.Atoi(strings.TrimSpace(levelAddField.GetText()))
		if err != nil || addLevels < 1 || addLevels > 20 {
			return CharacterBuild{}, fmt.Errorf("livelli da aggiungere non validi (1-20)")
		}
		newName := strings.TrimSpace(nameField.GetText())
		if newName == "" {
			return CharacterBuild{}, fmt.Errorf("invalid name")
		}
		_, classLabel := classDrop.GetCurrentOption()
		classLabel = strings.TrimSpace(classLabel)
		if classLabel == "" {
			return CharacterBuild{}, fmt.Errorf("invalid class")
		}
		if _, ok := ui.findClassByName(classLabel); !ok {
			return CharacterBuild{}, fmt.Errorf("selected class not found in data: %s", classLabel)
		}
		next := *cloneCharacterBuild(build)
		next.Name = newName
		next.Classes = normalizeClassLevels(next.Classes)
		found := false
		for i := range next.Classes {
			if strings.EqualFold(strings.TrimSpace(next.Classes[i].Name), classLabel) {
				next.Classes[i].Levels += addLevels
				found = true
				break
			}
		}
		if !found {
			next.Classes = append(next.Classes, CharacterClassLevel{Name: classLabel, Levels: addLevels})
		}
		next.Classes = normalizeClassLevels(next.Classes)
		return next, nil
	}

	refreshPreview := func() {
		next, err := nextBuild()
		if err != nil {
			preview.SetText(fmt.Sprintf("[white:red]Input error:[-:-] %v", err))
			return
		}
		preview.SetText(ui.previewCharacterBuildEdit(entry, next))
	}

	nameField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if isSubmitKey(key) {
			form.SetFocus(1)
		}
	})
	nameField.SetChangedFunc(func(_ string) { refreshPreview() })
	levelAddField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if isSubmitKey(key) {
			if apply != nil {
				apply()
			}
		}
	})
	levelAddField.SetChangedFunc(func(_ string) { refreshPreview() })
	classDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			closeModal()
			return
		}
		if isSubmitKey(key) {
			form.SetFocus(2)
		}
	})
	classDrop.SetSelectedFunc(func(_ string, _ int) { refreshPreview() })
	classDrop.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			closeModal()
			return nil
		}
		return event
	})

	setInput := func(f *tview.InputField) {
		f.SetLabelColor(tcell.ColorGold)
		f.SetFieldBackgroundColor(tcell.ColorWhite)
		f.SetFieldTextColor(tcell.ColorBlack)
		f.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	}
	setInput(nameField)
	setInput(levelAddField)

	setDrop := func(d *tview.DropDown) {
		d.SetLabelColor(tcell.ColorGold)
		d.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
		d.SetFieldTextColor(tcell.ColorWhite)
		d.SetListStyles(
			tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
			tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
		)
	}
	setDrop(classDrop)

	form.AddFormItem(nameField)
	form.AddFormItem(classDrop)
	form.AddFormItem(levelAddField)

	apply = func() {
		next, err := nextBuild()
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] %v[-:-]  %s", err, helpText))
			return
		}
		total := classLevelsTotal(next.Classes)
		overCap := total > 20
		ui.pushEncounterUndo()
		err = ui.applyCharacterBuildToEncounter(index, next)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] character update error[-:-] %v  %s", err, helpText))
			return
		}

		cur := &ui.encounterItems[index]
		ui.pages.RemovePage("encounter-edit-character")
		ui.encounterEditVisible = false
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(index)
		ui.renderDetailByEncounterIndex(index)
		ui.app.SetFocus(ui.encounter)
		if overCap {
			ui.status.SetText(fmt.Sprintf(" [black:gold] character updated[-:-] %s (warning: level %d > 20)  %s", cur.CustomName, total, helpText))
		} else {
			ui.status.SetText(fmt.Sprintf(" [black:gold] character updated[-:-] %s  %s", cur.CustomName, helpText))
		}
	}

	form.AddButton("Apply", apply)
	form.AddButton("Cancel", func() {
		closeModal()
	})

	refreshPreview()
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 0, 2, true).
			AddItem(preview, 0, 2, false).
			AddItem(nil, 0, 1, false), 120, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("encounter-edit-character", modal, true, true)
	ui.encounterEditVisible = true
	ui.app.SetFocus(form)
}

func (ui *UI) selectedEncounterCharacter() (*CharacterBuild, int, bool) {
	if len(ui.encounterItems) == 0 {
		return nil, -1, false
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return nil, -1, false
	}
	entry := ui.encounterItems[index]
	if !entry.Custom || entry.Character == nil {
		return nil, index, false
	}
	return cloneCharacterBuild(entry.Character), index, true
}

func (ui *UI) saveCharacterBuildAs(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	build, _, ok := ui.selectedEncounterCharacter()
	if !ok {
		return errors.New("no custom character selected")
	}
	out, err := yaml.Marshal(build)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return err
	}
	ui.buildPath = path
	_ = writeLastBuildPath(path)
	return nil
}

func (ui *UI) loadCharacterBuildFrom(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return errors.New("no encounter entry selected")
	}
	if !ui.encounterItems[index].Custom {
		return errors.New("build load available only for custom entry")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var build CharacterBuild
	if err := yaml.Unmarshal(b, &build); err != nil {
		return err
	}
	ui.pushEncounterUndo()
	if err := ui.applyCharacterBuildToEncounter(index, build); err != nil {
		return err
	}
	ui.buildPath = path
	_ = writeLastBuildPath(path)
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(index)
	ui.renderDetailByEncounterIndex(index)
	return nil
}

func (ui *UI) openCharacterBuildSaveInput() {
	if _, _, ok := ui.selectedEncounterCharacter(); !ok {
		ui.status.SetText(fmt.Sprintf(" [white:red] select a custom character in Encounters[-:-]  %s", helpText))
		return
	}
	input := tview.NewInputField().SetLabel("Build file: ").SetFieldWidth(56)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBorder(true)
	input.SetTitle(" Save Character Build As ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	if strings.TrimSpace(ui.buildPath) == "" {
		ui.buildPath = readLastBuildPath()
	}
	input.SetText(ui.buildPath)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 74, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("character-build-save")
		ui.app.SetFocus(ui.encounter)
		if key == tcell.KeyEscape || !isSubmitKey(key) {
			return
		}
		path := strings.TrimSpace(input.GetText())
		if path == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid file name[-:-]  %s", helpText))
			return
		}
		if _, err := os.Stat(path); err == nil {
			ui.status.SetText(fmt.Sprintf(" [black:gold] warning[-:-] sovrascrivo %s  %s", path, helpText))
		}
		if err := ui.saveCharacterBuildAs(path); err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] save error build[-:-] %v  %s", err, helpText))
			return
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold] build saved[-:-] %s  %s", path, helpText))
	})

	ui.pages.AddPage("character-build-save", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openCharacterBuildLoadInput() {
	input := tview.NewInputField().SetLabel("Build file: ").SetFieldWidth(56)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBorder(true)
	input.SetTitle(" Load Character Build ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	if strings.TrimSpace(ui.buildPath) == "" {
		ui.buildPath = readLastBuildPath()
	}
	input.SetText(ui.buildPath)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 74, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("character-build-load")
		ui.app.SetFocus(ui.encounter)
		if key == tcell.KeyEscape || !isSubmitKey(key) {
			return
		}
		path := strings.TrimSpace(input.GetText())
		if path == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid file name[-:-]  %s", helpText))
			return
		}
		if err := ui.loadCharacterBuildFrom(path); err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] loading error build[-:-] %v  %s", err, helpText))
			return
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold] build loaded[-:-] %s  %s", path, helpText))
	})

	ui.pages.AddPage("character-build-load", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openEncounterConditionModal() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]
	temp := cloneStringIntMap(entry.Conditions)
	if temp == nil {
		temp = map[string]int{}
	}

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Encounter Conditions (Space=toggle, Enter=apply, Esc=cancel) ")
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(false)

	render := func() {
		cur := list.GetCurrentItem()
		list.Clear()
		for _, d := range encounterConditionDefs {
			r := temp[d.Code]
			mark := "[ ]"
			if r > 0 {
				mark = fmt.Sprintf("[x%d]", r)
			}
			list.AddItem(fmt.Sprintf("%s %s (%s)", mark, d.Name, d.Code), "", 0, nil)
		}
		if cur < 0 {
			cur = 0
		}
		if cur >= list.GetItemCount() {
			cur = list.GetItemCount() - 1
		}
		if cur < 0 {
			cur = 0
		}
		list.SetCurrentItem(cur)
	}

	toggle := func() {
		idx := list.GetCurrentItem()
		if idx < 0 || idx >= len(encounterConditionDefs) {
			return
		}
		code := encounterConditionDefs[idx].Code
		if temp[code] > 0 {
			delete(temp, code)
		} else {
			temp[code] = 1
		}
		render()
	}

	closeModal := func(apply bool) {
		ui.pages.RemovePage("encounter-conditions")
		ui.app.SetFocus(ui.encounter)
		if !apply {
			return
		}
		ui.pushEncounterUndo()
		ui.encounterItems[index].Conditions = cloneStringIntMap(temp)
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(index)
		ui.renderDetailByEncounterIndex(index)
		ui.status.SetText(fmt.Sprintf(" [black:gold] conditions[-:-] updated on %s  %s", ui.encounterEntryDisplay(ui.encounterItems[index]), helpText))
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == ' ':
			toggle()
			return nil
		case event.Key() == tcell.KeyEnter:
			closeModal(true)
			return nil
		case event.Key() == tcell.KeyEscape:
			closeModal(false)
			return nil
		default:
			return event
		}
	})

	render()
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 20, 0, true).
			AddItem(nil, 0, 1, false), 74, 0, true).
		AddItem(nil, 0, 1, false)
	ui.pages.AddPage("encounter-conditions", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *UI) clearEncounterConditions() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	if len(ui.encounterItems[index].Conditions) == 0 {
		return
	}
	ui.pushEncounterUndo()
	ui.encounterItems[index].Conditions = nil
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(index)
	ui.renderDetailByEncounterIndex(index)
	ui.status.SetText(fmt.Sprintf(" [black:gold] conditions[-:-] removed from %s  %s", ui.encounterEntryDisplay(ui.encounterItems[index]), helpText))
}

func (ui *UI) removeEncounterConditionByCode(index int, code string) bool {
	if index < 0 || index >= len(ui.encounterItems) {
		return false
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" || len(ui.encounterItems[index].Conditions) == 0 {
		return false
	}
	if _, ok := ui.encounterItems[index].Conditions[code]; !ok {
		return false
	}
	delete(ui.encounterItems[index].Conditions, code)
	if len(ui.encounterItems[index].Conditions) == 0 {
		ui.encounterItems[index].Conditions = nil
	}
	return true
}

func (ui *UI) openEncounterConditionRemoveModal() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]
	if len(entry.Conditions) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no condition to remove[-:-]  %s", helpText))
		return
	}

	active := make([]encounterConditionDef, 0, len(entry.Conditions))
	for _, d := range encounterConditionDefs {
		if entry.Conditions[d.Code] > 0 {
			active = append(active, d)
		}
	}
	if len(active) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no condition to remove[-:-]  %s", helpText))
		return
	}

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Remove One Condition (Enter=remove, Esc=cancel) ")
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(false)
	for _, d := range active {
		rounds := entry.Conditions[d.Code]
		list.AddItem(fmt.Sprintf("%s (%s%d)", d.Name, d.Code, rounds), "", 0, nil)
	}

	closeModal := func() {
		ui.pages.RemovePage("encounter-condition-remove")
		ui.app.SetFocus(ui.encounter)
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			closeModal()
			return nil
		case tcell.KeyEnter:
			cur := list.GetCurrentItem()
			if cur < 0 || cur >= len(active) {
				closeModal()
				return nil
			}
			code := active[cur].Code
			ui.pushEncounterUndo()
			if ui.removeEncounterConditionByCode(index, code) {
				ui.renderEncounterList()
				ui.encounter.SetCurrentItem(index)
				ui.renderDetailByEncounterIndex(index)
				ui.status.SetText(fmt.Sprintf(" [black:gold] conditions[-:-] removed %s da %s  %s", conditionNameByCode(code), ui.encounterEntryDisplay(ui.encounterItems[index]), helpText))
			}
			closeModal()
			return nil
		default:
			return event
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 12, 0, true).
			AddItem(nil, 0, 1, false), 58, 0, true).
		AddItem(nil, 0, 1, false)
	ui.pages.AddPage("encounter-condition-remove", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *UI) adjustEncounterConditionRounds(delta int) {
	if delta == 0 || len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	if len(ui.encounterItems[index].Conditions) == 0 {
		return
	}
	ui.pushEncounterUndo()
	for code, r := range ui.encounterItems[index].Conditions {
		n := r + delta
		if n <= 0 {
			delete(ui.encounterItems[index].Conditions, code)
		} else {
			ui.encounterItems[index].Conditions[code] = n
		}
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(index)
	ui.renderDetailByEncounterIndex(index)
	if delta > 0 {
		ui.status.SetText(fmt.Sprintf(" [black:gold] conditions[-:-] round +1 on %s  %s", ui.encounterEntryDisplay(ui.encounterItems[index]), helpText))
	} else {
		ui.status.SetText(fmt.Sprintf(" [black:gold] conditions[-:-] round -1 on %s  %s", ui.encounterEntryDisplay(ui.encounterItems[index]), helpText))
	}
}

func (ui *UI) centerEncounterTurnItem() {
	_, _, _, h := ui.encounter.GetInnerRect()
	ui.encounter.SetCurrentItem(ui.turnIndex)
	offset := ui.turnIndex - h/2
	if offset < 0 {
		offset = 0
	}
	ui.encounter.SetOffset(offset, 0)
}

func (ui *UI) renderEncounterList() {
	ui.encounter.Clear()
	if len(ui.encounterItems) == 0 {
		ui.turnMode = false
		ui.encounter.AddItem("No monster in encounter", "", 0, nil)
		return
	}
	if ui.turnMode {
		if ui.turnRound <= 0 {
			ui.turnRound = 1
		}
		if ui.turnIndex < 0 {
			ui.turnIndex = 0
		}
		if ui.turnIndex >= len(ui.encounterItems) {
			ui.turnIndex = 0
		}
	}

	for i, item := range ui.encounterItems {
		label := ui.encounterEntryDisplay(item)
		if init, ok := ui.encounterInitBase(item); ok {
			if item.HasInitRoll {
				label = fmt.Sprintf("%s [Init %d/%d]", label, item.InitRoll, init)
			} else {
				label = fmt.Sprintf("%s [Init %d]", label, init)
			}
		}
		if item.Custom {
			if ac := strings.TrimSpace(item.CustomAC); ac != "" {
				label = fmt.Sprintf("%s [AC %s]", label, ac)
			}
		} else if item.MonsterIndex >= 0 && item.MonsterIndex < len(ui.monsters) {
			if ac := extractAC(ui.monsters[item.MonsterIndex].Raw); ac != "" {
				label = fmt.Sprintf("%s [AC %s]", label, ac)
			}
		}
		maxHP := ui.encounterMaxHP(item)
		if maxHP > 0 {
			if item.CurrentHP <= 0 {
				label = "X " + label
			}
			label = fmt.Sprintf("%s [HP %d/%d]", label, item.CurrentHP, maxHP)
			if item.CurrentHP > 0 && item.CurrentHP*2 < maxHP {
				label += " \U0001fa78"
			}
		} else {
			label = fmt.Sprintf("%s [HP ?]", label)
		}
		if item.TempHP > 0 {
			label = fmt.Sprintf("%s [THP %d]", label, item.TempHP)
		}
		if badge := ui.encounterConditionsBadge(item); badge != "" {
			if after, ok := strings.CutPrefix(label, "X "); ok {
				label = "X " + badge + " " + after
			} else {
				label = badge + " " + label
			}
		}
		if ui.turnMode {
			prefix := fmt.Sprintf("%d", i+1)
			if i == ui.turnIndex {
				prefix += fmt.Sprintf("*[%d]", ui.turnRound)
			}
			label = prefix + " " + label
		}
		ui.encounter.AddItem(label, "", 0, nil)
	}
}

func (ui *UI) openEncounterHPInput(direction int) {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]
	if entry.BaseHP <= 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] HP non disponibile per %s[-:-]  %s", ui.encounterEntryDisplay(entry), helpText))
		return
	}
	if direction == 0 {
		return
	}

	input := tview.NewInputField().
		SetLabel("HP ").
		SetFieldWidth(12)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	if direction < 0 {
		input.SetTitle(" Damage Encounter ")
	} else {
		input.SetTitle(" Heal Encounter ")
	}
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 40, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("encounter-damage")
		ui.app.SetFocus(ui.encounter)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}

		text := strings.TrimSpace(input.GetText())
		damage, err := strconv.Atoi(text)
		if err != nil || damage <= 0 {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid damage value[-:-] \"%s\"  %s", text, helpText))
			return
		}

		ui.pushEncounterUndo()
		if direction < 0 {
			tempSpent, hpSpent := ui.applyEncounterDamage(index, damage)
			_ = hpSpent
			_ = tempSpent
		} else {
			ui.applyEncounterHealing(index, damage)
		}
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(index)
		ui.renderDetailByEncounterIndex(index)

		if direction < 0 {
			ui.status.SetText(fmt.Sprintf(" [black:gold] damage[-:-] %s -%d HP (%d/%d, THP %d)  %s",
				ui.encounterEntryDisplay(ui.encounterItems[index]),
				damage,
				ui.encounterItems[index].CurrentHP,
				ui.encounterMaxHP(ui.encounterItems[index]),
				ui.encounterItems[index].TempHP,
				helpText,
			))
		} else {
			ui.status.SetText(fmt.Sprintf(" [black:gold] heal[-:-] %s +%d HP (%d/%d, THP %d)  %s",
				ui.encounterEntryDisplay(ui.encounterItems[index]),
				damage,
				ui.encounterItems[index].CurrentHP,
				ui.encounterMaxHP(ui.encounterItems[index]),
				ui.encounterItems[index].TempHP,
				helpText,
			))
		}
	})

	ui.pages.AddPage("encounter-damage", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) applyEncounterDamage(index int, damage int) (tempSpent int, hpSpent int) {
	if index < 0 || index >= len(ui.encounterItems) || damage <= 0 {
		return 0, 0
	}
	entry := &ui.encounterItems[index]
	remaining := damage
	if entry.TempHP > 0 {
		tempSpent = min(entry.TempHP, remaining)
		entry.TempHP -= tempSpent
		if entry.TempHP < 0 {
			entry.TempHP = 0
		}
		remaining -= tempSpent
	}
	if remaining > 0 {
		before := entry.CurrentHP
		entry.CurrentHP -= remaining
		if entry.CurrentHP < 0 {
			entry.CurrentHP = 0
		}
		hpSpent = max(0, before-entry.CurrentHP)
	}
	return tempSpent, hpSpent
}

func (ui *UI) applyEncounterHealing(index int, healing int) {
	if index < 0 || index >= len(ui.encounterItems) || healing <= 0 {
		return
	}
	ui.encounterItems[index].CurrentHP += healing
	maxHP := ui.encounterMaxHP(ui.encounterItems[index])
	if maxHP > 0 && ui.encounterItems[index].CurrentHP > maxHP {
		ui.encounterItems[index].CurrentHP = maxHP
	}
}

func (ui *UI) openEncounterTempHPInput() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]

	input := tview.NewInputField().
		SetLabel("Temp HP ").
		SetFieldWidth(12)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Set Temp HP (x or -x) ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 44, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("encounter-thp")
		ui.app.SetFocus(ui.encounter)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}

		text := strings.TrimSpace(input.GetText())
		val, err := strconv.Atoi(text)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid temp HP value[-:-] \"%s\"  %s", text, helpText))
			return
		}

		ui.pushEncounterUndo()
		next := ui.applyEncounterTempHPValue(index, val)
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(index)
		ui.renderDetailByEncounterIndex(index)
		ui.status.SetText(fmt.Sprintf(" [black:gold] temp hp[-:-] %s -> %d  %s",
			ui.encounterEntryDisplay(entry),
			next,
			helpText,
		))
	})

	ui.pages.AddPage("encounter-thp", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) applyEncounterTempHPValue(index int, val int) int {
	if index < 0 || index >= len(ui.encounterItems) {
		return 0
	}
	cur := ui.encounterItems[index].TempHP
	next := cur
	if val > 0 {
		next = max(cur, val)
	} else if val < 0 {
		next = max(0, cur+val)
	} else {
		next = 0
	}
	ui.encounterItems[index].TempHP = next
	return next
}

func (ui *UI) clearEncounterTempHP() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	if ui.encounterItems[index].TempHP <= 0 {
		return
	}
	ui.pushEncounterUndo()
	ui.encounterItems[index].TempHP = 0
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(index)
	ui.renderDetailByEncounterIndex(index)
	ui.status.SetText(fmt.Sprintf(" [black:gold] temp hp[-:-] cleared for %s  %s",
		ui.encounterEntryDisplay(ui.encounterItems[index]),
		helpText,
	))
}

func (ui *UI) deleteSelectedEncounterEntry() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]

	ui.pushEncounterUndo()
	ui.encounterItems = append(ui.encounterItems[:index], ui.encounterItems[index+1:]...)
	if ui.turnMode {
		if len(ui.encounterItems) == 0 {
			ui.turnMode = false
			ui.turnIndex = 0
			ui.turnRound = 0
		} else {
			if index < ui.turnIndex {
				ui.turnIndex--
			}
			if ui.turnIndex >= len(ui.encounterItems) {
				ui.turnIndex = 0
			}
			if ui.turnRound <= 0 {
				ui.turnRound = 1
			}
		}
	}
	ui.renderEncounterList()
	if len(ui.encounterItems) > 0 {
		if index >= len(ui.encounterItems) {
			index = len(ui.encounterItems) - 1
		}
		ui.encounter.SetCurrentItem(index)
		ui.renderDetailByEncounterIndex(index)
	} else {
		ui.detailMeta.SetText("No monster in encounter.")
		ui.detailRaw.SetText("")
		ui.rawText = ""
	}
	ui.status.SetText(fmt.Sprintf(" [black:gold] eliminato[-:-] %s  %s", ui.encounterEntryDisplay(entry), helpText))
}

func (ui *UI) deleteAllMonsterEncounterEntries() {
	if len(ui.encounterItems) == 0 {
		return
	}
	kept := make([]EncounterEntry, 0, len(ui.encounterItems))
	oldToNew := map[int]int{}
	for i, it := range ui.encounterItems {
		if it.Custom {
			oldToNew[i] = len(kept)
			kept = append(kept, it)
		}
	}
	removed := len(ui.encounterItems) - len(kept)
	if removed <= 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no monster to remove (custom/characters only)[-:-]  %s", helpText))
		return
	}

	ui.pushEncounterUndo()
	selectedOld := ui.encounter.GetCurrentItem()
	turnOld := ui.turnIndex
	ui.encounterItems = kept

	if len(ui.encounterItems) == 0 {
		ui.turnMode = false
		ui.turnIndex = 0
		ui.turnRound = 0
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(0)
		ui.detailMeta.SetText("No monster in encounter.")
		ui.detailRaw.SetText("")
		ui.rawText = ""
		ui.status.SetText(fmt.Sprintf(" [black:gold] removed[-:-] %d monsters (no entry left)  %s", removed, helpText))
		return
	}

	newSelected := 0
	if idx, ok := oldToNew[selectedOld]; ok {
		newSelected = idx
	} else if selectedOld >= len(ui.encounterItems) {
		newSelected = len(ui.encounterItems) - 1
	}
	if newSelected < 0 {
		newSelected = 0
	}
	if newSelected >= len(ui.encounterItems) {
		newSelected = len(ui.encounterItems) - 1
	}

	if ui.turnMode {
		if idx, ok := oldToNew[turnOld]; ok {
			ui.turnIndex = idx
		} else {
			ui.turnIndex = 0
		}
		if ui.turnIndex < 0 || ui.turnIndex >= len(ui.encounterItems) {
			ui.turnIndex = 0
		}
	}

	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(newSelected)
	ui.renderDetailByEncounterIndex(newSelected)
	ui.status.SetText(fmt.Sprintf(" [black:gold] removed[-:-] %d monsters (custom/characters kept)  %s", removed, helpText))
}

func (ui *UI) toggleEncounterHPMode() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}

	entry := ui.encounterItems[index]
	if strings.TrimSpace(entry.HPFormula) == "" {
		ui.status.SetText(fmt.Sprintf(" [white:red] formula HP non disponibile per %s[-:-]  %s", ui.encounterEntryDisplay(entry), helpText))
		return
	}

	if entry.UseRolledHP {
		ui.pushEncounterUndo()
		ui.encounterItems[index].UseRolledHP = false
		maxHP := ui.encounterMaxHP(ui.encounterItems[index])
		if maxHP > 0 && ui.encounterItems[index].CurrentHP > maxHP {
			ui.encounterItems[index].CurrentHP = maxHP
		}
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(index)
		ui.status.SetText(fmt.Sprintf(" [black:gold] hp mode[-:-] %s -> average  %s", ui.encounterEntryDisplay(entry), helpText))
		return
	}

	rolled, ok := rollHPFormula(entry.HPFormula)
	if !ok {
		ui.status.SetText(fmt.Sprintf(" [white:red] formula HP non supportata[-:-] \"%s\"  %s", entry.HPFormula, helpText))
		return
	}
	ui.pushEncounterUndo()
	ui.encounterItems[index].UseRolledHP = true
	ui.encounterItems[index].RolledHP = rolled
	maxHP := ui.encounterMaxHP(ui.encounterItems[index])
	if maxHP > 0 && ui.encounterItems[index].CurrentHP > maxHP {
		ui.encounterItems[index].CurrentHP = maxHP
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(index)
	ui.status.SetText(fmt.Sprintf(" [black:gold] hp mode[-:-] %s -> formula (%s = %d)  %s", ui.encounterEntryDisplay(entry), entry.HPFormula, rolled, helpText))
}

func (ui *UI) rollEncounterInitiative() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}

	entry := ui.encounterItems[index]
	initValue, ok := ui.encounterInitBase(entry)
	if !ok {
		ui.status.SetText(fmt.Sprintf(" [white:red] init non disponibile per %s[-:-]  %s", ui.encounterEntryDisplay(entry), helpText))
		return
	}

	ui.pushEncounterUndo()
	roll := (rand.Intn(20) + 1) + initValue
	ui.encounterItems[index].HasInitRoll = true
	ui.encounterItems[index].InitRoll = roll
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(index)
	ui.renderDetailByEncounterIndex(index)
	ui.status.SetText(fmt.Sprintf(" [black:gold] initiative[-:-] %s = %d/%d  %s", ui.encounterEntryDisplay(entry), roll, initValue, helpText))
}

func (ui *UI) rollAllEncounterInitiative() {
	if len(ui.encounterItems) == 0 {
		return
	}

	ui.pushEncounterUndo()
	rolledCount := 0
	for i := range ui.encounterItems {
		entry := ui.encounterItems[i]
		initValue, ok := ui.encounterInitBase(entry)
		if !ok {
			continue
		}
		ui.encounterItems[i].HasInitRoll = true
		ui.encounterItems[i].InitRoll = (rand.Intn(20) + 1) + initValue
		rolledCount++
	}

	ui.renderEncounterList()
	if len(ui.encounterItems) > 0 {
		idx := max(ui.encounter.GetCurrentItem(), 0)
		if idx >= len(ui.encounterItems) {
			idx = len(ui.encounterItems) - 1
		}
		ui.encounter.SetCurrentItem(idx)
		ui.renderDetailByEncounterIndex(idx)
	}

	if rolledCount == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no entry with available dex[-:-]  %s", helpText))
		return
	}
	ui.status.SetText(fmt.Sprintf(" [black:gold] initiative[-:-] tirata per %d entry  %s", rolledCount, helpText))
}

func (ui *UI) sortEncounterByInitiative() {
	if len(ui.encounterItems) < 2 {
		return
	}

	current := ui.encounter.GetCurrentItem()
	if current < 0 || current >= len(ui.encounterItems) {
		current = 0
	}
	selected := ui.encounterItems[current]
	active := EncounterEntry{}
	hasActive := false
	if ui.turnMode && ui.turnIndex >= 0 && ui.turnIndex < len(ui.encounterItems) {
		active = ui.encounterItems[ui.turnIndex]
		hasActive = true
	}

	ui.pushEncounterUndo()

	sort.SliceStable(ui.encounterItems, func(i, j int) bool {
		a := ui.encounterItems[i]
		b := ui.encounterItems[j]

		if a.HasInitRoll != b.HasInitRoll {
			return a.HasInitRoll
		}
		if a.HasInitRoll && b.HasInitRoll && a.InitRoll != b.InitRoll {
			return a.InitRoll > b.InitRoll
		}

		aInit, aok := ui.encounterInitBase(a)
		bInit, bok := ui.encounterInitBase(b)
		if aok != bok {
			return aok
		}
		if aok && bok && aInit != bInit {
			return aInit > bInit
		}

		an := ui.encounterEntryName(a)
		bn := ui.encounterEntryName(b)
		if !strings.EqualFold(an, bn) {
			return strings.ToLower(an) < strings.ToLower(bn)
		}
		return a.Ordinal < b.Ordinal
	})

	ui.renderEncounterList()

	newIndex := 0
	newTurnIndex := -1
	for i, it := range ui.encounterItems {
		if it.MonsterIndex == selected.MonsterIndex && it.Ordinal == selected.Ordinal {
			newIndex = i
		}
		if hasActive && it.MonsterIndex == active.MonsterIndex && it.Ordinal == active.Ordinal {
			newTurnIndex = i
		}
	}
	if ui.turnMode && hasActive && newTurnIndex >= 0 {
		ui.turnIndex = newTurnIndex
	}
	ui.encounter.SetCurrentItem(newIndex)
	ui.renderDetailByEncounterIndex(newIndex)
	ui.status.SetText(fmt.Sprintf(" [black:gold] sort[-:-] encounters ordinati per iniziativa  %s", helpText))
}

func (ui *UI) pushEncounterUndo() {
	snap := EncounterUndoState{
		Items:    cloneEncounterEntries(ui.encounterItems),
		Serial:   cloneIntMap(ui.encounterSerial),
		Selected: ui.encounter.GetCurrentItem(),
	}
	ui.encounterUndo = append(ui.encounterUndo, snap)
	ui.encounterRedo = ui.encounterRedo[:0]
}

func (ui *UI) undoEncounterCommand() {
	if len(ui.encounterUndo) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no operation to undo[-:-]  %s", helpText))
		return
	}
	current := ui.captureEncounterState()
	last := ui.encounterUndo[len(ui.encounterUndo)-1]
	ui.encounterUndo = ui.encounterUndo[:len(ui.encounterUndo)-1]
	ui.encounterRedo = append(ui.encounterRedo, current)
	ui.restoreEncounterState(last)
	ui.status.SetText(fmt.Sprintf(" [black:gold] undo[-:-] encounter operation undone  %s", helpText))
}

func (ui *UI) redoEncounterCommand() {
	if len(ui.encounterRedo) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no operation to redo[-:-]  %s", helpText))
		return
	}
	current := ui.captureEncounterState()
	last := ui.encounterRedo[len(ui.encounterRedo)-1]
	ui.encounterRedo = ui.encounterRedo[:len(ui.encounterRedo)-1]
	ui.encounterUndo = append(ui.encounterUndo, current)
	ui.restoreEncounterState(last)
	ui.status.SetText(fmt.Sprintf(" [black:gold] redo[-:-] encounter operation redone  %s", helpText))
}

func (ui *UI) captureEncounterState() EncounterUndoState {
	return EncounterUndoState{
		Items:    cloneEncounterEntries(ui.encounterItems),
		Serial:   cloneIntMap(ui.encounterSerial),
		Selected: ui.encounter.GetCurrentItem(),
	}
}

func (ui *UI) restoreEncounterState(state EncounterUndoState) {
	ui.encounterItems = cloneEncounterEntries(state.Items)
	ui.encounterSerial = cloneIntMap(state.Serial)
	ui.renderEncounterList()

	if len(ui.encounterItems) > 0 {
		idx := max(state.Selected, 0)
		if idx >= len(ui.encounterItems) {
			idx = len(ui.encounterItems) - 1
		}
		ui.encounter.SetCurrentItem(idx)
		ui.renderDetailByEncounterIndex(idx)
		return
	}

	ui.detailMeta.SetText("No monster in encounter.")
	ui.detailRaw.SetText("")
	ui.rawText = ""
}

func (ui *UI) toggleEncounterTurnMode() {
	if len(ui.encounterItems) == 0 {
		return
	}
	if ui.turnMode {
		ui.turnMode = false
		ui.turnRound = 0
		ui.renderEncounterList()
		idx := max(ui.encounter.GetCurrentItem(), 0)
		ui.encounter.SetCurrentItem(idx)
		ui.renderDetailByEncounterIndex(idx)
		ui.status.SetText(fmt.Sprintf(" [black:gold] turn mode[-:-] disabled  %s", helpText))
		return
	}
	idx := ui.findTopInitiativeEncounterIndex()
	ui.turnMode = true
	ui.turnIndex = idx
	ui.turnRound = 1
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(idx)
	ui.renderDetailByEncounterIndex(idx)
	ui.status.SetText(fmt.Sprintf(" [black:gold] turn mode[-:-] attivo (round 1)  %s", helpText))
}

func (ui *UI) findTopInitiativeEncounterIndex() int {
	if len(ui.encounterItems) == 0 {
		return 0
	}
	best := 0
	bestVal := -1 << 30
	bestHas := false
	for i, e := range ui.encounterItems {
		v, ok := ui.encounterInitBase(e)
		if e.HasInitRoll {
			v = e.InitRoll
			ok = true
		}
		if !ok {
			continue
		}
		if !bestHas || v > bestVal {
			bestHas = true
			bestVal = v
			best = i
		}
	}
	if bestHas {
		return best
	}
	return 0
}

func (ui *UI) nextEncounterTurn() {
	if !ui.turnMode || len(ui.encounterItems) == 0 {
		return
	}
	if ui.turnIndex >= len(ui.encounterItems)-1 {
		ui.turnIndex = 0
		ui.turnRound++
		if ui.turnRound <= 0 {
			ui.turnRound = 1
		}
		ui.bumpAllEncounterConditionRounds(1)
	} else {
		ui.turnIndex++
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(ui.turnIndex)
	ui.renderDetailByEncounterIndex(ui.turnIndex)
	ui.status.SetText(fmt.Sprintf(" [black:gold] turn[-:-] round %d, entry %d  %s", ui.turnRound, ui.turnIndex+1, helpText))
}

func (ui *UI) yankEncounterEntry() {
	idx := ui.encounter.GetCurrentItem()
	if idx < 0 || idx >= len(ui.encounterItems) {
		return
	}
	e := ui.encounterItems[idx]
	ui.encounterYank = &e
	name := ui.encounterEntryDisplay(e)
	ui.status.SetText(fmt.Sprintf(" [black:gold] copiato[-:-] %s  %s", name, helpText))
}

func (ui *UI) pasteEncounterEntry() {
	if ui.encounterYank == nil {
		return
	}
	ui.pushEncounterUndo()
	src := *ui.encounterYank
	var newEntry EncounterEntry
	if src.Custom {
		ordinal := ui.nextCustomOrdinal(src.CustomName)
		newEntry = EncounterEntry{
			MonsterIndex:     -1,
			Ordinal:          ordinal,
			Custom:           true,
			CustomName:       src.CustomName,
			CustomLevel:      src.CustomLevel,
			CustomInit:       src.CustomInit,
			CustomAC:         src.CustomAC,
			CustomPassive:    src.CustomPassive,
			HasCustomPassive: src.HasCustomPassive,
			CustomMeta:       src.CustomMeta,
			CustomBody:       src.CustomBody,
			BaseHP:           src.BaseHP,
			CurrentHP:        src.BaseHP,
		}
	} else {
		ui.encounterSerial[src.MonsterIndex]++
		ordinal := ui.encounterSerial[src.MonsterIndex]
		newEntry = EncounterEntry{
			MonsterIndex: src.MonsterIndex,
			Ordinal:      ordinal,
			BaseHP:       src.BaseHP,
			CurrentHP:    src.BaseHP,
			HPFormula:    src.HPFormula,
		}
	}
	idx := ui.encounter.GetCurrentItem()
	insertAt := idx + 1
	if insertAt > len(ui.encounterItems) {
		insertAt = len(ui.encounterItems)
	}
	ui.encounterItems = append(ui.encounterItems, EncounterEntry{})
	copy(ui.encounterItems[insertAt+1:], ui.encounterItems[insertAt:])
	ui.encounterItems[insertAt] = newEntry
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(insertAt)
	ui.renderDetailByEncounterIndex(insertAt)
	name := ui.encounterEntryDisplay(newEntry)
	ui.status.SetText(fmt.Sprintf(" [black:gold] incollato[-:-] %s  %s", name, helpText))
}

func (ui *UI) bumpAllEncounterConditionRounds(delta int) {
	if delta == 0 {
		return
	}
	for i := range ui.encounterItems {
		if len(ui.encounterItems[i].Conditions) == 0 {
			continue
		}
		for code, r := range ui.encounterItems[i].Conditions {
			n := r + delta
			if n <= 0 {
				n = 1
			}
			ui.encounterItems[i].Conditions[code] = n
		}
	}
}

func (ui *UI) loadEncounters() error {
	b, err := os.ReadFile(ui.encountersPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var data PersistedEncounters
	if err := yaml.Unmarshal(b, &data); err != nil {
		return err
	}

	idToIndex := make(map[int]int, len(ui.monsters))
	for i, m := range ui.monsters {
		idToIndex[m.ID] = i
	}

	ui.encounterItems = ui.encounterItems[:0]
	ui.encounterSerial = map[int]int{}
	ui.turnMode = false
	ui.turnIndex = 0
	ui.turnRound = 0

	for _, it := range data.Items {
		monsterIndex := -1
		if !it.Custom {
			var ok bool
			monsterIndex, ok = idToIndex[it.MonsterID]
			if !ok {
				continue
			}
		}

		ordinal := it.Ordinal
		if ordinal <= 0 {
			if it.Custom {
				maxOrd := 0
				for _, e := range ui.encounterItems {
					if e.Custom && strings.EqualFold(strings.TrimSpace(e.CustomName), strings.TrimSpace(it.CustomName)) && e.Ordinal > maxOrd {
						maxOrd = e.Ordinal
					}
				}
				ordinal = maxOrd + 1
			} else {
				ordinal = ui.encounterSerial[monsterIndex] + 1
			}
		}

		baseHP := it.BaseHP
		hpFormula := strings.TrimSpace(it.HPFormula)
		if !it.Custom {
			if baseHP <= 0 {
				if avg, ok := extractHPAverageInt(ui.monsters[monsterIndex].Raw); ok {
					baseHP = avg
				}
			}
			if hpFormula == "" {
				_, hpFormula = extractHP(ui.monsters[monsterIndex].Raw)
			}
		}

		currentHP := it.CurrentHP
		maxHP := baseHP
		if it.UseRolled && it.RolledHP > 0 {
			maxHP = it.RolledHP
		}
		if currentHP < 0 {
			currentHP = 0
		}
		if maxHP > 0 && currentHP > maxHP {
			currentHP = maxHP
		}

		entry := EncounterEntry{
			MonsterIndex:     monsterIndex,
			Ordinal:          ordinal,
			Custom:           it.Custom,
			CustomName:       it.CustomName,
			CustomLevel:      it.CustomLevel,
			CustomInit:       it.CustomInit,
			CustomAC:         it.CustomAC,
			CustomPassive:    it.CustomPassive,
			HasCustomPassive: it.HasCustomPassive,
			CustomMeta:       it.CustomMeta,
			CustomBody:       it.CustomBody,
			Conditions:       cloneStringIntMap(it.Conditions),
			BaseHP:           baseHP,
			CurrentHP:        currentHP,
			TempHP:           max(0, it.TempHP),
			HPFormula:        hpFormula,
			UseRolledHP:      it.UseRolled,
			RolledHP:         it.RolledHP,
			HasInitRoll:      it.InitRolled,
			InitRoll:         it.InitRoll,
			Character:        cloneCharacterBuild(it.Character),
		}
		ui.backfillCustomEncounterDetails(&entry)
		ui.encounterItems = append(ui.encounterItems, entry)
		if !it.Custom && ordinal > ui.encounterSerial[monsterIndex] {
			ui.encounterSerial[monsterIndex] = ordinal
		}
	}
	if len(ui.encounterItems) > 0 {
		ui.turnMode = data.TurnMode
		ui.turnIndex = data.TurnIndex
		ui.turnRound = data.TurnRound
		if ui.turnRound <= 0 {
			ui.turnRound = 1
		}
		if ui.turnIndex < 0 || ui.turnIndex >= len(ui.encounterItems) {
			ui.turnIndex = 0
		}
	}
	return nil
}

func (ui *UI) backfillCustomEncounterDetails(entry *EncounterEntry) {
	if entry == nil || !entry.Custom {
		return
	}
	if strings.TrimSpace(entry.CustomMeta) == "" {
		b := &strings.Builder{}
		fmt.Fprintf(b, "[yellow]%s[-]\n", ui.encounterEntryDisplay(*entry))
		if init, ok := ui.encounterInitBase(*entry); ok {
			if entry.HasInitRoll {
				fmt.Fprintf(b, "[white]Init:[-] %d/%d\n", entry.InitRoll, init)
			} else {
				fmt.Fprintf(b, "[white]Init:[-] %d\n", init)
			}
		}
		if strings.TrimSpace(entry.CustomAC) != "" {
			fmt.Fprintf(b, "[white]AC:[-] %s\n", entry.CustomAC)
		}
		if entry.CustomLevel > 0 {
			fmt.Fprintf(b, "[white]Level:[-] %d\n", entry.CustomLevel)
		}
		if entry.HasCustomPassive {
			fmt.Fprintf(b, "[white]Passive Perception:[-] %d\n", max(0, entry.CustomPassive))
		}
		maxHP := ui.encounterMaxHP(*entry)
		if maxHP > 0 {
			fmt.Fprintf(b, "[white]HP:[-] %d/%d\n", entry.CurrentHP, maxHP)
		} else {
			fmt.Fprintf(b, "[white]HP:[-] ?\n")
		}
		if entry.TempHP > 0 {
			fmt.Fprintf(b, "[white]Temp HP:[-] %d\n", entry.TempHP)
		}
		entry.CustomMeta = strings.TrimSpace(b.String())
	}
	if strings.TrimSpace(entry.CustomBody) == "" {
		entry.CustomBody = buildCustomDescriptionText(*entry, ui.encounterMaxHP(*entry))
	}
	if entry.Character == nil {
		if inferred, ok := ui.inferCharacterBuildFromEntry(*entry); ok {
			entry.Character = inferred
		}
	}
}

var levelLineRe = regexp.MustCompile(`(?i)^(.+)\s+\(level\s+(\d+)\)\s*$`)

func (ui *UI) inferCharacterBuildFromEntry(entry EncounterEntry) (*CharacterBuild, bool) {
	if !entry.Custom {
		return nil, false
	}
	lines := strings.Split(strings.TrimSpace(entry.CustomBody), "\n")
	if len(lines) == 0 {
		return nil, false
	}
	m := levelLineRe.FindStringSubmatch(strings.TrimSpace(lines[0]))
	if len(m) != 3 {
		return nil, false
	}
	combined := strings.TrimSpace(m[1])
	lv, err := strconv.Atoi(strings.TrimSpace(m[2]))
	if err != nil || lv <= 0 {
		return nil, false
	}
	className := ""
	raceName := ""
	for _, cl := range ui.classes {
		if strings.HasPrefix(strings.ToLower(combined), strings.ToLower(cl.Name+" ")) || strings.EqualFold(strings.TrimSpace(combined), strings.TrimSpace(cl.Name)) {
			className = cl.Name
			raceName = strings.TrimSpace(strings.TrimPrefix(combined, cl.Name))
			break
		}
	}
	if className == "" {
		return nil, false
	}
	if raceName == "" {
		for _, rc := range ui.races {
			if strings.Contains(strings.ToLower(combined), strings.ToLower(rc.Name)) {
				raceName = rc.Name
				break
			}
		}
	}
	if raceName == "" && len(ui.races) > 0 {
		raceName = ui.races[0].Name
	}
	build := &CharacterBuild{
		Name:       blankIfEmpty(strings.TrimSpace(entry.CustomName), combined),
		Race:       raceName,
		Classes:    []CharacterClassLevel{{Name: className, Levels: lv}},
		BaseScores: []int{10, 10, 10, 10, 10, 10},
	}
	return build, true
}

func (ui *UI) saveEncounters() error {
	data := PersistedEncounters{
		Version:   1,
		Items:     make([]PersistedEncounterItem, 0, len(ui.encounterItems)),
		TurnMode:  ui.turnMode,
		TurnIndex: ui.turnIndex,
		TurnRound: ui.turnRound,
	}

	for _, it := range ui.encounterItems {
		item := PersistedEncounterItem{
			Ordinal:          it.Ordinal,
			Custom:           it.Custom,
			CustomName:       it.CustomName,
			CustomLevel:      it.CustomLevel,
			CustomInit:       it.CustomInit,
			CustomAC:         it.CustomAC,
			CustomPassive:    it.CustomPassive,
			HasCustomPassive: it.HasCustomPassive,
			CustomMeta:       it.CustomMeta,
			CustomBody:       it.CustomBody,
			Conditions:       cloneStringIntMap(it.Conditions),
			BaseHP:           it.BaseHP,
			CurrentHP:        it.CurrentHP,
			TempHP:           max(0, it.TempHP),
			HPFormula:        it.HPFormula,
			UseRolled:        it.UseRolledHP,
			RolledHP:         it.RolledHP,
			InitRolled:       it.HasInitRoll,
			InitRoll:         it.InitRoll,
			Character:        cloneCharacterBuild(it.Character),
		}
		if !it.Custom {
			if it.MonsterIndex < 0 || it.MonsterIndex >= len(ui.monsters) {
				continue
			}
			item.MonsterID = ui.monsters[it.MonsterIndex].ID
		}
		data.Items = append(data.Items, item)
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(ui.encountersPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(ui.encountersPath, out, 0o644); err != nil {
		return err
	}
	_ = writeLastEncountersPath(ui.encountersPath)
	return nil
}

func (ui *UI) saveEncountersAs(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	prev := ui.encountersPath
	ui.encountersPath = path
	if err := ui.saveEncounters(); err != nil {
		ui.encountersPath = prev
		return err
	}
	return nil
}

func (ui *UI) loadDiceResults() error {
	b, err := os.ReadFile(ui.dicePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			ui.diceLog = nil
			ui.renderDiceList()
			return nil
		}
		return err
	}
	var data PersistedDice
	if err := yaml.Unmarshal(b, &data); err != nil {
		// Backward compatibility: old format had items as []string.
		var legacy struct {
			Version int      `yaml:"version"`
			Items   []string `yaml:"items"`
		}
		if legacyErr := yaml.Unmarshal(b, &legacy); legacyErr != nil {
			return err
		}
		data.Version = legacy.Version
		data.Items = make([]DiceResult, 0, len(legacy.Items))
		for _, it := range legacy.Items {
			text := strings.TrimSpace(it)
			if text == "" {
				continue
			}
			expr := text
			out := ""
			if parts := strings.SplitN(text, "=>", 2); len(parts) == 2 {
				expr = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(parts[0], "[black:gold]", ""), "[-:-]", ""))
				out = strings.TrimSpace(parts[1])
			}
			data.Items = append(data.Items, DiceResult{Expression: expr, Output: out})
		}
	}
	ui.diceLog = append([]DiceResult(nil), data.Items...)
	ui.renderDiceList()
	_ = writeLastDicePath(ui.dicePath)
	return nil
}

func (ui *UI) saveDiceResults() error {
	data := PersistedDice{
		Version: 1,
		Items:   append([]DiceResult(nil), ui.diceLog...),
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(ui.dicePath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(ui.dicePath, out, 0o644); err != nil {
		return err
	}
	_ = writeLastDicePath(ui.dicePath)
	return nil
}

func (ui *UI) saveDiceResultsAs(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	prev := ui.dicePath
	ui.dicePath = path
	if err := ui.saveDiceResults(); err != nil {
		ui.dicePath = prev
		return err
	}
	return nil
}

func modeToKey(mode BrowseMode) string {
	switch mode {
	case BrowseItems:
		return "items"
	case BrowseSpells:
		return "spells"
	case BrowseCharacters:
		return "characters"
	case BrowseRaces:
		return "races"
	case BrowseFeats:
		return "feats"
	case BrowseBooks:
		return "books"
	case BrowseAdventures:
		return "adventures"
	case BrowseRandom:
		return "random"
	default:
		return "monsters"
	}
}

func modeFromKey(s string) BrowseMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "items":
		return BrowseItems
	case "spells":
		return BrowseSpells
	case "characters":
		return BrowseCharacters
	case "races":
		return BrowseRaces
	case "feats":
		return BrowseFeats
	case "books":
		return BrowseBooks
	case "adventures":
		return BrowseAdventures
	case "random":
		return BrowseRandom
	default:
		return BrowseMonsters
	}
}

func (ui *UI) loadFilterStates() error {
	b, err := os.ReadFile(filtersStatePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var data PersistedFilters
	if err := yaml.Unmarshal(b, &data); err != nil {
		return err
	}
	ui.modeFilters[BrowseMonsters] = data.Monsters
	ui.modeFilters[BrowseItems] = data.Items
	ui.modeFilters[BrowseSpells] = data.Spells
	ui.modeFilters[BrowseCharacters] = data.Chars
	ui.modeFilters[BrowseRaces] = data.Races
	ui.modeFilters[BrowseFeats] = data.Feats
	ui.modeFilters[BrowseBooks] = data.Books
	ui.modeFilters[BrowseAdventures] = data.Advs
	ui.modeFilters[BrowseRandom] = data.Random
	ui.browseMode = modeFromKey(data.Active)
	return nil
}

func (ui *UI) saveFilterStates() error {
	ui.saveCurrentModeFilters()
	data := PersistedFilters{
		Version:  1,
		Active:   modeToKey(ui.browseMode),
		Monsters: ui.modeFilters[BrowseMonsters],
		Items:    ui.modeFilters[BrowseItems],
		Spells:   ui.modeFilters[BrowseSpells],
		Chars:    ui.modeFilters[BrowseCharacters],
		Races:    ui.modeFilters[BrowseRaces],
		Feats:    ui.modeFilters[BrowseFeats],
		Books:    ui.modeFilters[BrowseBooks],
		Advs:     ui.modeFilters[BrowseAdventures],
		Random:   ui.modeFilters[BrowseRandom],
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	path := filtersStatePath()
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, out, 0o644)
}

func lazy5eAppDir() string {
	if p := strings.TrimSpace(os.Getenv("LAZY5E_HOME")); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "."
	}
	return filepath.Join(home, defaultAppDirName)
}

func defaultEncountersPath() string  { return filepath.Join(lazy5eAppDir(), defaultEncountersFile) }
func defaultNotesPath() string       { return filepath.Join(lazy5eAppDir(), defaultNotesFile) }
func lastEncountersPathFile() string { return filepath.Join(lazy5eAppDir(), lastEncountersFile) }
func defaultDicePath() string        { return filepath.Join(lazy5eAppDir(), defaultDiceFile) }
func lastDicePathFile() string       { return filepath.Join(lazy5eAppDir(), lastDiceFile) }
func defaultRandomPath() string      { return filepath.Join(lazy5eAppDir(), defaultRandomFile) }
func lastRandomPathFile() string     { return filepath.Join(lazy5eAppDir(), lastRandomFile) }
func defaultBuildPath() string       { return filepath.Join(lazy5eAppDir(), defaultBuildFile) }
func lastBuildPathFile() string      { return filepath.Join(lazy5eAppDir(), lastBuildFile) }
func filtersStatePath() string       { return filepath.Join(lazy5eAppDir(), filtersStateFile) }
func descScrollStatePath() string    { return filepath.Join(lazy5eAppDir(), descScrollStateFile) }

func readLastEncountersPath() string {
	b, err := os.ReadFile(lastEncountersPathFile())
	if err != nil {
		return defaultEncountersPath()
	}
	p := strings.TrimSpace(string(b))
	if p == "" {
		return defaultEncountersPath()
	}
	return p
}

func writeLastEncountersPath(path string) error {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil
	}
	filePath := lastEncountersPathFile()
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(filePath, []byte(p+"\n"), 0o644)
}

func readLastDicePath() string {
	b, err := os.ReadFile(lastDicePathFile())
	if err != nil {
		return defaultDicePath()
	}
	p := strings.TrimSpace(string(b))
	if p == "" {
		return defaultDicePath()
	}
	return p
}

func writeLastDicePath(path string) error {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil
	}
	filePath := lastDicePathFile()
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(filePath, []byte(p+"\n"), 0o644)
}

func readLastRandomPath() string {
	b, err := os.ReadFile(lastRandomPathFile())
	if err != nil {
		return defaultRandomPath()
	}
	p := strings.TrimSpace(string(b))
	if p == "" {
		return defaultRandomPath()
	}
	return p
}

func writeLastRandomPath(path string) error {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil
	}
	filePath := lastRandomPathFile()
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(filePath, []byte(p+"\n"), 0o644)
}

func readLastBuildPath() string {
	b, err := os.ReadFile(lastBuildPathFile())
	if err != nil {
		return defaultBuildPath()
	}
	p := strings.TrimSpace(string(b))
	if p == "" {
		return defaultBuildPath()
	}
	return p
}

func writeLastBuildPath(path string) error {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil
	}
	filePath := lastBuildPathFile()
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(filePath, []byte(p+"\n"), 0o644)
}

func (ui *UI) renderRawWithHighlight(query string, lineToHighlight int) {
	ui.renderRawWithHighlightOccurrence(query, lineToHighlight, -1)
}

func (ui *UI) renderRawWithHighlightOccurrence(query string, lineToHighlight int, occToHighlight int) {
	if ui.rawText == "" {
		ui.detailRaw.SetText("")
		return
	}

	lines := strings.Split(ui.rawText, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if query != "" && i == lineToHighlight {
			b.WriteString(highlightEscapedOccurrenceWithRegion(line, query, occToHighlight, "rawmatch"))
		} else {
			b.WriteString(tview.Escape(line))
		}
	}
	ui.detailRaw.SetText(b.String())
	if query != "" && lineToHighlight >= 0 && occToHighlight >= 0 {
		ui.detailRaw.Highlight("rawmatch")
	} else {
		ui.detailRaw.Highlight()
	}
}

func highlightEscaped(line, query string) string {
	return highlightEscapedOccurrence(line, query, -1)
}

func highlightEscapedOccurrence(line, query string, occToHighlight int) string {
	return highlightEscapedOccurrenceWithRegion(line, query, occToHighlight, "")
}

func highlightEscapedOccurrenceWithRegion(line, query string, occToHighlight int, regionID string) string {
	if query == "" {
		return tview.Escape(line)
	}
	lowerLine := strings.ToLower(line)
	lowerQuery := strings.ToLower(query)

	var b strings.Builder
	start := 0
	occ := 0
	for {
		idx := strings.Index(lowerLine[start:], lowerQuery)
		if idx < 0 {
			b.WriteString(tview.Escape(line[start:]))
			break
		}
		abs := start + idx
		end := abs + len(query)
		b.WriteString(tview.Escape(line[start:abs]))
		if occToHighlight < 0 || occ == occToHighlight {
			if occ == occToHighlight && regionID != "" {
				b.WriteString("[\"")
				b.WriteString(regionID)
				b.WriteString("\"]")
			}
			b.WriteString("[black:gold]")
			b.WriteString(tview.Escape(line[abs:end]))
			b.WriteString("[-:-]")
			if occ == occToHighlight && regionID != "" {
				b.WriteString("[\"\"]")
			}
		} else {
			b.WriteString(tview.Escape(line[abs:end]))
		}
		start = end
		occ++
		if start >= len(line) {
			break
		}
	}
	return b.String()
}

func (ui *UI) findRawMatch(query string) (int, bool) {
	if strings.TrimSpace(query) == "" || ui.rawText == "" {
		return 0, false
	}
	lines := strings.Split(ui.rawText, "\n")
	if len(lines) == 0 {
		return 0, false
	}

	start, _ := ui.detailRaw.GetScrollOffset()
	if start < 0 {
		start = 0
	}
	if start >= len(lines) {
		start = len(lines) - 1
	}
	startOcc := rawLineMatchCount(lines[start], query) - 1
	line, _, ok := ui.findNextRawOccurrence(query, start, startOcc, true)
	if ok {
		return line, true
	}
	line, _, ok = ui.findNextRawOccurrence(query, -1, -1, true)
	return line, ok
}

func rawLineMatchCount(line, query string) int {
	if query == "" {
		return 0
	}
	lowerLine := strings.ToLower(line)
	lowerQuery := strings.ToLower(query)
	if lowerQuery == "" {
		return 0
	}
	count := 0
	start := 0
	for {
		idx := strings.Index(lowerLine[start:], lowerQuery)
		if idx < 0 {
			return count
		}
		count++
		start += idx + len(query)
		if start >= len(line) {
			return count
		}
	}
}

func (ui *UI) findNextRawOccurrence(query string, startLine int, startOcc int, forward bool) (int, int, bool) {
	if strings.TrimSpace(query) == "" || ui.rawText == "" {
		return 0, 0, false
	}
	lines := strings.Split(ui.rawText, "\n")
	if len(lines) == 0 {
		return 0, 0, false
	}
	if startLine < -1 {
		startLine = -1
	}
	if startLine > len(lines) {
		startLine = len(lines)
	}
	if startOcc < -1 {
		startOcc = -1
	}

	if forward {
		for l := max(0, startLine); l < len(lines); l++ {
			count := rawLineMatchCount(lines[l], query)
			if count == 0 {
				continue
			}
			first := 0
			if l == startLine {
				first = startOcc + 1
			}
			if first < 0 {
				first = 0
			}
			if first < count {
				return l, first, true
			}
		}
		for l := 0; l < len(lines); l++ {
			count := rawLineMatchCount(lines[l], query)
			if count == 0 {
				continue
			}
			if l == startLine && startOcc >= 0 && startOcc < count {
				if 0 <= startOcc {
					return l, 0, true
				}
			}
			return l, 0, true
		}
		return 0, 0, false
	}

	if startLine == len(lines) {
		startLine = len(lines) - 1
	}
	for l := min(startLine, len(lines)-1); l >= 0; l-- {
		count := rawLineMatchCount(lines[l], query)
		if count == 0 {
			continue
		}
		last := count - 1
		if l == startLine && startOcc >= 0 {
			last = startOcc - 1
		}
		if last >= 0 && last < count {
			return l, last, true
		}
	}
	for l := len(lines) - 1; l >= 0; l-- {
		count := rawLineMatchCount(lines[l], query)
		if count == 0 {
			continue
		}
		return l, count - 1, true
	}
	return 0, 0, false
}

func (ui *UI) findRawMatchFrom(query string, start int, forward bool) (int, bool) {
	if strings.TrimSpace(query) == "" || ui.rawText == "" {
		return 0, false
	}
	lines := strings.Split(ui.rawText, "\n")
	if len(lines) == 0 {
		return 0, false
	}
	if start < 0 {
		start = -1
	}
	if start >= len(lines) {
		start = len(lines)
	}

	q := strings.ToLower(query)
	if forward {
		for i := start + 1; i < len(lines); i++ {
			if strings.Contains(strings.ToLower(lines[i]), q) {
				return i, true
			}
		}
		return 0, false
	}
	for i := start - 1; i >= 0; i-- {
		if strings.Contains(strings.ToLower(lines[i]), q) {
			return i, true
		}
	}
	return 0, false
}

func loadMonstersFromPath(path string) ([]Monster, []string, []string, []string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return loadMonstersFromBytes(b)
}

func loadMonstersFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds dataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	if len(ds.Monsters) == 0 {
		return nil, nil, nil, nil, errors.New("no monster found in yaml")
	}

	// Resolve _copy references (two passes to handle chained copies).
	resolveMonstersWithCopy(ds.Monsters)
	resolveMonstersWithCopy(ds.Monsters)

	monsters := make([]Monster, 0, len(ds.Monsters))
	envSet := map[string]struct{}{}
	crSet := map[string]struct{}{}
	typeSet := map[string]struct{}{}

	for i, raw := range ds.Monsters {
		name := asString(raw["name"])
		if name == "" {
			continue
		}

		envs := asStringSlice(raw["environment"])
		for _, env := range envs {
			envSet[env] = struct{}{}
		}

		cr := extractCR(raw["cr"])
		if cr == "" {
			cr = "Unknown"
		}
		crSet[cr] = struct{}{}

		monsters = append(monsters, Monster{
			ID:          i,
			Name:        name,
			CR:          cr,
			Environment: envs,
			Source:      asString(raw["source"]),
			Type:        extractType(raw["type"]),
			Raw:         raw,
		})
		mType := extractType(raw["type"])
		if mType == "" {
			mType = "Unknown"
		}
		typeSet[mType] = struct{}{}
	}

	sort.Slice(monsters, func(i, j int) bool {
		return strings.ToLower(monsters[i].Name) < strings.ToLower(monsters[j].Name)
	})

	return monsters, keysSorted(envSet), sortCR(keysSorted(crSet)), keysSorted(typeSet), nil
}

// resolveMonstersWithCopy merges fields from _copy base entries into each entry that
// references one, then applies any _mod transformations. Fields already present on the
// copying entry are never overwritten.
func resolveMonstersWithCopy(rawEntries []map[string]any) {
	lookup := make(map[string]map[string]any, len(rawEntries))
	for _, raw := range rawEntries {
		name := asString(raw["name"])
		source := asString(raw["source"])
		if name != "" && raw["_copy"] == nil {
			key := strings.ToLower(name) + "::" + strings.ToLower(source)
			lookup[key] = raw
		}
	}
	for i, raw := range rawEntries {
		copySpec, hasCopy := raw["_copy"]
		if !hasCopy {
			continue
		}
		copyMap, ok := copySpec.(map[string]any)
		if !ok {
			continue
		}
		baseName := asString(copyMap["name"])
		baseSource := asString(copyMap["source"])
		key := strings.ToLower(baseName) + "::" + strings.ToLower(baseSource)
		base, found := lookup[key]
		if !found {
			continue
		}
		// Copy base fields that are not already present on the entry.
		for k, v := range base {
			if k == "_copy" {
				continue
			}
			if _, exists := rawEntries[i][k]; !exists {
				rawEntries[i][k] = deepCopyAny(v)
			}
		}
		// Apply _mod transformations.
		if mods, hasMod := copyMap["_mod"]; hasMod {
			applyMonsterMod(rawEntries[i], mods)
		}
		delete(rawEntries[i], "_copy")
	}
}

func applyMonsterMod(target map[string]any, mods any) {
	modMap, ok := mods.(map[string]any)
	if !ok {
		return
	}
	for field, modSpec := range modMap {
		if field == "*" {
			// replaceTxt on all fields is cosmetic (NPC name substitution); skip.
			continue
		}
		switch v := modSpec.(type) {
		case map[string]any:
			applyMonsterModOp(target, field, v)
		case []any:
			for _, item := range v {
				if m, ok2 := item.(map[string]any); ok2 {
					applyMonsterModOp(target, field, m)
				}
			}
		}
	}
}

func applyMonsterModOp(target map[string]any, field string, mod map[string]any) {
	mode := asString(mod["mode"])
	existing := func() []any {
		if s, ok := target[field].([]any); ok {
			return s
		}
		return nil
	}
	switch mode {
	case "appendArr":
		items := mod["items"]
		if items == nil {
			return
		}
		cur := existing()
		switch v := items.(type) {
		case []any:
			target[field] = append(cur, v...)
		default:
			target[field] = append(cur, v)
		}
	case "prependArr":
		items := mod["items"]
		if items == nil {
			return
		}
		cur := existing()
		switch v := items.(type) {
		case []any:
			target[field] = append(v, cur...)
		default:
			target[field] = append([]any{v}, cur...)
		}
	case "replaceArr":
		replaceName := asString(mod["replace"])
		newItem := mod["items"]
		if replaceName == "" || newItem == nil {
			return
		}
		cur := existing()
		for idx, item := range cur {
			if m, ok := item.(map[string]any); ok {
				if strings.EqualFold(asString(m["name"]), replaceName) {
					switch v := newItem.(type) {
					case []any:
						if len(v) > 0 {
							cur[idx] = v[0]
						}
					default:
						cur[idx] = v
					}
					break
				}
			}
		}
		target[field] = cur
	case "removeArr":
		name := asString(mod["names"])
		cur := existing()
		out := cur[:0]
		for _, item := range cur {
			if m, ok := item.(map[string]any); ok {
				if strings.EqualFold(asString(m["name"]), name) {
					continue
				}
			}
			out = append(out, item)
		}
		target[field] = out
	}
}

func deepCopyAny(v any) any {
	switch val := v.(type) {
	case map[string]any:
		cp := make(map[string]any, len(val))
		for k, v2 := range val {
			cp[k] = deepCopyAny(v2)
		}
		return cp
	case []any:
		cp := make([]any, len(val))
		for i, v2 := range val {
			cp[i] = deepCopyAny(v2)
		}
		return cp
	default:
		return v
	}
}

func loadItemsFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds itemsDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	if len(ds.Items) == 0 {
		return nil, nil, nil, nil, errors.New("no item found in yaml")
	}

	items := make([]Monster, 0, len(ds.Items))
	envSet := map[string]struct{}{}
	crSet := map[string]struct{}{}
	typeSet := map[string]struct{}{}

	for i, raw := range ds.Items {
		name := asString(raw["name"])
		if name == "" {
			continue
		}
		source := asString(raw["source"])
		envs := []string{}
		if source != "" {
			envs = []string{source}
			envSet[source] = struct{}{}
		}
		rarity := strings.TrimSpace(asString(raw["rarity"]))
		if rarity == "" {
			rarity = "Unknown"
		}
		crSet[rarity] = struct{}{}

		itemType := extractItemType(raw)
		if itemType == "" {
			itemType = "Unknown"
		}
		typeSet[itemType] = struct{}{}

		items = append(items, Monster{
			ID:          i,
			Name:        name,
			CR:          rarity,
			Environment: envs,
			Source:      source,
			Type:        itemType,
			Raw:         raw,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, keysSorted(envSet), keysSorted(crSet), keysSorted(typeSet), nil
}

func loadSpellsFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds spellsDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	if len(ds.Spells) == 0 {
		return nil, nil, nil, nil, errors.New("no spell found in yaml")
	}

	spells := make([]Monster, 0, len(ds.Spells))
	envSet := map[string]struct{}{}
	crSet := map[string]struct{}{}
	typeSet := map[string]struct{}{}

	for i, raw := range ds.Spells {
		name := asString(raw["name"])
		if name == "" {
			continue
		}
		source := asString(raw["source"])
		envs := []string{}
		if source != "" {
			envs = []string{source}
			envSet[source] = struct{}{}
		}
		level := extractSpellLevel(raw["level"])
		if level == "" {
			level = "Unknown"
		}
		crSet[level] = struct{}{}
		school := extractSpellSchool(raw["school"])
		if school == "" {
			school = "Unknown"
		}
		typeSet[school] = struct{}{}

		spells = append(spells, Monster{
			ID:          i,
			Name:        name,
			CR:          level,
			Environment: envs,
			Source:      source,
			Type:        school,
			Raw:         raw,
		})
	}

	sort.Slice(spells, func(i, j int) bool {
		return strings.ToLower(spells[i].Name) < strings.ToLower(spells[j].Name)
	})
	return spells, keysSorted(envSet), sortCR(keysSorted(crSet)), keysSorted(typeSet), nil
}

func loadClassesFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds classesDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	if len(ds.Classes) == 0 {
		return nil, nil, nil, nil, errors.New("no class found in yaml")
	}

	classes := make([]Monster, 0, len(ds.Classes))
	primarySet := map[string]struct{}{}
	hdSet := map[string]struct{}{}
	casterSet := map[string]struct{}{}
	subclassFeaturesByClass := loadSubclassFeatureIndexFromYAML(embeddedSubclassesYAML)
	classFeatureDetailsByClass, subclassFeatureDetailsByClass := loadClassFeatureDetailsIndexFromYAML(embeddedClassFeatureDetailsYAML)

	for i, raw := range ds.Classes {
		name := strings.TrimSpace(asString(raw["name"]))
		if name == "" {
			continue
		}
		source := strings.TrimSpace(asString(raw["source"]))
		primary := extractClassPrimary(raw)
		for _, p := range primary {
			primarySet[p] = struct{}{}
		}
		hitDie := extractClassHitDie(raw["hd"])
		if hitDie == "" {
			hitDie = "Unknown"
		}
		hdSet[hitDie] = struct{}{}
		caster := extractClassCaster(raw["casterProgression"])
		casterSet[caster] = struct{}{}
		classKey := classFeatureIndexKey(name, source)
		if features := subclassFeaturesByClass[classKey]; len(features) > 0 {
			raw["__subclassFeatures"] = append([]string(nil), features...)
		}
		if details := classFeatureDetailsByClass[classKey]; len(details) > 0 {
			raw["__classFeatureDetails"] = cloneAnySlice(details)
		}
		if details := subclassFeatureDetailsByClass[classKey]; len(details) > 0 {
			raw["__subclassFeatureDetails"] = cloneAnySlice(details)
			existingNames := asStringSlice(raw["__subclassFeatures"])
			if len(existingNames) == 0 {
				// Keep a lightweight name list for quick search.
				names := make([]string, 0, len(details))
				seen := map[string]struct{}{}
				for _, d := range details {
					n := strings.TrimSpace(asString(d["feature"]))
					if n == "" {
						continue
					}
					k := strings.ToLower(n)
					if _, ok := seen[k]; ok {
						continue
					}
					seen[k] = struct{}{}
					names = append(names, n)
				}
				if len(names) > 0 {
					raw["__subclassFeatures"] = names
				}
			}
		}

		classes = append(classes, Monster{
			ID:          i,
			Name:        name,
			CR:          hitDie,
			Environment: primary,
			Source:      source,
			Type:        caster,
			Raw:         raw,
		})
	}

	sort.Slice(classes, func(i, j int) bool {
		li := strings.ToLower(classes[i].Name + "|" + classes[i].Source)
		lj := strings.ToLower(classes[j].Name + "|" + classes[j].Source)
		return li < lj
	})
	return classes, keysSorted(primarySet), keysSorted(hdSet), keysSorted(casterSet), nil
}

func cloneAnySlice(items []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		cp := make(map[string]any, len(it))
		for k, v := range it {
			cp[k] = v
		}
		out = append(out, cp)
	}
	return out
}

func classFeatureIndexKey(className string, classSource string) string {
	return strings.ToLower(strings.TrimSpace(className)) + "|" + strings.ToLower(strings.TrimSpace(classSource))
}

func loadSubclassFeatureIndexFromYAML(b []byte) map[string][]string {
	if len(b) == 0 {
		return nil
	}
	var ds subclassesDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil
	}
	index := map[string][]string{}
	for _, sf := range ds.Features {
		className := strings.TrimSpace(sf.ClassName)
		classSource := strings.TrimSpace(sf.ClassSource)
		featureName := strings.TrimSpace(sf.Feature)
		if className == "" || classSource == "" || featureName == "" {
			continue
		}
		key := classFeatureIndexKey(className, classSource)
		index[key] = append(index[key], featureName)
	}
	for key, names := range index {
		seen := map[string]struct{}{}
		out := make([]string, 0, len(names))
		for _, name := range names {
			n := strings.TrimSpace(name)
			if n == "" {
				continue
			}
			k := strings.ToLower(n)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, n)
		}
		sort.Strings(out)
		index[key] = out
	}
	return index
}

func loadClassFeatureDetailsIndexFromYAML(b []byte) (map[string][]map[string]any, map[string][]map[string]any) {
	if len(b) == 0 {
		return nil, nil
	}
	var ds classFeatureDetailsDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil
	}
	classIdx := map[string][]map[string]any{}
	subclassIdx := map[string][]map[string]any{}
	for _, cf := range ds.ClassFeatures {
		className := strings.TrimSpace(cf.ClassName)
		classSource := strings.TrimSpace(cf.ClassSource)
		feature := strings.TrimSpace(cf.Feature)
		if className == "" || classSource == "" || feature == "" {
			continue
		}
		key := classFeatureIndexKey(className, classSource)
		classIdx[key] = append(classIdx[key], map[string]any{
			"feature": feature,
			"source":  strings.TrimSpace(cf.Source),
			"level":   cf.Level,
			"entries": cf.Entries,
		})
	}
	for _, sf := range ds.SubclassFeatures {
		className := strings.TrimSpace(sf.ClassName)
		classSource := strings.TrimSpace(sf.ClassSource)
		feature := strings.TrimSpace(sf.Feature)
		if className == "" || classSource == "" || feature == "" {
			continue
		}
		key := classFeatureIndexKey(className, classSource)
		subclassIdx[key] = append(subclassIdx[key], map[string]any{
			"subclass_name":   strings.TrimSpace(sf.SubclassName),
			"subclass_source": strings.TrimSpace(sf.SubclassSource),
			"feature":         feature,
			"source":          strings.TrimSpace(sf.Source),
			"level":           sf.Level,
			"entries":         sf.Entries,
		})
	}
	less := func(a, b map[string]any) bool {
		al, _ := anyToInt(a["level"])
		bl, _ := anyToInt(b["level"])
		if al != bl {
			return al < bl
		}
		as := strings.ToLower(strings.TrimSpace(asString(a["subclass_name"])))
		bs := strings.ToLower(strings.TrimSpace(asString(b["subclass_name"])))
		if as != bs {
			return as < bs
		}
		af := strings.ToLower(strings.TrimSpace(asString(a["feature"])))
		bf := strings.ToLower(strings.TrimSpace(asString(b["feature"])))
		return af < bf
	}
	for k, items := range classIdx {
		sort.Slice(items, func(i, j int) bool { return less(items[i], items[j]) })
		classIdx[k] = dedupeFeatureDetailRows(items)
	}
	for k, items := range subclassIdx {
		sort.Slice(items, func(i, j int) bool { return less(items[i], items[j]) })
		subclassIdx[k] = dedupeFeatureDetailRows(items)
	}
	return classIdx, subclassIdx
}

func dedupeFeatureDetailRows(items []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	seen := map[string]struct{}{}
	for _, it := range items {
		name := strings.ToLower(strings.TrimSpace(asString(it["feature"])))
		level, _ := anyToInt(it["level"])
		sub := strings.ToLower(strings.TrimSpace(asString(it["subclass_name"])))
		key := fmt.Sprintf("%s|%s|%d", sub, name, level)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, it)
	}
	return out
}

func extractClassHitDie(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	faces, ok := anyToInt(m["faces"])
	if !ok || faces <= 0 {
		return ""
	}
	return fmt.Sprintf("d%d", faces)
}

func extractClassPrimary(raw map[string]any) []string {
	vals := asStringSlice(raw["proficiency"])
	out := make([]string, 0, len(vals))
	seen := map[string]struct{}{}
	for _, v := range vals {
		s := strings.ToUpper(strings.TrimSpace(v))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func extractClassCaster(v any) string {
	s := strings.TrimSpace(asString(v))
	if s == "" {
		return "none"
	}
	return s
}

func loadRacesFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds racesDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	if len(ds.Races) == 0 {
		return nil, nil, nil, nil, errors.New("no race found in yaml")
	}

	races := make([]Monster, 0, len(ds.Races))
	abilitySet := map[string]struct{}{}
	sizeSet := map[string]struct{}{}
	lineageSet := map[string]struct{}{}

	for i, raw := range ds.Races {
		name := strings.TrimSpace(asString(raw["name"]))
		if name == "" {
			continue
		}
		source := strings.TrimSpace(asString(raw["source"]))
		abilities := extractRaceAbility(raw["ability"])
		for _, a := range abilities {
			abilitySet[a] = struct{}{}
		}
		size := extractRaceSize(raw["size"])
		if size == "" {
			size = "Unknown"
		}
		sizeSet[size] = struct{}{}
		lineage := extractRaceLineage(raw["lineage"])
		lineageSet[lineage] = struct{}{}

		races = append(races, Monster{
			ID:          i,
			Name:        name,
			CR:          size,
			Environment: abilities,
			Source:      source,
			Type:        lineage,
			Raw:         raw,
		})
	}

	sort.Slice(races, func(i, j int) bool {
		li := strings.ToLower(races[i].Name + "|" + races[i].Source)
		lj := strings.ToLower(races[j].Name + "|" + races[j].Source)
		return li < lj
	})
	return races, keysSorted(abilitySet), keysSorted(sizeSet), keysSorted(lineageSet), nil
}

func loadFeatsFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds featsDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	if len(ds.Feats) == 0 {
		return nil, nil, nil, nil, errors.New("no feat found in yaml")
	}

	feats := make([]Monster, 0, len(ds.Feats))
	prereqSet := map[string]struct{}{}
	categorySet := map[string]struct{}{}
	abilitySet := map[string]struct{}{}

	for i, raw := range ds.Feats {
		name := strings.TrimSpace(asString(raw["name"]))
		if name == "" {
			continue
		}
		source := strings.TrimSpace(asString(raw["source"]))
		prereq := extractFeatPrereq(raw["prerequisite"])
		for _, p := range prereq {
			prereqSet[p] = struct{}{}
		}
		category := extractFeatCategory(raw["category"])
		categorySet[category] = struct{}{}
		ability := extractFeatAbility(raw["ability"])
		abilitySet[ability] = struct{}{}

		feats = append(feats, Monster{
			ID:          i,
			Name:        name,
			CR:          category,
			Environment: prereq,
			Source:      source,
			Type:        ability,
			Raw:         raw,
		})
	}

	sort.Slice(feats, func(i, j int) bool {
		li := strings.ToLower(feats[i].Name + "|" + feats[i].Source)
		lj := strings.ToLower(feats[j].Name + "|" + feats[j].Source)
		return li < lj
	})
	return feats, keysSorted(prereqSet), keysSorted(categorySet), keysSorted(abilitySet), nil
}

func loadBooksFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds booksDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	books := make([]Monster, 0, len(ds.Books))
	groupSet := map[string]struct{}{}
	yearSet := map[string]struct{}{}
	authorSet := map[string]struct{}{}
	for i, raw := range ds.Books {
		name := strings.TrimSpace(asString(raw["name"]))
		if name == "" {
			continue
		}
		source := strings.TrimSpace(asString(raw["source"]))
		group := strings.TrimSpace(asString(raw["group"]))
		published := strings.TrimSpace(asString(raw["published"]))
		author := strings.TrimSpace(asString(raw["author"]))
		env := []string{}
		if group != "" {
			env = append(env, group)
			groupSet[group] = struct{}{}
		}
		if source != "" {
			groupSet[source] = struct{}{}
		}
		if published != "" {
			yearSet[published] = struct{}{}
		}
		if author != "" {
			authorSet[author] = struct{}{}
		}
		books = append(books, Monster{
			ID:          i,
			Name:        name,
			CR:          published,
			Environment: env,
			Source:      source,
			Type:        author,
			Raw:         raw,
		})
	}
	sort.Slice(books, func(i, j int) bool {
		return strings.ToLower(books[i].Name) < strings.ToLower(books[j].Name)
	})
	return books, keysSorted(groupSet), keysSorted(yearSet), keysSorted(authorSet), nil
}

func loadAdventuresFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds adventuresDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	adventures := make([]Monster, 0, len(ds.Adventures))
	groupSet := map[string]struct{}{}
	yearSet := map[string]struct{}{}
	authorSet := map[string]struct{}{}
	for i, raw := range ds.Adventures {
		name := strings.TrimSpace(asString(raw["name"]))
		if name == "" {
			continue
		}
		source := strings.TrimSpace(asString(raw["source"]))
		group := strings.TrimSpace(asString(raw["group"]))
		published := strings.TrimSpace(asString(raw["published"]))
		author := strings.TrimSpace(asString(raw["author"]))
		env := []string{}
		if group != "" {
			env = append(env, group)
			groupSet[group] = struct{}{}
		}
		if source != "" {
			groupSet[source] = struct{}{}
		}
		if published != "" {
			yearSet[published] = struct{}{}
		}
		if author != "" {
			authorSet[author] = struct{}{}
		}
		adventures = append(adventures, Monster{
			ID:          i,
			Name:        name,
			CR:          published,
			Environment: env,
			Source:      source,
			Type:        author,
			Raw:         raw,
		})
	}
	sort.Slice(adventures, func(i, j int) bool {
		return strings.ToLower(adventures[i].Name) < strings.ToLower(adventures[j].Name)
	})
	return adventures, keysSorted(groupSet), keysSorted(yearSet), keysSorted(authorSet), nil
}

func extractRaceSize(v any) string {
	sz := asStringSlice(v)
	if len(sz) == 0 {
		return ""
	}
	return strings.Join(sz, "/")
}

func extractMonsterSize(v any) string {
	sz := asStringSlice(v)
	if len(sz) == 0 {
		return ""
	}
	codeToName := map[string]string{
		"T": "Tiny",
		"S": "Small",
		"M": "Medium",
		"L": "Large",
		"H": "Huge",
		"G": "Gargantuan",
	}
	out := make([]string, 0, len(sz))
	for _, s := range sz {
		u := strings.ToUpper(strings.TrimSpace(s))
		if u == "" {
			continue
		}
		if full, ok := codeToName[u]; ok {
			out = append(out, full)
			continue
		}
		out = append(out, s)
	}
	return strings.Join(out, "/")
}

func extractRaceLineage(v any) string {
	s := strings.TrimSpace(asString(v))
	if s == "" {
		return "none"
	}
	return s
}

func extractRaceAbility(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 6)
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		for k, vv := range m {
			if k == "choose" || k == "from" || k == "weighted" || k == "hidden" {
				continue
			}
			if _, ok := vv.(int); ok {
				s := strings.ToUpper(strings.TrimSpace(k))
				if s == "" {
					continue
				}
				if _, ok := seen[s]; ok {
					continue
				}
				seen[s] = struct{}{}
				out = append(out, s)
			}
		}
	}
	sort.Strings(out)
	return out
}

func extractFeatCategory(v any) string {
	s := strings.TrimSpace(asString(v))
	if s == "" {
		return "Unknown"
	}
	return s
}

func extractFeatPrereq(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, it := range arr {
		txt := strings.TrimSpace(plainAny(it))
		if txt != "" {
			out = append(out, txt)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func extractFeatAbility(v any) string {
	txt := strings.TrimSpace(plainAny(v))
	if txt == "" {
		return "none"
	}
	return txt
}

func matchName(monsterName, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(monsterName), strings.ToLower(query))
}

func matchCR(monsterCR, query string) bool {
	if query == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(monsterCR), strings.TrimSpace(query))
}

func matchEnvMulti(values []string, selected map[string]struct{}) bool {
	if len(selected) == 0 {
		return true
	}
	for _, v := range values {
		if _, ok := selected[v]; ok {
			return true
		}
	}
	return false
}

func matchEnv(values []string, query string) bool {
	if strings.TrimSpace(query) == "" {
		return true
	}
	return matchEnvMulti(values, map[string]struct{}{strings.TrimSpace(query): {}})
}

func matchType(monsterType, query string) bool {
	if query == "" {
		return true
	}
	if strings.TrimSpace(monsterType) == "" {
		monsterType = "Unknown"
	}
	return strings.EqualFold(strings.TrimSpace(monsterType), strings.TrimSpace(query))
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return ""
	}
}

func asStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		if one := asString(v); one != "" {
			return []string{one}
		}
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s := asString(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func extractCR(v any) string {
	if s := asString(v); s != "" {
		return s
	}
	switch x := v.(type) {
	case map[string]any:
		return asString(x["cr"])
	case map[any]any:
		return asString(x["cr"])
	}
	return ""
}

func extractType(v any) string {
	if s := asString(v); s != "" {
		return s
	}
	switch x := v.(type) {
	case map[string]any:
		return asString(x["type"])
	case map[any]any:
		return asString(x["type"])
	}
	return ""
}

func extractItemType(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	base := strings.TrimSpace(extractType(raw["type"]))
	flags := make([]string, 0, 3)
	boolFlag := func(key, label string) {
		v, ok := raw[key]
		if !ok {
			return
		}
		b, ok := v.(bool)
		if ok && b {
			flags = append(flags, label)
		}
	}
	boolFlag("wondrous", "wondrous")
	boolFlag("weapon", "weapon")
	boolFlag("armor", "armor")
	boolFlag("staff", "staff")
	boolFlag("ring", "ring")
	boolFlag("potion", "potion")
	boolFlag("wand", "wand")
	boolFlag("rod", "rod")
	boolFlag("scroll", "scroll")

	if base == "" && len(flags) == 0 {
		return ""
	}
	if base == "" {
		return strings.Join(flags, ", ")
	}
	if len(flags) == 0 {
		return base
	}
	return base + " (" + strings.Join(flags, ", ") + ")"
}

type itemEconomyInfo struct {
	BuyCost   string
	FindTime  string
	CraftCost string
	CraftTime string
	Procedure []string
}

func formatItemBasePrice(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	v, ok := raw["value"]
	if !ok || v == nil {
		return ""
	}
	cp, ok := anyToInt64(v)
	if !ok || cp <= 0 {
		return ""
	}
	return formatCopperValue(cp)
}

func magicItemEconomy(raw map[string]any, rarity string) (itemEconomyInfo, bool) {
	if !isMagicalItem(raw, rarity) {
		return itemEconomyInfo{}, false
	}
	key := normalizeRarity(rarity)
	switch key {
	case "common":
		return itemEconomyInfo{
			BuyCost:   "50-100 gp",
			FindTime:  "1d4 giorni",
			CraftCost: "50 gp + componenti",
			CraftTime: "1 workweek",
			Procedure: []string{"trova formula/schemi", "materiali speciali adatti alla rarita", "strumenti/proficienze richieste", "tempo di downtime e spesa del costo"},
		}, true
	case "uncommon":
		return itemEconomyInfo{
			BuyCost:   "101-500 gp",
			FindTime:  "1d6 giorni",
			CraftCost: "200 gp + componenti",
			CraftTime: "2 workweeks",
			Procedure: []string{"formula dell'oggetto", "raccolta ingredienti rari", "proficienza strumenti o Arcana", "craft in downtime"},
		}, true
	case "rare":
		return itemEconomyInfo{
			BuyCost:   "501-5,000 gp",
			FindTime:  "1d4 settimane",
			CraftCost: "2,000 gp + componenti rari",
			CraftTime: "10 workweeks",
			Procedure: []string{"schema/formula completa", "componenti da creature o luoghi speciali", "supporto artigiano o incantatore esperto", "downtime continuativo"},
		}, true
	case "very rare":
		return itemEconomyInfo{
			BuyCost:   "5,001-50,000 gp",
			FindTime:  "1d6 settimane",
			CraftCost: "20,000 gp + componenti molto rari",
			CraftTime: "25 workweeks",
			Procedure: []string{"advanced formula research", "quest for key material", "adequate lab/forge", "extended downtime with DM check"},
		}, true
	case "legendary":
		return itemEconomyInfo{
			BuyCost:   "50,001+ gp",
			FindTime:  "2d6 settimane (o piu)",
			CraftCost: "100,000 gp + componenti leggendari",
			CraftTime: "50 workweeks",
			Procedure: []string{"formula unica o perduta", "componenti leggendari ottenuti tramite avventura", "maestria elevata e laboratorio speciale", "craft lungo supervisionato dal DM"},
		}, true
	case "artifact":
		return itemEconomyInfo{
			BuyCost:   "non acquistabile",
			FindTime:  "non disponibile in negozio",
			CraftCost: "non craftabile con regole standard",
			CraftTime: "n/a",
			Procedure: []string{"solo rituali/quest eccezionali", "intervento narrativo del DM", "fonti di potere uniche"},
		}, true
	default:
		return itemEconomyInfo{
			BuyCost:   "variabile (a discrezione DM)",
			FindTime:  "da alcuni giorni a settimane",
			CraftCost: "in base a rarita/effetto",
			CraftTime: "in base a rarita/effetto",
			Procedure: []string{"definisci rarita effettiva", "determina formula e componenti", "applica downtime coerente"},
		}, true
	}
}

func isMagicalItem(raw map[string]any, rarity string) bool {
	key := normalizeRarity(rarity)
	switch key {
	case "common", "uncommon", "rare", "very rare", "legendary", "artifact", "varies":
		return true
	}
	if raw == nil {
		return false
	}
	for _, k := range []string{"wondrous", "staff", "wand", "rod", "ring", "potion", "scroll"} {
		if b, ok := raw[k].(bool); ok && b {
			return true
		}
	}
	return false
}

func normalizeRarity(r string) string {
	s := strings.ToLower(strings.TrimSpace(r))
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func anyToInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int64:
		return x, true
	case float64:
		return int64(x), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil {
			return 0, false
		}
		return int64(f), true
	default:
		return 0, false
	}
}

func formatCopperValue(cp int64) string {
	if cp <= 0 {
		return "0 cp"
	}
	pp := cp / 1000
	cp = cp % 1000
	gp := cp / 100
	cp = cp % 100
	sp := cp / 10
	cp = cp % 10
	parts := make([]string, 0, 4)
	if pp > 0 {
		parts = append(parts, fmt.Sprintf("%d pp", pp))
	}
	if gp > 0 {
		parts = append(parts, fmt.Sprintf("%d gp", gp))
	}
	if sp > 0 {
		parts = append(parts, fmt.Sprintf("%d sp", sp))
	}
	if cp > 0 {
		parts = append(parts, fmt.Sprintf("%d cp", cp))
	}
	return strings.Join(parts, " ")
}

func extractSpellLevel(v any) string {
	switch x := v.(type) {
	case int:
		if x == 0 {
			return "0"
		}
		return strconv.Itoa(x)
	case int64:
		if x == 0 {
			return "0"
		}
		return strconv.FormatInt(x, 10)
	case float64:
		i := int(x)
		if i == 0 {
			return "0"
		}
		return strconv.Itoa(i)
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return ""
		}
		return s
	default:
		return ""
	}
}

func extractSpellSchool(v any) string {
	code := strings.TrimSpace(asString(v))
	switch strings.ToUpper(code) {
	case "A":
		return "Abjuration"
	case "C":
		return "Conjuration"
	case "D":
		return "Divination"
	case "E":
		return "Enchantment"
	case "V":
		return "Evocation"
	case "I":
		return "Illusion"
	case "N":
		return "Necromancy"
	case "T":
		return "Transmutation"
	default:
		return code
	}
}

func extractSpellRange(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	return plainAny(raw["range"])
}

func extractSpellTime(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	return plainAny(raw["time"])
}

func extractSpellDuration(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	return plainAny(raw["duration"])
}

func extractInitFromDex(raw map[string]any) (int, bool) {
	if raw == nil {
		return 0, false
	}
	dexRaw, ok := raw["dex"]
	if !ok {
		return 0, false
	}
	dex, ok := anyToInt(dexRaw)
	if !ok {
		return 0, false
	}
	return (dex / 2) - 5, true
}

func anyToInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(x))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func extractHP(raw map[string]any) (average string, formula string) {
	if raw == nil {
		return "", ""
	}
	hp := raw["hp"]
	if hp == nil {
		return "", ""
	}

	switch x := hp.(type) {
	case map[string]any:
		return asString(x["average"]), asString(x["formula"])
	case map[any]any:
		return asString(x["average"]), asString(x["formula"])
	default:
		// Fallback for odd records where hp can be a scalar.
		one := asString(hp)
		return one, ""
	}
}

func extractHPAverageInt(raw map[string]any) (int, bool) {
	if raw == nil {
		return 0, false
	}
	hp := raw["hp"]
	if hp == nil {
		return 0, false
	}

	getAvg := func(v any) (int, bool) {
		s := strings.TrimSpace(asString(v))
		if s == "" {
			return 0, false
		}
		i, err := strconv.Atoi(s)
		if err != nil {
			return 0, false
		}
		return i, true
	}

	switch x := hp.(type) {
	case map[string]any:
		return getAvg(x["average"])
	case map[any]any:
		return getAvg(x["average"])
	default:
		return 0, false
	}
}

func extractAC(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	ac := raw["ac"]
	if ac == nil {
		return ""
	}

	one := func(v any) string {
		switch x := v.(type) {
		case map[string]any:
			if s := asString(x["ac"]); s != "" {
				return s
			}
		case map[any]any:
			if s := asString(x["ac"]); s != "" {
				return s
			}
		default:
			return asString(v)
		}
		return ""
	}

	switch x := ac.(type) {
	case []any:
		values := make([]string, 0, len(x))
		for _, item := range x {
			if s := one(item); s != "" {
				values = append(values, s)
			}
		}
		return strings.Join(values, ", ")
	default:
		return one(ac)
	}
}

func extractSpeed(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	v := raw["speed"]
	if v == nil {
		return ""
	}

	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case map[string]any:
		return formatSpeedMapStringAny(x)
	case map[any]any:
		tmp := map[string]any{}
		for k, val := range x {
			key := asString(k)
			if key != "" {
				tmp[key] = val
			}
		}
		return formatSpeedMapStringAny(tmp)
	default:
		return asString(v)
	}
}

func formatSpeedMapStringAny(m map[string]any) string {
	order := []string{"walk", "burrow", "climb", "fly", "swim"}
	used := map[string]struct{}{}
	parts := make([]string, 0, len(m))

	formatVal := func(key string, val any) string {
		switch x := val.(type) {
		case int:
			return fmt.Sprintf("%s %d ft.", key, x)
		case int64:
			return fmt.Sprintf("%s %d ft.", key, x)
		case float64:
			if x == float64(int64(x)) {
				return fmt.Sprintf("%s %d ft.", key, int64(x))
			}
			return fmt.Sprintf("%s %s ft.", key, strconv.FormatFloat(x, 'f', -1, 64))
		case string:
			s := strings.TrimSpace(x)
			if s == "" {
				return ""
			}
			return fmt.Sprintf("%s %s", key, s)
		case map[string]any:
			n := asString(x["number"])
			c := asString(x["condition"])
			if n != "" && c != "" {
				return fmt.Sprintf("%s %s ft. %s", key, n, c)
			}
			if n != "" {
				return fmt.Sprintf("%s %s ft.", key, n)
			}
			if c != "" {
				return fmt.Sprintf("%s %s", key, c)
			}
			return ""
		case map[any]any:
			n := asString(x["number"])
			c := asString(x["condition"])
			if n != "" && c != "" {
				return fmt.Sprintf("%s %s ft. %s", key, n, c)
			}
			if n != "" {
				return fmt.Sprintf("%s %s ft.", key, n)
			}
			if c != "" {
				return fmt.Sprintf("%s %s", key, c)
			}
			return ""
		default:
			s := asString(val)
			if s == "" {
				return ""
			}
			return fmt.Sprintf("%s %s", key, s)
		}
	}

	for _, key := range order {
		if val, ok := m[key]; ok {
			if s := formatVal(key, val); s != "" {
				parts = append(parts, s)
			}
			used[key] = struct{}{}
		}
	}

	for key, val := range m {
		if _, ok := used[key]; ok || key == "canHover" {
			continue
		}
		if s := formatVal(key, val); s != "" {
			parts = append(parts, s)
		}
	}

	return strings.Join(parts, ", ")
}

func keysSorted(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func sortCR(values []string) []string {
	sort.Slice(values, func(i, j int) bool {
		a, aok := crToFloat(values[i])
		b, bok := crToFloat(values[j])
		if aok && bok {
			if a == b {
				return values[i] < values[j]
			}
			return a < b
		}
		if aok != bok {
			return aok
		}
		return strings.ToLower(values[i]) < strings.ToLower(values[j])
	})
	return values
}

func crToFloat(cr string) (float64, bool) {
	cr = strings.TrimSpace(strings.ToLower(cr))
	if cr == "" || cr == "unknown" {
		return 0, false
	}
	if strings.Contains(cr, "/") {
		parts := strings.SplitN(cr, "/", 2)
		n, err1 := strconv.ParseFloat(parts[0], 64)
		d, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 != nil || err2 != nil || d == 0 {
			return 0, false
		}
		return n / d, true
	}
	v, err := strconv.ParseFloat(cr, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func generateIndividualTreasure(crText string, randIntn func(int) int) (treasureOutcome, error) {
	cr, ok := crToFloat(crText)
	if !ok {
		return treasureOutcome{}, errors.New("invalid cr")
	}
	if randIntn == nil {
		randIntn = rand.Intn
	}
	d100 := randIntn(100) + 1

	roll := func(n, sides, mult int, cur string) (int, string) {
		sum := 0
		for range n {
			sum += randIntn(sides) + 1
		}
		total := sum * mult
		if mult == 1 {
			return total, fmt.Sprintf("%s: %dd%d = %d", cur, n, sides, total)
		}
		return total, fmt.Sprintf("%s: %dd%d x %d = %d", cur, n, sides, mult, total)
	}

	out := treasureOutcome{
		Kind:      "Individual Treasure",
		D100:      d100,
		Coins:     map[string]int{},
		Breakdown: []string{},
	}
	add := func(cur string, n, sides, mult int) {
		total, detail := roll(n, sides, mult, cur)
		out.Coins[cur] += total
		out.Breakdown = append(out.Breakdown, detail)
	}

	switch {
	case cr <= 4:
		out.Band = "CR 0-4"
		switch {
		case d100 <= 30:
			add("cp", 5, 6, 1)
		case d100 <= 60:
			add("sp", 4, 6, 1)
		case d100 <= 70:
			add("ep", 3, 6, 1)
		case d100 <= 95:
			add("gp", 3, 6, 1)
		default:
			add("pp", 1, 6, 1)
		}
	case cr <= 10:
		out.Band = "CR 5-10"
		switch {
		case d100 <= 30:
			add("cp", 4, 6, 100)
			add("ep", 1, 6, 10)
		case d100 <= 60:
			add("sp", 6, 6, 10)
			add("gp", 2, 6, 10)
		case d100 <= 70:
			add("ep", 3, 6, 10)
			add("gp", 2, 6, 10)
		case d100 <= 95:
			add("gp", 4, 6, 10)
		default:
			add("gp", 2, 6, 10)
			add("pp", 3, 6, 1)
		}
	case cr <= 16:
		out.Band = "CR 11-16"
		switch {
		case d100 <= 20:
			add("sp", 4, 6, 100)
			add("gp", 1, 6, 100)
		case d100 <= 35:
			add("ep", 1, 6, 100)
			add("gp", 1, 6, 100)
		case d100 <= 75:
			add("gp", 2, 6, 100)
			add("pp", 1, 6, 10)
		default:
			add("gp", 2, 6, 100)
			add("pp", 2, 6, 10)
		}
	default:
		out.Band = "CR 17+"
		switch {
		case d100 <= 15:
			add("ep", 2, 6, 1000)
			add("gp", 8, 6, 100)
		case d100 <= 55:
			add("gp", 1, 6, 1000)
			add("pp", 1, 6, 100)
		default:
			add("gp", 1, 6, 1000)
			add("pp", 2, 6, 100)
		}
	}
	return out, nil
}

func generateLairTreasure(crText string, randIntn func(int) int) (treasureOutcome, error) {
	cr, ok := crToFloat(crText)
	if !ok {
		return treasureOutcome{}, errors.New("invalid cr")
	}
	if randIntn == nil {
		randIntn = rand.Intn
	}
	d100 := randIntn(100) + 1

	roll := func(n, sides, mult int, label string) (int, string) {
		sum := 0
		for range n {
			sum += randIntn(sides) + 1
		}
		total := sum * mult
		if mult == 1 {
			return total, fmt.Sprintf("%s: %dd%d = %d", label, n, sides, total)
		}
		return total, fmt.Sprintf("%s: %dd%d x %d = %d", label, n, sides, mult, total)
	}

	out := treasureOutcome{
		Kind:      "Lair (Hoard) Treasure",
		D100:      d100,
		Coins:     map[string]int{},
		Breakdown: []string{},
		Extras:    []string{},
	}
	addCoin := func(cur string, n, sides, mult int) {
		total, detail := roll(n, sides, mult, cur)
		out.Coins[cur] += total
		out.Breakdown = append(out.Breakdown, detail)
	}
	addGemArt := func(kind string, n, sides, mult int, value int) {
		total, detail := roll(n, sides, mult, kind)
		out.Breakdown = append(out.Breakdown, detail)
		if total <= 0 {
			return
		}
		if kind == "gems" {
			types := rollNamedLootTypes(total, gemTypeTableByValue(value), randIntn)
			out.Extras = append(out.Extras, fmt.Sprintf("%d gems (%d gp ciascuna): %s", total, value, strings.Join(types, "; ")))
			return
		}
		types := rollNamedLootTypes(total, artObjectTableByValue(value), randIntn)
		out.Extras = append(out.Extras, fmt.Sprintf("%d art objects (%d gp ciascuno): %s", total, value, strings.Join(types, "; ")))
	}
	addMagic := func(n, sides int, table string) {
		total, detail := roll(n, sides, 1, "Magic Items")
		out.Breakdown = append(out.Breakdown, detail)
		if total <= 0 {
			return
		}
		types := rollNamedLootTypes(total, magicItemTypeByTable(table), randIntn)
		out.Extras = append(out.Extras, fmt.Sprintf("%d item/i da Magic Item Table %s: %s", total, table, strings.Join(types, "; ")))
	}

	switch {
	case cr <= 4:
		out.Band = "CR 0-4"
		addCoin("cp", 6, 6, 100)
		addCoin("sp", 3, 6, 100)
		addCoin("gp", 2, 6, 10)
		switch {
		case d100 <= 6:
		case d100 <= 16:
			addGemArt("gems", 2, 6, 1, 10)
		case d100 <= 26:
			addGemArt("art objects", 2, 4, 1, 25)
		case d100 <= 36:
			addGemArt("gems", 2, 6, 1, 50)
		case d100 <= 44:
			addGemArt("gems", 2, 6, 1, 10)
			addMagic(1, 6, "A")
		case d100 <= 52:
			addGemArt("art objects", 2, 4, 1, 25)
			addMagic(1, 6, "A")
		case d100 <= 60:
			addGemArt("gems", 2, 6, 1, 50)
			addMagic(1, 6, "A")
		case d100 <= 65:
			addGemArt("gems", 2, 6, 1, 10)
			addMagic(1, 4, "B")
		case d100 <= 70:
			addGemArt("art objects", 2, 4, 1, 25)
			addMagic(1, 4, "B")
		case d100 <= 75:
			addGemArt("gems", 2, 6, 1, 50)
			addMagic(1, 4, "B")
		case d100 <= 78:
			addGemArt("gems", 2, 6, 1, 10)
			addMagic(1, 4, "C")
		case d100 <= 80:
			addGemArt("art objects", 2, 4, 1, 25)
			addMagic(1, 4, "C")
		case d100 <= 85:
			addGemArt("gems", 2, 6, 1, 50)
			addMagic(1, 4, "C")
		case d100 <= 92:
			addGemArt("art objects", 2, 4, 1, 25)
			addMagic(1, 4, "F")
		case d100 <= 97:
			addGemArt("gems", 2, 6, 1, 50)
			addMagic(1, 4, "F")
		case d100 <= 99:
			addGemArt("art objects", 2, 4, 1, 25)
			addMagic(1, 4, "G")
		default:
			addGemArt("gems", 2, 6, 1, 50)
			addMagic(1, 4, "G")
		}
	case cr <= 10:
		out.Band = "CR 5-10"
		addCoin("cp", 2, 6, 100)
		addCoin("sp", 2, 6, 1000)
		addCoin("gp", 6, 6, 100)
		addCoin("pp", 3, 6, 10)
		switch {
		case d100 <= 4:
		case d100 <= 10:
			addGemArt("art objects", 2, 4, 1, 25)
		case d100 <= 16:
			addGemArt("gems", 3, 6, 1, 50)
		case d100 <= 22:
			addGemArt("gems", 3, 6, 1, 100)
		case d100 <= 28:
			addGemArt("art objects", 2, 4, 1, 250)
		case d100 <= 44:
			addGemArt("gems", 3, 6, 1, 100)
			addMagic(1, 6, "A")
		case d100 <= 63:
			addGemArt("art objects", 2, 4, 1, 250)
			addMagic(1, 4, "B")
		case d100 <= 74:
			addGemArt("gems", 3, 6, 1, 100)
			addMagic(1, 4, "C")
		case d100 <= 80:
			addGemArt("art objects", 2, 4, 1, 250)
			addMagic(1, 4, "D")
		case d100 <= 94:
			addGemArt("gems", 3, 6, 1, 100)
			addMagic(1, 4, "F")
		case d100 <= 98:
			addGemArt("art objects", 2, 4, 1, 250)
			addMagic(1, 4, "G")
		default:
			addGemArt("gems", 3, 6, 1, 100)
			addMagic(1, 4, "H")
		}
	case cr <= 16:
		out.Band = "CR 11-16"
		addCoin("gp", 4, 6, 1000)
		addCoin("pp", 5, 6, 100)
		switch {
		case d100 <= 3:
		case d100 <= 15:
			addGemArt("gems", 3, 6, 1, 500)
			addMagic(1, 4, "A")
			addMagic(1, 6, "B")
		case d100 <= 29:
			addGemArt("gems", 3, 6, 1, 1000)
			addMagic(1, 4, "A")
			addMagic(1, 6, "B")
		case d100 <= 50:
			addGemArt("art objects", 2, 4, 1, 250)
			addMagic(1, 6, "C")
		case d100 <= 66:
			addGemArt("gems", 3, 6, 1, 1000)
			addMagic(1, 4, "D")
		case d100 <= 74:
			addGemArt("art objects", 2, 4, 1, 750)
			addMagic(1, 6, "E")
		case d100 <= 82:
			addGemArt("gems", 3, 6, 1, 1000)
			addMagic(1, 4, "F")
			addMagic(1, 4, "G")
		case d100 <= 94:
			addGemArt("art objects", 2, 4, 1, 750)
			addMagic(1, 4, "H")
		default:
			addGemArt("gems", 3, 6, 1, 1000)
			addMagic(1, 4, "I")
		}
	default:
		out.Band = "CR 17+"
		addCoin("gp", 12, 6, 1000)
		addCoin("pp", 8, 6, 1000)
		switch {
		case d100 <= 2:
		case d100 <= 14:
			addGemArt("gems", 3, 6, 1, 1000)
			addMagic(1, 8, "C")
		case d100 <= 46:
			addGemArt("art objects", 1, 10, 1, 2500)
			addMagic(1, 6, "D")
		case d100 <= 68:
			addGemArt("gems", 1, 8, 1, 5000)
			addMagic(1, 6, "E")
		case d100 <= 76:
			addGemArt("art objects", 1, 4, 1, 7500)
			addMagic(1, 4, "F")
			addMagic(1, 4, "G")
		case d100 <= 93:
			addGemArt("gems", 1, 8, 1, 5000)
			addMagic(1, 6, "H")
		default:
			addGemArt("art objects", 1, 4, 1, 7500)
			addMagic(1, 4, "I")
		}
	}

	return out, nil
}

func rollNamedLootTypes(count int, pool []string, randIntn func(int) int) []string {
	if count <= 0 || len(pool) == 0 {
		return nil
	}
	out := make([]string, 0, count)
	for range count {
		idx := randIntn(len(pool))
		out = append(out, pool[idx])
	}
	return out
}

func gemTypeTableByValue(value int) []string {
	switch value {
	case 10:
		return []string{"azzurrite", "agata a bande", "occhio di tigre", "ematite", "lapislazzuli", "malachite"}
	case 50:
		return []string{"sardonica", "corniola", "diaspro sanguigno", "calcedonio", "quarzo stellato", "ambra"}
	case 100:
		return []string{"ametista", "granato", "perla", "spinello", "tormalina", "topazio"}
	case 500:
		return []string{"acquamarina", "perla nera", "peridoto", "zaffiro blu pallido", "topazio imperiale", "opale nero"}
	case 1000:
		return []string{"smeraldo", "rubino", "zaffiro", "diamante giallo", "opale di fuoco", "giada imperiale"}
	case 5000:
		return []string{"diamante", "rubino stellato", "smeraldo perfetto", "zaffiro stellato", "opale di fuoco puro", "diamante blu"}
	default:
		return []string{"gemma comune"}
	}
}

func artObjectTableByValue(value int) []string {
	switch value {
	case 25:
		return []string{"anello d'argento cesellato", "coppa di rame sbalzata", "maschera cerimoniale lignea", "bracciale d'avorio", "spilla in bronzo", "statuetta in osso"}
	case 250:
		return []string{"brocca d'argento filigranata", "collana con perle piccole", "arazzo fine", "specchio in argento", "cofanetto laccato con intarsi", "icona religiosa in argento"}
	case 750:
		return []string{"corona d'oro sottile", "calice d'oro e smalto", "pendente con zaffiro", "bracciale d'oro massiccio", "arazzo di corte", "strumento musicale intarsiato"}
	case 2500:
		return []string{"diadema con gemme", "scettro d'oro e avorio", "pettorale cerimoniale", "statuetta in oro pieno", "maschera rituale in oro", "coppa regale con rubini"}
	case 7500:
		return []string{"corona regale con diamanti", "scultura in giada e oro", "calice imperiale con zaffiri", "armilla in platino", "cofanetto reale tempestato di gemme", "statuetta divina in oro e gemme"}
	default:
		return []string{"oggetto d'arte comune"}
	}
}

func magicItemTypeByTable(table string) []string {
	switch strings.ToUpper(strings.TrimSpace(table)) {
	case "A":
		return []string{"pozione", "pergamena", "munizioni +1", "sacca utility", "piccolo oggetto wondrous", "trinket magico"}
	case "B":
		return []string{"pozione maggiore", "armatura +1", "arma +1", "bastone minore", "anello minore", "oggetto wondrous non comune"}
	case "C":
		return []string{"pergamena superiore", "pozione superiore", "scudo +1", "arma +2", "verga minore", "oggetto wondrous raro"}
	case "D":
		return []string{"armatura +2", "anello raro", "bastone raro", "bacchetta rara", "oggetto wondrous raro", "arma con proprietà speciale"}
	case "E":
		return []string{"pergamena alta magia", "pozione suprema", "verga rara", "anello potente", "bastone potente", "oggetto wondrous molto raro"}
	case "F":
		return []string{"weapon +1/+2", "shield +2", "armor +1 with property", "weapon with extra damage", "martial wondrous item", "defensive ring"}
	case "G":
		return []string{"arma +2", "armatura +2", "shield +2", "verga offensiva", "bastone di potere", "oggetto wondrous molto raro"}
	case "H":
		return []string{"arma +3", "armatura +3", "anello leggendario", "bastone leggendario", "verga leggendaria", "oggetto wondrous leggendario"}
	case "I":
		return []string{"artefatto minore", "arma reliquia", "oggetto unico", "focus leggendario", "armatura mitica", "reliquia antica"}
	default:
		return []string{"oggetto magico"}
	}
}

func blankIfEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func (ui *UI) nextCustomOrdinal(name string) int {
	name = strings.TrimSpace(strings.ToLower(name))
	maxOrd := 0
	for _, it := range ui.encounterItems {
		if it.Custom && strings.TrimSpace(strings.ToLower(it.CustomName)) == name && it.Ordinal > maxOrd {
			maxOrd = it.Ordinal
		}
	}
	return maxOrd + 1
}

func parseInitInput(s string) (hasRoll bool, roll int, base int, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return false, 0, 0, false
	}
	if strings.Contains(s, "/") {
		parts := strings.SplitN(s, "/", 2)
		a, errA := strconv.Atoi(strings.TrimSpace(parts[0]))
		b, errB := strconv.Atoi(strings.TrimSpace(parts[1]))
		if errA != nil || errB != nil {
			return false, 0, 0, false
		}
		return true, a, b, true
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return false, 0, 0, false
	}
	return false, 0, v, true
}

func parseHPInput(s string) (current int, max int, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, false
	}
	if strings.Contains(s, "/") {
		parts := strings.SplitN(s, "/", 2)
		c, errC := strconv.Atoi(strings.TrimSpace(parts[0]))
		m, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
		if errC != nil || errM != nil || m < 0 {
			return 0, 0, false
		}
		if c < 0 {
			c = 0
		}
		if c > m && m > 0 {
			c = m
		}
		return c, m, true
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return 0, 0, false
	}
	return v, v, true
}

func expandDiceRollInput(input string) ([]string, error) {
	return diceroll.ExpandRollInput(input)
}

func parseDiceRollBatch(input string) (expr string, times int, err error) {
	return diceroll.ParseRollBatch(input)
}

func rollDiceExpression(expr string) (total int, breakdown string, err error) {
	return diceroll.RollExpression(expr)
}

func chooseDiceMode(mode byte, a int, b int) int {
	return diceroll.ChooseMode(mode, a, b)
}

func (ui *UI) encounterEntryName(entry EncounterEntry) string {
	if entry.Custom {
		if strings.TrimSpace(entry.CustomName) == "" {
			return "Custom"
		}
		return entry.CustomName
	}
	if entry.MonsterIndex < 0 || entry.MonsterIndex >= len(ui.monsters) {
		return "Unknown"
	}
	return ui.monsters[entry.MonsterIndex].Name
}

func (ui *UI) encounterEntryDisplay(entry EncounterEntry) string {
	name := ui.encounterEntryName(entry)
	if entry.Custom {
		return name
	}
	return fmt.Sprintf("%s #%d", name, entry.Ordinal)
}

func (ui *UI) encounterConditionsBadge(entry EncounterEntry) string {
	if len(entry.Conditions) == 0 {
		return ""
	}
	parts := make([]string, 0, len(entry.Conditions))
	for _, d := range encounterConditionDefs {
		if n, ok := entry.Conditions[d.Code]; ok && n > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", d.Code, n))
		}
	}
	if len(parts) == 0 {
		keys := make([]string, 0, len(entry.Conditions))
		for k := range entry.Conditions {
			keys = append(keys, strings.ToUpper(k))
		}
		sort.Strings(keys)
		for _, k := range keys {
			n := entry.Conditions[k]
			if n <= 0 {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s%d", k, n))
		}
	}
	return strings.Join(parts, "")
}

func (ui *UI) encounterConditionsLong(entry EncounterEntry) string {
	ordered := orderedEncounterConditions(entry.Conditions)
	if len(ordered) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ordered))
	for _, p := range ordered {
		parts = append(parts, fmt.Sprintf("%s %s", fmt.Sprintf("%s%d", strings.ToUpper(strings.TrimSpace(p.Code)), p.Rounds), conditionNameByCode(p.Code)))
	}
	return strings.Join(parts, ", ")
}

type encounterConditionState struct {
	Code   string
	Rounds int
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
	extra := make([]string, 0, len(conditions))
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

func insertConditionsLine(meta string, cond string) string {
	trimmed := strings.TrimRight(meta, "\n")
	if trimmed == "" {
		return "[white]Conditions:[-] " + cond
	}
	lines := strings.Split(trimmed, "\n")
	filtered := make([]string, 0, len(lines)+1)
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "conditions:") {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) == 0 {
		return "[white]Conditions:[-] " + cond
	}
	withCond := make([]string, 0, len(filtered)+1)
	withCond = append(withCond, filtered[0])
	withCond = append(withCond, "[white]Conditions:[-] "+cond)
	withCond = append(withCond, filtered[1:]...)
	return strings.Join(withCond, "\n")
}

func (ui *UI) encounterInitBase(entry EncounterEntry) (int, bool) {
	if entry.Custom {
		return entry.CustomInit, true
	}
	if entry.MonsterIndex < 0 || entry.MonsterIndex >= len(ui.monsters) {
		return 0, false
	}
	return extractInitFromDex(ui.monsters[entry.MonsterIndex].Raw)
}

func abilityForSkill(skill string) string {
	for _, d := range skillDefs {
		if strings.EqualFold(d.Name, strings.TrimSpace(skill)) {
			return d.Ability
		}
	}
	return ""
}

func skillBonusFromMonster(raw map[string]any, skill string) (int, bool) {
	if raw == nil {
		return 0, false
	}
	if skills, ok := raw["skill"].(map[string]any); ok {
		for k, v := range skills {
			if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(skill)) {
				if n, ok := signedIntFromAny(v); ok {
					return n, true
				}
			}
		}
	}
	if skills, ok := raw["skill"].(map[any]any); ok {
		for k, v := range skills {
			if strings.EqualFold(strings.TrimSpace(asString(k)), strings.TrimSpace(skill)) {
				if n, ok := signedIntFromAny(v); ok {
					return n, true
				}
			}
		}
	}
	ability := abilityForSkill(skill)
	if ability == "" {
		return 0, false
	}
	if score, ok := anyToInt(raw[ability]); ok {
		return abilityMod(score), true
	}
	return 0, false
}

func skillBonusFromCharacterBuild(build *CharacterBuild, skill string) (int, bool) {
	if build == nil || len(build.BaseScores) < 6 {
		return 0, false
	}
	ability := abilityForSkill(skill)
	if ability == "" {
		return 0, false
	}
	idx := map[string]int{"str": 0, "dex": 1, "con": 2, "int": 3, "wis": 4, "cha": 5}[ability]
	if idx < 0 || idx >= len(build.BaseScores) {
		return 0, false
	}
	return abilityMod(build.BaseScores[idx]), true
}

func parseNamedSkillBonusFromText(text string, skill string) (int, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || strings.TrimSpace(skill) == "" {
		return 0, false
	}
	pattern := `(?mi)^\s*` + regexp.QuoteMeta(skill) + `\s*[: ]+\s*([+-]?\d+)\s*$`
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(trimmed)
	if len(m) < 2 {
		return 0, false
	}
	v, err := strconv.Atoi(strings.TrimSpace(m[1]))
	if err != nil {
		return 0, false
	}
	return v, true
}

func (ui *UI) encounterSkillBonus(entry EncounterEntry, skill string) (int, bool) {
	if entry.Custom {
		if v, ok := skillBonusFromCharacterBuild(entry.Character, skill); ok {
			return v, true
		}
		if strings.EqualFold(strings.TrimSpace(skill), "Perception") && entry.HasCustomPassive {
			return max(0, entry.CustomPassive) - 10, true
		}
		if v, ok := parseNamedSkillBonusFromText(entry.CustomMeta, skill); ok {
			return v, true
		}
		if v, ok := parseNamedSkillBonusFromText(entry.CustomBody, skill); ok {
			return v, true
		}
		return 0, false
	}
	if entry.MonsterIndex < 0 || entry.MonsterIndex >= len(ui.monsters) {
		return 0, false
	}
	return skillBonusFromMonster(ui.monsters[entry.MonsterIndex].Raw, skill)
}

func saveKeyFromName(name string) string {
	for _, d := range saveDefs {
		if strings.EqualFold(strings.TrimSpace(d.Name), strings.TrimSpace(name)) {
			return d.Key
		}
	}
	return ""
}

func saveBonusFromMonster(raw map[string]any, saveKey string) (int, bool) {
	if raw == nil || strings.TrimSpace(saveKey) == "" {
		return 0, false
	}
	if saves, ok := raw["save"].(map[string]any); ok {
		if v, ok := saves[saveKey]; ok {
			if n, ok := signedIntFromAny(v); ok {
				return n, true
			}
		}
		for k, v := range saves {
			if strings.EqualFold(strings.TrimSpace(k), saveKey) {
				if n, ok := signedIntFromAny(v); ok {
					return n, true
				}
			}
		}
	}
	if saves, ok := raw["save"].(map[any]any); ok {
		for k, v := range saves {
			if strings.EqualFold(strings.TrimSpace(asString(k)), saveKey) {
				if n, ok := signedIntFromAny(v); ok {
					return n, true
				}
			}
		}
	}
	if score, ok := anyToInt(raw[saveKey]); ok {
		return abilityMod(score), true
	}
	return 0, false
}

func saveBonusFromCharacterBuild(build *CharacterBuild, saveKey string) (int, bool) {
	if build == nil || len(build.BaseScores) < 6 || strings.TrimSpace(saveKey) == "" {
		return 0, false
	}
	idx := map[string]int{"str": 0, "dex": 1, "con": 2, "int": 3, "wis": 4, "cha": 5}[saveKey]
	if idx < 0 || idx >= len(build.BaseScores) {
		return 0, false
	}
	return abilityMod(build.BaseScores[idx]), true
}

func parseNamedSaveBonusFromText(text string, saveName string) (int, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || strings.TrimSpace(saveName) == "" {
		return 0, false
	}
	// Accept patterns like:
	// "Strength: +5", "Strength Save: +5", "STR +5"
	pattern := `(?mi)^\s*(?:` + regexp.QuoteMeta(saveName) + `|` + regexp.QuoteMeta(strings.ToUpper(saveName[:3])) + `)(?:\s+save)?\s*[: ]+\s*([+-]?\d+)\s*$`
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(trimmed)
	if len(m) < 2 {
		return 0, false
	}
	v, err := strconv.Atoi(strings.TrimSpace(m[1]))
	if err != nil {
		return 0, false
	}
	return v, true
}

func (ui *UI) encounterSaveBonus(entry EncounterEntry, saveName string) (int, bool) {
	saveKey := saveKeyFromName(saveName)
	if saveKey == "" {
		return 0, false
	}
	if entry.Custom {
		if v, ok := saveBonusFromCharacterBuild(entry.Character, saveKey); ok {
			return v, true
		}
		if v, ok := parseNamedSaveBonusFromText(entry.CustomMeta, saveName); ok {
			return v, true
		}
		if v, ok := parseNamedSaveBonusFromText(entry.CustomBody, saveName); ok {
			return v, true
		}
		return 0, false
	}
	if entry.MonsterIndex < 0 || entry.MonsterIndex >= len(ui.monsters) {
		return 0, false
	}
	return saveBonusFromMonster(ui.monsters[entry.MonsterIndex].Raw, saveKey)
}

func (ui *UI) openEncounterSkillCheckModal() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]

	skillNames := make([]string, 0, len(skillDefs))
	for _, d := range skillDefs {
		skillNames = append(skillNames, d.Name)
	}

	skillDrop := tview.NewDropDown().SetLabel("Skill: ")
	skillDrop.SetOptions(skillNames, nil)
	skillDrop.SetCurrentOption(0)
	skillDrop.SetLabelColor(tcell.ColorGold)
	skillDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	skillDrop.SetFieldTextColor(tcell.ColorWhite)
	skillDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)

	bonusField := tview.NewInputField().SetLabel("Bonus: ").SetFieldWidth(8)
	bonusField.SetLabelColor(tcell.ColorGold)
	bonusField.SetFieldBackgroundColor(tcell.ColorWhite)
	bonusField.SetFieldTextColor(tcell.ColorBlack)
	bonusField.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	dcField := tview.NewInputField().SetLabel("DC: ").SetFieldWidth(8)
	dcField.SetLabelColor(tcell.ColorGold)
	dcField.SetFieldBackgroundColor(tcell.ColorWhite)
	dcField.SetFieldTextColor(tcell.ColorBlack)
	dcField.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	dcField.SetText("15")

	fillBonus := func(skill string) {
		if b, ok := ui.encounterSkillBonus(entry, skill); ok {
			bonusField.SetText(strconv.Itoa(b))
			return
		}
		bonusField.SetText("")
	}
	fillBonus(skillNames[0])
	skillDrop.SetSelectedFunc(func(text string, _ int) {
		fillBonus(text)
	})

	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Skill Check ")
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetButtonBackgroundColor(tcell.ColorGold)
	form.SetButtonTextColor(tcell.ColorBlack)
	form.SetBackgroundColor(tcell.ColorBlack)
	form.AddFormItem(skillDrop)
	form.AddFormItem(bonusField)
	form.AddFormItem(dcField)

	closeModal := func() {
		ui.pages.RemovePage("encounter-skill-check")
		ui.skillCheckVisible = false
		ui.app.SetFocus(ui.encounter)
	}
	rollNow := func() {
		_, skill := skillDrop.GetCurrentOption()
		bonusText := strings.TrimSpace(bonusField.GetText())
		if bonusText == "" {
			bonusText = "0"
		}
		bonus, err := strconv.Atoi(bonusText)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid skill bonus[-:-] \"%s\"  %s", bonusText, helpText))
			return
		}
		dcText := strings.TrimSpace(dcField.GetText())
		if dcText == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid DC[-:-] \"%s\"  %s", dcText, helpText))
			return
		}
		dc, err := strconv.Atoi(dcText)
		if err != nil || dc < 0 {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid DC[-:-] \"%s\"  %s", dcText, helpText))
			return
		}
		roll := rand.Intn(20) + 1
		total := roll + bonus
		sign := "+"
		if bonus < 0 {
			sign = ""
		}
		outcome := "ko"
		if total >= dc {
			outcome = "ok"
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold] skill[-:-] %s %s vs DC %d: d20(%d) %s%d = %d (%s)  %s",
			ui.encounterEntryDisplay(entry), skill, dc, roll, sign, bonus, total, outcome, helpText))
		closeModal()
	}

	form.AddButton("Roll", rollNow)
	form.AddButton("Cancel", closeModal)
	form.SetCancelFunc(closeModal)

	skillDrop.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter, tcell.KeyTab:
			form.SetFocus(1)
		case tcell.KeyEscape:
			closeModal()
		}
	})
	skillDrop.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			form.SetFocus(1)
			return nil
		case tcell.KeyBacktab:
			form.SetFocus(4)
			return nil
		case tcell.KeyEnter:
			if !skillDrop.IsOpen() {
				form.SetFocus(1)
				return nil
			}
			return event
		case tcell.KeyEscape:
			if skillDrop.IsOpen() {
				return event
			}
			closeModal()
			return nil
		default:
			return event
		}
	})
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || (event.Key() == tcell.KeyRune && event.Rune() == 'q') {
			closeModal()
			return nil
		}
		return event
	})
	bonusField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			form.SetFocus(2)
			return nil
		case tcell.KeyTab:
			form.SetFocus(2)
			return nil
		case tcell.KeyBacktab:
			form.SetFocus(0)
			return nil
		case tcell.KeyEscape:
			closeModal()
			return nil
		default:
			if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
				closeModal()
				return nil
			}
			return event
		}
	})
	dcField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			rollNow()
			return nil
		case tcell.KeyTab:
			form.SetFocus(3)
			return nil
		case tcell.KeyBacktab:
			form.SetFocus(1)
			return nil
		case tcell.KeyEscape:
			closeModal()
			return nil
		default:
			if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
				closeModal()
				return nil
			}
			return event
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 12, 0, true).
			AddItem(nil, 0, 1, false), 56, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("encounter-skill-check", modal, true, true)
	ui.skillCheckVisible = true
	ui.app.SetFocus(form)
}

func (ui *UI) openEncounterSaveCheckModal() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]

	saveNames := make([]string, 0, len(saveDefs))
	for _, d := range saveDefs {
		saveNames = append(saveNames, d.Name)
	}

	saveDrop := tview.NewDropDown().SetLabel("Save: ")
	saveDrop.SetOptions(saveNames, nil)
	saveDrop.SetCurrentOption(0)
	saveDrop.SetLabelColor(tcell.ColorGold)
	saveDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	saveDrop.SetFieldTextColor(tcell.ColorWhite)
	saveDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)

	bonusField := tview.NewInputField().SetLabel("Bonus: ").SetFieldWidth(8)
	bonusField.SetLabelColor(tcell.ColorGold)
	bonusField.SetFieldBackgroundColor(tcell.ColorWhite)
	bonusField.SetFieldTextColor(tcell.ColorBlack)
	bonusField.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	dcField := tview.NewInputField().SetLabel("DC: ").SetFieldWidth(8)
	dcField.SetLabelColor(tcell.ColorGold)
	dcField.SetFieldBackgroundColor(tcell.ColorWhite)
	dcField.SetFieldTextColor(tcell.ColorBlack)
	dcField.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	dcField.SetText("15")

	fillBonus := func(save string) {
		if b, ok := ui.encounterSaveBonus(entry, save); ok {
			bonusField.SetText(strconv.Itoa(b))
			return
		}
		bonusField.SetText("")
	}
	fillBonus(saveNames[0])
	saveDrop.SetSelectedFunc(func(text string, _ int) {
		fillBonus(text)
	})

	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Saving Throw ")
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetButtonBackgroundColor(tcell.ColorGold)
	form.SetButtonTextColor(tcell.ColorBlack)
	form.SetBackgroundColor(tcell.ColorBlack)
	form.AddFormItem(saveDrop)
	form.AddFormItem(bonusField)
	form.AddFormItem(dcField)

	closeModal := func() {
		ui.pages.RemovePage("encounter-save-check")
		ui.saveCheckVisible = false
		ui.app.SetFocus(ui.encounter)
	}
	rollNow := func() {
		_, save := saveDrop.GetCurrentOption()
		bonusText := strings.TrimSpace(bonusField.GetText())
		if bonusText == "" {
			bonusText = "0"
		}
		bonus, err := strconv.Atoi(bonusText)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid save bonus[-:-] \"%s\"  %s", bonusText, helpText))
			return
		}
		dcText := strings.TrimSpace(dcField.GetText())
		if dcText == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid DC[-:-] \"%s\"  %s", dcText, helpText))
			return
		}
		dc, err := strconv.Atoi(dcText)
		if err != nil || dc < 0 {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid DC[-:-] \"%s\"  %s", dcText, helpText))
			return
		}
		roll := rand.Intn(20) + 1
		total := roll + bonus
		sign := "+"
		if bonus < 0 {
			sign = ""
		}
		outcome := "ko"
		if total >= dc {
			outcome = "ok"
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold] save[-:-] %s %s vs DC %d: d20(%d) %s%d = %d (%s)  %s",
			ui.encounterEntryDisplay(entry), save, dc, roll, sign, bonus, total, outcome, helpText))
		closeModal()
	}

	form.AddButton("Roll", rollNow)
	form.AddButton("Cancel", closeModal)
	form.SetCancelFunc(closeModal)

	saveDrop.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter, tcell.KeyTab:
			form.SetFocus(1)
		case tcell.KeyEscape:
			closeModal()
		}
	})
	saveDrop.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			form.SetFocus(1)
			return nil
		case tcell.KeyBacktab:
			form.SetFocus(4)
			return nil
		case tcell.KeyEnter:
			if !saveDrop.IsOpen() {
				form.SetFocus(1)
				return nil
			}
			return event
		case tcell.KeyEscape:
			if saveDrop.IsOpen() {
				return event
			}
			closeModal()
			return nil
		default:
			return event
		}
	})
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || (event.Key() == tcell.KeyRune && event.Rune() == 'q') {
			closeModal()
			return nil
		}
		return event
	})
	bonusField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			form.SetFocus(2)
			return nil
		case tcell.KeyTab:
			form.SetFocus(2)
			return nil
		case tcell.KeyBacktab:
			form.SetFocus(0)
			return nil
		case tcell.KeyEscape:
			closeModal()
			return nil
		default:
			if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
				closeModal()
				return nil
			}
			return event
		}
	})
	dcField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			rollNow()
			return nil
		case tcell.KeyTab:
			form.SetFocus(3)
			return nil
		case tcell.KeyBacktab:
			form.SetFocus(1)
			return nil
		case tcell.KeyEscape:
			closeModal()
			return nil
		default:
			if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
				closeModal()
				return nil
			}
			return event
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 12, 0, true).
			AddItem(nil, 0, 1, false), 56, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("encounter-save-check", modal, true, true)
	ui.saveCheckVisible = true
	ui.app.SetFocus(form)
}

func (ui *UI) encounterMaxHP(entry EncounterEntry) int {
	if entry.UseRolledHP && entry.RolledHP > 0 {
		return entry.RolledHP
	}
	return entry.BaseHP
}

var (
	hpFormulaRe   = regexp.MustCompile(`^\s*(\d+)\s*[dD]\s*(\d+)(?:\s*([+-])\s*(\d+))?\s*$`)
	finalResultRe = regexp.MustCompile(`[-+]?\d+`)
	ppLineRe      = regexp.MustCompile(`(?mi)^\s*\[?[^\n]*passive perception[^\n]*:\s*([0-9]+)\s*$`)
	thpLineRe     = regexp.MustCompile(`(?mi)^\s*\[?[^\n]*temp hp[^\n]*:\s*([0-9]+)\s*$`)
	numSuffixRe   = regexp.MustCompile(`\s+#\d+$`)
)

func extractPassivePerceptionFromMonster(raw map[string]any) (int, bool) {
	if raw == nil {
		return 0, false
	}
	if v, ok := anyToInt(raw["passive"]); ok {
		return max(v, 0), true
	}
	if skills, ok := raw["skill"].(map[string]any); ok {
		if p, ok := skills["perception"]; ok {
			if bonus, ok := signedIntFromAny(p); ok {
				return max(10+bonus, 0), true
			}
		}
	}
	if skills, ok := raw["skill"].(map[any]any); ok {
		if p, ok := skills["perception"]; ok {
			if bonus, ok := signedIntFromAny(p); ok {
				return max(10+bonus, 0), true
			}
		}
	}
	if wis, ok := anyToInt(raw["wis"]); ok {
		return max(10+abilityMod(wis), 0), true
	}
	return 0, false
}

func signedIntFromAny(v any) (int, bool) {
	if n, ok := anyToInt(v); ok {
		return n, true
	}
	s := strings.TrimSpace(asString(v))
	if s == "" {
		return 0, false
	}
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimPrefix(s, "+")
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

func passivePerceptionFromText(text string) (int, bool) {
	m := ppLineRe.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(m[1]))
	if err != nil {
		return 0, false
	}
	return max(n, 0), true
}

func passivePerceptionFromCharacterBuild(build *CharacterBuild) (int, bool) {
	if build == nil || len(build.BaseScores) < 5 {
		return 0, false
	}
	return max(10+abilityMod(build.BaseScores[4]), 0), true
}

func setPassivePerceptionLine(meta string, passive int) string {
	line := fmt.Sprintf("[white]Passive Perception:[-] %d", max(passive, 0))
	trimmed := strings.TrimRight(meta, "\n")
	if trimmed == "" {
		return line
	}
	lines := strings.Split(trimmed, "\n")
	filtered := make([]string, 0, len(lines)+1)
	for _, ln := range lines {
		if strings.Contains(strings.ToLower(ln), "passive perception:") {
			continue
		}
		filtered = append(filtered, ln)
	}
	filtered = append(filtered, line)
	return strings.Join(filtered, "\n")
}

func setPassivePerceptionUnknownLine(meta string) string {
	trimmed := strings.TrimRight(meta, "\n")
	line := "[white]Passive Perception:[-] ?"
	if trimmed == "" {
		return line
	}
	lines := strings.Split(trimmed, "\n")
	filtered := make([]string, 0, len(lines)+1)
	for _, ln := range lines {
		if strings.Contains(strings.ToLower(ln), "passive perception:") {
			continue
		}
		filtered = append(filtered, ln)
	}
	filtered = append(filtered, line)
	return strings.Join(filtered, "\n")
}

func (ui *UI) encounterPassivePerception(entry EncounterEntry) (int, bool) {
	if entry.Custom {
		if entry.HasCustomPassive {
			return max(entry.CustomPassive, 0), true
		}
		if p, ok := passivePerceptionFromCharacterBuild(entry.Character); ok {
			return p, true
		}
		if p, ok := passivePerceptionFromText(entry.CustomMeta); ok {
			return p, true
		}
		if p, ok := passivePerceptionFromText(entry.CustomBody); ok {
			return p, true
		}
		return 0, false
	}
	if entry.MonsterIndex < 0 || entry.MonsterIndex >= len(ui.monsters) {
		return 0, false
	}
	return extractPassivePerceptionFromMonster(ui.monsters[entry.MonsterIndex].Raw)
}

func (ui *UI) ensureEncounterPassivePerceptionLine(entry EncounterEntry, meta string) string {
	if p, ok := ui.encounterPassivePerception(entry); ok {
		return setPassivePerceptionLine(meta, p)
	}
	return setPassivePerceptionUnknownLine(meta)
}

func (ui *UI) ensureEncounterTempHPLine(entry EncounterEntry, meta string) string {
	trimmed := strings.TrimSpace(meta)
	lines := []string{}
	if trimmed != "" {
		raw := strings.Split(trimmed, "\n")
		lines = make([]string, 0, len(raw)+1)
		for _, ln := range raw {
			if thpLineRe.MatchString(ln) || strings.Contains(strings.ToLower(ln), "temp hp:") {
				continue
			}
			lines = append(lines, ln)
		}
	}
	if entry.TempHP > 0 {
		lines = append(lines, fmt.Sprintf("[white]Temp HP:[-] %d", entry.TempHP))
	}
	return strings.Join(lines, "\n")
}

func rollHPFormula(formula string) (int, bool) {
	m := hpFormulaRe.FindStringSubmatch(strings.TrimSpace(formula))
	if len(m) == 0 {
		return 0, false
	}

	nDice, err1 := strconv.Atoi(m[1])
	dieFaces, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil || nDice <= 0 || dieFaces <= 0 {
		return 0, false
	}
	if nDice > 200 || dieFaces > 10000 {
		return 0, false
	}

	total := 0
	for range nDice {
		total += rand.Intn(dieFaces) + 1
	}
	if m[3] != "" && m[4] != "" {
		mod, err := strconv.Atoi(m[4])
		if err != nil {
			return 0, false
		}
		if m[3] == "-" {
			total -= mod
		} else {
			total += mod
		}
	}
	if total < 0 {
		total = 0
	}
	return total, true
}

func cloneIntMap(src map[int]int) map[int]int {
	dst := make(map[int]int, len(src))
	maps.Copy(dst, src)
	return dst
}

func cloneStringIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]int, len(src))
	maps.Copy(dst, src)
	return dst
}

func cloneEncounterEntries(src []EncounterEntry) []EncounterEntry {
	if len(src) == 0 {
		return nil
	}
	out := make([]EncounterEntry, len(src))
	for i := range src {
		out[i] = src[i]
		out[i].Conditions = cloneStringIntMap(src[i].Conditions)
		out[i].Character = cloneCharacterBuild(src[i].Character)
	}
	return out
}

func cloneCharacterBuild(src *CharacterBuild) *CharacterBuild {
	if src == nil {
		return nil
	}
	out := &CharacterBuild{
		Name:       src.Name,
		Race:       src.Race,
		BaseScores: append([]int(nil), src.BaseScores...),
		Feats:      append([]string(nil), src.Feats...),
		Spells:     append([]string(nil), src.Spells...),
	}
	if len(src.Classes) > 0 {
		out.Classes = make([]CharacterClassLevel, len(src.Classes))
		copy(out.Classes, src.Classes)
	}
	return out
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

func setTheme() {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack
	tview.Styles.ContrastBackgroundColor = tcell.ColorBlack
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorBlack
	tview.Styles.BorderColor = tcell.ColorGold
	tview.Styles.TitleColor = tcell.ColorGold
	tview.Styles.GraphicsColor = tcell.ColorGold
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = tcell.ColorLightGray
	tview.Styles.TertiaryTextColor = tcell.ColorAqua
	tview.Styles.InverseTextColor = tcell.ColorBlack
	tview.Styles.ContrastSecondaryTextColor = tcell.ColorBlack
}

func isSubmitEvent(event *tcell.EventKey) bool {
	if event == nil {
		return false
	}
	if event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyCtrlJ || event.Key() == tcell.KeyCtrlM {
		return true
	}
	if event.Key() == tcell.KeyRune && (event.Rune() == '\n' || event.Rune() == '\r') {
		return true
	}
	return false
}

func isSubmitKey(key tcell.Key) bool {
	return key == tcell.KeyEnter || key == tcell.KeyCtrlJ || key == tcell.KeyCtrlM
}

type formSubmitAction int

const (
	submitNone formSubmitAction = iota
	submitGenerate
	submitApply
	submitCancel
	submitFocusRace
	submitFocusLevels
)

func resolveCreateCharacterSubmit(formItem int, button int, raceOpen bool) formSubmitAction {
	if raceOpen {
		return submitNone
	}
	if button == 1 {
		return submitCancel
	}
	if formItem == 0 {
		return submitFocusRace
	}
	return submitGenerate
}

func resolveEncounterEditSubmit(formItem int, button int, classOpen bool) formSubmitAction {
	if classOpen {
		return submitNone
	}
	if button == 1 {
		return submitCancel
	}
	if formItem == 0 {
		return submitFocusRace
	}
	if formItem == 1 {
		return submitFocusLevels
	}
	return submitApply
}

func (ui *UI) openCampaignSaveInput() {
	input := tview.NewInputField().
		SetLabel("Campaign name: ").
		SetFieldWidth(40)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Save Campaign ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("campaign-save")
		ui.app.SetFocus(ui.list)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		name := strings.TrimSpace(input.GetText())
		if name == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid campaign name[-:-]  %s", helpText))
			return
		}
		if strings.ContainsAny(name, "/\\") {
			ui.status.SetText(fmt.Sprintf(" [white:red] campaign name must not contain path separators[-:-]  %s", helpText))
			return
		}
		if err := ui.saveCampaign(name); err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] save campaign error[-:-] %v  %s", err, helpText))
			return
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold] campaign saved[-:-] %s  %s", name, helpText))
	})

	ui.pages.AddPage("campaign-save", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) saveCampaign(name string) error {
	dir := filepath.Join(lazy5eAppDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create campaign dir: %w", err)
	}
	if err := ui.saveEncountersAs(filepath.Join(dir, "encounters.yaml")); err != nil {
		return fmt.Errorf("save encounters: %w", err)
	}
	if err := ui.saveDiceResultsAs(filepath.Join(dir, "dice.yaml")); err != nil {
		return fmt.Errorf("save dice: %w", err)
	}
	content := strings.TrimSpace(ui.treasureText)
	if content != "" && !strings.EqualFold(content, "no treasure generated.") {
		if err := os.WriteFile(filepath.Join(dir, "treasure.yaml"), []byte(content+"\n"), 0o644); err != nil {
			return fmt.Errorf("save treasure: %w", err)
		}
	}
	ui.notesPath = filepath.Join(dir, defaultNotesFile)
	ui.saveNotes()
	ui.currentCampaign = name
	ui.updateHelpText()
	return nil
}

func (ui *UI) openCampaignLoadModal() {
	appDir := lazy5eAppDir()

	buildDirs := func() []string {
		entries, err := os.ReadDir(appDir)
		if err != nil {
			return nil
		}
		var out []string
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			out = append(out, e.Name())
		}
		return out
	}

	dirs := buildDirs()
	if len(dirs) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] no campaigns found[-:-]  %s", helpText))
		return
	}

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Load Campaign (Enter=load, r=rename, D=delete, Esc=cancel) ")
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(false)

	render := func() {
		cur := list.GetCurrentItem()
		list.Clear()
		for _, d := range dirs {
			list.AddItem(d, "", 0, nil)
		}
		if cur >= len(dirs) {
			cur = len(dirs) - 1
		}
		if cur >= 0 {
			list.SetCurrentItem(cur)
		}
	}
	render()

	ui.campaignLoadVisible = true

	closeModal := func() {
		ui.pages.RemovePage("campaign-load")
		ui.campaignLoadVisible = false
		ui.app.SetFocus(ui.list)
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			closeModal()
			return nil
		}
		if event.Key() != tcell.KeyRune {
			return event
		}
		switch event.Rune() {
		case 'D':
			idx := list.GetCurrentItem()
			if idx < 0 || idx >= len(dirs) {
				return nil
			}
			name := dirs[idx]
			if err := os.RemoveAll(filepath.Join(appDir, name)); err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] delete campaign error[-:-] %v  %s", err, helpText))
				return nil
			}
			dirs = append(dirs[:idx], dirs[idx+1:]...)
			if len(dirs) == 0 {
				closeModal()
				ui.status.SetText(fmt.Sprintf(" [black:gold] campaign deleted[-:-] %s  %s", name, helpText))
				return nil
			}
			render()
			ui.status.SetText(fmt.Sprintf(" [black:gold] campaign deleted[-:-] %s  %s", name, helpText))
			return nil
		case 'r':
			idx := list.GetCurrentItem()
			if idx < 0 || idx >= len(dirs) {
				return nil
			}
			oldName := dirs[idx]
			ui.openCampaignRenameInput(oldName, list, func(newName string) {
				if err := os.Rename(filepath.Join(appDir, oldName), filepath.Join(appDir, newName)); err != nil {
					ui.status.SetText(fmt.Sprintf(" [white:red] rename campaign error[-:-] %v  %s", err, helpText))
					return
				}
				dirs[idx] = newName
				render()
				ui.status.SetText(fmt.Sprintf(" [black:gold] campaign renamed[-:-] %s → %s  %s", oldName, newName, helpText))
			})
			return nil
		}
		return event
	})

	list.SetSelectedFunc(func(idx int, _ string, _ string, _ rune) {
		if idx < 0 || idx >= len(dirs) {
			return
		}
		name := dirs[idx]
		closeModal()
		if err := ui.loadCampaign(name); err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] load campaign error[-:-] %v  %s", err, helpText))
			return
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold] campaign loaded[-:-] %s  %s", name, helpText))
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, min(len(dirs)+2, 20), 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("campaign-load", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *UI) openCampaignRenameInput(currentName string, returnFocus tview.Primitive, onDone func(newName string)) {
	input := tview.NewInputField().
		SetLabel("New name: ").
		SetFieldWidth(40)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Rename Campaign ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	input.SetText(currentName)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("campaign-rename")
		ui.app.SetFocus(returnFocus)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		newName := strings.TrimSpace(input.GetText())
		if newName == "" || strings.ContainsAny(newName, "/\\") {
			ui.status.SetText(fmt.Sprintf(" [white:red] invalid campaign name[-:-]  %s", helpText))
			return
		}
		if newName == currentName {
			return
		}
		onDone(newName)
	})

	ui.pages.AddPage("campaign-rename", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) loadCampaign(name string) error {
	dir := filepath.Join(lazy5eAppDir(), name)
	encPath := filepath.Join(dir, "encounters.yaml")
	if fileExists(encPath) {
		prev := ui.encountersPath
		ui.encountersPath = encPath
		if err := ui.loadEncounters(); err != nil {
			ui.encountersPath = prev
			return fmt.Errorf("load encounters: %w", err)
		}
		ui.encounterUndo = ui.encounterUndo[:0]
		ui.encounterRedo = ui.encounterRedo[:0]
		ui.renderEncounterList()
		if len(ui.encounterItems) > 0 {
			ui.encounter.SetCurrentItem(0)
			ui.renderDetailByEncounterIndex(0)
		}
	}
	dicePath := filepath.Join(dir, "dice.yaml")
	if fileExists(dicePath) {
		snap := DiceUndoState{
			Items:    append([]DiceResult(nil), ui.diceLog...),
			Selected: ui.dice.GetCurrentItem(),
		}
		ui.diceUndo = append(ui.diceUndo, snap)
		ui.diceRedo = ui.diceRedo[:0]
		prev := ui.dicePath
		ui.dicePath = dicePath
		if err := ui.loadDiceResults(); err != nil {
			ui.dicePath = prev
			return fmt.Errorf("load dice: %w", err)
		}
	}
	treePath := filepath.Join(dir, "treasure.yaml")
	if fileExists(treePath) {
		b, err := os.ReadFile(treePath)
		if err != nil {
			return fmt.Errorf("load treasure: %w", err)
		}
		ui.treasureText = strings.TrimSpace(string(b))
		ui.detailTreasure.SetText(ui.treasureText)
		ui.detailTreasure.ScrollToBeginning()
	}
	ui.notesPath = filepath.Join(dir, defaultNotesFile)
	ui.loadNotes()
	if ui.browseMode == BrowseNotes {
		ui.rebuildNotesList()
	}
	ui.currentCampaign = name
	ui.updateHelpText()
	return nil
}
