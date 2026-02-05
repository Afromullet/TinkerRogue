package templates

// We are not creating the entities yet, so we use the JSON struct to store the template data.
var MonsterTemplates []JSONMonster
var MeleeWeaponTemplates []JSONMeleeWeapon
var RangedWeaponTemplates []JSONRangedWeapon
var ConsumableTemplates []JSONAttributeModifier
var CreatureModifierTemplates []JSONCreatureModifier
var EncounterDifficultyTemplates []JSONEncounterDifficulty
var SquadTypeTemplates []JSONSquadType
var AIConfigTemplate JSONAIConfig
var PowerConfigTemplate JSONPowerConfig
var OverworldConfigTemplate JSONOverworldConfig
var ThreatDefinitionTemplates []JSONThreatDefinition
var DefaultThreatTemplate *JSONDefaultThreat
var FactionArchetypeTemplates map[string]FactionArchetypeConfig

// Node definition templates (new system)
var NodeDefinitionTemplates []JSONNodeDefinition
var DefaultNodeTemplate *JSONDefaultNode
var NodeCategories []string

// Encounter definition templates (combat-only, replacing ThreatDefinitions for combat)
var EncounterDefinitionTemplates []JSONEncounterDefinition
var DefaultEncounterTemplate *JSONDefaultEncounter

func ReadGameData() {

	ReadMonsterData()
	ReadWeaponData()
	ReadConsumableData()
	ReadCreatureModifiers()
	ReadNodeDefinitions() // Load node definitions first
	ReadEncounterData()   // Then encounter data (validates links)
	ReadAIConfig()
	ReadPowerConfig()
	ReadOverworldConfig()

}
