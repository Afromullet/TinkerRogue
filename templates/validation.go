package templates

// checkRequired panics if any entry in the required map is still false.
func checkRequired(label string, required map[string]bool) {
	for key, found := range required {
		if !found {
			panic("Missing required " + label + ": " + key)
		}
	}
}

// markFound marks a key as found in a required map, if it exists.
func markFound(required map[string]bool, key string) {
	if _, exists := required[key]; exists {
		required[key] = true
	}
}

func validateNodeDefinitions(data *NodeDefinitionsData) {
	seenIDs := make(map[string]bool)

	// Build valid categories map
	validCategories := make(map[string]bool)
	for _, cat := range data.NodeCategories {
		validCategories[cat] = true
	}

	requiredNodes := map[string]bool{
		"necromancer": false,
		"banditcamp":  false,
		"corruption":  false,
		"beastnest":   false,
		"orcwarband":  false,
	}

	for _, node := range data.Nodes {
		nodeID := string(node.ID)
		factionID := string(node.FactionID)
		if node.ID == "" {
			panic("Node definition missing required 'id' field")
		}
		if node.DisplayName == "" {
			panic("Node definition '" + nodeID + "' missing required 'displayName' field")
		}
		if node.Category == "" {
			panic("Node definition '" + nodeID + "' missing required 'category' field")
		}

		if seenIDs[nodeID] {
			panic("Duplicate node definition ID: " + nodeID)
		}
		seenIDs[nodeID] = true

		if !validCategories[node.Category] {
			panic("Node '" + nodeID + "' has invalid category: " + node.Category)
		}

		if node.Category == "threat" && factionID == "" {
			panic("Threat node '" + nodeID + "' missing required 'factionId' field")
		}

		if node.Color.A == 0 {
			println("Warning: Node '" + nodeID + "' has zero alpha (invisible)")
		}

		markFound(requiredNodes, nodeID)
	}

	checkRequired("node definition", requiredNodes)

	if data.DefaultNode == nil {
		panic("Missing defaultNode in nodeDefinitions.json")
	}
}

// validateEncounterDefinitions validates the new encounter definitions format.
// Multiple encounters per faction are explicitly supported (e.g., basic/elite/boss variants).
func validateEncounterDefinitions(data *EncounterDataWithNew, validSquadTypes map[string]bool) {
	seenIDs := make(map[string]bool)
	seenEncounterTypeIDs := make(map[string]bool)

	requiredEncounters := map[string]bool{
		"necromancer": false,
		"banditcamp":  false,
		"corruption":  false,
		"beastnest":   false,
		"orcwarband":  false,
	}

	encountersPerFaction := make(map[string][]string)

	for _, encounter := range data.EncounterDefinitions {
		encID := string(encounter.ID)
		encTypeID := string(encounter.EncounterTypeID)
		factionID := string(encounter.FactionID)
		if encounter.ID == "" {
			panic("Encounter definition missing required 'id' field")
		}
		if encounter.EncounterTypeID == "" {
			panic("Encounter definition '" + encID + "' missing required 'encounterTypeId' field")
		}

		if seenIDs[encID] {
			panic("Duplicate encounter definition ID: " + encID)
		}
		seenIDs[encID] = true

		if seenEncounterTypeIDs[encTypeID] {
			panic("Duplicate encounterTypeId: " + encTypeID)
		}
		seenEncounterTypeIDs[encTypeID] = true

		for _, pref := range encounter.SquadPreferences {
			if !validSquadTypes[pref] {
				panic("Encounter '" + encID + "' references invalid squad type: " + pref)
			}
		}

		if factionID != "" {
			if _, exists := data.Factions[factionID]; !exists {
				panic("Encounter '" + encID + "' references unknown faction: " + factionID)
			}
			encountersPerFaction[factionID] = append(encountersPerFaction[factionID], encID)
		}

		markFound(requiredEncounters, encID)
	}

	checkRequired("encounter definition", requiredEncounters)

	for factionID, encounterIDs := range encountersPerFaction {
		if len(encounterIDs) > 1 {
			println("Faction '"+factionID+"' has multiple encounters:", len(encounterIDs))
		}
	}
}

// validateNodeEncounterLinks cross-validates that threat nodes' factions have encounters.
func validateNodeEncounterLinks() {
	encountersPerFaction := make(map[string]int)
	for _, enc := range EncounterDefinitionTemplates {
		if enc.FactionID != "" {
			encountersPerFaction[string(enc.FactionID)]++
		}
	}

	for _, node := range NodeDefinitionTemplates {
		if node.Category == "threat" && node.FactionID != "" {
			if encountersPerFaction[string(node.FactionID)] == 0 {
				panic("Threat node '" + string(node.ID) + "' has factionId '" + string(node.FactionID) + "' but no encounters exist for that faction")
			}
		}
	}

	println("Node-encounter links validated successfully")
}

func validateNameConfig(config *JSONNameConfig) {
	if _, exists := config.Pools["default"]; !exists {
		panic("Name config missing required 'default' pool")
	}

	if config.MinSyllables < 2 {
		panic("Name config minSyllables must be at least 2")
	}
	if config.MaxSyllables < config.MinSyllables {
		panic("Name config maxSyllables must be >= minSyllables")
	}

	for name, pool := range config.Pools {
		if len(pool.Prefixes) == 0 {
			panic("Name pool '" + name + "' must have at least one prefix")
		}
		if len(pool.Suffixes) == 0 {
			panic("Name pool '" + name + "' must have at least one suffix")
		}
	}
}

func validateAIConfig(config *JSONAIConfig) {
	requiredRoles := map[string]bool{"Tank": false, "DPS": false, "Support": false}
	for _, rb := range config.RoleBehaviors {
		markFound(requiredRoles, rb.Role)
	}
	checkRequired("AI role behavior", requiredRoles)

	for _, rb := range config.RoleBehaviors {
		if rb.MeleeWeight < -1.0 || rb.MeleeWeight > 1.0 ||
			rb.SupportWeight < -1.0 || rb.SupportWeight > 1.0 {
			panic("Role behavior weights must be between -1.0 and 1.0 for role: " + rb.Role)
		}
	}

	tc := config.ThreatCalculation
	if tc.FlankingThreatRangeBonus <= 0 || tc.IsolationThreshold <= 0 ||
		tc.RetreatSafeThreatThreshold <= 0 {
		panic("All threat calculation distances must be positive")
	}

	sl := config.SupportLayer
	if sl.HealRadius <= 0 {
		panic("All support layer parameters must be positive")
	}
}

func validatePowerConfig(config *JSONPowerConfig) {
	requiredProfiles := map[string]bool{"Balanced": false}
	for _, profile := range config.Profiles {
		markFound(requiredProfiles, profile.Name)

		categoryTotal := profile.OffensiveWeight + profile.DefensiveWeight + profile.UtilityWeight
		if categoryTotal < 0.99 || categoryTotal > 1.01 {
			panic("Profile '" + profile.Name + "' category weights must sum to 1.0")
		}

		if profile.OffensiveWeight < 0 || profile.DefensiveWeight < 0 || profile.UtilityWeight < 0 {
			panic("Profile '" + profile.Name + "' weights must be non-negative")
		}

		if profile.HealthPenalty <= 0 {
			panic("Profile '" + profile.Name + "' health penalty must be positive")
		}
	}
	checkRequired("power profile", requiredProfiles)

	requiredRoles := map[string]bool{"Tank": false, "DPS": false, "Support": false}
	for _, rm := range config.RoleMultipliers {
		markFound(requiredRoles, rm.Role)
		if rm.Multiplier <= 0 {
			panic("Role multiplier must be positive for role: " + rm.Role)
		}
	}
	checkRequired("power role multiplier", requiredRoles)

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

	if config.LeaderBonus <= 0 {
		panic("Leader bonus must be positive")
	}
}

func validateInfluenceConfig(config *JSONInfluenceConfig) {
	if config.BaseMagnitudeMultiplier <= 0 {
		panic("Influence baseMagnitudeMultiplier must be positive")
	}
	if config.DefaultPlayerNodeMagnitude < 0 {
		panic("Influence defaultPlayerNodeMagnitude must be non-negative")
	}
	if config.DefaultPlayerNodeRadius <= 0 {
		panic("Influence defaultPlayerNodeRadius must be positive")
	}

	if config.Synergy.GrowthBonus < 0 {
		panic("Influence synergy growthBonus must be non-negative")
	}

	if config.Competition.GrowthPenalty < 0 {
		panic("Influence competition growthPenalty must be non-negative")
	}

	if config.Suppression.GrowthPenalty < 0 {
		panic("Influence suppression growthPenalty must be non-negative")
	}
}

func validateMapGenConfig(config *JSONMapGenConfig) {
	g := config.Generators

	validateRoomsCorridorsConfig(g.RoomsCorridors)
	validateCavernConfig(g.Cavern)
	validateOverworldGenConfig(g.Overworld)
	validateGarrisonRaidConfig(g.GarrisonRaid)
}

func validateRoomsCorridorsConfig(rc *JSONRoomsCorridorsConfig) {
	if rc == nil {
		return
	}
	if rc.MinRoomSize <= 0 || rc.MaxRoomSize <= 0 || rc.MaxRooms <= 0 {
		panic("mapgenconfig: rooms_corridors values must be positive")
	}
	if rc.MinRoomSize > rc.MaxRoomSize {
		panic("mapgenconfig: rooms_corridors minRoomSize must be <= maxRoomSize")
	}
}

func validateCavernConfig(c *JSONCavernConfig) {
	if c == nil {
		return
	}
	if c.MinChamberRadius <= 0 || c.MaxChamberRadius <= 0 {
		panic("mapgenconfig: cavern chamber radius must be positive")
	}
	if c.MinChamberRadius > c.MaxChamberRadius {
		panic("mapgenconfig: cavern minChamberRadius must be <= maxChamberRadius")
	}
	if c.TargetWalkableMin > c.TargetWalkableMax {
		panic("mapgenconfig: cavern targetWalkableMin must be <= targetWalkableMax")
	}
	if c.NumChambers <= 0 {
		panic("mapgenconfig: cavern numChambers must be positive")
	}
	if c.BorderThickness < 0 {
		panic("mapgenconfig: cavern borderThickness must be non-negative")
	}
}

func validateOverworldGenConfig(ow *JSONOverworldGenConfig) {
	if ow == nil {
		return
	}
	if ow.ElevationOctaves <= 0 || ow.MoistureOctaves <= 0 {
		panic("mapgenconfig: overworld octaves must be positive")
	}
	if ow.WaterThresh >= ow.MountainThresh {
		panic("mapgenconfig: overworld waterThresh must be < mountainThresh")
	}
	if ow.FactionCount < 0 || ow.FactionMinSpacing < 0 {
		panic("mapgenconfig: overworld faction values must be non-negative")
	}
}

func validateGarrisonRaidConfig(gr *JSONGarrisonRaidConfig) {
	if gr == nil {
		return
	}
	validRoomTypes := map[string]bool{
		"barracks": true, "guard_post": true, "armory": true,
		"command_post": true, "patrol_route": true, "mage_tower": true,
		"rest_room": true, "stairs": true,
	}

	for key, rs := range gr.RoomSizes {
		if !validRoomTypes[key] {
			panic("mapgenconfig: garrison_raid roomSizes has invalid room type: " + key)
		}
		if rs.MinW > rs.MaxW || rs.MinH > rs.MaxH {
			panic("mapgenconfig: garrison_raid roomSizes min must be <= max for: " + key)
		}
	}

	for _, fs := range gr.FloorScaling {
		if fs.Floor <= 0 {
			panic("mapgenconfig: garrison_raid floorScaling floor must be positive")
		}
		if fs.MinCritPath > fs.MaxCritPath {
			panic("mapgenconfig: garrison_raid floorScaling minCritPath must be <= maxCritPath")
		}
		if fs.MinTotal > fs.MaxTotal {
			panic("mapgenconfig: garrison_raid floorScaling minTotal must be <= maxTotal")
		}
		for _, t := range fs.AllowedTypes {
			if !validRoomTypes[t] {
				panic("mapgenconfig: garrison_raid floorScaling has invalid room type: " + t)
			}
		}
	}

	for key, sc := range gr.SpawnCounts {
		if !validRoomTypes[key] {
			panic("mapgenconfig: garrison_raid spawnCounts has invalid room type: " + key)
		}
		if sc.MinPlayer > sc.MaxPlayer || sc.MinDefender > sc.MaxDefender {
			panic("mapgenconfig: garrison_raid spawnCounts min must be <= max for: " + key)
		}
	}
}

func validateOverworldConfig(config *JSONOverworldConfig) {
	tg := config.ThreatGrowth
	if tg.ContainmentSlowdown <= 0 || tg.MaxThreatIntensity <= 0 ||
		tg.ChildNodeSpawnThreshold <= 0 {
		panic("All threat growth parameters must be positive")
	}

	fa := config.FactionAI
	if fa.DefaultIntentTickDuration <= 0 ||
		fa.MaxTerritorySize <= 0 {
		panic("All faction AI parameters must be positive")
	}

	st := config.StrengthThresholds
	if st.Weak <= 0 || st.Strong <= 0 || st.Critical < 0 {
		panic("Strength thresholds must be positive (critical can be 0)")
	}
	if st.Critical > st.Weak || st.Weak >= st.Strong {
		panic("Strength thresholds must be: critical <= weak < strong")
	}

	vc := config.VictoryConditions
	if vc.HighIntensityThreshold <= 0 || vc.MaxHighIntensityThreats <= 0 || vc.MaxThreatInfluence <= 0 {
		panic("Victory condition thresholds must be positive")
	}

	fsc := config.FactionScoringControl
	if fsc.IdleScoreThreshold <= 0 || fsc.RaidBaseIntensity <= 0 || fsc.RaidIntensityScale <= 0 {
		panic("Faction scoring control parameters must be positive")
	}

	sp := config.SpawnProbabilities
	if sp.ExpansionThreatSpawnChance < 0 || sp.ExpansionThreatSpawnChance > 100 ||
		sp.FortifyThreatSpawnChance < 0 || sp.FortifyThreatSpawnChance > 100 {
		panic("Spawn probabilities must be between 0 and 100")
	}

	md := config.MapDimensions
	if md.DefaultMapWidth <= 0 || md.DefaultMapHeight <= 0 {
		panic("Map dimensions must be positive")
	}

	requiredStrategies := []string{"Expansionist", "Aggressor", "Raider", "Defensive", "Territorial"}
	for _, strategy := range requiredStrategies {
		if _, ok := config.StrategyBonuses[strategy]; !ok {
			panic("Missing required strategy bonus: " + strategy)
		}
	}
}
