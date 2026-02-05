package behavior

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// SupportValueLayer computes value of support actions (healing, buffs) at positions
// High values indicate positions where support squads can be most effective
type SupportValueLayer struct {
	*ThreatLayerBase

	// Core support data
	healPriority    map[ecs.EntityID]float64           // Squad -> heal urgency (0-1)
	buffPriority    map[ecs.EntityID]float64           // Squad -> buff value
	supportValuePos map[coords.LogicalPosition]float64 // Position -> support value
	allyProximity   map[coords.LogicalPosition]int     // Position -> count of nearby allies

	// Dependencies
	baseThreatMgr *FactionThreatLevelManager
}

// NewSupportValueLayer creates a new support value layer for a faction
func NewSupportValueLayer(
	factionID ecs.EntityID,
	manager *common.EntityManager,
	cache *combat.CombatQueryCache,
	baseThreatMgr *FactionThreatLevelManager,
) *SupportValueLayer {
	return &SupportValueLayer{
		ThreatLayerBase: NewThreatLayerBase(factionID, manager, cache),
		healPriority:    make(map[ecs.EntityID]float64),
		buffPriority:    make(map[ecs.EntityID]float64),
		supportValuePos: make(map[coords.LogicalPosition]float64),
		allyProximity:   make(map[coords.LogicalPosition]int),
		baseThreatMgr:   baseThreatMgr,
	}
}

// Compute recalculates support value for all allied squads
// Wounded allies create high support value in nearby positions
func (svl *SupportValueLayer) Compute() {
	// Clear existing data (reuse maps to reduce GC pressure)
	clear(svl.healPriority)
	clear(svl.buffPriority)
	clear(svl.supportValuePos)
	clear(svl.allyProximity)

	// Get all allied squads
	squadIDs := combat.GetActiveSquadsForFaction(svl.factionID, svl.manager)

	for _, squadID := range squadIDs {
		// Calculate heal priority (inverse of health percentage)
		// Use centralized squad health calculation
		avgHP := squads.GetSquadHealthPercent(squadID, svl.manager)
		svl.healPriority[squadID] = 1.0 - avgHP

		// Calculate buff priority based on engagement state
		svl.buffPriority[squadID] = svl.calculateBuffPriority(squadID)

		// Get squad position
		squadPos, err := combat.GetSquadMapPosition(squadID, svl.manager)
		if err != nil {
			continue
		}

		// Paint support value around squad position
		// Higher heal priority = higher support value radiating from that position
		healPriority := svl.healPriority[squadID]
		svl.paintSupportValue(squadPos, healPriority)
	}

	svl.markClean(0)
}

// NOTE: calculateAverageHP has been moved to squads.GetSquadHealthPercent()
// This centralizes health calculations and eliminates code duplication.

// calculateBuffPriority returns priority for buffing this squad
// Higher if squad is about to engage or is in active combat
func (svl *SupportValueLayer) calculateBuffPriority(squadID ecs.EntityID) float64 {
	// Get distance tracker for this squad
	factionThreat, exists := svl.baseThreatMgr.factions[svl.factionID]
	if !exists {
		return 0.0
	}

	squadThreat, exists := factionThreat.squadThreatLevels[squadID]
	if !exists || squadThreat.SquadDistances == nil {
		return 0.0
	}

	// Check if enemy is within engagement range
	// Buff priority increases as enemies get closer
	_, _, buffRange := GetSupportLayerParams()
	for distance := 1; distance <= buffRange; distance++ {
		if enemies, exists := squadThreat.SquadDistances.EnemiesByDistance[distance]; exists && len(enemies) > 0 {
			// Closer enemies = higher buff priority
			return 1.0 - (float64(distance) / float64(buffRange+1))
		}
	}

	return 0.1 // Default low priority if no nearby enemies
}

// paintSupportValue paints support value around a position
// Support value radiates from wounded squads (healers want to be near them)
func (svl *SupportValueLayer) paintSupportValue(
	center coords.LogicalPosition,
	healPriority float64,
) {
	// Paint support value with linear falloff using configured radius
	healRadius, proximityRadius, _ := GetSupportLayerParams()
	PaintThreatToMap(svl.supportValuePos, center, healRadius, healPriority, LinearFalloff, false)

	// Track ally proximity separately
	for dx := -proximityRadius; dx <= proximityRadius; dx++ {
		for dy := -proximityRadius; dy <= proximityRadius; dy++ {
			pos := coords.LogicalPosition{X: center.X + dx, Y: center.Y + dy}
			if center.ChebyshevDistance(&pos) <= proximityRadius {
				svl.allyProximity[pos]++
			}
		}
	}
}

// Query API methods

// GetSupportValueAt returns support value at a position
// Higher values indicate better positions for support squads
func (svl *SupportValueLayer) GetSupportValueAt(pos coords.LogicalPosition) float64 {
	return svl.supportValuePos[pos]
}

// GetAllyProximityAt returns number of nearby allies at a position
func (svl *SupportValueLayer) GetAllyProximityAt(pos coords.LogicalPosition) int {
	return svl.allyProximity[pos]
}

// GetMostDamagedAlly returns the squad with highest heal priority
// Returns 0 if no damaged allies found
func (svl *SupportValueLayer) GetMostDamagedAlly() ecs.EntityID {
	var mostDamaged ecs.EntityID
	highestPriority := 0.0

	for squadID, priority := range svl.healPriority {
		if priority > highestPriority {
			highestPriority = priority
			mostDamaged = squadID
		}
	}

	return mostDamaged
}

// GetAlliesInHealRange returns all allied squads within healing range of a position
func (svl *SupportValueLayer) GetAlliesInHealRange(
	pos coords.LogicalPosition,
	healRange int,
) []ecs.EntityID {
	var allies []ecs.EntityID

	squadIDs := combat.GetActiveSquadsForFaction(svl.factionID, svl.manager)

	for _, squadID := range squadIDs {
		squadPos, err := combat.GetSquadMapPosition(squadID, svl.manager)
		if err != nil {
			continue
		}

		distance := pos.ChebyshevDistance(&squadPos)
		if distance <= healRange {
			allies = append(allies, squadID)
		}
	}

	return allies
}
