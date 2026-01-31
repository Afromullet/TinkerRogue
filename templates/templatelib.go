package templates

// We are not creating the entities yet, so we use the JSON struct to store the template data.
var MonsterTemplates []JSONMonster
var MeleeWeaponTemplates []JSONMeleeWeapon
var RangedWeaponTemplates []JSONRangedWeapon
var ConsumableTemplates []JSONAttributeModifier
var CreatureModifierTemplates []JSONCreatureModifier
var EncounterDifficultyTemplates []JSONEncounterDifficulty
var EncounterTypeTemplates []JSONEncounterType
var SquadTypeTemplates []JSONSquadType
var AIConfigTemplate JSONAIConfig
var PowerConfigTemplate JSONPowerConfig
var OverworldConfigTemplate JSONOverworldConfig

func ReadGameData() {

	ReadMonsterData()
	ReadWeaponData()
	ReadConsumableData()
	ReadCreatureModifiers()
	ReadEncounterData()
	ReadAIConfig()
	ReadPowerConfig()
	ReadOverworldConfig()

}
