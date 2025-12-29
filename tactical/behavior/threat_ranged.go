package behavior

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// RangedThreatLayer computes threat from enemy ranged/magic squads
// Ranged squads threaten at their attack range WITHOUT needing to move
// No distance falloff - full threat at all ranges (ranged advantage)
type RangedThreatLayer struct {
	*ThreatLayerBase

	// Core threat data
	pressureByPos   map[coords.LogicalPosition]float64              // Position -> ranged pressure
	lineOfFireZones map[ecs.EntityID][]coords.LogicalPosition // Squad -> threatened positions

	// Dependencies
	baseThreatMgr *FactionThreatLevelManager
}

// NewRangedThreatLayer creates a new ranged threat layer for a faction
func NewRangedThreatLayer(
	factionID ecs.EntityID,
	manager *common.EntityManager,
	cache *combat.CombatQueryCache,
	baseThreatMgr *FactionThreatLevelManager,
) *RangedThreatLayer {
	return &RangedThreatLayer{
		ThreatLayerBase: NewThreatLayerBase(factionID, manager, cache),
		pressureByPos:   make(map[coords.LogicalPosition]float64),
		lineOfFireZones: make(map[ecs.EntityID][]coords.LogicalPosition),
		baseThreatMgr:   baseThreatMgr,
	}
}

// Compute recalculates ranged threat for all enemy squads
// Paints pressure values on the map based on ranged attack capabilities
func (rtl *RangedThreatLayer) Compute() {
	// Clear existing data (reuse maps to reduce GC pressure)
	clear(rtl.pressureByPos)
	clear(rtl.lineOfFireZones)

	enemyFactions := rtl.getEnemyFactions()

	for _, enemyFactionID := range enemyFactions {
		squadIDs := combat.GetSquadsForFaction(enemyFactionID, rtl.manager)

		for _, squadID := range squadIDs {
			if squads.IsSquadDestroyed(squadID, rtl.manager) {
				continue
			}

			// Check if squad has ranged units
			if !rtl.hasRangedUnits(squadID) {
				continue
			}

			squadPos, err := combat.GetSquadMapPosition(squadID, rtl.manager)
			if err != nil {
				continue
			}

			// Get max ranged attack range
			maxRange := rtl.getMaxRangedRange(squadID)

			// Get ranged damage from base threat system
			factionThreat, exists := rtl.baseThreatMgr.factions[enemyFactionID]
			if !exists {
				continue
			}

			squadThreat, exists := factionThreat.squadDangerLevel[squadID]
			if !exists {
				continue
			}

			// Use damage at max range
			rangedDamage := squadThreat.ExpectedDamageByRange[maxRange]

			// Paint ranged pressure (no distance falloff - full threat at max range)
			rtl.paintRangedPressure(squadID, squadPos, maxRange, rangedDamage)
		}
	}

	// Mark as clean (round will be updated by Update() call)
	// We don't track rounds internally - that's handled by CompositeThreatEvaluator
	rtl.markClean(0)
}

// hasRangedUnits checks if squad has any ranged/magic units
func (rtl *RangedThreatLayer) hasRangedUnits(squadID ecs.EntityID) bool {
	return hasUnitsWithAttackType(squadID, rtl.manager, RangedAttackTypes)
}

// getMaxRangedRange returns maximum ranged attack range
func (rtl *RangedThreatLayer) getMaxRangedRange(squadID ecs.EntityID) int {
	return getMaxRangeForAttackTypes(squadID, rtl.manager, RangedAttackTypes, 3)
}

// paintRangedPressure paints ranged threat without distance falloff
// Ranged attacks are equally effective at all ranges within their maximum
func (rtl *RangedThreatLayer) paintRangedPressure(
	squadID ecs.EntityID,
	center coords.LogicalPosition,
	maxRange int,
	pressure float64,
) {
	// Paint with no falloff and track positions for line-of-fire zones
	rtl.lineOfFireZones[squadID] = PaintThreatToMapWithTracking(
		rtl.pressureByPos,
		center,
		maxRange,
		pressure,
		NoFalloff,
	)
}

// Query API methods

// GetRangedPressureAt returns ranged pressure at a position
func (rtl *RangedThreatLayer) GetRangedPressureAt(pos coords.LogicalPosition) float64 {
	return rtl.pressureByPos[pos]
}

// GetRangedThreatsToPosition returns all squads that threaten a position
func (rtl *RangedThreatLayer) GetRangedThreatsToPosition(pos coords.LogicalPosition) []ecs.EntityID {
	var threats []ecs.EntityID

	for squadID, zone := range rtl.lineOfFireZones {
		for _, zonePos := range zone {
			if zonePos.X == pos.X && zonePos.Y == pos.Y {
				threats = append(threats, squadID)
				break
			}
		}
	}

	return threats
}

// IsInRangedZone checks if position is under ranged fire
func (rtl *RangedThreatLayer) IsInRangedZone(pos coords.LogicalPosition) bool {
	return rtl.pressureByPos[pos] > 0.0
}
