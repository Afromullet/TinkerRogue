package graphics

import (
	"game_main/common"

	"github.com/hajimehoshi/ebiten/v2"
)

// CoordinateManager provides a unified interface for all coordinate operations
// Addresses problems:
// 1. Eliminates scattered CoordTransformer calls (73+ instances)
// 2. Centralizes coordinate logic in one place
// 3. Provides type safety with Position wrapper
// 4. Handles viewport/camera logic consistently
type CoordinateManager struct {
	dungeonWidth  int
	dungeonHeight int
	tileSize      int
	scaleFactor   int
	screenWidth   int
	screenHeight  int
}

// LogicalPosition represents a position in the game world (tile-based)
// This becomes our single source of truth for positions
type LogicalPosition struct {
	X, Y int
}

// PixelPosition represents screen coordinates (only used for rendering)
type PixelPosition struct {
	X, Y int
}

// Viewport handles camera/centering logic that was scattered across files
type Viewport struct {
	centerX, centerY int // logical coordinates of viewport center
	manager          *CoordinateManager
}

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

// LogicalToIndex converts logical coordinates to flat map array index
// Replaces: CoordTransformer.IndexFromLogicalXY()
func (cm *CoordinateManager) LogicalToIndex(pos LogicalPosition) int {
	return (pos.Y * cm.dungeonWidth) + pos.X
}

// IndexToLogical converts flat map array index to logical coordinates
// Replaces: CoordTransformer.LogicalXYFromIndex()
func (cm *CoordinateManager) IndexToLogical(index int) LogicalPosition {
	x := index % cm.dungeonWidth
	y := index / cm.dungeonWidth
	return LogicalPosition{X: x, Y: y}
}

// LogicalToPixel converts logical coordinates to pixel coordinates (for rendering)
// Replaces: CoordTransformer.PixelsFromLogicalXY()
func (cm *CoordinateManager) LogicalToPixel(pos LogicalPosition) PixelPosition {
	return PixelPosition{
		X: pos.X * cm.tileSize,
		Y: pos.Y * cm.tileSize,
	}
}

// IndexToPixel converts flat map array index to pixel coordinates
// Replaces: CoordTransformer.PixelsFromIndex()
func (cm *CoordinateManager) IndexToPixel(index int) PixelPosition {
	logical := cm.IndexToLogical(index)
	return cm.LogicalToPixel(logical)
}

// PixelToLogical converts pixel coordinates to logical coordinates
// Replaces: CoordTransformer.LogicalXYFromPixels()
func (cm *CoordinateManager) PixelToLogical(pos PixelPosition) LogicalPosition {
	return LogicalPosition{
		X: pos.X / cm.tileSize,
		Y: pos.Y / cm.tileSize,
	}
}

// === VIEWPORT/CAMERA OPERATIONS ===
// These consolidate the scattered centering logic from coordtransform.go

func NewViewport(manager *CoordinateManager, centerPos LogicalPosition) *Viewport {
	return &Viewport{
		centerX: centerPos.X,
		centerY: centerPos.Y,
		manager: manager,
	}
}

// SetCenter updates the viewport center (typically player position)
func (v *Viewport) SetCenter(pos LogicalPosition) {
	v.centerX = pos.X
	v.centerY = pos.Y
}

// LogicalToScreen converts logical coordinates to screen coordinates with viewport centering
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

// ScreenToLogical converts screen coordinates back to logical coordinates
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

// IsValidLogical checks if logical coordinates are within dungeon bounds
func (cm *CoordinateManager) IsValidLogical(pos LogicalPosition) bool {
	return pos.X >= 0 && pos.X < cm.dungeonWidth &&
		pos.Y >= 0 && pos.Y < cm.dungeonHeight
}

// GetTilePositions converts a slice of indices to logical positions
// Replaces: GetTilePositions() from coordcalc.go
func (cm *CoordinateManager) GetTilePositions(indices []int) []LogicalPosition {
	positions := make([]LogicalPosition, len(indices))
	for i, index := range indices {
		positions[i] = cm.IndexToLogical(index)
	}
	return positions
}

// GetTilePositionsAsCommon converts a slice of indices to common.Position slice
// Convenience method for legacy code that still uses common.Position
func (cm *CoordinateManager) GetTilePositionsAsCommon(indices []int) []common.Position {
	positions := make([]common.Position, len(indices))
	for i, index := range indices {
		logical := cm.IndexToLogical(index)
		positions[i] = common.Position{X: logical.X, Y: logical.Y}
	}
	return positions
}


// === LOGICAL POSITION METHODS ===
// Distance and range methods for LogicalPosition to eliminate common.Position dependency

// ManhattanDistance calculates Manhattan distance between two logical positions
func (pos LogicalPosition) ManhattanDistance(other LogicalPosition) int {
	xDist := pos.X - other.X
	if xDist < 0 {
		xDist = -xDist
	}
	yDist := pos.Y - other.Y
	if yDist < 0 {
		yDist = -yDist
	}
	return xDist + yDist
}

// InRange checks if another logical position is within Manhattan distance
func (pos LogicalPosition) InRange(other LogicalPosition, distance int) bool {
	return pos.ManhattanDistance(other) <= distance
}

// Used for drawing only a section of the map.
// Different from TileSquare. Tilesquare returns indices
// DrawableSection uses logical coordinates to center the square around a point
type DrawableSection struct {
	StartX int
	StartY int
	EndX   int
	EndY   int
}

func NewDrawableSection(x, y, size int) DrawableSection {

	s := DrawableSection{}
	halfSize := size / 2

	s.StartX = x - halfSize
	s.StartY = y - halfSize

	if s.StartX < 0 {
		s.StartX = 0
	}
	if s.StartY < 0 {
		s.StartY = 0
	}
	s.EndX = x + halfSize
	s.EndY = y + halfSize

	if s.EndX >= ScreenInfo.DungeonWidth {
		s.EndX = ScreenInfo.DungeonWidth - 1
	}
	if s.EndY >= ScreenInfo.DungeonHeight {
		s.EndY = ScreenInfo.DungeonHeight - 1
	}

	return s

}

/*
Takes place of this in the drawmap

	scaledTileSize := graphics.ScreenInfo.TileSize * graphics.ScreenInfo.ScaleFactor
		scaledCenterOffsetX := float64(graphics.ScreenInfo.ScreenWidth)/2 - float64(playerPos.X*scaledTileSize)
		scaledCenterOffsetY := float64(graphics.ScreenInfo.ScreenHeight)/2 - float64(playerPos.Y*scaledTileSize)
		op.GeoM.Translate(
			float64(tile.PixelX)*float64(graphics.ScreenInfo.ScaleFactor)+scaledCenterOffsetX,
			float64(tile.PixelY)*float64(graphics.ScreenInfo.ScaleFactor)+scaledCenterOffsetY,
		)
*/
// DEPRECATED: Use Viewport.LogicalToScreen instead
// Offset the origin from the center
func OffsetFromCenter(centerX, centerY, pixelX, pixelY int, sc ScreenData) (float64, float64) {

	offsetX, offsetY := calculateCenterOffset(centerX, centerY, sc)

	finalX := float64(pixelX)*float64(sc.ScaleFactor) + offsetX
	finalY := float64(pixelY)*float64(sc.ScaleFactor) + offsetY

	return finalX, finalY

}

// DEPRECATED: Use Viewport.ScreenToLogical instead
// Used for when we want to get the cursor position from a centered map
// centerX and centerY are logical coordinates
func TransformPixelPosition(centerX, centerY, pixelX, pixelY int, sc ScreenData) (int, int) {

	offsetX, offsetY := calculateCenterOffset(centerX, centerY, sc)

	// Reverse the translation
	uncenteredX := float64(pixelX) - offsetX
	uncenteredY := float64(pixelY) - offsetY

	// Reverse the scaling
	finalX := uncenteredX / float64(sc.ScaleFactor)
	finalY := uncenteredY / float64(sc.ScaleFactor)

	return int(finalX), int(finalY)
}

// DEPRECATED: Use Viewport.LogicalToScreen instead
// New function for transforming logical coordinates
func TransformLogicalCoordinates(centerX, centerY, logicalX, logicalY int, sc ScreenData) (int, int) {
	offsetX, offsetY := calculateCenterOffset(centerX, centerY, sc)

	// Apply scaling
	scaledX := float64(logicalX) * float64(sc.ScaleFactor)
	scaledY := float64(logicalY) * float64(sc.ScaleFactor)

	// Apply translation
	finalX := scaledX + offsetX
	finalY := scaledY + offsetY

	return int(finalX), int(finalY)
}

// DEPRECATED: Internal function used by deprecated coordinate functions
func calculateCenterOffset(centerX, centerY int, sc ScreenData) (float64, float64) {
	// Calculate the offset to center the map on the given logical center coordinates
	offsetX := float64(sc.ScreenWidth)/2 - float64(centerX*sc.TileSize)*float64(sc.ScaleFactor)
	offsetY := float64(sc.ScreenHeight)/2 - float64(centerY*sc.TileSize)*float64(sc.ScaleFactor)

	return offsetX, offsetY
}

// DEPRECATED: Use Viewport.ScreenToLogical with ebiten.CursorPosition instead
// Takes the players position as input because the map is centered on the player
// Which means that the pixel position will have to be transformed
func CursorPosition(playerPosition common.Position) (int, int) {

	cursorX, cursorY := ebiten.CursorPosition()
	if MAP_SCROLLING_ENABLED {
		cursorX, cursorY = TransformPixelPosition(playerPosition.X, playerPosition.Y, cursorX, cursorY, ScreenInfo)

	}

	return cursorX, cursorY
}
