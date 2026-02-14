package templates

import (
	"math"
	"testing"
)

// newTestDifficultyManager builds a DifficultyManager with all 3 presets derived.
func newTestDifficultyManager() *DifficultyManager {
	presets := []DifficultyPreset{
		{Name: DifficultyEasy, CombatIntensity: 0.7, OverworldPressure: 0.6, AICompetence: 0.7, EncounterSizeOffset: -1},
		{Name: DifficultyMedium, CombatIntensity: 1.0, OverworldPressure: 1.0, AICompetence: 1.0, EncounterSizeOffset: 0},
		{Name: DifficultyHard, CombatIntensity: 1.4, OverworldPressure: 1.4, AICompetence: 1.3, EncounterSizeOffset: 1},
	}

	m := make(map[string]*DifficultyPreset, len(presets))
	for i := range presets {
		deriveDifficultyValues(&presets[i])
		m[presets[i].Name] = &presets[i]
	}

	dm := &DifficultyManager{presets: m}
	dm.current.Store(m[DifficultyMedium])
	return dm
}

func TestNewDefaultDifficultyManager(t *testing.T) {
	dm := NewDefaultDifficultyManager()

	if dm.GetCurrentDifficulty() != DifficultyMedium {
		t.Fatalf("expected Medium, got %s", dm.GetCurrentDifficulty())
	}

	// Medium should produce identity values
	enc := dm.Encounter()
	if enc.PowerMultiplierScale != 1.0 {
		t.Errorf("PowerMultiplierScale: got %f, want 1.0", enc.PowerMultiplierScale)
	}
	if enc.SquadCountOffset != 0 || enc.MinUnitsPerSquadOffset != 0 || enc.MaxUnitsPerSquadOffset != 0 {
		t.Errorf("encounter offsets should all be 0, got %d/%d/%d",
			enc.SquadCountOffset, enc.MinUnitsPerSquadOffset, enc.MaxUnitsPerSquadOffset)
	}

	ow := dm.Overworld()
	if ow.ThreatGrowthScale != 1.0 || ow.SpawnChanceScale != 1.0 || ow.RaidIntensityScale != 1.0 ||
		ow.FortificationStrengthGainScale != 1.0 || ow.ContainmentSlowdownScale != 1.0 {
		t.Errorf("overworld scales should all be 1.0")
	}
	if ow.MaxThreatIntensityOffset != 0 {
		t.Errorf("MaxThreatIntensityOffset: got %d, want 0", ow.MaxThreatIntensityOffset)
	}

	ai := dm.AI()
	if ai.SharedRangedWeightScale != 1.0 || ai.SharedPositionalWeightScale != 1.0 {
		t.Errorf("AI weight scales should be 1.0")
	}
	if ai.FlankingRangeBonusOffset != 0 || ai.IsolationThresholdOffset != 0 || ai.RetreatSafeThresholdOffset != 0 {
		t.Errorf("AI offsets should all be 0")
	}
}

func TestSetDifficulty(t *testing.T) {
	dm := newTestDifficultyManager()

	if err := dm.SetDifficulty(DifficultyHard); err != nil {
		t.Fatalf("SetDifficulty Hard: %v", err)
	}
	if dm.GetCurrentDifficulty() != DifficultyHard {
		t.Errorf("expected Hard, got %s", dm.GetCurrentDifficulty())
	}

	if err := dm.SetDifficulty(DifficultyEasy); err != nil {
		t.Fatalf("SetDifficulty Easy: %v", err)
	}
	if dm.GetCurrentDifficulty() != DifficultyEasy {
		t.Errorf("expected Easy, got %s", dm.GetCurrentDifficulty())
	}
}

func TestSetDifficultyInvalidName(t *testing.T) {
	dm := newTestDifficultyManager()

	err := dm.SetDifficulty("Nightmare")
	if err == nil {
		t.Fatal("expected error for unknown difficulty, got nil")
	}
}

func TestDerivationEasy(t *testing.T) {
	dm := newTestDifficultyManager()
	_ = dm.SetDifficulty(DifficultyEasy)

	enc := dm.Encounter()
	assertFloat(t, "PowerMultiplierScale", enc.PowerMultiplierScale, 0.7)
	assertInt(t, "SquadCountOffset", enc.SquadCountOffset, -1)
	assertInt(t, "MinUnitsPerSquadOffset", enc.MinUnitsPerSquadOffset, -1)
	assertInt(t, "MaxUnitsPerSquadOffset", enc.MaxUnitsPerSquadOffset, -1)

	ow := dm.Overworld()
	assertFloat(t, "ThreatGrowthScale", ow.ThreatGrowthScale, 0.6)
	assertFloat(t, "SpawnChanceScale", ow.SpawnChanceScale, 0.6)
	assertFloat(t, "RaidIntensityScale", ow.RaidIntensityScale, 0.6)
	assertFloat(t, "FortificationStrengthGainScale", ow.FortificationStrengthGainScale, 0.6)
	// ContainmentSlowdownScale = 1.0 + (0.6 - 1.0) * 0.75 = 1.0 - 0.3 = 0.7
	assertFloat(t, "ContainmentSlowdownScale", ow.ContainmentSlowdownScale, 0.7)
	// MaxThreatIntensityOffset = round((0.6 - 1.0) * 5) = round(-2.0) = -2
	assertInt(t, "MaxThreatIntensityOffset", ow.MaxThreatIntensityOffset, -2)

	ai := dm.AI()
	// FlankingRangeBonusOffset = round((0.7 - 1.0) * 5) = round(-1.5) = -2
	assertInt(t, "FlankingRangeBonusOffset", ai.FlankingRangeBonusOffset, -2)
	// IsolationThresholdOffset = -round((0.7 - 1.0) * 3) = -round(-0.9) = -(-1) = 1
	assertInt(t, "IsolationThresholdOffset", ai.IsolationThresholdOffset, 1)
	// RetreatSafeThresholdOffset = round((0.7 - 1.0) * 7) = round(-2.1) = -2
	assertInt(t, "RetreatSafeThresholdOffset", ai.RetreatSafeThresholdOffset, -2)
	assertFloat(t, "SharedRangedWeightScale", ai.SharedRangedWeightScale, 0.7)
	assertFloat(t, "SharedPositionalWeightScale", ai.SharedPositionalWeightScale, 0.7)
}

func TestDerivationHard(t *testing.T) {
	dm := newTestDifficultyManager()
	_ = dm.SetDifficulty(DifficultyHard)

	enc := dm.Encounter()
	assertFloat(t, "PowerMultiplierScale", enc.PowerMultiplierScale, 1.4)
	assertInt(t, "SquadCountOffset", enc.SquadCountOffset, 1)
	assertInt(t, "MinUnitsPerSquadOffset", enc.MinUnitsPerSquadOffset, 1)
	assertInt(t, "MaxUnitsPerSquadOffset", enc.MaxUnitsPerSquadOffset, 1)

	ow := dm.Overworld()
	assertFloat(t, "ThreatGrowthScale", ow.ThreatGrowthScale, 1.4)
	assertFloat(t, "SpawnChanceScale", ow.SpawnChanceScale, 1.4)
	assertFloat(t, "RaidIntensityScale", ow.RaidIntensityScale, 1.4)
	assertFloat(t, "FortificationStrengthGainScale", ow.FortificationStrengthGainScale, 1.4)
	// ContainmentSlowdownScale = 1.0 + (1.4 - 1.0) * 0.75 = 1.0 + 0.3 = 1.3
	assertFloat(t, "ContainmentSlowdownScale", ow.ContainmentSlowdownScale, 1.3)
	// MaxThreatIntensityOffset = round((1.4 - 1.0) * 5) = round(2.0) = 2
	assertInt(t, "MaxThreatIntensityOffset", ow.MaxThreatIntensityOffset, 2)

	ai := dm.AI()
	// FlankingRangeBonusOffset = round((1.3 - 1.0) * 5) = round(1.5) = 2
	assertInt(t, "FlankingRangeBonusOffset", ai.FlankingRangeBonusOffset, 2)
	// IsolationThresholdOffset = -round((1.3 - 1.0) * 3) = -round(0.9) = -1
	assertInt(t, "IsolationThresholdOffset", ai.IsolationThresholdOffset, -1)
	// RetreatSafeThresholdOffset = round((1.3 - 1.0) * 7) = round(2.1) = 2
	assertInt(t, "RetreatSafeThresholdOffset", ai.RetreatSafeThresholdOffset, 2)
	assertFloat(t, "SharedRangedWeightScale", ai.SharedRangedWeightScale, 1.3)
	assertFloat(t, "SharedPositionalWeightScale", ai.SharedPositionalWeightScale, 1.3)
}

func assertFloat(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s: got %f, want %f", name, got, want)
	}
}

func assertInt(t *testing.T, name string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %d, want %d", name, got, want)
	}
}
