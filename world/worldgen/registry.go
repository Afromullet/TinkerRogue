package worldgen

import "game_main/world/worldmapcore"

// ConfigOverride is an optional hook that returns a configured MapGenerator
// for the given name. If set and returns non-nil, the returned generator
// replaces the default-registered one for that generation call.
// Set by game_main after loading JSON config.
var ConfigOverride func(name string) worldmapcore.MapGenerator

// Generator registry for algorithm selection
var generators = make(map[string]worldmapcore.MapGenerator)

// RegisterGenerator adds a new algorithm to the registry
func RegisterGenerator(gen worldmapcore.MapGenerator) {
	generators[gen.Name()] = gen
}

// GetGenerator retrieves algorithm by name, checking ConfigOverride first,
// then falling back to registry, then to default "rooms_corridors".
func GetGenerator(name string) worldmapcore.MapGenerator {
	// Check for config override first
	if ConfigOverride != nil {
		if gen := ConfigOverride(name); gen != nil {
			return gen
		}
	}
	// Fall back to registry
	gen := generators[name]
	if gen == nil {
		gen = generators["rooms_corridors"] // Default fallback
	}
	return gen
}
