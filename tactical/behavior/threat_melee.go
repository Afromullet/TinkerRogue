package behavior

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// MeleeThreatLayer computes threat from enemy melee squads
// Threat = ExpectedDamageByRange[1] * RoleMultiplier * DistanceFalloff
// Melee squads need to move into position, so threat radius = MovementSpeed + AttackRange
type MeleeThreatLayer struct {
	*ThreatLayerBase

	// Core threat data (pure data, no logic)
	threatByPos    map[coords.LogicalPosition]float64 // Position -> threat value
	threatBySquad  map[ecs.EntityID]float64           // Squad -> total melee threat emitted
	effectiveRange map[ecs.EntityID]int               // Squad -> max melee engagement range

	// Dependencies
	baseThreatMgr *FactionThreatLevelManager
}

// NewMeleeThreatLayer creates a new melee threat layer for a faction
func NewMeleeThreatLayer(
	factionID ecs.EntityID,
	manager *common.EntityManager,
	cache *combat.CombatQueryCache,
	baseThreatMgr *FactionThreatLevelManager,
) *MeleeThreatLayer {
	return &MeleeThreatLayer{
		ThreatLayerBase: NewThreatLayerBase(factionID, manager, cache),
		threatByPos:     make(map[coords.LogicalPosition]float64),
		threatBySquad:   make(map[ecs.EntityID]float64),
		effectiveRange:  make(map[ecs.EntityID]int),
		baseThreatMgr:   baseThreatMgr,
	}
}

// Compute recalculates melee threat for all enemy squads
// Paints threat values on the map based on squad positions and capabilities
func (mtl *MeleeThreatLayer) Compute() {
	// Clear existing data (reuse maps to reduce GC pressure)
	clear(mtl.threatByPos)
	clear(mtl.threatBySquad)
	clear(mtl.effectiveRange)

	// Get all enemy factions
	enemyFactions := mtl.getEnemyFactions()

	for _, enemyFactionID := range enemyFactions {
		squadIDs := combat.GetSquadsForFaction(enemyFactionID, mtl.manager)

		for _, squadID := range squadIDs {
			// Skip if squad is destroyed
			if squads.IsSquadDestroyed(squadID, mtl.manager) {
				continue
			}

			// Check if squad has melee units
			if !hasUnitsWithAttackType(squadID, mtl.manager, MeleeAttackTypes) {
				continue
			}

			// Get squad position
			squadPos, err := combat.GetSquadMapPosition(squadID, mtl.manager)
			if err != nil {
				continue
			}

			// Calculate threat parameters
			moveSpeed := squads.GetSquadMovementSpeed(squadID, mtl.manager)
			maxMeleeRange := getMaxRangeForAttackTypes(squadID, mtl.manager, MeleeAttackTypes, 1)
			threatRadius := moveSpeed + maxMeleeRange

			// Get melee damage from base threat system
			factionThreat, exists := mtl.baseThreatMgr.factions[enemyFactionID]
			if !exists {
				continue
			}

			squadThreat, exists := factionThreat.squadDangerLevel[squadID]
			if !exists {
				continue
			}

			// Use damage at range 1 (melee range)
			meleeDamage := squadThreat.ExpectedDamageByRange[1]

			// Apply role modifier (tanks are more threatening due to durability)
			roleModifier := GetSquadRoleModifier(squadID, mtl.manager)
			totalThreat := meleeDamage * roleModifier

			// Store squad data
			mtl.threatBySquad[squadID] = totalThreat
			mtl.effectiveRange[squadID] = threatRadius

			// Paint threat on map
			mtl.paintThreatRadius(squadPos, threatRadius, totalThreat)
		}
	}

	// Mark as clean (round will be updated by Update() call)
	// We don't track rounds internally - that's handled by CompositeThreatEvaluator
	mtl.markClean(0)
}

// paintThreatRadius paints threat values onto the map with distance falloff
// Positions closer to the enemy squad have higher threat values
func (mtl *MeleeThreatLayer) paintThreatRadius(
	center coords.LogicalPosition,
	radius int,
	threat float64,
) {
	PaintThreatToMap(mtl.threatByPos, center, radius, threat, LinearFalloff)
}

// Query API methods

// GetMeleeThreatAt returns melee threat value at a position
func (mtl *MeleeThreatLayer) GetMeleeThreatAt(pos coords.LogicalPosition) float64 {
	return mtl.threatByPos[pos]
}

// GetMeleeThreatFrom returns total melee threat emitted by a squad
func (mtl *MeleeThreatLayer) GetMeleeThreatFrom(squadID ecs.EntityID) float64 {
	return mtl.threatBySquad[squadID]
}

// IsInMeleeZone checks if a position is within any melee threat zone
func (mtl *MeleeThreatLayer) IsInMeleeZone(pos coords.LogicalPosition) bool {
	return mtl.threatByPos[pos] > 0.0
}
