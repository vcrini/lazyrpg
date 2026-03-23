//go:build daggerheart

package main

import "github.com/vcrini/lazyrpg/internal/daggerheart"

func init() {
	registeredSystems = []systemEntry{
		{Name: "Daggerheart", ShortName: "daggerheart", Run: daggerheart.Run},
	}
}
