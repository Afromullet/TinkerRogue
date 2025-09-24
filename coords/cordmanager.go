// Package coords - coordinate manager and conversion utilities
package coords

// No imports needed currently

// CoordManager is a global coordinate manager instance.

var CoordManager *CoordinateManager

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
// TODO: This should only be calculated once instead of being called for every coordinate conversion.
func NewScreenData() ScreenData {
	g := ScreenData{
		DungeonWidth:  100,
		DungeonHeight: 80,
	}
	tilePixels := 32

	// Use a single scale value for both X and Y
	g.TileSize = tilePixels
	g.ScaleFactor = 3

	// Calculate the level dimensions based on the tile size
	g.LevelHeight = g.DungeonHeight * g.TileSize
	g.LevelWidth = g.DungeonWidth * g.TileSize

	g.PaddingRight = 500

	return g
}

// CoordinateManager provides a unified interface for all coordinate operations.
// This was moved from graphics package to eliminate dependency coupling.
// Addresses problems:
// 1. Eliminates scattered CoordTransformer calls (73+ instances)
// 2. Centralizes coordinate logic in one place
// 3. Provides type safety with LogicalPosition wrapper
// 4. Handles viewport/camera logic consistently
type CoordinateManager struct {
	dungeonWidth  int
	dungeonHeight int
	tileSize      int
	scaleFactor   int
	screenWidth   int
	screenHeight  int
}

// Viewport handles camera/centering logic that was scattered across files.
// Moved from graphics package to coords for unified coordinate management.
type Viewport struct {
	centerX, centerY int // logical coordinates of viewport center
	manager          *CoordinateManager
}

// NewCoordinateManager creates a new coordinate manager with the given screen configuration.
func NewCoordinateManager(screenData ScreenData) *CoordinateManager {
	return &CoordinateManager{
		dungeonWidth:  screenData.DungeonWidth,
		dungeonHeight: screenData.DungeonHeight,
		tileSize:      screenData.TileSize,
		scaleFactor:   screenData.ScaleFactor,
		screenWidth:   screenData.ScreenWidth,
		screenHeight:  screenData.ScreenHeight,
	}
}

// === CORE COORDINATE CONVERSIONS ===
// These replace the scattered CoordTransformer methods

// LogicalToIndex converts logical coordinates to flat map array index.
// Replaces: CoordTransformer.IndexFromLogicalXY()
func (cm *CoordinateManager) LogicalToIndex(pos LogicalPosition) int {
	return (pos.Y * cm.dungeonWidth) + pos.X
}

// IndexToLogical converts flat map array index to logical coordinates.
// Replaces: CoordTransformer.LogicalXYFromIndex()
func (cm *CoordinateManager) IndexToLogical(index int) LogicalPosition {
	x := index % cm.dungeonWidth
	y := index / cm.dungeonWidth
	return LogicalPosition{X: x, Y: y}
}

// LogicalToPixel converts logical coordinates to pixel coordinates (for rendering).
// Replaces: CoordTransformer.PixelsFromLogicalXY()
func (cm *CoordinateManager) LogicalToPixel(pos LogicalPosition) PixelPosition {
	return PixelPosition{
		X: pos.X * cm.tileSize,
		Y: pos.Y * cm.tileSize,
	}
}

// IndexToPixel converts flat map array index to pixel coordinates.
// Replaces: CoordTransformer.PixelsFromIndex()
func (cm *CoordinateManager) IndexToPixel(index int) PixelPosition {
	logical := cm.IndexToLogical(index)
	return cm.LogicalToPixel(logical)
}

// PixelToLogical converts pixel coordinates to logical coordinates.
// Replaces: CoordTransformer.LogicalXYFromPixels()
func (cm *CoordinateManager) PixelToLogical(pos PixelPosition) LogicalPosition {
	return LogicalPosition{
		X: pos.X / cm.tileSize,
		Y: pos.Y / cm.tileSize,
	}
}

// === VIEWPORT/CAMERA OPERATIONS ===
// These consolidate the scattered centering logic from coordtransform.go

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
// Replaces: OffsetFromCenter() and TransformLogicalCoordinates()
func (v *Viewport) LogicalToScreen(pos LogicalPosition) (float64, float64) {
	// Calculate offset to center the viewport
	offsetX := float64(v.manager.screenWidth)/2 - float64(v.centerX*v.manager.tileSize)*float64(v.manager.scaleFactor)
	offsetY := float64(v.manager.screenHeight)/2 - float64(v.centerY*v.manager.tileSize)*float64(v.manager.scaleFactor)

	// Convert logical to pixel, apply scaling and viewport offset
	scaledX := float64(pos.X*v.manager.tileSize) * float64(v.manager.scaleFactor)
	scaledY := float64(pos.Y*v.manager.tileSize) * float64(v.manager.scaleFactor)

	return scaledX + offsetX, scaledY + offsetY
}

// ScreenToLogical converts screen coordinates back to logical coordinates.
// Replaces: TransformPixelPosition()
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

// IsValidLogical checks if logical coordinates are within dungeon bounds.
func (cm *CoordinateManager) IsValidLogical(pos LogicalPosition) bool {
	return pos.X >= 0 && pos.X < cm.dungeonWidth &&
		pos.Y >= 0 && pos.Y < cm.dungeonHeight
}

// GetTilePositions converts a slice of indices to logical positions.
// Replaces: GetTilePositions() from coordcalc.go
func (cm *CoordinateManager) GetTilePositions(indices []int) []LogicalPosition {
	positions := make([]LogicalPosition, len(indices))
	for i, index := range indices {
		positions[i] = cm.IndexToLogical(index)
	}
	return positions
}

// GetTilePositionsAsCommon converts a slice of indices to coords.LogicalPosition slice.
// This is a compatibility method for legacy code that still uses coords.LogicalPosition.
// Since coords.LogicalPosition is now an alias to LogicalPosition, this just returns LogicalPosition slice.
func (cm *CoordinateManager) GetTilePositionsAsCommon(indices []int) []LogicalPosition {
	return cm.GetTilePositions(indices)
}

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
