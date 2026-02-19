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

// BiomeFromString converts a string to a Biome constant.
// Returns BiomeGrassland for unrecognized strings.
func BiomeFromString(s string) Biome {
	switch s {
	case "grassland":
		return BiomeGrassland
	case "forest":
		return BiomeForest
	case "desert":
		return BiomeDesert
	case "mountain":
		return BiomeMountain
	case "swamp":
		return BiomeSwamp
	default:
		return BiomeGrassland
	}
}
