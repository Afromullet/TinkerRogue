package commander

import (
	"game_main/core/common"

	"github.com/bytearena/ecs"
)

// cachedOverworldTurnStateID memoizes the singleton entity ID after first lookup
// so GetOverworldTurnState becomes an O(1) component fetch instead of an O(N) tag query.
// Refreshed lazily if the cached ID becomes stale (e.g., after save/load).
var cachedOverworldTurnStateID ecs.EntityID

// CreateOverworldTurnState creates the singleton overworld turn state entity
func CreateOverworldTurnState(manager *common.EntityManager) ecs.EntityID {
	entity := manager.World.NewEntity()
	entity.AddComponent(OverworldTurnStateComponent, &OverworldTurnStateData{
		CurrentTurn: 1,
		TurnActive:  true,
	})
	cachedOverworldTurnStateID = entity.GetID()
	return cachedOverworldTurnStateID
}

// GetOverworldTurnState retrieves the singleton turn state
func GetOverworldTurnState(manager *common.EntityManager) *OverworldTurnStateData {
	if cachedOverworldTurnStateID != 0 {
		if data := common.GetComponentTypeByID[*OverworldTurnStateData](manager, cachedOverworldTurnStateID, OverworldTurnStateComponent); data != nil {
			return data
		}
		// Cached ID is stale (e.g., after save/load). Fall through to refresh.
	}
	results := manager.World.Query(OverworldTurnTag)
	if len(results) == 0 {
		return nil
	}
	cachedOverworldTurnStateID = results[0].Entity.GetID()
	return common.GetComponentType[*OverworldTurnStateData](results[0].Entity, OverworldTurnStateComponent)
}

// StartNewTurn resets all commander action states for a new turn.
// Sets MovementRemaining from each commander's Attributes.MovementSpeed.
func StartNewTurn(manager *common.EntityManager, playerID ecs.EntityID) {
	roster := GetPlayerCommanderRoster(playerID, manager)
	if roster == nil {
		return
	}

	for _, cmdID := range roster.CommanderIDs {
		actionState := GetCommanderActionState(cmdID, manager)
		if actionState == nil {
			continue
		}

		// Reset action state
		actionState.HasMoved = false
		actionState.HasActed = false

		// Set movement from commander's attributes
		cmdEntity := manager.FindEntityByID(cmdID)
		if cmdEntity == nil {
			continue
		}
		attr := common.GetComponentType[*common.Attributes](cmdEntity, common.AttributeComponent)
		if attr == nil {
			continue // Invariant: every commander has AttributeComponent. Skip if violated.
		}
		actionState.MovementRemaining = attr.GetMovementSpeed()
	}
}

// GetCommanderActionState returns the action state for a specific commander (O(1) lookup).
// Action state is stored directly on the commander entity since CreateCommander.
func GetCommanderActionState(commanderID ecs.EntityID, manager *common.EntityManager) *CommanderActionStateData {
	return common.GetComponentTypeByID[*CommanderActionStateData](manager, commanderID, CommanderActionStateComponent)
}
