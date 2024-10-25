package graphics

import "github.com/hajimehoshi/ebiten/v2"

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
//Offset the origin from the center
func OffsetFromCenter(centerX, centerY, pixelX, pixelY int, sc ScreenData) (float64, float64) {

	offsetX, offsetY := calculateCenterOffset(centerX, centerY, sc)

	finalX := float64(pixelX)*float64(sc.ScaleFactor) + offsetX
	finalY := float64(pixelY)*float64(sc.ScaleFactor) + offsetY

	return finalX, finalY

}

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

func calculateCenterOffset(centerX, centerY int, sc ScreenData) (float64, float64) {
	// Calculate the offset to center the map on the given logical center coordinates
	offsetX := float64(sc.ScreenWidth)/2 - float64(centerX*sc.TileSize)*float64(sc.ScaleFactor)
	offsetY := float64(sc.ScreenHeight)/2 - float64(centerY*sc.TileSize)*float64(sc.ScaleFactor)

	return offsetX, offsetY
}

func CursorPosition(logicalX, logicalY int) (int, int) {

	cursorX, cursorY := ebiten.CursorPosition()
	if MAP_SCROLLING_ENABLED {
		cursorX, cursorY = TransformPixelPosition(logicalX, logicalY, cursorX, cursorY, ScreenInfo)

	}

	return cursorX, cursorY
}
