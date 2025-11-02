package combat

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// ========================================
// VICTORY CONDITION INTERFACE (STUB)
// ========================================

// VictoryCondition defines how combat victory is determined
type VictoryCondition interface {
	// CheckVictory returns (winnerFactionID, combatEnded)
	// Returns (0, false) if combat continues
	// Returns (factionID, true) if that faction won
	CheckVictory(manager *common.EntityManager) (ecs.EntityID, bool)

	// GetVictoryDescription returns a human-readable description
	GetVictoryDescription() string
}

// ========================================
// DEFAULT VICTORY CONDITION (STUB)
// ========================================

// EliminationVictory implements standard "last faction standing" victory
type EliminationVictory struct{}

// NewEliminationVictory creates the default victory condition
func NewEliminationVictory() VictoryCondition {
	return &EliminationVictory{}
}

// CheckVictory checks if only one faction has squads remaining
func (ev *EliminationVictory) CheckVictory(manager *common.EntityManager) (ecs.EntityID, bool) {
	// STUB: Always returns false (combat never ends)
	// TODO: Implement actual faction squad counting logic
	//
	// Proper implementation:
	// 1. Count squads per faction
	// 2. Find factions with 1+ squads
	// 3. If only one faction has squads, they win
	// 4. If multiple factions have squads, combat continues
	// 5. If no factions have squads, it's a draw (return 0, true)

	return 0, false // Combat never ends (stub)
}

// GetVictoryDescription returns the victory condition name
func (ev *EliminationVictory) GetVictoryDescription() string {
	return "Elimination (Last Faction Standing)"
}

// ========================================
// FUTURE VICTORY CONDITIONS (STUBS)
// ========================================

// ObjectiveVictory implements objective-based victory (capture points, etc.)
// STUB: Not implemented
type ObjectiveVictory struct {
	// TODO: Add objective tracking fields
	// RequiredObjectives []ObjectiveID
	// ObjectiveState     map[ObjectiveID]bool
}

// TurnLimitVictory implements turn-based victory (most points after N turns)
// STUB: Not implemented
type TurnLimitVictory struct {
	// TODO: Add turn limit fields
	// MaxTurns      int
	// PointsPerKill int
}
