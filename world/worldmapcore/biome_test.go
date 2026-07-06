package worldmapcore

import "testing"

func TestBiomeStringRoundTrip(t *testing.T) {
	biomes := []Biome{BiomeGrassland, BiomeForest, BiomeDesert, BiomeMountain, BiomeSwamp}
	for _, b := range biomes {
		t.Run(b.String(), func(t *testing.T) {
			if got := BiomeFromString(b.String()); got != b {
				t.Errorf("BiomeFromString(%q) = %v, want %v", b.String(), got, b)
			}
		})
	}
}

func TestBiomeUnknownFallbacks(t *testing.T) {
	if got := Biome(99).String(); got != "unknown" {
		t.Errorf("Biome(99).String() = %q, want %q", got, "unknown")
	}
	if got := BiomeFromString("nonsense"); got != BiomeGrassland {
		t.Errorf("BiomeFromString(\"nonsense\") = %v, want BiomeGrassland", got)
	}
	// Pinned asymmetry: "unknown" does not round-trip; it falls back to
	// grassland like any unrecognized string.
	if got := BiomeFromString("unknown"); got != BiomeGrassland {
		t.Errorf("BiomeFromString(\"unknown\") = %v, want BiomeGrassland", got)
	}
}
