package graphics

var UILocationX = 0
var LevelHeight int = 0
var LevelWidth int = 0
var StatsUIOffset int = 1000 //Offset to where the UI starts

// Contains the data we need to render the map. Also used for coordinate conversions
type ScreenData struct {
	ScreenWidth  int
	ScreenHeight int

	TileWidth     int
	TileHeight    int
	UIHeight      int
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

		UIHeight: 10,
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

	UILocationX = LevelWidth / 2

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
	gd := NewScreenData()
	return (y * gd.DungeonWidth) + x
}

// Gets XY coordinates from the map tile index
func XYFromIndex(i int) (int, int) {
	gd := NewScreenData()
	return i % gd.DungeonWidth, i / gd.DungeonWidth
}

// Gets the pixels from the index
func PixelsFromIndex(i int) (int, int) {
	gd := NewScreenData()
	x, y := XYFromIndex(i)
	return x * gd.TileWidth, y * gd.TileHeight
}

// Return the Grid X,Y coordinates from pixel positions
func XYFromPixels(x, y int) (int, int) {
	gd := NewScreenData()
	return x / gd.TileWidth, y / gd.TileHeight

}
