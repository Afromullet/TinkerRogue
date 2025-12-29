package behavior

import (
	"game_main/common"
	"game_main/tactical/combat"

	"github.com/bytearena/ecs"
)

// ThreatLayer is the base interface for all threat computation layers
// Each layer computes one aspect of tactical threat (melee, ranged, support, positional)
type ThreatLayer interface {
	// Compute recalculates threat values for all positions
	// Called once per AI turn before action evaluation
	Compute()

	// IsValid checks if layer data is current (not dirty)
	IsValid(currentRound int) bool

	// MarkDirty forces recomputation on next Compute()
	MarkDirty()
}

// ThreatLayerBase provides common functionality for all layers
// Uses composition pattern - embed this in concrete layers
type ThreatLayerBase struct {
	manager         *common.EntityManager
	cache           *combat.CombatQueryCache
	factionID       ecs.EntityID // The faction viewing this threat layer
	lastUpdateRound int          // Round number when last computed
	isDirty         bool         // Forces recomputation if true
}

// NewThreatLayerBase creates a new base layer with common dependencies
func NewThreatLayerBase(
	factionID ecs.EntityID,
	manager *common.EntityManager,
	cache *combat.CombatQueryCache,
) *ThreatLayerBase {
	return &ThreatLayerBase{
		manager:         manager,
		cache:           cache,
		factionID:       factionID,
		lastUpdateRound: -1,
		isDirty:         true,
	}
}

// IsValid checks if layer data is still current
// Returns false if round changed or layer marked dirty
func (tlb *ThreatLayerBase) IsValid(currentRound int) bool {
	return !tlb.isDirty && tlb.lastUpdateRound == currentRound
}

// MarkDirty forces layer to recompute on next Compute() call
// Call when squad moves, is destroyed, or combat state changes
func (tlb *ThreatLayerBase) MarkDirty() {
	tlb.isDirty = true
}

// markClean updates the layer state after successful computation
// Called internally by concrete layer Compute() methods
func (tlb *ThreatLayerBase) markClean(currentRound int) {
	tlb.isDirty = false
	tlb.lastUpdateRound = currentRound
}

// getEnemyFactions returns all factions hostile to this layer's faction
// For now, returns all factions except the viewing faction
// TODO: In future, support faction alliances and complex relationships
func (tlb *ThreatLayerBase) getEnemyFactions() []ecs.EntityID {
	var enemies []ecs.EntityID
	allFactions := combat.GetAllFactions(tlb.manager)

	for _, factionID := range allFactions {
		if factionID != tlb.factionID {
			enemies = append(enemies, factionID)
		}
	}

	return enemies
}
