package graphics

import "game_main/common"

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

func (c CoordinateTransformer) PixelsFromLogicalXY(x, y int) (int, int) {
	return x * ScreenInfo.TileSize, y * ScreenInfo.TileSize

}

func NewCoordTransformer(w, t int) CoordinateTransformer {
	return CoordinateTransformer{
		dungeonWidth: w,
		tileSize:     t,
	}
}

// A TileBasedShape returns indices that correspond to the tiles on the GameMap
// The TileBasedShape uses World Coordnates
func GetTilePositions(indices []int, dungeinWidth int) []common.Position {

	pos := make([]common.Position, len(indices))

	x, y := 0, 0
	for i, tileIndex := range indices {

		x, y = CoordTransformer.LogicalXYFromIndex(tileIndex)
		pos[i] = common.Position{X: x, Y: y}

	}

	return pos

}
