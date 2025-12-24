package worldmap

import (
	"game_main/common"
	"game_main/coords"
	"game_main/visual/graphics"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// WaveletGenerator creates tactical maps using Wavelet Noise Procedural (WNP)
// Provides precise control over terrain feature sizes at multiple scales
type WaveletGenerator struct {
	config WaveletConfig
}

// WaveletConfig holds parameters for wavelet-based generation
type WaveletConfig struct {
	// Wavelet band configuration
	NumBands        int     // Number of frequency bands (4-6 typical)
	BaseAmplitude   float64 // Weight of lowest frequency band
	AmplitudeDecay  float64 // Amplitude falloff per band (0.5 typical)

	// Terrain style mixing
	RidgeWeight     float64 // How much ridged noise to blend (0.0-1.0)
	TurbulenceBlend float64 // How much turbulent noise to add (0.0-0.5)

	// Biome determination
	MoistureBands   int     // Bands for moisture noise
	ElevationBands  int     // Bands for elevation noise

	// Feature enhancement
	DetailBoost     float64 // Amplify fine detail (1.0-2.0)
	ContrastPower   float64 // Terrain contrast adjustment (0.8-1.5)

	// Cellular automata smoothing
	SmoothingPasses int     // CA iterations (3-5)

	// Density
	BaseObstacleDensity float64

	Seed int64
}

// DefaultWaveletConfig returns sensible defaults
func DefaultWaveletConfig() WaveletConfig {
	return WaveletConfig{
		// Wavelet bands
		NumBands:       5,
		BaseAmplitude:  1.0,
		AmplitudeDecay: 0.55,

		// Style mixing
		RidgeWeight:     0.25, // Some ridge features
		TurbulenceBlend: 0.15, // Light turbulence

		// Biome
		MoistureBands:  4,
		ElevationBands: 4,

		// Enhancement
		DetailBoost:   1.3,
		ContrastPower: 1.1,

		// Smoothing
		SmoothingPasses: 4,

		// Density
		BaseObstacleDensity: 0.30,

		Seed: 0,
	}
}

// NewWaveletGenerator creates a new wavelet generator
func NewWaveletGenerator(config WaveletConfig) *WaveletGenerator {
	return &WaveletGenerator{config: config}
}

func (g *WaveletGenerator) Name() string {
	return "wavelet_procedural"
}

func (g *WaveletGenerator) Description() string {
	return "Wavelet Noise Procedural maps with precise multi-scale terrain control"
}

func (g *WaveletGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
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

	// Step 1: Generate primary wavelet terrain
	baseWavelet := noiseGen.GenerateWaveletNoise(width, height, g.config.NumBands, g.config.BaseAmplitude, g.config.AmplitudeDecay)

	// Step 2: Generate ridged features (mountains, canyon walls)
	ridgedWavelet := noiseGen.GenerateRidgedWavelet(width, height, g.config.NumBands, 0.0, 2.0)

	// Step 3: Generate turbulent variation
	turbulentWavelet := noiseGen.GenerateTurbulentWavelet(width, height, g.config.NumBands-1)

	// Step 4: Blend terrain types
	terrainNoise := g.blendTerrainTypes(width, height, baseWavelet, ridgedWavelet, turbulentWavelet)

	// Step 5: Apply detail boost and contrast
	terrainNoise = g.enhanceTerrain(width, height, terrainNoise)

	// Step 6: Generate biome maps using wavelet noise
	elevationNoise := noiseGen.GenerateWaveletNoise(width, height, g.config.ElevationBands, 1.0, 0.6)
	moistureNoise := noiseGen.GenerateWaveletNoise(width, height, g.config.MoistureBands, 1.0, 0.5)

	// Step 7: Determine biomes
	biomeMap := g.determineBiomeMap(width, height, elevationNoise, moistureNoise)

	// Step 8: Calculate density with biome modifiers
	densityMap := g.calculateDensityMap(width, height, biomeMap, terrainNoise)

	// Step 9: Generate terrain from density
	terrainMap := g.generateTerrain(width, height, densityMap, terrainNoise)

	// Step 10: Apply cellular automata smoothing
	for i := 0; i < g.config.SmoothingPasses; i++ {
		terrainMap = g.cellularAutomataStep(terrainMap, width, height)
	}

	// Step 11: Ensure connectivity
	g.ensureConnectivity(terrainMap, width, height)

	// Step 12: Convert to tiles
	g.convertToTiles(&result, terrainMap, biomeMap, width, height, images)

	// Step 13: Clear spawn area
	centerX := width / 2
	centerY := height / 2
	g.clearSpawnArea(&result, centerX, centerY, 5, width, height, images)

	return result
}

// blendTerrainTypes combines different wavelet noise styles
func (g *WaveletGenerator) blendTerrainTypes(width, height int, base, ridged, turbulent [][]float64) [][]float64 {
	result := make([][]float64, height)
	for i := range result {
		result[i] = make([]float64, width)
	}

	baseWeight := 1.0 - g.config.RidgeWeight - g.config.TurbulenceBlend
	if baseWeight < 0.3 {
		baseWeight = 0.3 // Ensure base is always significant
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			result[y][x] = base[y][x]*baseWeight +
				ridged[y][x]*g.config.RidgeWeight +
				turbulent[y][x]*g.config.TurbulenceBlend
		}
	}

	return result
}

// enhanceTerrain applies detail boost and contrast adjustment
func (g *WaveletGenerator) enhanceTerrain(width, height int, terrain [][]float64) [][]float64 {
	result := make([][]float64, height)
	for i := range result {
		result[i] = make([]float64, width)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v := terrain[y][x]

			// Detail boost: amplify deviation from 0.5
			v = 0.5 + (v-0.5)*g.config.DetailBoost

			// Contrast adjustment using power function
			// Values < 0.5 get pushed lower, > 0.5 get pushed higher
			if v < 0.5 {
				v = 0.5 * pow01(v*2.0, g.config.ContrastPower)
			} else {
				v = 0.5 + 0.5*pow01((v-0.5)*2.0, g.config.ContrastPower)
			}

			result[y][x] = clamp(v, 0.0, 1.0)
		}
	}

	return result
}

// pow01 is power function safe for 0-1 range
func pow01(base, exp float64) float64 {
	if base <= 0 {
		return 0
	}
	if base >= 1 {
		return 1
	}
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	// Handle fractional part
	frac := exp - float64(int(exp))
	if frac > 0 {
		result *= (1.0-frac) + frac*base
	}
	return result
}

// determineBiomeMap creates biome assignments based on elevation and moisture
func (g *WaveletGenerator) determineBiomeMap(width, height int, elevation, moisture [][]float64) [][]Biome {
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
func (g *WaveletGenerator) determineBiome(elevation, moisture float64) Biome {
	const (
		lowElevation  = 0.30
		midElevation  = 0.55
		highElevation = 0.75

		dryMoisture = 0.35
		wetMoisture = 0.60
	)

	// Very low = swamp/water
	if elevation < lowElevation {
		return BiomeSwamp
	}

	// Very high = mountain
	if elevation > highElevation {
		return BiomeMountain
	}

	// Mid elevation depends on moisture
	if moisture > wetMoisture {
		return BiomeForest
	}

	if moisture < dryMoisture {
		if elevation > midElevation {
			return BiomeMountain
		}
		return BiomeDesert
	}

	return BiomeGrassland
}

// calculateDensityMap creates per-tile obstacle density
func (g *WaveletGenerator) calculateDensityMap(width, height int, biomeMap [][]Biome, terrainNoise [][]float64) [][]float64 {
	densityMap := make([][]float64, height)
	for y := 0; y < height; y++ {
		densityMap[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			biome := biomeMap[y][x]
			baseDensity := g.getBiomeDensity(biome)

			// Use terrain noise for local variation
			// Areas with extreme noise values have different density
			noiseVal := terrainNoise[y][x]
			variationMod := 0.7 + noiseVal*0.6 // 0.7 to 1.3

			densityMap[y][x] = clamp(baseDensity*variationMod, 0.05, 0.75)
		}
	}
	return densityMap
}

// getBiomeDensity returns base obstacle density for each biome
func (g *WaveletGenerator) getBiomeDensity(biome Biome) float64 {
	switch biome {
	case BiomeGrassland:
		return 0.22
	case BiomeForest:
		return 0.40
	case BiomeDesert:
		return 0.18
	case BiomeMountain:
		return 0.52
	case BiomeSwamp:
		return 0.35
	default:
		return g.config.BaseObstacleDensity
	}
}

// generateTerrain creates initial terrain from density map
func (g *WaveletGenerator) generateTerrain(width, height int, densityMap, noiseMap [][]float64) []bool {
	terrainMap := make([]bool, width*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			density := densityMap[y][x]
			noiseVal := noiseMap[y][x]

			// Floor if noise exceeds density
			terrainMap[idx] = noiseVal > density
		}
	}

	return terrainMap
}

// cellularAutomataStep smooths terrain
func (g *WaveletGenerator) cellularAutomataStep(terrainMap []bool, width, height int) []bool {
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

			newMap[idx] = wallCount < 5
		}
	}

	return newMap
}

// ensureConnectivity connects all walkable regions
func (g *WaveletGenerator) ensureConnectivity(terrainMap []bool, width, height int) {
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

	if maxSize == 0 {
		// No walkable region - create center area
		for y := height / 4; y < (height * 3 / 4); y++ {
			for x := width / 4; x < (width * 3 / 4); x++ {
				terrainMap[y*width+x] = true
			}
		}
		return
	}

	// Mark largest and connect others
	visited = make([]bool, len(terrainMap))
	for _, idx := range largestRegion {
		visited[idx] = true
	}

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

// floodFill finds connected walkable tiles
func (g *WaveletGenerator) floodFill(terrainMap, visited []bool, startX, startY, width, height int) []int {
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

// carveCorridorBetween creates L-shaped corridor
func (g *WaveletGenerator) carveCorridorBetween(terrainMap []bool, width, fromIdx, toIdx int) {
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

// convertToTiles converts terrain and biome maps to tiles
func (g *WaveletGenerator) convertToTiles(result *GenerationResult, terrainMap []bool, biomeMap [][]Biome, width, height int, images TileImageSet) {
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
				var floorImage *ebiten.Image
				if len(floorImages) > 0 {
					floorImage = floorImages[common.GetRandomBetween(0, len(floorImages)-1)]
				}
				tile := NewTile(pixelX, pixelY, logicalPos, false, floorImage, FLOOR, false)
				result.Tiles[tileIdx] = &tile
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			} else {
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

// clearSpawnArea ensures spawn area is walkable
func (g *WaveletGenerator) clearSpawnArea(result *GenerationResult, centerX, centerY, radius, width, height int, images TileImageSet) {
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

// Register on package initialization
func init() {
	RegisterGenerator(NewWaveletGenerator(DefaultWaveletConfig()))
}
