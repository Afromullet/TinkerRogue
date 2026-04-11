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

// ThreatSnapshot holds all threat layer values at a single map position.
// Returned by ThreatLayerEvaluator.EvaluateAt so callers get all values in one call.
type ThreatSnapshot struct {
	MeleeThreat        float64
	RangedPressure     float64
	SupportValue       float64
	FlankingRisk       float64
	IsolationRisk      float64
	EngagementPressure float64
	RetreatQuality     float64
}

// ThreatLayerEvaluator evaluates composite threat layers at map positions.
// Implemented by behavior.CompositeThreatEvaluator; defined here so tactical/gui
// layers can query threat layers without importing mind/behavior.
type ThreatLayerEvaluator interface {
	Update(currentRound int)
	MarkDirty()
	EvaluateAt(pos coords.LogicalPosition) ThreatSnapshot
}
