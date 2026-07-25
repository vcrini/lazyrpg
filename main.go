package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"

	"github.com/vcrini/lazyrpg/internal/common"
)

const version = "0.1.0"

const (
	appStateFile = "state.yml"
)

type appState struct {
	LastSystem string `yaml:"last_system"`
}

func lazyrpgAppDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "."
	}
	return filepath.Join(home, ".lazyrpg")
}

func appStatePath() string {
	return filepath.Join(lazyrpgAppDir(), appStateFile)
}

func loadAppState() appState {
	data, err := os.ReadFile(appStatePath())
	if err != nil {
		return appState{LastSystem: "dnd5e"}
	}
	var st appState
	if err := yaml.Unmarshal(data, &st); err != nil {
		return appState{LastSystem: "dnd5e"}
	}
	if st.LastSystem == "" {
		st.LastSystem = "dnd5e"
	}
	return st
}

func saveAppState(st appState) error {
	if err := os.MkdirAll(lazyrpgAppDir(), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(appStatePath(), data, 0o644)
}

func main() {
	systemFlag := flag.String("system", "", "Sistema da avviare direttamente: dnd5e, dnd5.5e, swade, daggerheart")
	versionFlag := flag.Bool("version", false, "Mostra la versione")
	flag.Parse()

	if *versionFlag {
		fmt.Println("lazyrpg v" + version)
		return
	}

	state := loadAppState()

	systemName := normalizeSystem(*systemFlag)
	if systemName != "" && !validSystem(systemName) {
		var names []string
		for _, s := range registeredSystems {
			names = append(names, s.ShortName)
		}
		fmt.Fprintf(os.Stderr, "Sistema non valido: %q\nValori validi: %s\n", *systemFlag, strings.Join(names, ", "))
		os.Exit(1)
	}

	if systemName == "" {
		if len(registeredSystems) == 1 {
			systemName = registeredSystems[0].ShortName
		} else {
			var err error
			systemName, err = showSystemSelector(state.LastSystem)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Errore nella selezione del sistema: %v\n", err)
				os.Exit(1)
			}
			if systemName == "" {
				return // user quit
			}
		}
	}

	state.LastSystem = systemName
	if err := saveAppState(state); err != nil {
		log.Printf("attenzione: impossibile salvare lo stato: %v", err)
	}

	if err := runSystem(systemName); err != nil {
		fmt.Fprintf(os.Stderr, "Errore nell'esecuzione del sistema %s: %v\n", systemName, err)
		os.Exit(1)
	}
}

func showSystemSelector(lastSystem string) (string, error) {
	app := tview.NewApplication()

	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack
	tview.Styles.ContrastBackgroundColor = tcell.ColorBlack
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorBlack
	tview.Styles.BorderColor = tcell.ColorGold
	tview.Styles.TitleColor = tcell.ColorGold
	tview.Styles.GraphicsColor = tcell.ColorGold
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = tcell.ColorLightGray
	tview.Styles.TertiaryTextColor = tcell.ColorAqua
	var chosen string

	list := tview.NewList().
		ShowSecondaryText(false).
		SetSelectedFocusOnly(false).
		SetMainTextColor(tcell.ColorWhite).
		SetSelectedTextColor(tcell.ColorBlack).
		SetSelectedBackgroundColor(tcell.ColorGold)

	for _, s := range registeredSystems {
		shortName := s.ShortName
		list.AddItem(s.Name, "", 0, func() {
			chosen = shortName
			app.Stop()
		})
	}

	// Set initial selection based on last used system
	for i, s := range registeredSystems {
		if s.ShortName == lastSystem {
			list.SetCurrentItem(i)
			break
		}
	}

	list.SetDoneFunc(func() {
		chosen = ""
		app.Stop()
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || (event.Key() == tcell.KeyRune && event.Rune() == 'q') {
			chosen = ""
			app.Stop()
			return nil
		}
		return event
	})

	box := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(list, 0, 1, true)
	box.SetBorder(true).
		SetTitle(" LazyRPG - Seleziona Sistema (Enter=apri, Esc=esci) ").
		SetTitleColor(tcell.ColorGold).
		SetBorderColor(tcell.ColorGold)

	status := tview.NewTextView().
		SetDynamicColors(true).
		SetText(" [black:gold]↑↓[-:-] naviga  [black:gold]Enter[-:-] seleziona  [black:gold]q/Esc[-:-] esci ")
	status.SetBackgroundColor(tcell.ColorBlack)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(box, 60, 0, true).
			AddItem(tview.NewBox(), 0, 1, false),
			len(registeredSystems)+4, 0, true).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(status, 1, 0, false)

	if err := app.SetRoot(root, true).EnableMouse(true).Run(); err != nil {
		return "", err
	}
	return chosen, nil
}

func normalizeSystem(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "5e", "dnd5e", "dnd", "d&d":
		return "dnd5e"
	case "5.5e", "dnd5.5e", "dnd55e", "2024":
		return "dnd5.5e"
	case "sw", "swade":
		return "swade"
	case "dh", "daggerheart":
		return "daggerheart"
	}
	return s
}

func validSystem(s string) bool {
	for _, sys := range registeredSystems {
		if sys.ShortName == s {
			return true
		}
	}
	return false
}

func runSystem(systemName string) error {
	for _, sys := range registeredSystems {
		if sys.ShortName == systemName {
			return sys.Run(common.ConsoleProgress(sys.Name))
		}
	}
	return fmt.Errorf("sistema sconosciuto: %s", systemName)
}
