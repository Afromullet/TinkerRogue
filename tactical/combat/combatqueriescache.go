package combat

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// CombatQueryCache provides cached access to combat-related queries using ECS Views
// Views are automatically maintained by the ECS library when components are added/removed
type CombatQueryCache struct {
	// ECS Views (automatically maintained by ECS library)
	ActionStateView *ecs.View // All ActionStateTag entities
	FactionView     *ecs.View // All FactionTag entities
}

// NewCombatQueryCache creates a cache with new ECS Views
func NewCombatQueryCache(manager *common.EntityManager) *CombatQueryCache {
	return &CombatQueryCache{
		// Create Views - one-time O(n) cost per View
		// Views are automatically maintained when components are added/removed
		ActionStateView: manager.World.CreateView(ActionStateTag),
		FactionView:     manager.World.CreateView(FactionTag),
	}
}

// ========================================
// CACHED ACTION STATE QUERIES
// ========================================

// FindActionStateEntity finds ActionStateData for a squad using cached view
func (c *CombatQueryCache) FindActionStateEntity(squadID ecs.EntityID) *ecs.Entity {
	// Iterate through cached view results (not full World.Query)
	// View automatically updated when ActionStateComponent added/removed
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

// FindFactionByID finds a faction entity by faction ID using cached view
func (c *CombatQueryCache) FindFactionByID(factionID ecs.EntityID) *ecs.Entity {
	// Iterate through cached view results (not full World.Query)
	// View automatically updated when FactionComponent added/removed
	for _, result := range c.FactionView.Get() {
		faction := result.Entity
		factionData := common.GetComponentType[*FactionData](faction, CombatFactionComponent)
		if factionData != nil && factionData.FactionID == factionID {
			return faction
		}
	}
	return nil
}

// FindFactionDataByID returns FactionData for a faction ID using cached view
func (c *CombatQueryCache) FindFactionDataByID(factionID ecs.EntityID) *FactionData {
	for _, result := range c.FactionView.Get() {
		factionData := common.GetComponentType[*FactionData](result.Entity, CombatFactionComponent)
		if factionData != nil && factionData.FactionID == factionID {
			return factionData
		}
	}
	return nil
}
