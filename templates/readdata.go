package templates

import (
	"encoding/json"
	"game_main/core/config"
	"os"
)

// AssetPath delegates to config.AssetPath for working-directory-independent asset resolution.
func AssetPath(relative string) string {
	return config.AssetPath(relative)
}

// readAndUnmarshal reads a JSON file and unmarshals it into target. Panics on error.
func readAndUnmarshal[T any](path string, target *T) {
	data, err := os.ReadFile(AssetPath(path))
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		panic(err)
	}
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

func ReadMonsterData() {
	var monsters MonstersData
	readAndUnmarshal("gamedata/monsterdata.json", &monsters)
	MonsterTemplates = append(MonsterTemplates, monsters.Monsters...)
}

func ReadNodeDefinitions() {
	var nodeData NodeDefinitionsData
	readAndUnmarshal("gamedata/nodeDefinitions.json", &nodeData)

	validateNodeDefinitions(&nodeData)

	NodeDefinitionTemplates = nodeData.Nodes
	DefaultNodeTemplate = nodeData.DefaultNode
	NodeCategories = nodeData.NodeCategories

	println("Node definitions loaded:", len(NodeDefinitionTemplates), "nodes,",
		len(NodeCategories), "categories")
}

func ReadEncounterData() {
	var encounterData EncounterDataWithNew
	readAndUnmarshal("gamedata/encounterdata.json", &encounterData)

	// Validate difficulty levels are sequential (1-5)
	for i, diff := range encounterData.DifficultyLevels {
		expectedLevel := i + 1
		if diff.Level != expectedLevel {
			panic("Invalid difficulty level sequence: expected " + string(rune(expectedLevel+'0')) + ", got " + string(rune(diff.Level+'0')))
		}
	}

	// Build valid squad types map
	validSquadTypes := make(map[string]bool)
	for _, squadType := range encounterData.SquadTypes {
		validSquadTypes[squadType.ID] = true
	}

	if len(encounterData.EncounterDefinitions) > 0 {
		validateEncounterDefinitions(&encounterData, validSquadTypes)
	}

	EncounterDifficultyTemplates = encounterData.DifficultyLevels
	FactionArchetypeTemplates = encounterData.Factions
	EncounterDefinitionTemplates = encounterData.EncounterDefinitions

	if len(NodeDefinitionTemplates) > 0 && len(EncounterDefinitionTemplates) > 0 {
		validateNodeEncounterLinks()
	}

	println("Encounter data loaded:", len(EncounterDifficultyTemplates), "difficulty levels,",
		len(EncounterDefinitionTemplates), "encounter definitions,",
		len(FactionArchetypeTemplates), "factions")
}

func ReadNameData() {
	readAndUnmarshal("gamedata/namedata.json", &NameConfigTemplate)
	validateNameConfig(&NameConfigTemplate)
	println("Name config loaded:", len(NameConfigTemplate.Pools), "pools")
}

func ReadAIConfig() {
	readAndUnmarshal("gamedata/aiconfig.json", &AIConfigTemplate)
	validateAIConfig(&AIConfigTemplate)
	println("AI config loaded:", len(AIConfigTemplate.RoleBehaviors), "role behaviors")
}

func ReadPowerConfig() {
	readAndUnmarshal("gamedata/powerconfig.json", &PowerConfigTemplate)
	validatePowerConfig(&PowerConfigTemplate)
	println("Power config loaded:", len(PowerConfigTemplate.Profiles), "profiles,",
		len(PowerConfigTemplate.RoleMultipliers), "role multipliers")
}

func ReadInfluenceConfig() {
	readAndUnmarshal("gamedata/influenceconfig.json", &InfluenceConfigTemplate)
	validateInfluenceConfig(&InfluenceConfigTemplate)
	println("Influence config loaded")
}

// ReadMapGenConfig loads map generation configuration from JSON.
// This file is optional — if missing, generators use their code defaults.
func ReadMapGenConfig() {
	data, err := os.ReadFile(AssetPath("gamedata/mapgenconfig.json"))
	if err != nil {
		println("Map gen config not found, using code defaults")
		return
	}

	var config JSONMapGenConfig
	if err := json.Unmarshal(data, &config); err != nil {
		panic("Failed to parse mapgenconfig.json: " + err.Error())
	}

	validateMapGenConfig(&config)

	MapGenConfigTemplate = &config
	println("Map gen config loaded")
}

func ReadOverworldConfig() {
	readAndUnmarshal("gamedata/overworldconfig.json", &OverworldConfigTemplate)
	validateOverworldConfig(&OverworldConfigTemplate)
	println("Overworld config loaded")
}

func ReadInitialSetupConfig() {
	readAndUnmarshal("gamedata/initialsetup.json", &InitialSetupTemplate)
	validateInitialSetup(&InitialSetupTemplate)
	println("Initial setup config loaded:",
		len(InitialSetupTemplate.Commanders), "commanders,",
		len(InitialSetupTemplate.Factions.Entries), "factions")
}
