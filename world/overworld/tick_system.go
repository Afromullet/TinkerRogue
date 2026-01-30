// tick_system.go - Turn-based tick orchestration system
//
// The overworld operates on a turn-based tick system where time only advances
// through explicit player actions (manual advancement, movement, etc.).
//
// Core function: AdvanceTick() orchestrates all overworld subsystems in order.
// Victory/defeat conditions set IsGameOver flag to prevent further ticks.

package overworld

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// CreateTickStateEntity creates singleton tick state entity
func CreateTickStateEntity(manager *common.EntityManager) ecs.EntityID {
	entity := manager.World.NewEntity()
	entity.AddComponent(TickStateComponent, &TickStateData{
		CurrentTick: 0,
		IsGameOver:  false,
	})

	// Start recording session when tick state is created
	StartRecordingSession(0)

	return entity.GetID()
}

// AdvanceTick executes one turn of overworld simulation (turn-based system).
// This is the master orchestration function that updates all subsystems in sequence.
// Returns immediately if game is over (victory/defeat achieved).
//
// This function should be called when the player performs actions that advance time:
//   - Manual advancement (Space key)
//   - Movement/travel (future feature)
//   - Other turn-consuming actions
func AdvanceTick(manager *common.EntityManager) error {
	tickState := GetTickState(manager)
	if tickState == nil {
		return fmt.Errorf("tick state not initialized")
	}

	if tickState.IsGameOver {
		return nil
	}

	// Increment tick counter
	tickState.CurrentTick++
	tick := tickState.CurrentTick

	// Execute subsystems in order
	if err := UpdateThreatNodes(manager, tick); err != nil {
		return fmt.Errorf("threat update failed: %w", err)
	}

	if err := UpdateFactions(manager, tick); err != nil {
		return fmt.Errorf("faction update failed: %w", err)
	}

	// Note: Influence calculation is now handled by InfluenceCache (see influence_cache.go)
	// The cache is updated on-demand when threats are added/removed/moved

	// Note: Events are logged inline by individual systems via LogEvent()
	// No batch event processing needed currently

	// Check victory/loss conditions
	victoryCondition := CheckVictoryCondition(manager)
	if victoryCondition != VictoryNone {
		// Victory or defeat achieved - set game over flag
		tickState.IsGameOver = true
	}

	return nil
}

// GetTickState retrieves the singleton tick state
func GetTickState(manager *common.EntityManager) *TickStateData {
	for _, result := range manager.World.Query(TickStateTag) {
		return common.GetComponentType[*TickStateData](result.Entity, TickStateComponent)
	}
	return nil
}

