# LazyRPG

A terminal-based TUI application for managing tabletop RPG campaigns, inspired by lazygit/lazydocker. Supports three game systems in a unified interface.

## Supported Systems

| System | Flag | Language |
|--------|------|----------|
| D&D 5th Edition (5e/5.5e) | `dnd5e`, `5e`, `dnd` | English |
| Savage Worlds Adventure Edition | `swade`, `sw` | Italian |
| Daggerheart | `daggerheart`, `dh` | Italian |

## Installation

```bash
go build -o lazyrpg .
```

## Usage

```bash
lazyrpg                          # System selector
lazyrpg --system dnd5e           # Launch D&D 5e directly
lazyrpg --system swade           # Launch Savage Worlds directly
lazyrpg --system daggerheart     # Launch Daggerheart directly
lazyrpg --version                # Show version
```

The application remembers the last used system and highlights it in the selector.

## Data Storage

- **Saves**: `~/.lazyrpg/<system>/` for each system
- **Last system**: `~/.lazyrpg/state.yml`
- **Config data**: embedded in the binary (YAML)

---

## Dice Expression Syntax

All three systems share the same dice expression engine. Expressions are entered in the Dice panel (`a` to add, `e` to edit).

### Basic

| Expression | Description |
|-----------|-------------|
| `d6` / `1d6` | Single d6 |
| `2d6` | Roll 2d6, sum results |
| `2d6+3` | Roll 2d6, add 3 |
| `d20-1` | Roll d20, subtract 1 |
| `2d6+d8` | Sum different dice types |
| `4d10+6d6+5` | Complex multi-die expression |

### Advantage / Disadvantage

| Expression | Description |
|-----------|-------------|
| `d20v+5` | Advantage: roll 2d20, keep **higher**, add 5 |
| `d20s+1` | Disadvantage: roll 2d20, keep **lower**, add 1 |
| `d20a+5` | Alias for `v` (advantage) |
| `d20d+1` | Alias for `s` (disadvantage) |

### Exploding Dice *(SWADE only)*

| Expression | Description |
|-----------|-------------|
| `d6e` | Exploding die: on max value, reroll and add to total |
| `2d8e` | Two exploding d8s |
| `D6` | SWADE trait roll: `d6e` + `d6e` wild die, take highest |

### Multi-Expression

Roll several different expressions in one command (results stored separately):

```
d6,d8             → rolls d6 and d8 independently
d6,d8,1d20+4      → three separate rolls
```

### Batch Rolls

Repeat the same expression N times:

```
1d6 x3            → three independent d6 rolls
1d20+5 x4         → four d20+5 rolls
```

### Comparison (Pass / Fail vs DC)

| Expression | Description |
|-----------|-------------|
| `d20+5 > 15` | Roll and show ok/ko vs DC |
| `d20+5 >= 15` | Greater than or equal |
| `d20+5 < 10` | Less than |
| `d20+5 <= 10` | Less than or equal |

### Conditional Roll on Success

Roll a check; if it passes, also roll the second expression:

```
d20 > 15 2d6+1    → if d20 > 15, then roll 2d6+1
```

### Batch + Comparison

```
1d20+5 > 10 x3    → three independent pass/fail checks
```

### Labeled Damage *(D&D 5e only)*

Annotate damage types within a single expression:

```
(4d8+1:slash)+(3d6:acid)
(1d6:cold)+(2d4:poison)
```

### Inline Dice Roll *(Daggerheart only)*

In the detail panel, press `Enter` on any line containing a dice expression (e.g. `2d6+3`) to roll it instantly.

---

## D&D 5th Edition

### Panels

- **Dice** — Roll dice expressions, save/load roll history
- **Encounters** — Track HP, initiative, conditions, death saves for monsters and characters
- **Catalog** — Browse monsters, items, spells, characters, races, feats, books, adventures, random generators, notes
- **Description / Treasure** — Full text detail and treasure generation

### Encounters

- Add monsters from the catalog or create custom entries
- Track current/max HP with temporary HP support (temp is consumed before regular HP)
- Toggle HP mode between average value and dice formula (Space)
- Roll initiative per entry or all at once; sort by initiative (S); activate turn mode (*)
- Advance turns with `n`; re-sort or reset at any time
- **Death saves**: when a character reaches 0 HP, a 💀 appears. Press `R` to roll 1d20:
  - Natural 1 → 2 failures
  - 2–9 → 1 failure
  - 10–19 → 1 success
  - Natural 20 → critical: 3 successes, HP becomes 1
  - 3 failures → ☠️ (dead)
  - 3 successes → stabilized (skull removed, HP stays 0)
- Add/remove conditions ([ / ] adjust duration in rounds); condition badges use emoji symbols
- Yank/paste entries (y/p); undo/redo (u/r)
- Generate an encounter from the party composition (g)

### Random Generators

Accessible from the **Random** catalog panel:

- Dungeon room contents and layout
- NPCs, place names, social events
- Treasure caches, magic items, currency
- Adventure events, plot hooks
- Monster encounter tables
- Equipment and magic shops

### Keyboard Reference — D&D 5e

**Global**

| Key | Action |
|-----|--------|
| `?` / `Esc` | Open/close help |
| `q` | Quit |
| `f` | Fullscreen current panel |
| `X` | Clear all filters |
| `Tab` / `Shift+Tab` | Cycle focus |
| `0` `1` `2` `3` | Jump to Dice / Encounters / Catalog / Description |
| `G` | Panel jump modal |
| `[` / `]` | Previous/next catalog tab |
| `4`–`9` / `b` `v` `z` | Jump to catalog tabs |
| `Ctrl+S` / `Ctrl+O` | Save / load campaign |

**Dice**

| Key | Action |
|-----|--------|
| `a` | New roll (`2d6+d20+1`, `d20v+5`, …) |
| `A` | Re-roll all |
| `Enter` | Re-roll selected |
| `e` | Edit and re-roll selected |
| `d` / `D` | Delete selected / clear all |
| `s` / `l` | Save / load roll history |
| `g#` `g^` `g$` | Go to row # / first / last |

**Encounters**

| Key | Action |
|-----|--------|
| `j` `k` / `↑` `↓` | Navigate list |
| `a` | Add custom entry |
| `e` | Edit selected entry |
| `g` | Generate encounter from party |
| `d` / `D` | Delete selected / delete all monsters |
| `s` / `l` | Save / load encounter |
| `i` / `I` | Roll initiative (selected / all) |
| `S` | Sort by initiative |
| `*` | Toggle turn mode |
| `n` | Next turn |
| `K` | Skill check |
| `V` | Saving throw vs DC |
| `y` / `p` | Yank / paste entry |
| `u` / `r` | Undo / redo |
| `c` | Add/remove conditions |
| `x` / `C` | Remove one condition / clear all |
| `[` / `]` | Decrease/increase condition rounds |
| `L` / `H` | Set temp HP / clear temp HP |
| `Space` | Toggle HP average/formula |
| `h` / `←` | Subtract HP |
| `→` | Add HP |
| `R` | Roll death saving throw (at 0 HP) |

**Monsters (Catalog)**

| Key | Action |
|-----|--------|
| `a` | Add to Encounters |
| `m` / `l` | Generate treasure / lair treasure |
| `←` / `→` | Scale monster CR |
| `x` | Clear filters |
| `n` `e` `s` `c` `t` | Focus Name/Env/Source/CR/Type filter |
| `/` | Search description |

---

## Savage Worlds Adventure Edition

### Panels

- **Dice** — Roll SWADE dice expressions (exploding dice, e.g. `d6e`, `2d8e`)
- **PNG** — Player characters list
- **Encounter** — Monsters and extras with wounds and conditions
- **Catalog** — Monsters, Equipment, Cards, Rules

### Encounters

- Track wounds (max determined by monster stats or set manually)
- Initiative uses action cards (suit + rank) instead of d20 rolls
- Add/remove conditions from the SWADE condition list with round tracking
- Toggle extended condition effects display (`o`)
- Sort by initiative card (`S`), enter turn mode (`*`), advance turns (`n`)
- Edit encounter entries: name, initiative card, wounds

### Conditions (SWADE)

| Code | Condition | Symbol |
|------|-----------|--------|
| S | Scosso | 😵‍💫 |
| T | Stordito | 😵 |
| D | Distratto | 😬 |
| V | Vulnerabile | 🙃 |
| H | Impedito | 🫲 |
| F | Affaticato | 😴 |
| E | Intrappolato | 🪤 |
| B | Vincolato | ⛓️ |

### Keyboard Reference — SWADE

**Global**

| Key | Action |
|-----|--------|
| `?` / `Esc` | Open/close help |
| `q` | Quit |
| `Tab` / `Shift+Tab` | Cycle focus |
| `0` `1` `2` `3` | Dice / PNG / Encounter / Catalog |
| `[` / `]` | Previous/next catalog tab |
| `f` | Fullscreen |

**Dice**

| Key | Action |
|-----|--------|
| `a` | New roll |
| `Enter` | Re-roll selected |
| `e` | Edit and re-roll |
| `d` / `c` | Delete selected / clear all |

**PNG**

| Key | Action |
|-----|--------|
| `c` | Create PNG |
| `m` | Rename PNG |
| `x` | Delete PNG |
| `a` | Add to Encounter |

**Encounter**

| Key | Action |
|-----|--------|
| `h` `l` / `j` `k` | Wounds +1 / -1 |
| `c` | Add/remove conditions |
| `x` / `C` | Remove one condition / clear all |
| `[` / `]` | Decrease/increase condition rounds |
| `o` | Toggle extended condition effects |
| `i` / `I` | Roll initiative (selected / all) |
| `S` | Sort by initiative |
| `*` | Enter initiative turn mode |
| `n` | Next turn |
| `e` | Edit entry (name, initiative card) |
| `d` | Remove selected |

---

## Daggerheart

### Panels

- **Dice** — Full dice expression engine with batch rolls and multi-expression support
- **PNG** — Characters with PF, Stress, Hope, Armor tracking
- **Encounter** — Monsters with PF and Stress
- **Catalog** — Monsters, Environments, Equipment, Cards, Classes, Notes

### Key Features

- **Fear tracker**: 0–12 level displayed at the top, modified with `+`/`-`, saved separately
- **Stress mechanic**: Stress reduction at 0 also reduces PF
- **Armor thresholds**: Min/max armor values with scaling
- **Sequence tracking**: Track current sequence number for encounters
- **Undo/redo**: Full UI state snapshot history
- **Line numbers**: Toggle with `#`
- **Vim-style navigation**: `j`/`k`, `f`/`b`, `gg`/`G`, `Ctrl+D`/`Ctrl+U` in the detail panel

### Keyboard Reference — Daggerheart

**Global**

| Key | Action |
|-----|--------|
| `?` / `Esc` | Open/close help |
| `q` | Quit |
| `Tab` / `Shift+Tab` | Cycle focus |
| `0`–`7` | Focus panels |
| `+` / `-` | Increase/decrease Fear |
| `Shift+S` / `Shift+L` | Save / load Fear |
| `[` / `]` | Previous/next catalog tab |
| `G` | Go to panel modal |
| `N` | Focus Notes |
| `u` / `r` | Undo / redo |
| `f` | Fullscreen |
| `#` | Toggle line numbers |
| `g#` `g^` `g$` | Go to row # / first / last |

**Dice**

| Key | Action |
|-----|--------|
| `a` | New roll (`NdM`, `NdM+K`, batch `xN`, multi-expression) |
| `Enter` | Re-roll selected |
| `e` | Edit and re-roll |
| `d` / `c` | Delete selected / clear all |
| `s` / `l` | Save / load roll history |

**PNG**

| Key | Action |
|-----|--------|
| `a` | Create PNG |
| `e` | Edit selected PNG |
| `d` / `D` | Delete selected / delete all |
| `R` | Reset token for all PNG |
| `←` / `→` | Decrease/increase token |
| `Shift+←` / `Shift+→` | PF -1 / +1 |
| `Shift+↓` / `Shift+↑` | Stress -1 / +1 |
| `Alt+←` / `Alt+→` | Armor -1 / +1 |
| `Alt+↓` / `Alt+↑` | Hope -1 / +1 |
| `s` / `l` | Save / load PNG |

**Encounter**

| Key | Action |
|-----|--------|
| `Shift+←` / `Shift+→` | PF -1 / +1 |
| `Shift+↓` / `Shift+↑` | Stress -1 / +1 |
| `y` / `p` | Copy / paste entry (with incremented number) |
| `d` / `D` | Delete selected / clear encounter |
| `e` | Edit entry (name, PF, Stress) |
| `s` / `l` | Save / load encounter |

**Detail Panel**

| Key | Action |
|-----|--------|
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `f` / `b` | Page down / page up |
| `gg` | Go to beginning |
| `G` | Go to end |
| `Ctrl+D` / `Ctrl+U` | Half-page down / up |
| `Enter` | Roll dice from current line |

---

## Architecture

Built with [rivo/tview](https://github.com/rivo/tview). Each system is an independent package under `internal/` sharing common utilities from `internal/common/`.

- Color scheme: black background, gold borders and titles
- Mouse support enabled
- All config data embedded in the binary via `//go:embed`
- Saves use YAML format
