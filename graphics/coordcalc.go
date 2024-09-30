package graphics

var LevelHeight int = 0
var LevelWidth int = 0
var StatsUIOffset int = 1000 //Offset to where the UI starts

var ScreenInfo = NewScreenData()

// Contains the data we need to render the map. Also used for coordinate conversions
type ScreenData struct {
	ScreenWidth  int
	ScreenHeight int

	TileWidth  int
	TileHeight int

	DungeonWidth  int
	DungeonHeight int
	ScaleX        float64
	ScaleY        float64
}

// Todo, need to do less work inside of here because this is called every time we do coordinate conversions
// All of this needs to only be calculated once. Currently, everything that needs Screen Information calls
// NewScreenData, which calculates things again. Those values don't change, so it only needs to be calcualted once
func NewScreenData() ScreenData {
	g := ScreenData{
		DungeonWidth:  100,
		DungeonHeight: 80,
	}
	tileWidthPixels := 32
	tileHeightPixels := 32

	// Calculate the scale based on the dungeon dimensions

	g.ScaleX = float64(g.DungeonWidth) * float64(tileWidthPixels) / float64(g.DungeonWidth)
	g.ScaleY = float64(g.DungeonHeight) * float64(tileHeightPixels) / float64(g.DungeonHeight)

	g.TileWidth = int(g.ScaleX)
	g.TileHeight = int(g.ScaleY)

	LevelHeight = int(float64(g.DungeonHeight) * g.ScaleY)
	LevelWidth = int(float64(g.DungeonWidth) * g.ScaleX)

	return g
}

func (s ScreenData) GetCanvasWidth() int {

	return int((float64(s.TileWidth * s.DungeonWidth)))

}

func (s ScreenData) GetCanvasHeight() int {

	return int((float64(s.TileHeight * s.DungeonHeight)))

}

// IndexFromXY gets the index of the map array from a given X,Y TILE coordinate.
// This coordinate is logical tiles, not pixels.
func IndexFromXY(x int, y int) int {

	return (y * ScreenInfo.DungeonWidth) + x
}

// Gets XY coordinates from the map tile index
func XYFromIndex(i int) (int, int) {

	return i % ScreenInfo.DungeonWidth, i / ScreenInfo.DungeonWidth
}

// Gets the pixels from the index
func PixelsFromIndex(i int) (int, int) {

	x, y := XYFromIndex(i)
	return x * ScreenInfo.TileWidth, y * ScreenInfo.TileHeight
}

// Return the Grid X,Y coordinates from pixel positions
func XYFromPixels(x, y int) (int, int) {

	return x / ScreenInfo.TileWidth, y / ScreenInfo.TileHeight

}

// Calculates the offset to the center for an X,Y position
// Applies to the translate when drawing
// I.E op.GeoM.Translate(float64(tilePixelX+centerOffsetX), float64(tilePixelY+centerOffsetY))
func CenterOffset(x, y int) (int, int) {
	screenCenterX := LevelWidth / 2
	screenCenterY := LevelHeight / 2

	// Calculate the player's position in pixels
	pixelX := x * ScreenInfo.TileWidth
	pixelY := y * ScreenInfo.TileHeight

	return screenCenterX - pixelX, screenCenterY - pixelY
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
