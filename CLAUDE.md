# LazyRPG

Applicativo TUI per la gestione di campagne di giochi di ruolo da tavolo. Unifica tre sistemi in un unico programma con interfaccia stile lazygit/lazydocker.

## Sistemi supportati

| Sistema | Package | Lingua dati |
|---------|---------|-------------|
| D&D 5a Edizione (5e/5.5e) | `internal/dnd5e` | Inglese (`en`) |
| Savage Worlds Adventure Edition | `internal/swade` | Italiano (`it`) |
| Daggerheart | `internal/daggerheart` | Italiano (`it`) |

## Struttura progetto

```
lazyrpg/
├── main.go                          # Entry point: selettore sistema, flag CLI
├── internal/
│   ├── common/                      # Codice condiviso tra sistemi
│   │   ├── types.go                 # DiceResult, ClassPreset, Thresholds, CardItem, ClassItem, Environment, NameLists
│   │   ├── data.go                  # ReadData(), LoadYAMLList[T]()
│   │   ├── names.go                 # Generazione nomi casuali PNG
│   │   ├── text_utils.go            # CardDescriptionHead(), HighlightMatches()
│   │   └── class_presets.go         # ClassPresetFor() — 9 preset di classe
│   ├── dnd5e/
│   │   ├── ui.go                    # UI completa D&D 5e (tview)
│   │   └── config/en/               # YAML incorporati (//go:embed)
│   ├── swade/
│   │   ├── ui.go                    # UI Savage Worlds (tview)
│   │   ├── data.go                  # Strutture dati SWADE-specifiche
│   │   ├── encounter.go             # Logica encounter (condizioni, ferite)
│   │   └── config/                  # YAML incorporati
│   └── daggerheart/
│       ├── ui.go                    # UI Daggerheart (tview)
│       ├── data.go                  # Strutture dati Daggerheart-specifiche
│       ├── encounter.go             # Logica encounter (seq, vitalità)
│       └── config/                  # YAML incorporati
```

## Configurazione e dati

- **Config YAML**: `internal/<sistema>/config/<lingua>/*.yml` (embed nel binario)
- **Salvataggi**: `~/.lazyrpg/<sistema>/` per ogni sistema
- **Stato app**: `~/.lazyrpg/state.yml` (ultimo sistema usato)

## Avvio

```bash
go run .                        # Mostra selettore sistema
go run . --system dnd5e         # Avvia direttamente D&D 5e
go run . --system swade         # Avvia direttamente Savage Worlds
go run . --system daggerheart   # Avvia direttamente Daggerheart
go run . --version              # Mostra versione
```

## Framework TUI

**rivo/tview** per tutti e tre i sistemi. Pattern comune:
- Pannello sinistro: liste (mostri, PNG, encounter, dadi)
- Pannello destro: dettaglio testuale
- Navigazione a tastiera con focus order esplicito
- Schema colori: sfondo nero, bordi/titoli dorati

## Aggiungere un nuovo sistema

1. Crea `internal/<sistema>/` con `ui.go`, `data.go`, `encounter.go`
2. Aggiungi i YAML sotto `internal/<sistema>/config/<lingua>/`
3. Implementa la funzione `Run() error` che avvia l'applicazione tview
4. Registra il sistema in `main.go`: nella lista `systems` e nello switch `runSystem()`

## Note architetturali

- I file `ui.go` di ciascun sistema sono grandi (~6-15k righe). Questo rispecchia l'approccio delle app sorgente originali (lazy5e, lazysw, lazydaggerheart).
- Il codice condiviso tra swade e daggerheart vive in `internal/common/`.
- dnd5e ha architettura diversa (portato da lazy5e che era monolitico).
- **Non duplicare** tipi o funzioni già presenti in `internal/common/`.
