package squads

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// SquadQueryCache provides cached access to squad-related queries using ECS Views
// Views are automatically maintained by the ECS library when components are added/removed
type SquadQueryCache struct {
	// ECS Views (automatically maintained by ECS library)
	// Exported so they can be accessed by other systems (e.g., BuildSquadInfoCache)
	SquadView       *ecs.View // All SquadTag entities
	SquadMemberView *ecs.View // All SquadMemberTag entities
	LeaderView      *ecs.View // All LeaderTag entities
}

// NewSquadQueryCache creates a cache with new ECS Views
// Use this to create a standalone squad query cache
func NewSquadQueryCache(manager *common.EntityManager) *SquadQueryCache {
	return &SquadQueryCache{
		// Views are automatically maintained when components are added/removed
		SquadView:       manager.World.CreateView(SquadTag),
		SquadMemberView: manager.World.CreateView(SquadMemberTag),
		LeaderView:      manager.World.CreateView(LeaderTag),
	}
}

// ========================================
// CACHED QUERY FUNCTIONS
// ========================================
// These functions use ECS Views for O(k) performance where k = number of entities with the component.
// Prefer these over the non-cached versions in squadqueries.go when you have access to a SquadQueryCache.

// GetSquadEntity finds squad entity by ID using cached view (O(k) where k = squad count)
func (c *SquadQueryCache) GetSquadEntity(squadID ecs.EntityID) *ecs.Entity {
	// Iterate through cached view results (not full World.Query)
	// View automatically updated when SquadComponent added/removed from entities
	for _, result := range c.SquadView.Get() {
		squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
		if squadData != nil && squadData.SquadID == squadID {
			return result.Entity
		}
	}
	return nil
}

// GetUnitIDsInSquad returns unit IDs belonging to a squad using cached view
func (c *SquadQueryCache) GetUnitIDsInSquad(squadID ecs.EntityID) []ecs.EntityID {
	var unitIDs []ecs.EntityID

	// Iterate through cached view results
	// View automatically updated when SquadMemberComponent added/removed
	for _, result := range c.SquadMemberView.Get() {
		memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
		if memberData != nil && memberData.SquadID == squadID {
			unitIDs = append(unitIDs, result.Entity.GetID())
		}
	}

	return unitIDs
}

// GetLeaderID finds the leader unit ID of a squad using cached view
func (c *SquadQueryCache) GetLeaderID(squadID ecs.EntityID) ecs.EntityID {
	// Iterate through cached view results
	// View automatically updated when LeaderTag added/removed from entities
	for _, result := range c.LeaderView.Get() {
		memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
		if memberData != nil && memberData.SquadID == squadID {
			return result.Entity.GetID()
		}
	}

	return 0 // Returns 0 if not found
}

// GetSquadName returns the squad name using cached view
// Wraps GetSquadEntity with component data access
func (c *SquadQueryCache) GetSquadName(squadID ecs.EntityID) string {
	squadEntity := c.GetSquadEntity(squadID)
	if squadEntity == nil {
		return "Unknown Squad"
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	if squadData == nil {
		return "Unknown Squad"
	}

	return squadData.Name
}

// FindAllSquads returns all squad IDs using cached view
func (c *SquadQueryCache) FindAllSquads() []ecs.EntityID {
	allSquads := make([]ecs.EntityID, 0)

	for _, result := range c.SquadView.Get() {
		squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
		if squadData != nil {
			allSquads = append(allSquads, squadData.SquadID)
		}
	}

	return allSquads
}
