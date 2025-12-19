// Package coords - coordinate manager and conversion utilities
package coords

import "game_main/config"

// CoordManager is a global coordinate manager instance.
var CoordManager *CoordinateManager

// MAP_SCROLLING_ENABLED controls whether the game uses viewport scrolling (true) or full map view (false).
// When true: Uses config.DefaultScaleFactor scaling and centers viewport on player position
// When false: Uses 1x scaling and shows entire map
var MAP_SCROLLING_ENABLED = true

// Initialize the global coordinate manager with default screen data.
func init() {
	screenData := NewScreenData()
	CoordManager = NewCoordinateManager(screenData)
}

// ScreenData contains screen configuration data for the coordinate manager.
// This is used to initialize the coordinate system.
type ScreenData struct {
	ScreenWidth   int
	ScreenHeight  int
	TileSize      int
	DungeonWidth  int
	DungeonHeight int
	ScaleFactor   int
	LevelWidth    int
	LevelHeight   int
	PaddingRight  int // Extra padding added to the right outside of the map
}

// GetCanvasWidth calculates the total canvas width including padding.
func (s ScreenData) GetCanvasWidth() int {
	return int(float64(s.TileSize*s.DungeonWidth)) + s.PaddingRight
}

// GetCanvasHeight calculates the total canvas height.
func (s ScreenData) GetCanvasHeight() int {
	return int(float64(s.TileSize * s.DungeonHeight))
}

// NewScreenData creates default screen configuration.
//TODO,
func NewScreenData() ScreenData {
	g := ScreenData{
		DungeonWidth:  100,
		DungeonHeight: 80,
	}
	tilePixels := config.DefaultTilePixels

	// Use a single scale value for both X and Y
	g.TileSize = tilePixels
	g.ScaleFactor = config.DefaultScaleFactor

	// Calculate the level dimensions based on the tile size
	g.LevelHeight = g.DungeonHeight * g.TileSize
	g.LevelWidth = g.DungeonWidth * g.TileSize

	g.PaddingRight = config.DefaultRightPadding

	return g
}

// CoordinateManager provides a unified interface for all coordinate operations.
// 1. Centralizes coordinate logic in one place
// 2. Provides type safety with LogicalPosition wrapper

type CoordinateManager struct {
	dungeonWidth  int
	dungeonHeight int
	tileSize      int
	scaleFactor   int
	screenWidth   int
	screenHeight  int

	// Reusable viewport to avoid allocation storm (GUI_PERFORMANCE_ANALYSIS.md)
	viewport *Viewport
}

type Viewport struct {
	centerX, centerY int // logical coordinates of viewport center
	manager          *CoordinateManager
}

// NewCoordinateManager creates a new coordinate manager with the given screen configuration.
func NewCoordinateManager(screenData ScreenData) *CoordinateManager {
	cm := &CoordinateManager{
		dungeonWidth:  screenData.DungeonWidth,
		dungeonHeight: screenData.DungeonHeight,
		tileSize:      screenData.TileSize,
		scaleFactor:   screenData.ScaleFactor,
		screenWidth:   screenData.ScreenWidth,
		screenHeight:  screenData.ScreenHeight,
	}

	// Initialize viewport once with origin (will be updated via SetCenter)
	cm.viewport = NewViewport(cm, LogicalPosition{X: 0, Y: 0})

	return cm
}

// === CORE COORDINATE CONVERSIONS ===
// These replace the scattered CoordTransformer methods

// LogicalToIndex converts logical coordinates to flat map array index.
func (cm *CoordinateManager) LogicalToIndex(pos LogicalPosition) int {
	return (pos.Y * cm.dungeonWidth) + pos.X
}

// IndexToLogical converts flat map array index to logical coordinates.
func (cm *CoordinateManager) IndexToLogical(index int) LogicalPosition {
	x := index % cm.dungeonWidth
	y := index / cm.dungeonWidth
	return LogicalPosition{X: x, Y: y}
}

// LogicalToPixel converts logical coordinates to pixel coordinates (for rendering).
func (cm *CoordinateManager) LogicalToPixel(pos LogicalPosition) PixelPosition {
	return PixelPosition{
		X: pos.X * cm.tileSize,
		Y: pos.Y * cm.tileSize,
	}
}

// IndexToPixel converts flat map array index to pixel coordinates.
func (cm *CoordinateManager) IndexToPixel(index int) PixelPosition {
	logical := cm.IndexToLogical(index)
	return cm.LogicalToPixel(logical)
}

// PixelToLogical converts pixel coordinates to logical coordinates.
func (cm *CoordinateManager) PixelToLogical(pos PixelPosition) LogicalPosition {
	return LogicalPosition{
		X: pos.X / cm.tileSize,
		Y: pos.Y / cm.tileSize,
	}
}

// === VIEWPORT/CAMERA OPERATIONS ===

// NewViewport creates a new viewport centered on the given position.
func NewViewport(manager *CoordinateManager, centerPos LogicalPosition) *Viewport {
	return &Viewport{
		centerX: centerPos.X,
		centerY: centerPos.Y,
		manager: manager,
	}
}

// SetCenter updates the viewport center (typically player position).
func (v *Viewport) SetCenter(pos LogicalPosition) {
	v.centerX = pos.X
	v.centerY = pos.Y
}

// LogicalToScreen converts logical coordinates to screen coordinates with viewport centering.
// Returns the position where a sprite should be drawn (before sprite scaling is applied).
func (v *Viewport) LogicalToScreen(pos LogicalPosition) (float64, float64) {
	// Convert to pixel coordinates
	pixelX := float64(pos.X * v.manager.tileSize)
	pixelY := float64(pos.Y * v.manager.tileSize)

	// Calculate offset to center the viewport
	centerPixelX := float64(v.centerX * v.manager.tileSize)
	centerPixelY := float64(v.centerY * v.manager.tileSize)

	offsetX := float64(v.manager.screenWidth)/2 - centerPixelX*float64(v.manager.scaleFactor)
	offsetY := float64(v.manager.screenHeight)/2 - centerPixelY*float64(v.manager.scaleFactor)

	// Apply scaling to position and add centering offset
	scaledX := pixelX * float64(v.manager.scaleFactor)
	scaledY := pixelY * float64(v.manager.scaleFactor)

	return scaledX + offsetX, scaledY + offsetY
}

// ScreenToLogical converts screen coordinates back to logical coordinates.
func (v *Viewport) ScreenToLogical(screenX, screenY int) LogicalPosition {
	// Calculate offset to center the viewport
	offsetX := float64(v.manager.screenWidth)/2 - float64(v.centerX*v.manager.tileSize)*float64(v.manager.scaleFactor)
	offsetY := float64(v.manager.screenHeight)/2 - float64(v.centerY*v.manager.tileSize)*float64(v.manager.scaleFactor)

	// Reverse the viewport transformation
	uncenteredX := float64(screenX) - offsetX
	uncenteredY := float64(screenY) - offsetY

	// Reverse the scaling
	pixelX := uncenteredX / float64(v.manager.scaleFactor)
	pixelY := uncenteredY / float64(v.manager.scaleFactor)

	// Convert to logical coordinates
	return v.manager.PixelToLogical(PixelPosition{X: int(pixelX), Y: int(pixelY)})
}

// === UTILITY FUNCTIONS ===

// DrawableSection is used for drawing only a section of the map.
// Different from TileSquare. TileSquare returns indices.
// DrawableSection uses logical coordinates to center the square around a point.
type DrawableSection struct {
	StartX int
	StartY int
	EndX   int
	EndY   int
}

// NewDrawableSection creates a drawable section centered on the given position with the specified size.
func NewDrawableSection(centerX, centerY, size int) DrawableSection {
	halfSize := size / 2
	return DrawableSection{
		StartX: centerX - halfSize,
		StartY: centerY - halfSize,
		EndX:   centerX + halfSize,
		EndY:   centerY + halfSize,
	}
}

// === UNIFIED SCREEN TRANSFORMATION API ===

// UpdateScreenDimensions updates the screen dimensions in the coordinate manager.
// Call this each frame from Draw() to keep screen dimensions current for viewport calculations.
func (cm *CoordinateManager) UpdateScreenDimensions(width, height int) {
	cm.screenWidth = width
	cm.screenHeight = height
}

// LogicalToScreen converts logical position to screen coordinates.
// - When scrolling enabled with centerPos: applies viewport centering and scaling
// - When scrolling disabled or centerPos is nil: returns scaled pixels without centering
// centerPos: viewport center (typically player position) - pass nil for full map view
func (cm *CoordinateManager) LogicalToScreen(pos LogicalPosition, centerPos *LogicalPosition) (float64, float64) {
	// Determine scale factor based on mode
	scaleFactor := 1
	if MAP_SCROLLING_ENABLED {
		scaleFactor = cm.scaleFactor
	}

	// Convert to pixel coordinates
	pixelPos := cm.LogicalToPixel(pos)

	// If no center position or scrolling disabled, return scaled pixels
	if centerPos == nil || !MAP_SCROLLING_ENABLED {
		return float64(pixelPos.X) * float64(scaleFactor),
			float64(pixelPos.Y) * float64(scaleFactor)
	}

	// Update viewport center and use existing viewport instance
	cm.viewport.SetCenter(*centerPos)
	return cm.viewport.LogicalToScreen(pos)
}

// ScreenToLogical converts screen coordinates to logical position.
// - When scrolling enabled with centerPos: reverses viewport centering and scaling
// - When scrolling disabled or centerPos is nil: converts pixels directly to logical
// centerPos: viewport center (typically player position) - pass nil for full map view
func (cm *CoordinateManager) ScreenToLogical(screenX, screenY int, centerPos *LogicalPosition) LogicalPosition {
	// If no center position or scrolling disabled, convert pixels directly to logical
	if centerPos == nil || !MAP_SCROLLING_ENABLED {
		// Determine scale factor
		scaleFactor := 1
		if MAP_SCROLLING_ENABLED {
			scaleFactor = cm.scaleFactor
		}

		// Reverse scaling
		pixelX := screenX / scaleFactor
		pixelY := screenY / scaleFactor

		// Convert to logical
		return cm.PixelToLogical(PixelPosition{X: pixelX, Y: pixelY})
	}

	// Update viewport center and use existing viewport instance
	cm.viewport.SetCenter(*centerPos)
	return cm.viewport.ScreenToLogical(screenX, screenY)
}

// IndexToScreen converts array index to screen coordinates.
// Combines IndexToLogical + LogicalToScreen for convenience.
// centerPos: viewport center (typically player position) - pass nil for full map view
func (cm *CoordinateManager) IndexToScreen(index int, centerPos *LogicalPosition) (float64, float64) {
	return cm.LogicalToScreen(cm.IndexToLogical(index), centerPos)
}

// GetScaledTileSize returns the tile size with current scale factor applied.
// Returns tileSize * scaleFactor when scrolling enabled, otherwise just tileSize.
func (cm *CoordinateManager) GetScaledTileSize() int {
	if MAP_SCROLLING_ENABLED {
		return cm.tileSize * cm.scaleFactor
	}
	return cm.tileSize
}
