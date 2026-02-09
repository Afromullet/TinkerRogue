package rendering

import (
	"game_main/world/coords"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// MovementTileRenderer renders valid movement tiles
type MovementTileRenderer struct {
	fillColor color.Color
	viewport  CachedViewport
}

// NewMovementTileRenderer creates a renderer for movement tiles
func NewMovementTileRenderer() *MovementTileRenderer {
	return &MovementTileRenderer{
		fillColor: color.RGBA{R: 0, G: 255, B: 0, A: 80}, // Semi-transparent green
	}
}

// Render draws all valid movement tiles
func (mtr *MovementTileRenderer) Render(screen *ebiten.Image, centerPos coords.LogicalPosition, validTiles []coords.LogicalPosition) {
	vr := mtr.viewport.GetRenderer(screen, centerPos)

	for _, pos := range validTiles {
		vr.DrawTileOverlay(screen, pos, mtr.fillColor)
	}
}

// HealthBarRenderer renders health bars above squads
type HealthBarRenderer struct {
	dataProvider   SquadInfoProvider
	bgColor        color.Color
	fillColor      color.Color
	barHeight      int
	barWidthRatio  float64 // Ratio of tile width for bar width
	yOffset        int     // Offset above the tile (negative = above)
	viewport       CachedViewport
	bgImage        *ebiten.Image // Cached background bar image
	fillImage      *ebiten.Image // Cached fill bar image
	cachedBarWidth int           // Cached bar width for invalidation
}

// NewHealthBarRenderer creates a renderer for squad health bars
func NewHealthBarRenderer(dataProvider SquadInfoProvider) *HealthBarRenderer {
	return &HealthBarRenderer{
		dataProvider:  dataProvider,
		bgColor:       color.RGBA{R: 40, G: 40, B: 40, A: 200},  // Dark gray background
		fillColor:     color.RGBA{R: 220, G: 40, B: 40, A: 255}, // Red health fill
		barHeight:     5,
		barWidthRatio: 0.8, // 80% of tile width
		yOffset:       -10, // 10 pixels above the tile
	}
}

// Render draws health bars above all squads
func (hbr *HealthBarRenderer) Render(screen *ebiten.Image, centerPos coords.LogicalPosition) {
	vr := hbr.viewport.GetRenderer(screen, centerPos)

	tileSize := vr.TileSize()
	barWidth := int(float64(tileSize) * hbr.barWidthRatio)

	// Update cached images if bar width changed
	if hbr.bgImage == nil || hbr.cachedBarWidth != barWidth {
		hbr.bgImage = ebiten.NewImage(barWidth, hbr.barHeight)
		hbr.bgImage.Fill(hbr.bgColor)
		hbr.fillImage = ebiten.NewImage(barWidth, hbr.barHeight)
		hbr.fillImage.Fill(hbr.fillColor)
		hbr.cachedBarWidth = barWidth
	}

	// Get all squads
	allSquads := hbr.dataProvider.GetAllSquadIDs()

	for _, squadID := range allSquads {
		info := hbr.dataProvider.GetSquadRenderInfo(squadID)
		if info == nil || info.IsDestroyed || info.Position == nil {
			continue
		}

		// Calculate health ratio
		healthRatio := 0.0
		if info.MaxHP > 0 {
			healthRatio = float64(info.CurrentHP) / float64(info.MaxHP)
		}

		hbr.drawHealthBar(screen, vr, *info.Position, healthRatio, barWidth, tileSize)
	}
}

// drawHealthBar draws a single health bar at the given position
func (hbr *HealthBarRenderer) drawHealthBar(screen *ebiten.Image, vr *ViewportRenderer, pos coords.LogicalPosition, healthRatio float64, barWidth, tileSize int) {
	screenX, screenY := vr.LogicalToScreen(pos)

	// Center the bar horizontally above the tile
	barX := screenX + float64(tileSize-barWidth)/2
	barY := screenY + float64(hbr.yOffset)

	// Draw background bar
	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Translate(barX, barY)
	screen.DrawImage(hbr.bgImage, bgOp)

	// Draw health fill (proportional to health ratio)
	if healthRatio > 0 {
		fillWidth := int(float64(barWidth) * healthRatio)
		if fillWidth > 0 {
			// Create a sub-image for the fill portion
			fillOp := &ebiten.DrawImageOptions{}
			fillOp.GeoM.Translate(barX, barY)
			fillRect := hbr.fillImage.SubImage(image.Rect(0, 0, fillWidth, hbr.barHeight)).(*ebiten.Image)
			screen.DrawImage(fillRect, fillOp)
		}
	}
}
