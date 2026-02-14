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
	supportValuePos map[coords.LogicalPosition]float64 // Position -> support value
	allyProximity   map[coords.LogicalPosition]int     // Position -> count of nearby allies
}

// NewSupportValueLayer creates a new support value layer for a faction
func NewSupportValueLayer(
	factionID ecs.EntityID,
	manager *common.EntityManager,
	cache *combat.CombatQueryCache,
) *SupportValueLayer {
	return &SupportValueLayer{
		ThreatLayerBase: NewThreatLayerBase(factionID, manager, cache),
		healPriority:    make(map[ecs.EntityID]float64),
		supportValuePos: make(map[coords.LogicalPosition]float64),
		allyProximity:   make(map[coords.LogicalPosition]int),
	}
}

// Compute recalculates support value for all allied squads
// Wounded allies create high support value in nearby positions
func (svl *SupportValueLayer) Compute(currentRound int) {
	// Clear existing data (reuse maps to reduce GC pressure)
	clear(svl.healPriority)
	clear(svl.supportValuePos)
	clear(svl.allyProximity)

	// Get all allied squads
	squadIDs := combat.GetActiveSquadsForFaction(svl.factionID, svl.manager)

	for _, squadID := range squadIDs {
		// Calculate heal priority (inverse of health percentage)
		// Use centralized squad health calculation
		avgHP := squads.GetSquadHealthPercent(squadID, svl.manager)
		svl.healPriority[squadID] = 1.0 - avgHP

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

	svl.markClean(currentRound)
}

// paintSupportValue paints support value around a position
// Support value radiates from wounded squads (healers want to be near them)
func (svl *SupportValueLayer) paintSupportValue(
	center coords.LogicalPosition,
	healPriority float64,
) {
	// Paint support value with linear falloff using configured radius
	healRadius, proximityRadius := GetSupportLayerParams()
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
