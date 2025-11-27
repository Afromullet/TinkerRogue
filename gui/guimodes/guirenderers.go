package guimodes

import (
	"game_main/coords"
	"game_main/graphics"
	"game_main/gui/guicomponents"
	"game_main/squads"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// ViewportRenderer provides viewport-centered rendering utilities
type ViewportRenderer struct {
	screenData coords.ScreenData
	viewport   *coords.Viewport
}

// NewViewportRenderer creates a renderer for the current screen
func NewViewportRenderer(screen *ebiten.Image, centerPos coords.LogicalPosition) *ViewportRenderer {
	screenData := graphics.ScreenInfo
	screenData.ScreenWidth = screen.Bounds().Dx()
	screenData.ScreenHeight = screen.Bounds().Dy()

	manager := coords.NewCoordinateManager(screenData)
	viewport := coords.NewViewport(manager, centerPos)

	return &ViewportRenderer{
		screenData: screenData,
		viewport:   viewport,
	}
}

// TileSize returns the scaled tile size
func (vr *ViewportRenderer) TileSize() int {
	return vr.screenData.TileSize * vr.screenData.ScaleFactor
}

// LogicalToScreen converts logical position to screen coordinates
func (vr *ViewportRenderer) LogicalToScreen(pos coords.LogicalPosition) (float64, float64) {
	return vr.viewport.LogicalToScreen(pos)
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

	// Top border
	topBorder := ebiten.NewImage(tileSize, thickness)
	topBorder.Fill(borderColor)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY)
	screen.DrawImage(topBorder, op)

	// Bottom border
	bottomBorder := ebiten.NewImage(tileSize, thickness)
	bottomBorder.Fill(borderColor)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY+float64(tileSize-thickness))
	screen.DrawImage(bottomBorder, op)

	// Left border
	leftBorder := ebiten.NewImage(thickness, tileSize)
	leftBorder.Fill(borderColor)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY)
	screen.DrawImage(leftBorder, op)

	// Right border
	rightBorder := ebiten.NewImage(thickness, tileSize)
	rightBorder.Fill(borderColor)
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
	vr := NewViewportRenderer(screen, centerPos)

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
