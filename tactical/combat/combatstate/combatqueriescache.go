package combatstate

import (
	"game_main/core/common"

	"github.com/bytearena/ecs"
)

// CombatQueryCache provides cached access to combat-related queries using ECS Views.
// Views are automatically maintained by the ECS library when components are added/removed.
//
// Faction lookups use the package-level factionView (combatcomponents.go) so we keep
// a single canonical view per tag rather than duplicating across cache instances.
type CombatQueryCache struct {
	ActionStateView *ecs.View // All ActionStateTag entities
}

// NewCombatQueryCache creates a cache with new ECS Views
func NewCombatQueryCache(manager *common.EntityManager) *CombatQueryCache {
	return &CombatQueryCache{
		ActionStateView: manager.World.CreateView(ActionStateTag),
	}
}

// ========================================
// CACHED ACTION STATE QUERIES
// ========================================

// FindActionStateEntity finds ActionStateData for a squad using cached view
func (c *CombatQueryCache) FindActionStateEntity(squadID ecs.EntityID) *ecs.Entity {
	for _, result := range c.ActionStateView.Get() {
		actionState := common.GetComponentType[*ActionStateData](result.Entity, ActionStateComponent)
		if actionState != nil && actionState.SquadID == squadID {
			return result.Entity
		}
	}
	return nil
}

// FindActionStateBySquadID returns ActionStateData for a squad using cached view
func (c *CombatQueryCache) FindActionStateBySquadID(squadID ecs.EntityID) *ActionStateData {
	for _, result := range c.ActionStateView.Get() {
		actionState := common.GetComponentType[*ActionStateData](result.Entity, ActionStateComponent)
		if actionState != nil && actionState.SquadID == squadID {
			return actionState
		}
	}
	return nil
}

// ========================================
// CACHED FACTION QUERIES
// ========================================

// FindFactionByID finds a faction entity by faction ID using the package-level view.
func (c *CombatQueryCache) FindFactionByID(factionID ecs.EntityID) *ecs.Entity {
	for _, result := range factionView.Get() {
		faction := result.Entity
		factionData := common.GetComponentType[*FactionData](faction, CombatFactionComponent)
		if factionData != nil && factionData.FactionID == factionID {
			return faction
		}
	}
	return nil
}

// FindFactionDataByID returns FactionData for a faction ID using the package-level view.
func (c *CombatQueryCache) FindFactionDataByID(factionID ecs.EntityID) *FactionData {
	for _, result := range factionView.Get() {
		factionData := common.GetComponentType[*FactionData](result.Entity, CombatFactionComponent)
		if factionData != nil && factionData.FactionID == factionID {
			return factionData
		}
	}
	return nil
}
