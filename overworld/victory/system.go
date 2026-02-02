package victory

import (
	"fmt"
	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/threat"

	"github.com/bytearena/ecs"
)

// CheckVictoryCondition evaluates if player has won or lost
func CheckVictoryCondition(manager *common.EntityManager) core.VictoryCondition {
	// Get victory state (if configured)
	victoryState := GetVictoryState(manager)

	// Check defeat conditions first (highest priority) - single check replaces duplicate logic
	defeatCheck := CheckPlayerDefeat(manager)
	if defeatCheck.IsDefeated {
		if victoryState != nil {
			victoryState.Condition = core.VictoryPlayerLoses
			victoryState.VictoryAchieved = true
			victoryState.DefeatReason = defeatCheck.DefeatMessage
		}

		// Log defeat event
		core.LogEvent(core.EventDefeat, core.GetCurrentTick(manager), 0, defeatCheck.DefeatMessage, nil)

		// Export overworld log on defeat
		if err := core.FinalizeRecording("Defeat", defeatCheck.DefeatMessage); err != nil {
			fmt.Printf("WARNING: Failed to export overworld log: %v\n", err)
		}

		return core.VictoryPlayerLoses
	}

	// Check survival victory first (if configured) - takes priority over threat elimination
	if victoryState != nil && victoryState.TicksToSurvive > 0 {
		currentTick := core.GetCurrentTick(manager)
		if currentTick >= victoryState.TicksToSurvive {
			victoryState.Condition = core.VictoryTimeLimit
			victoryState.VictoryAchieved = true

			// Log victory event
			victoryReason := core.FormatEventString("Victory! Survived %d ticks", victoryState.TicksToSurvive)
			core.LogEvent(core.EventVictory, currentTick, 0, victoryReason, nil)

			// Export overworld log on survival victory
			if err := core.FinalizeRecording("Victory", victoryReason); err != nil {
				fmt.Printf("WARNING: Failed to export overworld log: %v\n", err)
			}

			return core.VictoryTimeLimit
		}
		// Still surviving - game continues
		return core.VictoryNone
	}

	// Check threat elimination victory (only if no survival condition)
	if HasPlayerEliminatedAllThreats(manager) {
		if victoryState != nil {
			victoryState.Condition = core.VictoryPlayerWins
			victoryState.VictoryAchieved = true
		}

		// Log victory event
		victoryReason := "Victory! All threats eliminated"
		core.LogEvent(core.EventVictory, core.GetCurrentTick(manager), 0, victoryReason, nil)

		// Export overworld log on threat elimination victory
		if err := core.FinalizeRecording("Victory", victoryReason); err != nil {
			fmt.Printf("WARNING: Failed to export overworld log: %v\n", err)
		}

		return core.VictoryPlayerWins
	}

	// Check faction-specific victory (if configured)
	if victoryState != nil && victoryState.TargetFactionType != core.FactionType(0) {
		if HasPlayerDefeatedFactionType(manager, victoryState.TargetFactionType) {
			victoryState.Condition = core.VictoryFactionDefeat
			victoryState.VictoryAchieved = true

			// Log victory event
			victoryReason := core.FormatEventString("Victory! Defeated all %s factions", victoryState.TargetFactionType.String())
			core.LogEvent(core.EventVictory, core.GetCurrentTick(manager), 0, victoryReason, nil)

			// Export overworld log on faction victory
			if err := core.FinalizeRecording("Victory", victoryReason); err != nil {
				fmt.Printf("WARNING: Failed to export overworld log: %v\n", err)
			}

			return core.VictoryFactionDefeat
		}
	}

	return core.VictoryNone
}

// CreateVictoryStateEntity creates singleton victory tracking entity
func CreateVictoryStateEntity(
	manager *common.EntityManager,
	ticksToSurvive int64,
	targetFactionType core.FactionType,
) ecs.EntityID {
	entity := manager.World.NewEntity()

	entity.AddComponent(core.VictoryStateComponent, &core.VictoryStateData{
		Condition:         core.VictoryNone,
		TicksToSurvive:    ticksToSurvive,
		TargetFactionType: targetFactionType,
		VictoryAchieved:   false,
		DefeatReason:      "",
	})

	return entity.GetID()
}

// HasPlayerEliminatedAllThreats checks if all threats are gone
func HasPlayerEliminatedAllThreats(manager *common.EntityManager) bool {
	threatCount := threat.CountThreatNodes(manager)
	return threatCount == 0
}

// HasPlayerDefeatedFactionType checks if specific faction type is eliminated
func HasPlayerDefeatedFactionType(manager *common.EntityManager, factionType core.FactionType) bool {
	for _, result := range manager.World.Query(core.OverworldFactionTag) {
		factionData := common.GetComponentType[*core.OverworldFactionData](result.Entity, core.OverworldFactionComponent)
		if factionData != nil && factionData.FactionType == factionType {
			return false // Faction still exists
		}
	}
	return true // No factions of this type found
}
