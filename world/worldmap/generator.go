package worldmap

import (
	"game_main/world/coords"
)

// GenerationResult contains the output of a map generation algorithm
type GenerationResult struct {
	Tiles          []*Tile
	Rooms          []Rect
	ValidPositions []coords.LogicalPosition
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
