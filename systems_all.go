//go:build !dnd5e && !swade && !daggerheart

package main

import (
	"github.com/vcrini/lazyrpg/internal/daggerheart"
	"github.com/vcrini/lazyrpg/internal/dnd5e"
	"github.com/vcrini/lazyrpg/internal/swade"
)

func init() {
	registeredSystems = []systemEntry{
		{Name: "D&D 5a Edizione", ShortName: "dnd5e", Run: dnd5e.Run},
		{Name: "Savage Worlds Adventure Edition", ShortName: "swade", Run: swade.Run},
		{Name: "Daggerheart", ShortName: "daggerheart", Run: daggerheart.Run},
	}
}
