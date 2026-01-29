package behavior

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// RoleThreatWeights defines how each role weighs different threat layers
// Negative weights = attraction (e.g., tanks seek melee danger, support seeks wounded allies)
// Positive weights = avoidance (e.g., support avoids all danger)
type RoleThreatWeights struct {
	MeleeWeight      float64
	RangedWeight     float64
	SupportWeight    float64
	PositionalWeight float64
}

// DefaultRoleWeights defines standard weights for each role
var DefaultRoleWeights = map[squads.UnitRole]RoleThreatWeights{
	squads.RoleTank: {
		MeleeWeight:      -0.5, // Tanks SEEK melee danger (intercept enemies)
		RangedWeight:     0.3,  // Moderate concern for ranged
		SupportWeight:    0.2,  // Stay near support for heals
		PositionalWeight: 0.5,  // High concern for isolation
	},
	squads.RoleDPS: {
		MeleeWeight:      0.7, // Avoid melee danger
		RangedWeight:     0.5, // Moderate concern for ranged
		SupportWeight:    0.1, // Low support priority
		PositionalWeight: 0.6, // High concern for flanking
	},
	squads.RoleSupport: {
		MeleeWeight:      1.0,  // Strongly avoid melee danger
		RangedWeight:     0.8,  // Strongly avoid ranged pressure
		SupportWeight:    -1.0, // SEEK high support value positions (wounded allies)
		PositionalWeight: 0.4,  // Moderate positional awareness
	},
}

// CompositeThreatEvaluator combines multiple threat layers
// Provides role-aware threat queries for AI decision-making
type CompositeThreatEvaluator struct {
	manager   *common.EntityManager
	cache     *combat.CombatQueryCache
	factionID ecs.EntityID

	// Individual layers
	meleeThreat    *MeleeThreatLayer
	rangedThreat   *RangedThreatLayer
	supportValue   *SupportValueLayer
	positionalRisk *PositionalRiskLayer

	// Cache invalidation
	lastUpdateRound int
	isDirty         bool
}

// NewCompositeThreatEvaluator creates a new composite threat evaluator
func NewCompositeThreatEvaluator(
	factionID ecs.EntityID,
	manager *common.EntityManager,
	cache *combat.CombatQueryCache,
	baseThreatMgr *FactionThreatLevelManager,
) *CompositeThreatEvaluator {
	// Create melee and ranged layers first (needed by positional layer)
	meleeLayer := NewMeleeThreatLayer(factionID, manager, cache, baseThreatMgr)
	rangedLayer := NewRangedThreatLayer(factionID, manager, cache, baseThreatMgr)

	return &CompositeThreatEvaluator{
		manager:         manager,
		cache:           cache,
		factionID:       factionID,
		meleeThreat:     meleeLayer,
		rangedThreat:    rangedLayer,
		supportValue:    NewSupportValueLayer(factionID, manager, cache, baseThreatMgr),
		positionalRisk:  NewPositionalRiskLayer(factionID, manager, cache, baseThreatMgr, meleeLayer, rangedLayer),
		lastUpdateRound: -1,
		isDirty:         true,
	}
}

// Update recomputes all layers if needed
// Should be called at the start of each AI turn
func (cte *CompositeThreatEvaluator) Update(currentRound int) {
	// Skip if already up-to-date
	if !cte.isDirty && cte.lastUpdateRound == currentRound {
		return
	}

	// Compute base threat layers first (melee/ranged)
	cte.meleeThreat.Compute()
	cte.rangedThreat.Compute()

	// Then compute derived layers (support/positional depend on melee/ranged)
	cte.supportValue.Compute()
	cte.positionalRisk.Compute()

	// Mark as clean
	cte.lastUpdateRound = currentRound
	cte.isDirty = false
}

// MarkDirty forces recomputation on next Update()
// Call when squad moves, is destroyed, or combat state changes
func (cte *CompositeThreatEvaluator) MarkDirty() {
	cte.isDirty = true
	cte.meleeThreat.MarkDirty()
	cte.rangedThreat.MarkDirty()
	cte.supportValue.MarkDirty()
	cte.positionalRisk.MarkDirty()
}

// GetRoleWeightedThreat returns combined threat score for a position
// based on the squad's role composition
// Lower score = better position
func (cte *CompositeThreatEvaluator) GetRoleWeightedThreat(
	squadID ecs.EntityID,
	pos coords.LogicalPosition,
) float64 {
	role := squads.GetSquadPrimaryRole(squadID, cte.manager)
	weights := DefaultRoleWeights[role]

	meleeThreat := cte.meleeThreat.GetMeleeThreatAt(pos)
	rangedThreat := cte.rangedThreat.GetRangedPressureAt(pos)
	supportValue := cte.supportValue.GetSupportValueAt(pos)
	positionalRisk := cte.positionalRisk.GetTotalRiskAt(pos)

	// Combine threats with role-specific weights
	// Negative weights invert threat (e.g., tanks attracted to melee, support attracted to wounded allies)
	totalThreat := meleeThreat*weights.MeleeWeight +
		rangedThreat*weights.RangedWeight +
		supportValue*weights.SupportWeight +
		positionalRisk*weights.PositionalWeight

	return totalThreat
}

// GetOptimalPositionForRole finds best position for a squad given its role
// Returns position with LOWEST threat score (best for survival/positioning)
func (cte *CompositeThreatEvaluator) GetOptimalPositionForRole(
	squadID ecs.EntityID,
	candidatePositions []coords.LogicalPosition,
) coords.LogicalPosition {
	if len(candidatePositions) == 0 {
		return coords.LogicalPosition{}
	}

	bestPos := candidatePositions[0]
	bestScore := cte.GetRoleWeightedThreat(squadID, bestPos)

	for _, pos := range candidatePositions[1:] {
		score := cte.GetRoleWeightedThreat(squadID, pos)
		if score < bestScore {
			bestScore = score
			bestPos = pos
		}
	}

	return bestPos
}

// GetMeleeLayer returns direct access to melee layer (for specific queries)
func (cte *CompositeThreatEvaluator) GetMeleeLayer() *MeleeThreatLayer {
	return cte.meleeThreat
}

// GetRangedLayer returns direct access to ranged layer
func (cte *CompositeThreatEvaluator) GetRangedLayer() *RangedThreatLayer {
	return cte.rangedThreat
}

// GetSupportLayer returns direct access to support value layer
func (cte *CompositeThreatEvaluator) GetSupportLayer() *SupportValueLayer {
	return cte.supportValue
}

// GetPositionalLayer returns direct access to positional risk layer
func (cte *CompositeThreatEvaluator) GetPositionalLayer() *PositionalRiskLayer {
	return cte.positionalRisk
}
