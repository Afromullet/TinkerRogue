package combat

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

//TODO: Remove this in the future. Just here for testing

// SetupGameplayFactions creates two factions (player and AI) with squads for gameplay testing.
// This is called during game initialization to create the initial combat setup.
// Each faction gets 3 squads positioned on the map.
func SetupGameplayFactions(manager *common.EntityManager, playerStartPos coords.LogicalPosition) error {
	// 1. Initialize combat components if not already done
	if FactionComponent == nil {
		InitializeCombatSystem(manager)
	}

	// 2. Create FactionManager
	fm := NewFactionManager(manager)

	// 3. Create Player Faction
	playerFactionID := fm.CreateFaction("Player Alliance", true)

	// 4. Create AI Faction
	aiFactionID := fm.CreateFaction("Goblin Horde", false)

	// 5. Check if we have units available
	if len(squads.Units) == 0 {
		return fmt.Errorf("no units available - call squads.InitUnitTemplatesFromJSON() first")
	}

	// 6. Create player squads positioned above player (north side)
	// Squad positions relative to player start:
	// - Squad 1: (-3, -3) from player (northwest)
	// - Squad 2: (+3, -3) from player (northeast)
	// - Squad 3: (0, +3) from player (south - slightly behind)
	playerSquadPositions := []coords.LogicalPosition{
		{X: clampPosition(playerStartPos.X - 3, 0, 99), Y: clampPosition(playerStartPos.Y - 3, 0, 79)},
		{X: clampPosition(playerStartPos.X + 3, 0, 99), Y: clampPosition(playerStartPos.Y - 3, 0, 79)},
		{X: clampPosition(playerStartPos.X, 0, 99), Y: clampPosition(playerStartPos.Y + 3, 0, 79)},
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

	// 7. Create AI squads positioned below player (south side)
	// AI squads are mirrored across the player, creating engagement distance
	// Squad positions relative to player start:
	// - Squad 1: (-3, +3) from player (southwest)
	// - Squad 2: (+3, +3) from player (southeast)
	// - Squad 3: (0, -3) from player (north - slightly ahead)
	aiSquadPositions := []coords.LogicalPosition{
		{X: clampPosition(playerStartPos.X - 3, 0, 99), Y: clampPosition(playerStartPos.Y + 3, 0, 79)},
		{X: clampPosition(playerStartPos.X + 3, 0, 99), Y: clampPosition(playerStartPos.Y + 3, 0, 79)},
		{X: clampPosition(playerStartPos.X, 0, 99), Y: clampPosition(playerStartPos.Y - 3, 0, 79)},
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

	fmt.Printf("Created gameplay factions:\n")
	fmt.Printf("  Player faction (%d) with %d squads\n", playerFactionID, len(playerSquadPositions))
	fmt.Printf("  AI faction (%d) with %d squads\n", aiFactionID, len(aiSquadPositions))

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

	for i := 0; i < maxUnits && i < len(positions); i++ {
		// Create a copy of the unit template
		unit := squads.Units[i%len(squads.Units)]

		// Set grid position
		unit.GridRow = positions[i][0]
		unit.GridCol = positions[i][1]

		// Make the first unit the leader
		if i == 0 {
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
