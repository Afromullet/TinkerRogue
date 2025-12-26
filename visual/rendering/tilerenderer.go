package rendering

import (
	"game_main/world/coords"
	"game_main/world/worldmap"
	"game_main/visual/graphics"

	"github.com/hajimehoshi/ebiten/v2"
)

// TileRenderer handles rendering of map tiles with batching for performance
type TileRenderer struct {
	tiles      []*worldmap.Tile
	colorScale ebiten.ColorScale
	drawOpts   ebiten.DrawImageOptions // Reusable draw options (eliminates 2,000 allocations/frame)
	batches    map[*ebiten.Image]*TileBatch // Batches tiles by image for efficient rendering

	// Cache state to avoid rebuilding batches every frame
	lastCenterX    int
	lastCenterY    int
	lastViewportSize int
	batchesBuilt   bool
}

// NewTileRenderer creates a renderer for the given tileset
func NewTileRenderer(tiles []*worldmap.Tile) *TileRenderer {
	return &TileRenderer{
		tiles:   tiles,
		batches: make(map[*ebiten.Image]*TileBatch, 20), // Pre-allocate for ~20 unique images
	}
}

// RenderOptions configures the rendering behavior
type RenderOptions struct {
	RevealAll    bool
	CenterOn     *coords.LogicalPosition // nil for full map
	ViewportSize int
	Screen       *ebiten.Image
}

// RenderedBounds tracks what was drawn and edge information
type RenderedBounds struct {
	MinX, MaxX int
	MinY, MaxY int
	RightEdgeX int
	RightEdgeY int
}

// Render draws tiles to screen based on options using batching for performance
func (r *TileRenderer) Render(opts RenderOptions) RenderedBounds {
	bounds := r.calculateBounds(opts)
	bounds.RightEdgeX = 0
	bounds.RightEdgeY = 0

	// Check if we need to rebuild batches (only rebuild when viewport changes or first render)
	needsRebuild := !r.batchesBuilt
	if opts.CenterOn != nil {
		if r.lastCenterX != opts.CenterOn.X || r.lastCenterY != opts.CenterOn.Y || r.lastViewportSize != opts.ViewportSize {
			needsRebuild = true
			r.lastCenterX = opts.CenterOn.X
			r.lastCenterY = opts.CenterOn.Y
			r.lastViewportSize = opts.ViewportSize
		}
	} else {
		// Full map rendering - only build once
		if !r.batchesBuilt {
			needsRebuild = true
		}
	}

	// Only rebuild batches if viewport changed or first render
	if needsRebuild {
		// Reset all batches for reuse (avoids allocations)
		for _, batch := range r.batches {
			batch.Reset()
		}

		// Collect tiles into batches grouped by image
		for x := bounds.MinX; x <= bounds.MaxX; x++ {
			for y := bounds.MinY; y <= bounds.MaxY; y++ {
				if !r.inMapBounds(x, y) {
					continue
				}

				r.addTileToBatch(x, y, opts, &bounds)
			}
		}

		r.batchesBuilt = true
	}

	// Draw all batches (much fewer draw calls than individual tiles!)
	for _, batch := range r.batches {
		batch.Draw(opts.Screen)
	}

	return bounds
}

// addTileToBatch adds a tile to the appropriate batch based on its image
func (r *TileRenderer) addTileToBatch(x, y int, opts RenderOptions, bounds *RenderedBounds) {
	logicalPos := coords.LogicalPosition{X: x, Y: y}
	idx := coords.CoordManager.LogicalToIndex(logicalPos)
	tile := r.tiles[idx]

	// Always reveal tiles (no FOV system)
	tile.IsRevealed = true

	// Get or create batch for this tile's image
	if r.batches[tile.Image] == nil {
		r.batches[tile.Image] = NewTileBatch(tile.Image)
	}
	batch := r.batches[tile.Image]

	// Calculate screen position based on viewport mode
	var screenX, screenY float32
	if opts.CenterOn != nil {
		screenX, screenY = r.calculateViewportPosition(tile, opts.CenterOn, bounds)
	} else {
		screenX = float32(tile.PixelX)
		screenY = float32(tile.PixelY)
	}

	// Get tile dimensions from image
	tileBounds := tile.Image.Bounds()
	tileW := float32(tileBounds.Dx())
	tileH := float32(tileBounds.Dy())

	// Apply scaling if in viewport mode
	if opts.CenterOn != nil {
		scale := float32(graphics.ScreenInfo.ScaleFactor)
		tileW *= scale
		tileH *= scale
	}

	// Get color values (default to white if no color matrix)
	colorR, colorG, colorB, colorA := float32(1), float32(1), float32(1), float32(1)
	if !tile.GetColorMatrix().IsEmpty() {
		cm := tile.GetColorMatrix()
		colorR, colorG, colorB, colorA = cm.R, cm.G, cm.B, cm.A
	}

	// Source rectangle (full tile image)
	srcX := float32(tileBounds.Min.X)
	srcY := float32(tileBounds.Min.Y)
	srcW := float32(tileBounds.Dx())
	srcH := float32(tileBounds.Dy())

	// Add tile to batch with both source dimensions (texture) and destination dimensions (scaled for rendering)
	batch.AddTile(screenX, screenY, srcX, srcY, srcW, srcH, tileW, tileH, colorR, colorG, colorB, colorA)
}

// calculateViewportPosition computes screen position for viewport-centered rendering
func (r *TileRenderer) calculateViewportPosition(tile *worldmap.Tile, center *coords.LogicalPosition, bounds *RenderedBounds) (float32, float32) {
	// Convert pixel position to logical position
	tileLogicalPos := coords.LogicalPosition{
		X: tile.PixelX / graphics.ScreenInfo.TileSize,
		Y: tile.PixelY / graphics.ScreenInfo.TileSize,
	}

	// Use unified coordinate transformation - handles scrolling mode and viewport centering
	screenX, screenY := coords.CoordManager.LogicalToScreen(tileLogicalPos, center)

	// Apply scaling
	scale := float32(graphics.ScreenInfo.ScaleFactor)
	tileBounds := tile.Image.Bounds()
	tileWidth := float32(tileBounds.Dx()) * scale

	// Track edges for UI layout
	tileRightEdge := int(screenX + float64(tileWidth))
	if tileRightEdge > bounds.RightEdgeX {
		bounds.RightEdgeX = tileRightEdge
	}

	tileTopEdge := int(screenY)
	if tileTopEdge < bounds.RightEdgeY {
		bounds.RightEdgeY = tileTopEdge
	}

	return float32(screenX), float32(screenY)
}

// renderTile handles single tile rendering with all effects
func (r *TileRenderer) renderTile(x, y int, opts RenderOptions, bounds *RenderedBounds) {
	logicalPos := coords.LogicalPosition{X: x, Y: y}
	idx := coords.CoordManager.LogicalToIndex(logicalPos)
	tile := r.tiles[idx]

	// Always reveal tiles (no FOV system)
	tile.IsRevealed = true

	// Reset draw options (reuse instead of allocate - eliminates 2,000 allocations/frame)
	r.drawOpts = ebiten.DrawImageOptions{}

	// Apply geometric transformation
	if opts.CenterOn != nil {
		r.applyViewportTransformWithBounds(&r.drawOpts, tile, opts.CenterOn, bounds)
	} else {
		r.applyFullMapTransform(&r.drawOpts, tile)
	}

	// Apply color matrix if present
	r.applyColorMatrix(&r.drawOpts, tile)

	opts.Screen.DrawImage(tile.Image, &r.drawOpts)
}

// applyViewportTransformWithBounds handles centered viewport rendering and edge tracking
func (r *TileRenderer) applyViewportTransformWithBounds(opts *ebiten.DrawImageOptions, tile *worldmap.Tile, center *coords.LogicalPosition, bounds *RenderedBounds) {
	// Convert pixel position to logical position
	tileLogicalPos := coords.LogicalPosition{
		X: tile.PixelX / graphics.ScreenInfo.TileSize,
		Y: tile.PixelY / graphics.ScreenInfo.TileSize,
	}

	// Apply sprite scaling (tiles need to be scaled when viewport scrolling is enabled)
	opts.GeoM.Scale(float64(graphics.ScreenInfo.ScaleFactor), float64(graphics.ScreenInfo.ScaleFactor))

	// Use unified coordinate transformation - handles scrolling mode and viewport centering
	screenX, screenY := coords.CoordManager.LogicalToScreen(tileLogicalPos, center)
	opts.GeoM.Translate(screenX, screenY)

	// Track edges for UI layout
	tileRightEdge := int(screenX + float64(tile.Image.Bounds().Dx()*graphics.ScreenInfo.ScaleFactor))
	if tileRightEdge > bounds.RightEdgeX {
		bounds.RightEdgeX = tileRightEdge
	}

	tileTopEdge := int(screenY)
	if tileTopEdge < bounds.RightEdgeY {
		bounds.RightEdgeY = tileTopEdge
	}
}

// applyFullMapTransform handles full map rendering
func (r *TileRenderer) applyFullMapTransform(opts *ebiten.DrawImageOptions, tile *worldmap.Tile) {
	opts.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
}

// applyColorMatrix applies tile-specific color effects
func (r *TileRenderer) applyColorMatrix(opts *ebiten.DrawImageOptions, tile *worldmap.Tile) {
	cm := tile.GetColorMatrix()
	if cm.IsEmpty() {
		return
	}

	r.colorScale.SetR(cm.R)
	r.colorScale.SetG(cm.G)
	r.colorScale.SetB(cm.B)
	r.colorScale.SetA(cm.A)
	opts.ColorScale.ScaleWithColorScale(r.colorScale)
}

// calculateBounds determines rendering area
func (r *TileRenderer) calculateBounds(opts RenderOptions) RenderedBounds {
	if opts.CenterOn != nil {
		sq := coords.NewDrawableSection(opts.CenterOn.X, opts.CenterOn.Y, opts.ViewportSize)
		return RenderedBounds{
			MinX: sq.StartX,
			MaxX: sq.EndX,
			MinY: sq.StartY,
			MaxY: sq.EndY,
		}
	}

	return RenderedBounds{
		MinX: 0,
		MaxX: graphics.ScreenInfo.DungeonWidth - 1,
		MinY: 0,
		MaxY: graphics.ScreenInfo.DungeonHeight - 1,
	}
}

func (r *TileRenderer) inMapBounds(x, y int) bool {
	return x >= 0 && x < graphics.ScreenInfo.DungeonWidth &&
		y >= 0 && y < graphics.ScreenInfo.DungeonHeight
}

// InvalidateCache forces batches to be rebuilt on next render
// Call this when tiles are modified (e.g., map changes, tile colors change)
func (r *TileRenderer) InvalidateCache() {
	r.batchesBuilt = false
}
