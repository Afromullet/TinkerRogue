package combat

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"
	"game_main/tactical/squads"
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

	// 2. Create Player 1 Faction
	playerFactionID := fm.CreateFactionWithPlayer("Player 1 Alliance", 1, "Player 1")

	// 3. Create Player 2 Faction (hot-seat multiplayer)
	player2FactionID := fm.CreateFactionWithPlayer("Player 2 Horde", 2, "Player 2")

	// 4. Create AI Goblin Faction
	aiFactionID := fm.CreateFactionWithPlayer("Goblin Horde", 0, "")

	// 5. Check if we have units available
	if len(squads.Units) == 0 {
		return fmt.Errorf("no units available - call squads.InitUnitTemplatesFromJSON() first")
	}

	// 6. Create Player 1 squads positioned around starting area
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

	// 6a. Add two ranged-only squads for Player 1
	playerRangedSquadPositions := []coords.LogicalPosition{
		{X: clampPosition(playerStartPos.X-6, 0, 99), Y: clampPosition(playerStartPos.Y+6, 0, 79)},
		{X: clampPosition(playerStartPos.X+6, 0, 99), Y: clampPosition(playerStartPos.Y+6, 0, 79)},
	}

	for i, pos := range playerRangedSquadPositions {
		squadID, err := createRangedSquadByAttackRange(manager, fmt.Sprintf("Player Ranged Squad %d", i+1), pos)
		if err != nil {
			return fmt.Errorf("failed to create player ranged squad %d: %w", i+1, err)
		}

		// Add squad to faction
		if err := fm.AddSquadToFaction(playerFactionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add ranged squad to player faction: %w", err)
		}

		// Create ActionStateData for squad
		if err := createActionStateForSquad(manager, squadID); err != nil {
			return fmt.Errorf("failed to create action state for player ranged squad: %w", err)
		}
	}

	// 7. Create Player 2 squads positioned on opposite side (5 squads total)
	// Player 2 squads are positioned on the opposite side of the map for hot-seat multiplayer
	// Squad positions relative to player start:
	// - Squad 1: (-5, +8) from player (southwest)
	// - Squad 2: (+5, +8) from player (southeast)
	// - Squad 3: (0, +8) from player (south center)
	// - Squad 4: (-8, +5) from player (west)
	// - Squad 5: (+8, +5) from player (east)
	player2SquadPositions := []coords.LogicalPosition{
		{X: clampPosition(playerStartPos.X-5, 0, 99), Y: clampPosition(playerStartPos.Y+8, 0, 79)},
		{X: clampPosition(playerStartPos.X+5, 0, 99), Y: clampPosition(playerStartPos.Y+8, 0, 79)},
		{X: clampPosition(playerStartPos.X, 0, 99), Y: clampPosition(playerStartPos.Y+8, 0, 79)},
		{X: clampPosition(playerStartPos.X-8, 0, 99), Y: clampPosition(playerStartPos.Y+5, 0, 79)},
		{X: clampPosition(playerStartPos.X+8, 0, 99), Y: clampPosition(playerStartPos.Y+5, 0, 79)},
	}

	for i, pos := range player2SquadPositions {
		squadID, err := createSquadForFaction(manager, fmt.Sprintf("Player 2 Squad %d", i+1), pos)
		if err != nil {
			return fmt.Errorf("failed to create Player 2 squad %d: %w", i+1, err)
		}

		// Add squad to faction
		if err := fm.AddSquadToFaction(player2FactionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add squad to Player 2 faction: %w", err)
		}

		// Create ActionStateData for squad
		if err := createActionStateForSquad(manager, squadID); err != nil {
			return fmt.Errorf("failed to create action state for Player 2 squad: %w", err)
		}
	}

	// 7a. Add two ranged-only squads for Player 2
	player2RangedSquadPositions := []coords.LogicalPosition{
		{X: clampPosition(playerStartPos.X-7, 0, 99), Y: clampPosition(playerStartPos.Y+12, 0, 79)},
		{X: clampPosition(playerStartPos.X+7, 0, 99), Y: clampPosition(playerStartPos.Y+12, 0, 79)},
	}

	for i, pos := range player2RangedSquadPositions {
		squadID, err := createRangedSquadByAttackRange(manager, fmt.Sprintf("Player 2 Ranged Squad %d", i+1), pos)
		if err != nil {
			return fmt.Errorf("failed to create Player 2 ranged squad %d: %w", i+1, err)
		}

		// Add squad to faction
		if err := fm.AddSquadToFaction(player2FactionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add ranged squad to Player 2 faction: %w", err)
		}

		// Create ActionStateData for squad
		if err := createActionStateForSquad(manager, squadID); err != nil {
			return fmt.Errorf("failed to create action state for Player 2 ranged squad: %w", err)
		}
	}

	// 8. Create AI Goblin Horde squads positioned on a different side (4 squads total)
	// AI squads are positioned to create a three-way battle scenario
	// Squad positions relative to player start:
	// - Squad 1: (-10, -5) from player (northwest)
	// - Squad 2: (-10, +5) from player (west)
	// - Squad 3: (+10, -5) from player (northeast)
	// - Squad 4: (+10, +5) from player (east)
	aiSquadPositions := []coords.LogicalPosition{
		{X: clampPosition(playerStartPos.X-10, 0, 99), Y: clampPosition(playerStartPos.Y-5, 0, 79)},
		{X: clampPosition(playerStartPos.X-10, 0, 99), Y: clampPosition(playerStartPos.Y+5, 0, 79)},
		{X: clampPosition(playerStartPos.X+10, 0, 99), Y: clampPosition(playerStartPos.Y-5, 0, 79)},
		{X: clampPosition(playerStartPos.X+10, 0, 99), Y: clampPosition(playerStartPos.Y+5, 0, 79)},
	}

	for i, pos := range aiSquadPositions {
		squadID, err := createSquadForFaction(manager, fmt.Sprintf("Goblin Squad %d", i+1), pos)
		if err != nil {
			return fmt.Errorf("failed to create AI squad %d: %w", i+1, err)
		}

		// Add squad to AI faction
		if err := fm.AddSquadToFaction(aiFactionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add squad to AI faction: %w", err)
		}

		// Create ActionStateData for squad
		if err := createActionStateForSquad(manager, squadID); err != nil {
			return fmt.Errorf("failed to create action state for AI squad: %w", err)
		}
	}

	// 8a. Add two ranged-only squads for AI Faction
	aiRangedSquadPositions := []coords.LogicalPosition{
		{X: clampPosition(playerStartPos.X-12, 0, 99), Y: clampPosition(playerStartPos.Y, 0, 79)},
		{X: clampPosition(playerStartPos.X+12, 0, 99), Y: clampPosition(playerStartPos.Y, 0, 79)},
	}

	for i, pos := range aiRangedSquadPositions {
		squadID, err := createRangedSquadByAttackRange(manager, fmt.Sprintf("Goblin Ranged Squad %d", i+1), pos)
		if err != nil {
			return fmt.Errorf("failed to create AI ranged squad %d: %w", i+1, err)
		}

		// Add squad to AI faction
		if err := fm.AddSquadToFaction(aiFactionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add ranged squad to AI faction: %w", err)
		}

		// Create ActionStateData for squad
		if err := createActionStateForSquad(manager, squadID); err != nil {
			return fmt.Errorf("failed to create action state for AI ranged squad: %w", err)
		}
	}

	fmt.Printf("Created three-faction battle:\n")
	fmt.Printf("  Player 1 faction (%d) with %d squads\n", playerFactionID, len(playerSquadPositions)+len(playerRangedSquadPositions))
	fmt.Printf("  Player 2 faction (%d) with %d squads\n", player2FactionID, len(player2SquadPositions)+len(player2RangedSquadPositions))
	fmt.Printf("  AI Goblin Horde (%d) with %d squads\n", aiFactionID, len(aiSquadPositions)+len(aiRangedSquadPositions))

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

// createRangedSquadByAttackRange creates a squad with only ranged units (AttackRange >= 3)
// Uses the AttackRange field to select units
func createRangedSquadByAttackRange(
	manager *common.EntityManager,
	squadName string,
	position coords.LogicalPosition,
) (ecs.EntityID, error) {
	// Get all units with attack range >= 3 (ranged units)
	rangedUnits := filterUnitsByAttackRange(3)
	if len(rangedUnits) == 0 {
		return 0, fmt.Errorf("no ranged units available (AttackRange >= 3)")
	}

	// Create squad with 3-5 ranged units
	unitCount := common.GetRandomBetween(3, 5)
	unitsToCreate := []squads.UnitTemplate{}

	// Spread units across all three rows for better squad positioning
	gridPositions := [][2]int{
		{0, 0}, // Row 0 - Front left
		{1, 1}, // Row 1 - Middle center
		{2, 2}, // Row 2 - Back right
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

	return squadID, nil
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

// SetupMultiplayerFactions creates N player factions for hot-seat multiplayer
// Each player gets their own faction with 3 squads positioned spread across the map
// playerCount must be between 2 and 4
func SetupMultiplayerFactions(manager *common.EntityManager, playerStartPos coords.LogicalPosition, playerCount int) error {
	fm := NewFactionManager(manager)

	if playerCount < 2 || playerCount > 4 {
		return fmt.Errorf("player count must be between 2 and 4, got %d", playerCount)
	}

	// Check if we have units available
	if len(squads.Units) == 0 {
		return fmt.Errorf("no units available - call squads.InitUnitTemplatesFromJSON() first")
	}

	// Create player factions
	for i := 0; i < playerCount; i++ {
		playerID := i + 1
		playerName := fmt.Sprintf("Player %d", playerID)
		factionName := fmt.Sprintf("%s's Army", playerName)

		factionID := fm.CreateFactionWithPlayer(factionName, playerID, playerName)

		// Create 3 squads per player faction (similar to existing setup)
		// Spread factions across the map horizontally using i*10 offset
		playerSquadPositions := []coords.LogicalPosition{
			{X: clampPosition(playerStartPos.X-3+(i*10), 0, 99), Y: clampPosition(playerStartPos.Y-3, 0, 79)},
			{X: clampPosition(playerStartPos.X+3+(i*10), 0, 99), Y: clampPosition(playerStartPos.Y-3, 0, 79)},
			{X: clampPosition(playerStartPos.X+(i*10), 0, 99), Y: clampPosition(playerStartPos.Y+3, 0, 79)},
		}

		for j, pos := range playerSquadPositions {
			squadID, err := createSquadForFaction(manager, fmt.Sprintf("%s Squad %d", playerName, j+1), pos)
			if err != nil {
				return fmt.Errorf("failed to create squad for %s: %w", playerName, err)
			}

			if err := fm.AddSquadToFaction(factionID, squadID, pos); err != nil {
				return fmt.Errorf("failed to add squad to %s faction: %w", playerName, err)
			}

			if err := createActionStateForSquad(manager, squadID); err != nil {
				return fmt.Errorf("failed to create action state: %w", err)
			}
		}
	}

	fmt.Printf("Created %d player factions for hot-seat multiplayer\n", playerCount)
	for i := 0; i < playerCount; i++ {
		fmt.Printf("  Player %d: 3 squads\n", i+1)
	}

	return nil
}
