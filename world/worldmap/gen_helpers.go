package worldmap

import (
	"game_main/common"
	"game_main/visual/graphics"
	"game_main/world/coords"

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

// ========================================
// TILE CARVING HELPERS
// ========================================

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
			if x >= 0 && x < width && fromY >= 0 && fromY < height {
				terrainMap[fromY*width+x] = true
			}
		}
	} else {
		for x := fromX; x >= toX; x-- {
			if x >= 0 && x < width && fromY >= 0 && fromY < height {
				terrainMap[fromY*width+x] = true
			}
		}
	}

	// Then vertical
	if fromY < toY {
		for y := fromY; y <= toY; y++ {
			if toX >= 0 && toX < width && y >= 0 && y < height {
				terrainMap[y*width+toX] = true
			}
		}
	} else {
		for y := fromY; y >= toY; y-- {
			if toX >= 0 && toX < width && y >= 0 && y < height {
				terrainMap[y*width+toX] = true
			}
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

// ========================================
// SHARED TERRAIN UTILITIES
// ========================================

// convertTerrainMapToTiles converts a boolean terrain map to actual Tile objects.
// terrainMap true = walkable floor, false = wall. Assigns biome images and populates ValidPositions.
func convertTerrainMapToTiles(result *GenerationResult, terrainMap []bool, width, height int, images TileImageSet, biome Biome) {
	wallImages, floorImages := getBiomeImages(images, biome)

	result.BiomeMap = make([]Biome, width*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := positionToIndex(x, y, width)
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			pixelX := x * graphics.ScreenInfo.TileSize
			pixelY := y * graphics.ScreenInfo.TileSize

			result.BiomeMap[idx] = biome

			if terrainMap[idx] {
				floorImage := selectRandomImage(floorImages)
				tile := NewTile(pixelX, pixelY, logicalPos, false, floorImage, FLOOR, false)
				tile.Biome = biome
				result.Tiles[idx] = &tile
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			} else {
				wallImage := selectRandomImage(wallImages)
				tile := NewTile(pixelX, pixelY, logicalPos, true, wallImage, WALL, false)
				tile.Biome = biome
				result.Tiles[idx] = &tile
			}
		}
	}
}

// scoreTerrainOpenness counts walkable tiles within a radius of (cx, cy) on a terrain map.
func scoreTerrainOpenness(terrainMap []bool, cx, cy, radius, width, height int) int {
	score := 0
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			nx, ny := cx+dx, cy+dy
			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				if terrainMap[ny*width+nx] {
					score++
				}
			}
		}
	}
	return score
}

// findBestOpenPosition finds the walkable position with the highest openness score
// within the given bounding box. Returns {-1, -1} if no walkable tile is found.
func findBestOpenPosition(terrainMap []bool, width, height, xMin, xMax, yMin, yMax, scanRadius int) coords.LogicalPosition {
	bestScore := -1
	bestPos := coords.LogicalPosition{X: -1, Y: -1}

	// Clamp scan bounds so the radius doesn't go off-map
	if xMin < scanRadius {
		xMin = scanRadius
	}
	if yMin < scanRadius {
		yMin = scanRadius
	}
	if xMax >= width-scanRadius {
		xMax = width - scanRadius - 1
	}
	if yMax >= height-scanRadius {
		yMax = height - scanRadius - 1
	}

	for y := yMin; y <= yMax; y++ {
		for x := xMin; x <= xMax; x++ {
			if !terrainMap[y*width+x] {
				continue
			}

			score := scoreTerrainOpenness(terrainMap, x, y, scanRadius, width, height)
			if score > bestScore {
				bestScore = score
				bestPos = coords.LogicalPosition{X: x, Y: y}
			}
		}
	}

	return bestPos
}

// isTooCloseToAny checks if (px, py) is within minSpacing of any position in placed
// using Chebyshev distance (both dx AND dy must be < minSpacing to be "too close").
func isTooCloseToAny(px, py int, placed [][2]int, minSpacing int) bool {
	for _, pp := range placed {
		dx := px - pp[0]
		dy := py - pp[1]
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dx < minSpacing && dy < minSpacing {
			return true
		}
	}
	return false
}

// tryPlace2x2PillarOnTerrain checks that a 2x2 area at (px, py) is fully walkable,
// then sets it to wall. Returns true if placed, false if blocked or out of bounds.
func tryPlace2x2PillarOnTerrain(terrainMap []bool, px, py, width, height int) bool {
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			nx, ny := px+dx, py+dy
			if nx < 0 || nx >= width || ny < 0 || ny >= height || !terrainMap[ny*width+nx] {
				return false
			}
		}
	}
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			terrainMap[(py+dy)*width+(px+dx)] = false
		}
	}
	return true
}
