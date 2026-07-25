package dnd5e

import "fmt"

// ensureCatalogLoaded lazily parses the embedded YAML for a browse tab the
// first time it's needed — either by switching to it (see setBrowseMode) or
// by a feature that reads its data without visiting the tab first (treasure
// generation, character creation, ...). Monsters are the only catalog
// loaded eagerly, in Run().
func (ui *UI) ensureCatalogLoaded(mode BrowseMode) {
	if ui.catalogLoaded == nil {
		ui.catalogLoaded = map[BrowseMode]bool{}
	}
	if ui.catalogLoaded[mode] {
		return
	}
	ui.catalogLoaded[mode] = true

	label := catalogLoadingLabel(mode)
	if label == "" {
		return
	}
	ui.showCatalogLoadingStatus(label)

	switch mode {
	case BrowseItems:
		if items, _, _, _, err := loadItemsFromBytes(embeddedItemsYAML); err == nil {
			ui.items, _, _, _ = filterCatalogByRuleset(items, ui.ruleset)
		}
	case BrowseSpells:
		if spells, _, _, _, err := loadSpellsFromBytes(embeddedSpellsYAML); err == nil {
			ui.spells, _, _, _ = filterCatalogByRuleset(spells, ui.ruleset)
		}
	case BrowseCharacters:
		if classes, _, _, _, err := loadClassesFromBytes(embeddedClassesYAML); err == nil {
			ui.classes, _, _, _ = filterCatalogByRuleset(classes, ui.ruleset)
		}
	case BrowseRaces:
		if races, _, _, _, err := loadRacesFromBytes(embeddedRacesYAML); err == nil {
			ui.races, _, _, _ = filterCatalogByRuleset(races, ui.ruleset)
		}
	case BrowseFeats:
		if feats, _, _, _, err := loadFeatsFromBytes(embeddedFeatsYAML); err == nil {
			ui.feats, _, _, _ = filterCatalogByRuleset(feats, ui.ruleset)
		}
	case BrowseBooks:
		// Shared/unfiltered across rulesets — see ruleset.go.
		if books, _, _, _, err := loadBooksFromBytes(embeddedBooksYAML); err == nil {
			ui.books = books
		}
	case BrowseAdventures:
		// Shared/unfiltered across rulesets — see ruleset.go.
		if advs, _, _, _, err := loadAdventuresFromBytes(embeddedAdventuresYAML); err == nil {
			ui.adventures = advs
		}
	}
}

// ensureBackgroundsLoaded lazily loads backgrounds.yml. Backgrounds have no
// browse tab of their own (they're only consumed by character creation), so
// they can't be keyed off a BrowseMode switch like the other catalogs.
func (ui *UI) ensureBackgroundsLoaded() {
	if ui.backgroundsLoaded {
		return
	}
	ui.backgroundsLoaded = true
	ui.showCatalogLoadingStatus("backgrounds")
	if backgrounds, err := loadBackgroundsFromBytes(embeddedBackgroundsYAML); err == nil {
		ui.backgrounds = backgrounds
	}
}

// showCatalogLoadingStatus briefly reports what's loading and forces an
// immediate redraw, since parsing the larger YAML files (books, adventures)
// can take up to a couple of seconds and would otherwise leave the UI
// looking frozen mid-keypress.
func (ui *UI) showCatalogLoadingStatus(label string) {
	if ui.status == nil || ui.app == nil {
		return
	}
	ui.status.SetText(fmt.Sprintf(" [black:gold]loading[-:-] %s...  %s", label, helpText))
	ui.app.ForceDraw()
}

func catalogLoadingLabel(mode BrowseMode) string {
	switch mode {
	case BrowseItems:
		return "items"
	case BrowseSpells:
		return "spells"
	case BrowseCharacters:
		return "classes"
	case BrowseRaces:
		return "races"
	case BrowseFeats:
		return "feats"
	case BrowseBooks:
		return "books"
	case BrowseAdventures:
		return "adventures"
	}
	return ""
}
