package worldmap

import (
	"game_main/world/coords"
)

// POIData describes a placed point of interest with type and terrain info
type POIData struct {
	Position coords.LogicalPosition
	NodeID   string // Maps to nodeDefinitions.json ID: POITown, POITemple, POIGuildHall, POIWatchtower
	Biome    Biome
}

// FactionStartPosition describes a generator-chosen starting position for a faction
type FactionStartPosition struct {
	Position coords.LogicalPosition
	Biome    Biome
	Sector   int // Which map sector (0-4)
}

// GenerationResult contains the output of a map generation algorithm
type GenerationResult struct {
	Tiles                 []*Tile
	Rooms                 []Rect
	ValidPositions        []coords.LogicalPosition
	POIs                  []POIData             // Typed points of interest
	FactionStartPositions []FactionStartPosition // Generator-chosen faction positions
	BiomeMap              []Biome               // Flat array indexed by positionToIndex
}

// MapGenerator defines the interface for all map generation algorithms
type MapGenerator interface {
	// Generate creates a new map layout
	Generate(width, height int, images TileImageSet) GenerationResult

	// Name returns the algorithm name for selection
	Name() string

	// Description returns a human-readable description
	Description() string
}

// GeneratorConfig holds common parameters for generators
type GeneratorConfig struct {
	MinRoomSize int
	MaxRoomSize int
	MaxRooms    int
	Seed        int64 // 0 = use time-based seed
}

// DefaultConfig returns sensible defaults for dungeon generation
func DefaultConfig() GeneratorConfig {
	return GeneratorConfig{
		MinRoomSize: 6,
		MaxRoomSize: 10,
		MaxRooms:    30,
		Seed:        0,
	}
}

// Generator registry for algorithm selection
var generators = make(map[string]MapGenerator)

// RegisterGenerator adds a new algorithm to the registry
func RegisterGenerator(gen MapGenerator) {
	generators[gen.Name()] = gen
}

// GetGeneratorOrDefault retrieves algorithm by name, falls back to default
func GetGeneratorOrDefault(name string) MapGenerator {
	gen := generators[name]
	if gen == nil {
		gen = generators["rooms_corridors"] // Default fallback
	}
	return gen
}
