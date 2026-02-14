package rendering

import (
	"game_main/world/coords"
	"image/color"

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
	overlayDrawOpts ebiten.DrawImageOptions    // Reusable draw options for overlays
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

	vr.overlayDrawOpts.GeoM.Reset()
	vr.overlayDrawOpts.GeoM.Translate(screenX, screenY)
	screen.DrawImage(vr.overlayCache, &vr.overlayDrawOpts)
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
