package templates

import "fmt"

// registry.go declares the package-level registry variables populated by
// ReadGameData(). Five load-order constraints are currently encoded in
// ReadGameData()'s phase list (and only there — no compile-time enforcement):
//
//  1. GameConfig and Difficulty load first; every other subsystem may read them.
//  2. NodeDefinitions before EncounterData (validateNodeEncounterLinks reads both;
//     validateEncounterDefinitions consults NodeDefinitionTemplates for the
//     required-IDs set via requiredNodeIDsFrom).
//  3. Spells before UnitSpells (UnitSpells validates spell IDs exist in registry).
//  4. InitialSetup last (references squad and faction types from earlier configs).
//  5. MapGenConfig is the only optional loader; absence is a warning, not fatal.
//
// Registries are mutated only by their loaders; all 56 downstream consumers
// read them after ReadGameData() returns.

// We are not creating the entities yet, so we use the JSON struct to store the template data.
var MonsterTemplates []JSONMonster
var EncounterDifficultyTemplates []JSONEncounterDifficulty
var AIConfigTemplate JSONAIConfig
var PowerConfigTemplate JSONPowerConfig
var OverworldConfigTemplate JSONOverworldConfig
var FactionArchetypeTemplates map[string]FactionArchetypeConfig

// Node definition templates (new system)
var NodeDefinitionTemplates []JSONNodeDefinition
var DefaultNodeTemplate *JSONDefaultNode
var NodeCategories []string

// Encounter definition templates (combat-only configuration)
var EncounterDefinitionTemplates []JSONEncounterDefinition

// Influence interaction configuration
var InfluenceConfigTemplate JSONInfluenceConfig

// Map generation configuration (optional — nil means use code defaults)
var MapGenConfigTemplate *JSONMapGenConfig

// Name generation configuration
var NameConfigTemplate JSONNameConfig

// Initial setup configuration (commanders, squads, roster units, factions at game start)
var InitialSetupTemplate JSONInitialSetup

// ReadGameData loads all static game data from JSON files.
//
// Load order is significant — see the phase comments below. Returns the first
// error encountered; partial loads are not rolled back (registries that loaded
// before the failure remain populated).
func ReadGameData() error {
	loaders := []struct {
		name string
		fn   func() error
	}{
		// Phase A: config that everything else depends on
		{"gameconfig", ReadGameConfig},
		{"difficulty", ReadDifficultyConfig},
		{"monsters", ReadMonsterData},
		{"names", ReadNameData},
		// Phase B: nodes before encounters (validateNodeEncounterLinks reads both)
		{"nodes", ReadNodeDefinitions},
		{"encounters", ReadEncounterData},
		// Phase C: independent subsystem configs
		{"ai", ReadAIConfig},
		{"power", ReadPowerConfig},
		{"overworld", ReadOverworldConfig},
		{"influence", ReadInfluenceConfig},
		{"mapgen", ReadMapGenConfig},
		// Phase D: spells before unit-spells (unit-spells validates spell IDs exist)
		{"spells", LoadSpellDefinitions},
		{"unitspells", LoadUnitSpellDefinitions},
		{"artifacts", LoadArtifactDefinitions},
		// Phase E: initial setup last (references squad and faction types from earlier configs)
		{"initialsetup", ReadInitialSetupConfig},
	}
	for _, l := range loaders {
		if err := l.fn(); err != nil {
			return fmt.Errorf("ReadGameData: %s: %w", l.name, err)
		}
	}
	return nil
}
