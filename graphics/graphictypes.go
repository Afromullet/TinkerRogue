package graphics

import (
	"game_main/coords"

	"github.com/hajimehoshi/ebiten/v2"
)

var GreenColorMatrix = ColorMatrix{0, 1, 0, 1, true}
var RedColorMatrix = ColorMatrix{1, 0, 0, 1, true}

// ScreenInfo and CoordManager now come from coords package
var ScreenInfo = coords.NewScreenData()
var CoordManager = coords.CoordManager

var ViewableSquareSize = 30
var MAP_SCROLLING_ENABLED = true

// var StatsUIOffset int = 1000 //Offset to where the UI starts
var StatsUIOffset int = 1000 //Offset to where the UI starts

// CursorPosition gets the cursor position relative to the player position
func CursorPosition(playerPos coords.LogicalPosition) (int, int) {
	// Get current cursor position from ebiten
	cursorX, cursorY := ebiten.CursorPosition()

	// Transform pixel coordinates using viewport logic
	return TransformPixelPosition(playerPos.X, playerPos.Y, cursorX, cursorY, ScreenInfo)
}

// OffsetFromCenter calculates screen offset for centering the map around a player position
func OffsetFromCenter(centerX, centerY, tileX, tileY int, screenData coords.ScreenData) (float64, float64) {
	// Calculate offset to center the viewport
	offsetX := float64(screenData.ScreenWidth)/2 - float64(centerX*screenData.TileSize)*float64(screenData.ScaleFactor)
	offsetY := float64(screenData.ScreenHeight)/2 - float64(centerY*screenData.TileSize)*float64(screenData.ScaleFactor)

	// Apply scaling and viewport offset to tile position
	scaledX := float64(tileX) * float64(screenData.ScaleFactor)
	scaledY := float64(tileY) * float64(screenData.ScaleFactor)

	return scaledX + offsetX, scaledY + offsetY
}

// TransformPixelPosition transforms pixel coordinates using viewport logic
func TransformPixelPosition(playerX, playerY, cursorX, cursorY int, screenData coords.ScreenData) (int, int) {
	// Create viewport centered on player
	manager := coords.NewCoordinateManager(screenData)
	viewport := coords.NewViewport(manager, coords.LogicalPosition{X: playerX, Y: playerY})

	// Convert cursor screen coordinates back to logical coordinates
	logicalPos := viewport.ScreenToLogical(cursorX, cursorY)

	// Convert to pixel coordinates
	pixelPos := manager.LogicalToPixel(logicalPos)
	return pixelPos.X, pixelPos.Y
}

// MouseToLogicalPosition converts mouse screen coordinates to logical tile coordinates
// Handles both MAP_SCROLLING_ENABLED modes correctly
func MouseToLogicalPosition(mouseX, mouseY int, centerPos coords.LogicalPosition) coords.LogicalPosition {
	if MAP_SCROLLING_ENABLED {
		// Use viewport transformation when scrolling is enabled
		manager := coords.NewCoordinateManager(ScreenInfo)
		viewport := coords.NewViewport(manager, centerPos)
		return viewport.ScreenToLogical(mouseX, mouseY)
	}
	// When scrolling is disabled, convert screen pixel coords directly to logical
	pixelPos := coords.PixelPosition{X: mouseX, Y: mouseY}
	return coords.CoordManager.PixelToLogical(pixelPos)
}
