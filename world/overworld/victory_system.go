package overworld

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// CheckVictoryCondition evaluates if player has won or lost
func CheckVictoryCondition(manager *common.EntityManager) VictoryCondition {
	// Get victory state (if configured)
	victoryState := GetVictoryState(manager)

	// Check defeat conditions first (highest priority)
	if IsPlayerDefeated(manager) {
		if victoryState != nil {
			victoryState.Condition = VictoryPlayerLoses
			victoryState.VictoryAchieved = true
			victoryState.DefeatReason = GetDefeatReason(manager)
		}

		// Log defeat event
		tickState := GetTickState(manager)
		currentTick := int64(0)
		if tickState != nil {
			currentTick = tickState.CurrentTick
		}
		defeatReason := GetDefeatReason(manager)
		LogEvent(EventDefeat, currentTick, 0, defeatReason)

		// Export overworld log on defeat
		FinalizeRecording("Defeat", defeatReason)

		return VictoryPlayerLoses
	}

	// Check survival victory first (if configured) - takes priority over threat elimination
	if victoryState != nil && victoryState.TicksToSurvive > 0 {
		tickState := GetTickState(manager)
		if tickState != nil && tickState.CurrentTick >= victoryState.TicksToSurvive {
			victoryState.Condition = VictoryTimeLimit
			victoryState.VictoryAchieved = true

			// Log victory event
			victoryReason := formatEventString("Victory! Survived %d ticks", victoryState.TicksToSurvive)
			LogEvent(EventVictory, tickState.CurrentTick, 0, victoryReason)

			// Export overworld log on survival victory
			FinalizeRecording("Victory", victoryReason)

			return VictoryTimeLimit
		}
		// Still surviving - game continues
		return VictoryNone
	}

	// Check threat elimination victory (only if no survival condition)
	if HasPlayerEliminatedAllThreats(manager) {
		if victoryState != nil {
			victoryState.Condition = VictoryPlayerWins
			victoryState.VictoryAchieved = true
		}

		// Log victory event
		tickState := GetTickState(manager)
		currentTick := int64(0)
		if tickState != nil {
			currentTick = tickState.CurrentTick
		}
		victoryReason := "Victory! All threats eliminated"
		LogEvent(EventVictory, currentTick, 0, victoryReason)

		// Export overworld log on threat elimination victory
		FinalizeRecording("Victory", victoryReason)

		return VictoryPlayerWins
	}

	// Check faction-specific victory (if configured)
	if victoryState != nil && victoryState.TargetFactionType != FactionType(0) {
		if HasPlayerDefeatedFactionType(manager, victoryState.TargetFactionType) {
			victoryState.Condition = VictoryFactionDefeat
			victoryState.VictoryAchieved = true

			// Log victory event
			tickState := GetTickState(manager)
			currentTick := int64(0)
			if tickState != nil {
				currentTick = tickState.CurrentTick
			}
			victoryReason := formatEventString("Victory! Defeated all %s factions", victoryState.TargetFactionType.String())
			LogEvent(EventVictory, currentTick, 0, victoryReason)

			// Export overworld log on faction victory
			FinalizeRecording("Victory", victoryReason)

			return VictoryFactionDefeat
		}
	}

	return VictoryNone
}

// CreateVictoryStateEntity creates singleton victory tracking entity
func CreateVictoryStateEntity(
	manager *common.EntityManager,
	ticksToSurvive int64,
	targetFactionType FactionType,
) ecs.EntityID {
	entity := manager.World.NewEntity()

	entity.AddComponent(VictoryStateComponent, &VictoryStateData{
		Condition:         VictoryNone,
		TicksToSurvive:    ticksToSurvive,
		TargetFactionType: targetFactionType,
		VictoryAchieved:   false,
		DefeatReason:      "",
	})

	return entity.GetID()
}
