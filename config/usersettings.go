package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// UserSettings holds player-configurable settings that persist across sessions.
type UserSettings struct {
	ResolutionWidth  int `json:"resolution_width"`
	ResolutionHeight int `json:"resolution_height"`
}

// CurrentSettings is the loaded user settings, available at startup.
var CurrentSettings *UserSettings

// ResolutionPreset represents a selectable resolution option.
type ResolutionPreset struct {
	Width  int
	Height int
	Label  string
}

// ResolutionPresets is the list of available resolution options.
var ResolutionPresets = []ResolutionPreset{
	{800, 600, "800x600"},
	{1024, 768, "1024x768"},
	{1280, 720, "1280x720 (720p)"},
	{1366, 768, "1366x768"},
	{1600, 900, "1600x900"},
	{1920, 1080, "1920x1080 (1080p)"},
	{2560, 1440, "2560x1440 (1440p)"},
}

// defaultSettings returns the default user settings (1920x1080).
func defaultSettings() *UserSettings {
	return &UserSettings{
		ResolutionWidth:  1920,
		ResolutionHeight: 1080,
	}
}

// LoadUserSettings reads settings from the given JSON path.
// If the file doesn't exist or is invalid, returns defaults and creates the file.
func LoadUserSettings(path string) {
	CurrentSettings = defaultSettings()

	data, err := os.ReadFile(path)
	if err != nil {
		// File missing — use defaults and save them
		fmt.Printf("No settings file found, using defaults (1920x1080)\n")
		SaveUserSettings(path)
		return
	}

	var settings UserSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		fmt.Printf("WARNING: Failed to parse settings.json: %v (using defaults)\n", err)
		SaveUserSettings(path)
		return
	}

	// Validate resolution values
	if settings.ResolutionWidth <= 0 || settings.ResolutionHeight <= 0 {
		fmt.Printf("WARNING: Invalid resolution in settings.json (using defaults)\n")
		SaveUserSettings(path)
		return
	}

	CurrentSettings = &settings
	fmt.Printf("Loaded settings: %dx%d\n", CurrentSettings.ResolutionWidth, CurrentSettings.ResolutionHeight)
}

// SaveUserSettings writes the current settings to the given JSON path.
func SaveUserSettings(path string) error {
	if CurrentSettings == nil {
		CurrentSettings = defaultSettings()
	}

	data, err := json.MarshalIndent(CurrentSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	return nil
}
