package worldgen

import (
	"log"
	"sort"

	"game_main/world/worldmapcore"
)

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
// then falling back to the registry, then to the default "rooms_corridors".
// Logs a warning when falling back from an unknown name so typos in save
// files or config don't silently swap the algorithm.
func GetGenerator(name string) worldmapcore.MapGenerator {
	if ConfigOverride != nil {
		if gen := ConfigOverride(name); gen != nil {
			return gen
		}
	}
	if gen, ok := generators[name]; ok {
		return gen
	}
	log.Printf("worldgen: generator %q not registered; falling back to rooms_corridors (known: %v)",
		name, registeredNames())
	return generators["rooms_corridors"]
}

// registeredNames returns the sorted list of registered generator names.
func registeredNames() []string {
	names := make([]string, 0, len(generators))
	for n := range generators {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
