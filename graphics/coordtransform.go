package graphics

var ViewableSquareSize = 30

var MAP_SCROLLING_ENABLED = true

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

func OffsetFromPlayer(playerX, playerY, originX, originY int, sc ScreenData) (float64, float64) {

	// Calculate the scaled tile size
	scaledTileSize := sc.TileSize * sc.ScaleFactor

	// Calculate the position to center the scaled map

	scaledCenterOffsetX := float64(sc.ScreenWidth)/2 - float64(playerX*scaledTileSize)
	scaledCenterOffsetY := float64(sc.ScreenHeight)/2 - float64(playerY*scaledTileSize)

	finalX := float64(originX)*float64(sc.ScaleFactor) + scaledCenterOffsetX
	finalY := float64(originY)*float64(sc.ScaleFactor) + scaledCenterOffsetY

	return finalX, finalY

}

// Used for when we want to get the cursor position from a centered map
func TransformCursorPosition(cursorX, cursorY int, playerX, playerY int, sc ScreenData) (int, int) {
	// Calculate the scaled tile size
	scaledTileSize := sc.TileSize * sc.ScaleFactor

	// Calculate the position to center the scaled map
	scaledCenterOffsetX := float64(sc.ScreenWidth)/2 - float64(playerX*scaledTileSize)
	scaledCenterOffsetY := float64(sc.ScreenHeight)/2 - float64(playerY*scaledTileSize)

	// Reverse the translation
	uncenteredX := float64(cursorX) - scaledCenterOffsetX
	uncenteredY := float64(cursorY) - scaledCenterOffsetY

	// Reverse the scaling
	finalX := uncenteredX / float64(sc.ScaleFactor)
	finalY := uncenteredY / float64(sc.ScaleFactor)

	return int(finalX), int(finalY)
}
