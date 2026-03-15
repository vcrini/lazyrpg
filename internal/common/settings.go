package common

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CampaignSettings holds per-system persistent settings.
type CampaignSettings struct {
	TurnTimerSeconds int    `yaml:"turn_timer_seconds"`
	LastPanel        string `yaml:"last_panel,omitempty"`
}

// DefaultCampaignSettings returns the default settings values.
func DefaultCampaignSettings() CampaignSettings {
	return CampaignSettings{TurnTimerSeconds: 20}
}

// LoadCampaignSettings reads settings from <appDir>/settings.yml.
// If the file is missing, defaults are written to disk so the user can edit them.
func LoadCampaignSettings(appDir string) CampaignSettings {
	path := filepath.Join(appDir, "settings.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		def := DefaultCampaignSettings()
		_ = SaveCampaignSettings(appDir, def)
		return def
	}
	var s CampaignSettings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return DefaultCampaignSettings()
	}
	if s.TurnTimerSeconds == 0 && s.LastPanel == "" {
		s = DefaultCampaignSettings()
	}
	return s
}

// SaveCampaignSettings writes settings to <appDir>/settings.yml.
func SaveCampaignSettings(appDir string, s CampaignSettings) error {
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(appDir, "settings.yml"), data, 0o644)
}
