package combatlifecycle

import (
	"fmt"

	"game_main/core/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/squads/squadcore"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// enrollSquadInFaction performs the 3-step enrollment of a squad into a combat faction:
// 1. AddSquadToFaction (faction membership + position)
// 2. EnsureUnitPositions (all units get positions at squad location)
// 3. CreateActionStateForSquad (combat action tracking)
//
// Internal helper for EnrollSquadsAtPositions.
func enrollSquadInFaction(
	fm *combatstate.CombatFactionManager,
	manager *common.EntityManager,
	factionID, squadID ecs.EntityID,
	pos coords.LogicalPosition,
) error {
	if err := fm.AddSquadToFaction(factionID, squadID, pos); err != nil {
		return err
	}

	EnsureUnitPositions(manager, squadID, pos)
	combatstate.CreateActionStateForSquad(manager, squadID)

	return nil
}

// CreateFactionPair creates a CombatQueryCache, CombatFactionManager, and two standard factions.
// This 3-line sequence is used by all combat setup paths (overworld, garrison, raid).
func CreateFactionPair(
	manager *common.EntityManager,
	playerName, enemyName string,
	encounterID ecs.EntityID,
) (*combatstate.CombatFactionManager, ecs.EntityID, ecs.EntityID) {
	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	playerFactionID, enemyFactionID := fm.CreateStandardFactions(playerName, enemyName, encounterID)
	return fm, playerFactionID, enemyFactionID
}

// EnrollSquadsAtPositions enrolls multiple squads into a faction at given positions.
// Positions and squadIDs must be the same length.
// Deployment status (SquadData.IsDeployed) is the caller's policy — use
// MarkSquadsDeployed separately if the squads should be marked deployed.
func EnrollSquadsAtPositions(
	fm *combatstate.CombatFactionManager,
	manager *common.EntityManager,
	factionID ecs.EntityID,
	squadIDs []ecs.EntityID,
	positions []coords.LogicalPosition,
) error {
	if len(squadIDs) != len(positions) {
		return fmt.Errorf("squad count (%d) != position count (%d)", len(squadIDs), len(positions))
	}
	for i, squadID := range squadIDs {
		if err := enrollSquadInFaction(fm, manager, factionID, squadID, positions[i]); err != nil {
			return fmt.Errorf("failed to enroll squad %d: %w", squadID, err)
		}
	}
	return nil
}

// MarkSquadsDeployed sets SquadData.IsDeployed = true on each squad.
// Use after EnrollSquadsAtPositions for squads that weren't already deployed
// (garrison defenders, raid player squads). Squads already marked deployed
// (e.g. overworld player rosters) do not need this call.
func MarkSquadsDeployed(manager *common.EntityManager, squadIDs []ecs.EntityID) {
	for _, squadID := range squadIDs {
		squadData := common.GetComponentTypeByID[*squadcore.SquadData](manager, squadID, squadcore.SquadComponent)
		if squadData != nil {
			squadData.IsDeployed = true
		}
	}
}

// EnsureUnitPositions ensures all units in a squad have position components.
// Units that already have positions are moved to the squad position.
// Units without positions get a new position component created.
func EnsureUnitPositions(manager *common.EntityManager, squadID ecs.EntityID, squadPos coords.LogicalPosition) {
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
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
