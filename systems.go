package main

// systemEntry describes a supported RPG system.
type systemEntry struct {
	Name      string
	ShortName string
	Run       func() error
}

// registeredSystems is populated by init() functions in the build-tag-gated
// files (systems_all.go, systems_dnd5e.go, …).
var registeredSystems []systemEntry
