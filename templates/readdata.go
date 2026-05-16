package templates

import (
	"encoding/json"
	"fmt"
	"game_main/core/config"
	"log"
	"os"
)

// AssetPath delegates to config.AssetPath for working-directory-independent asset resolution.
func AssetPath(relative string) string {
	return config.AssetPath(relative)
}

type MonstersData struct {
	Monsters []JSONMonster `json:"monsters"`
}

// EncounterDataWithNew is the root container for encounter configuration
type EncounterDataWithNew struct {
	Factions             map[string]FactionArchetypeConfig `json:"factions"`
	DifficultyLevels     []JSONEncounterDifficulty         `json:"difficultyLevels"`
	SquadTypes           []JSONSquadType                   `json:"squadTypes"`
	EncounterDefinitions []JSONEncounterDefinition         `json:"encounterDefinitions"`
}

// Loader declarations. Each names its config, points at a *Path constant, and
// (where applicable) wires in a validator. Optional=true loaders log a warning
// when the file is missing rather than failing.

var monsterDataLoader = Loader[MonstersData]{
	Name: "monsters",
	Path: MonsterDataPath,
}

var nodeDefinitionsLoader = Loader[NodeDefinitionsData]{
	Name:     "nodes",
	Path:     NodeDefinitionsPath,
	Validate: validateNodeDefinitions,
}

var encounterDataLoader = Loader[EncounterDataWithNew]{
	Name: "encounters",
	Path: EncounterDataPath,
	// Validation runs in ReadEncounterData because it needs both the loaded
	// data and a derived validSquadTypes map.
}

var nameDataLoader = Loader[JSONNameConfig]{
	Name:     "names",
	Path:     NameDataPath,
	Validate: validateNameConfig,
}

var aiConfigLoader = Loader[JSONAIConfig]{
	Name:     "ai",
	Path:     AIConfigPath,
	Validate: validateAIConfig,
}

var powerConfigLoader = Loader[JSONPowerConfig]{
	Name:     "power",
	Path:     PowerConfigPath,
	Validate: validatePowerConfig,
}

var influenceConfigLoader = Loader[JSONInfluenceConfig]{
	Name:     "influence",
	Path:     InfluenceConfigPath,
	Validate: validateInfluenceConfig,
}

var overworldConfigLoader = Loader[JSONOverworldConfig]{
	Name:     "overworld",
	Path:     OverworldConfigPath,
	Validate: validateOverworldConfig,
}

var initialSetupLoader = Loader[JSONInitialSetup]{
	Name:     "initialsetup",
	Path:     InitialSetupPath,
	Validate: validateInitialSetup,
}

func ReadMonsterData() error {
	data, err := monsterDataLoader.Load()
	if err != nil {
		return err
	}
	MonsterTemplates = append(MonsterTemplates, data.Monsters...)
	log.Printf("[templates] monsters loaded: %d entries", len(data.Monsters))
	return nil
}

func ReadNodeDefinitions() error {
	nodeData, err := nodeDefinitionsLoader.Load()
	if err != nil {
		return err
	}

	NodeDefinitionTemplates = nodeData.Nodes
	DefaultNodeTemplate = nodeData.DefaultNode
	NodeCategories = nodeData.NodeCategories

	log.Printf("[templates] nodes loaded: %d nodes, %d categories",
		len(NodeDefinitionTemplates), len(NodeCategories))
	return nil
}

func ReadEncounterData() error {
	encounterData, err := encounterDataLoader.Load()
	if err != nil {
		return err
	}

	// Validate difficulty levels are sequential (1-5)
	for i, diff := range encounterData.DifficultyLevels {
		expectedLevel := i + 1
		if diff.Level != expectedLevel {
			return fmt.Errorf("templates: encounters: invalid difficulty level sequence: expected %d, got %d",
				expectedLevel, diff.Level)
		}
	}

	// Build valid squad types map
	validSquadTypes := make(map[string]bool)
	for _, squadType := range encounterData.SquadTypes {
		validSquadTypes[squadType.ID] = true
	}

	if len(encounterData.EncounterDefinitions) > 0 {
		if err := validateEncounterDefinitions(&encounterData, validSquadTypes); err != nil {
			return fmt.Errorf("templates: validate encounters: %w", err)
		}
	}

	EncounterDifficultyTemplates = encounterData.DifficultyLevels
	FactionArchetypeTemplates = encounterData.Factions
	EncounterDefinitionTemplates = encounterData.EncounterDefinitions

	if len(NodeDefinitionTemplates) > 0 && len(EncounterDefinitionTemplates) > 0 {
		if err := validateNodeEncounterLinks(); err != nil {
			return fmt.Errorf("templates: validate node-encounter links: %w", err)
		}
	}

	log.Printf("[templates] encounters loaded: %d difficulty levels, %d definitions, %d factions",
		len(EncounterDifficultyTemplates), len(EncounterDefinitionTemplates), len(FactionArchetypeTemplates))
	return nil
}

func ReadNameData() error {
	cfg, err := nameDataLoader.Load()
	if err != nil {
		return err
	}
	NameConfigTemplate = cfg
	log.Printf("[templates] names loaded: %d pools", len(NameConfigTemplate.Pools))
	return nil
}

func ReadAIConfig() error {
	cfg, err := aiConfigLoader.Load()
	if err != nil {
		return err
	}
	AIConfigTemplate = cfg
	log.Printf("[templates] ai loaded: %d role behaviors", len(AIConfigTemplate.RoleBehaviors))
	return nil
}

func ReadPowerConfig() error {
	cfg, err := powerConfigLoader.Load()
	if err != nil {
		return err
	}
	PowerConfigTemplate = cfg
	log.Printf("[templates] power loaded: %d profiles, %d role multipliers",
		len(PowerConfigTemplate.Profiles), len(PowerConfigTemplate.RoleMultipliers))
	return nil
}

func ReadInfluenceConfig() error {
	cfg, err := influenceConfigLoader.Load()
	if err != nil {
		return err
	}
	InfluenceConfigTemplate = cfg
	log.Printf("[templates] influence loaded")
	return nil
}

// ReadMapGenConfig loads map generation configuration from JSON.
// This file is optional: if missing, MapGenConfigTemplate stays nil and
// generators fall back to their code defaults. Inlined (not Loader-backed)
// because the nil-pointer sentinel needs to distinguish "file missing" from
// "file present but empty" — Loader[T]'s Optional path can't express that.
func ReadMapGenConfig() error {
	data, err := os.ReadFile(AssetPath(MapGenConfigPath))
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[templates] mapgen: file not found at %s, using code defaults", MapGenConfigPath)
			return nil
		}
		return fmt.Errorf("templates: read mapgen (%s): %w", MapGenConfigPath, err)
	}
	var cfg JSONMapGenConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("templates: parse mapgen (%s): %w", MapGenConfigPath, err)
	}
	if err := validateMapGenConfig(&cfg); err != nil {
		return fmt.Errorf("templates: validate mapgen: %w", err)
	}
	MapGenConfigTemplate = &cfg
	log.Printf("[templates] mapgen loaded")
	return nil
}

func ReadOverworldConfig() error {
	cfg, err := overworldConfigLoader.Load()
	if err != nil {
		return err
	}
	OverworldConfigTemplate = cfg
	log.Printf("[templates] overworld loaded")
	return nil
}

func ReadInitialSetupConfig() error {
	cfg, err := initialSetupLoader.Load()
	if err != nil {
		return err
	}
	InitialSetupTemplate = cfg
	log.Printf("[templates] initialsetup loaded: %d commanders, %d factions",
		len(InitialSetupTemplate.Commanders), len(InitialSetupTemplate.Factions.Entries))
	return nil
}
