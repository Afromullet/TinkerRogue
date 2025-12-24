package worldmap

type Biome int

const (
	BiomeGrassland Biome = iota
	BiomeForest
	BiomeDesert
	BiomeMountain
	BiomeSwamp
)

func (b Biome) String() string {
	switch b {
	case BiomeGrassland:
		return "grassland"
	case BiomeForest:
		return "forest"
	case BiomeDesert:
		return "desert"
	case BiomeMountain:
		return "mountain"
	case BiomeSwamp:
		return "swamp"
	default:
		return "unknown"
	}
}

// BiomeFromNoise converts a noise value to a biome
// noise value should be in range [0, 1]
func BiomeFromNoise(noiseValue float64) Biome {
	// Map noise ranges to biomes
	// 0.0-0.2: Swamp
	// 0.2-0.4: Forest
	// 0.4-0.6: Grassland
	// 0.6-0.8: Desert
	// 0.8-1.0: Mountain

	if noiseValue < 0.2 {
		return BiomeSwamp
	} else if noiseValue < 0.4 {
		return BiomeForest
	} else if noiseValue < 0.6 {
		return BiomeGrassland
	} else if noiseValue < 0.8 {
		return BiomeDesert
	} else {
		return BiomeMountain
	}
}
