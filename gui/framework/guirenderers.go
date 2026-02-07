package framework

import (
	"game_main/world/coords"
	"image"
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
// Border images are filled with white and should be tinted using ColorScale.
func (cache *BorderImageCache) GetOrCreate(tileSize, thickness int) (*ebiten.Image, *ebiten.Image, *ebiten.Image, *ebiten.Image) {
	if cache.top == nil || cache.tileSize != tileSize || cache.thickness != thickness {
		// Recreate images only on size change
		cache.top = ebiten.NewImage(tileSize, thickness)
		cache.bottom = ebiten.NewImage(tileSize, thickness)
		cache.left = ebiten.NewImage(thickness, tileSize)
		cache.right = ebiten.NewImage(thickness, tileSize)

		// Fill with white once - will be tinted with ColorScale in DrawTileBorder
		white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		cache.top.Fill(white)
		cache.bottom.Fill(white)
		cache.left.Fill(white)
		cache.right.Fill(white)

		cache.tileSize = tileSize
		cache.thickness = thickness
	}
	return cache.top, cache.bottom, cache.left, cache.right
}

// ViewportRenderer provides viewport-centered rendering utilities.
// Now a thin wrapper around CoordinateManager for convenience.
type ViewportRenderer struct {
	centerPos       coords.LogicalPosition
	borderImages    BorderImageCache
	overlayCache    *ebiten.Image              // Reusable overlay image to avoid allocations
	overlayTileSize int                        // Track size for invalidation
	borderDrawOpts  [4]ebiten.DrawImageOptions // Reusable draw options for borders [top, bottom, left, right]
}

// NewViewportRenderer creates a renderer for the current screen
func NewViewportRenderer(screen *ebiten.Image, centerPos coords.LogicalPosition) *ViewportRenderer {
	// Update screen dimensions in global CoordManager
	coords.CoordManager.UpdateScreenDimensions(screen.Bounds().Dx(), screen.Bounds().Dy())

	return &ViewportRenderer{
		centerPos: centerPos,
	}
}

// UpdateCenter updates the viewport center position without recreating the renderer
func (vr *ViewportRenderer) UpdateCenter(centerPos coords.LogicalPosition) {
	vr.centerPos = centerPos
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

	// Create overlay image only once or when size changes
	if vr.overlayCache == nil || vr.overlayTileSize != tileSize {
		vr.overlayCache = ebiten.NewImage(tileSize, tileSize)
		vr.overlayTileSize = tileSize
	}

	// Fill with color (still necessary but much faster on existing image)
	vr.overlayCache.Fill(fillColor)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY)
	screen.DrawImage(vr.overlayCache, op)
}

// DrawTileBorder draws a colored border around a logical position
func (vr *ViewportRenderer) DrawTileBorder(screen *ebiten.Image, pos coords.LogicalPosition, borderColor color.Color, thickness int) {
	screenX, screenY := vr.LogicalToScreen(pos)
	tileSize := vr.TileSize()

	// Get cached white border images (created only on first use or size change)
	topBorder, bottomBorder, leftBorder, rightBorder := vr.borderImages.GetOrCreate(tileSize, thickness)

	// Convert border color to ColorScale for GPU-based tinting
	r, g, b, a := borderColor.RGBA()
	colorScale := ebiten.ColorScale{}
	colorScale.SetR(float32(r) / 0xffff)
	colorScale.SetG(float32(g) / 0xffff)
	colorScale.SetB(float32(b) / 0xffff)
	colorScale.SetA(float32(a) / 0xffff)

	// Top border - reuse draw options
	vr.borderDrawOpts[0].GeoM.Reset()
	vr.borderDrawOpts[0].GeoM.Translate(screenX, screenY)
	vr.borderDrawOpts[0].ColorScale = colorScale
	screen.DrawImage(topBorder, &vr.borderDrawOpts[0])

	// Bottom border - reuse draw options
	vr.borderDrawOpts[1].GeoM.Reset()
	vr.borderDrawOpts[1].GeoM.Translate(screenX, screenY+float64(tileSize-thickness))
	vr.borderDrawOpts[1].ColorScale = colorScale
	screen.DrawImage(bottomBorder, &vr.borderDrawOpts[1])

	// Left border - reuse draw options
	vr.borderDrawOpts[2].GeoM.Reset()
	vr.borderDrawOpts[2].GeoM.Translate(screenX, screenY)
	vr.borderDrawOpts[2].ColorScale = colorScale
	screen.DrawImage(leftBorder, &vr.borderDrawOpts[2])

	// Right border - reuse draw options
	vr.borderDrawOpts[3].GeoM.Reset()
	vr.borderDrawOpts[3].GeoM.Translate(screenX+float64(tileSize-thickness), screenY)
	vr.borderDrawOpts[3].ColorScale = colorScale
	screen.DrawImage(rightBorder, &vr.borderDrawOpts[3])
}

// ===== COMBAT RENDERING SYSTEMS =====

// CachedViewport manages a ViewportRenderer with caching to avoid recreation
// when only the center position changes. Shared by all tile renderers.
type CachedViewport struct {
	renderer    *ViewportRenderer
	lastCenter  coords.LogicalPosition
	lastScreenX int
	lastScreenY int
}

// GetRenderer returns a ViewportRenderer, creating or updating it as needed.
// Only recreates the renderer when screen dimensions change.
func (cv *CachedViewport) GetRenderer(screen *ebiten.Image, centerPos coords.LogicalPosition) *ViewportRenderer {
	screenX, screenY := screen.Bounds().Dx(), screen.Bounds().Dy()

	if cv.renderer == nil || cv.lastScreenX != screenX || cv.lastScreenY != screenY {
		cv.renderer = NewViewportRenderer(screen, centerPos)
		cv.lastCenter = centerPos
		cv.lastScreenX = screenX
		cv.lastScreenY = screenY
	} else if cv.lastCenter != centerPos {
		cv.renderer.UpdateCenter(centerPos)
		cv.lastCenter = centerPos
	}

	return cv.renderer
}

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

// SquadHighlightRenderer renders squad position highlights
type SquadHighlightRenderer struct {
	queries         *GUIQueries
	selectedColor   color.Color
	factionColors   map[ecs.EntityID]color.Color // Maps faction ID to unique color
	defaultColor    color.Color                  // Fallback color for unknown factions
	borderThickness int
	viewport        CachedViewport
}

// NewSquadHighlightRenderer creates a renderer for squad highlights
func NewSquadHighlightRenderer(queries *GUIQueries) *SquadHighlightRenderer {
	return &SquadHighlightRenderer{
		queries:         queries,
		selectedColor:   color.RGBA{R: 255, G: 255, B: 255, A: 255}, // White for selected
		factionColors:   make(map[ecs.EntityID]color.Color),
		defaultColor:    color.RGBA{R: 128, G: 128, B: 128, A: 150}, // Gray for unknown
		borderThickness: 3,
	}
}

// GetFactionColor returns a unique color for a faction, assigning one if needed
func (shr *SquadHighlightRenderer) GetFactionColor(factionID ecs.EntityID) color.Color {
	// Return cached color if already assigned
	if c, exists := shr.factionColors[factionID]; exists {
		return c
	}

	// Palette of distinct colors for factions
	colorPalette := []color.Color{
		color.RGBA{R: 0, G: 150, B: 255, A: 150},   // Blue
		color.RGBA{R: 255, G: 0, B: 0, A: 150},     // Red
		color.RGBA{R: 0, G: 200, B: 0, A: 150},     // Green
		color.RGBA{R: 255, G: 215, B: 0, A: 150},   // Gold/Yellow
		color.RGBA{R: 200, G: 0, B: 200, A: 150},   // Purple/Magenta
		color.RGBA{R: 255, G: 140, B: 0, A: 150},   // Orange
		color.RGBA{R: 0, G: 255, B: 255, A: 150},   // Cyan
		color.RGBA{R: 255, G: 105, B: 180, A: 150}, // Pink
	}

	// Assign color based on number of factions already assigned
	colorIndex := len(shr.factionColors) % len(colorPalette)
	assignedColor := colorPalette[colorIndex]
	shr.factionColors[factionID] = assignedColor

	return assignedColor
}

// Render draws highlights for all squads
func (shr *SquadHighlightRenderer) Render(
	screen *ebiten.Image,
	centerPos coords.LogicalPosition,
	currentFactionID ecs.EntityID,
	selectedSquadID ecs.EntityID,
) {
	vr := shr.viewport.GetRenderer(screen, centerPos)

	// Get all squads with positions
	// GetSquadInfo uses Views directly - no need for BuildSquadInfoCache
	allSquads := shr.queries.SquadCache.FindAllSquads()

	for _, squadID := range allSquads {
		// GetSquadInfo now uses Views directly via SquadCache
		squadInfo := shr.queries.GetSquadInfo(squadID)
		if squadInfo == nil || squadInfo.IsDestroyed || squadInfo.Position == nil {
			continue
		}

		// Determine highlight color
		var highlightColor color.Color
		if squadID == selectedSquadID {
			// Selected squad gets white border
			highlightColor = shr.selectedColor
		} else {
			// Each faction gets its unique color
			highlightColor = shr.GetFactionColor(squadInfo.FactionID)
		}

		// Draw border
		vr.DrawTileBorder(screen, *squadInfo.Position, highlightColor, shr.borderThickness)
	}
}

// HealthBarRenderer renders health bars above squads
type HealthBarRenderer struct {
	queries       *GUIQueries
	bgColor       color.Color
	fillColor     color.Color
	barHeight     int
	barWidthRatio float64 // Ratio of tile width for bar width
	yOffset       int     // Offset above the tile (negative = above)
	viewport      CachedViewport
	bgImage       *ebiten.Image // Cached background bar image
	fillImage     *ebiten.Image // Cached fill bar image
	cachedBarWidth int          // Cached bar width for invalidation
}

// NewHealthBarRenderer creates a renderer for squad health bars
func NewHealthBarRenderer(queries *GUIQueries) *HealthBarRenderer {
	return &HealthBarRenderer{
		queries:       queries,
		bgColor:       color.RGBA{R: 40, G: 40, B: 40, A: 200},   // Dark gray background
		fillColor:     color.RGBA{R: 220, G: 40, B: 40, A: 255},  // Red health fill
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
	allSquads := hbr.queries.SquadCache.FindAllSquads()

	for _, squadID := range allSquads {
		squadInfo := hbr.queries.GetSquadInfo(squadID)
		if squadInfo == nil || squadInfo.IsDestroyed || squadInfo.Position == nil {
			continue
		}

		// Calculate health ratio
		healthRatio := 0.0
		if squadInfo.MaxHP > 0 {
			healthRatio = float64(squadInfo.CurrentHP) / float64(squadInfo.MaxHP)
		}

		hbr.drawHealthBar(screen, *squadInfo.Position, healthRatio, barWidth, tileSize)
	}
}

// drawHealthBar draws a single health bar at the given position
func (hbr *HealthBarRenderer) drawHealthBar(screen *ebiten.Image, pos coords.LogicalPosition, healthRatio float64, barWidth, tileSize int) {
	screenX, screenY := hbr.viewport.renderer.LogicalToScreen(pos)

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
