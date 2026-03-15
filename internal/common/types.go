package common

import "gopkg.in/yaml.v3"

// DiceResult holds a single dice roll expression and its textual output.
type DiceResult struct {
	Expression string `yaml:"expression"`
	Output     string `yaml:"output"`
}

// ClassPreset holds default equipment and appearance data for a character class.
type ClassPreset struct {
	Traits    string
	Primary   string
	Secondary string
	Armor     string
	ExtraA    string
	ExtraB    string
	Abiti     []string
	Attitude  []string
}

// NameLists holds first and last name lists used for random PNG name generation.
type NameLists struct {
	First []string `yaml:"first"`
	Last  []string `yaml:"last"`
}

// Thresholds can be either a list of ints or a plain string in YAML.
type Thresholds struct {
	Values []int
	Text   string
}

// UnmarshalYAML implements yaml.Unmarshaler for Thresholds.
func (t *Thresholds) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var vals []int
		for i := 0; i < len(value.Content); i++ {
			var v int
			if err := value.Content[i].Decode(&v); err != nil {
				return err
			}
			vals = append(vals, v)
		}
		t.Values = vals
		t.Text = ""
		return nil
	case yaml.ScalarNode:
		t.Text = value.Value
		t.Values = nil
		return nil
	default:
		return nil
	}
}

// CardItem represents a domain card / spell entry.
type CardItem struct {
	Name        string   `yaml:"name"`
	Class       string   `yaml:"class"`
	Type        string   `yaml:"type"`
	CasterTrait string   `yaml:"caster_trait"`
	Description string   `yaml:"description"`
	Effects     []string `yaml:"effects"`
}

// ClassItem represents a character class definition.
type ClassItem struct {
	Source          string   `yaml:"source,omitempty"`
	Name            string   `yaml:"name"`
	Subclass        string   `yaml:"subclass"`
	Rank            int      `yaml:"rank"`
	Domains         string   `yaml:"domains"`
	Evasion         int      `yaml:"evasion"`
	HP              int      `yaml:"hp"`
	ClassItem       string   `yaml:"class_item"`
	HopePrivilege   string   `yaml:"hope_privilege"`
	ClassPrivileges []string `yaml:"class_privileges"`
	Description     string   `yaml:"description"`
	CasterTrait     string   `yaml:"caster_trait"`
	BasePrivileges  []string `yaml:"base_privileges"`
	Specialization  string   `yaml:"specialization"`
	Mastery         string   `yaml:"mastery"`
	BackgroundQs    []string `yaml:"background_questions"`
	Bonds           []string `yaml:"bonds"`
}

// Environment represents an environment / scenario entry.
type Environment struct {
	Name                 string `yaml:"name"`
	Kind                 string `yaml:"kind"`
	Rank                 int    `yaml:"rank"`
	Description          string `yaml:"description"`
	Impeti               string `yaml:"impeti"`
	Difficulty           string `yaml:"difficulty"`
	PotentialAdversaries string `yaml:"potential_adversaries"`
	Characteristics      []struct {
		Name string `yaml:"name"`
		Kind string `yaml:"kind"`
		Text string `yaml:"text"`
	} `yaml:"characteristics"`
}
