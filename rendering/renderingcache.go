package rendering

import (
	"game_main/common"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
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

	// Sprite Batching (for performance optimization)
	spriteBatches map[*ebiten.Image]*SpriteBatch // Batches grouped by image
}

// NewRenderingCache creates a cache with new ECS Views
// Call this during game initialization
func NewRenderingCache(manager *common.EntityManager) *RenderingCache {
	return &RenderingCache{
		// Create View - one-time O(n) cost
		// View is automatically maintained when RenderableComponent added/removed
		RenderablesView: manager.World.CreateView(RenderablesTag),

		// Pre-allocate sprite batch map (typical games have 5-20 unique sprite images)
		spriteBatches: make(map[*ebiten.Image]*SpriteBatch, 20),
	}
}

// GetOrCreateSpriteBatch returns the batch for an image, creating one if needed
func (rc *RenderingCache) GetOrCreateSpriteBatch(image *ebiten.Image) *SpriteBatch {
	batch, exists := rc.spriteBatches[image]
	if !exists {
		batch = NewSpriteBatch(image)
		rc.spriteBatches[image] = batch
	}
	return batch
}

// ClearSpriteBatches resets all sprite batches for the next frame
// Call this at the beginning of each render frame
func (rc *RenderingCache) ClearSpriteBatches() {
	for _, batch := range rc.spriteBatches {
		batch.Reset()
	}
}

// DrawSpriteBatches renders all sprite batches to the screen
// Call this after collecting all sprites for the frame
func (rc *RenderingCache) DrawSpriteBatches(screen *ebiten.Image) {
	for _, batch := range rc.spriteBatches {
		batch.Draw(screen)
	}
}
