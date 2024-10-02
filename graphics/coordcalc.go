package graphics

var StatsUIOffset int = 1000 //Offset to where the UI starts

type CoordinateTransformer struct {
	dungeonWidth int
	tileSize     int
}

// IndexFromXY gets the index of the map array from a given X,Y TILE coordinate.
// This coordinate is logical tiles, not pixels.
func (c CoordinateTransformer) IndexFromLogicalXY(x int, y int) int {

	return (y * c.dungeonWidth) + x
}

// Gets the pixels from the index
func (c CoordinateTransformer) PixelsFromIndex(i int) (int, int) {

	x, y := i%ScreenInfo.DungeonWidth, i/ScreenInfo.DungeonWidth
	return x * ScreenInfo.TileSize, y * ScreenInfo.TileSize
}

// Return the Grid X,Y coordinates from pixel positions
func (c CoordinateTransformer) LogicalXYFromPixels(x, y int) (int, int) {

	return x / ScreenInfo.TileSize, y / ScreenInfo.TileSize

}

// Gets the logical coordinates from the index
func (c CoordinateTransformer) LogicalXYFromIndex(i int) (int, int) {
	x, y := i%ScreenInfo.DungeonWidth, i/ScreenInfo.DungeonWidth
	return x, y
}

func (c CoordinateTransformer) PixelsFromLogicalXY(x, y, tileSize int) (int, int) {
	return x * tileSize, y * tileSize

}

func NewCoordTransformer(w, t int) CoordinateTransformer {
	return CoordinateTransformer{
		dungeonWidth: w,
		tileSize:     t,
	}
}
