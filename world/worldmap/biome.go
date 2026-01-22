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
