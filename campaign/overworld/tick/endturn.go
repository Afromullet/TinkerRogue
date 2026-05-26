package tick

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/commander"
)

// EndOverworldTurn advances the overworld tick simulation, increments the
// overworld turn counter, and resets every commander's per-turn action state.
// Returns the tick result so callers can react to raids or other tick events.
//
// Previously lived in tactical/commander; moved here so the commander package
// does not depend on tick (commander becomes reusable outside the overworld
// context, e.g. tutorials and tests).
func EndOverworldTurn(manager *common.EntityManager, playerData *common.PlayerData) (TickResult, error) {
	tickResult, err := AdvanceTick(manager, playerData)
	if err != nil {
		return tickResult, fmt.Errorf("failed to advance tick: %w", err)
	}

	if turnState := commander.GetOverworldTurnState(manager); turnState != nil {
		turnState.CurrentTurn++
	}

	commander.StartNewTurn(manager, playerData.PlayerEntityID)

	return tickResult, nil
}
