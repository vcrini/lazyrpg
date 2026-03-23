//go:build dnd5e

package main

import "github.com/vcrini/lazyrpg/internal/dnd5e"

func init() {
	registeredSystems = []systemEntry{
		{Name: "D&D 5a Edizione", ShortName: "dnd5e", Run: dnd5e.Run},
	}
}
