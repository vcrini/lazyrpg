//go:build swade

package main

import "github.com/vcrini/lazyrpg/internal/swade"

func init() {
	registeredSystems = []systemEntry{
		{Name: "Savage Worlds Adventure Edition", ShortName: "swade", Run: swade.Run},
	}
}
