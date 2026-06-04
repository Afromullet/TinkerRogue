package templates

import (
	"fmt"
	"log"

	"game_main/world/garrisongen/roomtypes"
)

// validation.go holds all per-subsystem validator functions. Each validator
// returns error (never panics — Loader[T] wraps the error with a "validate
// <name>" prefix so failures surface at the single boot-time log.Fatalf in
// setup/gamesetup.LoadGameData). Shared helpers (markFound, checkRequired,
// makeRequiredMap, requiredRoles) live in validate_shared.go.

// requiredNodeIDsFrom derives the "must exist" set from the JSON itself: any
// node entry with "required": true. The set is the single source of truth used
// by both validateNodeDefinitions (each required node was loaded) and
// validateEncounterDefinitions (every required node has an encounter with the
// same ID). Adding a new mandatory threat = one-line JSON edit.
func requiredNodeIDsFrom(nodes []JSONNodeDefinition) []string {
	var ids []string
	for _, n := range nodes {
		if n.Required {
			ids = append(ids, string(n.ID))
		}
	}
	return ids
}

// --- Node and encounter validators ---

func validateNodeDefinitions(data *NodeDefinitionsData) error {
	seenIDs := make(map[string]bool)

	// Build valid categories map
	validCategories := make(map[string]bool)
	for _, cat := range data.NodeCategories {
		validCategories[cat] = true
	}

	for _, node := range data.Nodes {
		nodeID := string(node.ID)
		factionID := string(node.FactionID)
		if node.ID == "" {
			return fmt.Errorf("node definition missing required 'id' field")
		}
		if node.DisplayName == "" {
			return fmt.Errorf("node %q missing required 'displayName' field", nodeID)
		}
		if node.Category == "" {
			return fmt.Errorf("node %q missing required 'category' field", nodeID)
		}

		if seenIDs[nodeID] {
			return fmt.Errorf("duplicate node definition ID: %q", nodeID)
		}
		seenIDs[nodeID] = true

		if !validCategories[node.Category] {
			return fmt.Errorf("node %q has invalid category: %q", nodeID, node.Category)
		}

		if node.Category == "threat" && factionID == "" {
			return fmt.Errorf("threat node %q missing required 'factionId' field", nodeID)
		}

		if node.Color.A == 0 {
			log.Printf("[templates] node %q has zero alpha (invisible)", nodeID)
		}
	}

	// Every "required: true" node must have been seen above.
	for _, id := range requiredNodeIDsFrom(data.Nodes) {
		if !seenIDs[id] {
			return fmt.Errorf("missing required node definition: %s", id)
		}
	}

	if data.DefaultNode == nil {
		return fmt.Errorf("missing defaultNode in nodeDefinitions.json")
	}
	return nil
}

// validateEncounterDefinitions validates the new encounter definitions format.
// Multiple encounters per faction are explicitly supported (e.g., basic/elite/boss variants).
// The "required encounter IDs" set is derived from NodeDefinitionTemplates' Required flag,
// so any node marked required in nodeDefinitions.json must have a same-named encounter.
func validateEncounterDefinitions(data *EncounterDataWithNew, validSquadTypes map[string]bool) error {
	seenIDs := make(map[string]bool)
	seenEncounterTypeIDs := make(map[string]bool)

	encountersPerFaction := make(map[string][]string)

	for _, encounter := range data.EncounterDefinitions {
		encID := string(encounter.ID)
		encTypeID := string(encounter.EncounterTypeID)
		factionID := string(encounter.FactionID)
		if encounter.ID == "" {
			return fmt.Errorf("encounter definition missing required 'id' field")
		}
		if encounter.EncounterTypeID == "" {
			return fmt.Errorf("encounter %q missing required 'encounterTypeId' field", encID)
		}

		if seenIDs[encID] {
			return fmt.Errorf("duplicate encounter definition ID: %q", encID)
		}
		seenIDs[encID] = true

		if seenEncounterTypeIDs[encTypeID] {
			return fmt.Errorf("duplicate encounterTypeId: %q", encTypeID)
		}
		seenEncounterTypeIDs[encTypeID] = true

		for _, pref := range encounter.SquadPreferences {
			if !validSquadTypes[pref] {
				return fmt.Errorf("encounter %q references invalid squad type: %q", encID, pref)
			}
		}

		if factionID != "" {
			if _, exists := data.Factions[factionID]; !exists {
				return fmt.Errorf("encounter %q references unknown faction: %q", encID, factionID)
			}
			encountersPerFaction[factionID] = append(encountersPerFaction[factionID], encID)
		}
	}

	// Each "required" node must have a same-named encounter.
	for _, id := range requiredNodeIDsFrom(NodeDefinitionTemplates) {
		if !seenIDs[id] {
			return fmt.Errorf("missing required encounter definition: %s", id)
		}
	}

	for factionID, encounterIDs := range encountersPerFaction {
		if len(encounterIDs) > 1 {
			log.Printf("[templates] faction %q has %d encounters", factionID, len(encounterIDs))
		}
	}
	return nil
}

// validateNodeEncounterLinks cross-validates that threat nodes' factions have encounters.
func validateNodeEncounterLinks() error {
	encountersPerFaction := make(map[string]int)
	for _, enc := range EncounterDefinitionTemplates {
		if enc.FactionID != "" {
			encountersPerFaction[string(enc.FactionID)]++
		}
	}

	for _, node := range NodeDefinitionTemplates {
		if node.Category == "threat" && node.FactionID != "" {
			if encountersPerFaction[string(node.FactionID)] == 0 {
				return fmt.Errorf("threat node %q has factionId %q but no encounters exist for that faction",
					string(node.ID), string(node.FactionID))
			}
		}
	}

	log.Printf("[templates] node-encounter links validated")
	return nil
}

// --- Name validator ---

func validateNameConfig(config *JSONNameConfig) error {
	if _, exists := config.Pools["default"]; !exists {
		return fmt.Errorf("missing required 'default' pool")
	}

	if config.MinSyllables < 2 {
		return fmt.Errorf("minSyllables must be at least 2, got %d", config.MinSyllables)
	}
	if config.MaxSyllables < config.MinSyllables {
		return fmt.Errorf("maxSyllables (%d) must be >= minSyllables (%d)",
			config.MaxSyllables, config.MinSyllables)
	}

	for name, pool := range config.Pools {
		if len(pool.Prefixes) == 0 {
			return fmt.Errorf("pool %q must have at least one prefix", name)
		}
		if len(pool.Suffixes) == 0 {
			return fmt.Errorf("pool %q must have at least one suffix", name)
		}
	}
	return nil
}

// --- AI validator ---

func validateAIConfig(config *JSONAIConfig) error {
	required := makeRequiredMap(requiredRoles)
	for _, rb := range config.RoleBehaviors {
		markFound(required, rb.Role)
	}
	if err := checkRequired("AI role behavior", required); err != nil {
		return err
	}

	for _, rb := range config.RoleBehaviors {
		if rb.MeleeWeight < -1.0 || rb.MeleeWeight > 1.0 ||
			rb.SupportWeight < -1.0 || rb.SupportWeight > 1.0 {
			return fmt.Errorf("role %q behavior weights must be between -1.0 and 1.0", rb.Role)
		}
	}

	tc := config.ThreatCalculation
	if tc.FlankingThreatRangeBonus <= 0 || tc.IsolationThreshold <= 0 ||
		tc.RetreatSafeThreatThreshold <= 0 || tc.IsolationMaxDistance <= 0 ||
		tc.EngagementPressureMax <= 0 {
		return fmt.Errorf("all threatCalculation distances must be positive")
	}

	sl := config.SupportLayer
	if sl.HealRadius <= 0 {
		return fmt.Errorf("supportLayer.healRadius must be positive")
	}

	prw := config.PositionalRiskWeights
	if prw.Flanking < 0 || prw.Isolation < 0 || prw.EngagementPressure < 0 || prw.Retreat < 0 {
		return fmt.Errorf("positionalRiskWeights must not be negative")
	}
	return nil
}

// --- Power validator ---

func validatePowerConfig(config *JSONPowerConfig) error {
	requiredProfiles := map[string]bool{"Balanced": false}
	for _, profile := range config.Profiles {
		markFound(requiredProfiles, profile.Name)

		categoryTotal := profile.OffensiveWeight + profile.DefensiveWeight + profile.UtilityWeight
		if categoryTotal < 0.99 || categoryTotal > 1.01 {
			return fmt.Errorf("profile %q category weights must sum to 1.0 (got %.3f)",
				profile.Name, categoryTotal)
		}

		if profile.OffensiveWeight < 0 || profile.DefensiveWeight < 0 || profile.UtilityWeight < 0 {
			return fmt.Errorf("profile %q weights must be non-negative", profile.Name)
		}

		if profile.HealthPenalty <= 0 {
			return fmt.Errorf("profile %q health penalty must be positive", profile.Name)
		}
	}
	if err := checkRequired("power profile", requiredProfiles); err != nil {
		return err
	}

	requiredRoleMap := makeRequiredMap(requiredRoles)
	for _, rm := range config.RoleMultipliers {
		markFound(requiredRoleMap, rm.Role)
		if rm.Multiplier <= 0 {
			return fmt.Errorf("role %q multiplier must be positive", rm.Role)
		}
	}
	if err := checkRequired("power role multiplier", requiredRoleMap); err != nil {
		return err
	}

	requiredTypes := map[int]bool{1: false, 2: false, 3: false, 4: false}
	for _, cb := range config.CompositionBonuses {
		if _, exists := requiredTypes[cb.UniqueTypes]; exists {
			requiredTypes[cb.UniqueTypes] = true
		}
	}
	for types, found := range requiredTypes {
		if !found {
			return fmt.Errorf("missing composition bonus for unique types: %d", types)
		}
	}

	if config.LeaderBonus <= 0 {
		return fmt.Errorf("leaderBonus must be positive")
	}
	return nil
}

// --- Influence validator ---

func validateInfluenceConfig(config *JSONInfluenceConfig) error {
	if config.BaseMagnitudeMultiplier <= 0 {
		return fmt.Errorf("baseMagnitudeMultiplier must be positive")
	}
	if config.DefaultPlayerNodeMagnitude < 0 {
		return fmt.Errorf("defaultPlayerNodeMagnitude must be non-negative")
	}
	if config.DefaultPlayerNodeRadius <= 0 {
		return fmt.Errorf("defaultPlayerNodeRadius must be positive")
	}

	if config.Synergy.GrowthBonus < 0 {
		return fmt.Errorf("synergy.growthBonus must be non-negative")
	}

	if config.Competition.GrowthPenalty < 0 {
		return fmt.Errorf("competition.growthPenalty must be non-negative")
	}

	if config.Suppression.GrowthPenalty < 0 {
		return fmt.Errorf("suppression.growthPenalty must be non-negative")
	}
	return nil
}

// --- MapGen validators ---

func validateMapGenConfig(config *JSONMapGenConfig) error {
	g := config.Generators

	if err := validateRoomsCorridorsConfig(g.RoomsCorridors); err != nil {
		return err
	}
	if err := validateCavernConfig(g.Cavern); err != nil {
		return err
	}
	if err := validateOverworldGenConfig(g.Overworld); err != nil {
		return err
	}
	if err := validateGarrisonRaidConfig(g.GarrisonRaid); err != nil {
		return err
	}
	return nil
}

func validateRoomsCorridorsConfig(rc *JSONRoomsCorridorsConfig) error {
	if rc == nil {
		return nil
	}
	if rc.MinRoomSize <= 0 || rc.MaxRoomSize <= 0 || rc.MaxRooms <= 0 {
		return fmt.Errorf("rooms_corridors values must be positive")
	}
	if rc.MinRoomSize > rc.MaxRoomSize {
		return fmt.Errorf("rooms_corridors minRoomSize must be <= maxRoomSize")
	}
	return nil
}

func validateCavernConfig(c *JSONCavernConfig) error {
	if c == nil {
		return nil
	}
	if c.MinChamberRadius <= 0 || c.MaxChamberRadius <= 0 {
		return fmt.Errorf("cavern chamber radius must be positive")
	}
	if c.MinChamberRadius > c.MaxChamberRadius {
		return fmt.Errorf("cavern minChamberRadius must be <= maxChamberRadius")
	}
	if c.TargetWalkableMin > c.TargetWalkableMax {
		return fmt.Errorf("cavern targetWalkableMin must be <= targetWalkableMax")
	}
	if c.NumChambers <= 0 {
		return fmt.Errorf("cavern numChambers must be positive")
	}
	if c.BorderThickness < 0 {
		return fmt.Errorf("cavern borderThickness must be non-negative")
	}
	return nil
}

func validateOverworldGenConfig(ow *JSONOverworldGenConfig) error {
	if ow == nil {
		return nil
	}
	if ow.ElevationOctaves <= 0 || ow.MoistureOctaves <= 0 {
		return fmt.Errorf("overworld octaves must be positive")
	}
	if ow.WaterThresh >= ow.MountainThresh {
		return fmt.Errorf("overworld waterThresh must be < mountainThresh")
	}
	if ow.FactionCount < 0 || ow.FactionMinSpacing < 0 {
		return fmt.Errorf("overworld faction values must be non-negative")
	}
	return nil
}

func validateGarrisonRaidConfig(gr *JSONGarrisonRaidConfig) error {
	if gr == nil {
		return nil
	}

	for key, rs := range gr.RoomSizes {
		if !roomtypes.Valid[key] {
			return fmt.Errorf("garrison_raid roomSizes invalid room type: %q", key)
		}
		if rs.MinW > rs.MaxW || rs.MinH > rs.MaxH {
			return fmt.Errorf("garrison_raid roomSizes min must be <= max for %q", key)
		}
	}

	for _, fs := range gr.FloorScaling {
		if fs.Floor <= 0 {
			return fmt.Errorf("garrison_raid floorScaling floor must be positive")
		}
		if fs.MinCritPath > fs.MaxCritPath {
			return fmt.Errorf("garrison_raid floorScaling minCritPath must be <= maxCritPath")
		}
		if fs.MinTotal > fs.MaxTotal {
			return fmt.Errorf("garrison_raid floorScaling minTotal must be <= maxTotal")
		}
		for _, t := range fs.AllowedTypes {
			if !roomtypes.Valid[t] {
				return fmt.Errorf("garrison_raid floorScaling invalid room type: %q", t)
			}
		}
	}

	return nil
}

// --- Overworld validator ---

func validateOverworldConfig(config *JSONOverworldConfig) error {
	tg := config.ThreatGrowth
	if tg.ContainmentSlowdown <= 0 || tg.MaxThreatIntensity <= 0 ||
		tg.ChildNodeSpawnThreshold <= 0 {
		return fmt.Errorf("all threatGrowth parameters must be positive")
	}

	fa := config.FactionAI
	if fa.DefaultIntentTickDuration <= 0 ||
		fa.MaxTerritorySize <= 0 {
		return fmt.Errorf("all factionAI parameters must be positive")
	}

	st := config.StrengthThresholds
	if st.Weak <= 0 || st.Strong <= 0 || st.Critical < 0 {
		return fmt.Errorf("strengthThresholds must be positive (critical can be 0)")
	}
	if st.Critical > st.Weak || st.Weak >= st.Strong {
		return fmt.Errorf("strengthThresholds must satisfy: critical <= weak < strong")
	}

	vc := config.VictoryConditions
	if vc.HighIntensityThreshold <= 0 || vc.MaxHighIntensityThreats <= 0 || vc.MaxThreatInfluence <= 0 {
		return fmt.Errorf("victoryConditions thresholds must be positive")
	}

	fsc := config.FactionScoringControl
	if fsc.IdleScoreThreshold <= 0 || fsc.RaidBaseIntensity <= 0 || fsc.RaidIntensityScale <= 0 {
		return fmt.Errorf("factionScoringControl parameters must be positive")
	}

	sp := config.SpawnProbabilities
	if sp.ExpansionThreatSpawnChance < 0 || sp.ExpansionThreatSpawnChance > 100 ||
		sp.FortifyThreatSpawnChance < 0 || sp.FortifyThreatSpawnChance > 100 {
		return fmt.Errorf("spawnProbabilities must be between 0 and 100")
	}

	md := config.MapDimensions
	if md.DefaultMapWidth <= 0 || md.DefaultMapHeight <= 0 {
		return fmt.Errorf("mapDimensions must be positive")
	}

	requiredStrategies := []string{"Expansionist", "Aggressor", "Raider", "Defensive", "Territorial"}
	for _, strategy := range requiredStrategies {
		if _, ok := config.StrategyBonuses[strategy]; !ok {
			return fmt.Errorf("missing required strategy bonus: %q", strategy)
		}
	}
	return nil
}
