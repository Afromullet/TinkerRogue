package worldmap

import (
	"game_main/common"
	"game_main/coords"
	"game_main/visual/graphics"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// CaveGenerator creates tactical cave maps using cellular automata
// Produces organic underground layouts with chambers, tunnels, and cave features
type CaveGenerator struct {
	config CaveConfig
}

// CaveConfig holds parameters for cave generation
type CaveConfig struct {
	// Cellular automata parameters
	InitialWallDensity float64 // Initial wall fill ratio (0.45-0.55)
	CAIterations       int     // Number of CA smoothing passes (5-7)
	BirthLimit         int     // Become wall if >= this many wall neighbors
	DeathLimit         int     // Stay floor if > this many wall neighbors

	// Cave features
	PillarDensity float64 // Chance of pillar clusters per tile (0.02-0.05)
	PoolChance    float64 // Chance to add water pools (0.0-0.3)
	RubbleDensity float64 // Scattered single-tile obstacles (0.01-0.03)

	// Erosion
	ErosionPasses int // Remove isolated walls (0-2)

	// Connectivity
	TunnelWidth    int  // Width of connecting tunnels (1-3)
	OrganicTunnels bool // Use curved paths vs straight corridors

	Seed int64
}

// DefaultCaveConfig returns sensible defaults for cave generation
func DefaultCaveConfig() CaveConfig {
	return CaveConfig{
		InitialWallDensity: 0.48,
		CAIterations:       6,
		BirthLimit:         5,
		DeathLimit:         4,

		PillarDensity: 0.03,
		PoolChance:    0.15,
		RubbleDensity: 0.02,

		ErosionPasses: 1,

		TunnelWidth:    2,
		OrganicTunnels: true,

		Seed: 0,
	}
}

// NewCaveGenerator creates a new cave generator
func NewCaveGenerator(config CaveConfig) *CaveGenerator {
	return &CaveGenerator{config: config}
}

func (g *CaveGenerator) Name() string {
	return "cave_tactical"
}

func (g *CaveGenerator) Description() string {
	return "Underground cave maps with organic chambers and tunnels for tactical combat"
}

func (g *CaveGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:          createEmptyTiles(width, height, images),
		Rooms:          make([]Rect, 0),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	// Initialize seed
	seed := g.config.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rand.Seed(seed)

	// Step 1: Initialize terrain with random walls
	terrainMap := g.initializeTerrain(width, height)

	// Step 2: Run cellular automata iterations
	for i := 0; i < g.config.CAIterations; i++ {
		terrainMap = g.cellularAutomataStep(terrainMap, width, height)
	}

	// Step 3: Apply erosion passes
	for i := 0; i < g.config.ErosionPasses; i++ {
		terrainMap = g.erode(terrainMap, width, height)
	}

	// Step 4: Ensure connectivity
	g.ensureConnectivity(terrainMap, width, height)

	// Step 5: Add cave features
	g.addPillars(terrainMap, width, height)
	g.addRubble(terrainMap, width, height)

	// Step 6: Create pool map for water features
	poolMap := g.createPoolMap(terrainMap, width, height)

	// Step 7: Clear spawn area
	centerX := width / 2
	centerY := height / 2
	g.clearArea(terrainMap, centerX, centerY, 5, width, height)

	// Step 8: Convert to tiles
	g.convertToTiles(&result, terrainMap, poolMap, width, height, images)

	return result
}

// initializeTerrain creates initial random wall distribution
func (g *CaveGenerator) initializeTerrain(width, height int) []bool {
	terrainMap := make([]bool, width*height)

	for idx := range terrainMap {
		// true = floor, false = wall
		randomVal := rand.Float64()
		terrainMap[idx] = randomVal > g.config.InitialWallDensity
	}

	// Ensure borders are walls for cleaner edges
	for x := 0; x < width; x++ {
		terrainMap[x] = false                    // Top
		terrainMap[(height-1)*width+x] = false   // Bottom
	}
	for y := 0; y < height; y++ {
		terrainMap[y*width] = false              // Left
		terrainMap[y*width+(width-1)] = false    // Right
	}

	return terrainMap
}

// cellularAutomataStep applies one iteration of cave-optimized CA
// Rule: become wall if wallNeighbors >= birthLimit OR wallNeighbors <= 1
func (g *CaveGenerator) cellularAutomataStep(terrainMap []bool, width, height int) []bool {
	newMap := make([]bool, len(terrainMap))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			wallCount := g.countWallNeighbors(terrainMap, x, y, width, height)

			// Cave CA rule: walls form with many neighbors OR isolation
			if wallCount >= g.config.BirthLimit {
				newMap[idx] = false // Become wall
			} else if wallCount <= 1 {
				newMap[idx] = false // Isolated cells become walls (prevents noise)
			} else if wallCount > g.config.DeathLimit {
				newMap[idx] = true // Stay/become floor
			} else {
				newMap[idx] = terrainMap[idx] // Keep current state
			}
		}
	}

	return newMap
}

// countWallNeighbors counts wall neighbors in 8 directions
func (g *CaveGenerator) countWallNeighbors(terrainMap []bool, x, y, width, height int) int {
	wallCount := 0

	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}

			nx, ny := x+dx, y+dy

			// Out of bounds counts as wall
			if nx < 0 || nx >= width || ny < 0 || ny >= height {
				wallCount++
				continue
			}

			if !terrainMap[ny*width+nx] {
				wallCount++
			}
		}
	}

	return wallCount
}

// erode removes isolated wall tiles for smoother edges
func (g *CaveGenerator) erode(terrainMap []bool, width, height int) []bool {
	newMap := make([]bool, len(terrainMap))
	copy(newMap, terrainMap)

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			idx := y*width + x

			// Only process walls
			if terrainMap[idx] {
				continue
			}

			wallCount := g.countWallNeighbors(terrainMap, x, y, width, height)

			// Remove isolated walls (few wall neighbors)
			if wallCount <= 2 {
				newMap[idx] = true // Convert to floor
			}
		}
	}

	return newMap
}

// ensureConnectivity connects all cave regions
func (g *CaveGenerator) ensureConnectivity(terrainMap []bool, width, height int) {
	visited := make([]bool, len(terrainMap))
	var largestRegion []int
	maxSize := 0

	// Find all regions
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if !visited[idx] && terrainMap[idx] {
				region := g.floodFill(terrainMap, visited, x, y, width, height)
				if len(region) > maxSize {
					largestRegion = region
					maxSize = len(region)
				}
			}
		}
	}

	// If no walkable area, carve out center
	if maxSize == 0 {
		for y := height / 4; y < (height * 3 / 4); y++ {
			for x := width / 4; x < (width * 3 / 4); x++ {
				terrainMap[y*width+x] = true
			}
		}
		return
	}

	// Mark largest region
	visited = make([]bool, len(terrainMap))
	for _, idx := range largestRegion {
		visited[idx] = true
	}

	// Connect other regions to largest
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if !visited[idx] && terrainMap[idx] {
				region := g.floodFill(terrainMap, visited, x, y, width, height)
				if len(region) > 0 {
					// Find closest points between regions
					fromIdx := largestRegion[rand.Intn(len(largestRegion))]
					toIdx := region[rand.Intn(len(region))]

					if g.config.OrganicTunnels {
						g.carveOrganicTunnel(terrainMap, width, height, fromIdx, toIdx)
					} else {
						g.carveStraightTunnel(terrainMap, width, fromIdx, toIdx)
					}
				}
			}
		}
	}
}

// floodFill finds connected floor tiles
func (g *CaveGenerator) floodFill(terrainMap, visited []bool, startX, startY, width, height int) []int {
	var region []int
	queue := [][2]int{{startX, startY}}
	visited[startY*width+startX] = true

	for len(queue) > 0 {
		x, y := queue[0][0], queue[0][1]
		queue = queue[1:]
		region = append(region, y*width+x)

		// Check 4 neighbors
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

// carveOrganicTunnel creates a winding tunnel between two points
func (g *CaveGenerator) carveOrganicTunnel(terrainMap []bool, width, height, fromIdx, toIdx int) {
	fromX, fromY := fromIdx%width, fromIdx/width
	toX, toY := toIdx%width, toIdx/width

	x, y := fromX, fromY
	tunnelWidth := g.config.TunnelWidth

	// Maximum iterations to prevent infinite loops
	maxIterations := width*height
	iterations := 0

	for (x != toX || y != toY) && iterations < maxIterations {
		iterations++

		// Carve current position with width
		for dy := -tunnelWidth / 2; dy <= tunnelWidth/2; dy++ {
			for dx := -tunnelWidth / 2; dx <= tunnelWidth/2; dx++ {
				nx, ny := x+dx, y+dy
				if nx >= 0 && nx < width && ny >= 0 && ny < height {
					terrainMap[ny*width+nx] = true
				}
			}
		}

		// Calculate direction toward target
		dx := toX - x
		dy := toY - y

		// If we've reached the target, break
		if dx == 0 && dy == 0 {
			break
		}

		// Normalize direction (guaranteed to move toward target)
		moveX, moveY := 0, 0
		if dx != 0 {
			moveX = dx / abs(dx)
		}
		if dy != 0 {
			moveY = dy / abs(dy)
		}

		// Add slight randomness (20% chance to ADD perpendicular movement)
		// Don't replace movement, just add deviation
		if rand.Float64() < 0.2 {
			if abs(dx) > abs(dy) {
				// Moving mostly horizontal, add vertical deviation
				if rand.Float64() < 0.5 && moveY == 0 {
					moveY = rand.Intn(3) - 1 // -1, 0, or 1
				}
			} else {
				// Moving mostly vertical, add horizontal deviation
				if rand.Float64() < 0.5 && moveX == 0 {
					moveX = rand.Intn(3) - 1
				}
			}
		}

		// Move in primary direction (guaranteed progress)
		if abs(dx) > abs(dy) {
			// Horizontal distance is greater - prioritize X movement
			x += moveX
			if moveY != 0 && rand.Float64() < 0.3 {
				y += moveY
			}
		} else {
			// Vertical distance is greater - prioritize Y movement
			y += moveY
			if moveX != 0 && rand.Float64() < 0.3 {
				x += moveX
			}
		}

		// Clamp to valid bounds (allow edges)
		x = max(0, min(x, width-1))
		y = max(0, min(y, height-1))
	}

	// If we hit max iterations, fall back to straight tunnel
	if iterations >= maxIterations {
		g.carveStraightTunnel(terrainMap, width, fromIdx, toIdx)
		return
	}

	// Carve final position
	for dy := -tunnelWidth / 2; dy <= tunnelWidth/2; dy++ {
		for dx := -tunnelWidth / 2; dx <= tunnelWidth/2; dx++ {
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				terrainMap[ny*width+nx] = true
			}
		}
	}
}

// carveStraightTunnel creates L-shaped corridor
func (g *CaveGenerator) carveStraightTunnel(terrainMap []bool, width, fromIdx, toIdx int) {
	fromX, fromY := fromIdx%width, fromIdx/width
	toX, toY := toIdx%width, toIdx/width

	// Horizontal
	if fromX < toX {
		for x := fromX; x <= toX; x++ {
			terrainMap[fromY*width+x] = true
		}
	} else {
		for x := fromX; x >= toX; x-- {
			terrainMap[fromY*width+x] = true
		}
	}

	// Vertical
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

// addPillars places rock pillars in open areas
func (g *CaveGenerator) addPillars(terrainMap []bool, width, height int) {
	numPillars := int(float64(width*height) * g.config.PillarDensity)

	for i := 0; i < numPillars; i++ {
		// Random position
		x := rand.Intn(width-4) + 2
		y := rand.Intn(height-4) + 2

		// Check if area is open (mostly floor)
		floorCount := 0
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if terrainMap[(y+dy)*width+(x+dx)] {
					floorCount++
				}
			}
		}

		// Only place pillar in open areas
		if floorCount >= 20 {
			// Create 2x2 or 3x3 pillar
			pillarSize := 2 + rand.Intn(2)
			for dy := 0; dy < pillarSize; dy++ {
				for dx := 0; dx < pillarSize; dx++ {
					px, py := x+dx, y+dy
					if px < width && py < height {
						terrainMap[py*width+px] = false
					}
				}
			}
		}
	}
}

// addRubble scatters single-tile obstacles
func (g *CaveGenerator) addRubble(terrainMap []bool, width, height int) {
	numRubble := int(float64(width*height) * g.config.RubbleDensity)

	for i := 0; i < numRubble; i++ {
		x := rand.Intn(width-2) + 1
		y := rand.Intn(height-2) + 1
		idx := y*width + x

		// Only place on floor tiles
		if terrainMap[idx] {
			// Check it won't block a corridor (has multiple floor neighbors)
			floorNeighbors := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					if terrainMap[(y+dy)*width+(x+dx)] {
						floorNeighbors++
					}
				}
			}

			// Only place if area is open enough
			if floorNeighbors >= 5 {
				terrainMap[idx] = false
			}
		}
	}
}

// createPoolMap identifies areas for water pools
func (g *CaveGenerator) createPoolMap(terrainMap []bool, width, height int) []bool {
	poolMap := make([]bool, width*height)

	if g.config.PoolChance <= 0 {
		return poolMap
	}

	// Find enclosed floor areas for potential pools
	for y := 2; y < height-2; y++ {
		for x := 2; x < width-2; x++ {
			idx := y*width + x

			if !terrainMap[idx] {
				continue // Skip walls
			}

			// Check if this could be a pool (surrounded by floors, random chance)
			if rand.Float64() > g.config.PoolChance {
				continue
			}

			// Check local area is mostly floor
			floorCount := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if terrainMap[(y+dy)*width+(x+dx)] {
						floorCount++
					}
				}
			}

			if floorCount >= 7 {
				poolMap[idx] = true
			}
		}
	}

	return poolMap
}

// clearArea clears a circular area
func (g *CaveGenerator) clearArea(terrainMap []bool, centerX, centerY, radius, width, height int) {
	for y := centerY - radius; y <= centerY+radius; y++ {
		for x := centerX - radius; x <= centerX+radius; x++ {
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}

			dx := x - centerX
			dy := y - centerY
			if dx*dx+dy*dy <= radius*radius {
				terrainMap[y*width+x] = true
			}
		}
	}
}

// convertToTiles creates actual tile objects
func (g *CaveGenerator) convertToTiles(result *GenerationResult, terrainMap, poolMap []bool, width, height int, images TileImageSet) {
	// Use mountain biome for rocky cave appearance
	caveBiome := BiomeMountain
	poolBiome := BiomeSwamp

	// Get biome-specific images
	caveWallImages := images.WallImages
	caveFloorImages := images.FloorImages

	if biomeTileSet := images.BiomeImages[caveBiome]; biomeTileSet != nil {
		if len(biomeTileSet.WallImages) > 0 {
			caveWallImages = biomeTileSet.WallImages
		}
		if len(biomeTileSet.FloorImages) > 0 {
			caveFloorImages = biomeTileSet.FloorImages
		}
	}

	// Pool images from swamp biome
	poolFloorImages := caveFloorImages
	if biomeTileSet := images.BiomeImages[poolBiome]; biomeTileSet != nil {
		if len(biomeTileSet.FloorImages) > 0 {
			poolFloorImages = biomeTileSet.FloorImages
		}
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			terrainIdx := y*width + x
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			tileIdx := coords.CoordManager.LogicalToIndex(logicalPos)
			pixelX := x * graphics.ScreenInfo.TileSize
			pixelY := y * graphics.ScreenInfo.TileSize

			if terrainMap[terrainIdx] {
				// Floor tile
				var floorImage *ebiten.Image
				floorImages := caveFloorImages

				// Use pool images for pool areas
				if poolMap[terrainIdx] {
					floorImages = poolFloorImages
				}

				if len(floorImages) > 0 {
					floorImage = floorImages[common.GetRandomBetween(0, len(floorImages)-1)]
				}

				tile := NewTile(pixelX, pixelY, logicalPos, false, floorImage, FLOOR, false)
				result.Tiles[tileIdx] = &tile
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			} else {
				// Wall tile
				var wallImage *ebiten.Image
				if len(caveWallImages) > 0 {
					wallImage = caveWallImages[common.GetRandomBetween(0, len(caveWallImages)-1)]
				}

				tile := NewTile(pixelX, pixelY, logicalPos, true, wallImage, WALL, false)
				result.Tiles[tileIdx] = &tile
			}
		}
	}
}

// abs returns absolute value of int
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Distance calculates Euclidean distance between two points
func distance(x1, y1, x2, y2 int) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	return math.Sqrt(dx*dx + dy*dy)
}

// Register on package initialization
func init() {
	RegisterGenerator(NewCaveGenerator(DefaultCaveConfig()))
}
