package graphics

import "github.com/hajimehoshi/ebiten/v2"

var ViewableSquareSize = 30
var ScaleFactor = 3
var MAP_SCROLLING_ENABLED = true
var ScreenBoundsX = 0
var ScreenBoundsY = 0

// Contains all of the structs and functions related to handling a "scrolling map" that centers around the player

// Calculates the offset to the center for a logical X,Y
// Applies to the translate when drawing
// I.E op.GeoM.Translate(float64(tilePixelX+centerOffsetX), float64(tilePixelY+centerOffsetY))
func CenterOffset(x, y int) (int, int) {
	screenCenterX := LevelWidth / 2
	screenCenterY := LevelHeight / 2

	// Calculate the player's position in pixels
	pixelX := x * ScreenInfo.TileWidth * ScaleFactor
	pixelY := y * ScreenInfo.TileHeight * ScaleFactor

	return screenCenterX - pixelX, screenCenterY - pixelY
}

func CenterOffsetPixel(x, y int) (int, int) {
	screenCenterX := LevelWidth / 2
	screenCenterY := LevelHeight / 2

	return screenCenterX - x, screenCenterY - y
}

// Gets the startX and startY for a square using logical X,Y, not pixel X,Y
func SquareStartXY(x, y, size int) (int, int) {

	halfSize := size / 2

	startX := x - halfSize
	startY := y - halfSize

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	return startX, startY

}

// Gets the endX and endX for a square using logical X,Y, not pixel X,Y
func SquareEndXY(x, y, size int) (int, int) {

	halfSize := size / 2

	endX := x + halfSize
	endY := y + halfSize

	if endX >= ScreenInfo.DungeonWidth {
		endX = ScreenInfo.DungeonWidth - 1
	}
	if endY >= ScreenInfo.DungeonHeight {
		endY = ScreenInfo.DungeonHeight - 1
	}

	return endX, endY

}

func CursorWithOffset() (int, int) {
	return CenterOffsetPixel(ebiten.CursorPosition())
}
