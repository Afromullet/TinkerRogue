package worldmap

import (
	"game_main/common"
	"game_main/world/coords"
	"game_main/visual/graphics"

	"github.com/hajimehoshi/ebiten/v2"
)

// ========================================
// HELPER FUNCTIONS
// ========================================

// positionToIndex converts x, y coordinates to a flat array index
func positionToIndex(x, y, width int) int {
	return y*width + x
}

// selectRandomImage returns a random image from the slice, or nil if empty
func selectRandomImage(images []*ebiten.Image) *ebiten.Image {
	if len(images) == 0 {
		return nil
	}
	return images[common.GetRandomBetween(0, len(images)-1)]
}

// getBiomeImages returns wall and floor images for a biome, falling back to defaults
func getBiomeImages(images TileImageSet, biome Biome) (wallImages, floorImages []*ebiten.Image) {
	wallImages = images.WallImages
	floorImages = images.FloorImages

	biomeTileSet := images.BiomeImages[biome]
	if biomeTileSet != nil {
		if len(biomeTileSet.WallImages) > 0 {
			wallImages = biomeTileSet.WallImages
		}
		if len(biomeTileSet.FloorImages) > 0 {
			floorImages = biomeTileSet.FloorImages
		}
	}

	return wallImages, floorImages
}

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
			index := positionToIndex(x, y, width)
			wallImg := selectRandomImage(images.WallImages)

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
func carveRoom(result *GenerationResult, room Rect, width int, images TileImageSet) {
	for y := room.Y1 + 1; y < room.Y2; y++ {
		for x := room.X1 + 1; x < room.X2; x++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := positionToIndex(x, y, width)

			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			result.Tiles[index].Image = selectRandomImage(images.FloorImages)

			// Add to valid positions
			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}

// carveTunnel creates a corridor along a horizontal or vertical line
func carveTunnel(result *GenerationResult, start, end, fixed, width int, isHorizontal bool, images TileImageSet) {
	for i := min(start, end); i <= max(start, end); i++ {
		var logicalPos coords.LogicalPosition
		var index int

		if isHorizontal {
			logicalPos = coords.LogicalPosition{X: i, Y: fixed}
			index = positionToIndex(i, fixed, width)
		} else {
			logicalPos = coords.LogicalPosition{X: fixed, Y: i}
			index = positionToIndex(fixed, i, width)
		}

		if index >= 0 && index < len(result.Tiles) {
			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			result.Tiles[index].Image = selectRandomImage(images.FloorImages)
			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}

// carveHorizontalTunnel creates horizontal corridor
func carveHorizontalTunnel(result *GenerationResult, x1, x2, y, width int, images TileImageSet) {
	carveTunnel(result, x1, x2, y, width, true, images)
}

// carveVerticalTunnel creates vertical corridor
func carveVerticalTunnel(result *GenerationResult, y1, y2, x, width int, images TileImageSet) {
	carveTunnel(result, y1, y2, x, width, false, images)
}
