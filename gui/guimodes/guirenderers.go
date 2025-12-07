package guimodes

import (
	"game_main/coords"
	"game_main/gui/guicomponents"
	"game_main/squads"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// BorderImageCache caches border images to avoid GPU allocations in render loop.
// Images are recreated only when tile size or thickness changes (e.g., window resize).
type BorderImageCache struct {
	top, bottom, left, right *ebiten.Image
	tileSize, thickness      int
}

// GetOrCreate returns cached border images, creating them only if dimensions changed.
func (cache *BorderImageCache) GetOrCreate(tileSize, thickness int) (*ebiten.Image, *ebiten.Image, *ebiten.Image, *ebiten.Image) {
	if cache.top == nil || cache.tileSize != tileSize || cache.thickness != thickness {
		// Recreate images only on size change
		cache.top = ebiten.NewImage(tileSize, thickness)
		cache.bottom = ebiten.NewImage(tileSize, thickness)
		cache.left = ebiten.NewImage(thickness, tileSize)
		cache.right = ebiten.NewImage(thickness, tileSize)
		cache.tileSize = tileSize
		cache.thickness = thickness
	}
	return cache.top, cache.bottom, cache.left, cache.right
}

// ViewportRenderer provides viewport-centered rendering utilities.
// Now a thin wrapper around CoordinateManager for convenience.
type ViewportRenderer struct {
	centerPos    coords.LogicalPosition
	borderImages BorderImageCache
}

// NewViewportRenderer creates a renderer for the current screen
func NewViewportRenderer(screen *ebiten.Image, centerPos coords.LogicalPosition) *ViewportRenderer {
	// Update screen dimensions in global CoordManager
	coords.CoordManager.UpdateScreenDimensions(screen.Bounds().Dx(), screen.Bounds().Dy())

	return &ViewportRenderer{
		centerPos: centerPos,
	}
}

// TileSize returns the scaled tile size
func (vr *ViewportRenderer) TileSize() int {
	return coords.CoordManager.GetScaledTileSize()
}

// LogicalToScreen converts logical position to screen coordinates.
// Delegates to CoordinateManager which handles scrolling mode automatically.
func (vr *ViewportRenderer) LogicalToScreen(pos coords.LogicalPosition) (float64, float64) {
	return coords.CoordManager.LogicalToScreen(pos, &vr.centerPos)
}

// DrawTileOverlay draws a colored rectangle at a logical position
func (vr *ViewportRenderer) DrawTileOverlay(screen *ebiten.Image, pos coords.LogicalPosition, fillColor color.Color) {
	screenX, screenY := vr.LogicalToScreen(pos)
	tileSize := vr.TileSize()

	rect := ebiten.NewImage(tileSize, tileSize)
	rect.Fill(fillColor)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY)
	screen.DrawImage(rect, op)
}

// DrawTileBorder draws a colored border around a logical position
func (vr *ViewportRenderer) DrawTileBorder(screen *ebiten.Image, pos coords.LogicalPosition, borderColor color.Color, thickness int) {
	screenX, screenY := vr.LogicalToScreen(pos)
	tileSize := vr.TileSize()

	// Get cached border images (creates only on first use or size change)
	topBorder, bottomBorder, leftBorder, rightBorder := vr.borderImages.GetOrCreate(tileSize, thickness)

	// Fill cached images with current color
	topBorder.Fill(borderColor)
	bottomBorder.Fill(borderColor)
	leftBorder.Fill(borderColor)
	rightBorder.Fill(borderColor)

	// Top border
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY)
	screen.DrawImage(topBorder, op)

	// Bottom border
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY+float64(tileSize-thickness))
	screen.DrawImage(bottomBorder, op)

	// Left border
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY)
	screen.DrawImage(leftBorder, op)

	// Right border
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX+float64(tileSize-thickness), screenY)
	screen.DrawImage(rightBorder, op)
}

// ===== COMBAT RENDERING SYSTEMS =====

// MovementTileRenderer renders valid movement tiles
type MovementTileRenderer struct {
	fillColor color.Color
}

// NewMovementTileRenderer creates a renderer for movement tiles
func NewMovementTileRenderer() *MovementTileRenderer {
	return &MovementTileRenderer{
		fillColor: color.RGBA{R: 0, G: 255, B: 0, A: 80}, // Semi-transparent green
	}
}

// Render draws all valid movement tiles
func (mtr *MovementTileRenderer) Render(screen *ebiten.Image, centerPos coords.LogicalPosition, validTiles []coords.LogicalPosition) {
	vr := NewViewportRenderer(screen, centerPos)

	for _, pos := range validTiles {
		vr.DrawTileOverlay(screen, pos, mtr.fillColor)
	}
}

// SquadHighlightRenderer renders squad position highlights
type SquadHighlightRenderer struct {
	queries         *guicomponents.GUIQueries
	selectedColor   color.Color
	playerColor     color.Color
	enemyColor      color.Color
	borderThickness int
	cachedRenderer  *ViewportRenderer
	lastCenterPos   coords.LogicalPosition
}

// NewSquadHighlightRenderer creates a renderer for squad highlights
func NewSquadHighlightRenderer(queries *guicomponents.GUIQueries) *SquadHighlightRenderer {
	return &SquadHighlightRenderer{
		queries:         queries,
		selectedColor:   color.RGBA{R: 255, G: 255, B: 255, A: 255}, // White
		playerColor:     color.RGBA{R: 0, G: 150, B: 255, A: 150},   // Blue
		enemyColor:      color.RGBA{R: 255, G: 0, B: 0, A: 150},     // Red
		borderThickness: 3,
	}
}

// Render draws highlights for all squads
func (shr *SquadHighlightRenderer) Render(
	screen *ebiten.Image,
	centerPos coords.LogicalPosition,
	currentFactionID ecs.EntityID,
	selectedSquadID ecs.EntityID,
) {
	// Only recreate renderer if viewport moved or not yet created
	if shr.cachedRenderer == nil || shr.lastCenterPos != centerPos {
		shr.cachedRenderer = NewViewportRenderer(screen, centerPos)
		shr.lastCenterPos = centerPos
	}
	vr := shr.cachedRenderer

	// Get all squads with positions
	allSquads := squads.FindAllSquads(shr.queries.ECSManager)

	for _, squadID := range allSquads {
		squadInfo := shr.queries.GetSquadInfo(squadID)
		if squadInfo == nil || squadInfo.IsDestroyed || squadInfo.Position == nil {
			continue
		}

		// Determine highlight color
		var highlightColor color.Color
		if squadID == selectedSquadID {
			highlightColor = shr.selectedColor
		} else if squadInfo.FactionID == currentFactionID {
			highlightColor = shr.playerColor
		} else {
			highlightColor = shr.enemyColor
		}

		// Draw border
		vr.DrawTileBorder(screen, *squadInfo.Position, highlightColor, shr.borderThickness)
	}
}
