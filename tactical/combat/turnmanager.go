package combat

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

type TurnManager struct {
	manager           *common.EntityManager
	combatCache       *CombatQueryCache
	turnStateEntityID ecs.EntityID
	movementSystem    *CombatMovementSystem
}

func NewTurnManager(manager *common.EntityManager, cache *CombatQueryCache) *TurnManager {
	return &TurnManager{
		manager:        manager,
		combatCache:    cache,
		movementSystem: NewMovementSystem(manager, common.GlobalPositionSystem, cache),
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

	// Cache the turn state entity ID to avoid O(n) queries
	tm.turnStateEntityID = turnEntity.GetID()

	// Create action states and check combat-start abilities for all squads
	// TODO: Abilities are not yet implemented, but I forsee not needing this.
	// It will overcomplicate things. Abilities should trigger during an attack, not at combat start
	for _, factionID := range factionIDs {
		factionSquads := GetSquadsForFaction(factionID, tm.manager)
		for _, squadID := range factionSquads {
			CreateActionStateForSquad(tm.manager, squadID)

			// Check for combat-start abilities (like Battle Cry)
			squads.CheckAndTriggerAbilities(squadID, tm.manager)
		}
	}

	//Reset actions for first faction (this will also check abilities again)
	firstFaction := turnOrder[0]
	tm.ResetSquadActions(firstFaction)

	return nil
}

func (tm *TurnManager) ResetSquadActions(factionID ecs.EntityID) error {
	factionSquads := GetSquadsForFaction(factionID, tm.manager)

	for _, squadID := range factionSquads {
		actionEntity := tm.combatCache.FindActionStateEntity(squadID, tm.manager)
		if actionEntity == nil {
			continue
		}

		actionState := common.GetComponentType[*ActionStateData](actionEntity, ActionStateComponent)

		actionState.HasMoved = false
		actionState.HasActed = false

		// Initialize MovementRemaining from squad speed
		squadSpeed := tm.movementSystem.GetSquadMovementSpeed(squadID)
		actionState.MovementRemaining = squadSpeed

		// Check and trigger abilities at start of turn
		squads.CheckAndTriggerAbilities(squadID, tm.manager)
	}

	return nil
}

// getTurnState retrieves the current turn state or returns nil if invalid
func (tm *TurnManager) getTurnState() *TurnStateData {
	if tm.turnStateEntityID == 0 {
		return nil // No active combat
	}

	turnEntity := tm.manager.World.GetEntityByID(tm.turnStateEntityID)
	if turnEntity == nil {
		return nil // Entity not found
	}

	return common.GetComponentType[*TurnStateData](turnEntity.Entity, TurnStateComponent)
}

func (tm *TurnManager) GetCurrentFaction() ecs.EntityID {
	turnState := tm.getTurnState()
	if turnState == nil {
		return 0
	}

	// Return faction ID at current index
	currentIndex := turnState.CurrentTurnIndex
	if currentIndex < 0 || currentIndex >= len(turnState.TurnOrder) {
		return 0 // Invalid state
	}

	return turnState.TurnOrder[currentIndex]
}

func (tm *TurnManager) EndTurn() error {
	turnState := tm.getTurnState()
	if turnState == nil {
		return fmt.Errorf("no active combat")
	}

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

func (tm *TurnManager) GetCurrentRound() int {
	turnState := tm.getTurnState()
	if turnState == nil {
		return 0
	}
	return turnState.CurrentRound
}

func (tm *TurnManager) EndCombat() error {
	turnState := tm.getTurnState()
	if turnState == nil {
		return fmt.Errorf("no active combat to end")
	}

	turnState.CombatActive = false

	// Invalidate cache when combat ends
	tm.turnStateEntityID = 0

	return nil
}
