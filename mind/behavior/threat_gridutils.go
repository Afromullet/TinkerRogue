package behavior

import "game_main/core/coords"

// GridIterator is a callback function for iterating over map grid positions.
type GridIterator func(pos coords.LogicalPosition)

// IterateMapGrid iterates over all tiles within map bounds, calling the callback for each position.
// This centralizes the common map iteration pattern used by multiple layers.
func IterateMapGrid(callback GridIterator) {
	mapWidth := coords.CoordManager.GetDungeonWidth()
	mapHeight := coords.CoordManager.GetDungeonHeight()

	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			callback(coords.LogicalPosition{X: x, Y: y})
		}
	}
}
