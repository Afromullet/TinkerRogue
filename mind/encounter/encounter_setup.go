package encounter

import (
	"fmt"
	"game_main/common"
	"game_main/mind/evaluation"
	"game_main/overworld/core"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"math"

	"github.com/bytearena/ecs"
)

// Note: EnemySquadSpec is defined in types.go and used for all enemy squad generation

// SetupBalancedEncounter creates player and enemy factions with power-based squad generation.
// Replaces SetupGameplayFactions with dynamic encounter balancing.
// Returns a list of enemy squad IDs created for this encounter.
//
// This function delegates to GenerateEncounterSpec for the core generation logic,
// then sets up combat factions and action states.
func SetupBalancedEncounter(
	manager *common.EntityManager,
	playerEntityID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
	encounterData *core.OverworldEncounterData,
	encounterID ecs.EntityID,
) ([]ecs.EntityID, error) {
	// 1. Generate encounter specification (handles validation, power calculation, squad creation)
	spec, err := GenerateEncounterSpec(manager, playerEntityID, playerStartPos, encounterData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate encounter spec: %w", err)
	}

	// 2. Create factions with encounter tracking
	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)
	playerFactionID := fm.CreateFactionWithPlayer("Player Forces", 1, "Player 1", encounterID)
	enemyFactionID := fm.CreateFactionWithPlayer("Enemy Forces", 0, "", encounterID)

	// 3. Add player's deployed squads to faction
	if err := assignPlayerSquadsToFaction(fm, playerEntityID, manager, playerFactionID, playerStartPos); err != nil {
		return nil, fmt.Errorf("failed to assign player squads: %w", err)
	}

	// 4. Add enemy squads to faction and track their IDs
	createdEnemySquadIDs := make([]ecs.EntityID, 0, len(spec.EnemySquads))
	for i, enemySpec := range spec.EnemySquads {
		if err := fm.AddSquadToFaction(enemyFactionID, enemySpec.SquadID, enemySpec.Position); err != nil {
			return nil, fmt.Errorf("failed to add enemy squad %d to faction: %w", i, err)
		}
		if err := createActionStateForSquad(manager, enemySpec.SquadID); err != nil {
			return nil, fmt.Errorf("failed to create action state for enemy squad %d: %w", i, err)
		}
		createdEnemySquadIDs = append(createdEnemySquadIDs, enemySpec.SquadID)
	}

	fmt.Printf("Created encounter: Player Faction (%d) vs Enemy Faction (%d) with %d squads\n",
		playerFactionID, enemyFactionID, len(spec.EnemySquads))

	return createdEnemySquadIDs, nil
}

// ensurePlayerSquadsDeployed checks if player has deployed squads, and auto-deploys all if none are deployed
func ensurePlayerSquadsDeployed(playerID ecs.EntityID, manager *common.EntityManager) error {
	roster := squads.GetPlayerSquadRoster(playerID, manager)
	if roster == nil {
		return fmt.Errorf("player has no squad roster")
	}

	deployedSquads := roster.GetDeployedSquads(manager)
	if len(deployedSquads) == 0 {
		fmt.Println("No deployed squads found - auto-deploying all player squads for encounter")
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
	playerID ecs.EntityID,
	manager *common.EntityManager,
	factionID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
) error {
	// Get player's squad roster
	roster := squads.GetPlayerSquadRoster(playerID, manager)
	if roster == nil {
		return fmt.Errorf("player has no squad roster")
	}

	// Get deployed squads (should already be deployed)
	deployedSquads := roster.GetDeployedSquads(manager)
	if len(deployedSquads) == 0 {
		return fmt.Errorf("player has no squads - this should not happen after ensurePlayerSquadsDeployed()")
	}

	fmt.Printf("Adding %d player squads to faction\n", len(deployedSquads))

	// Position player squads around starting position
	squadPositions := generatePlayerSquadPositions(playerStartPos, len(deployedSquads))

	// Add each deployed squad to faction
	for i, squadID := range deployedSquads {
		pos := squadPositions[i]

		// Add to faction (handles creating/updating squad position)
		if err := fm.AddSquadToFaction(factionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add squad %d to faction: %w", squadID, err)
		}

		// Ensure all squad units have positions at the squad's location
		ensureUnitPositions(manager, squadID, pos)

		// Create action state
		if err := createActionStateForSquad(manager, squadID); err != nil {
			return fmt.Errorf("failed to create action state for squad %d: %w", squadID, err)
		}
	}

	return nil
}

// generatePlayerSquadPositions creates positions for player squads around starting point
func generatePlayerSquadPositions(startPos coords.LogicalPosition, count int) []coords.LogicalPosition {
	positions := make([]coords.LogicalPosition, count)

	// Arrange squads in an arc behind/around the player
	for i := 0; i < count; i++ {
		// Position squads at distance 3-6 tiles from start, in a defensive arc
		angle := (float64(i) / float64(count)) * math.Pi // 0 to Pi (half circle)
		angle = angle - math.Pi/2                        // Rotate to face forward
		distance := 3 + (i % 2)                          // Alternate 3 and 4 distance

		offsetX := int(math.Round(float64(distance) * math.Cos(angle)))
		offsetY := int(math.Round(float64(distance) * math.Sin(angle)))

		positions[i] = coords.LogicalPosition{
			X: clampPosition(startPos.X+offsetX, 0, 99),
			Y: clampPosition(startPos.Y+offsetY, 0, 79),
		}
	}

	return positions
}

// getEncounterDifficulty extracts difficulty modifier from encounter data
func getEncounterDifficulty(encounterData *core.OverworldEncounterData) EncounterDifficultyModifier {

	level := encounterData.Level
	return GetDifficultyModifier(level) // Falls back to level 3 if invalid
}

// generateEnemySquadsByPower creates enemy squads matching target squad power.
// targetPower is now the per-squad target (average player squad power * difficulty).
// Returns EnemySquadSpec which includes Type and Name for full encounter specification.
func generateEnemySquadsByPower(
	manager *common.EntityManager,
	targetSquadPower float64,
	difficultyMod EncounterDifficultyModifier,
	encounterData *core.OverworldEncounterData,
	playerPos coords.LogicalPosition,
	config *evaluation.PowerConfig,
) []EnemySquadSpec {
	// Use fixed squad count from difficulty modifier
	squadCount := difficultyMod.SquadCount

	enemySquads := []EnemySquadSpec{}

	// Get squad composition preferences
	squadTypes := getSquadComposition(encounterData, squadCount)

	for i := 0; i < squadCount; i++ {
		// Generate position around player (circular distribution)
		pos := generateEnemyPosition(playerPos, i, squadCount)
		squadName := fmt.Sprintf("Enemy Squad %d", i+1)

		// Each enemy squad targets the same power (average player squad power * difficulty)
		squadID := createSquadForPowerBudget(
			manager,
			targetSquadPower,
			squadTypes[i],
			squadName,
			pos,
			config,
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

// generateEnemyPosition scatters enemies around player using circular distribution
func generateEnemyPosition(playerPos coords.LogicalPosition, index, total int) coords.LogicalPosition {
	// Circular distribution at fixed distance
	angle := (float64(index) / float64(total)) * 2.0 * math.Pi
	distance := 10 // Fixed distance from player

	offsetX := int(math.Round(float64(distance) * math.Cos(angle)))
	offsetY := int(math.Round(float64(distance) * math.Sin(angle)))

	x := clampPosition(playerPos.X+offsetX, 0, 99)
	y := clampPosition(playerPos.Y+offsetY, 0, 79)

	return coords.LogicalPosition{X: x, Y: y}
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
) ecs.EntityID {
	fmt.Printf("[DEBUG] Creating squad '%s' with target power: %.2f\n", name, targetPower)

	// Select unit pool based on squad type
	unitPool := filterUnitsBySquadType(squadType)
	if len(unitPool) == 0 {
		unitPool = squads.Units // Fallback to all units
	}

	if len(unitPool) == 0 {
		return 0 // No units available
	}

	fmt.Printf("[DEBUG] Unit pool size: %d units of type '%s'\n", len(unitPool), squadType)

	// Iteratively add units until power budget reached
	unitsToCreate := []squads.UnitTemplate{}
	currentPower := 0.0
	// Use safe grid positions that work for 2-wide units (avoid rightmost column)
	// Pattern: Front row (0,0 and 0,1), middle row (1,0 and 1,1), back row (2,0)
	gridPositions := [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 0}}

	for currentPower < targetPower && len(unitsToCreate) < 5 {
		// Pick random unit from pool
		unit := unitPool[common.RandomInt(len(unitPool))]

		// Estimate unit power contribution using shared function
		unitPower := evaluation.EstimateUnitPowerFromTemplate(unit, config)
		fmt.Printf("[DEBUG] Unit '%s' power: %.2f (current: %.2f / target: %.2f)\n",
			unit.Name, unitPower, currentPower, targetPower)

		// Set grid position
		unit.GridRow = gridPositions[len(unitsToCreate)][0]
		unit.GridCol = gridPositions[len(unitsToCreate)][1]
		unit.IsLeader = (len(unitsToCreate) == 0) // First unit is leader

		unitsToCreate = append(unitsToCreate, unit)
		currentPower += unitPower

		// Stop if we've reached 95% of target (was 85% - increased to allow fuller squads)
		if currentPower >= targetPower*0.95 {
			fmt.Printf("[DEBUG] Stopping - reached 95%% of target (%.2f >= %.2f)\n",
				currentPower, targetPower*0.95)
			break
		}
	}

	fmt.Printf("[DEBUG] After power loop: %d units created\n", len(unitsToCreate))

	// Ensure at least 3 units
	for len(unitsToCreate) < 3 && len(unitPool) > 0 {
		unit := unitPool[common.RandomInt(len(unitPool))]
		unit.GridRow = gridPositions[len(unitsToCreate)][0]
		unit.GridCol = gridPositions[len(unitsToCreate)][1]
		unitsToCreate = append(unitsToCreate, unit)
		fmt.Printf("[DEBUG] Added unit to reach minimum (now %d units)\n", len(unitsToCreate))
	}

	fmt.Printf("[DEBUG] Final unit count: %d units\n", len(unitsToCreate))

	// Set leader attributes
	if len(unitsToCreate) > 0 {
		unitsToCreate[0].Attributes.Leadership = 20
	}

	// Create squad
	squadID := squads.CreateSquadFromTemplate(
		manager,
		name,
		squads.FormationBalanced,
		position,
		unitsToCreate,
	)

	fmt.Printf("[DEBUG] Created squad ID: %d\n\n", squadID)
	return squadID
}

// filterUnitsBySquadType selects units matching squad archetype
func filterUnitsBySquadType(squadType string) []squads.UnitTemplate {
	switch squadType {
	case SquadTypeRanged:
		return filterUnitsByAttackRange(3) // Range >= 3
	case SquadTypeMagic:
		return filterUnitsByAttackType(squads.AttackTypeMagic)
	case SquadTypeMelee:
		// Melee: range <= 2
		var filtered []squads.UnitTemplate
		for _, unit := range squads.Units {
			if unit.AttackRange <= 2 {
				filtered = append(filtered, unit)
			}
		}
		return filtered
	default:
		return squads.Units
	}
}

// Helper functions

// createActionStateForSquad creates the ActionStateData component for a squad.
// Delegates to combat package's CreateActionStateForSquad.
func createActionStateForSquad(manager *common.EntityManager, squadID ecs.EntityID) error {
	return combat.CreateActionStateForSquad(manager, squadID)
}

// ensureUnitPositions ensures all units in a squad have position components
// Units that already have positions are moved to the squad position
// Units without positions get a new position component created
func ensureUnitPositions(manager *common.EntityManager, squadID ecs.EntityID, squadPos coords.LogicalPosition) {
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	for _, unitID := range unitIDs {
		unitEntity := manager.FindEntityByID(unitID)
		if unitEntity == nil {
			continue
		}

		unitPos := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
		if unitPos != nil {
			// Unit has position - move it to squad location
			manager.MoveEntity(unitID, unitEntity, *unitPos, squadPos)
		} else {
			// Unit has no position - create one at squad location
			newPos := new(coords.LogicalPosition)
			*newPos = squadPos
			unitEntity.AddComponent(common.PositionComponent, newPos)
			common.GlobalPositionSystem.AddEntity(unitID, squadPos)
		}
	}
}

// filterUnitsByAttackRange returns units with attack range >= minRange
func filterUnitsByAttackRange(minRange int) []squads.UnitTemplate {
	var filtered []squads.UnitTemplate
	for _, unit := range squads.Units {
		if unit.AttackRange >= minRange {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}

// filterUnitsByAttackType returns units matching the specified attack type
func filterUnitsByAttackType(attackType squads.AttackType) []squads.UnitTemplate {
	var filtered []squads.UnitTemplate
	for _, unit := range squads.Units {
		if unit.AttackType == attackType {
			filtered = append(filtered, unit)
		}
	}
	return filtered
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
