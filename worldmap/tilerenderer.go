package worldmap

import (
	"game_main/coords"
	"game_main/graphics"

	"github.com/hajimehoshi/ebiten/v2"
)

// TileRenderer handles rendering of map tiles with color matrices
type TileRenderer struct {
	tiles      []*Tile
	colorScale ebiten.ColorScale
	drawOpts   ebiten.DrawImageOptions // Reusable draw options (eliminates 2,000 allocations/frame)
}

// NewTileRenderer creates a renderer for the given tileset
func NewTileRenderer(tiles []*Tile) *TileRenderer {
	return &TileRenderer{
		tiles: tiles,
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

// Render draws tiles to screen based on options
func (r *TileRenderer) Render(opts RenderOptions) RenderedBounds {
	bounds := r.calculateBounds(opts)
	bounds.RightEdgeX = 0
	bounds.RightEdgeY = 0

	for x := bounds.MinX; x <= bounds.MaxX; x++ {
		for y := bounds.MinY; y <= bounds.MaxY; y++ {
			if !r.inMapBounds(x, y) {
				continue
			}

			r.renderTile(x, y, opts, &bounds)
		}
	}

	return bounds
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

	opts.Screen.DrawImage(tile.image, &r.drawOpts)
}

// applyViewportTransformWithBounds handles centered viewport rendering and edge tracking
func (r *TileRenderer) applyViewportTransformWithBounds(opts *ebiten.DrawImageOptions, tile *Tile, center *coords.LogicalPosition, bounds *RenderedBounds) {
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
	tileRightEdge := int(screenX + float64(tile.image.Bounds().Dx()*graphics.ScreenInfo.ScaleFactor))
	if tileRightEdge > bounds.RightEdgeX {
		bounds.RightEdgeX = tileRightEdge
	}

	tileTopEdge := int(screenY)
	if tileTopEdge < bounds.RightEdgeY {
		bounds.RightEdgeY = tileTopEdge
	}
}

// applyFullMapTransform handles full map rendering
func (r *TileRenderer) applyFullMapTransform(opts *ebiten.DrawImageOptions, tile *Tile) {
	opts.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
}

// applyColorMatrix applies tile-specific color effects
func (r *TileRenderer) applyColorMatrix(opts *ebiten.DrawImageOptions, tile *Tile) {
	if tile.cm.IsEmpty() {
		return
	}

	r.colorScale.SetR(tile.cm.R)
	r.colorScale.SetG(tile.cm.G)
	r.colorScale.SetB(tile.cm.B)
	r.colorScale.SetA(tile.cm.A)
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
