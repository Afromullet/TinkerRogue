package rendering

import (
	"game_main/common"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// RenderingCache provides cached access to renderable entities using ECS Views
// Views are automatically maintained by the ECS library when components are added/removed
type RenderingCache struct {
	// ECS Views (automatically maintained by ECS library)
	RenderablesView *ecs.View // All RenderablesTag entities

	// Sprite Batching (for performance optimization)
	spriteBatches map[*ebiten.Image]*QuadBatch // Batches grouped by image
}

// NewRenderingCache creates a cache with new ECS Views
// Call this during game initialization
func NewRenderingCache(manager *common.EntityManager) *RenderingCache {
	return &RenderingCache{
	
		// View is automatically maintained when RenderableComponent added/removed
		RenderablesView: manager.World.CreateView(RenderablesTag),

		// Pre-allocate sprite batch map (typical games have 5-20 unique sprite images)
		spriteBatches: make(map[*ebiten.Image]*QuadBatch, 20),
	}
}

// GetOrCreateSpriteBatch returns the batch for an image, creating one if needed
func (rc *RenderingCache) GetOrCreateSpriteBatch(image *ebiten.Image) *QuadBatch {
	batch, exists := rc.spriteBatches[image]
	if !exists {
		batch = NewQuadBatch(image, SpriteVerticesBatchSize, SpriteIndicesBatchSize)
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

// RefreshRenderablesView recreates the RenderablesView to force it to update
// Call this after batch entity disposal to ensure stale entities don't render
func (rc *RenderingCache) RefreshRenderablesView(manager *common.EntityManager) {
	rc.RenderablesView = manager.World.CreateView(RenderablesTag)
}
