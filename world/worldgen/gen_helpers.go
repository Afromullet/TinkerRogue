package worldgen

import (
	"game_main/common"
	"game_main/visual/graphics"
	"game_main/world/coords"
	"game_main/world/worldmapcore"

	"github.com/hajimehoshi/ebiten/v2"
)

// ========================================
// HELPER FUNCTIONS
// ========================================

// PositionToIndex converts x, y coordinates to a flat array index via CoordManager.
func PositionToIndex(x, y int) int {
	return coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: x, Y: y})
}

// SelectRandomImage returns a random image from the slice, or nil if empty
func SelectRandomImage(images []*ebiten.Image) *ebiten.Image {
	if len(images) == 0 {
		return nil
	}
	return images[common.GetRandomBetween(0, len(images)-1)]
}

// GetBiomeImages returns wall and floor images for a biome, falling back to defaults
func GetBiomeImages(images worldmapcore.TileImageSet, biome worldmapcore.Biome) (wallImages, floorImages []*ebiten.Image) {
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

// CreateEmptyTiles initializes all tiles as walls
// Optimized to reduce allocations: allocates one contiguous slice of Tile values
// and reuses pointers to those values, avoiding per-tile heap allocations
func CreateEmptyTiles(width, height int, images worldmapcore.TileImageSet) []*worldmapcore.Tile {
	numTiles := width * height

	// Allocate all tiles in one contiguous slice (single allocation instead of thousands)
	tileValues := make([]worldmapcore.Tile, numTiles)

	// Create pointer slice that points into the contiguous allocation
	tiles := make([]*worldmapcore.Tile, numTiles)

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := PositionToIndex(x, y)
			wallImg := SelectRandomImage(images.WallImages)

			// Initialize tile directly in the contiguous slice
			tileValues[index] = worldmapcore.NewTile(
				x*graphics.ScreenInfo.TileSize,
				y*graphics.ScreenInfo.TileSize,
				logicalPos, true, wallImg, worldmapcore.WALL, false,
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

// CarveRoom converts wall tiles to floor tiles within room bounds
func CarveRoom(result *worldmapcore.GenerationResult, room worldmapcore.Rect, width int, images worldmapcore.TileImageSet) {
	for y := room.Y1 + 1; y < room.Y2; y++ {
		for x := room.X1 + 1; x < room.X2; x++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := PositionToIndex(x, y)

			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = worldmapcore.FLOOR
			result.Tiles[index].Image = SelectRandomImage(images.FloorImages)

			// Add to valid positions
			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}

// CarveTunnel creates a corridor along a horizontal or vertical line
func CarveTunnel(result *worldmapcore.GenerationResult, start, end, fixed, width int, isHorizontal bool, images worldmapcore.TileImageSet) {
	for i := min(start, end); i <= max(start, end); i++ {
		var logicalPos coords.LogicalPosition
		var index int

		if isHorizontal {
			logicalPos = coords.LogicalPosition{X: i, Y: fixed}
			index = PositionToIndex(i, fixed)
		} else {
			logicalPos = coords.LogicalPosition{X: fixed, Y: i}
			index = PositionToIndex(fixed, i)
		}

		if index >= 0 && index < len(result.Tiles) {
			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = worldmapcore.FLOOR
			result.Tiles[index].Image = SelectRandomImage(images.FloorImages)
			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}

// CarveHorizontalTunnel creates horizontal corridor
func CarveHorizontalTunnel(result *worldmapcore.GenerationResult, x1, x2, y, width int, images worldmapcore.TileImageSet) {
	CarveTunnel(result, x1, x2, y, width, true, images)
}

// CarveVerticalTunnel creates vertical corridor
func CarveVerticalTunnel(result *worldmapcore.GenerationResult, y1, y2, x, width int, images worldmapcore.TileImageSet) {
	CarveTunnel(result, y1, y2, x, width, false, images)
}

// ========================================
// SHARED CONNECTIVITY HELPERS
// ========================================

// FloodFillRegion finds all connected walkable tiles starting from (startX, startY).
// terrainMap: true = walkable, false = obstacle.
// Returns slice of flat indices belonging to this connected region.
func FloodFillRegion(terrainMap []bool, visited []bool, startX, startY, width, height int) []int {
	var region []int
	queue := [][2]int{{startX, startY}}
	visited[PositionToIndex(startX, startY)] = true

	for len(queue) > 0 {
		x, y := queue[0][0], queue[0][1]
		queue = queue[1:]

		region = append(region, PositionToIndex(x, y))

		neighbors := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
		for _, dir := range neighbors {
			nx, ny := x+dir[0], y+dir[1]
			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				nidx := PositionToIndex(nx, ny)
				if !visited[nidx] && terrainMap[nidx] {
					visited[nidx] = true
					queue = append(queue, [2]int{nx, ny})
				}
			}
		}
	}

	return region
}

// CarveCorridorBetween creates an L-shaped corridor between two flat indices on the terrain map.
// Sets traversed cells to walkable (true).
func CarveCorridorBetween(terrainMap []bool, width, height, fromIdx, toIdx int) {
	fromX, fromY := fromIdx%width, fromIdx/width
	toX, toY := toIdx%width, toIdx/width

	// Horizontal first
	if fromX < toX {
		for x := fromX; x <= toX; x++ {
			if x >= 0 && x < width && fromY >= 0 && fromY < height {
				terrainMap[PositionToIndex(x, fromY)] = true
			}
		}
	} else {
		for x := fromX; x >= toX; x-- {
			if x >= 0 && x < width && fromY >= 0 && fromY < height {
				terrainMap[PositionToIndex(x, fromY)] = true
			}
		}
	}

	// Then vertical
	if fromY < toY {
		for y := fromY; y <= toY; y++ {
			if toX >= 0 && toX < width && y >= 0 && y < height {
				terrainMap[PositionToIndex(toX, y)] = true
			}
		}
	} else {
		for y := fromY; y >= toY; y-- {
			if toX >= 0 && toX < width && y >= 0 && y < height {
				terrainMap[PositionToIndex(toX, y)] = true
			}
		}
	}
}

// EnsureTerrainConnectivity finds all disconnected walkable regions and connects
// them to the largest region via L-shaped corridors.
func EnsureTerrainConnectivity(terrainMap []bool, width, height int) {
	visited := make([]bool, len(terrainMap))
	var largestRegion []int
	maxSize := 0

	// Find all regions and track the largest
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := PositionToIndex(x, y)
			if !visited[idx] && terrainMap[idx] {
				region := FloodFillRegion(terrainMap, visited, x, y, width, height)
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
				terrainMap[PositionToIndex(x, y)] = true
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
			idx := PositionToIndex(x, y)
			if !visited[idx] && terrainMap[idx] {
				region := FloodFillRegion(terrainMap, visited, x, y, width, height)
				if len(region) > 0 {
					CarveCorridorBetween(terrainMap, width, height, largestRegion[0], region[0])
				}
			}
		}
	}
}

// ========================================
// SHARED TERRAIN UTILITIES
// ========================================

// ConvertTerrainMapToTiles converts a boolean terrain map to actual Tile objects.
// terrainMap true = walkable floor, false = wall. Assigns biome images and populates ValidPositions.
func ConvertTerrainMapToTiles(result *worldmapcore.GenerationResult, terrainMap []bool, width, height int, images worldmapcore.TileImageSet, biome worldmapcore.Biome) {
	wallImages, floorImages := GetBiomeImages(images, biome)

	result.BiomeMap = make([]worldmapcore.Biome, width*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := PositionToIndex(x, y)
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			pixelX := x * graphics.ScreenInfo.TileSize
			pixelY := y * graphics.ScreenInfo.TileSize

			result.BiomeMap[idx] = biome

			if terrainMap[idx] {
				floorImage := SelectRandomImage(floorImages)
				tile := worldmapcore.NewTile(pixelX, pixelY, logicalPos, false, floorImage, worldmapcore.FLOOR, false)
				tile.Biome = biome
				result.Tiles[idx] = &tile
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			} else {
				wallImage := SelectRandomImage(wallImages)
				tile := worldmapcore.NewTile(pixelX, pixelY, logicalPos, true, wallImage, worldmapcore.WALL, false)
				tile.Biome = biome
				result.Tiles[idx] = &tile
			}
		}
	}
}

// ScoreTerrainOpenness counts walkable tiles within a radius of (cx, cy) on a terrain map.
func ScoreTerrainOpenness(terrainMap []bool, cx, cy, radius, width, height int) int {
	score := 0
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			nx, ny := cx+dx, cy+dy
			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				if terrainMap[PositionToIndex(nx, ny)] {
					score++
				}
			}
		}
	}
	return score
}

// FindBestOpenPosition finds the walkable position with the highest openness score
// within the given bounding box. Returns {-1, -1} if no walkable tile is found.
func FindBestOpenPosition(terrainMap []bool, width, height, xMin, xMax, yMin, yMax, scanRadius int) coords.LogicalPosition {
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
			if !terrainMap[PositionToIndex(x, y)] {
				continue
			}

			score := ScoreTerrainOpenness(terrainMap, x, y, scanRadius, width, height)
			if score > bestScore {
				bestScore = score
				bestPos = coords.LogicalPosition{X: x, Y: y}
			}
		}
	}

	return bestPos
}

// IsTooCloseToAny checks if (px, py) is within minSpacing of any position in placed
// using Chebyshev distance (both dx AND dy must be < minSpacing to be "too close").
func IsTooCloseToAny(px, py int, placed [][2]int, minSpacing int) bool {
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

// TryPlace2x2PillarOnTerrain checks that a 2x2 area at (px, py) is fully walkable,
// then sets it to wall. Returns true if placed, false if blocked or out of bounds.
func TryPlace2x2PillarOnTerrain(terrainMap []bool, px, py, width, height int) bool {
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			nx, ny := px+dx, py+dy
			if nx < 0 || nx >= width || ny < 0 || ny >= height || !terrainMap[PositionToIndex(nx, ny)] {
				return false
			}
		}
	}
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			terrainMap[PositionToIndex(px+dx, py+dy)] = false
		}
	}
	return true
}
