package combatsim

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// RecordedCombatExecutor wraps combat execution with battle recording.
// Integrates BattleRecorder to generate JSON battle logs for balance testing.
type RecordedCombatExecutor struct {
	manager        *common.EntityManager
	battleRecorder *battlelog.BattleRecorder
	currentRound   int
}

// NewRecordedCombatExecutor creates a new combat executor with battle recording.
func NewRecordedCombatExecutor(manager *common.EntityManager, recorder *battlelog.BattleRecorder) *RecordedCombatExecutor {
	return &RecordedCombatExecutor{
		manager:        manager,
		battleRecorder: recorder,
		currentRound:   0,
	}
}

// ExecuteRecordedAttack executes an attack from one squad to another and records the engagement.
// Returns the combat result with logged engagement data.
func (e *RecordedCombatExecutor) ExecuteRecordedAttack(attackerID, defenderID ecs.EntityID) *squads.CombatResult {
	// Execute the squad attack
	result := squads.ExecuteSquadAttack(attackerID, defenderID, e.manager)

	// Record engagement if recorder is enabled and combat log exists
	if e.battleRecorder != nil && e.battleRecorder.IsEnabled() && result.CombatLog != nil {
		e.battleRecorder.RecordEngagement(result.CombatLog)
	}

	return result
}

// ExecuteRecordedCounterattack executes a counterattack and records the engagement.
func (e *RecordedCombatExecutor) ExecuteRecordedCounterattack(defenderID, attackerID ecs.EntityID) *squads.CombatResult {
	// Execute the counterattack
	result := squads.ExecuteSquadCounterattack(defenderID, attackerID, e.manager)

	// Record engagement if recorder is enabled and combat log exists
	if e.battleRecorder != nil && e.battleRecorder.IsEnabled() && result.CombatLog != nil {
		e.battleRecorder.RecordEngagement(result.CombatLog)
	}

	return result
}

// RunBattle executes a complete battle with M squads until one victor remains.
// Supports 1v1, 1v1v1, and larger multi-squad battles.
// Returns the victor's squad ID (0 if draw/timeout).
func (e *RecordedCombatExecutor) RunBattle(squadIDs []ecs.EntityID, maxRounds int) (ecs.EntityID, error) {
	if len(squadIDs) < 2 {
		return 0, fmt.Errorf("battle requires at least 2 squads, got %d", len(squadIDs))
	}

	e.currentRound = 0

	// Battle loop
	for e.currentRound < maxRounds {
		e.currentRound++

		// Update recorder with current round
		if e.battleRecorder != nil && e.battleRecorder.IsEnabled() {
			e.battleRecorder.SetCurrentRound(e.currentRound)
		}

		// Get alive squads
		aliveSquads := e.getAliveSquads(squadIDs)

		// Check victory condition
		if len(aliveSquads) <= 1 {
			if len(aliveSquads) == 1 {
				return aliveSquads[0], nil // Winner
			}
			return 0, nil // Draw (all destroyed simultaneously)
		}

		// Each alive squad attacks all other alive squads
		for _, attackerID := range aliveSquads {
			// Get current alive squads again (may have changed after previous attacks)
			currentAlive := e.getAliveSquads(squadIDs)

			for _, defenderID := range currentAlive {
				if attackerID != defenderID {
					// Execute attack and record
					e.ExecuteRecordedAttack(attackerID, defenderID)
				}
			}
		}
	}

	// Battle timeout - return first alive squad or 0 for draw
	aliveSquads := e.getAliveSquads(squadIDs)
	if len(aliveSquads) == 1 {
		return aliveSquads[0], nil
	}
	return 0, fmt.Errorf("battle timeout after %d rounds", maxRounds)
}

// getAliveSquads returns the list of squad IDs that are still alive.
func (e *RecordedCombatExecutor) getAliveSquads(squadIDs []ecs.EntityID) []ecs.EntityID {
	alive := make([]ecs.EntityID, 0, len(squadIDs))
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, e.manager) {
			alive = append(alive, squadID)
		}
	}
	return alive
}

// GetCurrentRound returns the current battle round.
func (e *RecordedCombatExecutor) GetCurrentRound() int {
	return e.currentRound
}

// Finalize completes the battle record with victory information.
// Returns the complete BattleRecord ready for export.
func (e *RecordedCombatExecutor) Finalize(victorID ecs.EntityID, victorName string) *battlelog.BattleRecord {
	if e.battleRecorder == nil {
		return nil
	}

	victoryInfo := &battlelog.VictoryInfo{
		RoundsCompleted: e.currentRound,
		VictorFaction:   victorID,
		VictorName:      victorName,
	}

	return e.battleRecorder.Finalize(victoryInfo)
}
