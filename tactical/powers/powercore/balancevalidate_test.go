package powercore

import (
	"strings"
	"testing"
)

// testBalanceConfig mirrors the shape of PerkBalanceConfig/ArtifactBalanceConfig:
// a struct of per-power structs whose numeric fields carry balance tags.
type testBalanceConfig struct {
	Alpha testAlphaBalance `json:"alpha"`
	Beta  testBetaBalance  `json:"beta"`
}

type testAlphaBalance struct {
	CoverBonus float64 `json:"coverBonus" balance:"fraction"`
	DamageMult float64 `json:"damageMult" balance:"mult"`
	Note       string  // non-numeric fields need no tag
}

type testBetaBalance struct {
	MaxStacks int `json:"maxStacks" balance:"count"`
	HitBonus  int `json:"hitBonus" balance:"bonus"`
}

func validTestConfig() testBalanceConfig {
	return testBalanceConfig{
		Alpha: testAlphaBalance{CoverBonus: 0.15, DamageMult: 1.25},
		Beta:  testBetaBalance{MaxStacks: 3, HitBonus: 0},
	}
}

func TestValidateBalanceRanges_ValidConfigPasses(t *testing.T) {
	cfg := validTestConfig()
	if errs := ValidateBalanceRanges(&cfg); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

// errorContaining asserts exactly one error mentioning the given json path.
func errorContaining(t *testing.T, errs []error, path string) {
	t.Helper()
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), path) {
		t.Errorf("error %q does not mention %q", errs[0], path)
	}
}

func TestValidateBalanceRanges_KindViolations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testBalanceConfig)
		path   string
	}{
		{"fraction zero", func(c *testBalanceConfig) { c.Alpha.CoverBonus = 0 }, "alpha.coverBonus"},
		{"fraction one", func(c *testBalanceConfig) { c.Alpha.CoverBonus = 1.0 }, "alpha.coverBonus"},
		{"mult too low", func(c *testBalanceConfig) { c.Alpha.DamageMult = 0.05 }, "alpha.damageMult"},
		{"mult too high", func(c *testBalanceConfig) { c.Alpha.DamageMult = 11 }, "alpha.damageMult"},
		{"count zero", func(c *testBalanceConfig) { c.Beta.MaxStacks = 0 }, "beta.maxStacks"},
		{"count over cap", func(c *testBalanceConfig) { c.Beta.MaxStacks = 101 }, "beta.maxStacks"},
		{"bonus negative", func(c *testBalanceConfig) { c.Beta.HitBonus = -1 }, "beta.hitBonus"},
		{"bonus over cap", func(c *testBalanceConfig) { c.Beta.HitBonus = 101 }, "beta.hitBonus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validTestConfig()
			tt.mutate(&cfg)
			errorContaining(t, ValidateBalanceRanges(&cfg), tt.path)
		})
	}
}

func TestValidateBalanceRanges_BoundaryValuesPass(t *testing.T) {
	cfg := validTestConfig()
	cfg.Alpha.DamageMult = 0.1 // mult bounds are inclusive
	cfg.Beta.MaxStacks = 100   // count upper bound is inclusive
	cfg.Beta.HitBonus = 100    // bonus upper bound is inclusive
	if errs := ValidateBalanceRanges(&cfg); len(errs) != 0 {
		t.Fatalf("expected boundary values to pass, got %v", errs)
	}
	cfg.Alpha.DamageMult = 10.0
	if errs := ValidateBalanceRanges(&cfg); len(errs) != 0 {
		t.Fatalf("expected boundary values to pass, got %v", errs)
	}
}

func TestValidateBalanceRanges_MissingTag(t *testing.T) {
	type untagged struct {
		Value float64 `json:"value"`
	}
	cfg := struct {
		Gamma untagged `json:"gamma"`
	}{Gamma: untagged{Value: 0.5}}

	errs := ValidateBalanceRanges(&cfg)
	errorContaining(t, errs, "gamma.value")
	if !strings.Contains(errs[0].Error(), "missing balance tag") {
		t.Errorf("expected missing-tag error, got %q", errs[0])
	}
}

func TestValidateBalanceRanges_UnknownKind(t *testing.T) {
	type badKind struct {
		Value float64 `json:"value" balance:"percentage"`
	}
	cfg := struct {
		Gamma badKind `json:"gamma"`
	}{Gamma: badKind{Value: 0.5}}

	errs := ValidateBalanceRanges(&cfg)
	errorContaining(t, errs, "gamma.value")
	if !strings.Contains(errs[0].Error(), `unknown balance kind "percentage"`) {
		t.Errorf("expected unknown-kind error, got %q", errs[0])
	}
}

func TestValidateBalanceRanges_RejectsNonStructInput(t *testing.T) {
	if errs := ValidateBalanceRanges(42); len(errs) != 1 {
		t.Fatalf("expected 1 error for non-struct input, got %v", errs)
	}
	var nilCfg *testBalanceConfig
	if errs := ValidateBalanceRanges(nilCfg); len(errs) != 1 {
		t.Fatalf("expected 1 error for nil input, got %v", errs)
	}
}
