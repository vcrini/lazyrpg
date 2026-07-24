package main

import "github.com/vcrini/lazyrpg/internal/common"

// systemEntry describes a supported RPG system.
type systemEntry struct {
	Name      string
	ShortName string
	Run       func(progress common.ProgressFunc) error
}

// registeredSystems is populated by init() functions in the build-tag-gated
// files (systems_all.go, systems_dnd5e.go, …).
var registeredSystems []systemEntry
