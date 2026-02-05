package behavior

import "game_main/world/coords"

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

// IterateViewport iterates over tiles within a viewport around a center position.
// Used by visualizers to only process visible tiles.
func IterateViewport(center coords.LogicalPosition, viewportSize int, callback GridIterator) {
	minX := center.X - viewportSize/2
	maxX := center.X + viewportSize/2
	minY := center.Y - viewportSize/2
	maxY := center.Y + viewportSize/2

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			callback(coords.LogicalPosition{X: x, Y: y})
		}
	}
}
