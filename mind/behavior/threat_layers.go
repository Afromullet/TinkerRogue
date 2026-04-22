package behavior

import (
	"game_main/core/common"
	"game_main/tactical/combat/combatstate"

	"github.com/bytearena/ecs"
)

// ThreatLayerBase provides common functionality for all layers
// Uses composition pattern - embed this in concrete layers
type ThreatLayerBase struct {
	manager   *common.EntityManager
	cache     *combatstate.CombatQueryCache
	factionID ecs.EntityID // The faction viewing this threat layer
}

// NewThreatLayerBase creates a new base layer with common dependencies
func NewThreatLayerBase(
	factionID ecs.EntityID,
	manager *common.EntityManager,
	cache *combatstate.CombatQueryCache,
) *ThreatLayerBase {
	return &ThreatLayerBase{
		manager:   manager,
		cache:     cache,
		factionID: factionID,
	}
}

// getEnemyFactions returns all factions hostile to this layer's faction
// For now, returns all factions except the viewing faction
func (tlb *ThreatLayerBase) getEnemyFactions() []ecs.EntityID {
	var enemies []ecs.EntityID
	allFactions := combatstate.GetAllFactions(tlb.manager)

	for _, factionID := range allFactions {
		if factionID != tlb.factionID {
			enemies = append(enemies, factionID)
		}
	}

	return enemies
}
