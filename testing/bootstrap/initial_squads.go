package bootstrap

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CreateInitialPlayerSquads creates starting squads for a roster owner at game launch.
// Creates 10 diverse squads in reserves (not deployed on map).
// All squads are marked with IsDeployed = false and added to the owner's SquadRoster.
// rosterOwnerID is the entity that holds the SquadRosterComponent (commander or player).
// unitRosterOwnerID is the entity that holds the UnitRosterComponent (always the player).
// prefix is prepended to squad names (e.g. "Vanguard" -> "Vanguard Balanced 1").
func CreateInitialPlayerSquads(rosterOwnerID ecs.EntityID, unitRosterOwnerID ecs.EntityID, manager *common.EntityManager, prefix ...string) error {
	// 1. Verify unit templates are loaded
	if len(squads.Units) == 0 {
		return fmt.Errorf("no unit templates available - call InitUnitTemplatesFromJSON() first")
	}

	// 2. Get squad roster from owner entity
	roster := squads.GetPlayerSquadRoster(rosterOwnerID, manager)
	if roster == nil {
		return fmt.Errorf("entity %d has no squad roster component", rosterOwnerID)
	}

	// 3. Get unit roster for tracking unit ownership (from player entity)
	unitRoster := squads.GetPlayerRoster(unitRosterOwnerID, manager)
	if unitRoster == nil {
		return fmt.Errorf("entity %d has no unit roster component", unitRosterOwnerID)
	}

	// Determine name prefix
	namePrefix := "Commander"
	if len(prefix) > 0 && prefix[0] != "" {
		namePrefix = prefix[0]
	}

	// 4. Create 10 diverse squads
	squadConfigs := []struct {
		name     string
		createFn func(*common.EntityManager, string) (ecs.EntityID, error)
	}{
		{fmt.Sprintf("%s Balanced 1", namePrefix), createBalancedSquad},
		{fmt.Sprintf("%s Ranged 1", namePrefix), createRangedSquad},
		{fmt.Sprintf("%s Magic 1", namePrefix), createMagicSquad},
		{fmt.Sprintf("%s Balanced 2", namePrefix), createBalancedSquad},
		{fmt.Sprintf("%s Mixed 1", namePrefix), createMixedSquad},
		{fmt.Sprintf("%s Balanced 3", namePrefix), createBalancedSquad},
		{fmt.Sprintf("%s Ranged 2", namePrefix), createRangedSquad},
		{fmt.Sprintf("%s Magic 2", namePrefix), createMagicSquad},
		{fmt.Sprintf("%s Mixed 2", namePrefix), createMixedSquad},
		{fmt.Sprintf("%s Balanced 4", namePrefix), createBalancedSquad},
		{fmt.Sprintf("%s Cavalry 1", namePrefix), createCavalrySquad},
		{fmt.Sprintf("%s Cavalry 2", namePrefix), createCavalrySquad},
	}

	for _, config := range squadConfigs {
		// Create squad
		squadID, err := config.createFn(manager, config.name)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", config.name, err)
		}

		// Mark squad as not deployed (in reserves)
		squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
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
		squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
		if squadData != nil {
			fmt.Printf("  - Squad '%s': IsDeployed=%v, Units=%d\n",
				squadData.Name,
				squadData.IsDeployed,
				len(squads.GetUnitIDsInSquad(squadID, manager)))
		}
	}
	fmt.Printf("=====================================\n\n")

	return nil
}

// createBalancedSquad creates a balanced squad with mixed unit types
func createBalancedSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	unitsToCreate := []squads.UnitTemplate{}

	// Create 5 units with balanced formation
	positions := [][2]int{
		{0, 0}, // Front left
		{0, 1}, // Front center
		{0, 2}, // Front right
		{1, 1}, // Middle center
		{2, 1}, // Back center
	}

	maxUnits := 5
	if len(squads.Units) < maxUnits {
		maxUnits = len(squads.Units)
	}

	// Randomly select leader
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
			unit.Attributes.Leadership = 20
		}

		unitsToCreate = append(unitsToCreate, unit)
	}

	// Create squad at position (0,0) - not deployed
	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationBalanced,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)

	return squadID, nil
}

// createRangedSquad creates a squad with only ranged units
func createRangedSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	// Filter ranged units (AttackRange >= 3)
	rangedUnits := squads.FilterByAttackRange(3)
	if len(rangedUnits) == 0 {
		return 0, fmt.Errorf("no ranged units available (AttackRange >= 3)")
	}

	// Create squad with 3-5 ranged units
	unitCount := common.GetRandomBetween(3, 5)
	unitsToCreate := []squads.UnitTemplate{}

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
	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationRanged,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)

	return squadID, nil
}

// createMagicSquad creates a squad with magic units
func createMagicSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	// Filter magic units (AttackType == Magic)
	magicUnits := squads.FilterByAttackType(squads.AttackTypeMagic)
	if len(magicUnits) == 0 {
		return 0, fmt.Errorf("no magic units available (AttackType == Magic)")
	}

	// Create squad with exactly 3 magic units
	unitCount := 3
	unitsToCreate := []squads.UnitTemplate{}

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
	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationBalanced,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)

	return squadID, nil
}

// createMixedSquad creates a squad with mixed ranged and magic units
func createMixedSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	rangedUnits := squads.FilterByAttackRange(3)
	magicUnits := squads.FilterByAttackType(squads.AttackTypeMagic)

	if len(rangedUnits) == 0 || len(magicUnits) == 0 {
		// Fallback to balanced squad if we don't have both types
		return createBalancedSquad(manager, squadName)
	}

	// Create squad with 4-5 units (mix of ranged and magic)
	unitCount := common.GetRandomBetween(4, 5)
	unitsToCreate := []squads.UnitTemplate{}

	gridPositions := [][2]int{
		{0, 0}, // Row 0 - Front left
		{1, 1}, // Row 1 - Middle center
		{2, 2}, // Row 2 - Back right
		{1, 2}, // Row 1 - Middle right
		{2, 0}, // Row 2 - Back left
	}

	for i := 0; i < unitCount; i++ {
		var unit squads.UnitTemplate

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
	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationBalanced,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)

	return squadID, nil
}

// createCavalrySquad creates a squad with fast cavalry units (movementSpeed >= 6)
func createCavalrySquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	// Filter cavalry units by high movement speed
	cavalryUnits := squads.FilterByMinMovementSpeed(6)
	if len(cavalryUnits) == 0 {
		return 0, fmt.Errorf("no cavalry units available (MovementSpeed >= 6)")
	}

	// Create squad with 4-5 cavalry units
	unitCount := common.GetRandomBetween(4, 5)
	unitsToCreate := []squads.UnitTemplate{}

	gridPositions := [][2]int{
		{0, 0}, // Row 0 - Front left
		{0, 2}, // Row 0 - Front right
		{1, 1}, // Row 1 - Middle center
		{1, 0}, // Row 1 - Middle left
		{2, 1}, // Row 2 - Back center
	}

	for i := 0; i < unitCount; i++ {
		randomIdx := common.RandomInt(len(cavalryUnits))
		unit := cavalryUnits[randomIdx]

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
	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationOffensive,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)

	return squadID, nil
}

// registerSquadUnitsInRoster registers all units in a squad with the player's unit roster
func registerSquadUnitsInRoster(squadID ecs.EntityID, roster *squads.UnitRoster, manager *common.EntityManager) error {
	// Get all unit IDs in the squad
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)

	for _, unitID := range unitIDs {
		// Register unit in roster
		if err := squads.RegisterSquadUnitInRoster(roster, unitID, squadID, manager); err != nil {
			return fmt.Errorf("failed to register unit %d: %w", unitID, err)
		}
	}

	return nil
}
