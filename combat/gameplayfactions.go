package combat

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"
	"math"

	"github.com/bytearena/ecs"
)

//TODO: Remove this in the future. Just here for testing

// SetupGameplayFactions creates two factions (player and AI) with squads for gameplay testing.
// This is called during game initialization to create the initial combat setup.
// Each faction gets 3 squads positioned on the map.
// Note: Combat components are already registered in InitializeECS
func SetupGameplayFactions(manager *common.EntityManager, playerStartPos coords.LogicalPosition) error {
	// 1. Create FactionManager
	fm := NewFactionManager(manager)

	// 2. Create Player Faction
	playerFactionID := fm.CreateFaction("Player Alliance", true)

	// 3. Create AI Faction
	aiFactionID := fm.CreateFaction("Goblin Horde", false)

	// 4. Check if we have units available
	if len(squads.Units) == 0 {
		return fmt.Errorf("no units available - call squads.InitUnitTemplatesFromJSON() first")
	}

	// 5. Create player squads positioned above player (north side)
	// Squad positions relative to player start:
	// - Squad 1: (-3, -3) from player (northwest)
	// - Squad 2: (+3, -3) from player (northeast)
	// - Squad 3: (0, +3) from player (south - slightly behind)
	playerSquadPositions := []coords.LogicalPosition{
		{X: clampPosition(playerStartPos.X-3, 0, 99), Y: clampPosition(playerStartPos.Y-3, 0, 79)},
		{X: clampPosition(playerStartPos.X+3, 0, 99), Y: clampPosition(playerStartPos.Y-3, 0, 79)},
		{X: clampPosition(playerStartPos.X, 0, 99), Y: clampPosition(playerStartPos.Y+3, 0, 79)},
	}

	for i, pos := range playerSquadPositions {
		squadID, err := createSquadForFaction(manager, fmt.Sprintf("Player Squad %d", i+1), pos)
		if err != nil {
			return fmt.Errorf("failed to create player squad %d: %w", i+1, err)
		}

		// Add squad to faction
		if err := fm.AddSquadToFaction(playerFactionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add squad to player faction: %w", err)
		}

		// Create ActionStateData for squad using existing squad movement speed function
		if err := createActionStateForSquad(manager, squadID); err != nil {
			return fmt.Errorf("failed to create action state for player squad: %w", err)
		}
	}

	// 6. Create AI squads positioned below player (south side)
	// AI squads are mirrored across the player, creating engagement distance
	// Squad positions relative to player start:
	// - Squad 1: (-3, +3) from player (southwest)
	// - Squad 2: (+3, +3) from player (southeast)
	// - Squad 3: (0, -3) from player (north - slightly ahead)
	aiSquadPositions := []coords.LogicalPosition{
		{X: clampPosition(playerStartPos.X-3, 0, 99), Y: clampPosition(playerStartPos.Y+3, 0, 79)},
		{X: clampPosition(playerStartPos.X+3, 0, 99), Y: clampPosition(playerStartPos.Y+3, 0, 79)},
		{X: clampPosition(playerStartPos.X, 0, 99), Y: clampPosition(playerStartPos.Y-3, 0, 79)},
	}

	for i, pos := range aiSquadPositions {
		squadID, err := createSquadForFaction(manager, fmt.Sprintf("Goblin Squad %d", i+1), pos)
		if err != nil {
			return fmt.Errorf("failed to create AI squad %d: %w", i+1, err)
		}

		// Add squad to faction
		if err := fm.AddSquadToFaction(aiFactionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add squad to AI faction: %w", err)
		}

		// Create ActionStateData for squad
		if err := createActionStateForSquad(manager, squadID); err != nil {
			return fmt.Errorf("failed to create action state for AI squad: %w", err)
		}
	}

	// 7. Create 5 additional random test squads for player faction
	// These squads have random unit counts (1-5), random unit types, and random leaders
	// Positioned randomly 3-10 tiles from player for testing variety
	playerTestSquadCount := 5
	for i := 0; i < playerTestSquadCount; i++ {
		// Generate random position 3-10 tiles from player
		position := generateRandomPositionNearPlayer(playerStartPos, 3, 10)

		// Create squad with random units and leader
		squadName := fmt.Sprintf("Test Squad %d", i+1)
		err := createRandomSquad(fm, manager, playerFactionID, squadName, position)
		if err != nil {
			return fmt.Errorf("failed to create test squad %d: %w", i+1, err)
		}
	}

	// 8. Create 5 additional random enemy squads for AI faction
	// These squads have random unit counts (1-5), random unit types, and random leaders
	// Positioned randomly 5-15 tiles from player for testing variety
	enemyTestSquadCount := 5
	for i := 0; i < enemyTestSquadCount; i++ {
		// Generate random position 5-15 tiles from player (further out than player squads)
		position := generateRandomPositionNearPlayer(playerStartPos, 5, 15)

		// Create squad with random units and leader
		squadName := fmt.Sprintf("Enemy Squad %d", i+1)
		err := createRandomSquad(fm, manager, aiFactionID, squadName, position)
		if err != nil {
			return fmt.Errorf("failed to create enemy test squad %d: %w", i+1, err)
		}
	}

	// 9. Create ranged-focused test squads for player faction
	// These squads only contain ranged attackers (Archer, Ranger, Crossbowman, Marksman, Skeleton Archer)
	rangedSquadCount := 3
	for i := 0; i < rangedSquadCount; i++ {
		position := generateRandomPositionNearPlayer(playerStartPos, 4, 8)
		squadName := fmt.Sprintf("Ranged Squad %d", i+1)
		err := createRangedSquad(fm, manager, playerFactionID, squadName, position)
		if err != nil {
			return fmt.Errorf("failed to create ranged squad %d: %w", i+1, err)
		}
	}

	// 10. Create magic-focused test squads for player faction
	// These squads only contain magic attackers (Wizard, Sorcerer, Mage, Cleric, Priest, Warlock, Battle Mage)
	magicSquadCount := 3
	for i := 0; i < magicSquadCount; i++ {
		position := generateRandomPositionNearPlayer(playerStartPos, 4, 8)
		squadName := fmt.Sprintf("Magic Squad %d", i+1)
		err := createMagicSquad(fm, manager, playerFactionID, squadName, position)
		if err != nil {
			return fmt.Errorf("failed to create magic squad %d: %w", i+1, err)
		}
	}

	// 11. Create mixed ranged/magic squads for AI faction
	mixedSquadCount := 2
	for i := 0; i < mixedSquadCount; i++ {
		position := generateRandomPositionNearPlayer(playerStartPos, 6, 12)
		squadName := fmt.Sprintf("Mixed Ranged/Magic %d", i+1)
		err := createMixedRangedMagicSquad(fm, manager, aiFactionID, squadName, position)
		if err != nil {
			return fmt.Errorf("failed to create mixed ranged/magic squad %d: %w", i+1, err)
		}
	}

	fmt.Printf("Created gameplay factions:\n")
	fmt.Printf("  Player faction (%d) with %d squads (%d standard + %d random + %d ranged + %d magic)\n",
		playerFactionID, len(playerSquadPositions)+playerTestSquadCount+rangedSquadCount+magicSquadCount,
		len(playerSquadPositions), playerTestSquadCount, rangedSquadCount, magicSquadCount)
	fmt.Printf("  AI faction (%d) with %d squads (%d standard + %d random + %d mixed)\n",
		aiFactionID, len(aiSquadPositions)+enemyTestSquadCount+mixedSquadCount,
		len(aiSquadPositions), enemyTestSquadCount, mixedSquadCount)

	return nil
}

// createSquadForFaction creates a squad with units from available templates
func createSquadForFaction(manager *common.EntityManager, squadName string, position coords.LogicalPosition) (ecs.EntityID, error) {
	// Get available units (first 5 units from the Units array to create variety)
	unitsToCreate := []squads.UnitTemplate{}

	// We'll create a balanced squad with up to 5 units
	// Use the first few units from the Units array
	maxUnits := 5
	if len(squads.Units) < maxUnits {
		maxUnits = len(squads.Units)
	}

	// Create copies of units and set their grid positions
	positions := [][2]int{
		{0, 0}, // Front left
		{0, 1}, // Front center
		{0, 2}, // Front right
		{1, 1}, // Middle center
		{2, 1}, // Back center
	}

	// Randomly select which unit will be the leader
	leaderIndex := common.RandomInt(maxUnits)

	for i := 0; i < maxUnits && i < len(positions); i++ {
		// Create a copy of the unit template
		unit := squads.Units[i%len(squads.Units)]

		// Set grid position
		unit.GridRow = positions[i][0]
		unit.GridCol = positions[i][1]

		// Make the randomly selected unit the leader
		if i == leaderIndex {
			unit.IsLeader = true
			// Boost leadership for capacity
			unit.Attributes.Leadership = 20
		}

		unitsToCreate = append(unitsToCreate, unit)
	}

	// Create squad using the squad creation function
	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationBalanced,
		position,
		unitsToCreate,
	)

	return squadID, nil
}

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

	actionEntity.AddComponent(ActionStateComponent, &ActionStateData{
		SquadID:           squadID,
		HasMoved:          false,
		HasActed:          false,
		MovementRemaining: movementSpeed,
	})

	return nil
}

// createRandomSquad creates a squad with randomly selected units, random leader, and random size (1-5 units)
// This is used for testing purposes to create variety in squad composition
func createRandomSquad(
	fm *FactionManager,
	manager *common.EntityManager,
	factionID ecs.EntityID,
	squadName string,
	position coords.LogicalPosition,
) error {
	// 1. Determine random unit count (1-5)
	unitCount := common.GetRandomBetween(1, 5)

	// 2. Select random units from squads.Units (duplicates allowed)
	unitsToCreate := []squads.UnitTemplate{}

	// Grid positions for units (left-to-right, top-to-bottom)
	gridPositions := [][2]int{
		{0, 0}, // Front left
		{0, 1}, // Front center
		{0, 2}, // Front right
		{1, 0}, // Middle left
		{1, 1}, // Middle center
	}

	for i := 0; i < unitCount; i++ {
		// Select random unit index
		randomIdx := common.RandomInt(len(squads.Units))
		unit := squads.Units[randomIdx]

		// Set grid position (use sequential positions for simplicity)
		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]

		// Reset leader flag (we'll set it later)
		unit.IsLeader = false

		unitsToCreate = append(unitsToCreate, unit)
	}

	// 3. Randomly select leader index
	leaderIndex := common.RandomInt(unitCount)

	// 4. Mark the selected unit as leader and boost leadership
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	// 5. Create squad using CreateSquadFromTemplate
	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationBalanced,
		position,
		unitsToCreate,
	)

	// 6. Add squad to faction
	if err := fm.AddSquadToFaction(factionID, squadID, position); err != nil {
		return fmt.Errorf("failed to add squad to faction: %w", err)
	}

	// 7. Create action state for squad
	if err := createActionStateForSquad(manager, squadID); err != nil {
		return fmt.Errorf("failed to create action state: %w", err)
	}

	return nil
}

// clampPosition constrains a coordinate to valid map bounds
// Map is 100x80 (0-99 for X, 0-79 for Y)
func clampPosition(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// generateRandomPositionNearPlayer creates a random position around the player
// using circular distribution to scatter squads naturally
// minDist and maxDist control the distance range (in tiles) from the player
func generateRandomPositionNearPlayer(playerPos coords.LogicalPosition, minDist, maxDist int) coords.LogicalPosition {
	// Generate random angle (0 to 2π radians for full circle)
	angle := common.RandomFloat() * 2.0 * math.Pi

	// Generate random distance within range
	distance := common.GetRandomBetween(minDist, maxDist)

	// Calculate X,Y offset using circular trigonometry
	offsetX := int(math.Round(float64(distance) * math.Cos(angle)))
	offsetY := int(math.Round(float64(distance) * math.Sin(angle)))

	// Apply offset and clamp to map bounds
	newX := clampPosition(playerPos.X+offsetX, 0, 99)
	newY := clampPosition(playerPos.Y+offsetY, 0, 79)

	return coords.LogicalPosition{X: newX, Y: newY}
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

// createRangedSquad creates a squad with only ranged attackers
// Units are spread across different rows for better testing of ranged targeting
func createRangedSquad(
	fm *FactionManager,
	manager *common.EntityManager,
	factionID ecs.EntityID,
	squadName string,
	position coords.LogicalPosition,
) error {
	// Get all ranged units
	rangedUnits := filterUnitsByAttackType(squads.AttackTypeRanged)
	if len(rangedUnits) == 0 {
		return fmt.Errorf("no ranged units available")
	}

	// Create squad with 3-5 ranged units
	unitCount := common.GetRandomBetween(3, 5)
	unitsToCreate := []squads.UnitTemplate{}

	// Spread units across all three rows for testing ranged targeting
	gridPositions := [][2]int{
		{0, 0}, // Row 0 - Front
		{1, 1}, // Row 1 - Middle
		{2, 2}, // Row 2 - Back
		{0, 2}, // Row 0 - Front right
		{1, 0}, // Row 1 - Middle left
	}

	for i := 0; i < unitCount; i++ {
		// Randomly select a ranged unit
		randomIdx := common.RandomInt(len(rangedUnits))
		unit := rangedUnits[randomIdx]

		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = false

		unitsToCreate = append(unitsToCreate, unit)
	}

	// Set random leader
	leaderIndex := common.RandomInt(unitCount)
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	// Create squad
	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationBalanced,
		position,
		unitsToCreate,
	)

	if err := fm.AddSquadToFaction(factionID, squadID, position); err != nil {
		return err
	}

	return createActionStateForSquad(manager, squadID)
}

// createMagicSquad creates a squad with only magic attackers
// Units are spread across different rows for better testing of magic targeting patterns
func createMagicSquad(
	fm *FactionManager,
	manager *common.EntityManager,
	factionID ecs.EntityID,
	squadName string,
	position coords.LogicalPosition,
) error {
	// Get all magic units
	magicUnits := filterUnitsByAttackType(squads.AttackTypeMagic)
	if len(magicUnits) == 0 {
		return fmt.Errorf("no magic units available")
	}

	// Create squad with 3-5 magic units
	unitCount := common.GetRandomBetween(3, 5)
	unitsToCreate := []squads.UnitTemplate{}

	// Spread units across all three rows for testing magic targeting patterns
	gridPositions := [][2]int{
		{0, 1}, // Row 0 - Front center
		{1, 0}, // Row 1 - Middle left
		{2, 1}, // Row 2 - Back center
		{0, 0}, // Row 0 - Front left
		{2, 0}, // Row 2 - Back left
	}

	for i := 0; i < unitCount; i++ {
		// Randomly select a magic unit
		randomIdx := common.RandomInt(len(magicUnits))
		unit := magicUnits[randomIdx]

		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = false

		unitsToCreate = append(unitsToCreate, unit)
	}

	// Set random leader
	leaderIndex := common.RandomInt(unitCount)
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	// Create squad
	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationBalanced,
		position,
		unitsToCreate,
	)

	if err := fm.AddSquadToFaction(factionID, squadID, position); err != nil {
		return err
	}

	return createActionStateForSquad(manager, squadID)
}

// createMixedRangedMagicSquad creates a squad with a mix of ranged and magic attackers
// Units are spread across different rows for comprehensive testing
func createMixedRangedMagicSquad(
	fm *FactionManager,
	manager *common.EntityManager,
	factionID ecs.EntityID,
	squadName string,
	position coords.LogicalPosition,
) error {
	rangedUnits := filterUnitsByAttackType(squads.AttackTypeRanged)
	magicUnits := filterUnitsByAttackType(squads.AttackTypeMagic)

	if len(rangedUnits) == 0 || len(magicUnits) == 0 {
		return fmt.Errorf("insufficient ranged or magic units")
	}

	// Create squad with 4-5 units (mix of ranged and magic)
	unitCount := common.GetRandomBetween(4, 5)
	unitsToCreate := []squads.UnitTemplate{}

	// Spread units across all three rows for testing
	gridPositions := [][2]int{
		{0, 0}, // Row 0 - Front left
		{1, 1}, // Row 1 - Middle center
		{2, 2}, // Row 2 - Back right
		{1, 2}, // Row 1 - Middle right
		{2, 0}, // Row 2 - Back left
	}

	for i := 0; i < unitCount; i++ {
		var unit squads.UnitTemplate

		// Alternate between ranged and magic, or randomize
		if i%2 == 0 && len(rangedUnits) > 0 {
			randomIdx := common.RandomInt(len(rangedUnits))
			unit = rangedUnits[randomIdx]
		} else if len(magicUnits) > 0 {
			randomIdx := common.RandomInt(len(magicUnits))
			unit = magicUnits[randomIdx]
		} else if len(rangedUnits) > 0 {
			randomIdx := common.RandomInt(len(rangedUnits))
			unit = rangedUnits[randomIdx]
		}

		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = false

		unitsToCreate = append(unitsToCreate, unit)
	}

	// Set random leader
	leaderIndex := common.RandomInt(unitCount)
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	// Create squad
	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationBalanced,
		position,
		unitsToCreate,
	)

	if err := fm.AddSquadToFaction(factionID, squadID, position); err != nil {
		return err
	}

	return createActionStateForSquad(manager, squadID)
}
