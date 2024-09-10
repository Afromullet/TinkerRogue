package graphics

// Contains the data we need to render the map
type ScreenData struct {
	ScreenWidth  int
	ScreenHeight int
	TileWidth    int
	TileHeight   int
	UIHeight     int
}

func NewScreenData() ScreenData {
	g := ScreenData{
		ScreenWidth:  50,
		ScreenHeight: 50,
		TileWidth:    32,
		TileHeight:   32,
		UIHeight:     10,
	}

	LevelHeight = g.ScreenHeight - g.UIHeight

	//Todo refactor this. Only doing this here at the moment due to hwo we use NewScreenData in the code whenever
	//We want to access the parameters. Done that way because it was originally intended to eb stateless.
	//Probably do not need it to be stateless.

	return g
}

// IndexFromXY gets the index of the map array from a given X,Y TILE coordinate.
// This coordinate is logical tiles, not pixels.
func IndexFromXY(x int, y int) int {
	gd := NewScreenData()
	return (y * gd.ScreenWidth) + x
}

// Gets XY coordinates from the map tile index
func XYFromIndex(i int) (int, int) {
	gd := NewScreenData()
	return i % gd.ScreenWidth, i / gd.ScreenWidth
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
