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

// ========================================
// SHARED CONNECTIVITY HELPERS
// ========================================

// floodFillRegion finds all connected walkable tiles starting from (startX, startY).
// terrainMap: true = walkable, false = obstacle.
// Returns slice of flat indices belonging to this connected region.
func floodFillRegion(terrainMap []bool, visited []bool, startX, startY, width, height int) []int {
	var region []int
	queue := [][2]int{{startX, startY}}
	visited[startY*width+startX] = true

	for len(queue) > 0 {
		x, y := queue[0][0], queue[0][1]
		queue = queue[1:]

		region = append(region, y*width+x)

		neighbors := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
		for _, dir := range neighbors {
			nx, ny := x+dir[0], y+dir[1]
			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				nidx := ny*width + nx
				if !visited[nidx] && terrainMap[nidx] {
					visited[nidx] = true
					queue = append(queue, [2]int{nx, ny})
				}
			}
		}
	}

	return region
}

// carveCorridorBetween creates an L-shaped corridor between two flat indices on the terrain map.
// Sets traversed cells to walkable (true).
func carveCorridorBetween(terrainMap []bool, width, height, fromIdx, toIdx int) {
	fromX, fromY := fromIdx%width, fromIdx/width
	toX, toY := toIdx%width, toIdx/width

	// Horizontal first
	if fromX < toX {
		for x := fromX; x <= toX; x++ {
			terrainMap[fromY*width+x] = true
		}
	} else {
		for x := fromX; x >= toX; x-- {
			terrainMap[fromY*width+x] = true
		}
	}

	// Then vertical
	if fromY < toY {
		for y := fromY; y <= toY; y++ {
			terrainMap[y*width+toX] = true
		}
	} else {
		for y := fromY; y >= toY; y-- {
			terrainMap[y*width+toX] = true
		}
	}
}

// ensureTerrainConnectivity finds all disconnected walkable regions and connects
// them to the largest region via L-shaped corridors.
func ensureTerrainConnectivity(terrainMap []bool, width, height int) {
	visited := make([]bool, len(terrainMap))
	var largestRegion []int
	maxSize := 0

	// Find all regions and track the largest
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if !visited[idx] && terrainMap[idx] {
				region := floodFillRegion(terrainMap, visited, x, y, width, height)
				if len(region) > maxSize {
					largestRegion = region
					maxSize = len(region)
				}
			}
		}
	}

	// If no walkable region, make center 50% walkable
	if maxSize == 0 {
		for y := height / 4; y < (height * 3 / 4); y++ {
			for x := width / 4; x < (width * 3 / 4); x++ {
				terrainMap[y*width+x] = true
			}
		}
		return
	}

	// Mark largest region as visited
	visited = make([]bool, len(terrainMap))
	for _, idx := range largestRegion {
		visited[idx] = true
	}

	// Connect all other regions to the largest
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if !visited[idx] && terrainMap[idx] {
				region := floodFillRegion(terrainMap, visited, x, y, width, height)
				if len(region) > 0 {
					carveCorridorBetween(terrainMap, width, height, largestRegion[0], region[0])
				}
			}
		}
	}
}
