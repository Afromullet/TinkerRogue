package worldmap

import "path/filepath"

// Base directory for all tile assets
var tileAssetBase = filepath.Join("..", "assets", "tiles")

// Default tile set directories
var defaultFloorDir = "limestone"
var defaultWallDir = "marble"

// Stairs filename
var stairsFile = "stairs1.png"

// allBiomes is the canonical biome list (single source of truth)
var allBiomes = []Biome{BiomeGrassland, BiomeForest, BiomeDesert, BiomeMountain, BiomeSwamp}

// POI type identifiers (single source of truth for POI string keys)
const (
	POITown       = "town"
	POITemple     = "temple"
	POIGuildHall  = "guild_hall"
	POIWatchtower = "watchtower"
)

// poiAssetConfig maps POI type to its relative image path under tileAssetBase
var poiAssetConfig = map[string]string{
	POITown:       filepath.Join("maptiles", POITown, "dithmenos2.png"),
	POITemple:     filepath.Join("maptiles", POITemple, "golden_statue_1.png"),
	POIGuildHall:  filepath.Join("maptiles", POIGuildHall, "machine_tukima.png"),
	POIWatchtower: filepath.Join("maptiles", POIWatchtower, "crumbled_column_1.png"),
}

func defaultFloorPath() string { return filepath.Join(tileAssetBase, "floors", defaultFloorDir) }
func defaultWallPath() string  { return filepath.Join(tileAssetBase, "walls", defaultWallDir) }
func stairsPath() string       { return filepath.Join(tileAssetBase, stairsFile) }
func biomeFloorPath(biome Biome) string {
	return filepath.Join(tileAssetBase, "floors", biome.String())
}
func biomeWallPath(biome Biome) string { return filepath.Join(tileAssetBase, "walls", biome.String()) }

func poiAssetPath(poiType string) string {
	if rel, ok := poiAssetConfig[poiType]; ok {
		return filepath.Join(tileAssetBase, rel)
	}
	return ""
}
