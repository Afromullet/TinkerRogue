package behavior

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// ThreatType distinguishes between melee and ranged threats
type ThreatType int

const (
	ThreatTypeMelee ThreatType = iota
	ThreatTypeRanged
)

// CombatThreatLayer computes both melee and ranged threats from enemy squads.
// This is a unified layer that replaces the separate MeleeThreatLayer and RangedThreatLayer.
//
// Melee threat: DangerByRange[1] * LinearFalloff over (MoveSpeed + AttackRange)
// Ranged threat: DangerByRange[maxRange] * NoFalloff over maxRange
type CombatThreatLayer struct {
	*ThreatLayerBase

	// Melee threat data
	meleeThreatByPos   map[coords.LogicalPosition]float64 // Position -> melee threat value
	meleeThreatBySquad map[ecs.EntityID]float64           // Squad -> total melee threat emitted
	meleeEffectiveRange map[ecs.EntityID]int              // Squad -> max melee engagement range

	// Ranged threat data
	rangedPressureByPos map[coords.LogicalPosition]float64         // Position -> ranged pressure
	lineOfFireZones     map[ecs.EntityID][]coords.LogicalPosition  // Squad -> threatened positions

	// Dependencies
	baseThreatMgr *FactionThreatLevelManager
}

// NewCombatThreatLayer creates a new unified combat threat layer for a faction
func NewCombatThreatLayer(
	factionID ecs.EntityID,
	manager *common.EntityManager,
	cache *combat.CombatQueryCache,
	baseThreatMgr *FactionThreatLevelManager,
) *CombatThreatLayer {
	return &CombatThreatLayer{
		ThreatLayerBase:     NewThreatLayerBase(factionID, manager, cache),
		meleeThreatByPos:    make(map[coords.LogicalPosition]float64),
		meleeThreatBySquad:  make(map[ecs.EntityID]float64),
		meleeEffectiveRange: make(map[ecs.EntityID]int),
		rangedPressureByPos: make(map[coords.LogicalPosition]float64),
		lineOfFireZones:     make(map[ecs.EntityID][]coords.LogicalPosition),
		baseThreatMgr:       baseThreatMgr,
	}
}

// Compute recalculates both melee and ranged threat for all enemy squads
func (ctl *CombatThreatLayer) Compute() {
	// Clear existing data (reuse maps to reduce GC pressure)
	clear(ctl.meleeThreatByPos)
	clear(ctl.meleeThreatBySquad)
	clear(ctl.meleeEffectiveRange)
	clear(ctl.rangedPressureByPos)
	clear(ctl.lineOfFireZones)

	enemyFactions := ctl.getEnemyFactions()

	for _, enemyFactionID := range enemyFactions {
		squadIDs := combat.GetActiveSquadsForFaction(enemyFactionID, ctl.manager)

		for _, squadID := range squadIDs {
			squadPos, err := combat.GetSquadMapPosition(squadID, ctl.manager)
			if err != nil {
				continue
			}

			// Get threat data from base threat system
			factionThreat, exists := ctl.baseThreatMgr.factions[enemyFactionID]
			if !exists {
				continue
			}
			squadThreat, exists := factionThreat.squadDangerLevel[squadID]
			if !exists {
				continue
			}

			// Compute melee threat if squad has melee units
			if hasUnitsWithAttackType(squadID, ctl.manager, MeleeAttackTypes) {
				ctl.computeMeleeThreat(squadID, squadPos, squadThreat)
			}

			// Compute ranged threat if squad has ranged units
			if hasUnitsWithAttackType(squadID, ctl.manager, RangedAttackTypes) {
				ctl.computeRangedThreat(squadID, squadPos, squadThreat)
			}
		}
	}

	ctl.markClean(0)
}

// computeMeleeThreat computes melee threat for a single squad
func (ctl *CombatThreatLayer) computeMeleeThreat(
	squadID ecs.EntityID,
	squadPos coords.LogicalPosition,
	squadThreat *SquadThreatLevel,
) {
	moveSpeed := squads.GetSquadMovementSpeed(squadID, ctl.manager)
	maxMeleeRange := getMaxRangeForAttackTypes(squadID, ctl.manager, MeleeAttackTypes, 1)
	threatRadius := moveSpeed + maxMeleeRange

	// Use danger at range 1 (melee range) - already includes role multipliers
	totalThreat := squadThreat.DangerByRange[1]

	// Store squad data
	ctl.meleeThreatBySquad[squadID] = totalThreat
	ctl.meleeEffectiveRange[squadID] = threatRadius

	// Paint threat on map with linear falloff
	PaintThreatToMap(ctl.meleeThreatByPos, squadPos, threatRadius, totalThreat, LinearFalloff, false)
}

// computeRangedThreat computes ranged threat for a single squad
func (ctl *CombatThreatLayer) computeRangedThreat(
	squadID ecs.EntityID,
	squadPos coords.LogicalPosition,
	squadThreat *SquadThreatLevel,
) {
	// Get max ranged attack range
	maxRange := getMaxRangeForAttackTypes(squadID, ctl.manager, RangedAttackTypes, 3)

	// Use danger at max range - already includes role multipliers
	rangedDanger := squadThreat.DangerByRange[maxRange]

	// Paint ranged pressure with no falloff and track positions for line-of-fire zones
	ctl.lineOfFireZones[squadID] = PaintThreatToMap(
		ctl.rangedPressureByPos,
		squadPos,
		maxRange,
		rangedDanger,
		NoFalloff,
		true, // Track positions for line-of-fire zones
	)
}

// =========================================
// Melee Query API (backward compatible)
// =========================================

// GetMeleeThreatAt returns melee threat value at a position
func (ctl *CombatThreatLayer) GetMeleeThreatAt(pos coords.LogicalPosition) float64 {
	return ctl.meleeThreatByPos[pos]
}

// GetMeleeThreatFrom returns total melee threat emitted by a squad
func (ctl *CombatThreatLayer) GetMeleeThreatFrom(squadID ecs.EntityID) float64 {
	return ctl.meleeThreatBySquad[squadID]
}

// IsInMeleeZone checks if a position is within any melee threat zone
func (ctl *CombatThreatLayer) IsInMeleeZone(pos coords.LogicalPosition) bool {
	return ctl.meleeThreatByPos[pos] > 0.0
}

// =========================================
// Ranged Query API (backward compatible)
// =========================================

// GetRangedPressureAt returns ranged pressure at a position
func (ctl *CombatThreatLayer) GetRangedPressureAt(pos coords.LogicalPosition) float64 {
	return ctl.rangedPressureByPos[pos]
}

// GetRangedThreatsToPosition returns all squads that threaten a position with ranged attacks
func (ctl *CombatThreatLayer) GetRangedThreatsToPosition(pos coords.LogicalPosition) []ecs.EntityID {
	var threats []ecs.EntityID

	for squadID, zone := range ctl.lineOfFireZones {
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
func (ctl *CombatThreatLayer) IsInRangedZone(pos coords.LogicalPosition) bool {
	return ctl.rangedPressureByPos[pos] > 0.0
}

// =========================================
// Combined Query API
// =========================================

// GetCombinedThreatAt returns total threat (melee + ranged) at a position
func (ctl *CombatThreatLayer) GetCombinedThreatAt(pos coords.LogicalPosition) float64 {
	return ctl.meleeThreatByPos[pos] + ctl.rangedPressureByPos[pos]
}

// IsInAnyThreatZone checks if position is threatened by either melee or ranged
func (ctl *CombatThreatLayer) IsInAnyThreatZone(pos coords.LogicalPosition) bool {
	return ctl.IsInMeleeZone(pos) || ctl.IsInRangedZone(pos)
}
