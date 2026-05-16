package templates

import (
	"game_main/core/common"
	"strings"
	"testing"
)

// setupNameConfig creates a minimal name config for testing
func setupNameConfig() {
	NameConfigTemplate = JSONNameConfig{
		NameFormat:   "{name} the {type}",
		MinSyllables: 2,
		MaxSyllables: 3,
		Pools: map[string]JSONNamePool{
			"default": {
				Prefixes: []string{"Al", "Bran", "Cor"},
				Middles:  []string{"an", "en", "in"},
				Suffixes: []string{"ric", "dan", "wen"},
			},
			"elven": {
				Prefixes: []string{"Ael", "Cel", "Fae"},
				Middles:  []string{"li", "ri", "na"},
				Suffixes: []string{"thil", "wen", "dor"},
			},
		},
	}
	// Seed RNG for reproducible tests
	common.SetRNGSeed(42, 43)
}

func TestGenerateNameContainsType(t *testing.T) {
	setupNameConfig()

	name := GenerateName("default", "Knight")
	if !strings.Contains(name, "the Knight") {
		t.Errorf("expected name to contain 'the Knight', got %q", name)
	}
}

func TestGenerateNameUnknownPoolFallsBack(t *testing.T) {
	setupNameConfig()

	// Unknown pool should fall back to default and still work
	name := GenerateName("nonexistent", "Wizard")
	if !strings.Contains(name, "the Wizard") {
		t.Errorf("expected fallback name to contain 'the Wizard', got %q", name)
	}
	if name == "" {
		t.Error("expected non-empty name from fallback pool")
	}
}

func TestGenerateNameNamedPool(t *testing.T) {
	setupNameConfig()

	name := GenerateName("elven", "Archer")
	if !strings.Contains(name, "the Archer") {
		t.Errorf("expected elven name to contain 'the Archer', got %q", name)
	}
}

func TestGenerateNameFormatWithoutType(t *testing.T) {
	setupNameConfig()
	// Override format to exclude {type}
	NameConfigTemplate.NameFormat = "{name}"

	name := GenerateName("default", "Knight")
	if strings.Contains(name, "Knight") {
		t.Errorf("expected name without type, got %q", name)
	}
	if name == "" {
		t.Error("expected non-empty name")
	}
}

func TestGenerateNameProducesVariety(t *testing.T) {
	setupNameConfig()

	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		name := GenerateName("default", "Knight")
		seen[name] = true
	}

	if len(seen) < 2 {
		t.Errorf("expected at least 2 unique names from 20 generations, got %d", len(seen))
	}
}

func TestValidateNameConfigErrorsOnMissingDefault(t *testing.T) {
	config := &JSONNameConfig{
		NameFormat:   "{name} the {type}",
		MinSyllables: 2,
		MaxSyllables: 3,
		Pools: map[string]JSONNamePool{
			"elven": {
				Prefixes: []string{"Ael"},
				Suffixes: []string{"thil"},
			},
		},
	}
	if err := validateNameConfig(config); err == nil {
		t.Error("expected error for missing default pool")
	}
}

func TestValidateNameConfigErrorsOnEmptyPrefixes(t *testing.T) {
	config := &JSONNameConfig{
		NameFormat:   "{name} the {type}",
		MinSyllables: 2,
		MaxSyllables: 3,
		Pools: map[string]JSONNamePool{
			"default": {
				Prefixes: []string{},
				Suffixes: []string{"ric"},
			},
		},
	}
	if err := validateNameConfig(config); err == nil {
		t.Error("expected error for empty prefixes")
	}
}

func TestValidateNameConfigErrorsOnInvalidSyllables(t *testing.T) {
	config := &JSONNameConfig{
		NameFormat:   "{name} the {type}",
		MinSyllables: 1,
		MaxSyllables: 3,
		Pools: map[string]JSONNamePool{
			"default": {
				Prefixes: []string{"Al"},
				Suffixes: []string{"ric"},
			},
		},
	}
	if err := validateNameConfig(config); err == nil {
		t.Error("expected error for minSyllables < 2")
	}
}
