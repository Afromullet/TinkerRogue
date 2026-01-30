// Package worldmap handles game world generation, map management, and spatial operations.
package worldmap

import (
	"game_main/common"
	"game_main/world/coords"
	"game_main/visual/graphics"

	"github.com/hajimehoshi/ebiten/v2"
)

// TacticalBiomeGenerator creates tactical battle maps with distinct biome characteristics
// Uses cellular automata for natural terrain generation combined with tactical features
type TacticalBiomeGenerator struct {
	config GeneratorConfig
}

// BiomeTacticalProfile defines tactical characteristics for each biome
type BiomeTacticalProfile struct {
	// Terrain density (0.0 = all open, 1.0 = all obstacles)
	obstacleDensity float64
	// Tactical features
	hasCover         bool // Trees, rocks, ruins for protection
	hasElevation     bool // Hills, high ground
	hasChokePoints   bool // Natural bottlenecks
	hasOpenSpace     bool // Wide maneuvering areas
	hasRoughTerrain  bool // Difficult terrain that slows movement
	// Visual variety
	floorVariants int // Number of different floor tile variants
	wallVariants  int // Number of different obstacle variants
}

// NewTacticalBiomeGenerator creates a new tactical biome generator
func NewTacticalBiomeGenerator(config GeneratorConfig) *TacticalBiomeGenerator {
	return &TacticalBiomeGenerator{config: config}
}

func (g *TacticalBiomeGenerator) Name() string {
	return "tactical_biome"
}

func (g *TacticalBiomeGenerator) Description() string {
	return "Tactical battle maps with distinct biome characteristics using cellular automata"
}

func (g *TacticalBiomeGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:          createEmptyTiles(width, height, images),
		Rooms:          make([]Rect, 0),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	// Select biome based on seed
	biome := g.selectBiome()
	profile := g.getBiomeProfile(biome)

	// Generate base terrain using cellular automata
	terrainMap := g.generateCellularTerrain(width, height, profile)

	// Add biome-specific tactical features
	g.addTacticalFeatures(terrainMap, width, height, profile, biome)

	// Ensure connectivity
	g.ensureConnectivity(terrainMap, width, height)

	// Convert terrain map to tiles
	g.convertToTiles(&result, terrainMap, width, height, images, biome)

	// Clear spawn area
	centerX := width / 2
	centerY := height / 2
	g.clearSpawnArea(&result, centerX, centerY, 5, width, height, images.FloorImages)

	return result
}

// selectBiome chooses a biome based on the seed
func (g *TacticalBiomeGenerator) selectBiome() Biome {
	biomes := []Biome{BiomeGrassland, BiomeForest, BiomeDesert, BiomeMountain, BiomeSwamp}
	idx := common.GetDiceRoll(len(biomes)) - 1 // GetDiceRoll returns 1 to n
	return biomes[idx]
}

// getBiomeProfile returns tactical characteristics for a biome
func (g *TacticalBiomeGenerator) getBiomeProfile(biome Biome) BiomeTacticalProfile {
	switch biome {
	case BiomeGrassland:
		return BiomeTacticalProfile{
			obstacleDensity:  0.20, // Mostly open
			hasCover:         true, // Scattered trees, rocks
			hasElevation:     true, // Gentle hills
			hasChokePoints:   false,
			hasOpenSpace:     true, // Wide maneuvering
			hasRoughTerrain:  false,
			floorVariants:    3,
			wallVariants:     2,
		}

	case BiomeForest:
		return BiomeTacticalProfile{
			obstacleDensity:  0.35, // Dense trees
			hasCover:         true, // Many trees for cover
			hasElevation:     false,
			hasChokePoints:   true, // Tree clusters create paths
			hasOpenSpace:     false,
			hasRoughTerrain:  true, // Undergrowth
			floorVariants:    2,
			wallVariants:     3,
		}

	case BiomeDesert:
		return BiomeTacticalProfile{
			obstacleDensity:  0.15, // Very open
			hasCover:         false, // Minimal cover
			hasElevation:     true, // Sand dunes
			hasChokePoints:   false,
			hasOpenSpace:     true, // Wide open spaces
			hasRoughTerrain:  true, // Sand slows movement
			floorVariants:    2,
			wallVariants:     2,
		}

	case BiomeMountain:
		return BiomeTacticalProfile{
			obstacleDensity:  0.45, // Rocky, many obstacles
			hasCover:         true, // Boulders, cliffs
			hasElevation:     true, // High ground advantage
			hasChokePoints:   true, // Mountain passes
			hasOpenSpace:     false,
			hasRoughTerrain:  true, // Rocky terrain
			floorVariants:    2,
			wallVariants:     4,
		}

	case BiomeSwamp:
		return BiomeTacticalProfile{
			obstacleDensity:  0.30, // Water + vegetation
			hasCover:         true, // Reeds, trees
			hasElevation:     false,
			hasChokePoints:   true, // Islands of dry land
			hasOpenSpace:     false,
			hasRoughTerrain:  true, // Muddy, wet
			floorVariants:    3,
			wallVariants:     3,
		}

	default:
		return g.getBiomeProfile(BiomeGrassland)
	}
}

// generateCellularTerrain uses cellular automata to create natural terrain
func (g *TacticalBiomeGenerator) generateCellularTerrain(width, height int, profile BiomeTacticalProfile) []bool {
	terrainMap := make([]bool, width*height)

	// Initialize with random obstacles based on density
	for idx := 0; idx < len(terrainMap); idx++ {
		// GetDiceRoll(100) returns 1-100, convert to 0.01-1.00
		randomPercent := float64(common.GetDiceRoll(100)) / 100.0
		terrainMap[idx] = (randomPercent > profile.obstacleDensity)
	}

	// Apply cellular automata smoothing (4-5 iterations)
	for iteration := 0; iteration < 5; iteration++ {
		terrainMap = g.cellularAutomataStep(terrainMap, width, height)
	}

	return terrainMap
}

// cellularAutomataStep applies one iteration of cellular automata
// Rule: if cell has 5+ wall neighbors (of 8), become wall; else become floor
func (g *TacticalBiomeGenerator) cellularAutomataStep(terrainMap []bool, width, height int) []bool {
	newMap := make([]bool, len(terrainMap))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x

			// Count wall neighbors (including diagonals)
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

					nidx := ny*width + nx
					if !terrainMap[nidx] {
						wallCount++
					}
				}
			}

			// Apply rule: 5+ wall neighbors = wall, else floor
			newMap[idx] = (wallCount < 5)
		}
	}

	return newMap
}

// addTacticalFeatures adds biome-specific tactical elements
func (g *TacticalBiomeGenerator) addTacticalFeatures(terrainMap []bool, width, height int, profile BiomeTacticalProfile, biome Biome) {
	// Add cover positions (small obstacle clusters)
	if profile.hasCover {
		g.addCoverPositions(terrainMap, width, height)
	}

	// Add choke points (narrow passages)
	if profile.hasChokePoints {
		g.addChokePoints(terrainMap, width, height)
	}

	// Ensure minimum open space
	if profile.hasOpenSpace {
		g.ensureOpenAreas(terrainMap, width, height)
	}
}

// addCoverPositions creates small clusters of obstacles for cover
func (g *TacticalBiomeGenerator) addCoverPositions(terrainMap []bool, width, height int) {
	numCoverSpots := (width * height) / 100 // ~1% of map

	for i := 0; i < numCoverSpots; i++ {
		// Pick random center
		cx := common.GetDiceRoll(width) - 1
		cy := common.GetDiceRoll(height) - 1

		// Create small cluster (2x2 to 3x3)
		size := 2 + common.GetDiceRoll(2) - 1
		for dy := 0; dy < size; dy++ {
			for dx := 0; dx < size; dx++ {
				x := cx + dx
				y := cy + dy

				if x >= 0 && x < width && y >= 0 && y < height {
					idx := y*width + x
					// 60% chance to make this spot an obstacle
					if common.GetDiceRoll(100) <= 60 {
						terrainMap[idx] = false
					}
				}
			}
		}
	}
}

// addChokePoints creates narrow passages between larger obstacle regions
func (g *TacticalBiomeGenerator) addChokePoints(terrainMap []bool, width, height int) {
	numChokes := 3 + common.GetDiceRoll(3) - 1 // 3-5 choke points

	for i := 0; i < numChokes; i++ {
		// Pick random horizontal or vertical line
		if common.GetDiceRoll(2) == 1 {
			// Vertical choke
			x := common.GetDiceRoll(width) - 1
			for y := 0; y < height; y++ {
				idx := y*width + x
				// Make this column mostly obstacles with a few openings
				if common.GetDiceRoll(100) <= 70 {
					terrainMap[idx] = false
				}
			}
		} else {
			// Horizontal choke
			y := common.GetDiceRoll(height) - 1
			for x := 0; x < width; x++ {
				idx := y*width + x
				if common.GetDiceRoll(100) <= 70 {
					terrainMap[idx] = false
				}
			}
		}
	}
}

// ensureOpenAreas creates large open regions for maneuvering
func (g *TacticalBiomeGenerator) ensureOpenAreas(terrainMap []bool, width, height int) {
	numOpenAreas := 2 + common.GetDiceRoll(2) - 1 // 2-3 open areas

	for i := 0; i < numOpenAreas; i++ {
		// Pick random center
		cx := common.GetDiceRoll(width) - 1
		cy := common.GetDiceRoll(height) - 1

		// Create large open circle
		radius := 5 + common.GetDiceRoll(3) - 1
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				dx := x - cx
				dy := y - cy
				distSq := dx*dx + dy*dy

				if distSq <= radius*radius {
					idx := y*width + x
					terrainMap[idx] = true
				}
			}
		}
	}
}

// ensureConnectivity uses flood-fill to connect all walkable regions
func (g *TacticalBiomeGenerator) ensureConnectivity(terrainMap []bool, width, height int) {
	// Find largest connected region
	visited := make([]bool, len(terrainMap))
	var largestRegion []int
	maxSize := 0

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

	// If no walkable region, make center 50% walkable
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

	// Connect all other regions to largest
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if !visited[idx] && terrainMap[idx] {
				region := g.floodFill(terrainMap, visited, x, y, width, height)
				if len(region) > 0 {
					// Carve corridor from this region to largest
					g.carveCorridorToRegion(terrainMap, width, height, largestRegion[0], region[0])
				}
			}
		}
	}
}

// floodFill finds all connected walkable tiles
func (g *TacticalBiomeGenerator) floodFill(terrainMap, visited []bool, startX, startY, width, height int) []int {
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

// carveCorridorToRegion creates L-shaped corridor between regions
func (g *TacticalBiomeGenerator) carveCorridorToRegion(terrainMap []bool, width, height, fromIdx, toIdx int) {
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

// convertToTiles converts terrain map to tile array
func (g *TacticalBiomeGenerator) convertToTiles(result *GenerationResult, terrainMap []bool, width, height int, images TileImageSet, biome Biome) {
	// Get biome-specific images
	wallImages, floorImages := getBiomeImages(images, biome)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := positionToIndex(x, y, width)
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			pixelX := x * graphics.ScreenInfo.TileSize
			pixelY := y * graphics.ScreenInfo.TileSize

			if terrainMap[idx] {
				// Walkable tile
				floorImage := selectRandomImage(floorImages)
				tile := NewTile(pixelX, pixelY, logicalPos, false, floorImage, FLOOR, false)
				result.ValidPositions = append(result.ValidPositions, logicalPos)
				result.Tiles[idx] = &tile
			} else {
				// Obstacle tile
				wallImage := selectRandomImage(wallImages)
				tile := NewTile(pixelX, pixelY, logicalPos, true, wallImage, WALL, false)
				result.Tiles[idx] = &tile
			}
		}
	}
}

// clearSpawnArea ensures spawn area is walkable
func (g *TacticalBiomeGenerator) clearSpawnArea(result *GenerationResult, centerX, centerY, radius, width, height int, floorImages []*ebiten.Image) {
	for x := centerX - radius; x <= centerX+radius; x++ {
		for y := centerY - radius; y <= centerY+radius; y++ {
			// Validate coordinates are within bounds BEFORE using them
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}

			logicalPos := coords.LogicalPosition{X: x, Y: y}
			idx := positionToIndex(x, y, width)

			if idx >= 0 && idx < len(result.Tiles) {
				pixelX := x * graphics.ScreenInfo.TileSize
				pixelY := y * graphics.ScreenInfo.TileSize

				floorImage := selectRandomImage(floorImages)
				tile := NewTile(pixelX, pixelY, logicalPos, false, floorImage, FLOOR, false)
				result.Tiles[idx] = &tile

				// Also add to valid positions
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			}
		}
	}
}

// Register this generator on package initialization
func init() {
	RegisterGenerator(NewTacticalBiomeGenerator(DefaultConfig()))
}
