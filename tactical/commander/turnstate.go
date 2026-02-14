package commander

import (
	"fmt"
	"game_main/common"
	"game_main/overworld/tick"

	"github.com/bytearena/ecs"
)

// CreateOverworldTurnState creates the singleton overworld turn state entity
func CreateOverworldTurnState(manager *common.EntityManager) ecs.EntityID {
	entity := manager.World.NewEntity()
	entity.AddComponent(OverworldTurnStateComponent, &OverworldTurnStateData{
		CurrentTurn: 1,
		TurnActive:  true,
	})
	return entity.GetID()
}

// GetOverworldTurnState retrieves the singleton turn state
func GetOverworldTurnState(manager *common.EntityManager) *OverworldTurnStateData {
	results := manager.World.Query(OverworldTurnTag)
	if len(results) == 0 {
		return nil
	}
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
		if attr != nil {
			actionState.MovementRemaining = attr.GetMovementSpeed()
		} else {
			actionState.MovementRemaining = 3 // Default
		}
	}
}

// EndTurn advances the overworld tick and starts a new turn.
// Returns the tick result for raid/event handling.
func EndTurn(manager *common.EntityManager, playerData *common.PlayerData) (tick.TickResult, error) {
	// Advance the overworld simulation
	tickResult, err := tick.AdvanceTick(manager, playerData)
	if err != nil {
		return tickResult, fmt.Errorf("failed to advance tick: %w", err)
	}

	// Increment turn counter
	turnState := GetOverworldTurnState(manager)
	if turnState != nil {
		turnState.CurrentTurn++
	}

	// Reset all commanders for next turn
	StartNewTurn(manager, playerData.PlayerEntityID)

	return tickResult, nil
}

// GetCommanderActionState returns the action state for a specific commander (O(1) lookup).
// Action state is stored directly on the commander entity since CreateCommander.
func GetCommanderActionState(commanderID ecs.EntityID, manager *common.EntityManager) *CommanderActionStateData {
	return common.GetComponentTypeByID[*CommanderActionStateData](manager, commanderID, CommanderActionStateComponent)
}
