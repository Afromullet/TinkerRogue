package worldmap

import (
	"game_main/common"
	"game_main/coords"
	"game_main/graphics"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// PerlinBiomeGenerator generates maps using Perlin noise to select a biome,
// then creates a battlefield within that biome
type PerlinBiomeGenerator struct {
	config GeneratorConfig
}

// NewPerlinBiomeGenerator creates a new Perlin-based biome generator
func NewPerlinBiomeGenerator(config GeneratorConfig) *PerlinBiomeGenerator {
	return &PerlinBiomeGenerator{config: config}
}

func (g *PerlinBiomeGenerator) Name() string {
	return "perlin_biome"
}

func (g *PerlinBiomeGenerator) Description() string {
	return "Perlin noise biome selection with battlefield generation: Grassland, Forest, Desert, Mountain, Swamp"
}

func (g *PerlinBiomeGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	// Determine the dominant biome using Perlin noise
	biome := g.selectBiome(images)

	// Generate open terrain in that biome with natural features
	result := g.generateBiomeTermain(width, height, biome, images)

	return result
}

// selectBiome uses simplified Perlin-like noise to determine which biome to generate
func (g *PerlinBiomeGenerator) selectBiome(images TileImageSet) Biome {
	// Use seed for reproducibility if set, otherwise use random
	seed := g.config.Seed
	if seed == 0 {
		seed = rand.Int63()
	}

	// Create RNG with the seed
	rng := rand.New(rand.NewSource(seed))

	// Sample noise at 3 points and average for smoother biome selection
	var noiseSum float64
	for i := 0; i < 3; i++ {
		x := float64(rng.Intn(1000))
		y := float64(rng.Intn(1000))
		noiseVal := g.simplexNoise(x, y)
		// Normalize from [-1, 1] to [0, 1]
		normalized := (noiseVal + 1.0) / 2.0
		noiseSum += normalized
	}

	avgNoise := noiseSum / 3.0
	return BiomeFromNoise(avgNoise)
}

// simplexNoise generates a simple hash-based noise value
// Not true Simplex noise, but provides consistent pseudo-random values
// based on input coordinates with good distribution
func (g *PerlinBiomeGenerator) simplexNoise(x, y float64) float64 {
	// Use a simple hash function based on coordinates
	xi := int(math.Floor(x))
	yi := int(math.Floor(y))

	// Hash function combining coordinates
	hash := uint64(73856093)
	hash ^= uint64(xi) * 19349663
	hash ^= uint64(yi) * 83492791

	// Mix the hash bits
	hash ^= hash >> 33
	hash *= 0xff51afd7ed558ccd
	hash ^= hash >> 33

	// Convert to float in range [-1, 1]
	val := float64(hash%2000000000) / 1000000000.0
	return val - 1.0
}

// generateBiomeTerrain creates open terrain with natural features for the biome
func (g *PerlinBiomeGenerator) generateBiomeTermain(width, height int, biome Biome, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:          make([]*Tile, width*height),
		Rooms:          make([]Rect, 0),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	biomeTileSet := images.BiomeImages[biome]
	var wallImages []*ebiten.Image
	var floorImages []*ebiten.Image

	// Get biome-specific images
	if biomeTileSet != nil && len(biomeTileSet.WallImages) > 0 {
		wallImages = biomeTileSet.WallImages
		floorImages = biomeTileSet.FloorImages
	}
	if len(wallImages) == 0 {
		wallImages = images.WallImages
	}
	if len(floorImages) == 0 {
		floorImages = images.FloorImages
	}

	// Generate terrain using noise function
	// Different biomes have different walkability thresholds
	walkThreshold := g.getWalkThreshold(biome)

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)

			// Use noise to determine if this tile is walkable
			noiseVal := g.getTileNoise(x, y, biome)

			var tile Tile
			if noiseVal < walkThreshold {
				// Walkable floor tile
				tile = NewTile(
					x*graphics.ScreenInfo.TileSize,
					y*graphics.ScreenInfo.TileSize,
					logicalPos, false,
					floorImages[common.GetRandomBetween(0, len(floorImages)-1)],
					FLOOR, false,
				)
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			} else {
				// Obstacle/wall tile
				tile = NewTile(
					x*graphics.ScreenInfo.TileSize,
					y*graphics.ScreenInfo.TileSize,
					logicalPos, true,
					wallImages[common.GetRandomBetween(0, len(wallImages)-1)],
					WALL, false,
				)
			}
			result.Tiles[index] = &tile
		}
	}

	// Ensure some open area in the middle for spawn point
	g.clearSpawnArea(&result, width/2, height/2, 5, floorImages)

	return result
}

// getWalkThreshold returns the noise threshold for walkable tiles based on biome
// Lower threshold = more walkable tiles
func (g *PerlinBiomeGenerator) getWalkThreshold(biome Biome) float64 {
	switch biome {
	case BiomeGrassland:
		return 0.3 // Mostly walkable grassland
	case BiomeForest:
		return 0.5 // Medium walkability, scattered trees
	case BiomeDesert:
		return 0.25 // Mostly open sand
	case BiomeMountain:
		return 0.75 // Mostly rocky/mountainous
	case BiomeSwamp:
		return 0.55 // Medium walkability, some water
	default:
		return 0.5
	}
}

// getTileNoise returns the noise value for a specific tile
// Uses biome-specific noise patterns
func (g *PerlinBiomeGenerator) getTileNoise(x, y int, biome Biome) float64 {
	// Apply biome-specific noise scaling
	scale := g.getNoiseScale(biome)

	// Hash-based noise at this coordinate
	xi := int(math.Floor(float64(x) * scale))
	yi := int(math.Floor(float64(y) * scale))

	hash := uint64(73856093)
	hash ^= uint64(xi) * 19349663
	hash ^= uint64(yi) * 83492791

	// Mix the hash bits
	hash ^= hash >> 33
	hash *= 0xff51afd7ed558ccd
	hash ^= hash >> 33

	// Normalize to [0, 1]
	val := float64(hash%1000000000) / 1000000000.0
	return val
}

// getNoiseScale returns the noise scale for the biome
// Higher scale = larger features
func (g *PerlinBiomeGenerator) getNoiseScale(biome Biome) float64 {
	switch biome {
	case BiomeGrassland:
		return 0.1 // Small scattered features
	case BiomeForest:
		return 0.15 // Medium-sized tree clusters
	case BiomeDesert:
		return 0.2 // Large dune formations
	case BiomeMountain:
		return 0.25 // Large mountain ranges
	case BiomeSwamp:
		return 0.12 // Medium water patches
	default:
		return 0.15
	}
}

// clearSpawnArea ensures there's a clear area for player spawn
func (g *PerlinBiomeGenerator) clearSpawnArea(result *GenerationResult, centerX, centerY, radius int, floorImages []*ebiten.Image) {
	for x := centerX - radius; x <= centerX+radius; x++ {
		for y := centerY - radius; y <= centerY+radius; y++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)

			if index >= 0 && index < len(result.Tiles) {
				result.Tiles[index].Blocked = false
				result.Tiles[index].TileType = FLOOR
				if len(floorImages) > 0 {
					result.Tiles[index].image = floorImages[common.GetRandomBetween(0, len(floorImages)-1)]
				}
			}
		}
	}
}

// Register this generator on package initialization
func init() {
	RegisterGenerator(NewPerlinBiomeGenerator(DefaultConfig()))
}
