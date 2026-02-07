package templates

// We are not creating the entities yet, so we use the JSON struct to store the template data.
var MonsterTemplates []JSONMonster
var ConsumableTemplates []JSONAttributeModifier

var EncounterDifficultyTemplates []JSONEncounterDifficulty
var SquadTypeTemplates []JSONSquadType
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
var DefaultEncounterTemplate *JSONDefaultEncounter

func ReadGameData() {

	ReadMonsterData()
	ReadConsumableData()

	ReadNodeDefinitions() // Load node definitions first
	ReadEncounterData()   // Then encounter data (validates links)
	ReadAIConfig()
	ReadPowerConfig()
	ReadOverworldConfig()

}
