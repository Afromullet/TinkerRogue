package graphics

var UILocation = 0
var LevelHeight int = 0
var LevelWidth int = 0

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
func NewScreenData() ScreenData {
	g := ScreenData{
		DungeonWidth:  100,
		DungeonHeight: 80,
		ScreenWidth:   50,
		ScreenHeight:  50,

		UIHeight: 10,
	}

	tileWidthPixels := 64
	tileHeightPixels := 64

	g.ScaleX = float64(g.ScreenWidth) / float64(g.DungeonWidth)
	g.ScaleY = float64(g.ScreenHeight-g.UIHeight) / float64(g.DungeonHeight)

	g.TileWidth = int(float64(tileWidthPixels) * g.ScaleX)
	g.TileHeight = int(float64(tileHeightPixels) * g.ScaleY)

	LevelHeight = int(float64(g.DungeonHeight) * g.ScaleY)
	LevelWidth = int(float64(g.DungeonWidth) * g.ScaleX)

	UILocation = LevelWidth - g.ScreenWidth*2

	return g
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
