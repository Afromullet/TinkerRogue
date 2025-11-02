package combat

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

type TurnManager struct {
	manager *common.EntityManager
}

func NewTurnManager(manager *common.EntityManager) *TurnManager {
	return &TurnManager{
		manager: manager,
	}
}

func (tm *TurnManager) InitializeCombat(factionIDs []ecs.EntityID) error {
	//Randomize turn order using Fisher-Yates shuffle
	turnOrder := make([]ecs.EntityID, len(factionIDs))
	copy(turnOrder, factionIDs)
	shuffleFactionOrder(turnOrder)

	turnEntity := tm.manager.World.NewEntity()
	turnEntity.AddComponent(TurnStateComponent, &TurnStateData{
		CurrentRound:     1,
		TurnOrder:        turnOrder,
		CurrentTurnIndex: 0,
		CombatActive:     true,
	})

	for _, factionID := range factionIDs {
		squads := GetSquadsForFaction(factionID, tm.manager)
		for _, squadID := range squads {
			tm.createActionStateForSquad(squadID)
		}
	}

	//Reset actions for first faction
	firstFaction := turnOrder[0]
	tm.ResetSquadActions(firstFaction)

	return nil
}

func (tm *TurnManager) createActionStateForSquad(squadID ecs.EntityID) {
	actionEntity := tm.manager.World.NewEntity()
	actionEntity.AddComponent(ActionStateComponent, &ActionStateData{
		SquadID:           squadID,
		HasMoved:          false,
		HasActed:          false,
		MovementRemaining: 0, // Set by ResetSquadActions
	})
}

func (tm *TurnManager) ResetSquadActions(factionID ecs.EntityID) error {
	squads := GetSquadsForFaction(factionID, tm.manager)

	//TODO: Do we really need to create a new system, or can we just reset the existing system?
	moveSys := NewMovementSystem(tm.manager, common.GlobalPositionSystem)

	for _, squadID := range squads {
		actionEntity := findActionStateEntity(squadID, tm.manager)
		if actionEntity == nil {
			continue
		}

		actionState := common.GetComponentType[*ActionStateData](actionEntity, ActionStateComponent)

		actionState.HasMoved = false
		actionState.HasActed = false

		// Initialize MovementRemaining from squad speed
		squadSpeed := moveSys.GetSquadMovementSpeed(squadID)
		actionState.MovementRemaining = squadSpeed
	}

	return nil
}

func (tm *TurnManager) GetCurrentFaction() ecs.EntityID {

	turnEntity := findTurnStateEntity(tm.manager)
	if turnEntity == nil {
		return 0 // No active combat
	}

	turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)

	// Return faction ID at current index
	currentIndex := turnState.CurrentTurnIndex
	if currentIndex < 0 || currentIndex >= len(turnState.TurnOrder) {
		return 0 // Invalid state
	}

	return turnState.TurnOrder[currentIndex]
}

func (tm *TurnManager) EndTurn() error {
	turnEntity := findTurnStateEntity(tm.manager)
	if turnEntity == nil {
		return fmt.Errorf("no active combat")
	}

	turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)

	turnState.CurrentTurnIndex++

	// Check for wraparound to start new round
	if turnState.CurrentTurnIndex >= len(turnState.TurnOrder) {
		turnState.CurrentTurnIndex = 0
		turnState.CurrentRound++
	}

	// Reset action states for the new faction's squads
	newFactionID := turnState.TurnOrder[turnState.CurrentTurnIndex]
	if err := tm.ResetSquadActions(newFactionID); err != nil {
		return fmt.Errorf("failed to reset squad actions: %w", err)
	}

	return nil
}

func (tm *TurnManager) IsSquadActivatable(squadID ecs.EntityID) bool {

	currentFaction := tm.GetCurrentFaction()
	if currentFaction == 0 {
		return false // No active combat
	}

	squadFaction := getFactionOwner(squadID, tm.manager)
	if squadFaction != currentFaction {
		return false // Not this faction's squad
	}

	return canSquadAct(squadID, tm.manager)
}

func (tm *TurnManager) GetCurrentRound() int {
	turnEntity := findTurnStateEntity(tm.manager)
	if turnEntity == nil {
		return 0 // No active combat
	}

	turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)
	return turnState.CurrentRound
}

func (tm *TurnManager) EndCombat() error {
	turnEntity := findTurnStateEntity(tm.manager)
	if turnEntity == nil {
		return fmt.Errorf("no active combat to end")
	}

	turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)
	turnState.CombatActive = false

	return nil
}
