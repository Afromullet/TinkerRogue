// manager.go - Overworld tick orchestration
//
// This package orchestrates all overworld subsystems (threat, faction, travel, victory).
// It breaks the circular dependency by importing both core and subsystems.

package tick

import (
	"fmt"
	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/faction"
	"game_main/overworld/influence"
	"game_main/overworld/threat"

	"github.com/bytearena/ecs"
)

// TickResult captures the outcome of a single tick advancement.
type TickResult struct {
	PendingRaid *core.PendingRaid // Non-nil if a faction raid targets a garrisoned player node
}

// CreateTickStateEntity creates singleton tick state entity
func CreateTickStateEntity(manager *common.EntityManager) ecs.EntityID {
	entity := manager.World.NewEntity()
	entity.AddComponent(core.TickStateComponent, &core.TickStateData{
		CurrentTick: 0,
		IsGameOver:  false,
	})

	// Start recording session when tick state is created
	core.StartRecordingSession(0)

	return entity.GetID()
}

// AdvanceTick executes one turn of overworld simulation (turn-based system).
// This is the master orchestration function that updates all subsystems in sequence.
// Returns immediately if game is over (victory/defeat achieved).
// Returns TickResult containing travel completion status and any pending raids.
//
// This function should be called when the player performs actions that advance time:
//   - Manual advancement (Space key)
//   - Movement/travel
//   - Other turn-consuming actions
func AdvanceTick(manager *common.EntityManager, playerData *common.PlayerData) (TickResult, error) {
	result := TickResult{}

	tickState := core.GetTickState(manager)
	if tickState == nil {
		return result, fmt.Errorf("tick state not initialized")
	}

	if tickState.IsGameOver {
		return result, nil
	}

	// Increment tick counter
	tickState.CurrentTick++
	tick := tickState.CurrentTick

	// Resolve influence interactions before subsystems use the results
	influence.UpdateInfluenceInteractions(manager, tick)

	// Execute subsystems in order (world continues evolving during travel)
	if err := threat.UpdateThreatNodes(manager, tick); err != nil {
		return result, fmt.Errorf("threat update failed: %w", err)
	}

	pendingRaid, err := faction.UpdateFactions(manager, tick)
	if err != nil {
		return result, fmt.Errorf("faction update failed: %w", err)
	}
	result.PendingRaid = pendingRaid

	// Note: Influence calculation is now handled by InfluenceCache (see influence_cache.go)
	// The cache is updated on-demand when threats are added/removed/moved

	// Note: Events are logged inline by individual systems via LogEvent()
	// No batch event processing needed currently

	return result, nil
}
