package squads

import (
	"game_main/common"
	testfx "game_main/testing"
	"math/rand"
	"testing"
)

// setupExperienceTestManager creates a manager for experience tests
func setupExperienceTestManager(t *testing.T) *common.EntityManager {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	return manager
}

// createTestUnitWithGrowth creates a unit entity with experience and growth components
func createTestUnitWithGrowth(t *testing.T, manager *common.EntityManager, growths StatGrowthData) (*ExperienceData, *common.Attributes) {
	t.Helper()

	entity := manager.World.NewEntity()
	entity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength:      10,
		Dexterity:     20,
		Magic:         5,
		Leadership:    10,
		Armor:         5,
		Weapon:        5,
		MaxHealth:     40,
		CurrentHealth: 40,
		CanAct:        true,
	})

	expData := &ExperienceData{
		Level:         1,
		CurrentXP:     0,
		XPToNextLevel: 100,
	}
	entity.AddComponent(ExperienceComponent, expData)
	entity.AddComponent(StatGrowthComponent, &growths)

	return expData, common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
}

func TestGrowthChance(t *testing.T) {
	tests := []struct {
		grade    GrowthGrade
		expected int
	}{
		{GradeS, 90},
		{GradeA, 75},
		{GradeB, 60},
		{GradeC, 45},
		{GradeD, 30},
		{GradeE, 15},
		{GradeF, 5},
		{GrowthGrade(""), 0},
		{GrowthGrade("X"), 0},
	}

	for _, tc := range tests {
		got := GrowthChance(tc.grade)
		if got != tc.expected {
			t.Errorf("GrowthChance(%q) = %d, want %d", tc.grade, got, tc.expected)
		}
	}
}

func TestAwardExperience_BasicLevelUp(t *testing.T) {
	manager := setupExperienceTestManager(t)

	entity := manager.World.NewEntity()
	unitID := entity.GetID()

	entity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength:      10,
		Dexterity:     20,
		Magic:         5,
		Leadership:    10,
		Armor:         5,
		Weapon:        5,
		MaxHealth:     40,
		CurrentHealth: 40,
		CanAct:        true,
	})

	entity.AddComponent(ExperienceComponent, &ExperienceData{
		Level:         1,
		CurrentXP:     0,
		XPToNextLevel: 100,
	})

	// All S grades = 90% chance each, seeded RNG for determinism
	entity.AddComponent(StatGrowthComponent, &StatGrowthData{
		Strength:   GradeS,
		Dexterity:  GradeS,
		Magic:      GradeS,
		Leadership: GradeS,
		Armor:      GradeS,
		Weapon:     GradeS,
	})

	rng := rand.New(rand.NewSource(42))

	// Award enough XP for exactly 1 level up
	AwardExperience(unitID, 100, manager, rng)

	expData := GetExperienceData(unitID, manager)
	if expData.Level != 2 {
		t.Errorf("expected level 2, got %d", expData.Level)
	}
	if expData.CurrentXP != 0 {
		t.Errorf("expected 0 current XP, got %d", expData.CurrentXP)
	}
}

func TestAwardExperience_PartialXP(t *testing.T) {
	manager := setupExperienceTestManager(t)

	entity := manager.World.NewEntity()
	unitID := entity.GetID()

	entity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength: 10, Dexterity: 20, Magic: 5,
		Leadership: 10, Armor: 5, Weapon: 5,
		MaxHealth: 40, CurrentHealth: 40, CanAct: true,
	})
	entity.AddComponent(ExperienceComponent, &ExperienceData{
		Level: 1, CurrentXP: 0, XPToNextLevel: 100,
	})
	entity.AddComponent(StatGrowthComponent, &StatGrowthData{
		Strength: GradeC, Dexterity: GradeC, Magic: GradeC,
		Leadership: GradeC, Armor: GradeC, Weapon: GradeC,
	})

	rng := rand.New(rand.NewSource(42))

	// Award 50 XP - should not level up
	AwardExperience(unitID, 50, manager, rng)

	expData := GetExperienceData(unitID, manager)
	if expData.Level != 1 {
		t.Errorf("expected level 1 (no level up), got %d", expData.Level)
	}
	if expData.CurrentXP != 50 {
		t.Errorf("expected 50 current XP, got %d", expData.CurrentXP)
	}
}

func TestAwardExperience_MultiLevelJump(t *testing.T) {
	manager := setupExperienceTestManager(t)

	entity := manager.World.NewEntity()
	unitID := entity.GetID()

	entity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength: 10, Dexterity: 20, Magic: 5,
		Leadership: 10, Armor: 5, Weapon: 5,
		MaxHealth: 40, CurrentHealth: 40, CanAct: true,
	})
	entity.AddComponent(ExperienceComponent, &ExperienceData{
		Level: 1, CurrentXP: 0, XPToNextLevel: 100,
	})
	entity.AddComponent(StatGrowthComponent, &StatGrowthData{
		Strength: GradeC, Dexterity: GradeC, Magic: GradeC,
		Leadership: GradeC, Armor: GradeC, Weapon: GradeC,
	})

	rng := rand.New(rand.NewSource(42))

	// Award 350 XP = 3 level ups with 50 remaining
	AwardExperience(unitID, 350, manager, rng)

	expData := GetExperienceData(unitID, manager)
	if expData.Level != 4 {
		t.Errorf("expected level 4, got %d", expData.Level)
	}
	if expData.CurrentXP != 50 {
		t.Errorf("expected 50 remaining XP, got %d", expData.CurrentXP)
	}
}

func TestAwardExperience_ZeroAndNegative(t *testing.T) {
	manager := setupExperienceTestManager(t)

	entity := manager.World.NewEntity()
	unitID := entity.GetID()

	entity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength: 10, Dexterity: 20, Magic: 5,
		Leadership: 10, Armor: 5, Weapon: 5,
		MaxHealth: 40, CurrentHealth: 40, CanAct: true,
	})
	entity.AddComponent(ExperienceComponent, &ExperienceData{
		Level: 1, CurrentXP: 30, XPToNextLevel: 100,
	})
	entity.AddComponent(StatGrowthComponent, &StatGrowthData{
		Strength: GradeC, Dexterity: GradeC, Magic: GradeC,
		Leadership: GradeC, Armor: GradeC, Weapon: GradeC,
	})

	rng := rand.New(rand.NewSource(42))

	// Zero XP should not change anything
	AwardExperience(unitID, 0, manager, rng)
	expData := GetExperienceData(unitID, manager)
	if expData.CurrentXP != 30 {
		t.Errorf("zero XP changed state: expected 30 XP, got %d", expData.CurrentXP)
	}

	// Negative XP should not change anything
	AwardExperience(unitID, -10, manager, rng)
	if expData.CurrentXP != 30 {
		t.Errorf("negative XP changed state: expected 30 XP, got %d", expData.CurrentXP)
	}
}

func TestLevelUp_StatIncreases(t *testing.T) {
	manager := setupExperienceTestManager(t)

	entity := manager.World.NewEntity()
	unitID := entity.GetID()

	initialStr := 10
	entity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength: initialStr, Dexterity: 20, Magic: 5,
		Leadership: 10, Armor: 5, Weapon: 5,
		MaxHealth: 40, CurrentHealth: 40, CanAct: true,
	})
	entity.AddComponent(ExperienceComponent, &ExperienceData{
		Level: 1, CurrentXP: 0, XPToNextLevel: 100,
	})

	// All S grades = 90% chance; with seed 42, most should trigger
	entity.AddComponent(StatGrowthComponent, &StatGrowthData{
		Strength: GradeS, Dexterity: GradeS, Magic: GradeS,
		Leadership: GradeS, Armor: GradeS, Weapon: GradeS,
	})

	rng := rand.New(rand.NewSource(42))

	// Level up 10 times
	AwardExperience(unitID, 1000, manager, rng)

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)

	// With S grade (90%) and 10 level ups, statistically expect ~9 per stat
	// With seeded RNG this is deterministic. Just verify stats increased.
	if attr.Strength <= initialStr {
		t.Errorf("expected strength to increase from %d after 10 S-grade level ups, got %d",
			initialStr, attr.Strength)
	}
}

func TestLevelUp_MaxHealthRecalculated(t *testing.T) {
	manager := setupExperienceTestManager(t)

	entity := manager.World.NewEntity()
	unitID := entity.GetID()

	entity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength: 10, Dexterity: 20, Magic: 5,
		Leadership: 10, Armor: 5, Weapon: 5,
		MaxHealth: 40, CurrentHealth: 40, CanAct: true,
	})
	entity.AddComponent(ExperienceComponent, &ExperienceData{
		Level: 1, CurrentXP: 0, XPToNextLevel: 100,
	})

	// Guaranteed strength growth
	entity.AddComponent(StatGrowthComponent, &StatGrowthData{
		Strength: GradeS, Dexterity: GradeF, Magic: GradeF,
		Leadership: GradeF, Armor: GradeF, Weapon: GradeF,
	})

	initialMaxHP := 40 // 20 + 10*2

	rng := rand.New(rand.NewSource(42))
	AwardExperience(unitID, 1000, manager, rng) // 10 level ups

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)

	// MaxHealth should be recalculated: 20 + Strength*2
	expectedMaxHP := 20 + attr.Strength*2
	if attr.MaxHealth != expectedMaxHP {
		t.Errorf("MaxHealth not recalculated: got %d, expected %d (strength=%d)",
			attr.MaxHealth, expectedMaxHP, attr.Strength)
	}

	if attr.MaxHealth <= initialMaxHP {
		t.Errorf("MaxHealth should have increased from %d, got %d",
			initialMaxHP, attr.MaxHealth)
	}
}

func TestLevelUp_FGradeRarelyIncreases(t *testing.T) {
	manager := setupExperienceTestManager(t)

	entity := manager.World.NewEntity()
	unitID := entity.GetID()

	entity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength: 10, Dexterity: 20, Magic: 5,
		Leadership: 10, Armor: 5, Weapon: 5,
		MaxHealth: 40, CurrentHealth: 40, CanAct: true,
	})
	entity.AddComponent(ExperienceComponent, &ExperienceData{
		Level: 1, CurrentXP: 0, XPToNextLevel: 100,
	})

	// All F grades = 5% chance each
	entity.AddComponent(StatGrowthComponent, &StatGrowthData{
		Strength: GradeF, Dexterity: GradeF, Magic: GradeF,
		Leadership: GradeF, Armor: GradeF, Weapon: GradeF,
	})

	rng := rand.New(rand.NewSource(42))

	// 5 level ups with F grade (5% each) - total increase should be small
	AwardExperience(unitID, 500, manager, rng)

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)

	// Sum of all stat increases should be small (5% * 6 stats * 5 levels = ~1.5 expected)
	totalIncrease := (attr.Strength - 10) + (attr.Dexterity - 20) + (attr.Magic - 5) +
		(attr.Leadership - 10) + (attr.Armor - 5) + (attr.Weapon - 5)

	if totalIncrease > 10 {
		t.Errorf("F grade produced too many stat increases: %d total (expected low)", totalIncrease)
	}
}

func TestGetExperienceData_NilForMissingComponent(t *testing.T) {
	manager := setupExperienceTestManager(t)

	entity := manager.World.NewEntity()
	unitID := entity.GetID()

	// No ExperienceComponent added
	_ = entity

	expData := GetExperienceData(unitID, manager)
	if expData != nil {
		t.Error("expected nil for entity without ExperienceComponent")
	}
}

func TestGetStatGrowthData_NilForMissingComponent(t *testing.T) {
	manager := setupExperienceTestManager(t)

	entity := manager.World.NewEntity()
	unitID := entity.GetID()

	// No StatGrowthComponent added
	_ = entity

	growthData := GetStatGrowthData(unitID, manager)
	if growthData != nil {
		t.Error("expected nil for entity without StatGrowthComponent")
	}
}
