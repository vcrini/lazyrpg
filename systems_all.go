//go:build !dnd5e && !dnd55e && !swade && !daggerheart

package main

import (
	"github.com/vcrini/lazyrpg/internal/common"
	"github.com/vcrini/lazyrpg/internal/daggerheart"
	"github.com/vcrini/lazyrpg/internal/dnd5e"
	"github.com/vcrini/lazyrpg/internal/swade"
)

func init() {
	registeredSystems = []systemEntry{
		{Name: "D&D 5e (2014)", ShortName: "dnd5e", Run: func(p common.ProgressFunc) error {
			return dnd5e.Run(dnd5e.Ruleset2014, p)
		}},
		{Name: "D&D 5.5e (2024)", ShortName: "dnd5.5e", Run: func(p common.ProgressFunc) error {
			return dnd5e.Run(dnd5e.Ruleset2024, p)
		}},
		{Name: "Savage Worlds Adventure Edition", ShortName: "swade", Run: swade.Run},
		{Name: "Daggerheart", ShortName: "daggerheart", Run: daggerheart.Run},
	}
}
