package gamesetup

import (
	"fmt"
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/tactical/powers/spells"
	rstr "game_main/tactical/squads/roster"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// squadCreatorFn is the signature for the per-type squad factory funcs below.
type squadCreatorFn func(*common.EntityManager, string) (ecs.EntityID, error)

// squadCreators maps the JSON squad type ID to its creator. Keep keys in sync
// with templates.validSquadTypeIDs.
var squadCreators = map[string]squadCreatorFn{
	"balanced": createBalancedSquad,
	"ranged":   createRangedSquad,
	"magic":    createMagicSquad,
	"mixed":    createMixedSquad,
	"cavalry":  createCavalrySquad,
}

// CreateSquadsForCommander creates cfg.Count starting squads for the given roster
// owner, randomly picking each squad's type from cfg.TypePool. All squads start in
// reserves (IsDeployed = false). rosterOwnerID is the entity that holds the
// SquadRoster (commander or player). unitRosterOwnerID always holds the player's
// UnitRoster.
func CreateSquadsForCommander(rosterOwnerID, unitRosterOwnerID ecs.EntityID, manager *common.EntityManager, cfg templates.JSONSquadSetup) error {
	if cfg.Count <= 0 {
		return nil
	}
	if len(unitdefs.Units) == 0 {
		return fmt.Errorf("no unit templates available - call InitUnitTemplatesFromJSON() first")
	}

	roster := rstr.GetPlayerSquadRoster(rosterOwnerID, manager)
	if roster == nil {
		return fmt.Errorf("entity %d has no squad roster component", rosterOwnerID)
	}
	unitRoster := rstr.GetPlayerRoster(unitRosterOwnerID, manager)
	if unitRoster == nil {
		return fmt.Errorf("entity %d has no unit roster component", unitRosterOwnerID)
	}

	namePrefix := cfg.NamePrefix
	if namePrefix == "" {
		namePrefix = "Commander"
	}

	for i := 0; i < cfg.Count; i++ {
		typeID := cfg.TypePool[common.RandomInt(len(cfg.TypePool))]
		createFn, ok := squadCreators[typeID]
		if !ok {
			return fmt.Errorf("unknown squad type %q (validation should have caught this)", typeID)
		}

		squadName := fmt.Sprintf("%s Squad %d", namePrefix, i+1)
		squadID, err := createFn(manager, squadName)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", squadName, err)
		}

		squadData := common.GetComponentTypeByID[*squadcore.SquadData](manager, squadID, squadcore.SquadComponent)
		if squadData == nil {
			return fmt.Errorf("failed to get squad data for %s", squadName)
		}
		squadData.IsDeployed = false

		// Add to owner's squad roster FIRST so FindCommanderForSquad can resolve
		// the commander inside InitSquadSpellsFromLeader below.
		if err := roster.AddSquad(squadID); err != nil {
			return fmt.Errorf("failed to add %s to roster: %w", squadName, err)
		}

		// Attach spell capability now that the squad→commander link exists, so
		// spell filtering uses the commander's progression library.
		spells.InitSquadSpellsFromLeader(squadID, manager)

		if err := registerSquadUnitsInRoster(squadID, unitRoster, manager); err != nil {
			return fmt.Errorf("failed to register units for %s: %w", squadName, err)
		}
	}

	fmt.Printf("\n=== Initial Squads Created for %s ===\n", namePrefix)
	fmt.Printf("Total squads: %d\n", cfg.Count)
	for _, squadID := range roster.OwnedSquads {
		squadData := common.GetComponentTypeByID[*squadcore.SquadData](manager, squadID, squadcore.SquadComponent)
		if squadData != nil {
			fmt.Printf("  - Squad '%s': IsDeployed=%v, Units=%d\n",
				squadData.Name,
				squadData.IsDeployed,
				len(squadcore.GetUnitIDsInSquad(squadID, manager)))
		}
	}
	fmt.Printf("=====================================\n\n")

	return nil
}

// CreateInitialRosterUnits creates standalone units (not in any squad) and adds
// them to the player's UnitRoster. Count is read from initialsetup.json.
func CreateInitialRosterUnits(unitRosterOwnerID ecs.EntityID, manager *common.EntityManager) error {
	count := templates.InitialSetupTemplate.RosterUnits.Count
	if count <= 0 {
		return nil
	}
	if len(unitdefs.Units) == 0 {
		return fmt.Errorf("no unit templates available - call InitUnitTemplatesFromJSON() first")
	}

	roster := rstr.GetPlayerRoster(unitRosterOwnerID, manager)
	if roster == nil {
		return fmt.Errorf("entity %d has no unit roster component", unitRosterOwnerID)
	}

	for i := 0; i < count; i++ {
		template := unitdefs.Units[common.RandomInt(len(unitdefs.Units))]
		unitEntity, err := squadcore.CreateUnitEntity(manager, template)
		if err != nil {
			return fmt.Errorf("failed to create roster unit %d: %w", i, err)
		}
		unitID := unitEntity.GetID()
		if err := roster.AddUnit(unitID, template.UnitType); err != nil {
			return fmt.Errorf("failed to add unit %d to roster: %w", unitID, err)
		}
	}

	fmt.Printf("Created %d initial roster units for player (entity %d)\n", count, unitRosterOwnerID)
	return nil
}

func registerSquadUnitsInRoster(squadID ecs.EntityID, roster *rstr.UnitRoster, manager *common.EntityManager) error {
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
	for _, unitID := range unitIDs {
		if err := rstr.RegisterSquadUnitInRoster(roster, unitID, squadID, manager); err != nil {
			return fmt.Errorf("failed to register unit %d: %w", unitID, err)
		}
	}
	return nil
}

// createBalancedSquad creates a balanced squad with mixed unit types
func createBalancedSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	unitsToCreate := []unitdefs.UnitTemplate{}

	positions := [][2]int{
		{0, 0}, // Front left
		{0, 1}, // Front center
		{0, 2}, // Front right
		{1, 1}, // Middle center
		{2, 1}, // Back center
	}

	maxUnits := 5
	if len(unitdefs.Units) < maxUnits {
		maxUnits = len(unitdefs.Units)
	}

	leaderIndex := common.RandomInt(maxUnits)

	for i := 0; i < maxUnits && i < len(positions); i++ {
		unit := unitdefs.Units[i%len(unitdefs.Units)]
		unit.GridRow = positions[i][0]
		unit.GridCol = positions[i][1]
		if i == leaderIndex {
			unit.IsLeader = true
			unit.Attributes.Leadership = 20
		}
		unitsToCreate = append(unitsToCreate, unit)
	}

	squadID := squadcore.CreateSquadFromTemplate(
		manager,
		squadName,
		squadcore.FormationBalanced,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)
	return squadID, nil
}

func createRangedSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	rangedUnits := unitdefs.FilterByAttackRange(3)
	if len(rangedUnits) == 0 {
		return 0, fmt.Errorf("no ranged units available (AttackRange >= 3)")
	}

	unitCount := common.GetRandomBetween(3, 5)
	unitsToCreate := []unitdefs.UnitTemplate{}

	gridPositions := [][2]int{
		{0, 0},
		{1, 1},
		{2, 2},
		{0, 2},
		{1, 0},
	}

	for i := 0; i < unitCount; i++ {
		unit := rangedUnits[common.RandomInt(len(rangedUnits))]
		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = false
		unitsToCreate = append(unitsToCreate, unit)
	}

	leaderIndex := common.RandomInt(unitCount)
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	squadID := squadcore.CreateSquadFromTemplate(
		manager,
		squadName,
		squadcore.FormationRanged,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)
	return squadID, nil
}

func createMagicSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	magicUnits := unitdefs.FilterByAttackType(unitdefs.AttackTypeMagic)
	if len(magicUnits) == 0 {
		return 0, fmt.Errorf("no magic units available (AttackType == Magic)")
	}

	unitCount := 3
	unitsToCreate := []unitdefs.UnitTemplate{}

	gridPositions := [][2]int{
		{0, 1},
		{1, 0},
		{2, 1},
	}

	for i := 0; i < unitCount; i++ {
		unit := magicUnits[common.RandomInt(len(magicUnits))]
		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = false
		unitsToCreate = append(unitsToCreate, unit)
	}

	leaderIndex := common.RandomInt(unitCount)
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	squadID := squadcore.CreateSquadFromTemplate(
		manager,
		squadName,
		squadcore.FormationBalanced,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)
	return squadID, nil
}

func createMixedSquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	rangedUnits := unitdefs.FilterByAttackRange(3)
	magicUnits := unitdefs.FilterByAttackType(unitdefs.AttackTypeMagic)

	if len(rangedUnits) == 0 || len(magicUnits) == 0 {
		return createBalancedSquad(manager, squadName)
	}

	unitCount := common.GetRandomBetween(4, 5)
	unitsToCreate := []unitdefs.UnitTemplate{}

	gridPositions := [][2]int{
		{0, 0},
		{1, 1},
		{2, 2},
		{1, 2},
		{2, 0},
	}

	for i := 0; i < unitCount; i++ {
		var unit unitdefs.UnitTemplate
		if i%2 == 0 && len(rangedUnits) > 0 {
			unit = rangedUnits[common.RandomInt(len(rangedUnits))]
		} else if len(magicUnits) > 0 {
			unit = magicUnits[common.RandomInt(len(magicUnits))]
		} else if len(rangedUnits) > 0 {
			unit = rangedUnits[common.RandomInt(len(rangedUnits))]
		}
		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = false
		unitsToCreate = append(unitsToCreate, unit)
	}

	leaderIndex := common.RandomInt(unitCount)
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	squadID := squadcore.CreateSquadFromTemplate(
		manager,
		squadName,
		squadcore.FormationBalanced,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)
	return squadID, nil
}

func createCavalrySquad(manager *common.EntityManager, squadName string) (ecs.EntityID, error) {
	cavalryUnits := unitdefs.FilterByMinMovementSpeed(6)
	if len(cavalryUnits) == 0 {
		return 0, fmt.Errorf("no cavalry units available (MovementSpeed >= 6)")
	}

	unitCount := common.GetRandomBetween(4, 5)
	unitsToCreate := []unitdefs.UnitTemplate{}

	gridPositions := [][2]int{
		{0, 0},
		{0, 2},
		{1, 1},
		{1, 0},
		{2, 1},
	}

	for i := 0; i < unitCount; i++ {
		unit := cavalryUnits[common.RandomInt(len(cavalryUnits))]
		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = false
		unitsToCreate = append(unitsToCreate, unit)
	}

	leaderIndex := common.RandomInt(unitCount)
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	squadID := squadcore.CreateSquadFromTemplate(
		manager,
		squadName,
		squadcore.FormationOffensive,
		coords.LogicalPosition{X: 0, Y: 0},
		unitsToCreate,
	)
	return squadID, nil
}
