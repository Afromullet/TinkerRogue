package combatlifecycle

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// EnrollSquadInFaction performs the full 4-step enrollment of a squad into a combat faction:
// 1. AddSquadToFaction (faction membership + position)
// 2. EnsureUnitPositions (all units get positions at squad location)
// 3. CreateActionStateForSquad (combat action tracking)
// 4. Optionally marks squad as deployed
//
// This eliminates the duplicated 4-step sequence across encounter_setup.go,
// starters.go, and raidencounter.go.
func EnrollSquadInFaction(
	fm *combat.CombatFactionManager,
	manager *common.EntityManager,
	factionID, squadID ecs.EntityID,
	pos coords.LogicalPosition,
	markDeployed bool,
) error {
	if err := fm.AddSquadToFaction(factionID, squadID, pos); err != nil {
		return err
	}

	EnsureUnitPositions(manager, squadID, pos)
	combat.CreateActionStateForSquad(manager, squadID)

	if markDeployed {
		squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
		if squadData != nil {
			squadData.IsDeployed = true
		}
	}

	return nil
}

// EnsureUnitPositions ensures all units in a squad have position components.
// Units that already have positions are moved to the squad position.
// Units without positions get a new position component created.
func EnsureUnitPositions(manager *common.EntityManager, squadID ecs.EntityID, squadPos coords.LogicalPosition) {
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	for _, unitID := range unitIDs {
		unitEntity := manager.FindEntityByID(unitID)
		if unitEntity == nil {
			continue
		}

		unitPos := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
		if unitPos != nil {
			manager.MoveEntity(unitID, unitEntity, *unitPos, squadPos)
		} else {
			manager.RegisterEntityPosition(unitEntity, squadPos)
		}
	}
}
