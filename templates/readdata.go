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

func ReadEncounterData() {
	data, err := os.ReadFile("../assets//gamedata/encounterdata.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var encounterData EncounterData
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

	// Validate encounter types reference valid squad types
	validSquadTypes := make(map[string]bool)
	for _, squadType := range encounterData.SquadTypes {
		validSquadTypes[squadType.ID] = true
	}

	for _, encounterType := range encounterData.EncounterTypes {
		for _, pref := range encounterType.SquadPreferences {
			if !validSquadTypes[pref] {
				panic("Encounter type '" + encounterType.ID + "' references invalid squad type: " + pref)
			}
		}
	}

	// Store in global template arrays
	EncounterDifficultyTemplates = encounterData.DifficultyLevels
	EncounterTypeTemplates = encounterData.EncounterTypes
	SquadTypeTemplates = encounterData.SquadTypes

	// Log successful load
	println("Encounter data loaded:", len(EncounterDifficultyTemplates), "difficulty levels,",
		len(EncounterTypeTemplates), "encounter types,", len(SquadTypeTemplates), "squad types")
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

	// Validate positional risk weights sum to ~1.0
	total := config.PositionalRisk.FlankingWeight +
		config.PositionalRisk.IsolationWeight +
		config.PositionalRisk.PressureWeight +
		config.PositionalRisk.RetreatWeight
	if total < 0.99 || total > 1.01 {
		panic("Positional risk weights must sum to 1.0")
	}

	// Validate all weights are in valid range [-1.0, 1.0]
	for _, rb := range config.RoleBehaviors {
		if rb.MeleeWeight < -1.0 || rb.MeleeWeight > 1.0 ||
			rb.RangedWeight < -1.0 || rb.RangedWeight > 1.0 ||
			rb.SupportWeight < -1.0 || rb.SupportWeight > 1.0 ||
			rb.PositionalWeight < -1.0 || rb.PositionalWeight > 1.0 {
			panic("Role behavior weights must be between -1.0 and 1.0 for role: " + rb.Role)
		}
	}

	// Validate distance thresholds are positive
	tc := config.ThreatCalculation
	if tc.FlankingThreatRangeBonus <= 0 || tc.IsolationSafeDistance <= 0 ||
		tc.IsolationModerateDistance <= 0 || tc.IsolationHighDistance <= 0 ||
		tc.EngagementPressureMax <= 0 || tc.RetreatSafeThreatThreshold <= 0 {
		panic("All threat calculation distances must be positive")
	}

	// Validate support layer parameters are positive
	sl := config.SupportLayer
	if sl.HealRadius <= 0 || sl.AllyProximityRadius <= 0 || sl.BuffPriorityEngagementRange <= 0 {
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
	// Validate required profiles exist
	requiredProfiles := map[string]bool{"Balanced": false, "Offensive": false, "Defensive": false}
	for _, profile := range config.Profiles {
		if _, exists := requiredProfiles[profile.Name]; exists {
			requiredProfiles[profile.Name] = true
		}

		// Validate category weights sum to ~1.0
		categoryTotal := profile.OffensiveWeight + profile.DefensiveWeight + profile.UtilityWeight
		if categoryTotal < 0.99 || categoryTotal > 1.01 {
			panic("Profile '" + profile.Name + "' category weights must sum to 1.0")
		}

		// Validate offensive sub-weights sum to ~1.0
		offensiveTotal := profile.DamageWeight + profile.AccuracyWeight
		if offensiveTotal < 0.99 || offensiveTotal > 1.01 {
			panic("Profile '" + profile.Name + "' offensive sub-weights must sum to 1.0")
		}

		// Validate defensive sub-weights sum to ~1.0
		defensiveTotal := profile.HealthWeight + profile.ResistanceWeight + profile.AvoidanceWeight
		if defensiveTotal < 0.99 || defensiveTotal > 1.01 {
			panic("Profile '" + profile.Name + "' defensive sub-weights must sum to 1.0")
		}

		// Validate utility sub-weights sum to ~1.0
		utilityTotal := profile.RoleWeight + profile.AbilityWeight + profile.CoverWeight
		if utilityTotal < 0.99 || utilityTotal > 1.01 {
			panic("Profile '" + profile.Name + "' utility sub-weights must sum to 1.0")
		}

		// Validate all weights are non-negative
		if profile.OffensiveWeight < 0 || profile.DefensiveWeight < 0 || profile.UtilityWeight < 0 ||
			profile.DamageWeight < 0 || profile.AccuracyWeight < 0 ||
			profile.HealthWeight < 0 || profile.ResistanceWeight < 0 || profile.AvoidanceWeight < 0 ||
			profile.RoleWeight < 0 || profile.AbilityWeight < 0 || profile.CoverWeight < 0 {
			panic("Profile '" + profile.Name + "' weights must be non-negative")
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

	// Validate scaling constants are positive
	sc := config.ScalingConstants
	if sc.RoleScaling <= 0 || sc.DodgeScaling <= 0 || sc.CoverScaling <= 0 ||
		sc.CoverBeneficiaryMultiplier <= 0 || sc.LeaderBonus <= 0 {
		panic("All scaling constants must be positive")
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
	println("Overworld config loaded:", len(OverworldConfigTemplate.ThreatTypes), "threat types")
}

func validateOverworldConfig(config *JSONOverworldConfig) {
	// Validate threat growth parameters are positive
	tg := config.ThreatGrowth
	if tg.DefaultGrowthRate <= 0 || tg.ContainmentSlowdown <= 0 || tg.MaxThreatIntensity <= 0 ||
		tg.ChildNodeSpawnThreshold <= 0 || tg.PlayerContainmentRadius <= 0 ||
		tg.MaxChildNodeSpawnAttempts <= 0 {
		panic("All threat growth parameters must be positive")
	}

	// Validate faction AI parameters are positive
	fa := config.FactionAI
	if fa.DefaultIntentTickDuration <= 0 || fa.ExpansionStrengthThreshold <= 0 ||
		fa.ExpansionTerritoryLimit <= 0 || fa.FortificationWeakThreshold <= 0 ||
		fa.FortificationStrengthGain <= 0 || fa.RaidStrengthThreshold <= 0 ||
		fa.RaidProximityRange <= 0 || fa.RetreatCriticalStrength <= 0 ||
		fa.MaxTerritorySize <= 0 {
		panic("All faction AI parameters must be positive")
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

	// Validate required threat types exist
	requiredThreats := map[string]bool{
		"Necromancer": false,
		"BanditCamp":  false,
		"Corruption":  false,
		"BeastNest":   false,
		"OrcWarband":  false,
	}
	for _, tt := range config.ThreatTypes {
		if _, exists := requiredThreats[tt.ThreatType]; exists {
			requiredThreats[tt.ThreatType] = true
		}

		// Validate threat type parameters are positive
		if tt.BaseGrowthRate <= 0 || tt.BaseRadius <= 0 || tt.MaxIntensity <= 0 {
			panic("Threat type '" + tt.ThreatType + "' parameters must be positive")
		}

		// Validate primary effect is valid
		validEffects := map[string]bool{
			"SpawnBoost":         true,
			"ResourceDrain":      true,
			"TerrainCorruption":  true,
			"CombatDebuff":       true,
		}
		if !validEffects[tt.PrimaryEffect] {
			panic("Threat type '" + tt.ThreatType + "' has invalid primary effect: " + tt.PrimaryEffect)
		}
	}
	for threat, found := range requiredThreats {
		if !found {
			panic("Overworld config missing required threat type: " + threat)
		}
	}

	// Validate faction scoring parameters (allow negative values for penalties/modifiers)
	// Just ensure critical values exist (no strict validation needed since negative values are intentional)
}
