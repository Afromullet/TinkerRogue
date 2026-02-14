package effects

import (
	"game_main/common"
	"testing"
)

// setupTestManager creates a test EntityManager with effects components initialized.
func setupTestManager() *common.EntityManager {
	manager := common.NewEntityManager()

	common.PositionComponent = manager.World.NewComponent()
	common.AttributeComponent = manager.World.NewComponent()
	common.NameComponent = manager.World.NewComponent()

	common.GlobalPositionSystem = common.NewPositionSystem(manager.World)
	common.InitializeSubsystems(manager)

	return manager
}

// createTestUnit creates a unit entity with the given attributes.
func createTestUnit(manager *common.EntityManager, str, dex, mag int) *common.Attributes {
	attr := common.NewAttributes(str, dex, mag, 0, 0, 0)
	manager.World.NewEntity().
		AddComponent(common.AttributeComponent, &attr)
	return &attr
}

func TestApplyEffect_BuffIncreasesStrength(t *testing.T) {
	manager := setupTestManager()

	attr := common.NewAttributes(10, 5, 5, 0, 0, 0)
	entity := manager.World.NewEntity().
		AddComponent(common.AttributeComponent, &attr)
	entityID := entity.GetID()

	originalStrength := attr.Strength

	effect := ActiveEffect{
		Name:           "Test Buff",
		Source:         SourceSpell,
		Stat:           StatStrength,
		Modifier:       5,
		RemainingTurns: 3,
	}

	ApplyEffect(entityID, effect, manager)

	if attr.Strength != originalStrength+5 {
		t.Errorf("expected Strength %d, got %d", originalStrength+5, attr.Strength)
	}

	// Verify the effect was stored
	effectsData := GetActiveEffects(entityID, manager)
	if effectsData == nil || len(effectsData.Effects) != 1 {
		t.Fatalf("expected 1 active effect, got %v", effectsData)
	}
	if effectsData.Effects[0].RemainingTurns != 3 {
		t.Errorf("expected 3 remaining turns, got %d", effectsData.Effects[0].RemainingTurns)
	}
}

func TestApplyEffect_DebuffDecreasesDexterity(t *testing.T) {
	manager := setupTestManager()

	attr := common.NewAttributes(10, 10, 5, 0, 0, 0)
	entity := manager.World.NewEntity().
		AddComponent(common.AttributeComponent, &attr)
	entityID := entity.GetID()

	originalDex := attr.Dexterity

	effect := ActiveEffect{
		Name:           "Slow",
		Source:         SourceSpell,
		Stat:           StatDexterity,
		Modifier:       -3,
		RemainingTurns: 2,
	}

	ApplyEffect(entityID, effect, manager)

	if attr.Dexterity != originalDex-3 {
		t.Errorf("expected Dexterity %d, got %d", originalDex-3, attr.Dexterity)
	}
}

func TestApplyEffect_SkipsDeadUnits(t *testing.T) {
	manager := setupTestManager()

	attr := common.NewAttributes(10, 5, 5, 0, 0, 0)
	attr.CurrentHealth = 0 // dead
	entity := manager.World.NewEntity().
		AddComponent(common.AttributeComponent, &attr)
	entityID := entity.GetID()

	effect := ActiveEffect{
		Name:     "Should Not Apply",
		Source:   SourceSpell,
		Stat:     StatStrength,
		Modifier: 5,
	}

	ApplyEffect(entityID, effect, manager)

	if HasActiveEffects(entityID, manager) {
		t.Error("effect should not have been applied to dead unit")
	}
}

func TestTickEffects_ExpireAfterDuration(t *testing.T) {
	manager := setupTestManager()

	attr := common.NewAttributes(10, 5, 5, 0, 0, 0)
	entity := manager.World.NewEntity().
		AddComponent(common.AttributeComponent, &attr)
	entityID := entity.GetID()

	originalStrength := attr.Strength

	effect := ActiveEffect{
		Name:           "Short Buff",
		Source:         SourceSpell,
		Stat:           StatStrength,
		Modifier:       5,
		RemainingTurns: 3,
	}
	ApplyEffect(entityID, effect, manager)

	// Verify buff applied
	if attr.Strength != originalStrength+5 {
		t.Fatalf("buff not applied: expected %d, got %d", originalStrength+5, attr.Strength)
	}

	// Tick 1: 2 turns remaining
	TickEffects(entityID, manager)
	if attr.Strength != originalStrength+5 {
		t.Errorf("after tick 1: expected %d, got %d", originalStrength+5, attr.Strength)
	}

	// Tick 2: 1 turn remaining
	TickEffects(entityID, manager)
	if attr.Strength != originalStrength+5 {
		t.Errorf("after tick 2: expected %d, got %d", originalStrength+5, attr.Strength)
	}

	// Tick 3: 0 turns remaining — should expire and reverse
	TickEffects(entityID, manager)
	if attr.Strength != originalStrength {
		t.Errorf("after tick 3 (expired): expected %d, got %d", originalStrength, attr.Strength)
	}

	if HasActiveEffects(entityID, manager) {
		t.Error("effects should be empty after expiration")
	}
}

func TestTickEffects_PermanentEffectsSurvive(t *testing.T) {
	manager := setupTestManager()

	attr := common.NewAttributes(10, 5, 5, 0, 0, 0)
	entity := manager.World.NewEntity().
		AddComponent(common.AttributeComponent, &attr)
	entityID := entity.GetID()

	originalArmor := attr.Armor

	effect := ActiveEffect{
		Name:           "Permanent Armor",
		Source:         SourceAbility,
		Stat:           StatArmor,
		Modifier:       3,
		RemainingTurns: -1, // permanent
	}
	ApplyEffect(entityID, effect, manager)

	// Tick many times — permanent effect should survive
	for i := 0; i < 10; i++ {
		TickEffects(entityID, manager)
	}

	if attr.Armor != originalArmor+3 {
		t.Errorf("permanent effect should persist: expected Armor %d, got %d", originalArmor+3, attr.Armor)
	}

	if !HasActiveEffects(entityID, manager) {
		t.Error("permanent effect should still be present")
	}
}

func TestRemoveAllEffects_ReversesModifiers(t *testing.T) {
	manager := setupTestManager()

	attr := common.NewAttributes(10, 10, 5, 0, 0, 0)
	entity := manager.World.NewEntity().
		AddComponent(common.AttributeComponent, &attr)
	entityID := entity.GetID()

	originalStr := attr.Strength
	originalDex := attr.Dexterity

	ApplyEffect(entityID, ActiveEffect{
		Name: "Str Buff", Source: SourceSpell, Stat: StatStrength,
		Modifier: 5, RemainingTurns: 3,
	}, manager)

	ApplyEffect(entityID, ActiveEffect{
		Name: "Dex Debuff", Source: SourceSpell, Stat: StatDexterity,
		Modifier: -2, RemainingTurns: 2,
	}, manager)

	// Verify both applied
	if attr.Strength != originalStr+5 || attr.Dexterity != originalDex-2 {
		t.Fatal("effects not applied correctly")
	}

	RemoveAllEffects(entityID, manager)

	if attr.Strength != originalStr {
		t.Errorf("after RemoveAll: expected Strength %d, got %d", originalStr, attr.Strength)
	}
	if attr.Dexterity != originalDex {
		t.Errorf("after RemoveAll: expected Dexterity %d, got %d", originalDex, attr.Dexterity)
	}
}

func TestParseStatType(t *testing.T) {
	tests := []struct {
		input    string
		expected StatType
	}{
		{"strength", StatStrength},
		{"Strength", StatStrength},
		{"dexterity", StatDexterity},
		{"magic", StatMagic},
		{"leadership", StatLeadership},
		{"armor", StatArmor},
		{"weapon", StatWeapon},
		{"movementspeed", StatMovementSpeed},
		{"MovementSpeed", StatMovementSpeed},
		{"attackrange", StatAttackRange},
		{"unknown", StatStrength}, // default
	}

	for _, tt := range tests {
		result := ParseStatType(tt.input)
		if result != tt.expected {
			t.Errorf("ParseStatType(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestMultipleEffectsOnSameEntity(t *testing.T) {
	manager := setupTestManager()

	attr := common.NewAttributes(10, 10, 10, 0, 5, 5)
	entity := manager.World.NewEntity().
		AddComponent(common.AttributeComponent, &attr)
	entityID := entity.GetID()

	originalStr := attr.Strength
	originalArmor := attr.Armor

	// Apply buff and debuff to different stats
	ApplyEffect(entityID, ActiveEffect{
		Name: "Str Up", Source: SourceSpell, Stat: StatStrength,
		Modifier: 4, RemainingTurns: 2,
	}, manager)
	ApplyEffect(entityID, ActiveEffect{
		Name: "Armor Down", Source: SourceSpell, Stat: StatArmor,
		Modifier: -2, RemainingTurns: 3,
	}, manager)

	if attr.Strength != originalStr+4 {
		t.Errorf("expected Strength %d, got %d", originalStr+4, attr.Strength)
	}
	if attr.Armor != originalArmor-2 {
		t.Errorf("expected Armor %d, got %d", originalArmor-2, attr.Armor)
	}

	// Tick twice — str buff expires, armor debuff has 1 turn left
	TickEffects(entityID, manager)
	TickEffects(entityID, manager)

	if attr.Strength != originalStr {
		t.Errorf("str buff should have expired: expected %d, got %d", originalStr, attr.Strength)
	}
	if attr.Armor != originalArmor-2 {
		t.Errorf("armor debuff should still be active: expected %d, got %d", originalArmor-2, attr.Armor)
	}

	// Tick once more — armor debuff expires
	TickEffects(entityID, manager)
	if attr.Armor != originalArmor {
		t.Errorf("armor debuff should have expired: expected %d, got %d", originalArmor, attr.Armor)
	}
}
