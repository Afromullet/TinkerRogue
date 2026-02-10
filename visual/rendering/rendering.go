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

const (
	TileBatchDefaultNumImages = 20
	TileVerticeBatchSize      = 800
	TileIndicesBatchSize      = 1200
	SpriteVerticesBatchSize   = 256
	SpriteIndicesBatchSize    = 384
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

// viewportParams holds viewport filtering/scaling parameters.
// nil means full-map mode (no filtering, no scaling).
type viewportParams struct {
	centerPos *coords.LogicalPosition
	section   coords.DrawableSection
}

// processRenderablesCore collects visible sprites into batches.
// When vp is nil, renders all entities at pixel positions with no scaling.
// When vp is non-nil, filters to viewport bounds and applies coordinate transform + scaling.
func processRenderablesCore(cache *RenderingCache, gameMap worldmap.GameMap, screen *ebiten.Image, vp *viewportParams) {
	cache.ClearSpriteBatches()

	for _, result := range cache.RenderablesView.Get() {
		pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
		renderable := common.GetComponentType[*Renderable](result.Entity, RenderableComponent)
		img := renderable.Image

		if !renderable.Visible || img == nil {
			continue
		}

		bounds := img.Bounds()
		srcX := float32(bounds.Min.X)
		srcY := float32(bounds.Min.Y)
		srcW := float32(bounds.Dx())
		srcH := float32(bounds.Dy())

		var dstX, dstY, dstW, dstH float32

		if vp != nil {
			// Viewport mode: bounds check + scaled position
			if pos.X < vp.section.StartX || pos.X > vp.section.EndX ||
				pos.Y < vp.section.StartY || pos.Y > vp.section.EndY {
				continue
			}
			logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
			screenX, screenY := coords.CoordManager.LogicalToScreen(logicalPos, vp.centerPos)
			scale := float32(graphics.ScreenInfo.ScaleFactor)
			dstX = float32(screenX)
			dstY = float32(screenY)
			dstW = srcW * scale
			dstH = srcH * scale
		} else {
			// Full map mode: direct pixel position, no scaling
			logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)
			tile := gameMap.Tiles[index]
			dstX = float32(tile.PixelX)
			dstY = float32(tile.PixelY)
			dstW = srcW
			dstH = srcH
		}

		batch := cache.GetOrCreateSpriteBatch(img)
		batch.Add(dstX, dstY, srcX, srcY, srcW, srcH, dstW, dstH, 1.0, 1.0, 1.0, 1.0)
	}

	cache.DrawSpriteBatches(screen)
}

// ProcessRenderables draws all visible renderable entities (full map, no viewport).
func ProcessRenderables(gameMap worldmap.GameMap, screen *ebiten.Image, cache *RenderingCache) {
	processRenderablesCore(cache, gameMap, screen, nil)
}

// ProcessRenderablesInSquare renders entities in a square region around playerPos.
func ProcessRenderablesInSquare(gameMap worldmap.GameMap, screen *ebiten.Image, playerPos *coords.LogicalPosition, squareSize int, cache *RenderingCache) {
	sq := coords.NewDrawableSection(playerPos.X, playerPos.Y, squareSize)
	processRenderablesCore(cache, gameMap, screen, &viewportParams{
		centerPos: playerPos,
		section:   sq,
	})
}
