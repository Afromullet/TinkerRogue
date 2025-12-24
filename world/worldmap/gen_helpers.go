package worldmap

import (
	"game_main/common"
	"game_main/world/coords"
	"game_main/visual/graphics"

	"github.com/hajimehoshi/ebiten/v2"
)

// createEmptyTiles initializes all tiles as walls
// Optimized to reduce allocations: allocates one contiguous slice of Tile values
// and reuses pointers to those values, avoiding per-tile heap allocations
func createEmptyTiles(width, height int, images TileImageSet) []*Tile {
	numTiles := width * height

	// Allocate all tiles in one contiguous slice (single allocation instead of thousands)
	tileValues := make([]Tile, numTiles)

	// Create pointer slice that points into the contiguous allocation
	tiles := make([]*Tile, numTiles)

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)

			var wallImg *ebiten.Image
			if len(images.WallImages) > 0 {
				wallImg = images.WallImages[common.GetRandomBetween(0, len(images.WallImages)-1)]
			}

			// Initialize tile directly in the contiguous slice
			tileValues[index] = NewTile(
				x*graphics.ScreenInfo.TileSize,
				y*graphics.ScreenInfo.TileSize,
				logicalPos, true, wallImg, WALL, false,
			)

			// Store pointer to the tile in the contiguous slice
			tiles[index] = &tileValues[index]
		}
	}

	return tiles
}

// carveRoom converts wall tiles to floor tiles within room bounds
func carveRoom(result *GenerationResult, room Rect, images TileImageSet) {
	for y := room.Y1 + 1; y < room.Y2; y++ {
		for x := room.X1 + 1; x < room.X2; x++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)

			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			if len(images.FloorImages) > 0 {
				result.Tiles[index].image = images.FloorImages[common.GetRandomBetween(0, len(images.FloorImages)-1)]
			}

			// Add to valid positions
			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}

// carveHorizontalTunnel creates horizontal corridor
func carveHorizontalTunnel(result *GenerationResult, x1, x2, y int, images TileImageSet) {
	for x := min(x1, x2); x <= max(x1, x2); x++ {
		logicalPos := coords.LogicalPosition{X: x, Y: y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)

		if index >= 0 && index < len(result.Tiles) {
			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			if len(images.FloorImages) > 0 {
				result.Tiles[index].image = images.FloorImages[common.GetRandomBetween(0, len(images.FloorImages)-1)]
			}
			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}

// carveVerticalTunnel creates vertical corridor
func carveVerticalTunnel(result *GenerationResult, y1, y2, x int, images TileImageSet) {
	for y := min(y1, y2); y <= max(y1, y2); y++ {
		logicalPos := coords.LogicalPosition{X: x, Y: y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)

		if index >= 0 && index < len(result.Tiles) {
			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			if len(images.FloorImages) > 0 {
				result.Tiles[index].image = images.FloorImages[common.GetRandomBetween(0, len(images.FloorImages)-1)]
			}
			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}
