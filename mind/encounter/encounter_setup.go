package encounter

import (
	"fmt"
	"game_main/common"
	"game_main/mind/combatlifecycle"
	"game_main/mind/evaluation"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/templates"
	"game_main/world/coords"

	"math"

	"github.com/bytearena/ecs"
)

// Note: EnemySquadSpec is defined in types.go and used for all enemy squad generation

// getGarrisonForEncounter returns garrison data if this encounter targets a garrisoned node.
// Returns nil if there's no garrison or no threat node.
func getGarrisonForEncounter(manager *common.EntityManager, encounterData *core.OverworldEncounterData) *core.GarrisonData {
	if encounterData.ThreatNodeID == 0 {
		return nil
	}
	garrisonData := garrison.GetGarrisonAtNode(manager, encounterData.ThreatNodeID)
	if garrisonData == nil || len(garrisonData.SquadIDs) == 0 {
		return nil
	}
	return garrisonData
}

// SpawnCombatEntities creates player and enemy factions with power-based squad generation.
// Returns a list of enemy squad IDs created for this encounter.
//
// This function delegates to GenerateEncounterSpec for the core generation logic,
// then sets up combat factions and action states. For garrison encounters, the caller
// should check getGarrisonForEncounter and call spawnGarrisonEncounter directly.
func SpawnCombatEntities(
	manager *common.EntityManager,
	rosterOwnerID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
	encounterData *core.OverworldEncounterData,
	encounterID ecs.EntityID,
) (*SpawnResult, error) {
	// Generate encounter from power budget
	// 1. Generate encounter specification (handles validation, power calculation, squad creation)
	spec, err := GenerateEncounterSpec(manager, rosterOwnerID, playerStartPos, encounterData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate encounter spec: %w", err)
	}

	// 2. Create factions with encounter tracking
	fm, playerFactionID, enemyFactionID := combatlifecycle.CreateFactionPair(manager, "Player Forces", "Enemy Forces", encounterID)

	// 3. Add player's deployed squads to faction
	if err := assignPlayerSquadsToFaction(fm, rosterOwnerID, manager, playerFactionID, playerStartPos); err != nil {
		return nil, fmt.Errorf("failed to assign player squads: %w", err)
	}

	// 4. Add enemy squads to faction
	enemySquadIDs := make([]ecs.EntityID, len(spec.EnemySquads))
	enemyPositions := make([]coords.LogicalPosition, len(spec.EnemySquads))
	for i, es := range spec.EnemySquads {
		enemySquadIDs[i] = es.SquadID
		enemyPositions[i] = es.Position
	}
	if err := combatlifecycle.EnrollSquadsAtPositions(fm, manager, enemyFactionID, enemySquadIDs, enemyPositions, false); err != nil {
		return nil, fmt.Errorf("failed to add enemy squads: %w", err)
	}

	return &SpawnResult{
		EnemySquadIDs:   enemySquadIDs,
		PlayerFactionID: playerFactionID,
		EnemyFactionID:  enemyFactionID,
	}, nil
}

// spawnGarrisonEncounter uses existing garrison squads as enemies instead of generating new ones.
// This is used when the player attacks a node that has an NPC garrison.
func spawnGarrisonEncounter(
	manager *common.EntityManager,
	rosterOwnerID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
	garrisonData *core.GarrisonData,
	encounterID ecs.EntityID,
) (*SpawnResult, error) {
	// Create factions
	fm, playerFactionID, enemyFactionID := combatlifecycle.CreateFactionPair(manager, "Player Forces", "Garrison Forces", encounterID)

	// Add player squads
	if err := assignPlayerSquadsToFaction(fm, rosterOwnerID, manager, playerFactionID, playerStartPos); err != nil {
		return nil, fmt.Errorf("failed to assign player squads: %w", err)
	}

	// Add garrison squads as enemies
	enemyPositions := generatePositionsAroundPoint(playerStartPos, len(garrisonData.SquadIDs), 0, 2*math.Pi, EnemySpacingDistance, EnemySpacingDistance)

	if err := combatlifecycle.EnrollSquadsAtPositions(fm, manager, enemyFactionID, garrisonData.SquadIDs, enemyPositions, false); err != nil {
		return nil, fmt.Errorf("failed to add garrison squads: %w", err)
	}

	return &SpawnResult{
		EnemySquadIDs:   garrisonData.SquadIDs,
		PlayerFactionID: playerFactionID,
		EnemyFactionID:  enemyFactionID,
	}, nil
}

// ensurePlayerSquadsDeployed checks if player has deployed squads, and auto-deploys all if none are deployed
func ensurePlayerSquadsDeployed(rosterOwnerID ecs.EntityID, manager *common.EntityManager) error {
	roster := squads.GetPlayerSquadRoster(rosterOwnerID, manager)
	if roster == nil {
		return fmt.Errorf("player has no squad roster")
	}

	deployedSquads := roster.GetDeployedSquads(manager)
	if len(deployedSquads) == 0 {
		// Auto-deploy all squads if none are deployed
		for _, squadID := range roster.OwnedSquads {
			squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
			if squadData != nil {
				squadData.IsDeployed = true
			}
		}
	}

	return nil
}

// assignPlayerSquadsToFaction adds all deployed player squads to the player faction
// Assumes squads are already deployed (handled by ensurePlayerSquadsDeployed)
func assignPlayerSquadsToFaction(
	fm *combat.CombatFactionManager,
	rosterOwnerID ecs.EntityID,
	manager *common.EntityManager,
	factionID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
) error {
	// Get player's squad roster
	roster := squads.GetPlayerSquadRoster(rosterOwnerID, manager)
	if roster == nil {
		return fmt.Errorf("roster owner has no squad roster")
	}

	// Get deployed squads (should already be deployed)
	deployedSquads := roster.GetDeployedSquads(manager)
	if len(deployedSquads) == 0 {
		return fmt.Errorf("player has no squads - this should not happen after ensurePlayerSquadsDeployed()")
	}

	// Position player squads around starting position
	squadPositions := generatePlayerSquadPositions(playerStartPos, len(deployedSquads))

	return combatlifecycle.EnrollSquadsAtPositions(fm, manager, factionID, deployedSquads, squadPositions, false)
}

// generatePositionsAroundPoint creates positions distributed around a center point.
// arcStart/arcEnd define the angle range in radians (0 to 2*Pi for full circle).
// minDistance/maxDistance define the radius range from center.
func generatePositionsAroundPoint(
	center coords.LogicalPosition,
	count int,
	arcStart, arcEnd float64,
	minDistance, maxDistance int,
) []coords.LogicalPosition {
	positions := make([]coords.LogicalPosition, count)
	arcRange := arcEnd - arcStart
	mapWidth := coords.CoordManager.GetDungeonWidth()
	mapHeight := coords.CoordManager.GetDungeonHeight()

	for i := 0; i < count; i++ {
		angle := arcStart + (float64(i)/float64(count))*arcRange
		// Alternate between min and max distance for variety
		distance := minDistance + (i % (maxDistance - minDistance + 1))

		offsetX := int(math.Round(float64(distance) * math.Cos(angle)))
		offsetY := int(math.Round(float64(distance) * math.Sin(angle)))

		pos := coords.LogicalPosition{
			X: clampPosition(center.X+offsetX, 0, mapWidth-1),
			Y: clampPosition(center.Y+offsetY, 0, mapHeight-1),
		}
		positions[i] = pos
	}
	return positions
}

// generatePlayerSquadPositions creates positions for player squads around starting point
func generatePlayerSquadPositions(startPos coords.LogicalPosition, count int) []coords.LogicalPosition {
	// Player squads: arc from -Pi/2 to Pi/2 (facing forward), distance alternating 3-4
	return generatePositionsAroundPoint(startPos, count, -math.Pi/2, math.Pi/2, PlayerMinDistance, PlayerMaxDistance)
}

// generateEnemySquadsByPower creates enemy squads matching target squad power.
// targetPower is now the per-squad target (average player squad power * difficulty).
// Returns EnemySquadSpec which includes Type and Name for full encounter specification.
func generateEnemySquadsByPower(
	manager *common.EntityManager,
	targetSquadPower float64,
	difficultyMod templates.JSONEncounterDifficulty,
	encounterData *core.OverworldEncounterData,
	playerPos coords.LogicalPosition,
	config *evaluation.PowerConfig,
) []EnemySquadSpec {
	// Use fixed squad count from difficulty modifier
	squadCount := difficultyMod.SquadCount

	enemySquads := []EnemySquadSpec{}

	// Get squad composition preferences
	squadTypes := getSquadComposition(encounterData, squadCount)

	// Pre-compute all enemy positions once (avoids N² allocation)
	enemyPositions := generatePositionsAroundPoint(playerPos, squadCount, 0, 2*math.Pi, EnemySpacingDistance, EnemySpacingDistance)

	for i := 0; i < squadCount; i++ {
		pos := enemyPositions[i]
		squadName := fmt.Sprintf("Enemy Squad %d", i+1)

		// Each enemy squad targets the same power (average player squad power * difficulty)
		squadID := createSquadForPowerBudget(
			manager,
			targetSquadPower,
			squadTypes[i],
			squadName,
			pos,
			config,
			difficultyMod,
		)

		if squadID != 0 {
			enemySquads = append(enemySquads, EnemySquadSpec{
				SquadID:  squadID,
				Position: pos,
				Power:    targetSquadPower,
				Type:     squadTypes[i],
				Name:     squadName,
			})
		}
	}

	return enemySquads
}

// getSquadComposition returns squad type distribution based on encounter type
func getSquadComposition(encounterData *core.OverworldEncounterData, count int) []string {
	if encounterData == nil || encounterData.EncounterType == "" {
		// Random balanced composition
		return generateRandomComposition(count)
	}

	// Use encounter preferences from JSON configuration
	preferences := GetSquadPreferences(encounterData.EncounterType)
	if len(preferences) == 0 {
		return generateRandomComposition(count)
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = preferences[i%len(preferences)]
	}

	return result
}

// generateRandomComposition creates a random mix of squad types
func generateRandomComposition(count int) []string {
	types := []string{SquadTypeMelee, SquadTypeRanged, SquadTypeMagic}
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = types[common.RandomInt(len(types))]
	}
	return result
}

// createSquadForPowerBudget creates a squad matching target power.
// Uses the shared evaluation package for power estimation.
func createSquadForPowerBudget(
	manager *common.EntityManager,
	targetPower float64,
	squadType string,
	name string,
	position coords.LogicalPosition,
	config *evaluation.PowerConfig,
	difficultyMod templates.JSONEncounterDifficulty,
) ecs.EntityID {
	// Select unit pool based on squad type
	unitPool := filterUnitsBySquadType(squadType)
	if len(unitPool) == 0 {
		unitPool = squads.Units // Fallback to all units
	}

	if len(unitPool) == 0 {
		return 0 // No units available
	}

	// Iteratively add units until power budget reached
	unitsToCreate := []squads.UnitTemplate{}
	currentPower := 0.0
	// Use safe grid positions that work for 2-wide units (avoid rightmost column)
	// Extended pattern to support up to 8 units for Boss difficulty
	gridPositions := [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 0}, {2, 1}, {3, 0}, {3, 1}}

	maxUnits := difficultyMod.MaxUnitsPerSquad
	if maxUnits > len(gridPositions) {
		maxUnits = len(gridPositions)
	}

	for currentPower < targetPower && len(unitsToCreate) < maxUnits {
		// Pick random unit from pool
		unit := unitPool[common.RandomInt(len(unitPool))]

		// Estimate unit power contribution using shared function
		unitPower := evaluation.EstimateUnitPowerFromTemplate(unit, config)

		// Set grid position
		unit.GridRow = gridPositions[len(unitsToCreate)][0]
		unit.GridCol = gridPositions[len(unitsToCreate)][1]
		unit.IsLeader = (len(unitsToCreate) == 0) // First unit is leader

		unitsToCreate = append(unitsToCreate, unit)
		currentPower += unitPower

		// Stop if we've reached the power threshold
		if currentPower >= targetPower*PowerThreshold {
			break
		}
	}

	// Ensure minimum units based on difficulty
	minUnits := difficultyMod.MinUnitsPerSquad
	if minUnits > len(gridPositions) {
		minUnits = len(gridPositions)
	}
	for len(unitsToCreate) < minUnits && len(unitPool) > 0 {
		unit := unitPool[common.RandomInt(len(unitPool))]
		unit.GridRow = gridPositions[len(unitsToCreate)][0]
		unit.GridCol = gridPositions[len(unitsToCreate)][1]
		unitsToCreate = append(unitsToCreate, unit)
	}

	// Set leader attributes
	if len(unitsToCreate) > 0 {
		unitsToCreate[0].Attributes.Leadership = LeadershipAttributeBase
	}

	// Create squad
	squadID := squads.CreateSquadFromTemplate(
		manager,
		name,
		squads.FormationBalanced,
		position,
		unitsToCreate,
	)

	return squadID
}

// filterUnitsBySquadType selects units matching squad archetype
func filterUnitsBySquadType(squadType string) []squads.UnitTemplate {
	switch squadType {
	case SquadTypeMelee:
		return squads.FilterByMaxAttackRange(2) // Melee: range <= 2
	case SquadTypeRanged:
		return squads.FilterByAttackRange(3) // Ranged: range >= 3
	case SquadTypeMagic:
		return squads.FilterByAttackType(squads.AttackTypeMagic)
	case SquadTypeSupport:
		return squads.FilterByAttackType(squads.AttackTypeHeal)
	default:
		return squads.Units
	}
}

// clampPosition ensures a position stays within bounds
func clampPosition(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
