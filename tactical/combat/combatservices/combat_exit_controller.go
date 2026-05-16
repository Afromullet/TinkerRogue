package combatservices

import (
	"fmt"

	"game_main/core/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/combat/combatstate"

	"github.com/bytearena/ecs"
)

// VictoryCheckResult contains battle outcome information.
type VictoryCheckResult struct {
	BattleOver       bool
	VictorFaction    ecs.EntityID
	VictorName       string
	IsPlayerVictory  bool // True if a player-controlled faction won
	DefeatedFactions []ecs.EntityID
	RoundsCompleted  int
}

// CombatExitController owns the per-battle exit state — whether the player
// requested to flee, and the cached victory result populated by the turn flow
// when combat ends. Separating these from CombatService keeps the service
// focused on system ownership and lifecycle.
type CombatExitController struct {
	entityManager *common.EntityManager
	turnManager   *combatcore.TurnManager
	combatCache   *combatstate.CombatQueryCache

	fleeRequested       bool
	cachedVictoryResult *VictoryCheckResult
}

// NewCombatExitController wires the exit controller to the state it needs to
// compute victory conditions. The turn manager supplies the current round; the
// query cache resolves faction data; the entity manager is used to enumerate
// surviving squads.
func NewCombatExitController(em *common.EntityManager, tm *combatcore.TurnManager, cache *combatstate.CombatQueryCache) *CombatExitController {
	return &CombatExitController{
		entityManager: em,
		turnManager:   tm,
		combatCache:   cache,
	}
}

// MarkFleeRequested records that the player chose to flee.
func (ec *CombatExitController) MarkFleeRequested() {
	ec.fleeRequested = true
}

// IsFleeRequested reports whether the player requested to flee.
func (ec *CombatExitController) IsFleeRequested() bool {
	return ec.fleeRequested
}

// CacheVictoryResult stores the victory/flee outcome so the mode exit can
// consume it without re-running CheckVictoryCondition.
func (ec *CombatExitController) CacheVictoryResult(result *VictoryCheckResult) {
	ec.cachedVictoryResult = result
}

// GetExitResult returns the cached victory/flee outcome, falling back to a fresh
// CheckVictoryCondition if nothing was cached (e.g., abnormal exits).
func (ec *CombatExitController) GetExitResult() *VictoryCheckResult {
	if ec.cachedVictoryResult != nil {
		return ec.cachedVictoryResult
	}
	return ec.CheckVictoryCondition()
}

// ClearExitState resets the flee flag and cached victory result. Called after
// the mode exit has consumed them, so the next combat starts clean.
func (ec *CombatExitController) ClearExitState() {
	ec.fleeRequested = false
	ec.cachedVictoryResult = nil
}

// CheckVictoryCondition checks if battle has ended (zero or one faction has
// surviving squads). When exactly one faction survives, its name and player
// flag are populated on the result.
func (ec *CombatExitController) CheckVictoryCondition() *VictoryCheckResult {
	result := &VictoryCheckResult{
		RoundsCompleted: ec.turnManager.GetCurrentRound(),
	}

	aliveByFaction := make(map[ecs.EntityID]int)
	allFactions := combatstate.GetAllFactions(ec.entityManager)
	for _, factionID := range allFactions {
		activeSquads := combatstate.GetActiveSquadsForFaction(factionID, ec.entityManager)
		aliveByFaction[factionID] = len(activeSquads)
	}

	factionsWithSquads := 0
	var victorFaction ecs.EntityID
	for factionID, count := range aliveByFaction {
		if count > 0 {
			factionsWithSquads++
			victorFaction = factionID
		} else {
			result.DefeatedFactions = append(result.DefeatedFactions, factionID)
		}
	}

	if factionsWithSquads <= 1 {
		result.BattleOver = true
		result.VictorFaction = victorFaction

		factionData := ec.combatCache.FindFactionDataByID(victorFaction)
		if factionData != nil {
			result.IsPlayerVictory = factionData.IsPlayerControlled
			if factionData.PlayerID > 0 {
				result.VictorName = fmt.Sprintf("%s (%s)", factionData.Name, factionData.PlayerName)
			} else {
				result.VictorName = factionData.Name
			}
		} else {
			result.VictorName = "Unknown"
			result.IsPlayerVictory = false
		}
	}

	return result
}
