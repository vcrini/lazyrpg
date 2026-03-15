package common

import (
	"math/rand/v2"
	"strings"

	"gopkg.in/yaml.v3"
)

// CapitalizeWord returns s with its first letter upper-cased.
func CapitalizeWord(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// DefaultNameLists returns a built-in set of first and last names used as a
// fallback when no names.yml can be loaded.
func DefaultNameLists() NameLists {
	return NameLists{
		First: []string{
			"Alucard", "Ambrose", "Ash", "Bellamy", "Calder",
			"Calypso", "Chartreuse", "Clover", "Dahlia",
			"Darrow", "Deacon", "Elowen", "Emrys", "Fable",
			"Fiorella", "Flynn", "Gatlin", "Gerard", "Hadron",
			"Harlow", "Indigo", "Isla", "Jaden", "Kai", "Kismet",
			"Leo", "Mika", "Moon", "Nyx", "Orna", "Phaedra",
			"Quill", "Rani", "Raphael", "Reza", "Roux", "Saffron",
			"Sierra", "Skye", "Talon", "Thea", "Triton", "Vala",
			"Velo", "Wisteria", "Yanelle", "Zahara",
		},
		Last: []string{
			"Abbot", "Advani", "Agoston", "Baptiste", "Belgarde",
			"Blossom", "Chance", "Covault", "Dawn", "Dennison",
			"Drayer", "Emrick", "Foley", "Fury", "Grove",
			"Hartley", "Humfleet", "Hyland", "Ikeda", "Jones",
			"Jordon", "Kaan", "Knoth", "Lagrange", "Lockamy",
			"Lyon", "Marche", "Merrell", "Newland", "Novak",
			"Orwick", "Overholt", "Pray", "Rathbone", "Rose",
			"Seagrave", "Spurlock", "Thorn", "Tringle", "Vasquez",
			"Warren", "Worth", "York",
		},
	}
}

// LoadNameLists loads a NameLists from a YAML file using readFn.
func LoadNameLists(readFn func(string) ([]byte, error), path string) (NameLists, error) {
	data, err := readFn(path)
	if err != nil {
		return DefaultNameLists(), err
	}
	var names NameLists
	if err := yaml.Unmarshal(data, &names); err != nil {
		return DefaultNameLists(), err
	}
	var lists NameLists
	for _, name := range names.First {
		name = strings.TrimSpace(name)
		if name != "" {
			lists.First = append(lists.First, name)
		}
	}
	for _, name := range names.Last {
		name = strings.TrimSpace(name)
		if name != "" {
			lists.Last = append(lists.Last, name)
		}
	}
	if len(lists.First) == 0 {
		return DefaultNameLists(), nil
	}
	return lists, nil
}

// RandomPNGName returns a random name composed of a first (and optionally
// last) name drawn from lists.
func RandomPNGName(lists NameLists) string {
	if len(lists.First) == 0 {
		return "Unknown"
	}
	if len(lists.Last) == 0 {
		return lists.First[rand.IntN(len(lists.First))]
	}
	return lists.First[rand.IntN(len(lists.First))] + " " + lists.Last[rand.IntN(len(lists.Last))]
}

// UniqueRandomPNGName generates a random name that is not already present in
// existing. The existing slice elements must expose their name via a plain
// string; callers pass the existing names pre-extracted.
func UniqueRandomPNGName(existing []string, lists NameLists) string {
	seen := make(map[string]struct{}, len(existing))
	for _, n := range existing {
		seen[n] = struct{}{}
	}
	for {
		name := RandomPNGName(lists)
		if _, ok := seen[name]; !ok {
			return name
		}
	}
}
