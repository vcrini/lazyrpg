package swade

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/vcrini/diceroll"
	"github.com/vcrini/lazyrpg/internal/common"
	ng "github.com/vcrini/namegenerator"
)

const helpText = " [black:gold]SWADE[-:-]  [black:gold]q[-:-] esci  [black:gold]?[-:-] help  [black:gold]f[-:-] fullscreen  [black:gold]tab[-:-] focus  [black:gold]0-6[-:-] pannelli  [black:gold][[ / ]][-:-] catalogo  [black:gold]G[-:-] pannelli "

const (
	focusDice = iota
	focusPNG
	focusEncounter
	focusMonSearch
	focusMonRole
	focusMonRank
	focusMonSource
	focusMonList
	focusEqSearch
	focusEqType
	focusEqItemType
	focusEqRank
	focusEqSource
	focusEqList
	focusCardSearch
	focusCardClass
	focusCardType
	focusCardList
	focusClassSearch
	focusClassName
	focusClassSubclass
	focusClassSource
	focusClassList
	focusNotesSearch
	focusNotesList
	focusTreasure
	focusDetail
)

// DiceResult is re-exported from common.
type DiceResult = common.DiceResult

// classPreset is an alias for common.ClassPreset kept package-private.
type classPreset = common.ClassPreset

type tviewUI struct {
	app    *tview.Application
	pages  *tview.Pages
	status *tview.TextView

	dice            *tview.List
	diceLog         []DiceResult
	maxDiceLog      int
	diceRenderLock  bool
	diceGotoPending bool

	pngList         *tview.List
	encList         *tview.List
	search          *tview.InputField
	roleDrop        *tview.DropDown
	rankDrop        *tview.DropDown
	monSourceDrop   *tview.DropDown
	monList         *tview.List
	eqSearch        *tview.InputField
	eqTypeDrop      *tview.DropDown
	eqItemTypeDrop  *tview.DropDown
	eqRankDrop      *tview.DropDown
	eqSourceDrop    *tview.DropDown
	eqList          *tview.List
	cardSearch      *tview.InputField
	cardClassDrop   *tview.DropDown
	cardTypeDrop    *tview.DropDown
	cardList        *tview.List
	classSearch     *tview.InputField
	classNameDrop   *tview.DropDown
	classSubDrop    *tview.DropDown
	classSourceDrop *tview.DropDown
	classList       *tview.List
	notesSearch  *tview.InputField
	notesList    *tview.List
	detailBottom    *tview.Pages
	detail          *tview.TextView
	detailTreasure  *tview.TextView

	monstersPanel  *tview.Flex
	equipmentPanel *tview.Flex
	cardsPanel        *tview.Flex
	classesPanel      *tview.Flex
	notesPanel   *tview.Flex
	catalogPanel      *tview.Pages
	leftPanel         *tview.Flex
	mainRow           *tview.Flex

	focus    []tview.Primitive
	focusIdx int
	message  string

	pngs                []PNG
	selected            int
	monsters  []Monster
	equipment []EquipmentItem
	cards               []CardItem
	classes             []ClassItem
	encounter           []EncounterEntry
	filtered        []int
	filteredEq      []int
	filteredCards       []int
	filteredClasses     []int
	notes         []string
	filteredNotes []int
	roleOpts            []string
	rankOpts            []string
	monSourceOpts   []string
	monSourceValues []string
	eqTypeOpts      []string
	eqItemTypeOpts      []string
	eqRankOpts          []string
	eqSourceOpts        []string
	eqSourceValues      []string
	cardClassOpts       []string
	cardTypeOpts        []string
	classNameOpts       []string
	classSubOpts        []string
	classSourceOpts     []string
	classSourceValues   []string
	roleFilter          string
	rankFilter          string
	monSourceSelected map[string]bool
	eqTypeFilter      string
	eqItemTypeFilter    string
	eqRankFilter        string
	eqSourceSelected    map[string]bool
	cardClassFilter     string
	cardTypeFilter      string
	classNameFilter     string
	classSubFilter      string
	classSourceSelected map[string]bool
	catalogMode         string

	detailRaw   string
	detailQuery string
	treasureRaw string

	helpVisible     bool
	helpReturnFocus tview.Primitive

	contextMenu *contextMenuState

	gotoVisible bool

	modalVisible    bool
	modalName       string
	modalConfirmFunc func()

	fullscreenActive bool
	fullscreenTarget string
	activeBottomPane string

	encounterShowConditionEffects bool

	suppressMonSourceCallback   bool
	suppressEqSourceCallback    bool
	suppressClassSourceCallback bool
	sourceSpaceToggleActive     bool

	encInitModeActive bool
	encInitTurnIndex  int
	encInitRound      int
	encInitSorted     bool
	encGrouped        bool
	encLetterMode     bool

	campaignName string

	// g+number navigation on list panels
	listGotoPending bool
	listGotoTarget  *tview.List
	listGotoMulti   bool
	listGotoAccum   string

	// panel prefix with timer (digits 0-5)
	panelPrefixActive bool
	panelPrefixDigit  int
	panelPrefixTimer  *time.Timer

	// detail panel vim cursor
	detailCursorLine int
	detailGPending   bool

	// line number toggle
	showLineNumbers bool

	// copy/paste clipboard for encounter and PNG
	clipPNG       *PNG
	clipEncounter *EncounterEntry

	diceMacros []common.DiceMacro

	// undo/redo stacks
	undoStack []undoSnapshot
	redoStack []undoSnapshot

	timer *common.TurnTimer

	mainFlex    *tview.Flex
	mainContent tview.Primitive // currently displayed content panel

}

type undoSnapshot struct {
	pngs      []PNG
	selected  int
	encounter []EncounterEntry
}

type contextItem struct {
	label   string
	handler func()
}

type contextMenuState struct {
	items       []contextItem
	selected    int
	x, y        int
	width       int
	height      int
	returnFocus tview.Primitive
}

func Run() error {
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

	ui, err := newTViewUI()
	if err != nil {
		return err
	}
	settings := common.LoadCampaignSettings(lazyswAppDir())
	if settings.NameType != "" {
		currentNameType = settings.NameType
	}
	ui.timer = common.NewTurnTimer(settings.TurnTimerSeconds)
	if settings.LastPanel != "" && settings.LastPanel != "encounter" && settings.LastPanel != "png" && settings.LastPanel != "dice" {
		ui.switchToCatalog(settings.LastPanel)
	}
	ui.refreshAll()
	if settings.EncInitModeActive && len(ui.encounter) > 0 {
		ui.encInitModeActive = true
		ui.encInitTurnIndex = settings.EncInitTurnIndex
		if ui.encInitTurnIndex >= len(ui.encounter) {
			ui.encInitTurnIndex = 0
		}
		ui.refreshEncounter()
	}
	ui.app.SetRoot(ui.pages, true).EnableMouse(true)
	switch settings.LastPanel {
	case "encounter":
		ui.focusPanel(focusEncounter)
	case "png":
		ui.focusPanel(focusPNG)
	case "dice":
		ui.focusPanel(focusDice)
	}
	err = ui.app.Run()
	if ui.campaignName != "" {
		ui.saveCampaignState(ui.campaignName)
	}
	switch ui.focusIdx {
	case focusEncounter:
		settings.LastPanel = "encounter"
	case focusPNG:
		settings.LastPanel = "png"
	case focusDice:
		settings.LastPanel = "dice"
	default:
		settings.LastPanel = ui.catalogMode
	}
	settings.EncInitModeActive = ui.encInitModeActive
	settings.EncInitTurnIndex = ui.encInitTurnIndex
	settings.NameType = currentNameType
	_ = common.SaveCampaignSettings(lazyswAppDir(), settings)
	return err
}

func newTViewUI() (*tviewUI, error) {
	pngs, selectedName, err := loadPNGList(dataFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", dataFile, err)
	}
	monsters, err := loadMonsters(monstersFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", monstersFile, err)
	}
	equipment, err := loadEquipment(equipmentFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", equipmentFile, err)
	}
	classes, err := loadClasses(classesFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", classesFile, err)
	}
	notes, err := loadNotes(notesFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare note: %w", err)
	}
	encounter, err := loadEncounter(encounterFile, monsters)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", encounterFile, err)
	}
	diceLog, maxDiceLog, err := loadDiceHistory(diceHistoryFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", diceHistoryFile, err)
	}

	selected := -1
	if selectedName != "" {
		for i, p := range pngs {
			if p.Name == selectedName {
				selected = i
				break
			}
		}
	}
	if selected < 0 && len(pngs) > 0 {
		selected = 0
	}

	ui := &tviewUI{
		app:                           tview.NewApplication().EnableMouse(true),
		pngs:                          pngs,
		selected:                      selected,
		monsters:                      monsters,
		equipment:                     equipment,
		classes:                       classes,
		notes:                         notes,
		encounter:                     encounter,
		diceLog:                       diceLog,
		maxDiceLog:                    maxDiceLog,
		message:                       "Pronto.",
		catalogMode:                   "mostri",
		activeBottomPane:              "details",
		encounterShowConditionEffects: true,
	}
	ui.build()
	if data, err := os.ReadFile(diceMacrosFile); err == nil {
		if m, err := common.LoadDiceMacros(data); err == nil {
			ui.diceMacros = m
		}
	}
	return ui, nil
}

func (ui *tviewUI) build() {
	ui.dice = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.dice.SetBorder(true).SetTitle(" [0]-Dadi ")
	ui.dice.SetChangedFunc(func(int, string, string, rune) {
		if ui.diceRenderLock {
			return
		}
		ui.refreshDetail()
	})

	ui.pngList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.pngList.SetBorder(true).SetTitle(" [1]-PNG ")
	ui.pngList.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(ui.pngs) {
			ui.selected = index
			ui.persistPNGs()
		}
		ui.refreshDetail()
	})

	ui.encList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.encList.SetBorder(true).SetTitle(" [2]-Encounter ")
	ui.encList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

	ui.search = tview.NewInputField().SetLabel(" (u) Cerca ").SetFieldWidth(0).SetPlaceholder("nome mostro...")
	ui.search.SetChangedFunc(func(_ string) {
		ui.refreshMonsters()
		ui.refreshDetail()
	})
	ui.search.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.focusActiveCatalogList()
		}
	})
	ui.search.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown {
			ui.focusActiveCatalogList()
			return event
		}
		return event
	})

	ui.roleFilter = "Tutti"
	ui.rankFilter = "Tutti"
	ui.roleOpts, ui.rankOpts = ui.buildMonsterFilterOptions()
	ui.monSourceValues = ui.buildMonsterSourceValues()
	ui.monSourceSelected = newSourceSelection(ui.monSourceValues)
	ui.monSourceOpts = sourceMenuOptions(ui.monSourceValues, ui.monSourceSelected)

	ui.roleDrop = tview.NewDropDown().SetLabel(" (t) Ruolo ")
	ui.roleDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.roleDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.roleDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.roleDrop.SetOptions(ui.roleOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.roleFilter = text
		ui.refreshMonsters()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.roleDrop.SetCurrentOption(0)

	ui.rankDrop = tview.NewDropDown().SetLabel(" (g) Taglia ")
	ui.rankDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.rankDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.rankDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.rankDrop.SetOptions(ui.rankOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.rankFilter = text
		ui.refreshMonsters()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.rankDrop.SetCurrentOption(0)

	ui.monSourceDrop = tview.NewDropDown().SetLabel(" (y) Source ")
	ui.monSourceDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.monSourceDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.monSourceDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.monSourceDrop.SetOptions(ui.monSourceOpts, func(text string, index int) { ui.toggleMonsterSourceOption(text, index) })
	ui.suppressMonSourceCallback = true
	ui.monSourceDrop.SetCurrentOption(0)
	ui.suppressMonSourceCallback = false

	ui.monList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.monList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})
	ui.monList.SetSelectedFunc(func(int, string, string, rune) {
		ui.addSelectedMonsterToEncounter()
	})

	filters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.search, 0, 2, false).
		AddItem(ui.roleDrop, 0, 1, false).
		AddItem(ui.rankDrop, 0, 1, false).
		AddItem(ui.monSourceDrop, 0, 1, false)

	ui.monstersPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(filters, 2, 0, false).
		AddItem(ui.monList, 0, 1, true)
	ui.monstersPanel.SetBorder(true)

	ui.eqSearch = tview.NewInputField().SetLabel(" (u) Cerca ").SetFieldWidth(0).SetPlaceholder("nome equipaggiamento...")
	ui.eqSearch.SetChangedFunc(func(_ string) {
		ui.refreshEquipment()
		ui.refreshDetail()
	})
	ui.eqSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.focusActiveCatalogList()
		}
	})
	ui.eqSearch.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown {
			ui.focusActiveCatalogList()
			return event
		}
		return event
	})

	ui.eqTypeFilter = "Tutti"
	ui.eqItemTypeFilter = "Tutti"
	ui.eqRankFilter = "Tutti"
	ui.eqTypeOpts = ui.buildEquipmentTypeOptions()
	ui.eqItemTypeOpts = ui.buildEquipmentItemTypeOptions()
	ui.eqRankOpts = ui.buildEquipmentRankOptions()
	ui.eqSourceValues = ui.buildEquipmentSourceValues()
	ui.eqSourceSelected = newSourceSelection(ui.eqSourceValues)
	ui.eqSourceOpts = sourceMenuOptions(ui.eqSourceValues, ui.eqSourceSelected)

	ui.eqTypeDrop = tview.NewDropDown().SetLabel(" Categoria ")
	ui.eqTypeDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.eqTypeDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.eqTypeDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.eqTypeDrop.SetOptions(ui.eqTypeOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.eqTypeFilter = text
		ui.refreshEquipment()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.eqTypeDrop.SetCurrentOption(0)

	ui.eqItemTypeDrop = tview.NewDropDown().SetLabel(" (t) Tipo ")
	ui.eqItemTypeDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.eqItemTypeDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.eqItemTypeDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.eqItemTypeDrop.SetOptions(ui.eqItemTypeOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.eqItemTypeFilter = text
		ui.refreshEquipment()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.eqItemTypeDrop.SetCurrentOption(0)

	ui.eqRankDrop = tview.NewDropDown().SetLabel(" (g) Era ")
	ui.eqRankDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.eqRankDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.eqRankDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.eqRankDrop.SetOptions(ui.eqRankOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.eqRankFilter = text
		ui.refreshEquipment()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.eqRankDrop.SetCurrentOption(0)

	ui.eqSourceDrop = tview.NewDropDown().SetLabel(" (y) Source ")
	ui.eqSourceDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.eqSourceDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.eqSourceDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.eqSourceDrop.SetOptions(ui.eqSourceOpts, func(text string, index int) { ui.toggleEquipmentSourceOption(text, index) })
	ui.suppressEqSourceCallback = true
	ui.eqSourceDrop.SetCurrentOption(0)
	ui.suppressEqSourceCallback = false

	ui.eqList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.eqList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

	eqFilters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.eqSearch, 0, 2, false).
		AddItem(ui.eqTypeDrop, 0, 1, false).
		AddItem(ui.eqItemTypeDrop, 0, 1, false).
		AddItem(ui.eqRankDrop, 0, 1, false).
		AddItem(ui.eqSourceDrop, 0, 1, false)

	ui.equipmentPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(eqFilters, 2, 0, false).
		AddItem(ui.eqList, 0, 1, true)
	ui.equipmentPanel.SetBorder(true)

	ui.cardSearch = tview.NewInputField().SetLabel(" (u) Cerca ").SetFieldWidth(0).SetPlaceholder("nome carta...")
	ui.cardSearch.SetChangedFunc(func(_ string) {
		ui.refreshCards()
		ui.refreshDetail()
	})
	ui.cardSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.focusActiveCatalogList()
		}
	})
	ui.cardSearch.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown {
			ui.focusActiveCatalogList()
			return event
		}
		return event
	})

	ui.cardClassFilter = "Tutti"
	ui.cardTypeFilter = "Tutti"
	ui.cardClassOpts = ui.buildCardClassOptions()
	ui.cardTypeOpts = ui.buildCardTypeOptions()

	ui.cardClassDrop = tview.NewDropDown().SetLabel(" (t) Classe ")
	ui.cardClassDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.cardClassDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.cardClassDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.cardClassDrop.SetOptions(ui.cardClassOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.cardClassFilter = text
		ui.refreshCards()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.cardClassDrop.SetCurrentOption(0)

	ui.cardTypeDrop = tview.NewDropDown().SetLabel(" (g) Tipo ")
	ui.cardTypeDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.cardTypeDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.cardTypeDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.cardTypeDrop.SetOptions(ui.cardTypeOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.cardTypeFilter = text
		ui.refreshCards()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.cardTypeDrop.SetCurrentOption(0)

	ui.cardList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.cardList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

	cardFilters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.cardSearch, 0, 2, false).
		AddItem(ui.cardClassDrop, 0, 1, false).
		AddItem(ui.cardTypeDrop, 0, 1, false)

	ui.cardsPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(cardFilters, 2, 0, false).
		AddItem(ui.cardList, 0, 1, true)
	ui.cardsPanel.SetBorder(true)

	ui.classSearch = tview.NewInputField().SetLabel(" (u) Cerca ").SetFieldWidth(0).SetPlaceholder("categoria/voce...")
	ui.classSearch.SetChangedFunc(func(_ string) {
		ui.refreshClasses()
		ui.refreshDetail()
	})
	ui.classSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.focusActiveCatalogList()
		}
	})
	ui.classSearch.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown {
			ui.focusActiveCatalogList()
			return event
		}
		return event
	})

	ui.classNameFilter = "Tutti"
	ui.classSubFilter = "Tutti"
	ui.classNameOpts = ui.buildClassNameOptions()
	ui.classSubOpts = ui.buildClassSubclassOptions()
	ui.classSourceValues = ui.buildClassSourceValues()
	ui.classSourceSelected = newSourceSelection(ui.classSourceValues)
	ui.classSourceOpts = sourceMenuOptions(ui.classSourceValues, ui.classSourceSelected)

	ui.classNameDrop = tview.NewDropDown().SetLabel(" (t) Categoria ")
	ui.classNameDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.classNameDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.classNameDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.classNameDrop.SetOptions(ui.classNameOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.classNameFilter = text
		ui.refreshClasses()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.classNameDrop.SetCurrentOption(0)

	ui.classSubDrop = tview.NewDropDown().SetLabel(" (g) Voce ")
	ui.classSubDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.classSubDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.classSubDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.classSubDrop.SetOptions(ui.classSubOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.classSubFilter = text
		ui.refreshClasses()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.classSubDrop.SetCurrentOption(0)

	ui.classSourceDrop = tview.NewDropDown().SetLabel(" (y) Source ")
	ui.classSourceDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.classSourceDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.classSourceDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.classSourceDrop.SetOptions(ui.classSourceOpts, func(text string, index int) { ui.toggleClassSourceOption(text, index) })
	ui.suppressClassSourceCallback = true
	ui.classSourceDrop.SetCurrentOption(0)
	ui.suppressClassSourceCallback = false
	ui.updateSourceDropLabels()

	ui.classList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.classList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

	classFilters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.classSearch, 0, 2, false).
		AddItem(ui.classNameDrop, 0, 1, false).
		AddItem(ui.classSubDrop, 0, 1, false).
		AddItem(ui.classSourceDrop, 0, 1, false)

	ui.classesPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(classFilters, 2, 0, false).
		AddItem(ui.classList, 0, 1, true)
	ui.classesPanel.SetBorder(true)

	ui.notesSearch = tview.NewInputField().SetLabel(" (u) Cerca ").SetFieldWidth(0).SetPlaceholder("testo nota...")
	ui.notesSearch.SetChangedFunc(func(_ string) {
		ui.refreshNotes()
	})
	ui.notesSearch.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyTab, tcell.KeyEnter:
			ui.app.SetFocus(ui.notesList)
		case tcell.KeyBacktab:
			ui.focusPrev()
		}
	})
	ui.notesList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.notesList.SetBorder(false)
	ui.notesList.SetChangedFunc(func(_ int, _ string, _ string, _ rune) {
		ui.refreshDetail()
	})
	notesFilters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.notesSearch, 0, 1, false)
	ui.notesPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(notesFilters, 2, 0, false).
		AddItem(ui.notesList, 0, 1, true)
	ui.notesPanel.SetBorder(true)

	ui.catalogPanel = tview.NewPages().
		AddPage("mostri", ui.monstersPanel, true, true).
		AddPage("equipaggiamento", ui.equipmentPanel, true, false).
		AddPage("regole", ui.classesPanel, true, false).
		AddPage("note", ui.notesPanel, true, false)
	// Clicking on the border/title area of a catalog sub-panel focuses the active list.
	// For border clicks (outside the sub-panel's inner rect), no child primitive will
	// consume the event, so we must consume it ourselves to trigger a redraw.
	ui.catalogPanel.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		// Handle both MouseLeftDown and MouseLeftClick: MouseLeftDown on the top border
		// row is intercepted by the global divider handler; MouseLeftClick is the fallback.
		// IMPORTANT: SetMouseCapture fires for ALL events reaching catalogPanel via the
		// parent Flex iteration — including out-of-rect events from sibling panels
		// (dice, pngList, encList) that don't consume MouseLeftDown. Always verify the
		// click is actually within catalogPanel before acting.
		if action == tview.MouseLeftDown || action == tview.MouseLeftClick {
			x, y := event.Position()
			px, py, pw, ph := ui.catalogPanel.GetRect()
			if x < px || x >= px+pw || y < py || y >= py+ph {
				return action, event // click outside our rect, don't interfere
			}
			var activePanel *tview.Flex
			switch ui.catalogMode {
			case "mostri":
				activePanel = ui.monstersPanel
			case "equipaggiamento":
				activePanel = ui.equipmentPanel
			case "regole":
				activePanel = ui.classesPanel
			case "note":
				activePanel = ui.notesPanel
			}
			if activePanel != nil {
				ix, iy, iw, ih := activePanel.GetInnerRect()
				if x < ix || x >= ix+iw || y < iy || y >= iy+ih {
					// Border/title click: focus the active list and consume to redraw.
					ui.focusActiveCatalogList()
					return tview.MouseConsumed, nil
				}
				// Inner area click on MouseLeftDown: pre-emptively focus the active
				// list. tview only generates MouseLeftClick if the mouse doesn't move
				// between Down and Up; if it does, List.MouseHandler never fires and
				// focus is never set. Filter widgets (InputField, DropDown) handle
				// MouseLeftDown themselves and will override this focus if clicked.
				if action == tview.MouseLeftDown {
					ui.focusActiveCatalogList()
				}
			}
		}
		return action, event
	})
	ui.refreshCatalogTitles()

	ui.leftPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.dice, 7, 0, false).
		AddItem(ui.pngList, 0, 1, true).
		AddItem(ui.encList, 0, 1, false).
		AddItem(ui.catalogPanel, 0, 1, false)

	ui.detail = tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	ui.detail.SetBorder(true).SetTitle(" Dettagli ")

	ui.detailTreasure = tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	ui.detailTreasure.SetBorder(true).SetTitle(" Treasure ")
	ui.treasureRaw = "Nessun treasure generato."
	ui.renderTreasure()

	ui.detailBottom = tview.NewPages().
		AddPage("details", ui.detail, true, true).
		AddPage("treasure", ui.detailTreasure, true, false)

	ui.mainRow = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.leftPanel, 0, 1, false).
		AddItem(ui.detailBottom, 0, 1, false)

	ui.status = tview.NewTextView().SetDynamicColors(true).SetText(helpText)
	ui.status.SetBackgroundColor(tcell.ColorBlack)

	ui.mainFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	ui.pages = tview.NewPages().AddPage("main", ui.mainFlex, true, true)
	ui.rebuildMainLayout()
	ui.buildTimerOverlay()
	ui.focus = []tview.Primitive{
		ui.dice,
		ui.pngList, ui.encList, ui.search, ui.roleDrop, ui.rankDrop, ui.monSourceDrop, ui.monList,
		ui.eqSearch, ui.eqTypeDrop, ui.eqItemTypeDrop, ui.eqRankDrop, ui.eqSourceDrop, ui.eqList,
		ui.cardSearch, ui.cardClassDrop, ui.cardTypeDrop, ui.cardList,
		ui.classSearch, ui.classNameDrop, ui.classSubDrop, ui.classSourceDrop, ui.classList,
		ui.notesSearch, ui.notesList,
		ui.detailTreasure,
		ui.detail,
	}
	ui.focusIdx = focusMonList
	ui.app.SetFocus(ui.monList)
	ui.app.SetInputCapture(ui.handleGlobalKeys)
	ui.setupDividerResize()
	ui.renderDiceList()
	ui.refreshNotes()
	ui.setFocusCallbacks()
}

func (ui *tviewUI) setFocusCallbacks() {
	ui.dice.SetFocusFunc(func() { ui.focusIdx = focusDice; ui.refreshStatus() })
	ui.pngList.SetFocusFunc(func() { ui.focusIdx = focusPNG; ui.refreshStatus() })
	ui.encList.SetFocusFunc(func() { ui.focusIdx = focusEncounter; ui.refreshStatus() })
	ui.monList.SetFocusFunc(func() { ui.focusIdx = focusMonList; ui.refreshStatus() })
	ui.eqList.SetFocusFunc(func() { ui.focusIdx = focusEqList; ui.refreshStatus() })
	ui.cardList.SetFocusFunc(func() { ui.focusIdx = focusCardList; ui.refreshStatus() })
	ui.classList.SetFocusFunc(func() { ui.focusIdx = focusClassList; ui.refreshStatus() })
	ui.notesList.SetFocusFunc(func() { ui.focusIdx = focusNotesList; ui.refreshStatus() })
}

func (ui *tviewUI) setupDividerResize() {
	// Manage mouse tracking mode: motion events when context menu is open, drag otherwise.
	var currentMouseMode tcell.MouseFlags
	ui.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		want := tcell.MouseDragEvents
		if ui.contextMenu != nil {
			want = tcell.MouseMotionEvents
		}
		if want != currentMouseMode {
			screen.EnableMouse(want)
			currentMouseMode = want
		}
		return false
	})

	type vRow struct {
		flex  *tview.Flex
		items []tview.Primitive
	}
	vRows := []vRow{
		{ui.leftPanel, []tview.Primitive{ui.dice, ui.pngList, ui.encList, ui.catalogPanel}},
	}

	var hDragging, hDragged bool
	var vFlex *tview.Flex
	var vTopItem tview.Primitive
	var vItems []tview.Primitive
	var vDragged bool

	// Returning nil as the event sets consumed=true in tview and triggers a.draw().
	ui.app.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		if event == nil {
			return nil, action
		}
		col, row := event.Position()

		// Context menu hover: update selection by mouse position.
		if action == tview.MouseMove && ui.contextMenu != nil {
			m := ui.contextMenu
			idx := row - (m.y + 1)
			if idx >= 0 && idx < len(m.items) {
				m.selected = idx
			}
			return nil, action
		}

		// Context menu: left click inside executes item, outside closes.
		if (action == tview.MouseLeftDown || action == tview.MouseLeftClick) && ui.contextMenu != nil {
			m := ui.contextMenu
			if col >= m.x && col < m.x+m.width && row >= m.y && row < m.y+m.height {
				idx := row - (m.y + 1)
				if idx >= 0 && idx < len(m.items) {
					handler := m.items[idx].handler
					ui.closeContextMenu()
					handler()
				}
				return nil, action
			}
			ui.closeContextMenu()
			return nil, action
		}

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
				hDragged = true
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
				ui.mainRow.ResizeItem(ui.detailBottom, 0, 1)
				return nil, action
			}
			if vFlex != nil {
				vDragged = true
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
				dragged := hDragged
				hDragging = false
				hDragged = false
				if dragged {
					return nil, action
				}
				return event, action
			}
			if vFlex != nil {
				dragged := vDragged
				vFlex = nil
				vTopItem = nil
				vItems = nil
				vDragged = false
				if dragged {
					return nil, action
				}
				return event, action
			}
		case tview.MouseRightClick:
			pageName, _ := ui.pages.GetFrontPage()
			if pageName != "main" {
				return nil, action
			}
			for _, p := range []tview.Primitive{ui.dice, ui.pngList, ui.encList, ui.monList, ui.eqList, ui.cardList, ui.classList, ui.notesList} {
				px, py, pw, ph := p.GetRect()
				if col >= px && col < px+pw && row >= py && row < py+ph {
					items := ui.contextMenuItemsForFocus(p)
					if len(items) > 0 {
						ui.app.SetFocus(p)
						ui.showContextMenu(items, p, col, row)
						return nil, action
					}
				}
			}
		}
		return event, action
	})

	// Block left clicks from reaching main page content when a modal is in front.
	ui.mainFlex.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if event == nil {
			return action, event
		}
		if action == tview.MouseLeftDown || action == tview.MouseLeftClick {
			if pageName, _ := ui.pages.GetFrontPage(); pageName != "main" {
				return tview.MouseConsumed, nil
			}
		}
		return action, event
	})
}

func (ui *tviewUI) closeContextMenu() {
	if ui.contextMenu == nil {
		return
	}
	returnFocus := ui.contextMenu.returnFocus
	ui.contextMenu = nil
	ui.app.SetAfterDrawFunc(nil)
	ui.app.SetFocus(returnFocus)
}

func (ui *tviewUI) drawContextMenu(screen tcell.Screen) {
	m := ui.contextMenu
	if m == nil {
		return
	}
	borderSt := tcell.StyleDefault.Foreground(tcell.ColorGold).Background(tcell.ColorBlack)
	normalSt := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	selectedSt := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold)

	// Top border with title
	screen.SetContent(m.x, m.y, '┌', nil, borderSt)
	for i := 1; i < m.width-1; i++ {
		screen.SetContent(m.x+i, m.y, '─', nil, borderSt)
	}
	screen.SetContent(m.x+m.width-1, m.y, '┐', nil, borderSt)
	title := " Context Menu "
	titleX := m.x + (m.width-len(title))/2
	for i, ch := range title {
		screen.SetContent(titleX+i, m.y, ch, nil, borderSt)
	}

	// Item rows
	innerW := m.width - 2
	for i, item := range m.items {
		y := m.y + 1 + i
		st := normalSt
		if i == m.selected {
			st = selectedSt
		}
		screen.SetContent(m.x, y, '│', nil, borderSt)
		runes := []rune(" " + item.label)
		for j := 0; j < innerW; j++ {
			ch := ' '
			if j < len(runes) {
				ch = runes[j]
			}
			screen.SetContent(m.x+1+j, y, ch, nil, st)
		}
		screen.SetContent(m.x+m.width-1, y, '│', nil, borderSt)
	}

	// Bottom border
	bY := m.y + 1 + len(m.items)
	screen.SetContent(m.x, bY, '└', nil, borderSt)
	for i := 1; i < m.width-1; i++ {
		screen.SetContent(m.x+i, bY, '─', nil, borderSt)
	}
	screen.SetContent(m.x+m.width-1, bY, '┘', nil, borderSt)
}

func (ui *tviewUI) showContextMenu(items []contextItem, returnFocus tview.Primitive, clickCol, clickRow int) {
	maxLen := 0
	for _, item := range items {
		if l := len([]rune(item.label)); l > maxLen {
			maxLen = l
		}
	}
	width := maxLen + 4
	if width < 30 {
		width = 30
	}
	height := len(items) + 2

	_, _, screenW, screenH := ui.pages.GetRect()
	menuX := clickCol
	menuY := clickRow + 1
	if menuX+width > screenW {
		menuX = screenW - width
	}
	if menuX < 0 {
		menuX = 0
	}
	if menuY+height > screenH {
		menuY = clickRow - height
	}
	if menuY < 0 {
		menuY = 0
	}

	ui.contextMenu = &contextMenuState{
		items:       items,
		selected:    0,
		x:           menuX,
		y:           menuY,
		width:       width,
		height:      height,
		returnFocus: returnFocus,
	}
	ui.app.SetAfterDrawFunc(func(screen tcell.Screen) {
		if ui.contextMenu == nil {
			return
		}
		ui.drawContextMenu(screen)
	})
}

func (ui *tviewUI) handleGlobalKeys(ev *tcell.EventKey) *tcell.EventKey {
	if ui.contextMenu != nil {
		m := ui.contextMenu
		switch ev.Key() {
		case tcell.KeyEscape:
			ui.closeContextMenu()
			return nil
		case tcell.KeyUp:
			if m.selected > 0 {
				m.selected--
			}
			return nil
		case tcell.KeyDown:
			if m.selected < len(m.items)-1 {
				m.selected++
			}
			return nil
		case tcell.KeyEnter:
			handler := m.items[m.selected].handler
			ui.closeContextMenu()
			handler()
			return nil
		}
		return nil
	}

	if ev.Key() == tcell.KeyEscape && ui.timer.Running {
		ui.stopTurnTimer()
		return nil
	}

	realFocus := ui.app.GetFocus()
	_, focusIsInput := realFocus.(*tview.InputField)
	_, focusIsDropDown := realFocus.(*tview.DropDown)
	focusIsWidget := focusIsInput || focusIsDropDown

	// Panel prefix: when active, route key to the prefixed panel's context.
	focus := realFocus
	if ui.panelPrefixActive {
		if ev.Key() == tcell.KeyEscape {
			ui.cancelPanelPrefix()
			return nil
		}
		ui.ensureCatalogForDigit(ui.panelPrefixDigit)
		focus = ui.listForPanelDigit(ui.panelPrefixDigit)
		if ui.panelPrefixTimer != nil {
			ui.panelPrefixTimer.Stop()
			ui.panelPrefixTimer = nil
		}
		ui.panelPrefixActive = false
		ui.message = ""
	}

	if ui.helpVisible {
		if ev.Key() == tcell.KeyEscape || (ev.Key() == tcell.KeyRune && (ev.Rune() == '?' || ev.Rune() == 'q')) {
			ui.closeHelpOverlay()
			return nil
		}
		return ev
	}

	if ui.gotoVisible {
		return ev
	}
	if ui.modalVisible {
		if ev.Key() == tcell.KeyCtrlO && ui.modalConfirmFunc != nil {
			ui.modalConfirmFunc()
			return nil
		}
		if ev.Key() == tcell.KeyEscape {
			ui.closeModal()
			return nil
		}
		return ev
	}

	if ui.diceGotoPending {
		if ev.Key() == tcell.KeyEscape {
			ui.diceGotoPending = false
			ui.message = "Jump dadi annullato."
			ui.refreshStatus()
			return nil
		}
		if ev.Key() == tcell.KeyRune {
			ui.diceGotoPending = false
			if idx, ok := diceGotoIndexFromRune(ev.Rune(), len(ui.diceLog)); ok {
				ui.dice.SetCurrentItem(idx)
				ui.message = fmt.Sprintf("Jump dadi: #%d", idx+1)
				ui.refreshDetail()
			} else {
				ui.message = fmt.Sprintf("Riga dadi non valida: %q", string(ev.Rune()))
			}
			ui.refreshStatus()
			return nil
		}
		ui.diceGotoPending = false
		ui.message = "Jump dadi annullato."
		ui.refreshStatus()
		return nil
	}

	if ui.detailGPending {
		ui.detailGPending = false
		if ev.Key() == tcell.KeyRune && ev.Rune() == 'g' {
			ui.detailCursorLine = 0
			ui.renderDetail()
			ui.detail.ScrollTo(0, 0)
			ui.message = "Prima riga."
			ui.refreshStatus()
		} else {
			ui.message = ""
			ui.refreshStatus()
			return ev
		}
		return nil
	}

	if ui.listGotoPending {
		if ui.listGotoMulti {
			switch ev.Key() {
			case tcell.KeyEscape:
				ui.listGotoPending = false
				ui.listGotoMulti = false
				ui.message = "Jump lista annullato."
				ui.refreshStatus()
				return nil
			case tcell.KeyEnter:
				ui.listGotoPending = false
				ui.listGotoMulti = false
				n, err := strconv.Atoi(ui.listGotoAccum)
				if err == nil && n >= 1 {
					ui.jumpList(ui.listGotoTarget, n-1)
				} else {
					ui.message = fmt.Sprintf("Riga non valida: %q", ui.listGotoAccum)
					ui.refreshStatus()
				}
				return nil
			default:
				if ev.Key() == tcell.KeyRune && ev.Rune() >= '0' && ev.Rune() <= '9' {
					ui.listGotoAccum += string(ev.Rune())
					ui.message = fmt.Sprintf("Jump multi: %s (Invio per confermare)", ui.listGotoAccum)
					ui.refreshStatus()
					return nil
				}
				ui.listGotoPending = false
				ui.listGotoMulti = false
				ui.message = "Jump lista annullato."
				ui.refreshStatus()
				return nil
			}
		} else {
			ui.listGotoPending = false
			if ev.Key() == tcell.KeyEscape {
				ui.message = "Jump lista annullato."
				ui.refreshStatus()
				return nil
			}
			if ev.Key() == tcell.KeyRune {
				r := ev.Rune()
				switch r {
				case '^':
					ui.jumpList(ui.listGotoTarget, 0)
					return nil
				case '$':
					n := 0
					if ui.listGotoTarget != nil {
						n = ui.listGotoTarget.GetItemCount() - 1
					}
					if n < 0 {
						n = 0
					}
					ui.jumpList(ui.listGotoTarget, n)
					return nil
				case '\'':
					ui.listGotoPending = true
					ui.listGotoMulti = true
					ui.listGotoAccum = ""
					ui.message = "Jump multi: digita numero, Invio per confermare, Esc annulla."
					ui.refreshStatus()
					return nil
				default:
					if r >= '1' && r <= '9' {
						ui.jumpList(ui.listGotoTarget, int(r-'0')-1)
						return nil
					}
					ui.message = fmt.Sprintf("Jump lista: tasto non valido: %q", string(r))
					ui.refreshStatus()
					return nil
				}
			}
			ui.message = "Jump lista annullato."
			ui.refreshStatus()
			return nil
		}
	}

	if focusIsInput && ev.Key() == tcell.KeyEsc {
		ui.focusPanel(ui.activeCatalogListFocus())
		ui.refreshStatus()
		return nil
	}

	if ev.Key() == tcell.KeyEsc {
		if drop, ok := focus.(*tview.DropDown); ok && !drop.IsOpen() {
			ui.focusPanel(ui.activeCatalogListFocus())
			ui.refreshStatus()
			return nil
		}
	}

	// On source filters, let Space toggle without forcing the user to close the menu.
	if ev.Key() == tcell.KeyRune && ev.Rune() == ' ' {
		var openDrop *tview.DropDown
		switch {
		case ui.monSourceDrop != nil && ui.monSourceDrop.IsOpen():
			openDrop = ui.monSourceDrop
		case ui.eqSourceDrop != nil && ui.eqSourceDrop.IsOpen():
			openDrop = ui.eqSourceDrop
		case ui.classSourceDrop != nil && ui.classSourceDrop.IsOpen():
			openDrop = ui.classSourceDrop
		}
		if openDrop != nil {
			ui.sourceSpaceToggleActive = true
			if h := openDrop.InputHandler(); h != nil {
				h(tcell.NewEventKey(tcell.KeyEnter, 0, ev.Modifiers()), func(p tview.Primitive) {
					ui.app.SetFocus(p)
				})
			}
			ui.sourceSpaceToggleActive = false
			ui.openDropDown(openDrop)
			return nil
		}
		if focus == ui.monSourceDrop || focus == ui.eqSourceDrop || focus == ui.classSourceDrop {
			return tcell.NewEventKey(tcell.KeyEnter, 0, ev.Modifiers())
		}
	}

	// While a source dropdown is open: A selects all, N deselects all.
	if ev.Key() == tcell.KeyRune {
		var openDrop *tview.DropDown
		switch {
		case ui.monSourceDrop != nil && ui.monSourceDrop.IsOpen():
			openDrop = ui.monSourceDrop
		case ui.eqSourceDrop != nil && ui.eqSourceDrop.IsOpen():
			openDrop = ui.eqSourceDrop
		case ui.classSourceDrop != nil && ui.classSourceDrop.IsOpen():
			openDrop = ui.classSourceDrop
		}
		if openDrop != nil {
			switch ev.Rune() {
			case 'a', 'A':
				if openDrop == ui.monSourceDrop {
					ui.setMonsterSourceAll(true)
				} else if openDrop == ui.eqSourceDrop {
					ui.setEquipmentSourceAll(true)
				} else {
					ui.setClassSourceAll(true)
				}
				ui.openDropDown(openDrop)
				return nil
			case 'n', 'N':
				if openDrop == ui.monSourceDrop {
					ui.setMonsterSourceAll(false)
				} else if openDrop == ui.eqSourceDrop {
					ui.setEquipmentSourceAll(false)
				} else {
					ui.setClassSourceAll(false)
				}
				ui.openDropDown(openDrop)
				return nil
			}
		}
	}

	switch ev.Key() {
	case tcell.KeyCtrlC:
		ui.app.Stop()
		return nil
	case tcell.KeyCtrlS:
		if ui.campaignName != "" {
			ui.saveCampaignState(ui.campaignName)
		} else {
			ui.showSaveCampaignModal()
		}
		return nil
	case tcell.KeyCtrlO:
		ui.showCampaignManagerModal()
		return nil
	case tcell.KeyCtrlN:
		if !focusIsWidget {
			ui.openQuickNoteInput()
			return nil
		}
	case tcell.KeyCtrlT:
		if !focusIsWidget {
			ui.openUndoHistoryPanel()
			return nil
		}
	case tcell.KeyLeft:
		if focus == ui.pngList {
			ui.adjustPNGToken(-1)
			return nil
		}
	case tcell.KeyRight:
		if focus == ui.pngList {
			ui.adjustPNGToken(1)
			return nil
		}
	case tcell.KeyEnter:
		if focus == ui.detail {
			ui.rollDiceFromDetailCursorLine()
			return nil
		}
		if focus == ui.dice {
			ui.rerollSelectedDiceResult()
			return nil
		}
	case tcell.KeyCtrlD:
		if focus == ui.detail {
			ui.scrollDetailHalfPage(1)
			return nil
		}
		deck := buildInitiativeDeck()
		drawn := deck[rand.IntN(len(deck))]
		ui.message = fmt.Sprintf("Carta: %s", drawn)
		ui.refreshStatus()
		return nil
	case tcell.KeyCtrlU:
		if focus == ui.detail {
			ui.scrollDetailHalfPage(-1)
			return nil
		}
	case tcell.KeyTAB:
		ui.focusNext()
		return nil
	case tcell.KeyBacktab:
		ui.focusPrev()
		return nil
	case tcell.KeyPgUp:
		if focus == ui.detail || focus == ui.detailTreasure || focus == ui.dice || focus == ui.pngList || focus == ui.encList || focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.monSourceDrop || focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.eqSourceDrop || focus == ui.cardList || focus == ui.cardSearch || focus == ui.cardClassDrop || focus == ui.cardTypeDrop || focus == ui.classList || focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop || focus == ui.classSourceDrop {
			ui.scrollDetailByPage(-1)
			return nil
		}
	case tcell.KeyPgDn:
		if focus == ui.detail || focus == ui.detailTreasure || focus == ui.dice || focus == ui.pngList || focus == ui.encList || focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.monSourceDrop || focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.eqSourceDrop || focus == ui.cardList || focus == ui.cardSearch || focus == ui.cardClassDrop || focus == ui.cardTypeDrop || focus == ui.classList || focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop || focus == ui.classSourceDrop {
			ui.scrollDetailByPage(1)
			return nil
		}
	}

	// In grouped encounter view, block all per-entry operation keys.
	if focus == ui.encList && ui.encGrouped {
		switch ev.Rune() {
		case 'h', 'l', 'j', 'k', 'i', 'I', 'J', 'A', 'K', 'S', '*',
			'c', 'x', 'C', 'd', 'e', 'X', 'y', 'p', 't', 'n', '[', ']':
			return nil
		}
	}

	switch ev.Rune() {
	case '?':
		ui.openHelpOverlay(focus)
		return nil
	case 'f':
		if !focusIsInput {
			if focus == ui.detail {
				ui.scrollDetailHalfPage(1)
				return nil
			}
			ui.toggleFullscreenForFocus(focus)
			return nil
		}
	case 'q':
		ui.app.Stop()
		return nil
	case '0', '1', '2', '3', '4', '5', '6':
		if !focusIsWidget {
			ui.startPanelPrefix(int(ev.Rune() - '0'))
			return nil
		}
	case 'W':
		if !focusIsWidget {
			ui.focusPanel(focusDetail)
			return nil
		}
	case '#':
		if !focusIsWidget {
			ui.showLineNumbers = !ui.showLineNumbers
			if ui.showLineNumbers {
				ui.message = "Numeri di riga: ON."
			} else {
				ui.message = "Numeri di riga: OFF."
			}
			ui.refreshPNGs()
			ui.refreshMonsters()
			ui.refreshEncounter()
			ui.refreshStatus()
			return nil
		}
	case 'G':
		if !focusIsWidget {
			if focus == ui.detail {
				lines := strings.Split(ui.detailRaw, "\n")
				ui.detailCursorLine = len(lines) - 1
				if ui.detailCursorLine < 0 {
					ui.detailCursorLine = 0
				}
				ui.renderDetail()
				ui.detail.ScrollToEnd()
				ui.message = "Ultima riga."
				ui.refreshStatus()
				return nil
			}
			ui.openGotoModal()
			return nil
		}
	case '[':
		if focus == ui.encList {
			ui.adjustEncounterConditionRounds(-1)
			return nil
		}
		ui.switchCatalog(-1)
		return nil
	case ']':
		if focus == ui.encList {
			ui.adjustEncounterConditionRounds(1)
			return nil
		}
		ui.switchCatalog(1)
		return nil
	case '/':
		if !focusIsInput {
			ui.openRawSearch(focus)
			return nil
		}
	case 'c':
		if focusIsWidget {
			return ev
		}
		if focus == ui.dice {
			ui.clearDiceResults()
			return nil
		}
		if focus == ui.encList {
			ui.openEncounterConditionModal()
			return nil
		}
		if focus == ui.notesList || focus == ui.notesSearch {
			ui.openAddNoteModal()
			return nil
		}
		ui.openCreatePNGModal()
		return nil
	case 'x':
		if focusIsWidget {
			return ev
		}
		if focus == ui.encList {
			ui.openEncounterConditionRemoveModal()
			return nil
		}
		ui.openDeletePNGConfirm()
		return nil
	case 'C':
		if focus == ui.encList {
			ui.clearEncounterConditions()
			return nil
		}
	case 'm':
		if focus == ui.pngList {
			ui.openRenamePNGModal()
			return nil
		}
		if !focusIsWidget && focus == ui.dice {
			ui.openDiceMacroModal()
			return nil
		}
	case 'a':
		if focusIsInput {
			return ev
		}
		if focus == ui.dice {
			ui.openDiceRollInput()
			return nil
		}
		if focus == ui.pngList {
			ui.addSelectedPNGToEncounter()
			return nil
		}
		if ui.catalogMode == "mostri" && (focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.monSourceDrop) {
			ui.addSelectedMonsterToEncounter()
			return nil
		}
		if ui.catalogMode == "regole" && (focus == ui.classList || focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop || focus == ui.classSourceDrop) {
			ui.openClassPNGInput()
			return nil
		}
	case 'n':
		if focus == ui.encList && ui.encInitModeActive {
			ui.advanceEncounterInitiativeTurn()
			return nil
		}
		if focusIsInput {
			return ev
		}
		if ui.catalogMode == "mostri" && (focus == ui.monList || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.monSourceDrop) {
			ui.openRandomEncounterFromMonstersInput()
			return nil
		}
	case 'e':
		if focus == ui.pngList {
			ui.openEditPNGModal()
			return nil
		}
		if focus == ui.notesList {
			ui.openEditNoteModal()
			return nil
		}
		if focus == ui.encList {
			ui.openEncounterInitiativeEditModal()
			return nil
		}
		if focus == ui.dice {
			ui.openDiceReRollInput()
			return nil
		}
	case 'b':
		if focusIsWidget {
			return ev
		}
		if focus == ui.encList {
			ui.encGrouped = !ui.encGrouped
			if ui.encGrouped {
				ui.message = "Raggruppamento per nome: ON. (b = separa)"
			} else {
				ui.message = "Raggruppamento per nome: OFF."
			}
			ui.refreshEncounter()
			ui.refreshStatus()
			return nil
		}
		if focus == ui.detail {
			ui.scrollDetailHalfPage(-1)
			return nil
		}
		if focus == ui.pngList {
			ui.openPNGResourceModal()
			return nil
		}
		if ui.catalogMode == "equipaggiamento" && (focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.eqSourceDrop || focus == ui.detail || focus == ui.detailTreasure) {
			ui.openEquipmentTreasureInput()
			return nil
		}
	case 'T':
		if !focusIsWidget && focus == ui.encList {
			ui.message = "Numerazione già compatta (calcolata automaticamente)."
			ui.refreshStatus()
			return nil
		}
	case 'N':
		if !focusIsWidget && focus == ui.encList {
			ui.encLetterMode = !ui.encLetterMode
			if ui.encLetterMode {
				ui.message = "Nomenclatura lettere: ON. (N = torna a numeri)"
			} else {
				ui.message = "Nomenclatura lettere: OFF."
			}
			ui.refreshEncounter()
			ui.refreshStatus()
			return nil
		}
	case 'r':
		if focusIsWidget {
			return ev
		}
		if len(ui.redoStack) > 0 {
			ui.performRedo()
			return nil
		}
	case 'u':
		if focusIsWidget {
			return ev
		}
		if len(ui.undoStack) > 0 {
			ui.performUndo()
			return nil
		}
		if focus == ui.notesList {
			ui.app.SetFocus(ui.notesSearch)
			return nil
		}
		if ui.isMonsterPanelFocus(focus) {
			ui.focusPanel(focusMonSearch)
			return nil
		}
		if ui.isEquipmentPanelFocus(focus) {
			ui.focusPanel(focusEqSearch)
			return nil
		}
		if ui.isClassPanelFocus(focus) {
			ui.focusPanel(focusClassSearch)
			return nil
		}
	case 'g':
		if focusIsWidget {
			return ev
		}
		if focus == ui.detail {
			ui.detailGPending = true
			ui.message = "g — premi g per prima riga, Esc annulla."
			ui.refreshStatus()
			return nil
		}
		if focus == ui.dice {
			ui.diceGotoPending = true
			ui.message = "Jump dadi: premi 1-9, ^ (prima), $ (ultima)."
			ui.refreshStatus()
			return nil
		}
		if list := ui.focusedListWidget(focus); list != nil {
			ui.listGotoPending = true
			ui.listGotoTarget = list
			ui.listGotoMulti = false
			ui.listGotoAccum = ""
			ui.message = "Jump lista: 1-9, ' (multi-cifra), ^ (prima), $ (ultima), Esc annulla."
			ui.refreshStatus()
			return nil
		}
		if ui.isMonsterPanelFocus(focus) {
			ui.focusPanel(focusMonRank)
			return nil
		}
		if ui.isEquipmentPanelFocus(focus) {
			ui.focusPanel(focusEqRank)
			return nil
		}
		if ui.isClassPanelFocus(focus) {
			ui.focusPanel(focusClassSubclass)
			return nil
		}
	case 'y':
		if focusIsWidget {
			return ev
		}
		if focus == ui.pngList {
			ui.yankCurrentPNG()
			return nil
		}
		if focus == ui.encList {
			ui.yankCurrentEncounterEntry()
			return nil
		}
		if ui.isMonsterPanelFocus(focus) {
			ui.openDropDown(ui.monSourceDrop)
			return nil
		}
		if ui.isEquipmentPanelFocus(focus) {
			ui.openDropDown(ui.eqSourceDrop)
			return nil
		}
		if ui.isClassPanelFocus(focus) {
			ui.openDropDown(ui.classSourceDrop)
			return nil
		}
	case 'v':
		if focusIsWidget {
			return ev
		}
		if focus == ui.notesList || focus == ui.notesSearch {
			ui.notesSearch.SetText("")
			ui.refreshNotes()
			return nil
		}
		if ui.isMonsterPanelFocus(focus) {
			ui.resetMonsterFilters()
			return nil
		}
		if ui.isEquipmentPanelFocus(focus) {
			ui.resetEquipmentFilters()
			return nil
		}
		if ui.isClassPanelFocus(focus) {
			ui.resetClassFilters()
			return nil
		}
	case 'X':
		if focus == ui.encList {
			ui.toggleEncounterDisabled()
			return nil
		}
	case 'd':
		if focusIsWidget {
			return ev
		}
		if focus == ui.pngList {
			ui.deleteSelectedPNG()
			return nil
		}
		if focus == ui.dice {
			ui.deleteSelectedDiceResult()
			return nil
		}
		if focus == ui.encList {
			ui.openConfirmModal("Conferma", "Rimuovere il mostro selezionato dall'encounter?", func() {
				ui.removeSelectedEncounter()
			})
			return nil
		}
		if focus == ui.notesList {
			ui.openConfirmModal("Conferma", "Eliminare la nota selezionata?", func() {
				ui.deleteSelectedNote()
			})
			return nil
		}
		if ui.catalogMode == "equipaggiamento" && (focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.eqSourceDrop || focus == ui.detail || focus == ui.detailTreasure) {
			ui.toggleDetailsTreasureFocus()
			return nil
		}
	case 'h':
		if focus == ui.encList {
			ui.adjustEncounterWounds(1)
			return nil
		}
	case 'l':
		if focus == ui.encList {
			ui.adjustEncounterWounds(-1)
			return nil
		}
	case 'j':
		if focus == ui.detail {
			ui.moveDetailCursor(1)
			return nil
		}
		if focus == ui.encList {
			ui.adjustEncounterWounds(1)
			return nil
		}
	case 'k':
		if focus == ui.detail {
			ui.moveDetailCursor(-1)
			return nil
		}
		if focus == ui.encList {
			ui.adjustEncounterWounds(-1)
			return nil
		}
	case 'i':
		if focus == ui.encList {
			ui.rollEncounterInitiativeSelected()
			return nil
		}
	case 'I':
		if focus == ui.encList {
			ui.rollEncounterInitiativeAll()
			return nil
		}
	case 'J':
		if !focusIsWidget && (focus == ui.encList || focus == ui.pngList) {
			ui.dealInitiativeToAll()
			return nil
		}
	case 'M':
		if focus == ui.dice {
			ui.openMaxDiceLogInput()
			return nil
		}
	case 'A':
		if focus == ui.encList {
			ui.openEncounterAttackModal()
			return nil
		}
	case 'K':
		if focus == ui.encList {
			ui.openEncounterTraitModal()
			return nil
		}
	case '+':
		if focus == ui.pngList {
			ui.openPNGAdvancementModal()
			return nil
		}
	case 'S':
		if focus == ui.encList {
			ui.sortEncounterByInitiative()
			return nil
		}
	case '*':
		if focus == ui.encList {
			ui.enterEncounterInitiativeMode()
			return nil
		}
	case 't':
		if !focusIsWidget && focus == ui.encList {
			if ui.encList.GetItemCount() > 0 {
				ui.startTurnTimer()
			}
			return nil
		}
		if focusIsWidget {
			return ev
		}
		if ui.isMonsterPanelFocus(focus) {
			ui.focusPanel(focusMonRole)
			return nil
		}
		if ui.isEquipmentPanelFocus(focus) {
			ui.focusPanel(focusEqItemType)
			return nil
		}
		if ui.isClassPanelFocus(focus) {
			ui.focusPanel(focusClassName)
			return nil
		}
	case 'z':
		if focus == ui.encList && ui.encInitModeActive {
			_, _, _, h := ui.encList.GetInnerRect()
			ui.encList.SetCurrentItem(ui.encInitTurnIndex)
			offset := ui.encInitTurnIndex - h/2
			if offset < 0 {
				offset = 0
			}
			ui.encList.SetOffset(offset, 0)
			return nil
		}
	case 'o':
		if focus == ui.encList {
			ui.encounterShowConditionEffects = !ui.encounterShowConditionEffects
			if ui.encounterShowConditionEffects {
				ui.message = "Dettagli effetti condizioni: ON."
			} else {
				ui.message = "Dettagli effetti condizioni: OFF."
			}
			ui.refreshDetail()
			ui.refreshStatus()
			return nil
		}
	case 'p':
		if focusIsWidget {
			return ev
		}
		if focus == ui.pngList {
			ui.pasteClipPNG()
			return nil
		}
		if focus == ui.encList {
			ui.pasteClipEncounterEntry()
			return nil
		}
	}
	return ev
}

func (ui *tviewUI) isMonsterPanelFocus(focus tview.Primitive) bool {
	return focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.monSourceDrop || focus == ui.monList
}

func (ui *tviewUI) isEquipmentPanelFocus(focus tview.Primitive) bool {
	return focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.eqSourceDrop || focus == ui.eqList
}

func (ui *tviewUI) isClassPanelFocus(focus tview.Primitive) bool {
	return focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop || focus == ui.classSourceDrop || focus == ui.classList
}

func (ui *tviewUI) focusNext() {
	for i := 0; i < len(ui.focus); i++ {
		ui.focusIdx = (ui.focusIdx + 1) % len(ui.focus)
		if ui.isFocusVisible(ui.focusIdx) {
			ui.app.SetFocus(ui.focus[ui.focusIdx])
			break
		}
	}
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) focusPrev() {
	for i := 0; i < len(ui.focus); i++ {
		ui.focusIdx--
		if ui.focusIdx < 0 {
			ui.focusIdx = len(ui.focus) - 1
		}
		if ui.isFocusVisible(ui.focusIdx) {
			ui.app.SetFocus(ui.focus[ui.focusIdx])
			break
		}
	}
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) focusPanel(panel int) {
	if panel == focusMonRole && ui.catalogMode == "equipaggiamento" {
		panel = focusEqItemType
	}
	if panel == focusMonSearch && ui.catalogMode == "equipaggiamento" {
		panel = focusEqSearch
	}
	if panel == focusMonRank && ui.catalogMode == "equipaggiamento" {
		panel = focusEqRank
	}
	if panel == focusMonList && ui.catalogMode == "equipaggiamento" {
		panel = focusEqList
	}
	if panel == focusMonRole && ui.catalogMode == "carte" {
		panel = focusCardClass
	}
	if panel == focusMonSearch && ui.catalogMode == "carte" {
		panel = focusCardSearch
	}
	if panel == focusMonRank && ui.catalogMode == "carte" {
		panel = focusCardType
	}
	if panel == focusMonList && ui.catalogMode == "carte" {
		panel = focusCardList
	}
	if panel == focusMonRole && ui.catalogMode == "regole" {
		panel = focusClassName
	}
	if panel == focusMonSearch && ui.catalogMode == "regole" {
		panel = focusClassSearch
	}
	if panel == focusMonRank && ui.catalogMode == "regole" {
		panel = focusClassSubclass
	}
	if panel == focusMonList && ui.catalogMode == "regole" {
		panel = focusClassList
	}
	if panel < 0 || panel >= len(ui.focus) {
		return
	}
	if !ui.isFocusVisible(panel) {
		return
	}
	ui.focusIdx = panel
	ui.app.SetFocus(ui.focus[panel])
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) isFocusVisible(idx int) bool {
	switch idx {
	case focusMonSearch, focusMonRole, focusMonRank, focusMonSource, focusMonList:
		return ui.catalogMode == "mostri"
	case focusEqSearch, focusEqType, focusEqItemType, focusEqRank, focusEqSource, focusEqList:
		return ui.catalogMode == "equipaggiamento"
	case focusCardSearch, focusCardClass, focusCardType, focusCardList:
		return ui.catalogMode == "carte"
	case focusClassSearch, focusClassName, focusClassSubclass, focusClassSource, focusClassList:
		return ui.catalogMode == "regole"
	case focusNotesSearch, focusNotesList:
		return ui.catalogMode == "note"
	default:
		return true
	}
}

func (ui *tviewUI) activeCatalogListFocus() int {
	switch ui.catalogMode {
	case "equipaggiamento":
		return focusEqList
	case "regole":
		return focusClassList
	case "note":
		return focusNotesList
	}
	return focusMonList
}

func (ui *tviewUI) focusActiveCatalogList() {
	if len(ui.focus) == 0 {
		return
	}
	ui.focusPanel(ui.activeCatalogListFocus())
}

func (ui *tviewUI) openDropDown(drop *tview.DropDown) {
	if drop == nil {
		return
	}
	ui.app.SetFocus(drop)
	if h := drop.InputHandler(); h != nil {
		h(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) {
			ui.app.SetFocus(p)
		})
	}
}

func (ui *tviewUI) catalogLabel(mode string) string {
	switch mode {
	case "equipaggiamento":
		return "Equipaggiamento"
	case "regole":
		return "Regole"
	case "note":
		return "Note"
	default:
		return "Mostri"
	}
}

func (ui *tviewUI) activeMonsterFilterBadge() string {
	var parts []string
	if q := strings.TrimSpace(ui.search.GetText()); q != "" {
		parts = append(parts, "nome="+q)
	}
	if ui.roleFilter != "" && ui.roleFilter != "Tutti" {
		parts = append(parts, "ruolo="+ui.roleFilter)
	}
	if ui.rankFilter != "" && ui.rankFilter != "Tutti" {
		parts = append(parts, "rank="+ui.rankFilter)
	}
	if n := sourceSelectedCount(ui.monSourceValues, ui.monSourceSelected); n > 0 && n < len(ui.monSourceValues) {
		parts = append(parts, fmt.Sprintf("src=%d/%d", n, len(ui.monSourceValues)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[black:gold] " + strings.Join(parts, " | ") + " [-:-]"
}

func (ui *tviewUI) refreshCatalogTitles() {
	type entry struct {
		mode     string
		shortcut string
		panel    *tview.Flex
	}
	entries := []entry{
		{"mostri", "3", ui.monstersPanel},
		{"equipaggiamento", "4", ui.equipmentPanel},
		{"regole", "5", ui.classesPanel},
		{"note", "6", ui.notesPanel},
	}
	n := len(entries)
	for i, e := range entries {
		prev := entries[(i-1+n)%n]
		next := entries[(i+1)%n]
		title := fmt.Sprintf(" [%s] %s | '[' %s | ']' %s ", e.shortcut, ui.catalogLabel(e.mode), ui.catalogLabel(prev.mode), ui.catalogLabel(next.mode))
		if e.mode == "mostri" {
			if badge := ui.activeMonsterFilterBadge(); badge != "" {
				title = strings.TrimSuffix(title, " ") + "  " + badge + " "
			}
		}
		e.panel.SetTitle(title)
	}
}

func (ui *tviewUI) switchCatalog(delta int) {
	if delta == 0 {
		return
	}
	order := []string{"mostri", "equipaggiamento", "regole", "note"}
	cur := 0
	for i, name := range order {
		if name == ui.catalogMode {
			cur = i
			break
		}
	}
	nextIdx := (cur + delta) % len(order)
	if nextIdx < 0 {
		nextIdx += len(order)
	}
	next := order[nextIdx]
	ui.catalogMode = next
	ui.catalogPanel.SwitchToPage(next)
	ui.refreshCatalogTitles()
	switch next {
	case "equipaggiamento":
		ui.message = "Catalogo: Equipaggiamento"
	case "regole":
		ui.message = "Catalogo: Regole"
	case "note":
		ui.message = "Catalogo: Note"
	default:
		ui.message = "Catalogo: Mostri"
	}
	ui.focusPanel(ui.activeCatalogListFocus())
	ui.refreshStatus()
}

func (ui *tviewUI) refreshAll() {
	ui.refreshPNGs()
	ui.refreshMonsters()
	ui.refreshEquipment()
	ui.refreshClasses()
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) refreshPNGs() {
	current := ui.selected
	if current < 0 && len(ui.pngs) > 0 {
		current = 0
	}
	ui.pngList.Clear()
	if len(ui.pngs) == 0 {
		ui.pngList.AddItem("(nessun PNG)", "", 0, nil)
		return
	}
	for i, p := range ui.pngs {
		prefix := "  "
		if i == ui.selected {
			prefix = "* "
		}
		label := fmt.Sprintf("%s%s [T:%d]", prefix, p.Name, p.Token)
		if p.HasInit && p.InitiativeCard != "" {
			label += fmt.Sprintf(" [%s]", p.InitiativeCard)
		}
		if badge := pngResourcesBadge(p.Resources); badge != "" {
			label += " " + badge
		}
		if ui.showLineNumbers {
			label = fmt.Sprintf("%d. %s", i+1, label)
		}
		ui.pngList.AddItem(label, "", 0, nil)
	}
	if current >= len(ui.pngs) {
		current = len(ui.pngs) - 1
	}
	if current >= 0 {
		ui.pngList.SetCurrentItem(current)
		ui.selected = current
	}
}

func (ui *tviewUI) refreshMonsters() {
	query := strings.ToLower(strings.TrimSpace(ui.search.GetText()))
	ui.filtered = ui.filtered[:0]
	for i, m := range ui.monsters {
		if query != "" && !strings.Contains(strings.ToLower(m.Name), query) {
			continue
		}
		if ui.roleFilter != "" && ui.roleFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(m.Role), ui.roleFilter) {
			continue
		}
		if ui.rankFilter != "" && ui.rankFilter != "Tutti" && strconv.Itoa(m.Size) != ui.rankFilter {
			continue
		}
		if !sourceMatches(m.Source, ui.monSourceSelected) {
			continue
		}
		ui.filtered = append(ui.filtered, i)
	}

	// During initial build dropdown callbacks can fire before the list is created.
	if ui.monList == nil {
		return
	}

	current := ui.monList.GetCurrentItem()
	ui.monList.Clear()
	if len(ui.filtered) == 0 {
		ui.monList.AddItem("(nessun mostro)", "", 0, nil)
		ui.refreshCatalogTitles()
		return
	}
	for j, idx := range ui.filtered {
		m := ui.monsters[idx]
		wc := ""
		if m.WildCard {
			wc = " WC"
		}
		label := fmt.Sprintf("%s [S%d%s] Ferite:%d", m.Name, m.Size, wc, monsterWoundsCap(m))
		if ui.showLineNumbers {
			label = fmt.Sprintf("%d. %s", j+1, label)
		}
		ui.monList.AddItem(label, "", 0, nil)
	}
	if current >= len(ui.filtered) {
		current = len(ui.filtered) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.monList.SetCurrentItem(current)
	ui.refreshCatalogTitles()
}

func (ui *tviewUI) refreshEquipment() {
	query := strings.ToLower(strings.TrimSpace(ui.eqSearch.GetText()))
	ui.filteredEq = ui.filteredEq[:0]
	for i, it := range ui.equipment {
		if query != "" && !strings.Contains(strings.ToLower(it.Name), query) {
			continue
		}
		if ui.eqTypeFilter != "" && ui.eqTypeFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(it.Category), ui.eqTypeFilter) {
			continue
		}
		if ui.eqItemTypeFilter != "" && ui.eqItemTypeFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(it.Type), ui.eqItemTypeFilter) {
			continue
		}
		if ui.eqRankFilter != "" && ui.eqRankFilter != "Tutti" {
			era := strings.TrimSpace(it.Era)
			if era == "" && it.Rank > 0 {
				era = strconv.Itoa(it.Rank)
			}
			if !strings.EqualFold(era, ui.eqRankFilter) {
				continue
			}
		}
		if !sourceMatches(it.Source, ui.eqSourceSelected) {
			continue
		}
		ui.filteredEq = append(ui.filteredEq, i)
	}

	if ui.eqList == nil {
		return
	}
	current := ui.eqList.GetCurrentItem()
	ui.eqList.Clear()
	if len(ui.filteredEq) == 0 {
		ui.eqList.AddItem("(nessun equipaggiamento)", "", 0, nil)
		return
	}
	for _, idx := range ui.filteredEq {
		it := ui.equipment[idx]
		era := strings.TrimSpace(it.Era)
		if era == "" && it.Rank > 0 {
			era = strconv.Itoa(it.Rank)
		}
		if era != "" {
			ui.eqList.AddItem(fmt.Sprintf("%s [%s | %s]", it.Name, it.Category, era), "", 0, nil)
		} else {
			ui.eqList.AddItem(fmt.Sprintf("%s [%s]", it.Name, it.Category), "", 0, nil)
		}
	}
	if current >= len(ui.filteredEq) {
		current = len(ui.filteredEq) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.eqList.SetCurrentItem(current)
}

func (ui *tviewUI) refreshCards() {
	query := strings.ToLower(strings.TrimSpace(ui.cardSearch.GetText()))
	ui.filteredCards = ui.filteredCards[:0]
	for i, c := range ui.cards {
		if query != "" && !strings.Contains(strings.ToLower(c.Name), query) {
			continue
		}
		if ui.cardClassFilter != "" && ui.cardClassFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(c.Class), ui.cardClassFilter) {
			continue
		}
		if ui.cardTypeFilter != "" && ui.cardTypeFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(c.Type), ui.cardTypeFilter) {
			continue
		}
		ui.filteredCards = append(ui.filteredCards, i)
	}

	if ui.cardList == nil {
		return
	}
	current := ui.cardList.GetCurrentItem()
	ui.cardList.Clear()
	if len(ui.filteredCards) == 0 {
		ui.cardList.AddItem("(nessuna carta)", "", 0, nil)
		return
	}
	for _, idx := range ui.filteredCards {
		c := ui.cards[idx]
		head := cardDescriptionHead(c.Description)
		label := fmt.Sprintf("%s [%s - %s]", c.Name, c.Class, c.Type)
		if head != "" {
			label = fmt.Sprintf("%s | %s", head, label)
		}
		ui.cardList.AddItem(label, "", 0, nil)
	}
	if current >= len(ui.filteredCards) {
		current = len(ui.filteredCards) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.cardList.SetCurrentItem(current)
}

func (ui *tviewUI) refreshClasses() {
	query := strings.ToLower(strings.TrimSpace(ui.classSearch.GetText()))
	ui.filteredClasses = ui.filteredClasses[:0]
	for i, c := range ui.classes {
		if query != "" {
			text := strings.ToLower(strings.TrimSpace(c.Name) + " " + strings.TrimSpace(c.Subclass))
			if !strings.Contains(text, query) {
				continue
			}
		}
		if ui.classNameFilter != "" && ui.classNameFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(c.Name), ui.classNameFilter) {
			continue
		}
		if ui.classSubFilter != "" && ui.classSubFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(c.Subclass), ui.classSubFilter) {
			continue
		}
		if !sourceMatches(c.Source, ui.classSourceSelected) {
			continue
		}
		ui.filteredClasses = append(ui.filteredClasses, i)
	}

	if ui.classList == nil {
		return
	}
	current := ui.classList.GetCurrentItem()
	ui.classList.Clear()
	if len(ui.filteredClasses) == 0 {
		ui.classList.AddItem("(nessuna regola)", "", 0, nil)
		return
	}
	for _, idx := range ui.filteredClasses {
		c := ui.classes[idx]
		if c.Source == "carta" {
			itemType := strings.TrimSpace(c.Domains)
			if itemType == "" {
				itemType = "Regola"
			}
			ui.classList.AddItem(fmt.Sprintf("%s | %s [%s]", c.Subclass, c.Name, itemType), "", 0, nil)
		} else {
			ui.classList.AddItem(fmt.Sprintf("%s | %s", c.Subclass, c.Name), "", 0, nil)
		}
	}
	if current >= len(ui.filteredClasses) {
		current = len(ui.filteredClasses) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.classList.SetCurrentItem(current)
}

func cardDescriptionHead(desc string) string {
	return common.CardDescriptionHead(desc)
}

func (ui *tviewUI) refreshEncounter() {
	current := ui.encList.GetCurrentItem()
	ui.encList.Clear()
	if len(ui.encounter) == 0 {
		ui.encInitModeActive = false
		ui.encInitTurnIndex = 0
		ui.encInitRound = 1
		ui.encList.AddItem("(vuoto)", "", 0, nil)
		return
	}
	if ui.encGrouped {
		type group struct {
			name  string
			count int
		}
		var groups []group
		seen := make(map[string]int)
		for _, e := range ui.encounter {
			name := e.Monster.Name
			if idx, ok := seen[name]; ok {
				groups[idx].count++
			} else {
				seen[name] = len(groups)
				groups = append(groups, group{name: name, count: 1})
			}
		}
		for _, g := range groups {
			var label string
			if g.count == 1 {
				label = fmt.Sprintf("%s #1", g.name)
			} else {
				label = fmt.Sprintf("%s #1-%d", g.name, g.count)
			}
			ui.encList.AddItem(label, "", 0, nil)
		}
		ui.encList.SetCurrentItem(0)
		return
	}
	if ui.encInitTurnIndex < 0 || ui.encInitTurnIndex >= len(ui.encounter) {
		ui.encInitTurnIndex = 0
	}
	if ui.encInitRound < 1 {
		ui.encInitRound = 1
	}
	for i, e := range ui.encounter {
		base := encounterWoundsCap(e)
		label := ui.encounterLabelAt(i)
		if e.Disabled {
			label = "~ " + label
		}
		if badge := encounterConditionsBadge(e); badge != "" {
			label = badge + " " + label
		}
		if ui.encInitModeActive && i == ui.encInitTurnIndex {
			round := ui.encInitRound
			if round < 1 {
				round = 1
			}
			label = fmt.Sprintf("*[%d] %s", round, label)
		}
		remaining := base - e.Wounds
		if remaining < 0 {
			remaining = 0
		}
		initLabel := "--"
		if e.HasInit {
			initLabel = e.InitiativeCard
		}
		fullLabel := fmt.Sprintf("%s [Ini %s | Ferite %d/%d]", label, initLabel, remaining, base)
		if ui.showLineNumbers {
			fullLabel = fmt.Sprintf("%d. %s", i+1, fullLabel)
		}
		ui.encList.AddItem(fullLabel, "", 0, nil)
	}
	if current >= len(ui.encounter) {
		current = len(ui.encounter) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.encList.SetCurrentItem(current)
}

func (ui *tviewUI) refreshDetail() {
	if ui.detail == nil {
		return
	}
	ui.detailCursorLine = 0
	focus := ui.app.GetFocus()
	if focus == ui.detailTreasure {
		ui.renderTreasure()
		return
	}
	if focus == ui.dice {
		ui.detailRaw = ui.buildDiceDetail()
		ui.renderDetail()
		return
	}
	if focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.monSourceDrop {
		idx := ui.currentMonsterIndex()
		if idx < 0 {
			ui.detailRaw = "Nessun mostro selezionato."
			ui.renderDetail()
			return
		}
		ui.detailRaw = ui.buildMonsterDetails(ui.monsters[idx], ui.monsters[idx].Name, "")
		ui.renderDetail()
		return
	}
	if focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.eqSourceDrop {
		idx := ui.currentEquipmentIndex()
		if idx < 0 {
			ui.detailRaw = "Nessun equipaggiamento selezionato."
			ui.renderDetail()
			return
		}
		ui.detailRaw = ui.buildEquipmentDetails(ui.equipment[idx])
		ui.renderDetail()
		return
	}
	if focus == ui.classList || focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop || focus == ui.classSourceDrop {
		idx := ui.currentClassIndex()
		if idx < 0 {
			ui.detailRaw = "Nessuna regola selezionata."
			ui.renderDetail()
			return
		}
		ui.detailRaw = ui.buildClassDetails(ui.classes[idx])
		ui.renderDetail()
		return
	}
	if focus == ui.encList {
		idx := ui.currentEncounterIndex()
		if idx < 0 {
			ui.detailRaw = "Encounter vuoto."
			ui.renderDetail()
			return
		}
		e := ui.encounter[idx]
		base := encounterWoundsCap(e)
		remaining := base - e.Wounds
		if remaining < 0 {
			remaining = 0
		}
		initLabel := "--"
		if e.HasInit {
			initLabel = e.InitiativeCard
		}
		extra := fmt.Sprintf("Iniziativa: %s | Stato: %d/%d ferite residue (%s)", initLabel, remaining, base, encounterStateLabel(e))
		if cond := encounterConditionsLong(e); cond != "" {
			extra += " | Condizioni: " + cond
			if ui.encounterShowConditionEffects {
				if effects := encounterConditionEffectsLong(e); effects != "" {
					extra += "\nEffetti condizioni:\n" + effects
				}
			}
		}
		ui.detailRaw = ui.buildMonsterDetails(e.Monster, ui.encounterLabelAt(idx), extra)
		ui.renderDetail()
		return
	}
	if focus == ui.notesList || focus == ui.notesSearch {
		idx := ui.currentNoteIndex()
		if idx < 0 || idx >= len(ui.notes) {
			ui.detailRaw = "Nessuna nota selezionata."
			ui.renderDetail()
			return
		}
		ui.detailRaw = ui.notes[idx]
		ui.renderDetail()
		return
	}
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.detailRaw = "Nessun PNG selezionato."
		ui.renderDetail()
		return
	}
	p := ui.pngs[ui.selected]
	var b strings.Builder
	b.WriteString(p.Name)
	b.WriteString(fmt.Sprintf("\nToken: %d  (← decrementa, → incrementa)", p.Token))
	if p.HasInit && p.InitiativeCard != "" {
		b.WriteString(fmt.Sprintf("\nIniziativa: %s  (J per distribuire a tutti)", p.InitiativeCard))
	}
	if strings.TrimSpace(p.Class) != "" || strings.TrimSpace(p.Subclass) != "" || p.Level > 0 {
		classLine := ""
		if strings.TrimSpace(p.Subclass) != "" {
			classLine += strings.TrimSpace(p.Subclass)
		}
		if strings.TrimSpace(p.Class) != "" {
			if classLine != "" {
				classLine += " | "
			}
			classLine += strings.TrimSpace(p.Class)
		}
		if p.Level > 0 {
			if classLine != "" {
				classLine += " "
			}
			classLine += fmt.Sprintf("L%d", p.Level)
		}
		if classLine != "" {
			b.WriteString("\nClasse: " + classLine)
		}
	}
	if p.Level > 0 {
		rank := p.Rank
		if rank <= 0 {
			rank = rankFromLevel(p.Level)
		}
		compBonus := p.CompBonus
		expBonus := p.ExpBonus
		if compBonus == 0 && p.Level > 1 {
			compBonus = progressionBonusAtLevel(p.Level)
		}
		if expBonus == 0 && p.Level > 1 {
			expBonus = progressionBonusAtLevel(p.Level)
		}
		b.WriteString(fmt.Sprintf("\nLivello: %d | Rango: %d", p.Level, rank))
		b.WriteString(fmt.Sprintf("\nBonus Competenza (livello): +%d", compBonus))
		b.WriteString(fmt.Sprintf("\nEsperienze aggiuntive (livello): +%d", expBonus))
	}
	if def := ui.findClassDefinition(p.Class, p.Subclass); def != nil {
		if def.Evasion > 0 {
			b.WriteString(fmt.Sprintf("\nEvasione iniziale: %d", def.Evasion))
		}
		if def.HP > 0 {
			b.WriteString(fmt.Sprintf("\nPF iniziali: %d", def.HP))
		}
		if p.Level > 0 {
			b.WriteString("\nRegola soglie: aggiungi il livello attuale alle soglie base dell'armatura.")
		}
		if strings.TrimSpace(def.CasterTrait) != "" {
			b.WriteString("\nTratto da Incantatore: " + strings.TrimSpace(def.CasterTrait))
		}
		if strings.TrimSpace(def.HopePrivilege) != "" {
			b.WriteString("\n\nPrivilegio della Speranza:\n" + strings.TrimSpace(def.HopePrivilege))
		}
		if len(def.ClassPrivileges) > 0 {
			b.WriteString("\n\nPrivilegi di Classe:")
			for _, it := range def.ClassPrivileges {
				it = strings.TrimSpace(it)
				if it == "" {
					continue
				}
				b.WriteString("\n- " + it)
			}
		}
		if len(def.BasePrivileges) > 0 {
			b.WriteString("\n\nPrivilegi Base:")
			for _, it := range def.BasePrivileges {
				it = strings.TrimSpace(it)
				if it == "" {
					continue
				}
				b.WriteString("\n- " + it)
			}
		}
		if strings.TrimSpace(def.Specialization) != "" {
			b.WriteString("\n\nSpecializzazione:\n" + strings.TrimSpace(def.Specialization))
		}
		if strings.TrimSpace(def.Mastery) != "" {
			b.WriteString("\n\nMaestria:\n" + strings.TrimSpace(def.Mastery))
		}
	}
	if p.Agilita != "" || p.Intelligenza != "" || p.Spirito != "" || p.Forza != "" || p.Vigore != "" {
		b.WriteString(fmt.Sprintf("\n\nCaratteristiche: Agi %s | Int %s | Spi %s | For %s | Vig %s",
			pngAttrDie(&p, "Agilità"), pngAttrDie(&p, "Intelligenza"), pngAttrDie(&p, "Spirito"),
			pngAttrDie(&p, "Forza"), pngAttrDie(&p, "Vigore")))
	}
	if strings.TrimSpace(p.Traits) != "" {
		b.WriteString("\n\nTratti / Speciali:\n" + strings.TrimSpace(p.Traits))
	}
	if strings.TrimSpace(p.Primary) != "" {
		b.WriteString("\nArma primaria:\n" + strings.TrimSpace(p.Primary))
	}
	if strings.TrimSpace(p.Secondary) != "" {
		b.WriteString("\nArma secondaria:\n" + strings.TrimSpace(p.Secondary))
	}
	if strings.TrimSpace(p.Armor) != "" {
		b.WriteString("\nArmatura:\n" + strings.TrimSpace(p.Armor))
	}
	if strings.TrimSpace(p.Inventory) != "" {
		b.WriteString("\nInventario:\n" + strings.TrimSpace(p.Inventory))
	}
	if strings.TrimSpace(p.Look) != "" {
		b.WriteString("\nAspetto:\n" + strings.TrimSpace(p.Look))
	}
	if strings.TrimSpace(p.Description) != "" {
		b.WriteString("\n\nDescrizione:\n" + strings.TrimSpace(p.Description))
	}
	if len(p.Resources) > 0 {
		b.WriteString("\n\nRisorse:")
		for _, r := range p.Resources {
			b.WriteString(fmt.Sprintf("\n  %s: %d/%d", r.Name, r.Current, r.Max))
		}
		b.WriteString("\n  (premi 'b' per gestire le risorse)")
	}
	ui.detailRaw = b.String()
	ui.renderDetail()
}

func (ui *tviewUI) renderDetail() {
	if ui.detail == nil {
		return
	}
	text := ui.detailRaw
	if strings.TrimSpace(text) == "" {
		text = "Nessun dettaglio."
	}
	out := tview.Escape(text)
	out = strings.ReplaceAll(out, "\x01", "[::b]")
	out = strings.ReplaceAll(out, "\x02", "[::-]")
	lines := strings.Split(out, "\n")
	if len(lines) > 0 {
		lines[0] = "[yellow]" + lines[0] + "[-]"
	}
	// Show cursor when detail panel is focused
	if ui.app != nil && ui.app.GetFocus() == ui.detail {
		cursor := ui.detailCursorLine
		if cursor < 0 {
			cursor = 0
		}
		if cursor >= len(lines) {
			cursor = len(lines) - 1
		}
		if cursor >= 0 && len(lines) > 0 {
			lines[cursor] = "[::r]" + lines[cursor] + "[::-]"
		}
	}
	out = strings.Join(lines, "\n")
	if strings.TrimSpace(ui.detailQuery) != "" {
		out = highlightMatches(out, ui.detailQuery)
	}
	ui.detail.SetText(out)
}

func highlightMatches(text, query string) string {
	return common.HighlightMatches(text, query)
}

func (ui *tviewUI) refreshStatus() {
	focusLabel := "PNG"
	switch ui.app.GetFocus() {
	case ui.dice:
		focusLabel = "Dadi"
	case ui.encList:
		focusLabel = "Encounter"
	case ui.search:
		focusLabel = "Nome Mostri"
	case ui.roleDrop:
		focusLabel = "Ruolo Mostri"
	case ui.rankDrop:
		focusLabel = "Taglia Mostri"
	case ui.monSourceDrop:
		focusLabel = "Source Mostri"
	case ui.monList:
		focusLabel = "Mostri"
	case ui.eqSearch:
		focusLabel = "Nome Equip."
	case ui.eqTypeDrop:
		focusLabel = "Categoria Equip."
	case ui.eqItemTypeDrop:
		focusLabel = "Tipo Equip."
	case ui.eqRankDrop:
		focusLabel = "Rango Equip."
	case ui.eqSourceDrop:
		focusLabel = "Source Equip."
	case ui.eqList:
		focusLabel = "Equipaggiamento"
	case ui.cardSearch:
		focusLabel = "Nome Carte"
	case ui.cardClassDrop:
		focusLabel = "Classe Carte"
	case ui.cardTypeDrop:
		focusLabel = "Tipo Carte"
	case ui.cardList:
		focusLabel = "Carte"
	case ui.classSearch:
		focusLabel = "Cerca Regole"
	case ui.classNameDrop:
		focusLabel = "Categoria"
	case ui.classSubDrop:
		focusLabel = "Voce"
	case ui.classSourceDrop:
		focusLabel = "Source Regole"
	case ui.classList:
		focusLabel = "Regole"
	case ui.detailTreasure:
		focusLabel = "Treasure"
	case ui.detail:
		focusLabel = "Dettagli"
	}
	msg := ui.message
	if msg == "" {
		msg = "Pronto."
	}
	catalogLabel := "Mostri"
	if ui.catalogMode == "equipaggiamento" {
		catalogLabel = "Equipaggiamento"
	} else if ui.catalogMode == "regole" {
		catalogLabel = "Regole"
	}
	campPart := ""
	if ui.campaignName != "" {
		campPart = fmt.Sprintf("| campagna:[black:gold] %s [-:-] ", ui.campaignName)
	}
	statsPart := ""
	if stats := ui.contextStats(); stats != "" {
		statsPart = fmt.Sprintf("| %s ", stats)
	}
	ui.refreshPanelTitles()
	ui.status.SetText(fmt.Sprintf("focus:[black:gold] %s [-:-] | catalogo:[black:gold] %s [-:-] %s%s| %s", focusLabel, catalogLabel, campPart, statsPart, msg))
}

func (ui *tviewUI) contextStats() string {
	switch ui.focusIdx {
	case focusDice:
		return fmt.Sprintf("%d tiri", ui.dice.GetItemCount())
	case focusPNG:
		return fmt.Sprintf("%d PNG", len(ui.pngs))
	case focusEncounter:
		return fmt.Sprintf("%d in encounter", len(ui.encounter))
	case focusMonList, focusMonSearch, focusMonRole, focusMonRank, focusMonSource:
		total := len(ui.monsters)
		filt := len(ui.filtered)
		if filt == total {
			return fmt.Sprintf("%d mostri", total)
		}
		return fmt.Sprintf("%d/%d mostri", filt, total)
	case focusNotesList, focusNotesSearch:
		return fmt.Sprintf("%d note", len(ui.notes))
	}
	return ""
}

func (ui *tviewUI) refreshPanelTitles() {
	diceTitle := " [0]-Dadi "
	pngTitle := " [1]-PNG "
	encTitle := " [2]-Encounter "
	switch ui.focusIdx {
	case focusDice:
		diceTitle = " [0]-Dadi  Invio:lancia  a:aggiungi  d:rimuovi "
	case focusPNG:
		pngTitle = " [1]-PNG  a:nuovo  e:edita  J:init tutti  x:rimuovi "
	case focusEncounter:
		encTitle = " [2]-Encounter  a:aggiungi  J:init tutti  d:rimuovi  e:edita "
	}
	ui.dice.SetTitle(diceTitle)
	ui.pngList.SetTitle(pngTitle)
	ui.encList.SetTitle(encTitle)
}

func (ui *tviewUI) currentMonsterIndex() int {
	if len(ui.filtered) == 0 {
		return -1
	}
	if ui.monList == nil {
		return -1
	}
	cur := ui.monList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filtered) {
		return -1
	}
	return ui.filtered[cur]
}

func (ui *tviewUI) currentEquipmentIndex() int {
	if len(ui.filteredEq) == 0 || ui.eqList == nil {
		return -1
	}
	cur := ui.eqList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filteredEq) {
		return -1
	}
	return ui.filteredEq[cur]
}

func (ui *tviewUI) currentCardIndex() int {
	if len(ui.filteredCards) == 0 || ui.cardList == nil {
		return -1
	}
	cur := ui.cardList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filteredCards) {
		return -1
	}
	return ui.filteredCards[cur]
}

func (ui *tviewUI) currentClassIndex() int {
	if len(ui.filteredClasses) == 0 || ui.classList == nil {
		return -1
	}
	cur := ui.classList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filteredClasses) {
		return -1
	}
	return ui.filteredClasses[cur]
}

func (ui *tviewUI) buildMonsterFilterOptions() ([]string, []string) {
	roleSet := map[string]struct{}{}
	sizeSet := map[int]struct{}{}

	for _, m := range ui.monsters {
		role := strings.TrimSpace(m.Role)
		if role != "" {
			roleSet[role] = struct{}{}
		}
		if m.Size != 0 {
			sizeSet[m.Size] = struct{}{}
		}
	}

	roles := make([]string, 0, len(roleSet)+1)
	roles = append(roles, "Tutti")
	for role := range roleSet {
		roles = append(roles, role)
	}
	sort.Strings(roles[1:])

	sizesInt := make([]int, 0, len(sizeSet))
	for size := range sizeSet {
		sizesInt = append(sizesInt, size)
	}
	sort.Ints(sizesInt)

	ranks := make([]string, 0, len(sizesInt)+1)
	ranks = append(ranks, "Tutti")
	for _, size := range sizesInt {
		ranks = append(ranks, strconv.Itoa(size))
	}

	return roles, ranks
}

func normalizeSource(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "core"
	}
	return s
}

func buildSourceValues(values []string) []string {
	set := map[string]struct{}{}
	for _, v := range values {
		set[normalizeSource(v)] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func newSourceSelection(values []string) map[string]bool {
	sel := make(map[string]bool, len(values))
	for _, v := range values {
		sel[v] = true
	}
	return sel
}

func sourceMenuOptions(values []string, selected map[string]bool) []string {
	all := len(values) > 0
	none := true
	for _, v := range values {
		if selected[v] {
			none = false
		} else {
			all = false
		}
	}
	allMark := "[ ]"
	if all {
		allMark = "[x]"
	}
	noneMark := "[ ]"
	if none {
		noneMark = "[x]"
	}
	opts := []string{allMark + " Tutti", noneMark + " Nessuno"}
	for _, v := range values {
		if selected[v] {
			opts = append(opts, "[x] "+v)
		} else {
			opts = append(opts, "[ ] "+v)
		}
	}
	return opts
}

func sourceSelectedCount(values []string, selected map[string]bool) int {
	n := 0
	for _, v := range values {
		if selected[v] {
			n++
		}
	}
	return n
}

func sourceMatches(source string, selected map[string]bool) bool {
	if len(selected) == 0 {
		return false
	}
	return selected[normalizeSource(source)]
}

func (ui *tviewUI) buildEquipmentTypeOptions() []string {
	typeSet := map[string]struct{}{}
	for _, it := range ui.equipment {
		k := strings.TrimSpace(it.Category)
		if k != "" {
			typeSet[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(typeSet)+1)
	opts = append(opts, "Tutti")
	for k := range typeSet {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildMonsterSourceValues() []string {
	values := make([]string, 0, len(ui.monsters))
	for _, m := range ui.monsters {
		values = append(values, m.Source)
	}
	return buildSourceValues(values)
}

func (ui *tviewUI) buildEquipmentItemTypeOptions() []string {
	typeSet := map[string]struct{}{}
	for _, it := range ui.equipment {
		k := strings.TrimSpace(it.Type)
		if k != "" {
			typeSet[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(typeSet)+1)
	opts = append(opts, "Tutti")
	for k := range typeSet {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildEquipmentRankOptions() []string {
	set := map[string]struct{}{}
	for _, it := range ui.equipment {
		era := strings.TrimSpace(it.Era)
		if era == "" && it.Rank > 0 {
			era = strconv.Itoa(it.Rank)
		}
		if era != "" {
			set[era] = struct{}{}
		}
	}
	vals := make([]string, 0, len(set))
	for r := range set {
		vals = append(vals, r)
	}
	sort.Strings(vals)
	opts := make([]string, 0, len(vals)+1)
	opts = append(opts, "Tutti")
	opts = append(opts, vals...)
	return opts
}

func (ui *tviewUI) buildEquipmentSourceValues() []string {
	values := make([]string, 0, len(ui.equipment))
	for _, it := range ui.equipment {
		values = append(values, it.Source)
	}
	return buildSourceValues(values)
}

func (ui *tviewUI) buildCardClassOptions() []string {
	set := map[string]struct{}{}
	for _, c := range ui.cards {
		k := strings.TrimSpace(c.Class)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(set)+1)
	opts = append(opts, "Tutti")
	for k := range set {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildCardTypeOptions() []string {
	set := map[string]struct{}{}
	for _, c := range ui.cards {
		k := strings.TrimSpace(c.Type)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(set)+1)
	opts = append(opts, "Tutti")
	for k := range set {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildClassNameOptions() []string {
	set := map[string]struct{}{}
	for _, c := range ui.classes {
		k := strings.TrimSpace(c.Name)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(set)+1)
	opts = append(opts, "Tutti")
	for k := range set {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildClassSubclassOptions() []string {
	set := map[string]struct{}{}
	for _, c := range ui.classes {
		k := strings.TrimSpace(c.Subclass)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(set)+1)
	opts = append(opts, "Tutti")
	for k := range set {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildClassSourceValues() []string {
	values := make([]string, 0, len(ui.classes))
	for _, c := range ui.classes {
		values = append(values, c.Source)
	}
	return buildSourceValues(values)
}

func toggleSourceSelection(text string, values []string, selected map[string]bool) {
	raw := strings.TrimSpace(text)
	switch {
	case strings.Contains(strings.ToLower(raw), "tutti"):
		for _, v := range values {
			selected[v] = true
		}
		return
	case strings.Contains(strings.ToLower(raw), "nessuno"):
		for _, v := range values {
			selected[v] = false
		}
		return
	}
	if strings.HasPrefix(raw, "[x] ") || strings.HasPrefix(raw, "[ ] ") {
		raw = strings.TrimSpace(raw[4:])
	}
	raw = normalizeSource(raw)
	if _, ok := selected[raw]; ok {
		selected[raw] = !selected[raw]
	}
}

func setAllSourceSelection(values []string, selected map[string]bool, enabled bool) {
	for _, v := range values {
		selected[v] = enabled
	}
}

func (ui *tviewUI) updateSourceDropLabels() {
	if ui.monSourceDrop != nil {
		ui.monSourceDrop.SetLabel(fmt.Sprintf(" (y) Source (%d/%d) ", sourceSelectedCount(ui.monSourceValues, ui.monSourceSelected), len(ui.monSourceValues)))
	}
	if ui.eqSourceDrop != nil {
		ui.eqSourceDrop.SetLabel(fmt.Sprintf(" (y) Source (%d/%d) ", sourceSelectedCount(ui.eqSourceValues, ui.eqSourceSelected), len(ui.eqSourceValues)))
	}
	if ui.classSourceDrop != nil {
		ui.classSourceDrop.SetLabel(fmt.Sprintf(" (y) Source (%d/%d) ", sourceSelectedCount(ui.classSourceValues, ui.classSourceSelected), len(ui.classSourceValues)))
	}
}

func (ui *tviewUI) toggleMonsterSourceOption(text string, index int) {
	if ui.suppressMonSourceCallback {
		return
	}
	toggleSourceSelection(text, ui.monSourceValues, ui.monSourceSelected)
	ui.monSourceOpts = sourceMenuOptions(ui.monSourceValues, ui.monSourceSelected)
	ui.suppressMonSourceCallback = true
	ui.monSourceDrop.SetOptions(ui.monSourceOpts, func(t string, i int) { ui.toggleMonsterSourceOption(t, i) })
	if index < 0 || index >= len(ui.monSourceOpts) {
		index = 0
	}
	ui.monSourceDrop.SetCurrentOption(index)
	ui.suppressMonSourceCallback = false
	ui.updateSourceDropLabels()
	ui.refreshMonsters()
	ui.refreshDetail()
	if !ui.sourceSpaceToggleActive {
		ui.focusActiveCatalogList()
	}
}

func (ui *tviewUI) toggleEquipmentSourceOption(text string, index int) {
	if ui.suppressEqSourceCallback {
		return
	}
	toggleSourceSelection(text, ui.eqSourceValues, ui.eqSourceSelected)
	ui.eqSourceOpts = sourceMenuOptions(ui.eqSourceValues, ui.eqSourceSelected)
	ui.suppressEqSourceCallback = true
	ui.eqSourceDrop.SetOptions(ui.eqSourceOpts, func(t string, i int) { ui.toggleEquipmentSourceOption(t, i) })
	if index < 0 || index >= len(ui.eqSourceOpts) {
		index = 0
	}
	ui.eqSourceDrop.SetCurrentOption(index)
	ui.suppressEqSourceCallback = false
	ui.updateSourceDropLabels()
	ui.refreshEquipment()
	ui.refreshDetail()
	if !ui.sourceSpaceToggleActive {
		ui.focusActiveCatalogList()
	}
}

func (ui *tviewUI) toggleClassSourceOption(text string, index int) {
	if ui.suppressClassSourceCallback {
		return
	}
	toggleSourceSelection(text, ui.classSourceValues, ui.classSourceSelected)
	ui.classSourceOpts = sourceMenuOptions(ui.classSourceValues, ui.classSourceSelected)
	ui.suppressClassSourceCallback = true
	ui.classSourceDrop.SetOptions(ui.classSourceOpts, func(t string, i int) { ui.toggleClassSourceOption(t, i) })
	if index < 0 || index >= len(ui.classSourceOpts) {
		index = 0
	}
	ui.classSourceDrop.SetCurrentOption(index)
	ui.suppressClassSourceCallback = false
	ui.updateSourceDropLabels()
	ui.refreshClasses()
	ui.refreshDetail()
	if !ui.sourceSpaceToggleActive {
		ui.focusActiveCatalogList()
	}
}

func (ui *tviewUI) setMonsterSourceAll(enabled bool) {
	setAllSourceSelection(ui.monSourceValues, ui.monSourceSelected, enabled)
	ui.monSourceOpts = sourceMenuOptions(ui.monSourceValues, ui.monSourceSelected)
	ui.suppressMonSourceCallback = true
	ui.monSourceDrop.SetOptions(ui.monSourceOpts, func(t string, i int) { ui.toggleMonsterSourceOption(t, i) })
	ui.monSourceDrop.SetCurrentOption(0)
	ui.suppressMonSourceCallback = false
	ui.updateSourceDropLabels()
	ui.refreshMonsters()
	ui.refreshDetail()
}

func (ui *tviewUI) setEquipmentSourceAll(enabled bool) {
	setAllSourceSelection(ui.eqSourceValues, ui.eqSourceSelected, enabled)
	ui.eqSourceOpts = sourceMenuOptions(ui.eqSourceValues, ui.eqSourceSelected)
	ui.suppressEqSourceCallback = true
	ui.eqSourceDrop.SetOptions(ui.eqSourceOpts, func(t string, i int) { ui.toggleEquipmentSourceOption(t, i) })
	ui.eqSourceDrop.SetCurrentOption(0)
	ui.suppressEqSourceCallback = false
	ui.updateSourceDropLabels()
	ui.refreshEquipment()
	ui.refreshDetail()
}

func (ui *tviewUI) setClassSourceAll(enabled bool) {
	setAllSourceSelection(ui.classSourceValues, ui.classSourceSelected, enabled)
	ui.classSourceOpts = sourceMenuOptions(ui.classSourceValues, ui.classSourceSelected)
	ui.suppressClassSourceCallback = true
	ui.classSourceDrop.SetOptions(ui.classSourceOpts, func(t string, i int) { ui.toggleClassSourceOption(t, i) })
	ui.classSourceDrop.SetCurrentOption(0)
	ui.suppressClassSourceCallback = false
	ui.updateSourceDropLabels()
	ui.refreshClasses()
	ui.refreshDetail()
}

func (ui *tviewUI) resetMonsterFilters() {
	ui.roleFilter = "Tutti"
	ui.rankFilter = "Tutti"
	ui.monSourceSelected = newSourceSelection(ui.monSourceValues)
	ui.monSourceOpts = sourceMenuOptions(ui.monSourceValues, ui.monSourceSelected)
	ui.search.SetText("")
	if ui.roleDrop != nil {
		ui.roleDrop.SetCurrentOption(0)
	}
	if ui.rankDrop != nil {
		ui.rankDrop.SetCurrentOption(0)
	}
	if ui.monSourceDrop != nil {
		ui.monSourceDrop.SetOptions(ui.monSourceOpts, func(text string, index int) { ui.toggleMonsterSourceOption(text, index) })
		ui.suppressMonSourceCallback = true
		ui.monSourceDrop.SetCurrentOption(0)
		ui.suppressMonSourceCallback = false
	}
	ui.updateSourceDropLabels()
	ui.refreshMonsters()
	ui.refreshDetail()
	ui.message = "Filtri Mostri resettati."
	ui.refreshStatus()
}

func (ui *tviewUI) resetEquipmentFilters() {
	ui.eqTypeFilter = "Tutti"
	ui.eqItemTypeFilter = "Tutti"
	ui.eqRankFilter = "Tutti"
	ui.eqSourceSelected = newSourceSelection(ui.eqSourceValues)
	ui.eqSourceOpts = sourceMenuOptions(ui.eqSourceValues, ui.eqSourceSelected)
	ui.eqSearch.SetText("")
	if ui.eqTypeDrop != nil {
		ui.eqTypeDrop.SetCurrentOption(0)
	}
	if ui.eqItemTypeDrop != nil {
		ui.eqItemTypeDrop.SetCurrentOption(0)
	}
	if ui.eqRankDrop != nil {
		ui.eqRankDrop.SetCurrentOption(0)
	}
	if ui.eqSourceDrop != nil {
		ui.eqSourceDrop.SetOptions(ui.eqSourceOpts, func(text string, index int) { ui.toggleEquipmentSourceOption(text, index) })
		ui.suppressEqSourceCallback = true
		ui.eqSourceDrop.SetCurrentOption(0)
		ui.suppressEqSourceCallback = false
	}
	ui.updateSourceDropLabels()
	ui.refreshEquipment()
	ui.refreshDetail()
	ui.message = "Filtri Equipaggiamento resettati."
	ui.refreshStatus()
}

func (ui *tviewUI) resetCardFilters() {
	ui.cardClassFilter = "Tutti"
	ui.cardTypeFilter = "Tutti"
	ui.cardSearch.SetText("")
	if ui.cardClassDrop != nil {
		ui.cardClassDrop.SetCurrentOption(0)
	}
	if ui.cardTypeDrop != nil {
		ui.cardTypeDrop.SetCurrentOption(0)
	}
	ui.refreshCards()
	ui.refreshDetail()
	ui.message = "Filtri Carte resettati."
	ui.refreshStatus()
}

func (ui *tviewUI) resetClassFilters() {
	ui.classNameFilter = "Tutti"
	ui.classSubFilter = "Tutti"
	ui.classSourceSelected = newSourceSelection(ui.classSourceValues)
	ui.classSourceOpts = sourceMenuOptions(ui.classSourceValues, ui.classSourceSelected)
	ui.classSearch.SetText("")
	if ui.classNameDrop != nil {
		ui.classNameDrop.SetCurrentOption(0)
	}
	if ui.classSubDrop != nil {
		ui.classSubDrop.SetCurrentOption(0)
	}
	if ui.classSourceDrop != nil {
		ui.classSourceDrop.SetOptions(ui.classSourceOpts, func(text string, index int) { ui.toggleClassSourceOption(text, index) })
		ui.suppressClassSourceCallback = true
		ui.classSourceDrop.SetCurrentOption(0)
		ui.suppressClassSourceCallback = false
	}
	ui.updateSourceDropLabels()
	ui.refreshClasses()
	ui.refreshDetail()
	ui.message = "Filtri Regole resettati."
	ui.refreshStatus()
}

func (ui *tviewUI) buildEquipmentDetails(it EquipmentItem) string {
	hasValue := func(v string) bool {
		s := strings.TrimSpace(v)
		return s != "" && s != "—" && s != "-"
	}

	var b strings.Builder
	b.WriteString(it.Name + "\n")
	era := strings.TrimSpace(it.Era)
	if era == "" && it.Rank > 0 {
		era = strconv.Itoa(it.Rank)
	}
	b.WriteString(fmt.Sprintf("Categoria: %s | Tipo: %s", strings.TrimSpace(it.Category), strings.TrimSpace(it.Type)))
	if era != "" {
		b.WriteString(" | Era: " + era)
	}
	if src := strings.TrimSpace(it.Source); src != "" {
		b.WriteString(" | Source: " + src)
	}
	b.WriteString("\n")

	if hasValue(it.Cost) {
		b.WriteString("Costo: " + strings.TrimSpace(it.Cost) + "\n")
	}
	if hasValue(it.Weight) {
		b.WriteString("Peso: " + strings.TrimSpace(it.Weight) + "\n")
	}
	if hasValue(it.MinStrength) {
		b.WriteString("Forza Minima: " + strings.TrimSpace(it.MinStrength) + "\n")
	} else if hasValue(it.Trait) {
		b.WriteString("Forza Minima: " + strings.TrimSpace(it.Trait) + "\n")
	}
	if hasValue(it.Range) {
		b.WriteString("Gittata: " + strings.TrimSpace(it.Range) + "\n")
	}
	if hasValue(it.Damage) {
		b.WriteString("Danni: " + strings.TrimSpace(it.Damage) + "\n")
	}
	if hasValue(it.AP) {
		b.WriteString("PA: " + strings.TrimSpace(it.AP) + "\n")
	}
	if hasValue(it.ROF) {
		b.WriteString("CdT: " + strings.TrimSpace(it.ROF) + "\n")
	}
	if hasValue(it.Shots) {
		b.WriteString("Colpi: " + strings.TrimSpace(it.Shots) + "\n")
	}
	if hasValue(it.Armor) {
		b.WriteString("Armatura: " + strings.TrimSpace(it.Armor) + "\n")
	}
	if hasValue(it.Parry) {
		b.WriteString("Parata: " + strings.TrimSpace(it.Parry) + "\n")
	}
	if hasValue(it.Cover) {
		b.WriteString("Copertura: " + strings.TrimSpace(it.Cover) + "\n")
	}

	notes := strings.TrimSpace(it.Note)
	if notes == "" {
		notes = strings.TrimSpace(it.Characteristic)
	}
	if hasValue(notes) {
		b.WriteString("\nNote:\n" + notes)
	}
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) buildCardDetails(c CardItem) string {
	var b strings.Builder
	b.WriteString(c.Name + "\n")
	b.WriteString(fmt.Sprintf("Classe: %s | Tipo: %s\n", strings.TrimSpace(c.Class), strings.TrimSpace(c.Type)))
	if strings.TrimSpace(c.CasterTrait) != "" {
		b.WriteString("Tratto da Incantatore: " + strings.TrimSpace(c.CasterTrait) + "\n")
	}
	if strings.TrimSpace(c.Description) != "" {
		b.WriteString("\n" + strings.TrimSpace(c.Description) + "\n")
	}
	if len(c.Effects) > 0 {
		b.WriteString("\nEffetti:\n")
		for _, e := range c.Effects {
			if strings.TrimSpace(e) == "" {
				continue
			}
			b.WriteString("- " + strings.TrimSpace(e) + "\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func isSWADERuleEntry(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(n, "razza") ||
		strings.HasPrefix(n, "svantaggio") ||
		strings.HasPrefix(n, "vantaggio") ||
		strings.HasPrefix(n, "attributo") ||
		strings.HasPrefix(n, "abilit") ||
		strings.HasPrefix(n, "tabella") ||
		strings.HasPrefix(n, "regola") ||
		strings.HasPrefix(n, "meccanica")
}

func (ui *tviewUI) buildClassDetails(c ClassItem) string {
	if c.Source == "swade" || isSWADERuleEntry(c.Name) {
		var b strings.Builder
		b.WriteString(c.Subclass + "\n")
		b.WriteString("Tipo: " + c.Name + "\n")
		if strings.TrimSpace(c.Domains) != "" {
			b.WriteString("Requisiti: " + strings.TrimSpace(c.Domains) + "\n")
		}
		if strings.TrimSpace(c.Description) != "" {
			b.WriteString("\n" + strings.TrimSpace(c.Description) + "\n")
		}
		if len(c.ClassPrivileges) > 0 {
			n := strings.ToLower(strings.TrimSpace(c.Name))
			label := "Capacità"
			switch {
			case strings.HasPrefix(n, "razza"):
				label = "Capacità Razziali"
			case strings.HasPrefix(n, "tabella"):
				label = "Voci"
			case strings.HasPrefix(n, "regola"), strings.HasPrefix(n, "meccanica"):
				label = "Regole"
			}
			b.WriteString("\n" + label + ":\n")
			for _, p := range c.ClassPrivileges {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				b.WriteString("- " + p + "\n")
			}
		}
		return strings.TrimSpace(b.String())
	}
	if c.Source == "carta" {
		var b strings.Builder
		b.WriteString(c.Subclass + "\n")
		b.WriteString(fmt.Sprintf("Categoria: %s", strings.TrimSpace(c.Name)))
		if strings.TrimSpace(c.Source) != "" {
			b.WriteString(" | Source: " + strings.TrimSpace(c.Source))
		}
		if strings.TrimSpace(c.Domains) != "" {
			b.WriteString(" | Tipo: " + strings.TrimSpace(c.Domains))
		}
		b.WriteString("\n")
		if strings.TrimSpace(c.CasterTrait) != "" {
			b.WriteString("Tratto: " + strings.TrimSpace(c.CasterTrait) + "\n")
		}
		if strings.TrimSpace(c.Description) != "" {
			b.WriteString("\n" + strings.TrimSpace(c.Description) + "\n")
		}
		if len(c.BasePrivileges) > 0 {
			b.WriteString("\nEffetti:\n")
			for _, p := range c.BasePrivileges {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				b.WriteString("- " + p + "\n")
			}
		}
		return strings.TrimSpace(b.String())
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s - %s\n", c.Name, c.Subclass))
	if strings.TrimSpace(c.Source) != "" {
		b.WriteString("Source: " + strings.TrimSpace(c.Source) + "\n")
	}
	b.WriteString(fmt.Sprintf("Rango: %d\n", c.Rank))
	if strings.TrimSpace(c.Domains) != "" {
		b.WriteString("Domini: " + strings.TrimSpace(c.Domains) + "\n")
	}
	if c.Evasion > 0 {
		b.WriteString(fmt.Sprintf("Evasione iniziale: %d\n", c.Evasion))
	}
	if c.HP > 0 {
		b.WriteString(fmt.Sprintf("Punti Ferita iniziali: %d\n", c.HP))
	}
	if strings.TrimSpace(c.ClassItem) != "" {
		b.WriteString("Oggetti di classe: " + strings.TrimSpace(c.ClassItem) + "\n")
	}
	if strings.TrimSpace(c.HopePrivilege) != "" {
		b.WriteString("\nPrivilegio della Speranza:\n" + strings.TrimSpace(c.HopePrivilege) + "\n")
	}
	if len(c.ClassPrivileges) > 0 {
		b.WriteString("\nPrivilegi di Classe:\n")
		for _, p := range c.ClassPrivileges {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			b.WriteString("- " + p + "\n")
		}
	}
	if strings.TrimSpace(c.Description) != "" {
		b.WriteString("\nDescrizione:\n" + strings.TrimSpace(c.Description) + "\n")
	}
	if strings.TrimSpace(c.CasterTrait) != "" {
		b.WriteString("\nTratto da Incantatore:\n" + strings.TrimSpace(c.CasterTrait) + "\n")
	}
	if len(c.BasePrivileges) > 0 {
		b.WriteString("\nPrivilegi Base:\n")
		for _, p := range c.BasePrivileges {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			b.WriteString("- " + p + "\n")
		}
	}
	if strings.TrimSpace(c.Specialization) != "" {
		b.WriteString("\nSpecializzazione:\n" + strings.TrimSpace(c.Specialization) + "\n")
	}
	if strings.TrimSpace(c.Mastery) != "" {
		b.WriteString("\nMaestria:\n" + strings.TrimSpace(c.Mastery) + "\n")
	}
	if len(c.BackgroundQs) > 0 {
		b.WriteString("\nDomande sul Background:\n")
		for _, q := range c.BackgroundQs {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			b.WriteString("- " + q + "\n")
		}
	}
	if len(c.Bonds) > 0 {
		b.WriteString("\nLegami:\n")
		for _, q := range c.Bonds {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			b.WriteString("- " + q + "\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) currentEncounterIndex() int {
	if len(ui.encounter) == 0 {
		return -1
	}
	cur := ui.encList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.encounter) {
		return -1
	}
	return cur
}

func (ui *tviewUI) buildMonsterDetails(m Monster, title string, extraLine string) string {
	var b strings.Builder
	b.WriteString(title + "\n")
	wc := "No"
	if m.WildCard {
		wc = "Si"
	}
	pace := strings.TrimSpace(m.Pace)
	if pace == "" {
		pace = "-"
	}
	b.WriteString(fmt.Sprintf("Ruolo: %s | Taglia: %d | Wild Card: %s | Passo: %s\n", m.Role, m.Size, wc, pace))
	if src := strings.TrimSpace(m.Source); src != "" {
		b.WriteString("Source: " + src + "\n")
	}
	if extraLine != "" {
		b.WriteString(extraLine + "\n")
	}
	parry := strings.TrimSpace(m.Parry)
	if parry == "" {
		parry = "-"
	}
	toughness := strings.TrimSpace(m.Toughness)
	if toughness == "" {
		toughness = "-"
	}
	b.WriteString(fmt.Sprintf("Parata: %s | Robustezza: %s | Ferite max: %d\n", parry, toughness, monsterWoundsCap(m)))
	a := m.Attributes
	if a.Agilita != "" || a.Vigore != "" {
		orDash := func(s string) string {
			if strings.TrimSpace(s) == "" {
				return "-"
			}
			return s
		}
		b.WriteString(fmt.Sprintf("Agi %s  Int %s  Spi %s  For %s  Vig %s\n",
			orDash(a.Agilita), orDash(a.Intelligenza), orDash(a.Spirito), orDash(a.Forza), orDash(a.Vigore)))
	}
	if m.Attack.Name != "" {
		bonus := strings.TrimSpace(m.Attack.Bonus)
		bonus = strings.ReplaceAll(bonus, "−", "-")
		bonus = strings.ReplaceAll(bonus, "–", "-")
		if bonus != "" && !strings.HasPrefix(bonus, "+") && !strings.HasPrefix(bonus, "-") {
			bonus = "+" + bonus
		}
		if bonus != "" {
			b.WriteString(fmt.Sprintf("Attacco: %s (%s) %s %s (%s)\n", m.Attack.Name, m.Attack.Range, m.Attack.Damage, m.Attack.DamageType, bonus))
		} else {
			b.WriteString(fmt.Sprintf("Attacco: %s (%s) %s %s\n", m.Attack.Name, m.Attack.Range, m.Attack.Damage, m.Attack.DamageType))
		}
	}
	if strings.TrimSpace(m.MotivationsTactics) != "" {
		b.WriteString("\nMotivazioni/Tattiche:\n" + strings.TrimSpace(m.MotivationsTactics) + "\n")
	}
	if len(m.Skills) > 0 {
		b.WriteString("\nAbilita:\n")
		for _, sk := range m.Skills {
			sk = strings.TrimSpace(sk)
			if sk == "" {
				continue
			}
			b.WriteString("- " + sk + "\n")
		}
	}
	if len(m.Traits) > 0 {
		b.WriteString("\nTratti:\n")
		for _, t := range m.Traits {
			line := "- \x01" + t.Name + "\x02"
			if strings.TrimSpace(t.Kind) != "" {
				line += " (" + t.Kind + ")"
			}
			if strings.TrimSpace(t.Text) != "" {
				line += ": " + strings.TrimSpace(t.Text)
			}
			b.WriteString(line + "\n")
		}
	}
	if strings.TrimSpace(m.Description) != "" {
		b.WriteString("\n" + strings.TrimSpace(m.Description))
	}
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) encounterLabelAt(idx int) string {
	if idx < 0 || idx >= len(ui.encounter) {
		return ""
	}
	name := ui.encounter[idx].Monster.Name
	seen := 0
	for i := 0; i <= idx; i++ {
		if ui.encounter[i].Monster.Name == name {
			seen++
		}
	}
	if ui.encLetterMode {
		return fmt.Sprintf("%s #%s", name, common.IndexToLetter(seen))
	}
	return fmt.Sprintf("%s #%d", name, seen)
}

func monsterWoundsCap(m Monster) int {
	if m.WoundsMax > 0 {
		return m.WoundsMax
	}
	if m.PF > 0 {
		return m.PF
	}
	return 3
}

func encounterWoundsCap(e EncounterEntry) int {
	if e.WoundsMax > 0 {
		return e.WoundsMax
	}
	if e.BasePF > 0 {
		return e.BasePF
	}
	return monsterWoundsCap(e.Monster)
}

func encounterStateLabel(e EncounterEntry) string {
	max := encounterWoundsCap(e)
	if e.Wounds >= max {
		return "KO/Incapacitato"
	}
	if e.Wounds > 0 {
		return "Ferito"
	}
	return "Operativo"
}

func (ui *tviewUI) openCreatePNGModal() {
	input := tview.NewInputField().SetLabel(" Nome PNG ").SetFieldWidth(24)
	input.SetText(uniqueRandomPNGName(ui.pngs))
	input.SetBorder(true).SetTitle("Crea PNG")
	returnFocus := ui.app.GetFocus()

	nameTypeOpts := []string{"fantasy", "cyberpunk"}
	initIdx := 0
	for i, o := range nameTypeOpts {
		if o == currentNameType {
			initIdx = i
			break
		}
	}
	styleBar := tview.NewDropDown().SetLabel(" Stile nomi ")
	styleBar.SetFieldBackgroundColor(tcell.ColorBlack)
	styleBar.SetFieldTextColor(tcell.ColorWhite)
	styleBar.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	styleBar.SetOptions(nameTypeOpts, func(text string, _ int) {
		currentNameType = text
	})
	styleBar.SetCurrentOption(initIdx)
	styleBar.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyEnter, tcell.KeyBacktab:
			ui.app.SetFocus(input)
			return nil
		}
		return event
	})

	inner := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(styleBar, 1, 0, false).
		AddItem(input, 5, 0, true)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(inner, 44, 0, true).
			AddItem(nil, 0, 1, false), 6, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalVisible = true
	ui.modalName = "create_png"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(input)

	ng.BindRandomNameInput(input, func() string { return uniqueRandomPNGName(ui.pngs) })
	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEsc:
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		case tcell.KeyTab, tcell.KeyBacktab:
			ui.app.SetFocus(styleBar)
			return
		}
		name := strings.TrimSpace(input.GetText())
		if name == "" {
			name = uniqueRandomPNGName(ui.pngs)
		}
		for _, p := range ui.pngs {
			if strings.EqualFold(p.Name, name) {
				ui.message = "Nome già esistente."
				ui.refreshStatus()
				return
			}
		}
		ui.pushUndo()
		ui.pngs = append(ui.pngs, PNG{Name: name})
		ui.selected = len(ui.pngs) - 1
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(focusPNG)
		ui.message = fmt.Sprintf("Creato PNG %s.", name)
		ui.refreshAll()
	})
}

func (ui *tviewUI) openRenamePNGModal() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}

	currentName := ui.pngs[ui.selected].Name
	input := tview.NewInputField().SetLabel(" Nuovo nome ").SetFieldWidth(28)
	input.SetText(currentName)
	input.SetBorder(true).SetTitle("Rinomina PNG")
	returnFocus := ui.app.GetFocus()

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(input, 48, 0, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalVisible = true
	ui.modalName = "rename_png"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(input)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}
		newName := strings.TrimSpace(input.GetText())
		if newName == "" {
			ui.message = "Nome PNG non valido."
			ui.refreshStatus()
			return
		}
		for i, p := range ui.pngs {
			if i == ui.selected {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(p.Name), newName) {
				ui.message = "Nome già esistente."
				ui.refreshStatus()
				return
			}
		}
		ui.pushUndo()
		ui.pngs[ui.selected].Name = newName
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(focusPNG)
		ui.message = fmt.Sprintf("PNG rinominato in %s.", newName)
		ui.refreshAll()
	})
}

func (ui *tviewUI) weaponOptions() []string {
	weaponCats := map[string]bool{
		"Armi da Mischia": true, "Armi a Distanza": true, "Armi da Fuoco": true,
		"Armi da Tiro": true, "Armi Pesanti": true, "Armi Speciali": true,
	}
	opts := []string{"(nessuna)"}
	for _, it := range ui.equipment {
		if weaponCats[it.Category] {
			opts = append(opts, it.Name)
		}
	}
	return opts
}

func (ui *tviewUI) armorOptions() []string {
	armorCats := map[string]bool{"Armature": true, "Scudi": true}
	opts := []string{"(nessuna)"}
	for _, it := range ui.equipment {
		if armorCats[it.Category] {
			opts = append(opts, it.Name)
		}
	}
	return opts
}

func optionIndex(opts []string, val string) int {
	for i, o := range opts {
		if o == val {
			return i
		}
	}
	return 0
}

func pngResourcesBadge(resources []common.PNGResource) string {
	if len(resources) == 0 {
		return ""
	}
	parts := make([]string, 0, len(resources))
	for _, r := range resources {
		parts = append(parts, fmt.Sprintf("%s:%d/%d", r.Name, r.Current, r.Max))
	}
	return strings.Join(parts, " ")
}

func (ui *tviewUI) openPNGResourceModal() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	if ui.modalVisible {
		return
	}

	pngIdx := ui.selected
	resources := make([]common.PNGResource, len(ui.pngs[pngIdx].Resources))
	copy(resources, ui.pngs[pngIdx].Resources)

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(fmt.Sprintf(" Risorse: %s (-/+: usa/ripristina, a:aggiungi, d:elimina, R:ripristina tutti, 0:azz., Esc:chiudi) ", ui.pngs[pngIdx].Name))
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(false)

	render := func() {
		cur := list.GetCurrentItem()
		list.Clear()
		if len(resources) == 0 {
			list.AddItem("(nessuna risorsa — premi 'a' per aggiungerne una)", "", 0, nil)
		} else {
			for _, r := range resources {
				list.AddItem(fmt.Sprintf("%-20s %d / %d", r.Name, r.Current, r.Max), "", 0, nil)
			}
		}
		if cur >= 0 && cur < list.GetItemCount() {
			list.SetCurrentItem(cur)
		}
	}

	save := func() {
		ui.pngs[pngIdx].Resources = resources
		ui.persistPNGs()
		ui.refreshPNGs()
		ui.pngList.SetCurrentItem(pngIdx)
		ui.refreshDetail()
	}

	closeModal := func() {
		save()
		ui.pages.RemovePage("png-resources")
		ui.modalVisible = false
		ui.app.SetFocus(ui.pngList)
		ui.message = fmt.Sprintf("Risorse salvate per %s.", ui.pngs[pngIdx].Name)
		ui.refreshStatus()
	}

	openAddResource := func() {
		nameInput := tview.NewInputField().SetLabel("Nome risorsa: ").SetFieldWidth(20)
		maxInput := tview.NewInputField().SetLabel("Valore max: ").SetFieldWidth(6).SetAcceptanceFunc(tview.InputFieldInteger)
		frame := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nameInput, 1, 0, true).
			AddItem(maxInput, 1, 0, false)
		frame.SetBorder(true).SetTitle(" Nuova Risorsa (Tab: campo successivo, Invio: conferma, Esc: annulla) ").SetTitleAlign(tview.AlignLeft)
		frame.SetBorderColor(tcell.ColorGold)
		frame.SetTitleColor(tcell.ColorGold)
		frame.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			switch ev.Key() {
			case tcell.KeyEscape:
				ui.pages.RemovePage("png-resource-add")
				ui.app.SetFocus(list)
				return nil
			case tcell.KeyTab:
				if ui.app.GetFocus() == nameInput {
					ui.app.SetFocus(maxInput)
				} else {
					ui.app.SetFocus(nameInput)
				}
				return nil
			case tcell.KeyBacktab:
				if ui.app.GetFocus() == maxInput {
					ui.app.SetFocus(nameInput)
				} else {
					ui.app.SetFocus(maxInput)
				}
				return nil
			case tcell.KeyEnter:
				name := strings.TrimSpace(nameInput.GetText())
				maxVal := 0
				fmt.Sscanf(maxInput.GetText(), "%d", &maxVal)
				if name != "" && maxVal > 0 {
					resources = append(resources, common.PNGResource{Name: name, Current: maxVal, Max: maxVal})
					render()
					list.SetCurrentItem(len(resources) - 1)
				}
				ui.pages.RemovePage("png-resource-add")
				ui.app.SetFocus(list)
				return nil
			}
			return ev
		})
		addModal := tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(frame, 5, 0, true).
				AddItem(nil, 0, 1, false), 50, 0, true).
			AddItem(nil, 0, 1, false)
		ui.pages.AddPage("png-resource-add", addModal, true, true)
		ui.app.SetFocus(nameInput)
	}

	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		cur := list.GetCurrentItem()
		switch {
		case ev.Key() == tcell.KeyEscape || (ev.Key() == tcell.KeyRune && ev.Rune() == 'q'):
			closeModal()
			return nil
		case ev.Key() == tcell.KeyRune && ev.Rune() == 'a':
			openAddResource()
			return nil
		case ev.Key() == tcell.KeyRune && ev.Rune() == 'd':
			if cur >= 0 && cur < len(resources) {
				resources = append(resources[:cur], resources[cur+1:]...)
				render()
			}
			return nil
		case ev.Key() == tcell.KeyRune && (ev.Rune() == '-' || ev.Rune() == ' '):
			if cur >= 0 && cur < len(resources) {
				if resources[cur].Current > 0 {
					resources[cur].Current--
				}
				render()
			}
			return nil
		case ev.Key() == tcell.KeyRune && (ev.Rune() == '+' || ev.Rune() == '='):
			if cur >= 0 && cur < len(resources) {
				if resources[cur].Current < resources[cur].Max {
					resources[cur].Current++
				}
				render()
			}
			return nil
		case ev.Key() == tcell.KeyRune && ev.Rune() == 'R':
			for i := range resources {
				resources[i].Current = resources[i].Max
			}
			render()
			return nil
		case ev.Key() == tcell.KeyRune && ev.Rune() == '0':
			for i := range resources {
				resources[i].Current = 0
			}
			render()
			return nil
		case ev.Key() == tcell.KeyEnter:
			closeModal()
			return nil
		default:
			return ev
		}
	})

	render()
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 0, 1, true).
			AddItem(nil, 0, 1, false), 80, 0, true).
		AddItem(nil, 0, 1, false)
	ui.modalVisible = true
	ui.pages.AddPage("png-resources", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *tviewUI) openEditPNGModal() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	cur := ui.pngs[ui.selected]
	returnFocus := ui.app.GetFocus()

	selName := cur.Name
	selToken := cur.Token
	selDescription := cur.Description
	selTraits := cur.Traits
	selPrimary := cur.Primary
	selSecondary := cur.Secondary
	selArmor := cur.Armor
	selInventory := cur.Inventory
	selLook := cur.Look

	weapons := ui.weaponOptions()
	armors := ui.armorOptions()

	save := func() {
		newName := strings.TrimSpace(selName)
		if newName == "" {
			ui.message = "Nome PNG non valido."
			ui.refreshStatus()
			return
		}
		for i, p := range ui.pngs {
			if i == ui.selected {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(p.Name), newName) {
				ui.message = "Nome già esistente."
				ui.refreshStatus()
				return
			}
		}
		ui.pushUndo()
		ui.pngs[ui.selected].Name = newName
		ui.pngs[ui.selected].Token = selToken
		ui.pngs[ui.selected].Description = strings.TrimSpace(selDescription)
		ui.pngs[ui.selected].Traits = strings.TrimSpace(selTraits)
		ui.pngs[ui.selected].Primary = selPrimary
		ui.pngs[ui.selected].Secondary = selSecondary
		ui.pngs[ui.selected].Armor = selArmor
		ui.pngs[ui.selected].Inventory = strings.TrimSpace(selInventory)
		ui.pngs[ui.selected].Look = strings.TrimSpace(selLook)
		ui.persistPNGs()
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.message = fmt.Sprintf("PNG '%s' aggiornato.", newName)
		ui.refreshDetail()
		ui.refreshPNGs()
		ui.refreshStatus()
	}

	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf(" Modifica PNG: %s ", cur.Name)).SetTitleAlign(tview.AlignLeft)

	// formFocusAt moves focus to the given absolute index (items first, then buttons).
	formFocusAt := func(idx int) {
		itemCount := form.GetFormItemCount()
		if idx < itemCount {
			ui.app.SetFocus(form.GetFormItem(idx))
		} else {
			ui.app.SetFocus(form.GetButton(idx - itemCount))
		}
		form.SetFocus(idx)
	}

	// helper: add dropdown with proper styling (navigation handled at form level)
	addDD := func(label string, opts []string, initIdx int, ch func(string, int)) {
		form.AddDropDown(label, opts, initIdx, ch)
		idx := form.GetFormItemCount() - 1
		if item := form.GetFormItem(idx); item != nil {
			if dd, ok := item.(*tview.DropDown); ok {
				dd.SetFieldBackgroundColor(tcell.ColorBlack)
				dd.SetFieldTextColor(tcell.ColorWhite)
				dd.SetListStyles(
					tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
					tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
				)
			}
		}
	}

	// item indices: Stile=0, Nome=1, Token=2, Descrizione=3, Tratti=4, ArmaPrim=5, ArmaSecond=6, Armatura=7, Inventario=8, Aspetto=9
	nameTypeOpts := []string{"fantasy", "cyberpunk"}
	initNameTypeIdx := 0
	for i, o := range nameTypeOpts {
		if o == currentNameType {
			initNameTypeIdx = i
			break
		}
	}
	addDD("Stile nomi", nameTypeOpts, initNameTypeIdx, func(text string, _ int) {
		currentNameType = text
	})
	form.AddInputField("Nome", cur.Name, 40, nil, func(s string) { selName = s })
	if nameItem, ok := form.GetFormItem(1).(*tview.InputField); ok {
		ng.BindRandomNameInput(nameItem, func() string { return uniqueRandomPNGName(ui.pngs) })
	}
	form.AddInputField("Token", strconv.Itoa(cur.Token), 5, func(s string, _ rune) bool {
		_, err := strconv.Atoi(s)
		return s == "" || err == nil
	}, func(s string) {
		if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			selToken = v
		}
	})
	form.AddInputField("Descrizione", cur.Description, 50, nil, func(s string) { selDescription = s })
	form.AddInputField("Tratti", cur.Traits, 50, nil, func(s string) { selTraits = s })
	addDD("Arma primaria", weapons, optionIndex(weapons, cur.Primary), func(opt string, _ int) {
		if opt == "(nessuna)" {
			selPrimary = ""
		} else {
			selPrimary = opt
		}
	})
	addDD("Arma secondaria", weapons, optionIndex(weapons, cur.Secondary), func(opt string, _ int) {
		if opt == "(nessuna)" {
			selSecondary = ""
		} else {
			selSecondary = opt
		}
	})
	addDD("Armatura", armors, optionIndex(armors, cur.Armor), func(opt string, _ int) {
		if opt == "(nessuna)" {
			selArmor = ""
		} else {
			selArmor = opt
		}
	})
	form.AddInputField("Inventario", cur.Inventory, 50, nil, func(s string) { selInventory = s })
	form.AddInputField("Aspetto", cur.Look, 50, nil, func(s string) { selLook = s })
	form.AddButton("Salva", save)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
	})

	// Unified Tab/Shift+Tab navigation at the form level.
	// form.SetInputCapture runs before any item handler, and is NOT overwritten by form.Focus().
	// item indices: Stile=0, Nome=1, Token=2, Descrizione=3, Tratti=4,
	//               ArmaPrim=5, ArmaSecond=6, Armatura=7, Inventario=8, Aspetto=9;
	//               Salva=btn0(10), Annulla=btn1(11)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlO:
			save()
			return nil
		case tcell.KeyTab, tcell.KeyBacktab:
			itemIdx, btnIdx := form.GetFocusedItemIndex()
			var cur int
			if itemIdx >= 0 {
				// If a DropDown list is open, pass the event through to it.
				if dd, ok := form.GetFormItem(itemIdx).(*tview.DropDown); ok && dd.IsOpen() {
					return event
				}
				cur = itemIdx
			} else if btnIdx >= 0 {
				cur = form.GetFormItemCount() + btnIdx
			} else {
				return event
			}
			total := form.GetFormItemCount() + form.GetButtonCount()
			var next int
			if event.Key() == tcell.KeyTab {
				next = (cur + 1) % total
			} else {
				next = (cur + total - 1) % total
			}
			formFocusAt(next)
			return nil
		}
		return event
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 76, 0, true).
			AddItem(nil, 0, 1, false), 18, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = save
	ui.modalVisible = true
	ui.modalName = "edit_png"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

func swadeRankName(advances int) string {
	switch {
	case advances < 4:
		return "Novizio"
	case advances < 8:
		return "Esperto"
	case advances < 12:
		return "Veterano"
	case advances < 16:
		return "Eroico"
	default:
		return "Leggendario"
	}
}

var swadeDieSteps = []string{"d4", "d6", "d8", "d10", "d12", "d12+2", "d12+4"}

func swadeNextDie(current string) string {
	if current == "" {
		return "d6"
	}
	for i, d := range swadeDieSteps {
		if d == current && i < len(swadeDieSteps)-1 {
			return swadeDieSteps[i+1]
		}
	}
	return current
}

func pngAttrDie(p *PNG, attr string) string {
	switch attr {
	case "Agilità":
		if p.Agilita == "" {
			return "d4"
		}
		return p.Agilita
	case "Intelligenza":
		if p.Intelligenza == "" {
			return "d4"
		}
		return p.Intelligenza
	case "Spirito":
		if p.Spirito == "" {
			return "d4"
		}
		return p.Spirito
	case "Forza":
		if p.Forza == "" {
			return "d4"
		}
		return p.Forza
	case "Vigore":
		if p.Vigore == "" {
			return "d4"
		}
		return p.Vigore
	}
	return "d4"
}

func setPngAttrDie(p *PNG, attr, die string) {
	switch attr {
	case "Agilità":
		p.Agilita = die
	case "Intelligenza":
		p.Intelligenza = die
	case "Spirito":
		p.Spirito = die
	case "Forza":
		p.Forza = die
	case "Vigore":
		p.Vigore = die
	}
}

var swadeSkillList = []string{
	"Atletica", "Conoscenze Comuni", "Furtività", "Percezione", "Persuasione",
	"Arte del Furto", "Cavalcare", "Combattere", "Guidare", "Navigare", "Pilotare",
	"Sparare", "Esibirsi", "Intimidire", "Provocare",
	"Battaglia", "Conoscenze Accademiche", "Elettronica", "Gioco d'Azzardo",
	"Guarigione", "Hackerare", "Linguaggi", "Occulto", "Ricerca", "Riparare",
	"Scienze", "Sopravvivenza",
	"Arti Psioniche", "Fede", "Focus", "Lanciare Incantesimi", "Scienza Folle",
}

var swadeAttrList = []string{"Agilità", "Intelligenza", "Spirito", "Forza", "Vigore"}

func (ui *tviewUI) openPNGAdvancementModal() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	p := &ui.pngs[ui.selected]
	advances := p.Level
	rankName := swadeRankName(advances)
	nextRankAt := ((advances / 4) + 1) * 4
	toNext := nextRankAt - advances

	headerText := fmt.Sprintf("%s | Avanzamenti: %d | Rango: %s | Prossimo rango in: %d", p.Name, advances, rankName, toNext)

	advTypes := []string{
		"Nuova Speciale (Edge)",
		"Aumenta Abilità",
		"Aumenta Caratteristica",
	}

	typeList := tview.NewList().ShowSecondaryText(false)
	typeList.SetBorder(true).SetTitle(" Tipo di Avanzamento (Invio=scegli, Esc=annulla) ")
	typeList.SetBorderColor(tcell.ColorGold).SetTitleColor(tcell.ColorGold)
	typeList.SetSelectedTextColor(tcell.ColorBlack).SetSelectedBackgroundColor(tcell.ColorGold)
	for _, t := range advTypes {
		typeList.AddItem(t, "", 0, nil)
	}

	header := tview.NewTextView().SetText(headerText).SetTextColor(tcell.ColorSilver)

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 2, 0, false).
		AddItem(typeList, len(advTypes)+2, 0, true)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(container, 70, 0, true).
			AddItem(nil, 0, 1, false), len(advTypes)+4, 0, true).
		AddItem(nil, 0, 1, false)

	closeAndBack := func() {
		ui.closeModal()
		ui.focusPanel(focusPNG)
	}

	typeList.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			closeAndBack()
			return nil
		}
		return ev
	})

	typeList.SetSelectedFunc(func(idx int, _ string, _ string, _ rune) {
		ui.closeModal()

		switch idx {
		case 0: // Nuova Speciale
			ui.openAdvancementEdgeModal(p)
		case 1: // Aumenta Abilità
			ui.openAdvancementSkillModal(p)
		case 2: // Aumenta Caratteristica
			ui.openAdvancementAttrModal(p)
		}
	})

	ui.modalVisible = true
	ui.modalName = "png_advancement"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(typeList)
}

func (ui *tviewUI) openAdvancementEdgeModal(p *PNG) {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" Nuova Speciale (Edge) ").SetTitleAlign(tview.AlignLeft)
	form.SetBorderColor(tcell.ColorGold).SetTitleColor(tcell.ColorGold)
	selEdge := ""
	form.AddInputField("Nome Speciale", "", 40, nil, func(s string) { selEdge = s })
	confirm := func() {
		edge := strings.TrimSpace(selEdge)
		if edge == "" {
			return
		}
		ui.pushUndo()
		p.Level++
		p.Rank = p.Level / 4
		if p.Traits != "" {
			p.Traits += ", " + edge
		} else {
			p.Traits = edge
		}
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(focusPNG)
		ui.message = fmt.Sprintf("%s avanzato! Edge: %s | Rango: %s", p.Name, edge, swadeRankName(p.Level))
		ui.refreshDetail()
		ui.refreshStatus()
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlO {
			confirm()
			return nil
		}
		return event
	})
	form.AddButton("Conferma", confirm)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.focusPanel(focusPNG)
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.focusPanel(focusPNG)
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 60, 0, true).
			AddItem(nil, 0, 1, false), 8, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = confirm
	ui.modalVisible = true
	ui.modalName = "png_advancement_edge"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

func (ui *tviewUI) openAdvancementSkillModal(p *PNG) {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" Aumenta Abilità ").SetTitleAlign(tview.AlignLeft)
	form.SetBorderColor(tcell.ColorGold).SetTitleColor(tcell.ColorGold)

	selSkill := swadeSkillList[0]
	selDie := swadeDieSteps[1]

	formFocusAt := func(idx int) {
		itemCount := form.GetFormItemCount()
		if idx < itemCount {
			ui.app.SetFocus(form.GetFormItem(idx))
		} else {
			ui.app.SetFocus(form.GetButton(idx - itemCount))
		}
		form.SetFocus(idx)
	}
	styleDD := func(idx int, next int, prev int) {
		if item := form.GetFormItem(idx); item != nil {
			if dd, ok := item.(*tview.DropDown); ok {
				dd.SetFieldBackgroundColor(tcell.ColorBlack)
				dd.SetFieldTextColor(tcell.ColorWhite)
				dd.SetListStyles(
					tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
					tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
				)
				dd.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
					if dd.IsOpen() {
						return event
					}
					switch event.Key() {
					case tcell.KeyTab:
						formFocusAt(next)
						return nil
					case tcell.KeyBacktab:
						formFocusAt(prev)
						return nil
					}
					return event
				})
			}
		}
	}
	form.AddDropDown("Abilità", swadeSkillList, 0, func(opt string, _ int) { selSkill = opt })
	styleDD(0, 1, 3) // next=Dado, prev=Annulla (wrap)
	form.AddDropDown("Dado", swadeDieSteps, 1, func(opt string, _ int) { selDie = opt })
	styleDD(1, 2, 0) // next=Conferma, prev=Abilità
	confirm := func() {
		ui.pushUndo()
		p.Level++
		p.Rank = p.Level / 4
		entry := selSkill + " " + selDie
		if p.Traits != "" {
			p.Traits += ", " + entry
		} else {
			p.Traits = entry
		}
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(focusPNG)
		ui.message = fmt.Sprintf("%s avanzato! Abilità: %s | Rango: %s", p.Name, entry, swadeRankName(p.Level))
		ui.refreshDetail()
		ui.refreshStatus()
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlO {
			confirm()
			return nil
		}
		return event
	})
	form.AddButton("Conferma", confirm)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.focusPanel(focusPNG)
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.focusPanel(focusPNG)
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 60, 0, true).
			AddItem(nil, 0, 1, false), 9, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = confirm
	ui.modalVisible = true
	ui.modalName = "png_advancement_skill"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

func (ui *tviewUI) openAdvancementAttrModal(p *PNG) {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" Aumenta Caratteristica ").SetTitleAlign(tview.AlignLeft)
	form.SetBorderColor(tcell.ColorGold).SetTitleColor(tcell.ColorGold)

	selAttr := swadeAttrList[0]

	attrLabels := make([]string, len(swadeAttrList))
	for i, a := range swadeAttrList {
		cur := pngAttrDie(p, a)
		next := swadeNextDie(cur)
		attrLabels[i] = fmt.Sprintf("%s (%s → %s)", a, cur, next)
	}

	form.AddDropDown("Caratteristica", attrLabels, 0, func(_ string, idx int) {
		selAttr = swadeAttrList[idx]
	})
	if item := form.GetFormItem(0); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			dd.SetFieldBackgroundColor(tcell.ColorBlack)
			dd.SetFieldTextColor(tcell.ColorWhite)
			dd.SetListStyles(
				tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
				tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
			)
			dd.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
					if dd.IsOpen() {
						return event
					}
					itemCount := form.GetFormItemCount()
					switch event.Key() {
					case tcell.KeyTab:
						ui.app.SetFocus(form.GetButton(0)) // → Conferma
						form.SetFocus(itemCount)
						return nil
					case tcell.KeyBacktab:
						ui.app.SetFocus(form.GetButton(1)) // → Annulla (wrap)
						form.SetFocus(itemCount + 1)
						return nil
					}
					return event
				})
		}
	}
	confirm := func() {
		cur := pngAttrDie(p, selAttr)
		next := swadeNextDie(cur)
		ui.pushUndo()
		p.Level += 2
		p.Rank = p.Level / 4
		setPngAttrDie(p, selAttr, next)
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(focusPNG)
		ui.message = fmt.Sprintf("%s avanzato! %s: %s→%s | Rango: %s", p.Name, selAttr, cur, next, swadeRankName(p.Level))
		ui.refreshDetail()
		ui.refreshStatus()
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlO {
			confirm()
			return nil
		}
		return event
	})
	form.AddButton("Conferma (+2 Avanz.)", confirm)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.focusPanel(focusPNG)
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.focusPanel(focusPNG)
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 70, 0, true).
			AddItem(nil, 0, 1, false), 9, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = confirm
	ui.modalVisible = true
	ui.modalName = "png_advancement_attr"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

func (ui *tviewUI) openClassPNGInput() {
	idx := ui.currentClassIndex()
	if idx < 0 || idx >= len(ui.classes) {
		ui.message = "Nessuna regola selezionata."
		ui.refreshStatus()
		return
	}
	c := ui.classes[idx]
	if strings.EqualFold(strings.TrimSpace(c.Source), "carta") {
		ui.message = "Seleziona una classe per generare un PNG."
		ui.refreshStatus()
		return
	}
	returnFocus := ui.app.GetFocus()

	levels := make([]string, 0, 10)
	for i := 1; i <= 10; i++ {
		levels = append(levels, strconv.Itoa(i))
	}
	selectedLevel := 1
	ready := false

	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Crea PNG da Regole").SetTitleAlign(tview.AlignLeft)
	advanceToGenerate := func() {
		form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Genera"))
	}
	form.AddDropDown("Livello", levels, 0, func(option string, _ int) {
		if option == "" {
			return
		}
		if v, err := strconv.Atoi(option); err == nil && v > 0 {
			selectedLevel = v
		}
		if ready {
			advanceToGenerate()
		}
	})
	if item := form.GetFormItem(0); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			dd.SetFieldBackgroundColor(tcell.ColorBlack)
			dd.SetFieldTextColor(tcell.ColorWhite)
			dd.SetListStyles(
				tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
				tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
			)
			dd.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if dd.IsOpen() {
					return event
				}
				switch event.Key() {
				case tcell.KeyTab:
					genIdx := form.GetFormItemCount() + form.GetButtonIndex("Genera")
					ui.app.SetFocus(form.GetButton(form.GetButtonIndex("Genera")))
					form.SetFocus(genIdx)
					return nil
				case tcell.KeyBacktab:
					annIdx := form.GetFormItemCount() + form.GetButtonIndex("Annulla")
					ui.app.SetFocus(form.GetButton(form.GetButtonIndex("Annulla")))
					form.SetFocus(annIdx)
					return nil
				}
				return event
			})
		}
	}

	genera := func() {
		baseName := uniqueRandomPNGName(ui.pngs)
		preset := classPresetFor(c.Name)
		inv := buildSuggestedInventory(preset)
		look := buildSuggestedLook(preset)
		png := PNG{
			Name:        fmt.Sprintf("%s (%s | %s L%d)", baseName, c.Subclass, c.Name, selectedLevel),
			Class:       strings.TrimSpace(c.Name),
			Subclass:    strings.TrimSpace(c.Subclass),
			Level:       selectedLevel,
			Rank:        rankFromLevel(selectedLevel),
			CompBonus:   progressionBonusAtLevel(selectedLevel),
			ExpBonus:    progressionBonusAtLevel(selectedLevel),
			Description: strings.TrimSpace(c.Description),
			Traits:      strings.TrimSpace(preset.Traits),
			Primary:     strings.TrimSpace(preset.Primary),
			Secondary:   strings.TrimSpace(preset.Secondary),
			Armor:       strings.TrimSpace(preset.Armor),
			Look:        look,
			Inventory:   inv,
		}
		ui.pushUndo()
		ui.pngs = append(ui.pngs, png)
		ui.selected = len(ui.pngs) - 1
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(focusPNG)
		ui.message = fmt.Sprintf("Creato PNG da regole: %s | %s L%d", c.Subclass, c.Name, selectedLevel)
		ui.refreshAll()
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlO {
			genera()
			return nil
		}
		return event
	})
	form.AddButton("Genera", genera)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetButtonsAlign(tview.AlignLeft)
	ready = true

	info := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	info.SetText(fmt.Sprintf("[yellow]%s | %s[-]\nScegli il livello e genera un PNG.", c.Subclass, c.Name))

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(info, 2, 0, false).
		AddItem(form, 0, 1, true)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(container, 66, 0, true).
			AddItem(nil, 0, 1, false), 11, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = genera
	ui.modalVisible = true
	ui.modalName = "class_png"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

func chooseOne(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.TrimSpace(items[rand.IntN(len(items))])
}

func buildSuggestedInventory(p classPreset) string {
	base := []string{
		"torcia",
		"16 metri di corda",
		"provviste di base",
		"una manciata d'oro",
	}
	potion := chooseOne([]string{"Pozione di Guarigione Minore", "Pozione di Recupero Minore"})
	extra := chooseOne([]string{strings.TrimSpace(p.ExtraA), strings.TrimSpace(p.ExtraB)})
	if potion != "" {
		base = append(base, potion)
	}
	if extra != "" {
		base = append(base, extra)
	}
	return strings.Join(base, ", ")
}

func buildSuggestedLook(p classPreset) string {
	eyes := []string{
		"vivaci", "del colore della terra", "dell'oceano", "di fuoco", "verde edera", "lilla", "la notte", "schiuma del mare", "gelidi",
	}
	body := []string{
		"spalle larghe", "scolpita", "formosa", "allampanata", "tondeggiante", "piccola statura", "robusta", "alta", "slanciata", "minuta", "allenata",
	}
	skin := []string{
		"cenere", "nivea", "sabbia", "ossidiana", "rosea", "trifoglio", "zaffiro", "glicine",
	}

	abiti := chooseOne(p.Abiti)
	atteggiamento := chooseOne(p.Attitude)
	occhi := chooseOne(eyes)
	corporatura := chooseOne(body)
	carnagione := chooseOne(skin)

	parts := []string{}
	if abiti != "" {
		parts = append(parts, "Abiti: "+abiti)
	}
	if occhi != "" {
		parts = append(parts, "Occhi: "+occhi)
	}
	if corporatura != "" {
		parts = append(parts, "Corporatura: "+corporatura)
	}
	if carnagione != "" {
		parts = append(parts, "Carnagione: "+carnagione)
	}
	if atteggiamento != "" {
		parts = append(parts, "Atteggiamento: "+atteggiamento)
	}
	return strings.Join(parts, " | ")
}

func classPresetFor(className string) classPreset {
	return common.ClassPresetFor(className)
}

func rankFromLevel(level int) int {
	switch {
	case level <= 1:
		return 1
	case level <= 4:
		return 2
	case level <= 7:
		return 3
	default:
		return 4
	}
}

func progressionBonusAtLevel(level int) int {
	bonus := 0
	if level >= 2 {
		bonus++
	}
	if level >= 5 {
		bonus++
	}
	if level >= 8 {
		bonus++
	}
	return bonus
}

func (ui *tviewUI) findClassDefinition(className, subclass string) *ClassItem {
	for i := range ui.classes {
		c := &ui.classes[i]
		if strings.EqualFold(strings.TrimSpace(c.Source), "carta") {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(c.Name), strings.TrimSpace(className)) &&
			strings.EqualFold(strings.TrimSpace(c.Subclass), strings.TrimSpace(subclass)) {
			return c
		}
	}
	return nil
}

func (ui *tviewUI) openDeletePNGConfirm() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	name := ui.pngs[ui.selected].Name
	ui.openConfirmModal("Conferma", fmt.Sprintf("Eliminare PNG '%s'?", name), func() {
		ui.deleteSelectedPNG()
	})
}

func (ui *tviewUI) openConfirmModal(title, message string, onConfirm func()) {
	returnFocus := ui.app.GetFocus()
	text := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	text.SetBorder(true).SetTitle(title)
	text.SetText(message + "\n\n[yellow]Invio/y[-] conferma  [yellow]Esc/n[-] annulla")
	text.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEnter || (ev.Key() == tcell.KeyRune && (ev.Rune() == 'y' || ev.Rune() == 'Y')) {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			onConfirm()
			return nil
		}
		if ev.Key() == tcell.KeyEscape || (ev.Key() == tcell.KeyRune && (ev.Rune() == 'n' || ev.Rune() == 'N')) {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			ui.refreshStatus()
			return nil
		}
		return ev
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(text, 56, 0, true).
			AddItem(nil, 0, 1, false), 8, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalVisible = true
	ui.modalName = "confirm"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(text)
}

func (ui *tviewUI) openRawSearch(focus tview.Primitive) {
	input := tview.NewInputField().SetLabel(" Cerca ").SetFieldWidth(28)
	input.SetBorder(true).SetTitle("Ricerca")
	if focus == ui.dice {
		if len(ui.diceLog) > 0 {
			cur := ui.dice.GetCurrentItem()
			if cur >= 0 && cur < len(ui.diceLog) {
				input.SetText(ui.diceLog[cur].Expression)
			}
		}
	}
	if focus == ui.search || focus == ui.monList || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.monSourceDrop {
		input.SetText(ui.search.GetText())
	}
	if focus == ui.eqSearch || focus == ui.eqList || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.eqSourceDrop {
		input.SetText(ui.eqSearch.GetText())
	}
	if focus == ui.cardSearch || focus == ui.cardList || focus == ui.cardClassDrop || focus == ui.cardTypeDrop {
		input.SetText(ui.cardSearch.GetText())
	}
	if focus == ui.classSearch || focus == ui.classList || focus == ui.classNameDrop || focus == ui.classSubDrop || focus == ui.classSourceDrop {
		input.SetText(ui.classSearch.GetText())
	}
	if focus == ui.detail {
		input.SetText(ui.detailQuery)
	}
	if focus == ui.detailTreasure {
		input.SetText(ui.detailQuery)
	}

	returnFocus := focus
	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(input, 48, 0, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalVisible = true
	ui.modalName = "raw_search"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(input)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}
		query := strings.TrimSpace(input.GetText())
		switch returnFocus {
		case ui.dice:
			ui.jumpToDiceResult(query)
			ui.focusPanel(focusDice)
		case ui.search, ui.monList, ui.roleDrop, ui.rankDrop, ui.monSourceDrop:
			ui.search.SetText(query)
			ui.refreshMonsters()
			ui.focusPanel(focusMonList)
			ui.message = "Filtro mostri aggiornato."
		case ui.eqSearch, ui.eqList, ui.eqTypeDrop, ui.eqItemTypeDrop, ui.eqRankDrop, ui.eqSourceDrop:
			ui.eqSearch.SetText(query)
			ui.refreshEquipment()
			ui.focusPanel(focusEqList)
			ui.message = "Filtro equipaggiamento aggiornato."
		case ui.cardSearch, ui.cardList, ui.cardClassDrop, ui.cardTypeDrop:
			ui.cardSearch.SetText(query)
			ui.refreshCards()
			ui.focusPanel(focusCardList)
			ui.message = "Filtro carte aggiornato."
		case ui.classSearch, ui.classList, ui.classNameDrop, ui.classSubDrop, ui.classSourceDrop:
			ui.classSearch.SetText(query)
			ui.refreshClasses()
			ui.focusPanel(focusClassList)
			ui.message = "Filtro regole aggiornato."
		case ui.encList:
			ui.jumpToEncounter(query)
		case ui.detail:
			ui.detailQuery = query
			ui.renderDetail()
			if query == "" {
				ui.message = "Highlight dettagli rimosso."
			} else {
				ui.message = fmt.Sprintf("Highlight dettagli: %s", query)
			}
		case ui.detailTreasure:
			ui.detailQuery = query
			ui.renderTreasure()
			if query == "" {
				ui.message = "Highlight treasure rimosso."
			} else {
				ui.message = fmt.Sprintf("Highlight treasure: %s", query)
			}
		default:
			ui.search.SetText(query)
			ui.refreshMonsters()
			ui.focusPanel(focusMonList)
		}
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
}

func (ui *tviewUI) jumpToEncounter(query string) {
	if strings.TrimSpace(query) == "" {
		ui.message = "Ricerca encounter vuota."
		return
	}
	q := strings.ToLower(query)
	for i, e := range ui.encounter {
		if strings.Contains(strings.ToLower(e.Monster.Name), q) {
			ui.encList.SetCurrentItem(i)
			ui.message = fmt.Sprintf("Trovato in encounter: %s", e.Monster.Name)
			ui.refreshDetail()
			return
		}
	}
	ui.message = fmt.Sprintf("Nessun match encounter per '%s'.", query)
}

func (ui *tviewUI) jumpToDiceResult(query string) {
	if strings.TrimSpace(query) == "" {
		ui.message = "Ricerca dadi vuota."
		return
	}
	if idx, ok := parseDiceJumpIndex(query, len(ui.diceLog)); ok {
		ui.dice.SetCurrentItem(idx)
		ui.message = fmt.Sprintf("Jump dadi: #%d", idx+1)
		ui.refreshDetail()
		return
	}
	q := strings.ToLower(query)
	for i, e := range ui.diceLog {
		line := strings.ToLower(e.Expression + " " + e.Output)
		if strings.Contains(line, q) {
			ui.dice.SetCurrentItem(i)
			ui.message = fmt.Sprintf("Trovato in dadi: #%d", i+1)
			ui.refreshDetail()
			return
		}
	}
	ui.message = fmt.Sprintf("Nessun match dadi per '%s'.", query)
}

func parseDiceJumpIndex(query string, total int) (int, bool) {
	if total <= 0 {
		return 0, false
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, false
	}
	if strings.HasPrefix(q, "#") {
		q = strings.TrimSpace(strings.TrimPrefix(q, "#"))
	}
	if q == "" {
		return 0, false
	}
	for _, r := range q {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(q)
	if err != nil || n < 1 || n > total {
		return 0, false
	}
	return n - 1, true
}

func diceGotoIndexFromRune(r rune, total int) (int, bool) {
	if total <= 0 {
		return 0, false
	}
	if r == '^' {
		return 0, true
	}
	if r == '$' {
		return total - 1, true
	}
	if r < '0' || r > '9' {
		return 0, false
	}
	return parseDiceJumpIndex(string(r), total)
}

func (ui *tviewUI) scrollDetailByPage(direction int) {
	target := ui.detail
	if ui.app.GetFocus() == ui.detailTreasure {
		target = ui.detailTreasure
	}
	row, col := target.GetScrollOffset()
	_, _, _, h := target.GetInnerRect()
	if h <= 0 {
		h = 24
	}
	step := h / 2
	if step < 1 {
		step = 1
	}
	row += direction * step
	if row < 0 {
		row = 0
	}
	target.ScrollTo(row, col)
}

func (ui *tviewUI) deleteSelectedPNG() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	name := ui.pngs[ui.selected].Name
	ui.pngs = append(ui.pngs[:ui.selected], ui.pngs[ui.selected+1:]...)
	if len(ui.pngs) == 0 {
		ui.selected = -1
	} else if ui.selected >= len(ui.pngs) {
		ui.selected = len(ui.pngs) - 1
	}
	ui.persistPNGs()
	ui.message = fmt.Sprintf("PNG %s eliminato.", name)
	ui.refreshAll()
}

func (ui *tviewUI) addSelectedMonsterToEncounter() {
	idx := ui.currentMonsterIndex()
	if idx < 0 {
		ui.message = "Nessun mostro selezionato."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	mon := ui.monsters[idx]
	woundsMax := monsterWoundsCap(mon)
	ui.encounter = append(ui.encounter, EncounterEntry{
		Monster:    mon,
		WoundsMax:  woundsMax,
		BasePF:     woundsMax,
		Stress:     0,
		BaseStress: 0,
	})
	ui.clearEncounterInitTracking()
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Aggiunto %s a encounter.", mon.Name)
	ui.refreshEncounter()
	ui.encList.SetCurrentItem(len(ui.encounter) - 1)
	ui.focusPanel(focusEncounter)
	ui.refreshStatus()
}

func (ui *tviewUI) addSelectedPNGToEncounter() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	p := ui.pngs[ui.selected]
	name := strings.TrimSpace(p.Name)
	if name == "" {
		ui.message = "Nome PNG non valido."
		ui.refreshStatus()
		return
	}

	woundsMax := 3
	if def := ui.findClassDefinition(p.Class, p.Subclass); def != nil && def.HP > 0 {
		woundsMax = def.HP
	}
	if p.Level > 1 {
		woundsMax += (p.Level - 1) / 2
	}
	if woundsMax < 1 {
		woundsMax = 1
	}

	mon := Monster{
		Name:               "PNG: " + name,
		Role:               "PNG",
		WildCard:           true,
		Size:               0,
		Pace:               "6",
		Parry:              "-",
		Toughness:          "-",
		WoundsMax:          woundsMax,
		Description:        strings.TrimSpace(p.Description),
		MotivationsTactics: strings.TrimSpace(p.Traits),
	}
	if strings.TrimSpace(p.Primary) != "" {
		mon.Skills = append(mon.Skills, "Primario: "+strings.TrimSpace(p.Primary))
	}
	if strings.TrimSpace(p.Secondary) != "" {
		mon.Skills = append(mon.Skills, "Secondario: "+strings.TrimSpace(p.Secondary))
	}
	if strings.TrimSpace(p.Armor) != "" {
		mon.Skills = append(mon.Skills, "Armatura: "+strings.TrimSpace(p.Armor))
	}

	ui.encounter = append(ui.encounter, EncounterEntry{
		Monster:    mon,
		WoundsMax:  woundsMax,
		BasePF:     woundsMax,
		Stress:     0,
		BaseStress: 0,
	})
	ui.clearEncounterInitTracking()
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Aggiunto PNG %s a encounter.", name)
	ui.refreshEncounter()
	ui.encList.SetCurrentItem(len(ui.encounter) - 1)
	ui.focusPanel(focusEncounter)
	ui.refreshStatus()
}

func battleCostForRole(role string) int {
	r := strings.ToLower(strings.TrimSpace(role))
	switch {
	case strings.Contains(r, "seguace"):
		return 1
	case strings.Contains(r, "controparte"), strings.Contains(r, "supporto"):
		return 1
	case strings.Contains(r, "orda"), strings.Contains(r, "tiratore"), strings.Contains(r, "sicario"), strings.Contains(r, "base"):
		return 2
	case strings.Contains(r, "condottiero"):
		return 3
	case strings.Contains(r, "bruto"):
		return 4
	case strings.Contains(r, "solitario"):
		return 5
	default:
		return 0
	}
}

func isFollowerRole(role string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(role)), "seguace")
}

func battleBudgetModifierByDifficulty(label string) int {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "facile":
		return -2
	case "difficile":
		return 2
	default:
		return 0
	}
}

type generatedEncounterSummary struct {
	Size          int
	PGCount       int
	BaseBudget    int
	BudgetMod     int
	FinalBudget   int
	Spent         int
	Remaining     int
	AddedEntries  int
	AddedGroups   int
	ByMonsterName map[string]int
}

func (ui *tviewUI) openRandomEncounterFromMonstersInput() {
	sizeOptions := make([]string, 0, len(ui.rankOpts))
	defaultSizeIdx := 0
	currentMonsterIdx := ui.currentMonsterIndex()
	currentSize := 0
	if currentMonsterIdx >= 0 && currentMonsterIdx < len(ui.monsters) {
		currentSize = ui.monsters[currentMonsterIdx].Size
	}
	for _, opt := range ui.rankOpts {
		if strings.EqualFold(strings.TrimSpace(opt), "Tutti") {
			continue
		}
		sizeOptions = append(sizeOptions, opt)
		if currentSize != 0 && opt == strconv.Itoa(currentSize) {
			defaultSizeIdx = len(sizeOptions) - 1
		}
	}
	if len(sizeOptions) == 0 {
		ui.message = "Nessuna taglia disponibile nei Mostri."
		ui.refreshStatus()
		return
	}

	defaultPG := len(ui.pngs)
	if defaultPG < 1 {
		defaultPG = 1
	}
	selectedSize, _ := strconv.Atoi(sizeOptions[defaultSizeIdx])
	if selectedSize == 0 {
		selectedSize = 1
	}
	selectedPG := defaultPG
	difficultyOptions := []string{"Normale", "Facile", "Difficile"}
	selectedDifficulty := difficultyOptions[0]
	ready := false
	returnFocus := ui.app.GetFocus()

	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Genera Encounter Random da Mostri").SetTitleAlign(tview.AlignLeft)
	advanceToGenerate := func() {
		form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Genera"))
	}
	form.AddDropDown("Taglia gruppo", sizeOptions, defaultSizeIdx, func(option string, _ int) {
		if option == "" {
			return
		}
		if v, err := strconv.Atoi(strings.TrimSpace(option)); err == nil && v != 0 {
			selectedSize = v
		}
		if ready {
			form.SetFocus(1)
		}
	})
	form.AddInputField("PG in combatt.", strconv.Itoa(defaultPG), 5, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v > 0 {
			selectedPG = v
		}
	})
	form.AddDropDown("Difficoltà", difficultyOptions, 0, func(option string, _ int) {
		if option == "" {
			return
		}
		selectedDifficulty = option
		if ready {
			advanceToGenerate()
		}
	})

	encFormFocusAt := func(idx int) {
		itemCount := form.GetFormItemCount()
		if idx < itemCount {
			ui.app.SetFocus(form.GetFormItem(idx))
		} else {
			ui.app.SetFocus(form.GetButton(idx - itemCount))
		}
		form.SetFocus(idx)
	}
	if item := form.GetFormItem(0); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			dd.SetFieldBackgroundColor(tcell.ColorBlack)
			dd.SetFieldTextColor(tcell.ColorWhite)
			dd.SetListStyles(
				tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
				tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
			)
			dd.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if dd.IsOpen() {
					return event
				}
				switch event.Key() {
				case tcell.KeyTab:
					encFormFocusAt(1)
					return nil
				case tcell.KeyBacktab:
					encFormFocusAt(form.GetFormItemCount() + form.GetButtonIndex("Annulla"))
					return nil
				}
				return event
			})
		}
	}
	if item := form.GetFormItem(1); item != nil {
		if input, ok := item.(*tview.InputField); ok {
			input.SetFieldBackgroundColor(tcell.ColorBlack)
			input.SetFieldTextColor(tcell.ColorWhite)
			input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Key() {
				case tcell.KeyTab:
					encFormFocusAt(2)
					return nil
				case tcell.KeyBacktab:
					encFormFocusAt(0)
					return nil
				}
				return event
			})
		}
	}
	if item := form.GetFormItem(2); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			dd.SetFieldBackgroundColor(tcell.ColorBlack)
			dd.SetFieldTextColor(tcell.ColorWhite)
			dd.SetListStyles(
				tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
				tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
			)
			dd.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if dd.IsOpen() {
					return event
				}
				switch event.Key() {
				case tcell.KeyTab:
					encFormFocusAt(form.GetFormItemCount() + form.GetButtonIndex("Genera"))
					return nil
				case tcell.KeyBacktab:
					encFormFocusAt(1)
					return nil
				}
				return event
			})
		}
	}

	genera := func() {
		v := strings.TrimSpace(form.GetFormItem(1).(*tview.InputField).GetText())
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			ui.message = "Numero PG non valido."
			ui.refreshStatus()
			return
		}
		selectedPG = n
		mod := battleBudgetModifierByDifficulty(selectedDifficulty)
		ui.pushUndo()
		summary := ui.generateRandomEncounterFromMonsters(selectedSize, selectedPG, mod)
		if summary.AddedEntries == 0 {
			ui.message = fmt.Sprintf("Nessun mostro generato (S%d, %d PG).", selectedSize, selectedPG)
			ui.refreshStatus()
			return
		}
		ui.closeModal()
		ui.focusPanel(focusEncounter)
		ui.message = fmt.Sprintf("Encounter random S%d: +%d nemici (%d PB spesi, %d residui).", selectedSize, summary.AddedEntries, summary.Spent, summary.Remaining)
		ui.refreshEncounter()
		ui.detailRaw = buildGeneratedEncounterDetails(summary)
		ui.renderDetail()
		ui.detail.ScrollTo(0, 0)
		ui.refreshStatus()
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlO {
			genera()
			return nil
		}
		return event
	})
	form.AddButton("Genera", genera)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetButtonsAlign(tview.AlignLeft)

	info := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	info.SetText("Punti Battaglia: (3 x PG in combattimento) + 2.\nDifficoltà: Facile -2, Normale 0, Difficile +2.\nCosti ruolo: Seguace/Controparte/Supporto=1, Base/Tiratore/Sicario/Orda=2, Condottiero=3, Bruto=4, Solitario=5.")

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(info, 4, 0, false).
		AddItem(form, 0, 1, true)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(container, 80, 0, true).
			AddItem(nil, 0, 1, false), 13, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = genera
	ui.modalVisible = true
	ui.modalName = "monster_random_encounter"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
	ready = true
}

func (ui *tviewUI) generateRandomEncounterFromMonsters(size int, pgCount int, budgetMod int) generatedEncounterSummary {
	summary := generatedEncounterSummary{
		Size:          size,
		PGCount:       pgCount,
		BaseBudget:    3*pgCount + 2,
		BudgetMod:     budgetMod,
		ByMonsterName: map[string]int{},
	}
	if pgCount < 1 {
		return summary
	}
	finalBudget := summary.BaseBudget + budgetMod
	if finalBudget < 1 {
		finalBudget = 1
	}
	summary.FinalBudget = finalBudget

	type candidate struct {
		mon  Monster
		cost int
	}
	candidates := make([]candidate, 0, len(ui.monsters))
	for _, m := range ui.monsters {
		if m.Size != size {
			continue
		}
		cost := battleCostForRole(m.Role)
		if cost <= 0 {
			continue
		}
		candidates = append(candidates, candidate{mon: m, cost: cost})
	}
	if len(candidates) == 0 {
		summary.Remaining = finalBudget
		return summary
	}

	remaining := finalBudget
	added := 0
	spent := 0
	for remaining > 0 {
		affordable := make([]candidate, 0, len(candidates))
		for _, c := range candidates {
			if c.cost <= remaining {
				affordable = append(affordable, c)
			}
		}
		if len(affordable) == 0 {
			break
		}
		pick := affordable[rand.IntN(len(affordable))]
		qty := 1
		if isFollowerRole(pick.mon.Role) {
			qty = pgCount
		}
		for i := 0; i < qty; i++ {
			woundsMax := monsterWoundsCap(pick.mon)
			ui.encounter = append(ui.encounter, EncounterEntry{
				Monster:    pick.mon,
				WoundsMax:  woundsMax,
				BasePF:     woundsMax,
				Stress:     0,
				BaseStress: 0,
			})
			added++
		}
		summary.AddedGroups++
		summary.ByMonsterName[pick.mon.Name] += qty
		remaining -= pick.cost
		spent += pick.cost
	}
	if added > 0 {
		ui.clearEncounterInitTracking()
		ui.persistEncounter()
	}
	summary.AddedEntries = added
	summary.Spent = spent
	summary.Remaining = remaining
	return summary
}

func buildGeneratedEncounterDetails(s generatedEncounterSummary) string {
	var b strings.Builder
	b.WriteString("Encounter random generato\n")
	b.WriteString(fmt.Sprintf("Taglia gruppo: %d | PG: %d\n", s.Size, s.PGCount))
	b.WriteString(fmt.Sprintf("Punti Battaglia: %d %+d = %d\n", s.BaseBudget, s.BudgetMod, s.FinalBudget))
	b.WriteString(fmt.Sprintf("Spesi: %d | Residui: %d\n", s.Spent, s.Remaining))
	b.WriteString(fmt.Sprintf("Nemici aggiunti: %d (gruppi estratti: %d)\n", s.AddedEntries, s.AddedGroups))
	b.WriteString("\nDettaglio:\n")
	if len(s.ByMonsterName) == 0 {
		b.WriteString("- Nessun mostro aggiunto.\n")
		return strings.TrimSpace(b.String())
	}
	names := make([]string, 0, len(s.ByMonsterName))
	for name := range s.ByMonsterName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		b.WriteString(fmt.Sprintf("- %s x%d\n", name, s.ByMonsterName[name]))
	}
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) toggleEncounterDisabled() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	ui.encounter[idx].Disabled = !ui.encounter[idx].Disabled
	name := ui.encounter[idx].Monster.Name
	if ui.encounter[idx].Disabled {
		ui.message = fmt.Sprintf("%s disabilitato (saltato nell'iniziativa).", name)
	} else {
		ui.message = fmt.Sprintf("%s riabilitato.", name)
	}
	ui.persistEncounter()
	ui.refreshEncounter()
	ui.refreshStatus()
}

func (ui *tviewUI) removeSelectedEncounter() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	name := ui.encounter[idx].Monster.Name
	ui.encounter = append(ui.encounter[:idx], ui.encounter[idx+1:]...)
	ui.clearEncounterInitTracking()
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Rimosso %s da encounter.", name)
	ui.refreshAll()
}

func (ui *tviewUI) adjustEncounterWounds(delta int) {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	e := &ui.encounter[idx]
	prevWounds := e.Wounds
	e.Wounds += delta
	if e.Wounds < 0 {
		e.Wounds = 0
	}
	base := encounterWoundsCap(*e)
	if base > 0 && e.Wounds > base {
		e.Wounds = base
	}
	appliedShaken := applyShakenOnWoundReduction(prevWounds, e)
	ui.persistEncounter()
	if appliedShaken {
		ui.message = fmt.Sprintf("Ferite %s: %d/%d | Stato aggiunto: Scosso", e.Monster.Name, e.Wounds, base)
	} else {
		ui.message = fmt.Sprintf("Ferite %s: %d/%d", e.Monster.Name, e.Wounds, base)
	}
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) rollEncounterInitiativeSelected() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	e := &ui.encounter[idx]
	card, ok := ui.drawInitiativeCard(idx)
	if !ok {
		ui.message = "Mazzo esaurito per iniziative uniche."
		ui.refreshStatus()
		return
	}
	e.InitiativeCard = card
	e.HasInit = true
	ui.clearEncounterInitTracking()
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Iniziativa %s: %s", e.Monster.Name, e.InitiativeCard)
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) rollEncounterInitiativeAll() {
	if len(ui.encounter) == 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	deck := buildInitiativeDeck()
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	for i := range ui.encounter {
		if i < len(deck) {
			ui.encounter[i].InitiativeCard = deck[i]
			ui.encounter[i].HasInit = true
			continue
		}
		ui.encounter[i].InitiativeCard = ""
		ui.encounter[i].HasInit = false
	}
	ui.clearEncounterInitTracking()
	ui.persistEncounter()
	if len(ui.encounter) > len(deck) {
		ui.message = "Iniziativa tirata per i primi 52; mazzo esaurito."
	} else {
		ui.message = "Iniziativa tirata per tutti."
	}
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.refreshStatus()
}

// dealInitiativeToAll shuffles a fresh deck and distributes one card each to
// all PNGs and all encounter entries, then persists both lists.
func (ui *tviewUI) dealInitiativeToAll() {
	total := len(ui.pngs) + len(ui.encounter)
	if total == 0 {
		ui.message = "Nessun combattente (PNG o Encounter) presente."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	deck := buildInitiativeDeck()
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	di := 0
	for i := range ui.pngs {
		if di < len(deck) {
			ui.pngs[i].InitiativeCard = deck[di]
			ui.pngs[i].HasInit = true
			di++
		} else {
			ui.pngs[i].InitiativeCard = ""
			ui.pngs[i].HasInit = false
		}
	}
	for i := range ui.encounter {
		if di < len(deck) {
			ui.encounter[i].InitiativeCard = deck[di]
			ui.encounter[i].HasInit = true
			di++
		} else {
			ui.encounter[i].InitiativeCard = ""
			ui.encounter[i].HasInit = false
		}
	}
	ui.clearEncounterInitTracking()
	ui.persistPNGs()
	ui.persistEncounter()
	dealt := min(total, len(deck))
	if total > len(deck) {
		ui.message = fmt.Sprintf("Carte distribuite a %d combattenti; mazzo esaurito (%d rimasti senza carta).", dealt, total-dealt)
	} else {
		ui.message = fmt.Sprintf("Carte distribuite a %d PNG e %d mostri.", len(ui.pngs), len(ui.encounter))
	}
	ui.refreshAll()
}

func (ui *tviewUI) sortEncounterByInitiative() {
	if len(ui.encounter) == 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	currentLabel := ui.encounterLabelAt(ui.currentEncounterIndex())
	sort.SliceStable(ui.encounter, func(i, j int) bool {
		ai, aj := ui.encounter[i], ui.encounter[j]
		if ai.HasInit != aj.HasInit {
			return ai.HasInit
		}
		if ai.HasInit && aj.HasInit {
			cmp := compareInitiativeCards(ai.InitiativeCard, aj.InitiativeCard)
			if cmp != 0 {
				return cmp < 0
			}
		}
		return ai.Monster.Name < aj.Monster.Name
	})
	ui.persistEncounter()
	ui.encInitSorted = true
	ui.encInitModeActive = false
	ui.encInitTurnIndex = 0
	ui.encInitRound = 1
	ui.refreshEncounter()
	if currentLabel != "" {
		for i := range ui.encounter {
			if ui.encounterLabelAt(i) == currentLabel {
				ui.encList.SetCurrentItem(i)
				break
			}
		}
	}
	ui.refreshDetail()
	ui.message = "Encounter ordinato per iniziativa."
	ui.refreshStatus()
}

func (ui *tviewUI) clearEncounterInitTracking() {
	ui.encInitSorted = false
	ui.encInitModeActive = false
	ui.encInitTurnIndex = 0
	ui.encInitRound = 1
}

func (ui *tviewUI) enterEncounterInitiativeMode() {
	if len(ui.encounter) == 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	if !ui.encInitSorted {
		ui.message = "Ordina prima con S per attivare la modalita iniziativa."
		ui.refreshStatus()
		return
	}
	ui.encInitModeActive = true
	ui.encInitTurnIndex = 0
	ui.encInitRound = 1
	ui.encList.SetCurrentItem(0)
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.message = "Modalita iniziativa: ON (n = prossimo turno)."
	ui.refreshStatus()
}

func (ui *tviewUI) advanceEncounterInitiativeTurn() {
	if len(ui.encounter) == 0 || !ui.encInitModeActive {
		return
	}
	n := len(ui.encounter)
	wrapped := false
	for range n {
		ui.encInitTurnIndex++
		if ui.encInitTurnIndex >= n {
			ui.encInitTurnIndex = 0
			ui.encInitRound++
			if ui.encInitRound < 1 {
				ui.encInitRound = 1
			}
			wrapped = true
		}
		if !ui.encounter[ui.encInitTurnIndex].Disabled {
			break
		}
	}
	ui.incrementEncounterConditionRoundsAt(ui.encInitTurnIndex)
	ui.encList.SetCurrentItem(ui.encInitTurnIndex)
	ui.refreshEncounter()
	ui.refreshDetail()
	if wrapped {
		ui.message = fmt.Sprintf("Round %d: nuovo giro iniziativa.", ui.encInitRound)
	} else {
		ui.message = fmt.Sprintf("Turno %d/%d.", ui.encInitTurnIndex+1, len(ui.encounter))
	}
	ui.refreshStatus()
}

func (ui *tviewUI) incrementEncounterConditionRoundsAt(idx int) {
	if idx < 0 || idx >= len(ui.encounter) {
		return
	}
	if len(ui.encounter[idx].Conditions) == 0 {
		return
	}
	ui.pushUndo()
	for code, rounds := range ui.encounter[idx].Conditions {
		if rounds <= 0 {
			ui.encounter[idx].Conditions[code] = 1
			continue
		}
		ui.encounter[idx].Conditions[code] = rounds + 1
	}
	ui.persistEncounter()
}

func (ui *tviewUI) openEncounterInitiativeEditModal() {
	idx := ui.currentEncounterIndex()
	if idx < 0 || idx >= len(ui.encounter) {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	entry := ui.encounter[idx]
	returnFocus := ui.app.GetFocus()

	defaultRank := ""
	defaultSuitIdx := 0
	if entry.HasInit {
		if rankIdx, suitIdx, ok := parseInitiativeCard(entry.InitiativeCard); ok {
			defaultRank = initiativeRanks[rankIdx]
			defaultSuitIdx = suitIdx
		}
	}
	selectedSuit := defaultSuitIdx

	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Modifica Voce Encounter").SetTitleAlign(tview.AlignLeft)
	form.AddInputField("Nome", entry.Monster.Name, 28, nil, nil)
	form.AddInputField("Rank", defaultRank, 4, nil, nil)
	form.AddDropDown("Seme", initiativeSuits, defaultSuitIdx, func(_ string, index int) {
		if index >= 0 && index < len(initiativeSuits) {
			selectedSuit = index
		}
	})

	save := func() {
		newName := strings.TrimSpace(form.GetFormItem(0).(*tview.InputField).GetText())
		rankText := strings.ToUpper(strings.TrimSpace(form.GetFormItem(1).(*tview.InputField).GetText()))
		if newName == "" {
			ui.message = "Nome non valido."
			ui.refreshStatus()
			return
		}
		if rankText == "" {
			ui.pushUndo()
			ui.encounter[idx].Monster.Name = newName
			ui.encounter[idx].HasInit = false
			ui.encounter[idx].InitiativeCard = ""
			ui.clearEncounterInitTracking()
			ui.persistEncounter()
			ui.closeModal()
			ui.app.SetFocus(ui.encList)
			ui.refreshEncounter()
			ui.encList.SetCurrentItem(idx)
			ui.refreshDetail()
			ui.message = fmt.Sprintf("Voce aggiornata: %s.", newName)
			ui.refreshStatus()
			return
		}

		cardText := rankText
		if _, _, ok := parseInitiativeCard(rankText); !ok {
			cardText = rankText + initiativeSuits[selectedSuit]
		}
		card, ok := normalizeInitiativeCard(cardText)
		if !ok {
			ui.message = "Carta iniziativa non valida. Usa rank (A,K,Q,J,10..2) + seme."
			ui.refreshStatus()
			return
		}

		ui.pushUndo()
		ui.encounter[idx].Monster.Name = newName
		ui.encounter[idx].HasInit = true
		ui.encounter[idx].InitiativeCard = card
		ui.clearEncounterInitTracking()
		ui.persistEncounter()
		ui.closeModal()
		ui.app.SetFocus(ui.encList)
		ui.refreshEncounter()
		ui.encList.SetCurrentItem(idx)
		ui.refreshDetail()
		ui.message = fmt.Sprintf("Voce aggiornata: %s (%s).", newName, card)
		ui.refreshStatus()
	}

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlO {
			save()
			return nil
		}
		return event
	})
	form.AddButton("Salva", save)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})

	if item := form.GetFormItem(2); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			dd.SetFieldBackgroundColor(tcell.ColorBlack)
			dd.SetFieldTextColor(tcell.ColorWhite)
			dd.SetListStyles(
				tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
				tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
			)
		}
	}

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 62, 0, true).
			AddItem(nil, 0, 1, false), 11, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = save
	ui.modalVisible = true
	ui.modalName = "encounter_init_edit"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

func (ui *tviewUI) openEncounterConditionModal() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	temp := cloneStringIntMap(ui.encounter[idx].Conditions)
	if temp == nil {
		temp = map[string]int{}
	}

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Encounter Conditions (Space=toggle, Enter=apply, Esc=cancel, a=tutti/nessuno) ")
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
			sym := d.Symbol
		if sym == "" {
			sym = d.Code
		}
		list.AddItem(fmt.Sprintf("%s %s %s", mark, sym, d.Name), "", 0, nil)
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
		cur := list.GetCurrentItem()
		if cur < 0 || cur >= len(encounterConditionDefs) {
			return
		}
		code := encounterConditionDefs[cur].Code
		if temp[code] > 0 {
			delete(temp, code)
		} else {
			temp[code] = 1
		}
		render()
	}

	toggleAll := func() {
		allOn := true
		for _, d := range encounterConditionDefs {
			if temp[d.Code] <= 0 {
				allOn = false
				break
			}
		}
		for _, d := range encounterConditionDefs {
			if allOn {
				delete(temp, d.Code)
			} else if temp[d.Code] <= 0 {
				temp[d.Code] = 1
			}
		}
		render()
	}

	closeModal := func(apply bool) {
		ui.pages.RemovePage("encounter-conditions")
		ui.app.SetFocus(ui.encList)
		if !apply {
			return
		}
		ui.pushUndo()
		ui.encounter[idx].Conditions = cloneStringIntMap(temp)
		ui.persistEncounter()
		ui.refreshEncounter()
		ui.encList.SetCurrentItem(idx)
		ui.refreshDetail()
		ui.message = fmt.Sprintf("Condizioni aggiornate su %s.", ui.encounterLabelAt(idx))
		ui.refreshStatus()
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == ' ':
			toggle()
			return nil
		case event.Key() == tcell.KeyRune && (event.Rune() == 'a' || event.Rune() == 'A'):
			toggleAll()
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

func (ui *tviewUI) clearEncounterConditions() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	if len(ui.encounter[idx].Conditions) == 0 {
		ui.message = "Nessuna condizione da rimuovere."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	ui.encounter[idx].Conditions = nil
	ui.persistEncounter()
	ui.refreshEncounter()
	ui.encList.SetCurrentItem(idx)
	ui.refreshDetail()
	ui.message = fmt.Sprintf("Condizioni rimosse da %s.", ui.encounterLabelAt(idx))
	ui.refreshStatus()
}

func (ui *tviewUI) removeEncounterConditionByCode(index int, code string) bool {
	if index < 0 || index >= len(ui.encounter) {
		return false
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" || len(ui.encounter[index].Conditions) == 0 {
		return false
	}
	if _, ok := ui.encounter[index].Conditions[code]; !ok {
		return false
	}
	ui.pushUndo()
	delete(ui.encounter[index].Conditions, code)
	if len(ui.encounter[index].Conditions) == 0 {
		ui.encounter[index].Conditions = nil
	}
	return true
}

func (ui *tviewUI) openEncounterConditionRemoveModal() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	entry := ui.encounter[idx]
	if len(entry.Conditions) == 0 {
		ui.message = "Nessuna condizione da rimuovere."
		ui.refreshStatus()
		return
	}

	active := make([]encounterConditionDef, 0, len(entry.Conditions))
	for _, d := range encounterConditionDefs {
		if entry.Conditions[d.Code] > 0 {
			active = append(active, d)
		}
	}
	if len(active) == 0 {
		ui.message = "Nessuna condizione da rimuovere."
		ui.refreshStatus()
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
		sym := d.Symbol
		if sym == "" {
			sym = d.Code
		}
		list.AddItem(fmt.Sprintf("%s %s (%d)", sym, d.Name, rounds), "", 0, nil)
	}

	closeModal := func() {
		ui.pages.RemovePage("encounter-condition-remove")
		ui.app.SetFocus(ui.encList)
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
			if ui.removeEncounterConditionByCode(idx, code) {
				ui.persistEncounter()
				ui.refreshEncounter()
				ui.encList.SetCurrentItem(idx)
				ui.refreshDetail()
				ui.message = fmt.Sprintf("Rimossa condizione %s da %s.", conditionNameByCode(code), ui.encounterLabelAt(idx))
				ui.refreshStatus()
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

func (ui *tviewUI) adjustEncounterConditionRounds(delta int) {
	if delta == 0 {
		return
	}
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	if len(ui.encounter[idx].Conditions) == 0 {
		ui.message = "Nessuna condizione attiva."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	for code, r := range ui.encounter[idx].Conditions {
		n := r + delta
		if n <= 0 {
			delete(ui.encounter[idx].Conditions, code)
		} else {
			ui.encounter[idx].Conditions[code] = n
		}
	}
	if len(ui.encounter[idx].Conditions) == 0 {
		ui.encounter[idx].Conditions = nil
	}
	ui.persistEncounter()
	ui.encInitSorted = true
	ui.encInitModeActive = false
	ui.encInitTurnIndex = 0
	ui.encInitRound = 1
	ui.refreshEncounter()
	ui.encList.SetCurrentItem(idx)
	ui.refreshDetail()
	if delta > 0 {
		ui.message = fmt.Sprintf("Round condizioni +1 su %s.", ui.encounterLabelAt(idx))
	} else {
		ui.message = fmt.Sprintf("Round condizioni -1 su %s.", ui.encounterLabelAt(idx))
	}
	ui.refreshStatus()
}

var initiativeRanks = []string{"A", "K", "Q", "J", "10", "9", "8", "7", "6", "5", "4", "3", "2"}
var initiativeSuits = []string{"♥", "♦", "♣", "♠"}

func buildInitiativeDeck() []string {
	deck := make([]string, 0, len(initiativeRanks)*len(initiativeSuits))
	for _, r := range initiativeRanks {
		for _, s := range initiativeSuits {
			deck = append(deck, r+s)
		}
	}
	return deck
}

func (ui *tviewUI) drawInitiativeCard(excludeIdx int) (string, bool) {
	used := make(map[string]struct{}, len(ui.encounter))
	for i, e := range ui.encounter {
		if i == excludeIdx {
			continue
		}
		if e.HasInit && strings.TrimSpace(e.InitiativeCard) != "" {
			used[e.InitiativeCard] = struct{}{}
		}
	}
	deck := buildInitiativeDeck()
	available := make([]string, 0, len(deck))
	for _, c := range deck {
		if _, ok := used[c]; !ok {
			available = append(available, c)
		}
	}
	if len(available) == 0 {
		return "", false
	}
	return available[rand.IntN(len(available))], true
}

func compareInitiativeCards(a, b string) int {
	ar, as, aok := parseInitiativeCard(a)
	br, bs, bok := parseInitiativeCard(b)
	if !aok && !bok {
		return strings.Compare(a, b)
	}
	if !aok {
		return 1
	}
	if !bok {
		return -1
	}
	if as != bs {
		return as - bs
	}
	return ar - br
}

func parseInitiativeCard(card string) (rankIdx int, suitIdx int, ok bool) {
	card = strings.ToUpper(strings.TrimSpace(card))
	if card == "" {
		return 0, 0, false
	}
	for ri, r := range initiativeRanks {
		for si, s := range initiativeSuits {
			if card == r+s || card == s+r {
				return ri, si, true
			}
		}
	}
	return 0, 0, false
}

func normalizeInitiativeCard(card string) (string, bool) {
	rankIdx, suitIdx, ok := parseInitiativeCard(card)
	if !ok {
		return "", false
	}
	if rankIdx < 0 || rankIdx >= len(initiativeRanks) || suitIdx < 0 || suitIdx >= len(initiativeSuits) {
		return "", false
	}
	return initiativeRanks[rankIdx] + initiativeSuits[suitIdx], true
}

func (ui *tviewUI) openHelpOverlay(focus tview.Primitive) {
	if ui.helpVisible {
		return
	}
	ui.helpVisible = true
	ui.helpReturnFocus = focus

	text := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	text.SetBorder(true).SetTitle("Help")
	text.SetText(ui.buildHelpContent(focus))

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(text, 0, 5, true).
			AddItem(nil, 0, 1, false), 0, 5, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddAndSwitchToPage("help", modal, true)
	ui.app.SetFocus(text)
}

func (ui *tviewUI) buildHelpContent(focus tview.Primitive) string {
	var b strings.Builder
	b.WriteString("LazySW - scorciatoie\n\n")

	panel := "Dettagli"
	var panelLines []string
	switch focus {
	case ui.dice:
		panel = "Dadi"
		panelLines = []string{
			"- a: nuovo tiro (es. d6e, D6, 1d20+3 x2, d6,d8)",
			"- Invio: rilancia il tiro selezionato",
			"- e: modifica + rilancia il tiro selezionato",
			"- d: elimina il tiro selezionato",
			"- m: apri lista macro (espressioni nominate, Invio per lanciare)",
			"- M: imposta max voci log dadi (0=illimitato, persistente per campagna)",
			"- c: svuota storico tiri",
			"",
			"Legenda notazione:",
			"- NdM, NdM+K, NdM-K: tiro base (es. 2d6+3, d8-1)",
			"- d6e: dado esplosivo (sul massimo ritira e somma)",
			"- D6: SWADE — tratto d6e + wild die d6e, prendi max",
			"- d8v: vantaggio (max tra 2 tiri) | d8s: svantaggio (min)",
			"- d6,d8,1d20+4: multi-espressione (lancia separatamente)",
			"- 1d20+4 x3: batch — ripeti l'espressione N volte",
			"- d20+5>15 o >=15: confronto ok/ko vs DC",
			"- d20>15 2d6+1: se successo lancia espressione secondaria",
		}
	case ui.pngList:
		panel = "PNG"
		panelLines = []string{
			"- c: crea PNG",
			"- m: rinomina PNG selezionato",
			"- x: elimina PNG selezionato",
			"- a: aggiungi PNG selezionato a Encounter",
			"- e: modifica campi del PNG selezionato",
			"- b: gestisci risorse esauribili (-/+:usa/ripristina, a:aggiungi, R:ripristina tutti)",
			"- +: avanzamento SWADE (mostra opzioni e avanza)",
		}
	case ui.encList:
		panel = "Encounter"
		panelLines = []string{
			"- d: rimuovi mostro selezionato",
			"- h / l: ferite +1 / -1 sul selezionato",
			"- j / k: ferite +1 / -1 sul selezionato",
			"- c: aggiungi/togli condizioni (multi select)",
			"- x: rimuovi una condizione dall'entry",
			"- C: rimuovi tutte le condizioni dall'entry",
			"- [ / ]: diminuisci/aumenta round condizioni",
			"- o: toggle effetti estesi condizioni nei dettagli",
			"- i: tira iniziativa sul selezionato",
			"- I: tira iniziativa per tutti",
			"- A: tiro attacco del selezionato (nel log dadi)",
			"- K: tiro di tratto del selezionato (nel log dadi)",
			"- S: ordina encounter per iniziativa",
			"- *: entra in modalita iniziativa (solo dopo S)",
			"- n: prossimo turno (in modalita iniziativa)",
			"- e: modifica carta iniziativa selezionata",
			"- z: centra riga corrente a schermo",
			"- X: abilita/disabilita entry (disabilitato = saltato nell'iniziativa)",
			"- b: raggruppa/separa voci per nome (solo vista, non modifica i dati)",
			"- N: alterna numerazione numeri/lettere (es. Mostro #1 ↔ Mostro #A)",
		}
	case ui.search, ui.roleDrop, ui.rankDrop, ui.monSourceDrop, ui.monList:
		panel = "Mostri"
		panelLines = []string{
			"- a: aggiungi mostro selezionato a Encounter",
			"- n: genera Encounter random (Punti Battaglia)",
			"- u / t / g / y: focus filtro Nome / Ruolo / Taglia / Source",
			"- nel menu Source aperto: Space/Enter toggle, A tutti, N nessuno",
			"- v: reset filtri Mostri (Nome/Ruolo/Taglia/Source)",
		}
	case ui.eqSearch, ui.eqTypeDrop, ui.eqItemTypeDrop, ui.eqRankDrop, ui.eqSourceDrop, ui.eqList:
		panel = "Equipaggiamento"
		panelLines = []string{
			"- u / t / g / y: focus filtro Nome / Tipo / Era / Source",
			"- nel menu Source aperto: Space/Enter toggle, A tutti, N nessuno",
			"- v: reset filtri Equipaggiamento (Nome/Categoria/Tipo/Era/Source)",
			"- b: genera bottino (Treasure) da categoria + dadi",
			"- d: switch Dettagli <-> Treasure",
		}
	case ui.detailTreasure:
		panel = "Treasure"
		panelLines = []string{
			"- d: switch Treasure <-> Dettagli",
			"- /: evidenzia testo nel treasure corrente",
		}
	case ui.cardSearch, ui.cardClassDrop, ui.cardTypeDrop, ui.cardList:
		panel = "Carte"
		panelLines = []string{
			"- u / t / g: focus filtro Nome / Classe / Tipo",
			"- v: reset filtri Carte (Nome/Classe/Tipo)",
		}
	case ui.classSearch, ui.classNameDrop, ui.classSubDrop, ui.classSourceDrop, ui.classList:
		panel = "Regole"
		panelLines = []string{
			"- u / t / g / y: focus filtro Cerca / Categoria / Voce / Source",
			"- nel menu Source aperto: Space/Enter toggle, A tutti, N nessuno",
			"- v: reset filtri Regole (Cerca/Categoria/Voce/Source)",
			"- a: genera PNG dalla classe selezionata (solo voci classe)",
		}
	case ui.notesSearch, ui.notesList:
		panel = "Note"
		panelLines = []string{
			"- c: crea nuova nota",
			"- e: modifica nota selezionata",
			"- d: elimina nota selezionata",
			"- u: focus filtro ricerca",
			"- v: cancella filtro",
		}
	default:
		panelLines = []string{"- /: evidenzia testo nei dettagli"}
	}

	b.WriteString("[yellow]" + panel + "[-]\n")
	for _, line := range panelLines {
		b.WriteString(line + "\n")
	}

	b.WriteString("\n[yellow]Globali[-]\n")
	b.WriteString("- q: esci\n")
	b.WriteString("- ?: apri/chiudi help\n")
	b.WriteString("- tasto destro: menu contestuale con tutte le scorciatoie del pannello\n")
	b.WriteString("- tab / shift+tab: cambia focus\n")
	b.WriteString("- 0/1/2/3/4/5/6: focus Dadi/PNG/Encounter/Mostri/Equip/Regole/Note\n")
	b.WriteString("- i / I / S (su Encounter): iniziativa selezionato/tutti(solo mostri)/ordina\n")
	b.WriteString("- J (su PNG o Encounter): distribuisce carte iniziativa a TUTTI (PNG + mostri)\n")
	b.WriteString("- * / n / e (su Encounter): init mode / next turn / edit card\n")
	b.WriteString("- [ / ]: alterna Catalogo (oppure round condizioni su Encounter)\n")
	b.WriteString("- /: ricerca rapida sul pannello corrente\n")
	b.WriteString("- f: fullscreen pannello corrente\n")
	b.WriteString("- PgUp / PgDn: scroll Dettagli\n")
	b.WriteString("- Ctrl+N: nota rapida con timestamp round/turno (salva nelle Note)\n")
	b.WriteString("- Ctrl+O: gestione campagne (nei modali: conferma/invia il form)\n")
	b.WriteString("- Ctrl+T: storico modifiche (undo/redo navigabile)\n")
	b.WriteString("- Ctrl+D: pesca carta casuale dal mazzo\n")
	b.WriteString("- g+numero: goto riga nella lista (g1..g9, g^ prima, g$ ultima)\n")
	b.WriteString("- g'NN: goto riga numero a due cifre (es. g'12 va alla riga 12)\n")
	b.WriteString("\nEsc/?/q per chiudere")
	return b.String()
}

func (ui *tviewUI) contextMenuItemsForFocus(focus tview.Primitive) []contextItem {
	switch focus {
	case ui.dice:
		return []contextItem{
			{"a       - nuovo tiro (es. D6, 1d20+3)", ui.openDiceRollInput},
			{"Invio   - rilancia il tiro selezionato", ui.rerollSelectedDiceResult},
			{"e       - modifica + rilancia il tiro selezionato", ui.openDiceReRollInput},
			{"d       - elimina il tiro selezionato", ui.deleteSelectedDiceResult},
			{"m       - apri lista macro", ui.openDiceMacroModal},
			{"M       - imposta max voci log dadi", ui.openMaxDiceLogInput},
			{"c       - svuota storico tiri", ui.clearDiceResults},
		}
	case ui.pngList:
		return []contextItem{
			{"c       - crea PNG", ui.openCreatePNGModal},
			{"m       - rinomina PNG selezionato", ui.openRenamePNGModal},
			{"a       - aggiungi PNG selezionato a Encounter", ui.addSelectedPNGToEncounter},
			{"e       - modifica campi del PNG selezionato", ui.openEditPNGModal},
			{"b       - gestisci risorse esauribili", ui.openPNGResourceModal},
			{"+       - avanzamento SWADE", ui.openPNGAdvancementModal},
			{"y       - copia PNG corrente", ui.yankCurrentPNG},
			{"x       - elimina PNG selezionato", ui.openDeletePNGConfirm},
		}
	case ui.encList:
		return []contextItem{
			{"d       - rimuovi mostro selezionato", func() {
				ui.openConfirmModal("Conferma", "Rimuovere il mostro selezionato dall'encounter?", func() {
					ui.removeSelectedEncounter()
				})
			}},
			{"h / j   - ferite +1 sul selezionato", func() { ui.adjustEncounterWounds(1) }},
			{"l / k   - ferite -1 sul selezionato", func() { ui.adjustEncounterWounds(-1) }},
			{"c       - aggiungi/togli condizioni", ui.openEncounterConditionModal},
			{"x       - rimuovi una condizione", ui.openEncounterConditionRemoveModal},
			{"C       - rimuovi tutte le condizioni", ui.clearEncounterConditions},
			{"[       - diminuisci round condizioni", func() { ui.adjustEncounterConditionRounds(-1) }},
			{"]       - aumenta round condizioni", func() { ui.adjustEncounterConditionRounds(1) }},
			{"i       - tira iniziativa sul selezionato", ui.rollEncounterInitiativeSelected},
			{"I       - tira iniziativa per tutti", ui.rollEncounterInitiativeAll},
			{"A       - tiro attacco del selezionato", ui.openEncounterAttackModal},
			{"K       - tiro di tratto del selezionato", ui.openEncounterTraitModal},
			{"S       - ordina encounter per iniziativa", ui.sortEncounterByInitiative},
			{"*       - entra in modalita iniziativa", ui.enterEncounterInitiativeMode},
			{"n       - prossimo turno (in modalita iniziativa)", ui.advanceEncounterInitiativeTurn},
			{"e       - modifica carta iniziativa selezionata", ui.openEncounterInitiativeEditModal},
			{"y       - copia riga corrente", ui.yankCurrentEncounterEntry},
			{"J       - distribuisce carte iniziativa a tutti", ui.dealInitiativeToAll},
			{"X       - abilita/disabilita entry", ui.toggleEncounterDisabled},
		}
	case ui.monList:
		return []contextItem{
			{"a       - aggiungi mostro selezionato a Encounter", ui.addSelectedMonsterToEncounter},
			{"n       - genera Encounter random (Punti Battaglia)", ui.openRandomEncounterFromMonstersInput},
		}
	case ui.eqList:
		return []contextItem{
			{"b       - genera bottino da categoria + dadi", ui.openEquipmentTreasureInput},
			{"d       - switch Dettagli <-> Treasure", ui.toggleDetailsTreasureFocus},
		}
	case ui.classList:
		return []contextItem{
			{"a       - genera PNG dalla classe selezionata", ui.openClassPNGInput},
		}
	case ui.notesList:
		return []contextItem{
			{"c       - crea nuova nota", ui.openAddNoteModal},
			{"e       - modifica nota selezionata", ui.openEditNoteModal},
			{"d       - elimina nota selezionata", func() {
				ui.openConfirmModal("Conferma", "Eliminare la nota selezionata?", func() {
					ui.deleteSelectedNote()
				})
			}},
		}
	}
	return nil
}

func (ui *tviewUI) closeHelpOverlay() {
	if !ui.helpVisible {
		return
	}
	ui.helpVisible = false
	ui.pages.RemovePage("help")
	if ui.helpReturnFocus != nil {
		ui.app.SetFocus(ui.helpReturnFocus)
	}
}

func (ui *tviewUI) closeModal() {
	if !ui.modalVisible {
		return
	}
	if ui.modalName != "" {
		ui.pages.RemovePage(ui.modalName)
	}
	ui.modalVisible = false
	ui.modalName = ""
	ui.modalConfirmFunc = nil
}

func (ui *tviewUI) fullscreenTargetForFocus(focus tview.Primitive) string {
	switch focus {
	case ui.dice:
		return "dadi"
	case ui.pngList:
		return "png"
	case ui.encList:
		return "encounter"
	case ui.search, ui.monList, ui.roleDrop, ui.rankDrop, ui.monSourceDrop:
		return "monsters"
	case ui.eqSearch, ui.eqList, ui.eqTypeDrop, ui.eqItemTypeDrop, ui.eqRankDrop, ui.eqSourceDrop:
		return "equipaggiamento"
	case ui.cardSearch, ui.cardList, ui.cardClassDrop, ui.cardTypeDrop:
		return "carte"
	case ui.classSearch, ui.classList, ui.classNameDrop, ui.classSubDrop, ui.classSourceDrop:
		return "regole"
	case ui.notesSearch, ui.notesList:
		return "note"
	case ui.detailTreasure:
		return "treasure"
	case ui.detail:
		return "details"
	default:
		return ""
	}
}

func (ui *tviewUI) toggleFullscreenForFocus(focus tview.Primitive) {
	target := ui.fullscreenTargetForFocus(focus)
	if target == "" {
		return
	}
	if ui.fullscreenActive && ui.fullscreenTarget == target {
		ui.fullscreenActive = false
		ui.fullscreenTarget = ""
		ui.rebuildMainLayout()
		ui.message = "Fullscreen off"
		ui.refreshStatus()
		return
	}
	ui.fullscreenActive = true
	ui.fullscreenTarget = target
	ui.rebuildMainLayout()
	ui.message = "Fullscreen " + target
	ui.refreshStatus()
}

func (ui *tviewUI) rebuildMainLayout() {
	var content tview.Primitive = ui.mainRow
	if ui.fullscreenActive {
		switch ui.fullscreenTarget {
		case "dadi":
			content = ui.dice
		case "png":
			content = ui.pngList
		case "encounter":
			content = ui.encList
		case "monsters":
			content = ui.monstersPanel
		case "equipaggiamento":
			content = ui.equipmentPanel
		case "carte":
			content = ui.cardsPanel
		case "regole":
			content = ui.classesPanel
		case "note":
			content = ui.notesPanel
		case "treasure":
			content = ui.detailTreasure
		case "details":
			content = ui.detail
		}
	}

	// Update mainFlex in-place: remove old items, re-add in correct order.
	if ui.mainContent != nil {
		ui.mainFlex.RemoveItem(ui.mainContent)
	}
	if ui.timer != nil {
		ui.mainFlex.RemoveItem(ui.timer.Bar)
	}
	ui.mainFlex.RemoveItem(ui.status)

	ui.mainContent = content
	ui.mainFlex.AddItem(content, 0, 1, true)
	if ui.timer != nil && ui.timer.Running {
		ui.mainFlex.AddItem(ui.timer.Bar, 3, 0, false)
	}
	ui.mainFlex.AddItem(ui.status, 1, 0, false)
}

func (ui *tviewUI) persistPNGs() {
	_ = savePNGList(dataFile, ui.pngs, selectedPNGName(ui.pngs, ui.selected))
}

func (ui *tviewUI) persistEncounter() {
	entries := make([]struct {
		Name             string         `yaml:"name"`
		Wounds           int            `yaml:"wounds"`
		PF               int            `yaml:"pf"`
		InitiativeCard   string         `yaml:"initiative_card,omitempty"`
		LegacyInitiative int            `yaml:"initiative,omitempty"`
		HasInit          bool           `yaml:"has_initiative,omitempty"`
		Conditions       map[string]int `yaml:"conditions,omitempty"`
		Stress           int            `yaml:"stress,omitempty"`
		BaseStress       int            `yaml:"base_stress,omitempty"`
		Disabled         bool           `yaml:"disabled,omitempty"`
	}, 0, len(ui.encounter))
	for _, e := range ui.encounter {
		base := encounterWoundsCap(e)
		entries = append(entries, struct {
			Name             string         `yaml:"name"`
			Wounds           int            `yaml:"wounds"`
			PF               int            `yaml:"pf"`
			InitiativeCard   string         `yaml:"initiative_card,omitempty"`
			LegacyInitiative int            `yaml:"initiative,omitempty"`
			HasInit          bool           `yaml:"has_initiative,omitempty"`
			Conditions       map[string]int `yaml:"conditions,omitempty"`
			Stress           int            `yaml:"stress,omitempty"`
			BaseStress       int            `yaml:"base_stress,omitempty"`
			Disabled         bool           `yaml:"disabled,omitempty"`
		}{Name: e.Monster.Name, Wounds: e.Wounds, PF: base, InitiativeCard: e.InitiativeCard, HasInit: e.HasInit, Conditions: cloneStringIntMap(e.Conditions), Disabled: e.Disabled})
	}
	_ = saveEncounter(encounterFile, entries)
}

func (ui *tviewUI) toggleDetailsTreasureFocus() {
	if ui.activeBottomPane == "treasure" {
		ui.activeBottomPane = "details"
		ui.detailBottom.SwitchToPage("details")
		ui.app.SetFocus(ui.detail)
		ui.message = "Focus: Dettagli"
		ui.refreshStatus()
		return
	}
	ui.activeBottomPane = "treasure"
	ui.detailBottom.SwitchToPage("treasure")
	ui.app.SetFocus(ui.detailTreasure)
	ui.message = "Focus: Treasure"
	ui.refreshStatus()
}

func (ui *tviewUI) renderTreasure() {
	text := ui.treasureRaw
	if strings.TrimSpace(text) == "" {
		text = "Nessun treasure generato."
	}
	out := tview.Escape(text)
	lines := strings.Split(out, "\n")
	if len(lines) > 0 {
		lines[0] = "[yellow]" + lines[0] + "[-]"
		out = strings.Join(lines, "\n")
	}
	if strings.TrimSpace(ui.detailQuery) != "" {
		out = highlightMatches(out, ui.detailQuery)
	}
	ui.detailTreasure.SetText(out)
}

func (ui *tviewUI) openEquipmentTreasureInput() {
	categories := []string{"Comune", "Non Comune", "Raro", "Leggendario"}
	diceByCategory := map[string][]string{
		"Comune":      {"1d12", "2d12"},
		"Non Comune":  {"2d12", "3d12"},
		"Raro":        {"3d12", "4d12"},
		"Leggendario": {"4d12", "5d12"},
	}
	selectedCategory := categories[0]
	selectedDice := diceByCategory[selectedCategory][0]
	ready := false
	suppressDiceAdvance := false

	form := tview.NewForm()
	var categoryDrop *tview.DropDown
	var diceDrop *tview.DropDown
	advanceToGenerate := func() {
		form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Genera"))
	}
	form.SetBorder(true).SetTitle("Genera Treasure da Bottino").SetTitleAlign(tview.AlignLeft)
	form.AddDropDown("Categoria", categories, 0, func(option string, _ int) {
		if option == "" {
			return
		}
		selectedCategory = option
		selectedDice = diceByCategory[selectedCategory][0]
		if diceDrop == nil {
			return
		}
		diceDrop.SetOptions(diceByCategory[selectedCategory], func(text string, _ int) {
			if text != "" {
				selectedDice = text
			}
			if ready && !suppressDiceAdvance {
				advanceToGenerate()
			}
		})
		diceDrop.SetListStyles(
			tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
			tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
		)
		suppressDiceAdvance = true
		diceDrop.SetCurrentOption(0)
		suppressDiceAdvance = false
		if ready {
			form.SetFocus(1)
		}
	})
	form.AddDropDown("Dadi", diceByCategory[selectedCategory], 0, func(option string, _ int) {
		if option != "" {
			selectedDice = option
		}
		if ready && !suppressDiceAdvance {
			advanceToGenerate()
		}
	})
	if item := form.GetFormItem(0); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			categoryDrop = dd
			categoryDrop.SetFinishedFunc(func(key tcell.Key) {
				switch key {
				case tcell.KeyEnter, tcell.KeyTab:
					form.SetFocus(1)
				case tcell.KeyBacktab:
					form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Annulla"))
				}
			})
		}
	}
	if item := form.GetFormItem(1); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			diceDrop = dd
			diceDrop.SetFinishedFunc(func(key tcell.Key) {
				switch key {
				case tcell.KeyEnter, tcell.KeyTab:
					form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Genera"))
				case tcell.KeyBacktab:
					form.SetFocus(0)
				}
			})
		}
	}
	applyDropStyle := func(dd *tview.DropDown) {
		if dd == nil {
			return
		}
		dd.SetFieldBackgroundColor(tcell.ColorBlack)
		dd.SetFieldTextColor(tcell.ColorWhite)
		dd.SetListStyles(
			tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
			tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
		)
	}
	applyDropStyle(categoryDrop)
	applyDropStyle(diceDrop)

	returnFocus := ui.app.GetFocus()
	genera := func() {
		total, breakdown, err := rollDiceExpression(selectedDice)
		if err != nil {
			ui.message = "Errore tiro treasure: " + err.Error()
			ui.refreshStatus()
			return
		}
		matches := ui.matchBottinoByTiro(total)
		ui.renderEquipmentTreasure(selectedCategory, selectedDice, total, breakdown, matches)
		ui.closeModal()
		ui.activeBottomPane = "treasure"
		ui.detailBottom.SwitchToPage("treasure")
		ui.app.SetFocus(ui.detailTreasure)
		ui.message = fmt.Sprintf("Treasure generato: %s %s = %02d", selectedCategory, selectedDice, total)
		ui.refreshStatus()
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlO {
			genera()
			return nil
		}
		return event
	})
	form.AddButton("Genera", genera)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 72, 0, true).
			AddItem(nil, 0, 1, false), 13, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = genera
	ui.modalVisible = true
	ui.modalName = "equip_treasure"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form)
	ready = true
}

func (ui *tviewUI) matchBottinoByTiro(total int) []EquipmentItem {
	var matches []EquipmentItem
	for _, it := range ui.equipment {
		if !strings.EqualFold(strings.TrimSpace(it.Type), "bottino") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(it.Trait))
		if err != nil {
			continue
		}
		if n == total {
			matches = append(matches, it)
		}
	}
	return matches
}

func (ui *tviewUI) renderEquipmentTreasure(category, dice string, total int, breakdown string, matches []EquipmentItem) {
	var b strings.Builder
	b.WriteString("Treasure Equipaggiamento\n")
	b.WriteString(fmt.Sprintf("Categoria: %s\n", category))
	b.WriteString(fmt.Sprintf("Tiro: %s => %s\n", dice, breakdown))
	b.WriteString(fmt.Sprintf("Valore Tiro: %02d\n", total))
	b.WriteString("\nRisultati:\n")
	if len(matches) == 0 {
		b.WriteString("- Nessun bottino con Tiro corrispondente.\n")
	} else {
		for _, it := range matches {
			b.WriteString(fmt.Sprintf("- %s (Tiro %02d)\n", it.Name, total))
			if strings.TrimSpace(it.Characteristic) != "" && strings.TrimSpace(it.Characteristic) != "—" && strings.TrimSpace(it.Characteristic) != "-" {
				b.WriteString("  " + strings.TrimSpace(it.Characteristic) + "\n")
			}
		}
	}
	ui.treasureRaw = strings.TrimSpace(b.String())
	ui.renderTreasure()
	ui.detailTreasure.ScrollToBeginning()
}

func (ui *tviewUI) buildDiceDetail() string {
	var b strings.Builder
	b.WriteString("Dadi\n")
	if len(ui.diceLog) == 0 {
		b.WriteString("Nessun tiro registrato.\n\n")
		b.WriteString("Shortcut:\n")
		b.WriteString("- a: nuovo tiro\n")
		b.WriteString("- Invio: rilancia selezionato\n")
		b.WriteString("- e: modifica + rilancia\n")
		b.WriteString("- d: elimina selezionato\n")
		b.WriteString("- c: svuota storico\n")
		b.WriteString("\nLegenda notazione:\n")
		for _, line := range diceNotationLegend() {
			b.WriteString("- " + line + "\n")
		}
		return strings.TrimSpace(b.String())
	}

	cur := ui.dice.GetCurrentItem()
	if cur < 0 || cur >= len(ui.diceLog) {
		cur = len(ui.diceLog) - 1
	}
	entry := ui.diceLog[cur]
	b.WriteString(fmt.Sprintf("Tiro #%d\n", cur+1))
	b.WriteString("Espressione: " + entry.Expression + "\n")
	b.WriteString("Risultato: " + entry.Output + "\n")
	b.WriteString(fmt.Sprintf("\nTotale tiri: %d", len(ui.diceLog)))
	b.WriteString("\n\nLegenda notazione:\n")
	for _, line := range diceNotationLegend() {
		b.WriteString("- " + line + "\n")
	}
	return strings.TrimSpace(b.String())
}

func diceNotationLegend() []string {
	return []string{
		"NdM, NdM+K, NdM-K: tiro base (es. 2d6+3, d8-1)",
		"d6e: dado esplosivo (sul massimo ritira e somma)",
		"2d8e: più dadi esplosivi",
		"D6: SWADE — tratto d6e + wild die d6e, prendi max",
		"d8v: vantaggio — max tra 2 tiri dello stesso dado",
		"d8s: svantaggio — min tra 2 tiri dello stesso dado",
		"d6,d8,1d20+4: multi-espressione (lancia separatamente)",
		"1d20+4 x3: batch — ripeti l'espressione N volte",
		"d20+5>15 o >=15: confronto ok/ko vs DC",
		"d20>15 2d6+1: se successo lancia espressione secondaria",
		"d20+5>10 x3: batch confronto (N tiri)",
	}
}

func (ui *tviewUI) openDiceRollInput() {
	input := tview.NewInputField().SetLabel(" Tiro ").SetFieldWidth(36)
	input.SetBorder(true).SetTitle("Dadi (es. 2d6+3, 1d20+5>15, d6,d8, 1d20+4 x3)")
	returnFocus := ui.app.GetFocus()

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(input, 72, 0, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalVisible = true
	ui.modalName = "dice_roll"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(input)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}
		raw := strings.TrimSpace(input.GetText())
		if raw == "" {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}

		exprs, err := expandDiceRollInput(raw)
		if err != nil {
			ui.message = "Errore dadi: " + err.Error()
			ui.refreshStatus()
			return
		}
		for _, expr := range exprs {
			_, breakdown, rollErr := rollDiceExpression(expr)
			if rollErr != nil {
				ui.message = "Errore dadi: " + rollErr.Error()
				ui.refreshStatus()
				continue
			}
			ui.appendDiceLog(DiceResult{Expression: expr, Output: breakdown})
		}
		ui.closeModal()
		ui.focusPanel(focusDice)
		ui.message = fmt.Sprintf("Registrati %d tiri.", len(exprs))
		ui.refreshDetail()
		ui.refreshStatus()
	})
}

func (ui *tviewUI) openDiceReRollInput() {
	if len(ui.diceLog) == 0 {
		ui.openDiceRollInput()
		return
	}

	cur := ui.dice.GetCurrentItem()
	if cur < 0 || cur >= len(ui.diceLog) {
		cur = len(ui.diceLog) - 1
	}

	input := tview.NewInputField().SetLabel(" Tiro ").SetFieldWidth(36)
	input.SetBorder(true).SetTitle("Modifica + Rilancia")
	input.SetText(ui.diceLog[cur].Expression)
	returnFocus := ui.app.GetFocus()

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(input, 64, 0, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalVisible = true
	ui.modalName = "dice_reroll"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(input)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}
		expr := strings.TrimSpace(input.GetText())
		if expr == "" {
			return
		}
		exprs, err := expandDiceRollInput(expr)
		if err != nil {
			ui.message = "Errore dadi: " + err.Error()
			ui.refreshStatus()
			return
		}
		results := make([]DiceResult, 0, len(exprs))
		for _, ex := range exprs {
			_, breakdown, rollErr := rollDiceExpression(ex)
			if rollErr != nil {
				ui.message = "Errore dadi: " + rollErr.Error()
				ui.refreshStatus()
				return
			}
			results = append(results, DiceResult{Expression: ex, Output: breakdown})
		}
		if len(results) == 0 {
			ui.message = "Nessun tiro valido."
			ui.refreshStatus()
			return
		}
		ui.diceLog[cur] = results[0]
		if len(results) > 1 {
			insertAt := cur + 1
			tail := append([]DiceResult{}, ui.diceLog[insertAt:]...)
			ui.diceLog = append(ui.diceLog[:insertAt], results[1:]...)
			ui.diceLog = append(ui.diceLog, tail...)
		}
		if len(ui.diceLog) > 200 {
			ui.diceLog = ui.diceLog[len(ui.diceLog)-200:]
		}
		ui.persistDiceHistory()
		ui.renderDiceList()
		lastIdx := cur + len(results) - 1
		if lastIdx >= len(ui.diceLog) {
			lastIdx = len(ui.diceLog) - 1
		}
		if lastIdx < 0 {
			lastIdx = 0
		}
		ui.dice.SetCurrentItem(lastIdx)
		ui.closeModal()
		ui.focusPanel(focusDice)
		if len(results) > 1 {
			ui.message = fmt.Sprintf("Tiro aggiornato in batch (%d).", len(results))
		} else {
			ui.message = "Tiro aggiornato."
		}
		ui.refreshDetail()
		ui.refreshStatus()
	})
}

func (ui *tviewUI) rerollSelectedDiceResult() {
	if len(ui.diceLog) == 0 {
		ui.message = "Nessun tiro da rilanciare."
		ui.refreshStatus()
		return
	}
	cur := ui.dice.GetCurrentItem()
	if cur < 0 || cur >= len(ui.diceLog) {
		cur = len(ui.diceLog) - 1
	}
	expr := strings.TrimSpace(ui.diceLog[cur].Expression)
	if expr == "" {
		ui.message = "Espressione tiro vuota."
		ui.refreshStatus()
		return
	}
	exprs, err := expandDiceRollInput(expr)
	if err != nil {
		ui.message = "Errore dadi: " + err.Error()
		ui.refreshStatus()
		return
	}
	results := make([]DiceResult, 0, len(exprs))
	for _, ex := range exprs {
		_, breakdown, rollErr := rollDiceExpression(ex)
		if rollErr != nil {
			ui.message = "Errore dadi: " + rollErr.Error()
			ui.refreshStatus()
			return
		}
		results = append(results, DiceResult{Expression: ex, Output: breakdown})
	}
	if len(results) == 0 {
		ui.message = "Nessun tiro da rilanciare."
		ui.refreshStatus()
		return
	}
	ui.diceLog[cur] = results[0]
	if len(results) > 1 {
		insertAt := cur + 1
		tail := append([]DiceResult{}, ui.diceLog[insertAt:]...)
		ui.diceLog = append(ui.diceLog[:insertAt], results[1:]...)
		ui.diceLog = append(ui.diceLog, tail...)
	}
	if len(ui.diceLog) > 200 {
		ui.diceLog = ui.diceLog[len(ui.diceLog)-200:]
	}
	ui.persistDiceHistory()
	ui.renderDiceList()
	lastIdx := cur + len(results) - 1
	if lastIdx >= len(ui.diceLog) {
		lastIdx = len(ui.diceLog) - 1
	}
	if lastIdx < 0 {
		lastIdx = 0
	}
	ui.dice.SetCurrentItem(lastIdx)
	if len(results) > 1 {
		ui.message = fmt.Sprintf("Tiro rilanciato in batch (%d).", len(results))
	} else {
		ui.message = "Tiro rilanciato."
	}
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) appendDiceLog(entry DiceResult) {
	ui.diceLog = append(ui.diceLog, entry)
	if ui.maxDiceLog > 0 && len(ui.diceLog) > ui.maxDiceLog {
		ui.diceLog = ui.diceLog[len(ui.diceLog)-ui.maxDiceLog:]
	}
	ui.persistDiceHistory()
	ui.renderDiceList()
	if len(ui.diceLog) > 0 {
		ui.dice.SetCurrentItem(len(ui.diceLog) - 1)
	}
}

func (ui *tviewUI) openMaxDiceLogInput() {
	cur := ""
	if ui.maxDiceLog > 0 {
		cur = strconv.Itoa(ui.maxDiceLog)
	}
	input := tview.NewInputField().
		SetLabel("Max voci log dadi (0=illimitato): ").
		SetText(cur).
		SetFieldWidth(6).
		SetAcceptanceFunc(func(text string, ch rune) bool {
			return ch >= '0' && ch <= '9'
		})
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			ui.pages.RemovePage("maxdicelog")
			ui.app.SetFocus(ui.dice)
			return
		}
		val := 0
		if t := strings.TrimSpace(input.GetText()); t != "" {
			if n, err := strconv.Atoi(t); err == nil && n >= 0 {
				val = n
			}
		}
		ui.maxDiceLog = val
		if val > 0 && len(ui.diceLog) > val {
			ui.diceLog = ui.diceLog[len(ui.diceLog)-val:]
			ui.renderDiceList()
		}
		ui.persistDiceHistory()
		if val == 0 {
			ui.message = "Log dadi: limite rimosso (illimitato)"
		} else {
			ui.message = fmt.Sprintf("Log dadi: limite impostato a %d voci", val)
		}
		ui.refreshStatus()
		ui.pages.RemovePage("maxdicelog")
		ui.app.SetFocus(ui.dice)
	})
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)
	ui.pages.AddPage("maxdicelog", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *tviewUI) renderDiceList() {
	ui.diceRenderLock = true
	defer func() { ui.diceRenderLock = false }()

	cur := 0
	if ui.dice != nil {
		cur = ui.dice.GetCurrentItem()
		ui.dice.Clear()
	}

	if len(ui.diceLog) == 0 {
		ui.dice.AddItem("(nessun tiro) premi 'a' per lanciare", "", 0, nil)
		ui.dice.SetCurrentItem(0)
		return
	}

	for i, row := range ui.diceLog {
		ui.dice.AddItem(fmt.Sprintf("%d) %s => %s", i+1, row.Expression, row.Output), "", 0, nil)
	}
	if cur >= len(ui.diceLog) {
		cur = len(ui.diceLog) - 1
	}
	if cur < 0 {
		cur = 0
	}
	ui.dice.SetCurrentItem(cur)
}

func (ui *tviewUI) deleteSelectedDiceResult() {
	if len(ui.diceLog) == 0 {
		ui.message = "Nessun tiro da eliminare."
		ui.refreshStatus()
		return
	}
	cur := ui.dice.GetCurrentItem()
	if cur < 0 || cur >= len(ui.diceLog) {
		cur = len(ui.diceLog) - 1
	}
	ui.diceLog = append(ui.diceLog[:cur], ui.diceLog[cur+1:]...)
	ui.persistDiceHistory()
	ui.renderDiceList()
	if len(ui.diceLog) == 0 {
		ui.message = "Storico dadi svuotato."
	} else {
		if cur >= len(ui.diceLog) {
			cur = len(ui.diceLog) - 1
		}
		ui.dice.SetCurrentItem(cur)
		ui.message = "Tiro eliminato."
	}
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) clearDiceResults() {
	if len(ui.diceLog) == 0 {
		ui.message = "Storico dadi già vuoto."
		ui.refreshStatus()
		return
	}
	ui.diceLog = nil
	ui.persistDiceHistory()
	ui.renderDiceList()
	ui.message = "Storico dadi svuotato."
	ui.refreshDetail()
	ui.refreshStatus()
}

func expandDiceRollInput(input string) ([]string, error) {
	return diceroll.ExpandRollInput(input)
}

func rollDiceExpression(expr string) (int, string, error) {
	return diceroll.RollExpression(expr)
}

// swadeSkillDie extracts the die type from a bonus string like "Combattere d8" → "d8".
func swadeSkillDie(bonus string) string {
	re := regexp.MustCompile(`[dD](\d+)`)
	m := re.FindString(bonus)
	if m == "" {
		return "d6"
	}
	return strings.ToLower(m)
}

// swadeDamageExpr resolves attribute references and makes all dice exploding.
// e.g. "For+d6" with Forza=d10 → "d10e+d6e"
func swadeDamageExpr(damage string, mon Monster) string {
	s := damage
	type attrSub struct {
		pattern string
		value   string
	}
	subs := []attrSub{
		{`(?i)\bforza\b`, mon.Attributes.Forza},
		{`(?i)\bfor\b`, mon.Attributes.Forza},
		{`(?i)\bstr\b`, mon.Attributes.Forza},
		{`(?i)\bagilita\b`, mon.Attributes.Agilita},
		{`(?i)\bagi\b`, mon.Attributes.Agilita},
		{`(?i)\bspirito\b`, mon.Attributes.Spirito},
		{`(?i)\bspi\b`, mon.Attributes.Spirito},
		{`(?i)\bvigore\b`, mon.Attributes.Vigore},
		{`(?i)\bvig\b`, mon.Attributes.Vigore},
		{`(?i)\bintelligenza\b`, mon.Attributes.Intelligenza},
		{`(?i)\bint\b`, mon.Attributes.Intelligenza},
	}
	for _, sub := range subs {
		if sub.value == "" || sub.value == "-" {
			continue
		}
		re := regexp.MustCompile(sub.pattern)
		s = re.ReplaceAllString(s, sub.value)
	}
	// Make all dice exploding: d\d+ not already ending in 'e'
	s = regexp.MustCompile(`d(\d+)(?:e)?`).ReplaceAllStringFunc(s, func(match string) string {
		if strings.HasSuffix(match, "e") {
			return match
		}
		return match + "e"
	})
	return s
}

func (ui *tviewUI) openEncounterAttackModal() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	entry := ui.encounter[idx]
	atk := entry.Monster.Attack
	if atk.Name == "" && atk.Damage == "" {
		ui.message = "Nessun attacco disponibile per " + entry.Monster.Name + "."
		ui.refreshStatus()
		return
	}

	hitDie := swadeSkillDie(atk.Bonus)
	dmgExpr := swadeDamageExpr(atk.Damage, entry.Monster)
	label := atk.Name
	if label == "" {
		label = "Attacco"
	}
	expr := fmt.Sprintf("%se >0 %s", hitDie, dmgExpr)

	list := tview.NewList()
	list.SetBorder(true).SetTitle(fmt.Sprintf(" A: %s ", entry.Monster.Name))
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSecondaryTextColor(tcell.ColorSilver)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(true)
	list.AddItem(label, expr, 0, nil)

	doRoll := func() {
		_, breakdown, err := rollDiceExpression(expr)
		if err != nil {
			ui.message = "Errore nel tiro: " + err.Error()
			ui.refreshStatus()
		} else {
			// Re-roll existing entry if the same expression is already in the dice log.
			existing := -1
			for i, dr := range ui.diceLog {
				if dr.Expression == expr {
					existing = i
					break
				}
			}
			if existing >= 0 {
				ui.diceLog[existing].Output = breakdown
				ui.renderDiceList()
				ui.dice.SetCurrentItem(existing)
			} else {
				ui.appendDiceLog(DiceResult{Expression: expr, Output: breakdown})
			}
		}
		ui.pages.RemovePage("encounter-attack")
		ui.app.SetFocus(ui.encList)
		ui.refreshStatus()
	}

	list.SetSelectedFunc(func(int, string, string, rune) { doRoll() })
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			ui.pages.RemovePage("encounter-attack")
			ui.app.SetFocus(ui.encList)
			return nil
		}
		return ev
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 5, 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)
	ui.pages.AddPage("encounter-attack", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *tviewUI) openEncounterTraitModal() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	entry := ui.encounter[idx]
	attrs := entry.Monster.Attributes

	type traitEntry struct {
		Name string
		Die  string
	}
	var traits []traitEntry
	for _, t := range []traitEntry{
		{"Agilità", attrs.Agilita},
		{"Intelligenza", attrs.Intelligenza},
		{"Spirito", attrs.Spirito},
		{"Forza", attrs.Forza},
		{"Vigore", attrs.Vigore},
	} {
		if t.Die != "" && t.Die != "-" {
			traits = append(traits, t)
		}
	}
	if len(traits) == 0 {
		ui.message = "Nessun tratto disponibile per " + entry.Monster.Name + "."
		ui.refreshStatus()
		return
	}

	buildTraitExpr := func(die string) string {
		s := strings.TrimSpace(die)
		// Make die exploding: "d6" → "d6e", "d12+6" → "d12e+6"
		result := regexp.MustCompile(`d(\d+)`).ReplaceAllString(s, "d${1}e")
		if entry.Monster.WildCard {
			return result + "+D6"
		}
		return result
	}

	list := tview.NewList()
	list.SetBorder(true).SetTitle(fmt.Sprintf(" K: Tiro di Tratto – %s ", entry.Monster.Name))
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSecondaryTextColor(tcell.ColorSilver)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(true)

	exprs := make([]string, len(traits))
	for i, t := range traits {
		expr := buildTraitExpr(t.Die)
		exprs[i] = expr
		list.AddItem(fmt.Sprintf("%s %s", t.Name, t.Die), expr, 0, nil)
	}

	rollTrait := func(i int) {
		expr := exprs[i]
		_, breakdown, err := rollDiceExpression(expr)
		if err != nil {
			ui.message = "Errore nel tiro: " + err.Error()
			ui.refreshStatus()
		} else {
			existing := -1
			for j, dr := range ui.diceLog {
				if dr.Expression == expr {
					existing = j
					break
				}
			}
			if existing >= 0 {
				ui.diceLog[existing].Output = breakdown
				ui.renderDiceList()
				ui.dice.SetCurrentItem(existing)
			} else {
				ui.appendDiceLog(DiceResult{Expression: expr, Output: breakdown})
			}
		}
		ui.pages.RemovePage("encounter-trait")
		ui.app.SetFocus(ui.encList)
		ui.refreshStatus()
	}

	list.SetSelectedFunc(func(i int, _ string, _ string, _ rune) { rollTrait(i) })
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			ui.pages.RemovePage("encounter-trait")
			ui.app.SetFocus(ui.encList)
			return nil
		}
		return ev
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, len(traits)+4, 0, true).
			AddItem(nil, 0, 1, false), 70, 0, true).
		AddItem(nil, 0, 1, false)
	ui.pages.AddPage("encounter-trait", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *tviewUI) currentNoteIndex() int {
	idx := ui.notesList.GetCurrentItem()
	if idx < 0 || idx >= len(ui.filteredNotes) {
		return -1
	}
	return ui.filteredNotes[idx]
}

func (ui *tviewUI) refreshNotes() {
	query := strings.ToLower(strings.TrimSpace(ui.notesSearch.GetText()))
	ui.filteredNotes = nil
	for i, n := range ui.notes {
		if query == "" || strings.Contains(strings.ToLower(n), query) {
			ui.filteredNotes = append(ui.filteredNotes, i)
		}
	}
	ui.notesList.Clear()
	for _, i := range ui.filteredNotes {
		label := ui.notes[i]
		if idx := strings.IndexByte(label, '\n'); idx >= 0 {
			label = label[:idx]
		}
		runes := []rune(label)
		if len(runes) > 60 {
			label = string(runes[:60]) + "…"
		}
		ui.notesList.AddItem(label, "", 0, nil)
	}
}

func (ui *tviewUI) persistNotes() {
	_ = saveNotes(notesFile, ui.notes)
	if ui.campaignName != "" {
		_ = saveNotes(filepath.Join(campaignDir(ui.campaignName), "notes.yml"), ui.notes)
	}
}

func (ui *tviewUI) quickNoteTimestamp() string {
	if ui.encInitModeActive && ui.encInitRound > 0 {
		return fmt.Sprintf("[R%d T%d] ", ui.encInitRound, ui.encInitTurnIndex+1)
	}
	if ui.encInitRound > 0 {
		return fmt.Sprintf("[R%d] ", ui.encInitRound)
	}
	return ""
}

func (ui *tviewUI) openQuickNoteInput() {
	if ui.modalVisible {
		return
	}
	ts := ui.quickNoteTimestamp()
	input := tview.NewInputField().SetLabel(fmt.Sprintf("Nota rapida %s> ", ts)).SetFieldWidth(50)
	frame := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(input, 1, 0, true)
	frame.SetBorder(true).SetTitle(" Nota Rapida (Invio: salva, Esc: annulla) ").SetTitleAlign(tview.AlignLeft)
	frame.SetBorderColor(tcell.ColorGold)
	frame.SetTitleColor(tcell.ColorGold)

	closeAdd := func(save bool) {
		ui.pages.RemovePage("quick-note")
		ui.modalVisible = false
		ui.app.SetFocus(ui.notesList)
		if !save {
			return
		}
		text := strings.TrimSpace(input.GetText())
		if text == "" {
			return
		}
		note := ts + text
		ui.notes = append(ui.notes, note)
		ui.persistNotes()
		ui.switchToCatalog("note")
		ui.refreshNotes()
		if len(ui.filteredNotes) > 0 {
			ui.notesList.SetCurrentItem(len(ui.filteredNotes) - 1)
		}
		ui.focusPanel(focusNotesList)
		ui.message = "Nota rapida aggiunta."
		ui.refreshDetail()
		ui.refreshStatus()
	}

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			closeAdd(true)
		} else {
			closeAdd(false)
		}
	})
	frame.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			closeAdd(false)
			return nil
		}
		return ev
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(frame, 3, 0, true).
			AddItem(nil, 0, 1, false), 70, 0, true).
		AddItem(nil, 0, 1, false)
	ui.modalVisible = true
	ui.pages.AddPage("quick-note", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *tviewUI) openAddNoteModal() {
	if ui.modalVisible {
		return
	}
	returnFocus := ui.app.GetFocus()
	editor := tview.NewTextArea()
	editor.SetBorder(true).SetTitle(" Nuova Nota (Ctrl+S salva, Esc annulla) ").SetTitleAlign(tview.AlignLeft)

	editor.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			ui.refreshStatus()
			return nil
		}
		if ev.Key() == tcell.KeyCtrlS {
			text := strings.TrimSpace(editor.GetText())
			if text == "" {
				ui.message = "Nota vuota: annullata."
				ui.closeModal()
				ui.app.SetFocus(returnFocus)
				ui.refreshStatus()
				return nil
			}
			ui.notes = append(ui.notes, text)
			ui.persistNotes()
			ui.switchToCatalog("note")
			ui.refreshNotes()
			if len(ui.filteredNotes) > 0 {
				ui.notesList.SetCurrentItem(len(ui.filteredNotes) - 1)
			}
			ui.closeModal()
			ui.focusPanel(focusNotesList)
			ui.message = "Nota aggiunta."
			ui.refreshDetail()
			ui.refreshStatus()
			return nil
		}
		return ev
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 2, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 2, 0, false).
			AddItem(editor, 0, 1, true).
			AddItem(nil, 2, 0, false), 0, 1, true).
		AddItem(nil, 2, 0, false)

	ui.modalVisible = true
	ui.modalName = "add_note"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(editor)
}

func (ui *tviewUI) openEditNoteModal() {
	idx := ui.currentNoteIndex()
	if idx < 0 || idx >= len(ui.notes) {
		ui.message = "Nessuna nota da modificare."
		ui.refreshStatus()
		return
	}
	if ui.modalVisible {
		return
	}
	returnFocus := ui.app.GetFocus()
	editor := tview.NewTextArea()
	editor.SetBorder(true).SetTitle(" Modifica Nota (Ctrl+S salva, Esc annulla) ").SetTitleAlign(tview.AlignLeft)
	editor.SetText(ui.notes[idx], true)

	editor.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			ui.refreshStatus()
			return nil
		}
		if ev.Key() == tcell.KeyCtrlS {
			text := strings.TrimSpace(editor.GetText())
			if text == "" {
				ui.message = "Nota vuota: modifica annullata."
				ui.closeModal()
				ui.app.SetFocus(returnFocus)
				ui.refreshStatus()
				return nil
			}
			ui.notes[idx] = text
			ui.persistNotes()
			ui.refreshNotes()
			for li, ni := range ui.filteredNotes {
				if ni == idx {
					ui.notesList.SetCurrentItem(li)
					break
				}
			}
			ui.closeModal()
			ui.focusPanel(focusNotesList)
			ui.message = "Nota aggiornata."
			ui.refreshDetail()
			ui.refreshStatus()
			return nil
		}
		return ev
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 2, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 2, 0, false).
			AddItem(editor, 0, 1, true).
			AddItem(nil, 2, 0, false), 0, 1, true).
		AddItem(nil, 2, 0, false)

	ui.modalVisible = true
	ui.modalName = "edit_note"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(editor)
}

func (ui *tviewUI) deleteSelectedNote() {
	idx := ui.currentNoteIndex()
	if idx < 0 || idx >= len(ui.notes) {
		ui.message = "Nessuna nota da eliminare."
		ui.refreshStatus()
		return
	}
	ui.notes = append(ui.notes[:idx], ui.notes[idx+1:]...)
	ui.persistNotes()
	ui.refreshNotes()
	ui.message = "Nota eliminata."
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) persistDiceHistory() {
	_ = saveDiceHistory(diceHistoryFile, ui.diceLog, ui.maxDiceLog)
}

func (ui *tviewUI) persistDiceMacros() {
	if data, err := common.SaveDiceMacros(ui.diceMacros); err == nil {
		_ = os.WriteFile(diceMacrosFile, data, 0o644)
	}
}

func (ui *tviewUI) openDiceMacroModal() {
	if ui.modalVisible {
		return
	}
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Macro Dadi (Invio:lancia, a:aggiungi, e:rinomina, d:elimina, Esc:chiudi) ")
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(false)

	render := func() {
		cur := list.GetCurrentItem()
		list.Clear()
		if len(ui.diceMacros) == 0 {
			list.AddItem("(nessuna macro — premi 'a' per aggiungerne una)", "", 0, nil)
		} else {
			for _, m := range ui.diceMacros {
				list.AddItem(fmt.Sprintf("%-20s  %s", m.Name, m.Expr), "", 0, nil)
			}
		}
		if cur >= 0 && cur < list.GetItemCount() {
			list.SetCurrentItem(cur)
		}
	}
	render()

	closeModal := func() {
		ui.pages.RemovePage("dice-macros")
		ui.modalVisible = false
		ui.modalName = ""
		ui.app.SetFocus(ui.dice)
	}

	openAddMacro := func(editIdx int) {
		nameInput := tview.NewInputField().SetLabel("Nome: ").SetFieldWidth(20)
		exprInput := tview.NewInputField().SetLabel("Espressione: ").SetFieldWidth(20)
		if editIdx >= 0 && editIdx < len(ui.diceMacros) {
			nameInput.SetText(ui.diceMacros[editIdx].Name)
			exprInput.SetText(ui.diceMacros[editIdx].Expr)
		}
		doClose := func() {
			ui.pages.RemovePage("dice-macro-edit")
			ui.app.SetFocus(list)
		}
		doSave := func() {
			name := strings.TrimSpace(nameInput.GetText())
			expr := strings.TrimSpace(exprInput.GetText())
			if name != "" && expr != "" {
				if editIdx >= 0 && editIdx < len(ui.diceMacros) {
					ui.diceMacros[editIdx] = common.DiceMacro{Name: name, Expr: expr}
				} else {
					ui.diceMacros = append(ui.diceMacros, common.DiceMacro{Name: name, Expr: expr})
				}
				ui.persistDiceMacros()
				render()
				if editIdx < 0 {
					list.SetCurrentItem(len(ui.diceMacros) - 1)
				}
			}
			doClose()
		}
		nameInput.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			switch ev.Key() {
			case tcell.KeyTab:
				ui.app.SetFocus(exprInput)
				return nil
			case tcell.KeyBacktab:
				ui.app.SetFocus(exprInput) // solo 2 campi: wrap
				return nil
			case tcell.KeyEscape:
				doClose()
				return nil
			case tcell.KeyEnter:
				doSave()
				return nil
			}
			return ev
		})
		exprInput.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			switch ev.Key() {
			case tcell.KeyTab:
				ui.app.SetFocus(nameInput)
				return nil
			case tcell.KeyBacktab:
				ui.app.SetFocus(nameInput) // solo 2 campi: wrap
				return nil
			case tcell.KeyEscape:
				doClose()
				return nil
			case tcell.KeyEnter:
				doSave()
				return nil
			}
			return ev
		})
		frame := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nameInput, 1, 0, true).
			AddItem(exprInput, 1, 0, false)
		frame.SetBorder(true).SetTitle(" Macro (Tab: campo, Invio: salva, Esc: annulla) ").SetTitleAlign(tview.AlignLeft)
		frame.SetBorderColor(tcell.ColorGold)
		frame.SetTitleColor(tcell.ColorGold)
		editModal := tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(frame, 5, 0, true).
				AddItem(nil, 0, 1, false), 55, 0, true).
			AddItem(nil, 0, 1, false)
		ui.pages.AddPage("dice-macro-edit", editModal, true, true)
		ui.app.SetFocus(nameInput)
	}

	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEscape:
			closeModal()
			return nil
		case tcell.KeyEnter:
			idx := list.GetCurrentItem()
			if idx < 0 || idx >= len(ui.diceMacros) {
				return nil
			}
			macro := ui.diceMacros[idx]
			closeModal()
			total, breakdown, err := rollDiceExpression(macro.Expr)
			if err != nil {
				ui.message = fmt.Sprintf("Errore macro '%s': %v", macro.Name, err)
				ui.refreshStatus()
				return nil
			}
			result := fmt.Sprintf("%s [%s] = %d", macro.Name, macro.Expr, total)
			marker := "[" + macro.Expr + "]"
			existingIdx := -1
			for i := len(ui.diceLog) - 1; i >= 0; i-- {
				if strings.Contains(ui.diceLog[i].Expression, marker) {
					existingIdx = i
					break
				}
			}
			if existingIdx >= 0 {
				ui.diceLog[existingIdx] = DiceResult{Expression: result, Output: breakdown}
				ui.renderDiceList()
				ui.dice.SetCurrentItem(existingIdx)
			} else {
				ui.diceLog = append(ui.diceLog, DiceResult{Expression: result, Output: breakdown})
				ui.renderDiceList()
				ui.dice.SetCurrentItem(len(ui.diceLog) - 1)
			}
			ui.focusPanel(focusDice)
			ui.message = fmt.Sprintf("Macro '%s': %d", macro.Name, total)
			ui.refreshStatus()
			ui.refreshDetail()
			return nil
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'a':
				openAddMacro(-1)
				return nil
			case 'e':
				idx := list.GetCurrentItem()
				if idx >= 0 && idx < len(ui.diceMacros) {
					openAddMacro(idx)
				}
				return nil
			case 'd':
				idx := list.GetCurrentItem()
				if idx >= 0 && idx < len(ui.diceMacros) {
					ui.diceMacros = append(ui.diceMacros[:idx], ui.diceMacros[idx+1:]...)
					ui.persistDiceMacros()
					render()
				}
				return nil
			}
		}
		return ev
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 0, 1, true).
			AddItem(nil, 0, 1, false), 70, 0, true).
		AddItem(nil, 0, 1, false)
	ui.modalVisible = true
	ui.modalName = "dice-macros"
	ui.pages.AddPage("dice-macros", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *tviewUI) openGotoModal() {
	if ui.gotoVisible {
		return
	}
	ui.gotoVisible = true

	type gotoEntry struct {
		shortcut rune
		label    string
		action   func()
	}

	entries := []gotoEntry{
		{'0', "Dadi", func() { ui.focusPanel(focusDice) }},
		{'1', "PNG", func() { ui.focusPanel(focusPNG) }},
		{'2', "Encounter", func() { ui.focusPanel(focusEncounter) }},
		{'3', "Mostri", func() {
			ui.switchToCatalog("mostri")
			ui.focusPanel(focusMonList)
		}},
		{'4', "Equipaggiamento", func() {
			ui.switchToCatalog("equipaggiamento")
			ui.focusPanel(focusEqList)
		}},
		{'5', "Regole", func() {
			ui.switchToCatalog("regole")
			ui.focusPanel(focusClassList)
		}},
		{'6', "Note", func() {
			ui.switchToCatalog("note")
			ui.focusPanel(focusNotesList)
		}},
	}

	list := tview.NewList()
	list.SetBorder(true).SetTitle(" Vai a pannello (Esc per chiudere) ")
	list.ShowSecondaryText(false)
	list.SetHighlightFullLine(true)

	closeModal := func() {
		ui.gotoVisible = false
		ui.pages.RemovePage("goto")
	}

	for _, e := range entries {
		e := e
		list.AddItem(fmt.Sprintf("[black:gold]%c[-:-]  %s", e.shortcut, e.label), "", e.shortcut, func() {
			closeModal()
			e.action()
		})
	}

	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEsc {
			closeModal()
			return nil
		}
		return ev
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(list, 40, 0, true).
			AddItem(nil, 0, 1, false), len(entries)+4, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddAndSwitchToPage("goto", modal, true)
	ui.app.SetFocus(list)
}

func (ui *tviewUI) switchToCatalog(mode string) {
	if ui.catalogMode == mode {
		return
	}
	ui.catalogMode = mode
	ui.catalogPanel.SwitchToPage(mode)
	ui.refreshCatalogTitles()
	switch mode {
	case "equipaggiamento":
		ui.message = "Catalogo: Equipaggiamento"
	case "regole":
		ui.message = "Catalogo: Regole"
	case "note":
		ui.message = "Catalogo: Note"
	default:
		ui.message = "Catalogo: Mostri"
	}
}

// saveCampaignState saves pngs, encounter and dice history to ~/.lazysw/<name>/.
func (ui *tviewUI) saveCampaignState(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		ui.message = "Nome campagna non valido."
		ui.refreshStatus()
		return
	}
	dir := campaignDir(name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		ui.message = "Errore campagna: " + err.Error()
		ui.refreshStatus()
		return
	}
	if err := savePNGList(filepath.Join(dir, "pngs.yml"), ui.pngs, selectedPNGName(ui.pngs, ui.selected)); err != nil {
		ui.message = "Errore salvataggio PNG: " + err.Error()
		ui.refreshStatus()
		return
	}
	encEntries := make([]struct {
		Name             string         `yaml:"name"`
		Wounds           int            `yaml:"wounds"`
		PF               int            `yaml:"pf"`
		InitiativeCard   string         `yaml:"initiative_card,omitempty"`
		LegacyInitiative int            `yaml:"initiative,omitempty"`
		HasInit          bool           `yaml:"has_initiative,omitempty"`
		Conditions       map[string]int `yaml:"conditions,omitempty"`
		Stress           int            `yaml:"stress,omitempty"`
		BaseStress       int            `yaml:"base_stress,omitempty"`
		Disabled         bool           `yaml:"disabled,omitempty"`
	}, 0, len(ui.encounter))
	for _, e := range ui.encounter {
		base := encounterWoundsCap(e)
		encEntries = append(encEntries, struct {
			Name             string         `yaml:"name"`
			Wounds           int            `yaml:"wounds"`
			PF               int            `yaml:"pf"`
			InitiativeCard   string         `yaml:"initiative_card,omitempty"`
			LegacyInitiative int            `yaml:"initiative,omitempty"`
			HasInit          bool           `yaml:"has_initiative,omitempty"`
			Conditions       map[string]int `yaml:"conditions,omitempty"`
			Stress           int            `yaml:"stress,omitempty"`
			BaseStress       int            `yaml:"base_stress,omitempty"`
			Disabled         bool           `yaml:"disabled,omitempty"`
		}{Name: e.Monster.Name, Wounds: e.Wounds, PF: base, InitiativeCard: e.InitiativeCard, HasInit: e.HasInit, Conditions: cloneStringIntMap(e.Conditions), Disabled: e.Disabled})
	}
	if err := saveEncounter(filepath.Join(dir, "encounter.yml"), encEntries); err != nil {
		ui.message = "Errore salvataggio encounter: " + err.Error()
		ui.refreshStatus()
		return
	}
	if err := saveDiceHistory(filepath.Join(dir, "dice_history.yml"), ui.diceLog, ui.maxDiceLog); err != nil {
		ui.message = "Errore salvataggio dadi: " + err.Error()
		ui.refreshStatus()
		return
	}
	_ = saveNotes(filepath.Join(dir, "notes.yml"), ui.notes)
	ui.campaignName = name
	ui.message = fmt.Sprintf("Campagna '%s' salvata.", name)
	ui.refreshStatus()
}

// loadCampaignState loads pngs, encounter and dice history from ~/.lazysw/<name>/.
func (ui *tviewUI) loadCampaignState(name string) {
	dir := campaignDir(name)
	pngs, selectedName, err := loadPNGList(filepath.Join(dir, "pngs.yml"))
	if err != nil {
		pngs = []PNG{}
	}
	selectedIdx := -1
	for i, p := range pngs {
		if p.Name == selectedName {
			selectedIdx = i
			break
		}
	}
	enc, err := loadEncounter(filepath.Join(dir, "encounter.yml"), ui.monsters)
	if err != nil {
		enc = []EncounterEntry{}
	}
	diceLog, maxDiceLog, err := loadDiceHistory(filepath.Join(dir, "dice_history.yml"))
	if err != nil {
		diceLog = []DiceResult{}
		maxDiceLog = 0
	}
	ui.pngs = pngs
	ui.selected = selectedIdx
	ui.encounter = enc
	ui.diceLog = diceLog
	ui.maxDiceLog = maxDiceLog
	ui.campaignName = name
	ui.refreshPNGs()
	ui.refreshEncounter()
	ui.renderDiceList()
	if ns, err := loadNotes(filepath.Join(dir, "notes.yml")); err == nil {
		ui.notes = ns
		ui.refreshNotes()
	}
	ui.message = fmt.Sprintf("Campagna '%s' caricata.", name)
	ui.refreshStatus()
}

func (ui *tviewUI) resetCampaignState() {
	ui.pngs = []PNG{}
	ui.selected = -1
	ui.encounter = []EncounterEntry{}
	ui.diceLog = []DiceResult{}
	ui.notes = []string{}
	ui.campaignName = ""
	ui.refreshPNGs()
	ui.refreshEncounter()
	ui.renderDiceList()
	ui.refreshNotes()
}

// showSaveCampaignModal shows a form to name and save the current campaign.
func (ui *tviewUI) showSaveCampaignModal() {
	if ui.modalVisible {
		return
	}
	returnFocus := ui.app.GetFocus()

	form := tview.NewForm()
	form.AddInputField("Nome campagna", ui.campaignName, 40, nil, nil)
	save := func() {
		name := strings.TrimSpace(form.GetFormItemByLabel("Nome campagna").(*tview.InputField).GetText())
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		if name != "" {
			ui.saveCampaignState(name)
		}
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlO {
			save()
			return nil
		}
		return event
	})
	form.AddButton("Salva", save)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
	})
	form.SetBorder(true).SetTitle(" Salva Campagna ").SetTitleAlign(tview.AlignLeft)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 50, 0, true).
			AddItem(nil, 0, 1, false), 7, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = save
	ui.modalVisible = true
	ui.modalName = "saveCampaign"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

// showCampaignManagerModal shows a navigable list of campaigns with load/rename/delete actions.
func (ui *tviewUI) showCampaignManagerModal() {
	if ui.modalVisible {
		return
	}
	ui.openCampaignList()
}

func (ui *tviewUI) openCampaignList() {
	campaigns, _ := listCampaigns()

	campaignList := tview.NewList().ShowSecondaryText(false)
	for _, c := range campaigns {
		label := c
		if c == ui.campaignName {
			label = "* " + c
		}
		campaignList.AddItem(label, c, 0, nil)
	}
	if len(campaigns) == 0 {
		campaignList.AddItem("(nessuna campagna)", "", 0, nil)
	}

	hints := tview.NewTextView().
		SetDynamicColors(true).
		SetText(" [black:gold]Invio[-:-] Carica  [black:gold]r[-:-] Rinomina  [black:gold]d[-:-] Elimina  [black:gold]n[-:-] Nuova  [black:gold]Esc[-:-] Chiudi")

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(campaignList, 0, 1, true).
		AddItem(hints, 1, 0, false)
	container.SetBorder(true).SetTitle(" Gestione Campagne [Ctrl+O] ").SetTitleAlign(tview.AlignLeft)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(container, 60, 0, true).
			AddItem(nil, 0, 1, false), 20, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalVisible = true
	ui.modalName = "campaignManager"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(campaignList)

	campaignList.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		// 'n' works regardless of whether any campaigns exist
		if ev.Key() == tcell.KeyRune && (ev.Rune() == 'n' || ev.Rune() == 'N') {
			ui.closeModal()
			ui.resetCampaignState()
			ui.showSaveCampaignModal()
			return nil
		}
		if len(campaigns) == 0 {
			return ev
		}
		idx := campaignList.GetCurrentItem()
		if idx < 0 || idx >= len(campaigns) {
			return ev
		}
		selected := campaigns[idx]

		switch ev.Key() {
		case tcell.KeyEnter:
			ui.closeModal()
			ui.loadCampaignState(selected)
			return nil
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'r', 'R':
				ui.closeModal()
				ui.showRenameCampaignModal(selected)
				return nil
			case 'd', 'D':
				ui.closeModal()
				ui.showDeleteCampaignModal(selected)
				return nil
			}
		}
		return ev
	})
}

// showRenameCampaignModal asks for a new name and renames the campaign folder.
func (ui *tviewUI) showRenameCampaignModal(oldName string) {
	if ui.modalVisible {
		return
	}

	form := tview.NewForm()
	form.AddInputField("Nuovo nome", oldName, 40, nil, nil)
	rinomina := func() {
		newName := strings.TrimSpace(form.GetFormItemByLabel("Nuovo nome").(*tview.InputField).GetText())
		ui.closeModal()
		if newName != "" && newName != oldName {
			if err := renameCampaign(oldName, newName); err != nil {
				ui.message = "Errore rinomina: " + err.Error()
				ui.refreshStatus()
				return
			}
			if ui.campaignName == oldName {
				ui.campaignName = newName
			}
			ui.message = fmt.Sprintf("Campagna rinominata: '%s' → '%s'", oldName, newName)
			ui.refreshStatus()
		}
		ui.openCampaignList()
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlO {
			rinomina()
			return nil
		}
		return event
	})
	form.AddButton("Rinomina", rinomina)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.openCampaignList()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.openCampaignList()
	})
	form.SetBorder(true).SetTitle(fmt.Sprintf(" Rinomina '%s' ", oldName)).SetTitleAlign(tview.AlignLeft)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 52, 0, true).
			AddItem(nil, 0, 1, false), 7, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = rinomina
	ui.modalVisible = true
	ui.modalName = "campaignRename"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

// showDeleteCampaignModal asks for confirmation before deleting a campaign.
func (ui *tviewUI) showDeleteCampaignModal(name string) {
	if ui.modalVisible {
		return
	}

	form := tview.NewForm()
	elimina := func() {
		ui.closeModal()
		if err := deleteCampaign(name); err != nil {
			ui.message = "Errore eliminazione: " + err.Error()
			ui.refreshStatus()
		} else {
			if ui.campaignName == name {
				ui.campaignName = ""
			}
			ui.message = fmt.Sprintf("Campagna '%s' eliminata.", name)
			ui.refreshStatus()
		}
		ui.openCampaignList()
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlO {
			elimina()
			return nil
		}
		return event
	})
	form.AddButton("Sì, Elimina", elimina)
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.openCampaignList()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.openCampaignList()
	})

	info := tview.NewTextView().SetText(fmt.Sprintf("Eliminare la campagna '%s'?\nQuesta operazione non è reversibile.", name))
	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(info, 2, 0, false).
		AddItem(form, 0, 1, true)
	container.SetBorder(true).SetTitle(" Elimina Campagna ").SetTitleAlign(tview.AlignLeft)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(container, 52, 0, true).
			AddItem(nil, 0, 1, false), 9, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalConfirmFunc = elimina
	ui.modalVisible = true
	ui.modalName = "campaignDelete"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItemByLabel(""))
	// Focus the first button
	if h := form.GetButton(0); h != nil {
		ui.app.SetFocus(form)
	}
}

// ── Feature 1: g+number navigation on list panels ────────────────────────────

// focusedListWidget returns the *tview.List if focus is directly on a list widget.
func (ui *tviewUI) focusedListWidget(focus tview.Primitive) *tview.List {
	switch focus {
	case ui.pngList:
		return ui.pngList
	case ui.encList:
		return ui.encList
	case ui.monList:
		return ui.monList
	case ui.eqList:
		return ui.eqList
	case ui.cardList:
		return ui.cardList
	case ui.classList:
		return ui.classList
	}
	return nil
}

// jumpList sets the current item in list to idx (clamped), updates message.
func (ui *tviewUI) jumpList(list *tview.List, idx int) {
	if list == nil {
		return
	}
	n := list.GetItemCount()
	if n <= 0 {
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	list.SetCurrentItem(idx)
	ui.message = fmt.Sprintf("Jump: riga %d", idx+1)
	ui.refreshDetail()
	ui.refreshStatus()
}

// ── Feature 2: panel prefix with timer ───────────────────────────────────────

// listForPanelDigit returns the primary list widget for digits 0-5.
func (ui *tviewUI) listForPanelDigit(digit int) tview.Primitive {
	switch digit {
	case 0:
		return ui.dice
	case 1:
		return ui.pngList
	case 2:
		return ui.encList
	case 3:
		return ui.monList
	case 4:
		return ui.eqList
	case 5:
		return ui.classList
	case 6:
		return ui.notesList
	}
	return ui.app.GetFocus()
}

// ensureCatalogForDigit switches the catalog page for digits 3-5 without moving focus.
func (ui *tviewUI) ensureCatalogForDigit(digit int) {
	switch digit {
	case 3:
		if ui.catalogMode != "mostri" {
			ui.catalogMode = "mostri"
			ui.catalogPanel.SwitchToPage("mostri")
			ui.refreshCatalogTitles()
		}
	case 4:
		if ui.catalogMode != "equipaggiamento" {
			ui.catalogMode = "equipaggiamento"
			ui.catalogPanel.SwitchToPage("equipaggiamento")
			ui.refreshCatalogTitles()
		}
	case 5:
		if ui.catalogMode != "regole" {
			ui.catalogMode = "regole"
			ui.catalogPanel.SwitchToPage("regole")
			ui.refreshCatalogTitles()
		}
	case 6:
		if ui.catalogMode != "note" {
			ui.catalogMode = "note"
			ui.catalogPanel.SwitchToPage("note")
			ui.refreshCatalogTitles()
		}
	}
}

// startPanelPrefix activates the panel-prefix timer for the given digit.
func (ui *tviewUI) startPanelPrefix(digit int) {
	if ui.panelPrefixTimer != nil {
		ui.panelPrefixTimer.Stop()
	}
	ui.panelPrefixActive = true
	ui.panelPrefixDigit = digit
	ui.message = fmt.Sprintf("Prefisso pannello: %d — shortcut o 500ms per focus", digit)
	ui.refreshStatus()
	ui.panelPrefixTimer = time.AfterFunc(500*time.Millisecond, func() {
		ui.app.QueueUpdateDraw(func() {
			if ui.panelPrefixActive && ui.panelPrefixDigit == digit {
				ui.panelPrefixActive = false
				ui.panelPrefixTimer = nil
				ui.activatePanelForDigit(digit)
				ui.message = ""
				ui.refreshStatus()
			}
		})
	})
}

// cancelPanelPrefix stops the timer and clears prefix state.
func (ui *tviewUI) cancelPanelPrefix() {
	if ui.panelPrefixTimer != nil {
		ui.panelPrefixTimer.Stop()
		ui.panelPrefixTimer = nil
	}
	ui.panelPrefixActive = false
	ui.message = "Prefisso annullato."
	ui.refreshStatus()
}

// activatePanelForDigit focuses the panel corresponding to digit 0-5.
func (ui *tviewUI) activatePanelForDigit(digit int) {
	switch digit {
	case 0:
		ui.focusPanel(focusDice)
	case 1:
		ui.focusPanel(focusPNG)
	case 2:
		ui.focusPanel(focusEncounter)
	case 3:
		ui.switchToCatalog("mostri")
		ui.focusPanel(focusMonList)
	case 4:
		ui.switchToCatalog("equipaggiamento")
		ui.focusPanel(focusEqList)
	case 5:
		ui.switchToCatalog("regole")
		ui.focusPanel(focusClassList)
	case 6:
		ui.switchToCatalog("note")
		ui.focusPanel(focusNotesList)
	}
}

// ── Feature 3: detail panel vim navigation ───────────────────────────────────

// moveDetailCursor moves the cursor by delta lines and re-renders.
func (ui *tviewUI) moveDetailCursor(delta int) {
	lines := strings.Split(ui.detailRaw, "\n")
	ui.detailCursorLine += delta
	if ui.detailCursorLine < 0 {
		ui.detailCursorLine = 0
	}
	if ui.detailCursorLine >= len(lines) {
		ui.detailCursorLine = len(lines) - 1
	}
	if ui.detailCursorLine < 0 {
		ui.detailCursorLine = 0
	}
	ui.renderDetail()
	ui.detail.ScrollTo(ui.detailCursorLine, 0)
}

// scrollDetailHalfPage scrolls the detail panel by half a page and moves cursor.
func (ui *tviewUI) scrollDetailHalfPage(direction int) {
	_, _, _, h := ui.detail.GetInnerRect()
	if h <= 0 {
		h = 24
	}
	step := h / 2
	if step < 1 {
		step = 1
	}
	row, col := ui.detail.GetScrollOffset()
	row += direction * step
	if row < 0 {
		row = 0
	}
	ui.detail.ScrollTo(row, col)
	ui.detailCursorLine += direction * step
	lines := strings.Split(ui.detailRaw, "\n")
	if ui.detailCursorLine < 0 {
		ui.detailCursorLine = 0
	}
	if ui.detailCursorLine >= len(lines) {
		ui.detailCursorLine = len(lines) - 1
	}
	ui.renderDetail()
}

// rollDiceFromDetailCursorLine finds the first dice expression on the cursor line and rolls it.
func (ui *tviewUI) rollDiceFromDetailCursorLine() {
	lines := strings.Split(ui.detailRaw, "\n")
	if ui.detailCursorLine < 0 || ui.detailCursorLine >= len(lines) {
		ui.message = "Nessuna riga cursore."
		ui.refreshStatus()
		return
	}
	line := lines[ui.detailCursorLine]
	re := regexp.MustCompile(`(?i)\d*[dD]\d+([+-]\d+)?`)
	expr := re.FindString(line)
	if expr == "" {
		ui.message = "Nessun dado trovato sulla riga."
		ui.refreshStatus()
		return
	}
	_, breakdown, err := rollDiceExpression(expr)
	if err != nil {
		ui.message = "Errore dado: " + err.Error()
		ui.refreshStatus()
		return
	}
	ui.appendDiceLog(DiceResult{Expression: expr, Output: breakdown})
	ui.renderDiceList()
	ui.message = fmt.Sprintf("Dado %s: %s", expr, breakdown)
	ui.refreshStatus()
}

// ── Copy/paste for encounter & PNG panels ────────────────────────────────────

// pngEntryBaseName strips the trailing " #N" from a PNG name.
func pngEntryBaseName(name string) string {
	re := regexp.MustCompile(`^(.*?) #\d+$`)
	if m := re.FindStringSubmatch(name); m != nil {
		return m[1]
	}
	return name
}

// nextPNGName returns "<base> #N" where N is one more than the highest existing #N for that base.
func nextPNGName(base string, names []string) string {
	max := 1
	for _, n := range names {
		if n == base || strings.HasPrefix(n, base+" #") {
			rest := strings.TrimPrefix(n, base+" #")
			if v, err := strconv.Atoi(rest); err == nil && v > max {
				max = v
			}
		}
	}
	return base + " #" + strconv.Itoa(max+1)
}

func (ui *tviewUI) yankCurrentPNG() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG da copiare."
		ui.refreshStatus()
		return
	}
	p := ui.pngs[ui.selected]
	ui.clipPNG = &p
	ui.clipEncounter = nil
	ui.message = fmt.Sprintf("PNG copiato: %s", p.Name)
	ui.refreshStatus()
}

func (ui *tviewUI) pasteClipPNG() {
	if ui.clipPNG == nil {
		ui.message = "Clipboard vuoto (y per copiare)."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	names := make([]string, len(ui.pngs))
	for i, p := range ui.pngs {
		names[i] = p.Name
	}
	base := pngEntryBaseName(ui.clipPNG.Name)
	newName := nextPNGName(base, names)
	newPNG := *ui.clipPNG
	newPNG.Name = newName
	// Insert after current position
	pos := ui.selected + 1
	if pos < 0 || pos > len(ui.pngs) {
		pos = len(ui.pngs)
	}
	ui.pngs = append(ui.pngs, PNG{})
	copy(ui.pngs[pos+1:], ui.pngs[pos:])
	ui.pngs[pos] = newPNG
	ui.selected = pos
	ui.persistPNGs()
	ui.message = fmt.Sprintf("PNG incollato: %s", newName)
	ui.refreshPNGs()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) yankCurrentEncounterEntry() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Nessun elemento encounter da copiare."
		ui.refreshStatus()
		return
	}
	e := ui.encounter[idx]
	ui.clipEncounter = &e
	ui.clipPNG = nil
	ui.message = fmt.Sprintf("Encounter copiato: %s", e.Monster.Name)
	ui.refreshStatus()
}

func (ui *tviewUI) pasteClipEncounterEntry() {
	if ui.clipEncounter == nil {
		ui.message = "Clipboard vuoto (y per copiare)."
		ui.refreshStatus()
		return
	}
	ui.pushUndo()
	names := make([]string, len(ui.encounter))
	for i, e := range ui.encounter {
		names[i] = e.Monster.Name
	}
	base := pngEntryBaseName(ui.clipEncounter.Monster.Name)
	newName := nextPNGName(base, names)
	newEntry := *ui.clipEncounter
	newEntry.Monster.Name = newName
	newEntry.Wounds = 0
	newEntry.Conditions = cloneStringIntMap(ui.clipEncounter.Conditions)
	newEntry.HasInit = false
	newEntry.InitiativeCard = ""
	// Insert after current item
	cur := ui.encList.GetCurrentItem()
	pos := cur + 1
	if pos < 0 || pos > len(ui.encounter) {
		pos = len(ui.encounter)
	}
	ui.encounter = append(ui.encounter, EncounterEntry{})
	copy(ui.encounter[pos+1:], ui.encounter[pos:])
	ui.encounter[pos] = newEntry
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Encounter incollato: %s", newName)
	ui.refreshEncounter()
	ui.encList.SetCurrentItem(pos)
	ui.refreshDetail()
	ui.refreshStatus()
}

// ── Undo / Redo ───────────────────────────────────────────────────────────────

func deepCopyEncounter(src []EncounterEntry) []EncounterEntry {
	if src == nil {
		return nil
	}
	dst := make([]EncounterEntry, len(src))
	copy(dst, src)
	for i, e := range src {
		dst[i].Conditions = cloneStringIntMap(e.Conditions)
	}
	return dst
}

func (ui *tviewUI) pushUndo() {
	pngsCopy := make([]PNG, len(ui.pngs))
	copy(pngsCopy, ui.pngs)
	snap := undoSnapshot{
		pngs:      pngsCopy,
		selected:  ui.selected,
		encounter: deepCopyEncounter(ui.encounter),
	}
	ui.undoStack = append(ui.undoStack, snap)
	if len(ui.undoStack) > 50 {
		ui.undoStack = ui.undoStack[len(ui.undoStack)-50:]
	}
	ui.redoStack = nil
}

func (ui *tviewUI) performUndo() {
	if len(ui.undoStack) == 0 {
		ui.message = "Niente da annullare."
		ui.refreshStatus()
		return
	}
	// Save current state to redo stack
	pngsCopy := make([]PNG, len(ui.pngs))
	copy(pngsCopy, ui.pngs)
	ui.redoStack = append(ui.redoStack, undoSnapshot{
		pngs:      pngsCopy,
		selected:  ui.selected,
		encounter: deepCopyEncounter(ui.encounter),
	})
	if len(ui.redoStack) > 50 {
		ui.redoStack = ui.redoStack[len(ui.redoStack)-50:]
	}
	// Restore previous state
	prev := ui.undoStack[len(ui.undoStack)-1]
	ui.undoStack = ui.undoStack[:len(ui.undoStack)-1]
	ui.pngs = prev.pngs
	ui.selected = prev.selected
	ui.encounter = prev.encounter
	_ = savePNGList(dataFile, ui.pngs, selectedPNGName(ui.pngs, ui.selected))
	ui.persistEncounter()
	ui.refreshAll()
	ui.message = fmt.Sprintf("Annullato. (undo: %d, redo: %d)", len(ui.undoStack), len(ui.redoStack))
	ui.refreshStatus()
}

func (ui *tviewUI) performRedo() {
	if len(ui.redoStack) == 0 {
		ui.message = "Niente da ripristinare."
		ui.refreshStatus()
		return
	}
	// Save current state to undo stack (without clearing redo)
	pngsCopy := make([]PNG, len(ui.pngs))
	copy(pngsCopy, ui.pngs)
	ui.undoStack = append(ui.undoStack, undoSnapshot{
		pngs:      pngsCopy,
		selected:  ui.selected,
		encounter: deepCopyEncounter(ui.encounter),
	})
	if len(ui.undoStack) > 50 {
		ui.undoStack = ui.undoStack[len(ui.undoStack)-50:]
	}
	// Restore redo state
	next := ui.redoStack[len(ui.redoStack)-1]
	ui.redoStack = ui.redoStack[:len(ui.redoStack)-1]
	ui.pngs = next.pngs
	ui.selected = next.selected
	ui.encounter = next.encounter
	_ = savePNGList(dataFile, ui.pngs, selectedPNGName(ui.pngs, ui.selected))
	ui.persistEncounter()
	ui.refreshAll()
	ui.message = fmt.Sprintf("Ripristinato. (undo: %d, redo: %d)", len(ui.undoStack), len(ui.redoStack))
	ui.refreshStatus()
}

func swadeSnapshotDesc(s undoSnapshot) string {
	return fmt.Sprintf("%d PNG, %d in scontro", len(s.pngs), len(s.encounter))
}

func (ui *tviewUI) openUndoHistoryPanel() {
	if ui.modalVisible {
		return
	}

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Storico modifiche (u:annulla, r:ripristina, Invio:vai, Esc:chiudi) ")
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(false)

	captureCurrentSnap := func() undoSnapshot {
		pngsCopy := make([]PNG, len(ui.pngs))
		copy(pngsCopy, ui.pngs)
		return undoSnapshot{pngs: pngsCopy, selected: ui.selected, encounter: deepCopyEncounter(ui.encounter)}
	}

	buildList := func() int {
		list.Clear()
		for i := 0; i < len(ui.undoStack); i++ {
			list.AddItem(fmt.Sprintf("  ← %s", swadeSnapshotDesc(ui.undoStack[i])), "", 0, nil)
		}
		current := captureCurrentSnap()
		currentIdx := list.GetItemCount()
		list.AddItem(fmt.Sprintf("→ %s  [attuale]", swadeSnapshotDesc(current)), "", 0, nil)
		for i := len(ui.redoStack) - 1; i >= 0; i-- {
			list.AddItem(fmt.Sprintf("  → %s", swadeSnapshotDesc(ui.redoStack[i])), "", 0, nil)
		}
		return currentIdx
	}

	currentIdx := buildList()
	list.SetCurrentItem(currentIdx)

	closeModal := func() {
		ui.pages.RemovePage("undo-history")
		ui.modalVisible = false
		ui.app.SetFocus(ui.pngList)
		ui.refreshStatus()
	}

	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEscape:
			closeModal()
			return nil
		case tcell.KeyEnter:
			sel := list.GetCurrentItem()
			undoCount := len(ui.undoStack)
			if sel == undoCount {
				closeModal()
				return nil
			} else if sel < undoCount {
				steps := undoCount - sel
				for i := 0; i < steps; i++ {
					ui.performUndo()
				}
			} else {
				steps := sel - undoCount
				for i := 0; i < steps; i++ {
					ui.performRedo()
				}
			}
			closeModal()
			return nil
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'u':
				ui.performUndo()
				newIdx := buildList()
				list.SetCurrentItem(newIdx)
				return nil
			case 'r':
				ui.performRedo()
				newIdx := buildList()
				list.SetCurrentItem(newIdx)
				return nil
			}
		}
		return ev
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 0, 1, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)
	ui.modalVisible = true
	ui.pages.AddPage("undo-history", modal, true, true)
	ui.app.SetFocus(list)
}

// adjustPNGToken increments or decrements the token counter for the selected PNG.
func (ui *tviewUI) adjustPNGToken(delta int) {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		return
	}
	ui.pushUndo()
	ui.pngs[ui.selected].Token += delta
	if ui.pngs[ui.selected].Token < 0 {
		ui.pngs[ui.selected].Token = 0
	}
	ui.persistPNGs()
	ui.refreshPNGs()
	ui.refreshDetail()
	ui.message = fmt.Sprintf("Token %s: %d", ui.pngs[ui.selected].Name, ui.pngs[ui.selected].Token)
	ui.refreshStatus()
}
