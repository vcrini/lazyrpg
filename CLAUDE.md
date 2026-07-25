# LazyRPG

Applicativo TUI per la gestione di campagne di giochi di ruolo da tavolo. Unifica quattro sistemi (tre giochi) in un unico programma con interfaccia stile lazygit/lazydocker.

## Sistemi supportati

| Sistema | ShortName | Package | Lingua dati |
|---------|-----------|---------|-------------|
| D&D 5a Edizione (regole 2014) | `dnd5e` | `internal/dnd5e` | Inglese (`en`) |
| D&D 5e (regole 2024, "5.5e") | `dnd5.5e` | `internal/dnd5e` | Inglese (`en`) |
| Savage Worlds Adventure Edition | `swade` | `internal/swade` | Italiano (`it`) |
| Daggerheart | `daggerheart` | `internal/daggerheart` | Italiano (`it`) |

`dnd5e` e `dnd5.5e` condividono lo stesso package/UI (`internal/dnd5e`): sono la stessa `Run()` chiamata con un `Ruleset` diverso (`Ruleset2014`/`Ruleset2024`, vedi `internal/dnd5e/ruleset.go`), non due implementazioni separate — non duplicare questo codice se serve estendere l'edizione a nuovi contenuti.

### Filtro per edizione (2014 vs 2024)

Ogni voce di mostri/oggetti/incantesimi/classi/razze/talenti ha un campo `source` (codice manuale, es. `MM`, `PHB` per il 2014; `XMM`, `XPHB`, `XDMG` per il 2024). `filterCatalogByRuleset` in `internal/dnd5e/ruleset.go` applica questa regola dopo ogni caricamento:
- **Ruleset2014**: esclude tutto ciò che viene solo da un manuale core 2024 (`XPHB`/`XDMG`/`XMM`).
- **Ruleset2024**: se esiste una voce 2024 con lo stesso nome, quella sostituisce la versione 2014 (niente doppioni); il resto del materiale 2014/terze parti mai aggiornato resta visibile.
- **Non filtrati** (condivisi tra i due sistemi): manuali (`books`) e avventure (`adventures`) — materiale di riferimento, non legato a un'edizione delle regole — e i background (`backgrounds`), perché il dataset attuale copre solo la lista 2024 e non ha un campo `source` per distinguere le edizioni.

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
│   │   ├── ui.go                    # UI completa D&D 5e (tview), condivisa da dnd5e e dnd5.5e
│   │   ├── ruleset.go               # Ruleset2014/Ruleset2024, filtro catalogo per edizione
│   │   └── config/en/               # YAML incorporati (//go:embed), nomi in inglese (monsters.yml, items.yml, ...)
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

### ⚠️ I file YAML di config NON sono versionati in git

`internal/*/config/*/*.yml` è in `.gitignore`: questi file esistono solo sul filesystem locale di ogni installazione, non nella cronologia git. Il motivo è evitare conflitti di `git pull` quando più installazioni hanno versioni locali diverse degli stessi dati.

**Implicazione critica**: `go build`/`go run` falliscono su un clone pulito o in CI se questi file non sono presenti (le direttive `//go:embed` in `internal/dnd5e/ui.go`, `internal/swade/data.go`, `internal/daggerheart/data.go` richiedono i file sul disco al momento della build). Prima di compilare da un clone nuovo, questi YAML vanno recuperati/copiati manualmente da un'altra installazione o da un backup — non c'è (ancora) un meccanismo automatico di distribuzione.

## Avvio

```bash
go run .                        # Mostra selettore sistema
go run . --system dnd5e         # Avvia direttamente D&D 5e (2014)
go run . --system dnd5.5e       # Avvia direttamente D&D 5.5e (2024)
go run . --system swade         # Avvia direttamente Savage Worlds
go run . --system daggerheart   # Avvia direttamente Daggerheart
go run . --version              # Mostra versione
```

## Build per singolo sistema (build tags)

```bash
go build -tags dnd5e -o lazyrpg-dnd5e .
go build -tags dnd55e -o lazyrpg-dnd55e .
go build -tags swade -o lazyrpg-swade .
go build -tags daggerheart -o lazyrpg-dh .
go install -tags dnd5e .        # installa solo D&D 5e (2014)
```

Con un solo sistema compilato il selettore viene saltato e l'app si avvia direttamente. File coinvolti: `systems.go` (struct + var registeredSystems), `systems_all.go` (default, tutti e quattro), `systems_dnd5e.go`, `systems_dnd55e.go`, `systems_swade.go`, `systems_daggerheart.go`. Nota: il build tag di `dnd5.5e` è `dnd55e` (niente punto, non valido nei build tag Go).

## Framework TUI

**rivo/tview** per tutti e tre i sistemi. Pattern comune:
- Pannello sinistro: liste (mostri, PNG, encounter, dadi)
- Pannello destro: dettaglio testuale
- Navigazione a tastiera con focus order esplicito
- Schema colori: sfondo nero, bordi/titoli dorati

## Aggiungere un nuovo sistema

1. Crea `internal/<sistema>/` con `ui.go`, `data.go`, `encounter.go`
2. Aggiungi i YAML sotto `internal/<sistema>/config/<lingua>/`
3. Implementa la funzione `Run(progress common.ProgressFunc) error` che avvia l'applicazione tview; chiama `progress(step, current, total)` prima di ogni caricamento dati (YAML grandi, file di salvataggio) per mostrare l'avanzamento nel terminale prima che parta la UI tview
4. Registra il sistema aggiungendo un file `systems_<nome>.go` con build tag e un `init()` che appende a `registeredSystems`; aggiungi anche la voce in `systems_all.go` e aggiorna il build tag constraint di quest'ultimo

## Note architetturali

- I file `ui.go` di ciascun sistema sono grandi (~6-15k righe). Questo rispecchia l'approccio delle app sorgente originali (lazy5e, lazysw, lazydaggerheart).
- Il codice condiviso tra swade e daggerheart vive in `internal/common/`.
- dnd5e ha architettura diversa (portato da lazy5e che era monolitico).
- **Non duplicare** tipi o funzioni già presenti in `internal/common/`.
