// Package rendering handles the display and drawing of game entities to the screen.
// It processes renderable components, manages sprite drawing, and coordinates
// with the graphics package to render entities at their correct screen positions.
package rendering

import (
	"game_main/common"
	"game_main/visual/graphics"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

var (
	RenderableComponent *ecs.Component //Putting this here for now rather than in graphics
	RenderablesTag      ecs.Tag        // Tag for querying renderable entities
)

// init registers the rendering subsystem with the ECS component registry.
// This allows the rendering package to self-register its components without
// game_main needing to know about rendering internals.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		InitializeRenderingComponents(em)
		InitializeRenderingTags(em)
	})
}

// InitializeRenderingComponents registers rendering-related components.
func InitializeRenderingComponents(em *common.EntityManager) {
	RenderableComponent = em.World.NewComponent()
}

// InitializeRenderingTags creates tags for querying rendering-related entities.
func InitializeRenderingTags(em *common.EntityManager) {
	RenderablesTag = ecs.BuildTag(RenderableComponent, common.PositionComponent)
	em.WorldTags["renderables"] = RenderablesTag
}

type Renderable struct {
	Image   *ebiten.Image
	Visible bool
}

// Draw everything with a renderable component that's visible
// Uses sprite batching to reduce draw calls from hundreds to a few per frame
func ProcessRenderables(ecsmanager *common.EntityManager, gameMap worldmap.GameMap, screen *ebiten.Image, debugMode bool, cache *RenderingCache) {
	// Clear sprite batches from previous frame
	cache.ClearSpriteBatches()

	// Collect all sprites into batches (grouped by image)
	for _, result := range cache.RenderablesView.Get() {
		pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
		renderable := common.GetComponentType[*Renderable](result.Entity, RenderableComponent)
		img := renderable.Image

		if !renderable.Visible || img == nil {
			continue
		}

		logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)
		tile := gameMap.Tiles[index]

		// Get sprite dimensions
		bounds := img.Bounds()
		srcX := float32(bounds.Min.X)
		srcY := float32(bounds.Min.Y)
		srcW := float32(bounds.Dx())
		srcH := float32(bounds.Dy())

		// Destination position and size (no scaling in this mode)
		dstX := float32(tile.PixelX)
		dstY := float32(tile.PixelY)
		dstW := srcW
		dstH := srcH

		// Add sprite to batch (grouped by image)
		batch := cache.GetOrCreateSpriteBatch(img)
		batch.AddSprite(dstX, dstY, srcX, srcY, srcW, srcH, dstW, dstH, 1.0, 1.0, 1.0, 1.0)
	}

	// Draw all batches in a single pass
	cache.DrawSpriteBatches(screen)
}

// ProcessRenderablesInSquare renders entities in a square region around playerPos
func ProcessRenderablesInSquare(ecsmanager *common.EntityManager, gameMap worldmap.GameMap, screen *ebiten.Image, playerPos *coords.LogicalPosition, squareSize int, debugMode bool, cache *RenderingCache) {
	// Clear sprite batches from previous frame
	cache.ClearSpriteBatches()

	// Calculate the starting and ending coordinates of the square
	sq := coords.NewDrawableSection(playerPos.X, playerPos.Y, squareSize)

	// Collect all sprites into batches (grouped by image)
	for _, result := range cache.RenderablesView.Get() {
		pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
		renderable := common.GetComponentType[*Renderable](result.Entity, RenderableComponent)
		img := renderable.Image

		if !renderable.Visible || img == nil {
			continue
		}

		// Check if the entity's position is within the square bounds
		if pos.X >= sq.StartX && pos.X <= sq.EndX && pos.Y >= sq.StartY && pos.Y <= sq.EndY {
			logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}

			// Use unified coordinate transformation - handles scrolling mode and viewport centering automatically
			screenX, screenY := coords.CoordManager.LogicalToScreen(logicalPos, playerPos)

			// Get sprite dimensions
			bounds := img.Bounds()
			srcX := float32(bounds.Min.X)
			srcY := float32(bounds.Min.Y)
			srcW := float32(bounds.Dx())
			srcH := float32(bounds.Dy())

			// Apply scaling to destination size
			scale := float32(graphics.ScreenInfo.ScaleFactor)
			dstX := float32(screenX)
			dstY := float32(screenY)
			dstW := srcW * scale
			dstH := srcH * scale

			// Add sprite to batch (grouped by image)
			batch := cache.GetOrCreateSpriteBatch(img)
			batch.AddSprite(dstX, dstY, srcX, srcY, srcW, srcH, dstW, dstH, 1.0, 1.0, 1.0, 1.0)
		}
	}

	// Draw all batches in a single pass
	cache.DrawSpriteBatches(screen)
}
