package rendering

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// RenderingCache provides cached access to renderable entities using ECS Views
// Views are automatically maintained by the ECS library when components are added/removed
//
// Performance Impact:
// - ProcessRenderables: O(n) World.Query every frame → O(k) view iteration (k = renderables, typically 100-500)
// - ProcessRenderablesInSquare: O(n) World.Query every frame → O(k) view iteration
// - Expected: 3-5x faster per frame (scanning 10,000+ entities → iterating 100-500 cached results)
//
// Key Benefits:
// - Views are automatically maintained (no manual invalidation needed)
// - Thread-safe (views have built-in RWMutex)
// - Zero per-frame memory allocations (views are persistent)
// - Single allocation per view, reused every frame
type RenderingCache struct {
	// ECS Views (automatically maintained by ECS library)
	RenderablesView *ecs.View // All RenderablesTag entities
}

// NewRenderingCache creates a cache with new ECS Views
// Call this during game initialization
func NewRenderingCache(manager *common.EntityManager) *RenderingCache {
	return &RenderingCache{
		// Create View - one-time O(n) cost
		// View is automatically maintained when RenderableComponent added/removed
		RenderablesView: manager.World.CreateView(RenderablesTag),
	}
}
