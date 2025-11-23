package worldmap

import (
	"game_main/common"
	"game_main/coords"
	"math"
	"math/rand"
)

// OverworldGenerator generates large-scale world maps using noise-based terrain
type OverworldGenerator struct {
	config OverworldConfig
	seed   int64
}

// OverworldConfig holds parameters for overworld generation
type OverworldConfig struct {
	ElevationScale float64 // Noise scale for elevation (smaller = larger features)
	MoistureScale  float64 // Noise scale for moisture
	MountainThresh float64 // Elevation threshold for mountains (0.0-1.0)
	WaterThresh    float64 // Elevation threshold for water
	ForestThresh   float64 // Moisture threshold for forests
	POICount       int     // Number of towns/points of interest to place
	POIMinDistance int     // Minimum distance between POIs
	Seed           int64   // 0 = use time-based seed
}

// DefaultOverworldConfig returns sensible defaults
func DefaultOverworldConfig() OverworldConfig {
	return OverworldConfig{
		ElevationScale: 0.08,
		MoistureScale:  0.08,
		MountainThresh: 0.6,
		WaterThresh:    0.3,
		ForestThresh:   0.55,
		POICount:       5,
		POIMinDistance: 15,
		Seed:           0,
	}
}

// NewOverworldGenerator creates a new overworld generator
func NewOverworldGenerator(config OverworldConfig) *OverworldGenerator {
	return &OverworldGenerator{config: config}
}

func (g *OverworldGenerator) Name() string {
	return "overworld"
}

func (g *OverworldGenerator) Description() string {
	return "Large-scale world map with biomes: grasslands, forests, mountains, water"
}

func (g *OverworldGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:          createEmptyTiles(width, height, images),
		Rooms:          make([]Rect, 0),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	// Initialize RNG with seed
	if g.config.Seed != 0 {
		rand.Seed(g.config.Seed)
	}

	// Generate elevation and moisture maps
	elevationMap := g.generateNoiseMap(width, height, g.config.ElevationScale)
	moistureMap := g.generateNoiseMap(width, height, g.config.MoistureScale)

	// Create terrain based on elevation and moisture
	g.createTerrain(&result, width, height, elevationMap, moistureMap, images)

	// Place towns/POIs
	g.placePOIs(&result, width, height, images)

	return result
}

// generateNoiseMap creates a Perlin-like noise map using simple interpolation
// This is a simplified noise generator that creates organic-looking patterns
func (g *OverworldGenerator) generateNoiseMap(width, height int, scale float64) [][]float64 {
	noiseMap := make([][]float64, height)
	for i := range noiseMap {
		noiseMap[i] = make([]float64, width)
	}

	// Generate random gradient vectors at grid points
	gridWidth := int(math.Ceil(float64(width)*scale)) + 2
	gridHeight := int(math.Ceil(float64(height)*scale)) + 2
	gradients := g.generateGradients(gridWidth, gridHeight)

	// Interpolate across the map
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			noiseValue := g.perlinInterpolate(float64(x), float64(y), scale, gradients, gridWidth, gridHeight)
			// Normalize to 0-1
			noiseMap[y][x] = (noiseValue + 1.0) / 2.0
			if noiseMap[y][x] < 0 {
				noiseMap[y][x] = 0
			}
			if noiseMap[y][x] > 1.0 {
				noiseMap[y][x] = 1.0
			}
		}
	}

	return noiseMap
}

// generateGradients creates random gradient vectors for Perlin-like noise
func (g *OverworldGenerator) generateGradients(width, height int) [][]float64 {
	gradients := make([][]float64, height)
	for i := range gradients {
		gradients[i] = make([]float64, width)
		for j := range gradients[i] {
			gradients[i][j] = rand.Float64() * 2.0 * math.Pi
		}
	}
	return gradients
}

// perlinInterpolate interpolates noise value at a given point
func (g *OverworldGenerator) perlinInterpolate(x, y, scale float64, gradients [][]float64, gridWidth, gridHeight int) float64 {
	// Scale coordinates to grid space
	scaledX := x * scale
	scaledY := y * scale

	// Get grid cell
	gridX := int(scaledX)
	gridY := int(scaledY)

	// Bounds check
	if gridX < 0 || gridX >= gridWidth-1 || gridY < 0 || gridY >= gridHeight-1 {
		return 0
	}

	// Local coordinates within grid cell (0-1)
	fx := scaledX - float64(gridX)
	fy := scaledY - float64(gridY)

	// Smooth interpolation curve
	u := fx * fx * (3.0 - 2.0*fx)
	v := fy * fy * (3.0 - 2.0*fy)

	// Sample gradients at four corners
	g00 := math.Sin(gradients[gridY][gridX])
	g10 := math.Sin(gradients[gridY][gridX+1])
	g01 := math.Sin(gradients[gridY+1][gridX])
	g11 := math.Sin(gradients[gridY+1][gridX+1])

	// Interpolate
	nx0 := g00*(1-u) + g10*u
	nx1 := g01*(1-u) + g11*u
	result := nx0*(1-v) + nx1*v

	return result
}

// determineBiome classifies terrain based on elevation and moisture
func (g *OverworldGenerator) determineBiome(elevation, moisture float64) Biome {
	// Water is lowest elevation (treat as Swamp for now since we don't have a water biome)
	if elevation < g.config.WaterThresh {
		return BiomeSwamp
	}

	// Mountains are highest elevation
	if elevation > g.config.MountainThresh {
		return BiomeMountain
	}

	// Mid elevation: forest if wet, grassland/desert if dry
	if moisture > g.config.ForestThresh {
		return BiomeForest
	}

	// Check if it's desert-like (high elevation, low moisture) or grassland
	if elevation > g.config.MountainThresh*0.8 {
		return BiomeDesert
	}

	return BiomeGrassland
}

// createTerrain converts noise maps to tile types
func (g *OverworldGenerator) createTerrain(result *GenerationResult, width, height int, elevationMap, moistureMap [][]float64, images TileImageSet) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			elevation := elevationMap[y][x]
			moisture := moistureMap[y][x]
			biome := g.determineBiome(elevation, moisture)

			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)

			if index < 0 || index >= len(result.Tiles) {
				continue
			}

			tile := result.Tiles[index]

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

			switch biome {
			case BiomeSwamp:
				tile.TileType = WALL // Swamp/Water is impassable
				tile.Blocked = true
				if len(wallImages) > 0 {
					tile.image = wallImages[common.GetRandomBetween(0, len(wallImages)-1)]
				}

			case BiomeMountain:
				tile.TileType = WALL // Mountain is impassable
				tile.Blocked = true
				if len(wallImages) > 0 {
					tile.image = wallImages[common.GetRandomBetween(0, len(wallImages)-1)]
				}

			case BiomeDesert:
				tile.TileType = FLOOR // Desert is traversable but harsh
				tile.Blocked = false
				if len(floorImages) > 0 {
					tile.image = floorImages[common.GetRandomBetween(0, len(floorImages)-1)]
				}
				result.ValidPositions = append(result.ValidPositions, logicalPos)

			case BiomeForest:
				tile.TileType = FLOOR // Forest is traversable
				tile.Blocked = false
				if len(floorImages) > 0 {
					tile.image = floorImages[common.GetRandomBetween(0, len(floorImages)-1)]
				}
				result.ValidPositions = append(result.ValidPositions, logicalPos)

			case BiomeGrassland:
				tile.TileType = FLOOR // Grassland is traversable
				tile.Blocked = false
				if len(floorImages) > 0 {
					tile.image = floorImages[common.GetRandomBetween(0, len(floorImages)-1)]
				}
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			}
		}
	}
}

// placePOIs places towns and points of interest on valid terrain
func (g *OverworldGenerator) placePOIs(result *GenerationResult, width, height int, images TileImageSet) {
	if len(result.ValidPositions) == 0 || g.config.POICount <= 0 {
		return
	}

	placedPOIs := 0
	maxAttempts := g.config.POICount * 10

	for attempt := 0; attempt < maxAttempts && placedPOIs < g.config.POICount; attempt++ {
		// Pick random valid position
		validIdx := common.GetDiceRoll(len(result.ValidPositions) - 1)
		poiPos := result.ValidPositions[validIdx]

		// Check distance from other POIs
		tooClose := false
		for _, placedPos := range result.Rooms {
			centerX, centerY := placedPos.Center()
			dx := float64(centerX - poiPos.X)
			dy := float64(centerY - poiPos.Y)
			distance := math.Sqrt(dx*dx + dy*dy)
			if distance < float64(g.config.POIMinDistance) {
				tooClose = true
				break
			}
		}

		if !tooClose {
			// Store POI location as a "room" (using Rect for simplicity)
			poiRect := NewRect(poiPos.X, poiPos.Y, 1, 1)
			result.Rooms = append(result.Rooms, poiRect)
			placedPOIs++
		}
	}
}

// Register this generator on package initialization
func init() {
	RegisterGenerator(NewOverworldGenerator(DefaultOverworldConfig()))
}
