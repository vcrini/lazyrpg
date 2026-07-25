//go:build dnd55e

package main

import (
	"github.com/vcrini/lazyrpg/internal/common"
	"github.com/vcrini/lazyrpg/internal/dnd5e"
)

func init() {
	registeredSystems = []systemEntry{
		{Name: "D&D 5.5e (2024)", ShortName: "dnd5.5e", Run: func(p common.ProgressFunc) error {
			return dnd5e.Run(dnd5e.Ruleset2024, p)
		}},
	}
}
