package encounter

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/evaluation"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"math"

	"github.com/bytearena/ecs"
)

// TODO, this should be moved to the encounter package
// EnemySquadInfo holds information about a generated enemy squad
type EnemySquadInfo struct {
	SquadID  ecs.EntityID
	Position coords.LogicalPosition
	Power    float64
}

// SetupBalancedEncounter creates player and enemy factions with power-based squad generation
// Replaces SetupGameplayFactions with dynamic encounter balancing
// Returns a list of enemy squad IDs created for this encounter
func SetupBalancedEncounter(
	manager *common.EntityManager,
	playerEntityID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
	encounterData *OverworldEncounterData,
) ([]ecs.EntityID, error) {
	// Track created enemy squad IDs for cleanup
	createdEnemySquadIDs := []ecs.EntityID{}

	// 1. Use provided player entity ID
	if playerEntityID == 0 {
		return nil, fmt.Errorf("invalid player entity ID")
	}

	// 2. Ensure player has deployed squads (auto-deploy if needed)
	if err := ensurePlayerSquadsDeployed(playerEntityID, manager); err != nil {
		return nil, fmt.Errorf("failed to deploy player squads: %w", err)
	}

	// 3. Calculate average squad power (squad-centric approach)
	config := GetDefaultConfig()
	roster := squads.GetPlayerSquadRoster(playerEntityID, manager)
	if roster == nil {
		return nil, fmt.Errorf("player has no squad roster")
	}

	deployedSquads := roster.GetDeployedSquads(manager)
	if len(deployedSquads) == 0 {
		return nil, fmt.Errorf("no deployed squads - this should not happen after ensurePlayerSquadsDeployed()")
	}

	// Calculate average power per squad
	totalPlayerPower := 0.0
	for _, squadID := range deployedSquads {
		squadPower := CalculateSquadPower(squadID, manager, config)
		totalPlayerPower += squadPower
	}
	avgPlayerSquadPower := totalPlayerPower / float64(len(deployedSquads))

	// 4. Determine enemy difficulty from encounter data
	difficultyMod := getEncounterDifficulty(encounterData)
	targetEnemySquadPower := avgPlayerSquadPower * difficultyMod.PowerMultiplier

	// Handle edge cases
	if avgPlayerSquadPower <= 0.0 {
		// Fallback if power calculation failed - use minimal encounter
		targetEnemySquadPower = 50.0
		difficultyMod.MinSquads = 1
		difficultyMod.MaxSquads = 1
	}
	if targetEnemySquadPower > 2000.0 {
		// Cap per-squad power to prevent overpowered units
		targetEnemySquadPower = 2000.0
	}

	fmt.Printf("Player: %d squads, Avg Power: %.2f | Target Enemy Squad Power: %.2f (%.0f%% multiplier)\n",
		len(deployedSquads), avgPlayerSquadPower, targetEnemySquadPower, difficultyMod.PowerMultiplier*100)

	// 5. Create factions
	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)
	playerFactionID := fm.CreateFactionWithPlayer("Player Forces", 1, "Player 1")
	enemyFactionID := fm.CreateFactionWithPlayer("Enemy Forces", 0, "")

	// 6. Add player's deployed squads to faction
	if err := assignPlayerSquadsToFaction(fm, playerEntityID, manager, playerFactionID, playerStartPos); err != nil {
		return nil, fmt.Errorf("failed to assign player squads: %w", err)
	}

	// 6. Generate enemy squads targeting average player squad power
	enemySquads := generateEnemySquadsByPower(
		manager,
		targetEnemySquadPower,
		difficultyMod,
		encounterData,
		playerStartPos,
		config,
	)

	// 7. Add enemy squads to faction and track their IDs
	for i, squadInfo := range enemySquads {
		if err := fm.AddSquadToFaction(enemyFactionID, squadInfo.SquadID, squadInfo.Position); err != nil {
			return nil, fmt.Errorf("failed to add enemy squad %d to faction: %w", i, err)
		}
		if err := createActionStateForSquad(manager, squadInfo.SquadID); err != nil {
			return nil, fmt.Errorf("failed to create action state for enemy squad %d: %w", i, err)
		}
		// Track this enemy squad ID for cleanup
		createdEnemySquadIDs = append(createdEnemySquadIDs, squadInfo.SquadID)
	}

	fmt.Printf("Created encounter: Player Faction (%d) vs Enemy Faction (%d) with %d squads\n",
		playerFactionID, enemyFactionID, len(enemySquads))

	return createdEnemySquadIDs, nil
}

// getPlayerEntityID is no longer used - player entity ID is now passed as a parameter
// Kept for reference - DO NOT USE (iterating 10000 entities is wasteful)
/*
func getPlayerEntityID(manager *common.EntityManager) ecs.EntityID {
	for id := ecs.EntityID(1); id < ecs.EntityID(10000); id++ {
		entity := manager.FindEntityByID(id)
		if entity != nil && entity.HasComponent(common.PlayerComponent) {
			return id
		}
	}
	return 0
}
*/

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

		// Update squad's position component for combat
		squadEntity := manager.FindEntityByID(squadID)
		if squadEntity != nil {
			squadPos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
			if squadPos != nil {
				// Update position for combat
				oldPos := *squadPos
				manager.MoveEntity(squadID, squadEntity, oldPos, pos)
			}
		}

		// Add to faction
		if err := fm.AddSquadToFaction(factionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add squad %d to faction: %w", squadID, err)
		}

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
func getEncounterDifficulty(encounterData *OverworldEncounterData) EncounterDifficultyModifier {
	if encounterData == nil {
		// Default to level 2 for testing
		return EncounterDifficultyTable[2]
	}

	level := encounterData.Level
	if mod, exists := EncounterDifficultyTable[level]; exists {
		return mod
	}

	// Fallback to balanced encounter
	return EncounterDifficultyTable[3]
}

// generateEnemySquadsByPower creates enemy squads matching target squad power
// targetPower is now the per-squad target (average player squad power * difficulty)
func generateEnemySquadsByPower(
	manager *common.EntityManager,
	targetSquadPower float64,
	difficultyMod EncounterDifficultyModifier,
	encounterData *OverworldEncounterData,
	playerPos coords.LogicalPosition,
	config *EvaluationConfigData,
) []EnemySquadInfo {
	// Determine number of squads
	squadCount := common.GetRandomBetween(difficultyMod.MinSquads, difficultyMod.MaxSquads)

	enemySquads := []EnemySquadInfo{}

	// Get squad composition preferences
	squadTypes := getSquadComposition(encounterData, squadCount)

	for i := 0; i < squadCount; i++ {
		// Generate position around player (circular distribution)
		pos := generateEnemyPosition(playerPos, i, squadCount)

		// Each enemy squad targets the same power (average player squad power * difficulty)
		squadID := createSquadForPowerBudget(
			manager,
			targetSquadPower,
			squadTypes[i],
			fmt.Sprintf("Enemy Squad %d", i+1),
			pos,
			config,
		)

		if squadID != 0 {
			enemySquads = append(enemySquads, EnemySquadInfo{
				SquadID:  squadID,
				Position: pos,
				Power:    targetSquadPower,
			})
		}
	}

	return enemySquads
}

// getSquadComposition returns squad type distribution based on encounter type
func getSquadComposition(encounterData *OverworldEncounterData, count int) []string {
	if encounterData == nil || encounterData.EncounterType == "" {
		// Random balanced composition
		return generateRandomComposition(count)
	}

	// Use encounter preferences
	preferences := EncounterSquadPreferences[encounterData.EncounterType]
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

// createSquadForPowerBudget creates a squad matching target power
func createSquadForPowerBudget(
	manager *common.EntityManager,
	targetPower float64,
	squadType string,
	name string,
	position coords.LogicalPosition,
	config *EvaluationConfigData,
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

		// Estimate unit power contribution
		unitPower := estimateUnitPower(unit, config)
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

// estimateUnitPower calculates accurate power contribution matching CalculateUnitPower
// Now properly includes all sub-weights, accuracy modifiers, defensive components, and utility scaling
func estimateUnitPower(unit squads.UnitTemplate, config *EvaluationConfigData) float64 {
	attr := &unit.Attributes

	// === OFFENSIVE POWER (matches calculateOffensivePower) ===
	physicalDmg := float64(attr.GetPhysicalDamage())
	magicDmg := float64(attr.GetMagicDamage())
	avgDamage := (physicalDmg + magicDmg) / 2.0

	// Apply accuracy modifiers
	hitRate := float64(attr.GetHitRate()) / 100.0 // Normalize to 0-1
	critChance := float64(attr.GetCritChance()) / 100.0
	critMultiplier := 1.0 + (critChance * 0.5) // CritDamageMultiplier = 0.5
	effectiveDamage := avgDamage * hitRate * critMultiplier

	// Sub-weighted combination
	damageComponent := avgDamage * config.DamageWeight
	accuracyComponent := effectiveDamage * config.AccuracyWeight
	offensivePower := damageComponent + accuracyComponent

	// === DEFENSIVE POWER (matches calculateDefensivePower) ===
	maxHP := float64(attr.GetMaxHealth())
	effectiveHealth := maxHP // Assume full HP for new units

	// Resistance component
	physicalResist := float64(attr.GetPhysicalResistance())
	magicResist := float64(attr.GetMagicDefense())
	avgResistance := (physicalResist + magicResist) / 2.0

	// Avoidance component
	dodgeChance := float64(attr.GetDodgeChance()) / 100.0
	dodgeScaled := dodgeChance * 100.0 // DodgeScalingFactor = 100.0

	// Sub-weighted combination
	healthComponent := effectiveHealth * config.HealthWeight
	resistanceComponent := avgResistance * config.ResistanceWeight
	avoidanceComponent := dodgeScaled * config.AvoidanceWeight
	defensivePower := healthComponent + resistanceComponent + avoidanceComponent

	// === UTILITY POWER (matches calculateUtilityPower) ===
	// Role component
	roleMultiplier := evaluation.GetRoleMultiplier(unit.Role)
	roleValue := roleMultiplier * 10.0 // RoleScalingFactor = 10.0
	roleComponent := roleValue * config.RoleWeight

	// Ability component (simplified - assume leader gets average ability value)
	abilityValue := 0.0
	if unit.IsLeader {
		abilityValue = 15.0 // Average of Rally (15.0), Heal (20.0), BattleCry (12.0)
	}
	abilityComponent := abilityValue * config.AbilityWeight

	// Cover component
	coverValue := 0.0
	if unit.CoverValue > 0 {
		coverValue = unit.CoverValue * 100.0 * 2.5 // CoverScalingFactor * CoverBeneficiaryMultiplier
	}
	coverComponent := coverValue * config.CoverWeight

	utilityPower := roleComponent + abilityComponent + coverComponent

	// === WEIGHTED SUM ===
	totalPower := (offensivePower * config.OffensiveWeight) +
		(defensivePower * config.DefensiveWeight) +
		(utilityPower * config.UtilityWeight)

	return totalPower
}

// Helper functions

// createActionStateForSquad creates the ActionStateData component for a squad
// Uses the existing GetSquadMovementSpeed function from squads package
func createActionStateForSquad(manager *common.EntityManager, squadID ecs.EntityID) error {
	// Create ActionStateData entity
	actionEntity := manager.World.NewEntity()

	// Get squad movement speed using existing function
	movementSpeed := squads.GetSquadMovementSpeed(squadID, manager)
	if movementSpeed == 0 {
		movementSpeed = 3 // Default if no valid units found
	}

	actionEntity.AddComponent(combat.ActionStateComponent, &combat.ActionStateData{
		SquadID:           squadID,
		HasMoved:          false,
		HasActed:          false,
		MovementRemaining: movementSpeed,
	})

	return nil
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
