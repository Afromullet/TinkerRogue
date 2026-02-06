package templates

import (
	"encoding/json"

	"os"
)

type MonstersData struct {
	Monsters []JSONMonster `json:"monsters"`
}

// WeaponList struct to hold all weapons
type WeaponData struct {
	Weps []JSONWeapon `json:"weapons"` // List of weapons
}

type MeleeWeapons struct {
	Weapons []JSONMeleeWeapon
}

type RangedWeapons struct {
	Weapons []JSONRangedWeapon
}

type ConsumableData struct {
	Consumables []JSONAttributeModifier
}

type CreatureModifiers struct {
	CreatureMods []JSONCreatureModifier
}

// EncounterDataWithNew is the root container for encounter configuration
type EncounterDataWithNew struct {
	Factions             map[string]FactionArchetypeConfig `json:"factions"`
	DifficultyLevels     []JSONEncounterDifficulty         `json:"difficultyLevels"`
	SquadTypes           []JSONSquadType                   `json:"squadTypes"`
	EncounterDefinitions []JSONEncounterDefinition         `json:"encounterDefinitions"`
	DefaultEncounter     *JSONDefaultEncounter             `json:"defaultEncounter"`
}

func ReadMonsterData() {
	data, err := os.ReadFile("../assets//gamedata/monsterdata.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var monsters MonstersData
	err = json.Unmarshal(data, &monsters)

	if err != nil {
		panic(err)
	}

	// Iterate over monsters
	for _, monster := range monsters.Monsters {
		MonsterTemplates = append(MonsterTemplates, NewJSONMonster(monster))
	}

}

func ReadWeaponData() {
	data, err := os.ReadFile("../assets//gamedata/weapondata.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var weaponData WeaponData
	err = json.Unmarshal(data, &weaponData)

	if err != nil {
		panic(err)
	}

	// Iterate over monsters
	for _, w := range weaponData.Weps {

		if w.Type == "MeleeWeapon" {
			wep := NewJSONMeleeWeapon(w)
			MeleeWeaponTemplates = append(MeleeWeaponTemplates, wep)

		} else if w.Type == "RangedWeapon" {

			wep := NewJSONRangedWeapon(w)
			RangedWeaponTemplates = append(RangedWeaponTemplates, wep)

		} else {
			// ERROR HANDLING IN FUTURE
		}
	}

}

func ReadConsumableData() {
	data, err := os.ReadFile("../assets//gamedata/consumabledata.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var consumables ConsumableData
	err = json.Unmarshal(data, &consumables)

	if err != nil {
		panic(err)
	}

	// Iterate over monsters
	for _, c := range consumables.Consumables {

		ConsumableTemplates = append(ConsumableTemplates, NewJSONAttributeModifier(c))

	}

}

func ReadCreatureModifiers() {
	data, err := os.ReadFile("../assets//gamedata/creaturemodifiers.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var mod CreatureModifiers
	err = json.Unmarshal(data, &mod)

	if err != nil {
		panic(err)
	}

	// Iterate over monsters
	for _, c := range mod.CreatureMods {

		CreatureModifierTemplates = append(CreatureModifierTemplates, CreatureModifierFromJSON(c))

	}

}

func ReadNodeDefinitions() {
	data, err := os.ReadFile("../assets//gamedata/nodeDefinitions.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var nodeData NodeDefinitionsData
	err = json.Unmarshal(data, &nodeData)

	if err != nil {
		panic(err)
	}

	// Validate node definitions
	validateNodeDefinitions(&nodeData)

	// Store in global template arrays
	NodeDefinitionTemplates = nodeData.Nodes
	DefaultNodeTemplate = nodeData.DefaultNode
	NodeCategories = nodeData.NodeCategories

	// Log successful load
	println("Node definitions loaded:", len(NodeDefinitionTemplates), "nodes,",
		len(NodeCategories), "categories")
}

func validateNodeDefinitions(data *NodeDefinitionsData) {
	seenIDs := make(map[string]bool)

	// Build valid categories map
	validCategories := make(map[string]bool)
	for _, cat := range data.NodeCategories {
		validCategories[cat] = true
	}

	// Required node IDs for backwards compatibility with existing enum
	requiredNodes := map[string]bool{
		"necromancer": false,
		"banditcamp":  false,
		"corruption":  false,
		"beastnest":   false,
		"orcwarband":  false,
	}

	// Valid primary effects
	validEffects := map[string]bool{
		"SpawnBoost":        true,
		"ResourceDrain":     true,
		"TerrainCorruption": true,
		"CombatDebuff":      true,
		"":                  true, // Allow empty for non-combat nodes
	}

	for _, node := range data.Nodes {
		// Required fields
		if node.ID == "" {
			panic("Node definition missing required 'id' field")
		}
		if node.DisplayName == "" {
			panic("Node definition '" + node.ID + "' missing required 'displayName' field")
		}
		if node.Category == "" {
			panic("Node definition '" + node.ID + "' missing required 'category' field")
		}

		// Check for duplicate IDs
		if seenIDs[node.ID] {
			panic("Duplicate node definition ID: " + node.ID)
		}
		seenIDs[node.ID] = true

		// Validate category
		if !validCategories[node.Category] {
			panic("Node '" + node.ID + "' has invalid category: " + node.Category)
		}

		// Validate primary effect
		if !validEffects[node.Overworld.PrimaryEffect] {
			panic("Node '" + node.ID + "' has invalid primary effect: " + node.Overworld.PrimaryEffect)
		}

		// Threat nodes must have a factionId
		if node.Category == "threat" && node.FactionID == "" {
			panic("Threat node '" + node.ID + "' missing required 'factionId' field")
		}

		// Warn about invisible color
		if node.Color.A == 0 {
			println("Warning: Node '" + node.ID + "' has zero alpha (invisible)")
		}

		// Mark required nodes as found
		if _, exists := requiredNodes[node.ID]; exists {
			requiredNodes[node.ID] = true
		}
	}

	// Check all required nodes exist
	for id, found := range requiredNodes {
		if !found {
			panic("Missing required node definition: " + id)
		}
	}

	// Validate default node
	if data.DefaultNode == nil {
		panic("Missing defaultNode in nodeDefinitions.json")
	}
}

func ReadEncounterData() {
	data, err := os.ReadFile("../assets//gamedata/encounterdata.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON with extended struct to support both old and new format
	var encounterData EncounterDataWithNew
	err = json.Unmarshal(data, &encounterData)

	if err != nil {
		panic(err)
	}

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

	// Validate encounter definitions
	if len(encounterData.EncounterDefinitions) > 0 {
		validateEncounterDefinitions(&encounterData, validSquadTypes)
	}

	// Store in global template arrays
	EncounterDifficultyTemplates = encounterData.DifficultyLevels
	SquadTypeTemplates = encounterData.SquadTypes
	FactionArchetypeTemplates = encounterData.Factions

	// Store new encounter definitions
	EncounterDefinitionTemplates = encounterData.EncounterDefinitions
	DefaultEncounterTemplate = encounterData.DefaultEncounter

	// Cross-validate node-encounter links if both are loaded
	if len(NodeDefinitionTemplates) > 0 && len(EncounterDefinitionTemplates) > 0 {
		validateNodeEncounterLinks()
	}

	// Log successful load
	println("Encounter data loaded:", len(EncounterDifficultyTemplates), "difficulty levels,",
		len(SquadTypeTemplates), "squad types,",
		len(EncounterDefinitionTemplates), "encounter definitions,",
		len(FactionArchetypeTemplates), "factions")
}

// validateEncounterDefinitions validates the new encounter definitions format
// NOTE: Multiple encounters per faction are explicitly supported (e.g., basic/elite/boss variants)
func validateEncounterDefinitions(data *EncounterDataWithNew, validSquadTypes map[string]bool) {
	seenIDs := make(map[string]bool)
	seenEncounterTypeIDs := make(map[string]bool)

	// Required encounter IDs for backwards compatibility
	requiredEncounters := map[string]bool{
		"necromancer": false,
		"banditcamp":  false,
		"corruption":  false,
		"beastnest":   false,
		"orcwarband":  false,
	}

	// Track encounters per faction to log multi-encounter factions
	encountersPerFaction := make(map[string][]string)

	for _, encounter := range data.EncounterDefinitions {
		// Required fields
		if encounter.ID == "" {
			panic("Encounter definition missing required 'id' field")
		}
		if encounter.EncounterTypeID == "" {
			panic("Encounter definition '" + encounter.ID + "' missing required 'encounterTypeId' field")
		}

		// Check for duplicate IDs
		if seenIDs[encounter.ID] {
			panic("Duplicate encounter definition ID: " + encounter.ID)
		}
		seenIDs[encounter.ID] = true

		// Check for duplicate encounter type IDs
		if seenEncounterTypeIDs[encounter.EncounterTypeID] {
			panic("Duplicate encounterTypeId: " + encounter.EncounterTypeID)
		}
		seenEncounterTypeIDs[encounter.EncounterTypeID] = true

		// Validate squad preferences reference valid squad types
		for _, pref := range encounter.SquadPreferences {
			if !validSquadTypes[pref] {
				panic("Encounter '" + encounter.ID + "' references invalid squad type: " + pref)
			}
		}

		// Validate factionId references an existing faction
		if encounter.FactionID != "" {
			if _, exists := data.Factions[encounter.FactionID]; !exists {
				panic("Encounter '" + encounter.ID + "' references unknown faction: " + encounter.FactionID)
			}
			// Track encounters per faction
			encountersPerFaction[encounter.FactionID] = append(encountersPerFaction[encounter.FactionID], encounter.ID)
		}

		// Mark required encounters as found
		if _, exists := requiredEncounters[encounter.ID]; exists {
			requiredEncounters[encounter.ID] = true
		}
	}

	// Check all required encounters exist
	for id, found := range requiredEncounters {
		if !found {
			panic("Missing required encounter definition: " + id)
		}
	}

	// Log factions with multiple encounters (informational, not an error)
	for factionID, encounterIDs := range encountersPerFaction {
		if len(encounterIDs) > 1 {
			println("Faction '" + factionID + "' has multiple encounters:", len(encounterIDs))
		}
	}
}

// validateNodeEncounterLinks cross-validates that threat nodes' factions have encounters
func validateNodeEncounterLinks() {
	// Build faction → encounter count lookup
	encountersPerFaction := make(map[string]int)
	for _, enc := range EncounterDefinitionTemplates {
		if enc.FactionID != "" {
			encountersPerFaction[enc.FactionID]++
		}
	}

	// Validate each threat node's faction has at least one encounter
	for _, node := range NodeDefinitionTemplates {
		if node.Category == "threat" && node.FactionID != "" {
			if encountersPerFaction[node.FactionID] == 0 {
				panic("Threat node '" + node.ID + "' has factionId '" + node.FactionID + "' but no encounters exist for that faction")
			}
		}
	}

	println("Node-encounter links validated successfully")
}

func ReadAIConfig() {
	data, err := os.ReadFile("../assets//gamedata/aiconfig.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	err = json.Unmarshal(data, &AIConfigTemplate)
	if err != nil {
		panic(err)
	}

	// Validate
	validateAIConfig(&AIConfigTemplate)

	// Log successful load
	println("AI config loaded:", len(AIConfigTemplate.RoleBehaviors), "role behaviors")
}

func validateAIConfig(config *JSONAIConfig) {
	// Validate role behaviors exist for required roles
	requiredRoles := map[string]bool{"Tank": false, "DPS": false, "Support": false}
	for _, rb := range config.RoleBehaviors {
		if _, exists := requiredRoles[rb.Role]; exists {
			requiredRoles[rb.Role] = true
		}
	}
	for role, found := range requiredRoles {
		if !found {
			panic("AI config missing role behavior for: " + role)
		}
	}

	// Validate all weights are in valid range [-1.0, 1.0]
	for _, rb := range config.RoleBehaviors {
		if rb.MeleeWeight < -1.0 || rb.MeleeWeight > 1.0 ||
			rb.SupportWeight < -1.0 || rb.SupportWeight > 1.0 {
			panic("Role behavior weights must be between -1.0 and 1.0 for role: " + rb.Role)
		}
	}

	// Validate distance thresholds are positive
	tc := config.ThreatCalculation
	if tc.FlankingThreatRangeBonus <= 0 || tc.IsolationThreshold <= 0 ||
		tc.RetreatSafeThreatThreshold <= 0 {
		panic("All threat calculation distances must be positive")
	}

	// Validate support layer parameters are positive
	sl := config.SupportLayer
	if sl.HealRadius <= 0 || sl.BuffPriorityEngagementRange <= 0 {
		panic("All support layer parameters must be positive")
	}
}

func ReadPowerConfig() {
	data, err := os.ReadFile("../assets//gamedata/powerconfig.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	err = json.Unmarshal(data, &PowerConfigTemplate)
	if err != nil {
		panic(err)
	}

	// Validate
	validatePowerConfig(&PowerConfigTemplate)

	// Log successful load
	println("Power config loaded:", len(PowerConfigTemplate.Profiles), "profiles,",
		len(PowerConfigTemplate.RoleMultipliers), "role multipliers")
}

func validatePowerConfig(config *JSONPowerConfig) {
	// Validate required profiles exist (only Balanced is required)
	requiredProfiles := map[string]bool{"Balanced": false}
	for _, profile := range config.Profiles {
		if _, exists := requiredProfiles[profile.Name]; exists {
			requiredProfiles[profile.Name] = true
		}

		// Validate category weights sum to ~1.0
		categoryTotal := profile.OffensiveWeight + profile.DefensiveWeight + profile.UtilityWeight
		if categoryTotal < 0.99 || categoryTotal > 1.01 {
			panic("Profile '" + profile.Name + "' category weights must sum to 1.0")
		}

		// Validate all weights are non-negative
		if profile.OffensiveWeight < 0 || profile.DefensiveWeight < 0 || profile.UtilityWeight < 0 {
			panic("Profile '" + profile.Name + "' weights must be non-negative")
		}

		// Validate health penalty is positive
		if profile.HealthPenalty <= 0 {
			panic("Profile '" + profile.Name + "' health penalty must be positive")
		}
	}

	for profile, found := range requiredProfiles {
		if !found {
			panic("Power config missing required profile: " + profile)
		}
	}

	// Validate role multipliers exist for required roles
	requiredRoles := map[string]bool{"Tank": false, "DPS": false, "Support": false}
	for _, rm := range config.RoleMultipliers {
		if _, exists := requiredRoles[rm.Role]; exists {
			requiredRoles[rm.Role] = true
		}
		if rm.Multiplier <= 0 {
			panic("Role multiplier must be positive for role: " + rm.Role)
		}
	}
	for role, found := range requiredRoles {
		if !found {
			panic("Power config missing role multiplier for: " + role)
		}
	}

	// Validate composition bonuses exist for 1-4 unique types
	requiredTypes := map[int]bool{1: false, 2: false, 3: false, 4: false}
	for _, cb := range config.CompositionBonuses {
		if _, exists := requiredTypes[cb.UniqueTypes]; exists {
			requiredTypes[cb.UniqueTypes] = true
		}
	}
	for types, found := range requiredTypes {
		if !found {
			panic("Power config missing composition bonus for unique types: " + string(rune(types+'0')))
		}
	}

	// Validate leader bonus is positive
	if config.LeaderBonus <= 0 {
		panic("Leader bonus must be positive")
	}
}

func ReadOverworldConfig() {
	data, err := os.ReadFile("../assets//gamedata/overworldconfig.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	err = json.Unmarshal(data, &OverworldConfigTemplate)
	if err != nil {
		panic(err)
	}

	// Validate
	validateOverworldConfig(&OverworldConfigTemplate)

	// Log successful load
	println("Overworld config loaded")
}

func validateOverworldConfig(config *JSONOverworldConfig) {
	// Validate threat growth parameters are positive
	tg := config.ThreatGrowth
	if tg.ContainmentSlowdown <= 0 || tg.MaxThreatIntensity <= 0 ||
		tg.ChildNodeSpawnThreshold <= 0 || tg.MaxChildNodeSpawnAttempts <= 0 {
		panic("All threat growth parameters must be positive")
	}

	// Validate faction AI parameters are positive
	fa := config.FactionAI
	if fa.DefaultIntentTickDuration <= 0 ||
		fa.ExpansionTerritoryLimit <= 0 ||
		fa.FortificationStrengthGain <= 0 ||
		fa.MaxTerritorySize <= 0 {
		panic("All faction AI parameters must be positive")
	}

	// Validate strength thresholds
	st := config.StrengthThresholds
	if st.Weak <= 0 || st.Strong <= 0 || st.Critical < 0 {
		panic("Strength thresholds must be positive (critical can be 0)")
	}
	if st.Critical > st.Weak || st.Weak >= st.Strong {
		panic("Strength thresholds must be: critical <= weak < strong")
	}

	// Note: Faction archetypes are now validated in encounterdata.json

	// Validate victory conditions
	vc := config.VictoryConditions
	if vc.HighIntensityThreshold <= 0 || vc.MaxHighIntensityThreats <= 0 || vc.MaxThreatInfluence <= 0 {
		panic("Victory condition thresholds must be positive")
	}

	// Validate faction scoring control
	fsc := config.FactionScoringControl
	if fsc.IdleScoreThreshold <= 0 || fsc.RaidBaseIntensity <= 0 || fsc.RaidIntensityScale <= 0 {
		panic("Faction scoring control parameters must be positive")
	}

	// Validate spawn probabilities are in valid range [0-100]
	sp := config.SpawnProbabilities
	if sp.ExpansionThreatSpawnChance < 0 || sp.ExpansionThreatSpawnChance > 100 ||
		sp.FortifyThreatSpawnChance < 0 || sp.FortifyThreatSpawnChance > 100 ||
		sp.BonusItemDropChance < 0 || sp.BonusItemDropChance > 100 {
		panic("Spawn probabilities must be between 0 and 100")
	}

	// Validate map dimensions are positive
	md := config.MapDimensions
	if md.DefaultMapWidth <= 0 || md.DefaultMapHeight <= 0 {
		panic("Map dimensions must be positive")
	}

	// Validate strategy bonuses
	requiredStrategies := []string{"Expansionist", "Aggressor", "Raider", "Defensive", "Territorial"}
	for _, strategy := range requiredStrategies {
		if _, ok := config.StrategyBonuses[strategy]; !ok {
			panic("Missing required strategy bonus: " + strategy)
		}
	}
}
