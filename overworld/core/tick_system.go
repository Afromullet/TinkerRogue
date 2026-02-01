// tick_system.go - Turn-based tick orchestration system
//
// The overworld operates on a turn-based tick system where time only advances
// through explicit player actions (manual advancement, movement, etc.).
//
// Core function: AdvanceTick() orchestrates all overworld subsystems in order.
// Victory/defeat conditions set IsGameOver flag to prevent further ticks.

package core

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// TickAdvancer is an interface for systems that can be advanced per tick.
// This allows the core tick system to orchestrate subsystems without direct imports.
type TickAdvancer interface {
	AdvanceTick(manager *common.EntityManager, currentTick int64) error
}

// TravelAdvancer is an interface for the travel system
type TravelAdvancer interface {
	AdvanceTravelTick(manager *common.EntityManager, playerData *common.PlayerData) (bool, error)
}

// VictoryChecker is an interface for checking victory conditions
type VictoryChecker interface {
	CheckVictoryCondition(manager *common.EntityManager) VictoryCondition
}

// Subsystem registrations - set these from the respective packages
var (
	ThreatSystem  TickAdvancer
	FactionSystem TickAdvancer
	TravelSystem  TravelAdvancer
	VictorySystem VictoryChecker
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
// Returns true if travel was completed this tick, false otherwise.
//
// This function should be called when the player performs actions that advance time:
//   - Manual advancement (Space key)
//   - Movement/travel
//   - Other turn-consuming actions
func AdvanceTick(manager *common.EntityManager, playerData *common.PlayerData) (bool, error) {
	tickState := GetTickState(manager)
	if tickState == nil {
		return false, fmt.Errorf("tick state not initialized")
	}

	if tickState.IsGameOver {
		return false, nil
	}

	// Increment tick counter
	tickState.CurrentTick++
	tick := tickState.CurrentTick

	// Advance travel if active (before other subsystems)
	travelCompleted := false
	if TravelSystem != nil {
		var err error
		travelCompleted, err = TravelSystem.AdvanceTravelTick(manager, playerData)
		if err != nil {
			return false, fmt.Errorf("travel update failed: %w", err)
		}
	}

	// Execute subsystems in order (world continues evolving during travel)
	if ThreatSystem != nil {
		if err := ThreatSystem.AdvanceTick(manager, tick); err != nil {
			return false, fmt.Errorf("threat update failed: %w", err)
		}
	}

	if FactionSystem != nil {
		if err := FactionSystem.AdvanceTick(manager, tick); err != nil {
			return false, fmt.Errorf("faction update failed: %w", err)
		}
	}

	// Note: Influence calculation is now handled by InfluenceCache (see influence_cache.go)
	// The cache is updated on-demand when threats are added/removed/moved

	// Note: Events are logged inline by individual systems via LogEvent()
	// No batch event processing needed currently

	// Check victory/loss conditions
	if VictorySystem != nil {
		victoryCondition := VictorySystem.CheckVictoryCondition(manager)
		if victoryCondition != VictoryNone {
			// Victory or defeat achieved - set game over flag
			tickState.IsGameOver = true
		}
	}

	return travelCompleted, nil
}
