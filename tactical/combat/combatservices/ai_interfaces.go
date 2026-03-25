package combatservices

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// QueuedAttack represents an attack queued for animation.
type QueuedAttack struct {
	AttackerID ecs.EntityID
	DefenderID ecs.EntityID
}

// AITurnController handles AI decision-making for a faction turn.
// Used by combat_turn_flow.go to avoid importing mind/ai directly.
type AITurnController interface {
	DecideFactionTurn(factionID ecs.EntityID) bool
	HasQueuedAttacks() bool
	GetQueuedAttacks() []QueuedAttack
	ClearAttackQueue()
}

// ThreatProvider provides faction-level threat data (base threat values per squad).
// Implemented by behavior.FactionThreatLevelManager; defined here so tactical/gui
// layers can use threat data without importing mind/behavior.
type ThreatProvider interface {
	AddFaction(factionID ecs.EntityID)
	UpdateAllFactions()
	GetSquadThreatAtRange(factionID, squadID ecs.EntityID, distance int) (float64, bool)
}

// ThreatLayerEvaluator evaluates composite threat layers at map positions.
// Implemented by behavior.CompositeThreatEvaluator; defined here so tactical/gui
// layers can query threat layers without importing mind/behavior.
type ThreatLayerEvaluator interface {
	Update(currentRound int)
	MarkDirty()
	GetMeleeThreatAt(pos coords.LogicalPosition) float64
	GetRangedPressureAt(pos coords.LogicalPosition) float64
	GetSupportValueAt(pos coords.LogicalPosition) float64
	GetFlankingRiskAt(pos coords.LogicalPosition) float64
	GetIsolationRiskAt(pos coords.LogicalPosition) float64
	GetEngagementPressureAt(pos coords.LogicalPosition) float64
	GetRetreatQuality(pos coords.LogicalPosition) float64
}
