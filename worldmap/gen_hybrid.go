package worldmap

import (
	"game_main/common"
	"game_main/coords"
	"game_main/graphics"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// HybridGenerator combines Multi-layer Perlin + Domain Warping + Voronoi
// for tactical combat maps with natural biome variety and intra-biome terrain variation
type HybridGenerator struct {
	config HybridConfig
}

// HybridConfig holds parameters for hybrid generation
type HybridConfig struct {
	// Multi-layer Perlin noise parameters
	BaseScale   float64 // Large features scale (0.05-0.1)
	DetailScale float64 // Small features scale (0.15-0.25)
	Octaves     int     // Number of noise layers (3-5)
	Persistence float64 // Octave amplitude falloff (0.5)

	// Domain warping parameters
	WarpAmount float64 // Distortion strength (2.0-5.0)
	WarpScale  float64 // Warp noise frequency (0.08-0.15)

	// Voronoi parameters (for regional density variation)
	VoronoiRegions      int     // Number of density zones (8-15)
	VoronoiRelaxation   int     // Lloyd relaxation iterations (2-4)
	VoronoiDensityBlend float64 // How much Voronoi affects density (0.0-1.0)

	// Biome determination (noise-based blending)
	MoistureScale  float64 // Moisture noise scale
	ElevationScale float64 // Elevation noise scale

	// Terrain generation
	BaseObstacleDensity float64 // Starting obstacle density before biome modifiers
	CellularIterations  int     // Cellular automata smoothing passes

	// Seed (0 = time-based)
	Seed int64
}

// DefaultHybridConfig returns sensible defaults for tactical maps
func DefaultHybridConfig() HybridConfig {
	return HybridConfig{
		// Multi-layer Perlin
		BaseScale:   0.08,
		DetailScale: 0.20,
		Octaves:     4,
		Persistence: 0.5,

		// Domain warping
		WarpAmount: 3.5,
		WarpScale:  0.10,

		// Voronoi
		VoronoiRegions:      12,
		VoronoiRelaxation:   3,
		VoronoiDensityBlend: 0.4,

		// Biome
		MoistureScale:  0.06,
		ElevationScale: 0.07,

		// Terrain
		BaseObstacleDensity: 0.30,
		CellularIterations:  4,

		Seed: 0,
	}
}

// NewHybridGenerator creates a new hybrid generator
func NewHybridGenerator(config HybridConfig) *HybridGenerator {
	return &HybridGenerator{config: config}
}

func (g *HybridGenerator) Name() string {
	return "hybrid_tactical"
}

func (g *HybridGenerator) Description() string {
	return "Hybrid tactical maps using Multi-layer Perlin + Domain Warping + Voronoi for natural biome variety"
}

func (g *HybridGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
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
	noiseGen := NewNoiseGenerator(seed)

	// Step 1: Generate noise layers
	baseNoise := noiseGen.GeneratePerlinNoise(width, height, g.config.BaseScale, g.config.Octaves, g.config.Persistence)
	detailNoise := noiseGen.GeneratePerlinNoise(width, height, g.config.DetailScale, 2, 0.6)
	moistureNoise := noiseGen.GeneratePerlinNoise(width, height, g.config.MoistureScale, 3, 0.5)
	elevationNoise := noiseGen.GeneratePerlinNoise(width, height, g.config.ElevationScale, 3, 0.5)

	// Step 2: Apply domain warping to base noise for organic shapes
	warpedBase := noiseGen.ApplyDomainWarping(baseNoise, g.config.WarpAmount, g.config.WarpScale)

	// Step 3: Generate Voronoi for regional density variation
	voronoiDist, _ := noiseGen.GenerateVoronoiWithJitter(width, height, g.config.VoronoiRegions, g.config.VoronoiRelaxation)

	// Step 4: Combine layers and determine biomes
	combinedNoise := BlendNoiseMaps(warpedBase, detailNoise, 0.6, 0.4)
	biomeMap := g.determineBiomeMap(width, height, elevationNoise, moistureNoise)

	// Step 5: Calculate local density with intra-biome variation
	densityMap := g.calculateDensityMap(width, height, biomeMap, voronoiDist, detailNoise)

	// Step 6: Generate terrain from density map
	terrainMap := g.generateTerrain(width, height, densityMap, combinedNoise)

	// Step 7: Apply cellular automata smoothing
	for i := 0; i < g.config.CellularIterations; i++ {
		terrainMap = g.cellularAutomataStep(terrainMap, width, height)
	}

	// Step 8: Ensure connectivity
	g.ensureConnectivity(terrainMap, width, height)

	// Step 9: Convert to tiles with biome-specific visuals
	g.convertToTiles(&result, terrainMap, biomeMap, width, height, images)

	// Step 10: Clear spawn area in center
	centerX := width / 2
	centerY := height / 2
	g.clearSpawnArea(&result, centerX, centerY, 5, width, height, images)

	return result
}

// determineBiomeMap creates a biome for each tile based on elevation and moisture
func (g *HybridGenerator) determineBiomeMap(width, height int, elevation, moisture [][]float64) [][]Biome {
	biomeMap := make([][]Biome, height)
	for y := 0; y < height; y++ {
		biomeMap[y] = make([]Biome, width)
		for x := 0; x < width; x++ {
			biomeMap[y][x] = g.determineBiome(elevation[y][x], moisture[y][x])
		}
	}
	return biomeMap
}

// determineBiome classifies terrain based on elevation and moisture
// Creates gradual transitions between biomes
func (g *HybridGenerator) determineBiome(elevation, moisture float64) Biome {
	// Thresholds for biome determination
	const (
		waterThresh    = 0.25
		lowlandThresh  = 0.45
		highlandThresh = 0.65
		mountainThresh = 0.80

		dryThresh = 0.35
		wetThresh = 0.60
	)

	// Very low elevation = swamp/water
	if elevation < waterThresh {
		return BiomeSwamp
	}

	// Very high elevation = mountain
	if elevation > mountainThresh {
		return BiomeMountain
	}

	// Mid elevations depend on moisture
	if moisture > wetThresh {
		return BiomeForest
	}

	if moisture < dryThresh {
		if elevation > highlandThresh {
			return BiomeMountain
		}
		return BiomeDesert
	}

	// Default to grassland for moderate elevation and moisture
	return BiomeGrassland
}

// calculateDensityMap creates per-tile obstacle density with intra-biome variation
func (g *HybridGenerator) calculateDensityMap(width, height int, biomeMap [][]Biome, voronoiDist, detailNoise [][]float64) [][]float64 {
	densityMap := make([][]float64, height)
	for y := 0; y < height; y++ {
		densityMap[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			// Get base density from biome
			biome := biomeMap[y][x]
			baseDensity := g.getBiomeDensity(biome)

			// Voronoi influence: high distance = lower density (clearings)
			// Low distance = higher density (clusters near region centers)
			voronoiMod := 1.0 - (voronoiDist[y][x] * g.config.VoronoiDensityBlend)

			// Detail noise adds local micro-variations
			// Centered around 1.0 with +/- 0.3 variation
			detailMod := 0.7 + detailNoise[y][x]*0.6

			// Combine modifiers
			finalDensity := baseDensity * voronoiMod * detailMod

			// Clamp to valid range
			densityMap[y][x] = clamp(finalDensity, 0.05, 0.80)
		}
	}
	return densityMap
}

// getBiomeDensity returns base obstacle density for each biome
func (g *HybridGenerator) getBiomeDensity(biome Biome) float64 {
	switch biome {
	case BiomeGrassland:
		return 0.20 // Mostly open with scattered obstacles
	case BiomeForest:
		return 0.38 // Dense trees
	case BiomeDesert:
		return 0.15 // Very open with occasional rocks/dunes
	case BiomeMountain:
		return 0.50 // Rocky, many obstacles
	case BiomeSwamp:
		return 0.35 // Water + vegetation
	default:
		return g.config.BaseObstacleDensity
	}
}

// generateTerrain creates initial terrain from density map
func (g *HybridGenerator) generateTerrain(width, height int, densityMap [][]float64, noiseMap [][]float64) []bool {
	terrainMap := make([]bool, width*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			density := densityMap[y][x]

			// Use noise to determine if this tile is an obstacle
			// Higher noise value = more likely to be floor
			noiseVal := noiseMap[y][x]

			// Tile is floor if noise exceeds density threshold
			terrainMap[idx] = noiseVal > density
		}
	}

	return terrainMap
}

// cellularAutomataStep smooths the terrain
func (g *HybridGenerator) cellularAutomataStep(terrainMap []bool, width, height int) []bool {
	newMap := make([]bool, len(terrainMap))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x

			wallCount := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}

					nx, ny := x+dx, y+dy

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

			// 5+ wall neighbors = wall, else floor
			newMap[idx] = wallCount < 5
		}
	}

	return newMap
}

// ensureConnectivity connects all walkable regions
func (g *HybridGenerator) ensureConnectivity(terrainMap []bool, width, height int) {
	visited := make([]bool, len(terrainMap))
	var largestRegion []int
	maxSize := 0

	// Find all regions and identify largest
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

	// If no walkable region, create one in center
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

	// Connect all other regions to largest
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if !visited[idx] && terrainMap[idx] {
				region := g.floodFill(terrainMap, visited, x, y, width, height)
				if len(region) > 0 {
					g.carveCorridorBetween(terrainMap, width, largestRegion[0], region[0])
				}
			}
		}
	}
}

// floodFill finds all connected walkable tiles
func (g *HybridGenerator) floodFill(terrainMap, visited []bool, startX, startY, width, height int) []int {
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

// carveCorridorBetween creates L-shaped corridor between two points
func (g *HybridGenerator) carveCorridorBetween(terrainMap []bool, width, fromIdx, toIdx int) {
	fromX, fromY := fromIdx%width, fromIdx/width
	toX, toY := toIdx%width, toIdx/width

	// Horizontal segment
	if fromX < toX {
		for x := fromX; x <= toX; x++ {
			terrainMap[fromY*width+x] = true
		}
	} else {
		for x := fromX; x >= toX; x-- {
			terrainMap[fromY*width+x] = true
		}
	}

	// Vertical segment
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

// convertToTiles converts terrain and biome maps to actual tiles
func (g *HybridGenerator) convertToTiles(result *GenerationResult, terrainMap []bool, biomeMap [][]Biome, width, height int, images TileImageSet) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			terrainIdx := y*width + x
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			tileIdx := coords.CoordManager.LogicalToIndex(logicalPos)
			pixelX := x * graphics.ScreenInfo.TileSize
			pixelY := y * graphics.ScreenInfo.TileSize

			biome := biomeMap[y][x]

			// Get biome-specific images
			wallImages := images.WallImages
			floorImages := images.FloorImages

			biomeTileSet := images.BiomeImages[biome]
			if biomeTileSet != nil {
				if len(biomeTileSet.WallImages) > 0 {
					wallImages = biomeTileSet.WallImages
				}
				if len(biomeTileSet.FloorImages) > 0 {
					floorImages = biomeTileSet.FloorImages
				}
			}

			if terrainMap[terrainIdx] {
				// Walkable floor tile
				var floorImage *ebiten.Image
				if len(floorImages) > 0 {
					floorImage = floorImages[common.GetRandomBetween(0, len(floorImages)-1)]
				}
				tile := NewTile(pixelX, pixelY, logicalPos, false, floorImage, FLOOR, false)
				result.Tiles[tileIdx] = &tile
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			} else {
				// Wall/obstacle tile
				var wallImage *ebiten.Image
				if len(wallImages) > 0 {
					wallImage = wallImages[common.GetRandomBetween(0, len(wallImages)-1)]
				}
				tile := NewTile(pixelX, pixelY, logicalPos, true, wallImage, WALL, false)
				result.Tiles[tileIdx] = &tile
			}
		}
	}
}

// clearSpawnArea ensures the spawn area is walkable
func (g *HybridGenerator) clearSpawnArea(result *GenerationResult, centerX, centerY, radius, width, height int, images TileImageSet) {
	for x := centerX - radius; x <= centerX+radius; x++ {
		for y := centerY - radius; y <= centerY+radius; y++ {
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}

			logicalPos := coords.LogicalPosition{X: x, Y: y}
			idx := coords.CoordManager.LogicalToIndex(logicalPos)

			if idx >= 0 && idx < len(result.Tiles) {
				pixelX := x * graphics.ScreenInfo.TileSize
				pixelY := y * graphics.ScreenInfo.TileSize

				var floorImage *ebiten.Image
				if len(images.FloorImages) > 0 {
					floorImage = images.FloorImages[common.GetRandomBetween(0, len(images.FloorImages)-1)]
				}
				tile := NewTile(pixelX, pixelY, logicalPos, false, floorImage, FLOOR, false)
				result.Tiles[idx] = &tile
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			}
		}
	}
}

// Register this generator on package initialization
func init() {
	RegisterGenerator(NewHybridGenerator(DefaultHybridConfig()))
}
