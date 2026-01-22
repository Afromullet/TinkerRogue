package squads

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CreateInitialPlayerSquads creates starting squads for the player at game launch.
// Creates 10 diverse squads in player reserves (not deployed on map).
// All squads are marked with IsDeployed = false and added to player's SquadRoster.
func CreateInitialPlayerSquads(playerID ecs.EntityID, manager *common.EntityManager) error {
	// 1. Verify unit templates are loaded
	if len(Units) == 0 {
		return fmt.Errorf("no unit templates available - call InitUnitTemplatesFromJSON() first")
	}

	// 2. Get player's squad roster
	roster := GetPlayerSquadRoster(playerID, manager)
	if roster == nil {
		return fmt.Errorf("player has no squad roster component")
	}

	// 3. Get player's unit roster for tracking unit ownership
	unitRoster := GetPlayerUnitRoster(playerID, manager)
	if unitRoster == nil {
		return fmt.Errorf("player has no unit roster component")
	}

	// 4. Create 10 diverse squads
	squadConfigs := []struct {
		name      string
		createFn  func(*common.EntityManager, string) (ecs.EntityID, error)
	}{
		{"Player Balanced Squad 1", createBalancedSquad},
		{"Player Ranged Squad", createRangedSquad},
		{"Player Magic Squad", createMagicSquad},
		{"Player Balanced Squad 2", createBalancedSquad},
		{"Player Mixed Squad", createMixedSquad},
		{"Player Balanced Squad 3", createBalancedSquad},
		{"Player Ranged Squad 2", createRangedSquad},
		{"Player Magic Squad 2", createMagicSquad},
		{"Player Mixed Squad 2", createMixedSquad},
		{"Player Balanced Squad 4", createBalancedSquad},
	}

	for _, config := range squadConfigs {
		// Create squad
		squadID, err := config.createFn(manager, config.name)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", config.name, err)
		}

		// Mark squad as not deployed (in reserves)
		squadData := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)
		if squadData == nil {
			return fmt.Errorf("failed to get squad data for %s", config.name)
		}
		squadData.IsDeployed = false

		// Add to player's squad roster
		if err := roster.AddSquad(squadID); err != nil {
			return fmt.Errorf("failed to add %s to roster: %w", config.name, err)
		}

		// Register all units in player's unit roster
		if err := registerSquadUnitsInRoster(squadID, unitRoster, manager); err != nil {
			return fmt.Errorf("failed to register units for %s: %w", config.name, err)
		}
	}

	// Verification: Log squad creation details
	fmt.Printf("\n=== Initial Player Squads Created ===\n")
	fmt.Printf("Total squads: %d\n", len(squadConfigs))
	for _, squadID := range roster.OwnedSquads {
		squadData := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)
		if squadData != nil {
			fmt.Printf("  - Squad '%s': IsDeployed=%v, Units=%d\n",
				squadData.Name,
				squadData.IsDeployed,
				len(GetUnitIDsInSquad(squadID, manager)))
		}
	}
	fmt.Printf("=====================================\n\n")

	return nil
}

// createBalancedSquad creates a balanced squad with mixed unit types
// Pattern from gameplayfactions.go createSquadForFaction
func createBalancedSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	unitsToCreate := []UnitTemplate{}

	// Create 5 units with balanced formation
	positions := [][2]int{
		{0, 0}, // Front left
		{0, 1}, // Front center
		{0, 2}, // Front right
		{1, 1}, // Middle center
		{2, 1}, // Back center
	}

	maxUnits := 5
	if len(Units) < maxUnits {
		maxUnits = len(Units)
	}

	// Randomly select leader
	leaderIndex := common.RandomInt(maxUnits)

	for i := 0; i < maxUnits && i < len(positions); i++ {
		// Create a copy of the unit template
		unit := Units[i%len(Units)]

		// Set grid position
		unit.GridRow = positions[i][0]
		unit.GridCol = positions[i][1]

		// Make the randomly selected unit the leader
		if i == leaderIndex {
			unit.IsLeader = true
			unit.Attributes.Leadership = 20
		}

		unitsToCreate = append(unitsToCreate, unit)
	}

	// Create squad at position (0,0) - not deployed
	squadID := CreateSquadFromTemplate(
		manager,
		squadName,
		FormationBalanced,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)

	return squadID, nil
}

// createRangedSquad creates a squad with only ranged units
// Pattern from gameplayfactions.go createRangedSquadByAttackRange
func createRangedSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	// Filter ranged units (AttackRange >= 3)
	rangedUnits := filterUnitsByAttackRange(3)
	if len(rangedUnits) == 0 {
		return 0, fmt.Errorf("no ranged units available (AttackRange >= 3)")
	}

	// Create squad with 3-5 ranged units
	unitCount := common.GetRandomBetween(3, 5)
	unitsToCreate := []UnitTemplate{}

	// Spread units across rows
	gridPositions := [][2]int{
		{0, 0}, // Row 0 - Front left
		{1, 1}, // Row 1 - Middle center
		{2, 2}, // Row 2 - Back right
		{0, 2}, // Row 0 - Front right
		{1, 0}, // Row 1 - Middle left
	}

	for i := 0; i < unitCount; i++ {
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

	// Create squad at position (0,0) - not deployed
	squadID := CreateSquadFromTemplate(
		manager,
		squadName,
		FormationRanged,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)

	return squadID, nil
}

// createMagicSquad creates a squad with magic units
// Pattern from gameplayfactions.go createMagicSquadByAttackType
func createMagicSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	// Filter magic units (AttackType == Magic)
	magicUnits := filterUnitsByAttackType(AttackTypeMagic)
	if len(magicUnits) == 0 {
		return 0, fmt.Errorf("no magic units available (AttackType == Magic)")
	}

	// Create squad with exactly 3 magic units
	unitCount := 3
	unitsToCreate := []UnitTemplate{}

	gridPositions := [][2]int{
		{0, 1}, // Row 0 - Front center
		{1, 0}, // Row 1 - Middle left
		{2, 1}, // Row 2 - Back center
	}

	for i := 0; i < unitCount; i++ {
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

	// Create squad at position (0,0) - not deployed
	squadID := CreateSquadFromTemplate(
		manager,
		squadName,
		FormationBalanced,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)

	return squadID, nil
}

// createMixedSquad creates a squad with mixed ranged and magic units
// Pattern from gameplayfactions.go createMixedRangedMagicSquad
func createMixedSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	rangedUnits := filterUnitsByAttackRange(3)
	magicUnits := filterUnitsByAttackType(AttackTypeMagic)

	if len(rangedUnits) == 0 || len(magicUnits) == 0 {
		// Fallback to balanced squad if we don't have both types
		return createBalancedSquad(manager, squadName)
	}

	// Create squad with 4-5 units (mix of ranged and magic)
	unitCount := common.GetRandomBetween(4, 5)
	unitsToCreate := []UnitTemplate{}

	gridPositions := [][2]int{
		{0, 0}, // Row 0 - Front left
		{1, 1}, // Row 1 - Middle center
		{2, 2}, // Row 2 - Back right
		{1, 2}, // Row 1 - Middle right
		{2, 0}, // Row 2 - Back left
	}

	for i := 0; i < unitCount; i++ {
		var unit UnitTemplate

		// Alternate between ranged and magic
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

	// Create squad at position (0,0) - not deployed
	squadID := CreateSquadFromTemplate(
		manager,
		squadName,
		FormationBalanced,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)

	return squadID, nil
}

// filterUnitsByAttackRange returns units matching the specified attack range
func filterUnitsByAttackRange(minRange int) []UnitTemplate {
	var filtered []UnitTemplate
	for _, unit := range Units {
		if unit.AttackRange >= minRange {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}

// filterUnitsByAttackType returns units matching the specified attack type
func filterUnitsByAttackType(attackType AttackType) []UnitTemplate {
	var filtered []UnitTemplate
	for _, unit := range Units {
		if unit.AttackType == attackType {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}

// registerSquadUnitsInRoster registers all units in a squad with the player's unit roster
func registerSquadUnitsInRoster(squadID ecs.EntityID, roster *UnitRoster, manager *common.EntityManager) error {
	// Get all unit IDs in the squad
	unitIDs := GetUnitIDsInSquad(squadID, manager)

	for _, unitID := range unitIDs {
		// Register unit in roster
		if err := RegisterSquadUnitInRoster(roster, unitID, squadID, manager); err != nil {
			return fmt.Errorf("failed to register unit %d: %w", unitID, err)
		}
	}

	return nil
}

// GetPlayerUnitRoster retrieves player's unit roster from ECS
// Returns nil if player has no roster component
func GetPlayerUnitRoster(playerID ecs.EntityID, manager *common.EntityManager) *UnitRoster {
	return common.GetComponentTypeByID[*UnitRoster](manager, playerID, UnitRosterComponent)
}
