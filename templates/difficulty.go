package templates

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync/atomic"
)

// Difficulty level constants
const (
	DifficultyEasy   = "Easy"
	DifficultyMedium = "Medium"
	DifficultyHard   = "Hard"
)

// EncounterDifficultyMultipliers controls encounter generation scaling.
type EncounterDifficultyMultipliers struct {
	PowerMultiplierScale   float64
	SquadCountOffset       int
	MinUnitsPerSquadOffset int
	MaxUnitsPerSquadOffset int
}

// OverworldDifficultyMultipliers controls overworld threat/faction scaling.
type OverworldDifficultyMultipliers struct {
	ThreatGrowthScale              float64
	ContainmentSlowdownScale       float64
	MaxThreatIntensityOffset       int
	SpawnChanceScale               float64
	FortificationStrengthGainScale float64
	RaidIntensityScale             float64
}

// AIDifficultyMultipliers controls AI behavior threshold adjustments.
type AIDifficultyMultipliers struct {
	FlankingRangeBonusOffset    int
	IsolationThresholdOffset    int
	RetreatSafeThresholdOffset  int
	SharedRangedWeightScale     float64
	SharedPositionalWeightScale float64
}

// DifficultyPreset holds the 4 master knobs and derived multipliers for a difficulty level.
// Master knobs are deserialized from JSON; derived sub-structs are computed at load time.
type DifficultyPreset struct {
	Name string `json:"name"`

	// Master knobs (JSON-facing)
	CombatIntensity     float64 `json:"combatIntensity"`
	OverworldPressure   float64 `json:"overworldPressure"`
	AICompetence        float64 `json:"aiCompetence"`
	EncounterSizeOffset int     `json:"encounterSizeOffset"`

	// Derived values (computed, not from JSON)
	encounter EncounterDifficultyMultipliers
	overworld OverworldDifficultyMultipliers
	ai        AIDifficultyMultipliers
}

// DifficultyConfigData is the root container for difficulty configuration JSON.
type DifficultyConfigData struct {
	Difficulties      []DifficultyPreset `json:"difficulties"`
	DefaultDifficulty string             `json:"defaultDifficulty"`
}

// DifficultyManager holds all difficulty presets and the currently active one.
// Uses atomic.Pointer for safe mid-game difficulty changes.
type DifficultyManager struct {
	presets map[string]*DifficultyPreset
	current atomic.Pointer[DifficultyPreset]
}

// GlobalDifficulty is the package-level difficulty manager instance.
var GlobalDifficulty *DifficultyManager

// SetDifficulty switches the active difficulty preset.
// Returns an error if the name doesn't match any loaded preset.
func (dm *DifficultyManager) SetDifficulty(name string) error {
	preset, ok := dm.presets[name]
	if !ok {
		return fmt.Errorf("unknown difficulty: %q", name)
	}
	dm.current.Store(preset)
	return nil
}

// GetCurrentDifficulty returns the name of the active difficulty preset.
func (dm *DifficultyManager) GetCurrentDifficulty() string {
	return dm.current.Load().Name
}

// Encounter returns the active encounter difficulty multipliers.
func (dm *DifficultyManager) Encounter() EncounterDifficultyMultipliers {
	return dm.current.Load().encounter
}

// Overworld returns the active overworld difficulty multipliers.
func (dm *DifficultyManager) Overworld() OverworldDifficultyMultipliers {
	return dm.current.Load().overworld
}

// AI returns the active AI difficulty multipliers.
func (dm *DifficultyManager) AI() AIDifficultyMultipliers {
	return dm.current.Load().ai
}

// deriveDifficultyValues computes all derived sub-struct fields from the 4 master knobs.
// Medium (1.0/1.0/1.0/0) produces identity multipliers (all scales 1.0, all offsets 0).
func deriveDifficultyValues(preset *DifficultyPreset) {
	// From combatIntensity
	preset.encounter.PowerMultiplierScale = preset.CombatIntensity

	// From encounterSizeOffset
	preset.encounter.SquadCountOffset = preset.EncounterSizeOffset
	preset.encounter.MinUnitsPerSquadOffset = preset.EncounterSizeOffset
	preset.encounter.MaxUnitsPerSquadOffset = preset.EncounterSizeOffset

	// From overworldPressure
	preset.overworld.ThreatGrowthScale = preset.OverworldPressure
	preset.overworld.SpawnChanceScale = preset.OverworldPressure
	preset.overworld.RaidIntensityScale = preset.OverworldPressure
	preset.overworld.FortificationStrengthGainScale = preset.OverworldPressure
	preset.overworld.ContainmentSlowdownScale = 1.0 + (preset.OverworldPressure-1.0)*0.75
	preset.overworld.MaxThreatIntensityOffset = int(math.Round((preset.OverworldPressure - 1.0) * 5))

	// From aiCompetence
	preset.ai.FlankingRangeBonusOffset = int(math.Round((preset.AICompetence - 1.0) * 5))
	preset.ai.IsolationThresholdOffset = -int(math.Round((preset.AICompetence - 1.0) * 3))
	preset.ai.RetreatSafeThresholdOffset = int(math.Round((preset.AICompetence - 1.0) * 7))
	preset.ai.SharedRangedWeightScale = preset.AICompetence
	preset.ai.SharedPositionalWeightScale = preset.AICompetence
}

// NewDefaultDifficultyManager returns a DifficultyManager with Medium-equivalent defaults.
// All master knobs are identity values (1.0/1.0/1.0/0). Useful for tests that don't load JSON.
func NewDefaultDifficultyManager() *DifficultyManager {
	preset := &DifficultyPreset{
		Name:                DifficultyMedium,
		CombatIntensity:     1.0,
		OverworldPressure:   1.0,
		AICompetence:        1.0,
		EncounterSizeOffset: 0,
	}
	deriveDifficultyValues(preset)

	dm := &DifficultyManager{
		presets: map[string]*DifficultyPreset{
			DifficultyMedium: preset,
		},
	}
	dm.current.Store(preset)
	return dm
}

// ReadDifficultyConfig loads difficulty configuration from JSON and initializes GlobalDifficulty.
func ReadDifficultyConfig() {
	data, err := os.ReadFile(AssetPath("gamedata/difficultyconfig.json"))
	if err != nil {
		panic(err)
	}

	var configData DifficultyConfigData
	err = json.Unmarshal(data, &configData)
	if err != nil {
		panic(err)
	}

	validateDifficultyConfig(&configData)

	// Build preset map and derive values
	presets := make(map[string]*DifficultyPreset, len(configData.Difficulties))
	for i := range configData.Difficulties {
		preset := &configData.Difficulties[i]
		deriveDifficultyValues(preset)
		presets[preset.Name] = preset
	}

	dm := &DifficultyManager{
		presets: presets,
	}

	// Set default difficulty
	defaultPreset, ok := presets[configData.DefaultDifficulty]
	if !ok {
		panic("default difficulty not found: " + configData.DefaultDifficulty)
	}
	dm.current.Store(defaultPreset)

	GlobalDifficulty = dm

	println("Difficulty config loaded:", len(presets), "presets, default:", configData.DefaultDifficulty)
}

// validateDifficultyConfig checks that the difficulty config is well-formed.
func validateDifficultyConfig(config *DifficultyConfigData) {
	if len(config.Difficulties) == 0 {
		panic("difficulty config must have at least one preset")
	}

	// Required presets
	required := map[string]bool{
		DifficultyEasy:   false,
		DifficultyMedium: false,
		DifficultyHard:   false,
	}

	seenNames := make(map[string]bool)
	for _, preset := range config.Difficulties {
		if preset.Name == "" {
			panic("difficulty preset missing name")
		}
		if seenNames[preset.Name] {
			panic("duplicate difficulty preset name: " + preset.Name)
		}
		seenNames[preset.Name] = true

		if _, exists := required[preset.Name]; exists {
			required[preset.Name] = true
		}

		// Validate master knobs
		if preset.CombatIntensity <= 0 {
			panic("difficulty " + preset.Name + ": combatIntensity must be positive")
		}
		if preset.OverworldPressure <= 0 {
			panic("difficulty " + preset.Name + ": overworldPressure must be positive")
		}
		if preset.AICompetence <= 0 {
			panic("difficulty " + preset.Name + ": aiCompetence must be positive")
		}
	}

	for name, found := range required {
		if !found {
			panic("missing required difficulty preset: " + name)
		}
	}

	if config.DefaultDifficulty == "" {
		panic("defaultDifficulty must be set")
	}
	if !seenNames[config.DefaultDifficulty] {
		panic("defaultDifficulty references unknown preset: " + config.DefaultDifficulty)
	}
}
