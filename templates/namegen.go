package templates

import (
	"game_main/common"
	"strings"
)

// GenerateName creates a random fantasy name from syllable pools and applies the format template.
// poolName selects which syllable pool to use (falls back to "default" if not found).
// unitType is substituted into the {type} token in the name format.
func GenerateName(poolName string, unitType string) string {
	config := NameConfigTemplate

	// If no pools are configured (e.g., in tests), fall back to just the unit type
	if len(config.Pools) == 0 {
		return unitType
	}

	pool, exists := config.Pools[poolName]
	if !exists {
		pool = config.Pools["default"]
	}

	// Pick syllable count between min and max
	syllableCount := config.MinSyllables
	if config.MaxSyllables > config.MinSyllables {
		syllableCount = config.MinSyllables + common.RandomInt(config.MaxSyllables-config.MinSyllables+1)
	}

	// Build the syllable name: prefix + middles + suffix
	var parts []string

	// Prefix (always first)
	parts = append(parts, pool.Prefixes[common.RandomInt(len(pool.Prefixes))])

	// Middle syllables (count - 2, only if we have middles and count > 2)
	if len(pool.Middles) > 0 {
		for i := 0; i < syllableCount-2; i++ {
			parts = append(parts, pool.Middles[common.RandomInt(len(pool.Middles))])
		}
	}

	// Suffix (always last)
	parts = append(parts, pool.Suffixes[common.RandomInt(len(pool.Suffixes))])

	name := strings.Join(parts, "")

	// Apply format template
	result := config.NameFormat
	result = strings.ReplaceAll(result, "{name}", name)
	result = strings.ReplaceAll(result, "{type}", unitType)

	return result
}
