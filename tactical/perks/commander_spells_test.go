package perks

import (
	"game_main/common"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// TEST HELPERS
// ========================================

func addCommanderPerkComponent(manager *common.EntityManager, entityID ecs.EntityID, perks [3]string) {
	entity := manager.FindEntityByID(entityID)
	if entity != nil {
		entity.AddComponent(CommanderPerkComponent, &CommanderPerkData{EquippedPerks: perks})
	}
}

func createTestCommander(manager *common.EntityManager, perks [3]string) ecs.EntityID {
	entity := manager.World.NewEntity()
	entity.AddComponent(CommanderPerkComponent, &CommanderPerkData{EquippedPerks: perks})
	return entity.GetID()
}

// ========================================
// SPELL COST TESTS
// ========================================

func TestModifySpellCost_EfficientCasting(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"efficient_casting", "", ""})

	result := ModifySpellCost(cmdID, 15, manager)
	// 15 * 0.8 = 12
	if result != 12 {
		t.Errorf("expected cost 12, got %d", result)
	}
}

func TestModifySpellCost_Overcharge(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"overcharge", "", ""})

	result := ModifySpellCost(cmdID, 15, manager)
	// 15 * 1.5 = 22 (int truncation)
	if result != 22 {
		t.Errorf("expected cost 22, got %d", result)
	}
}

func TestModifySpellCost_NoPerks(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"", "", ""})

	result := ModifySpellCost(cmdID, 15, manager)
	if result != 15 {
		t.Errorf("expected cost 15 unchanged, got %d", result)
	}
}

func TestModifySpellCost_MinimumOne(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"efficient_casting", "", ""})

	// Very low cost: 1 * 0.8 = 0.8 -> int(0) -> clamped to 1
	result := ModifySpellCost(cmdID, 1, manager)
	if result != 1 {
		t.Errorf("expected minimum cost 1, got %d", result)
	}
}

func TestModifySpellCost_NoComponent(t *testing.T) {
	manager := setupTestManager(t)
	// Entity without CommanderPerkComponent
	entity := manager.World.NewEntity()
	entityID := entity.GetID()

	result := ModifySpellCost(entityID, 15, manager)
	if result != 15 {
		t.Errorf("expected cost 15 unchanged without component, got %d", result)
	}
}

// ========================================
// SPELL DAMAGE TESTS
// ========================================

func TestModifySpellDamage_SpellMastery(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"spell_mastery", "", ""})

	result := ModifySpellDamage(cmdID, 20, manager)
	// 20 * 1.25 = 25
	if result != 25 {
		t.Errorf("expected damage 25, got %d", result)
	}
}

func TestModifySpellDamage_Overcharge(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"overcharge", "", ""})

	result := ModifySpellDamage(cmdID, 20, manager)
	// 20 * 1.75 = 35
	if result != 35 {
		t.Errorf("expected damage 35, got %d", result)
	}
}

func TestModifySpellDamage_NoPerks(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"", "", ""})

	result := ModifySpellDamage(cmdID, 20, manager)
	if result != 20 {
		t.Errorf("expected damage 20 unchanged, got %d", result)
	}
}

// ========================================
// SPELL DURATION TESTS
// ========================================

func TestModifySpellDuration_LingeringMagic(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"lingering_magic", "", ""})

	result := ModifySpellDuration(cmdID, 3, manager)
	// 3 + 2 = 5
	if result != 5 {
		t.Errorf("expected duration 5, got %d", result)
	}
}

func TestModifySpellDuration_NoPerks(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"", "", ""})

	result := ModifySpellDuration(cmdID, 3, manager)
	if result != 3 {
		t.Errorf("expected duration 3 unchanged, got %d", result)
	}
}

// ========================================
// SPELL MODIFIER TESTS
// ========================================

func TestModifySpellModifier_PotentEnchantment(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"potent_enchantment", "", ""})

	result := ModifySpellModifier(cmdID, 4, manager)
	// 4 * 1.5 = 6
	if result != 6 {
		t.Errorf("expected modifier 6, got %d", result)
	}
}

func TestModifySpellModifier_NegativeDebuff(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"potent_enchantment", "", ""})

	result := ModifySpellModifier(cmdID, -3, manager)
	// abs(3) * 1.5 = 4.5 -> int(4) == 3? No: int(4.5) = 4, 4 != 3, so scaled=4, result=-4
	if result != -4 {
		t.Errorf("expected modifier -4, got %d", result)
	}
}

func TestModifySpellModifier_NegativeDebuff_SmallValue(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"potent_enchantment", "", ""})

	// -1 * 1.5 = 1.5 -> int(1) == abs(-1), so ensure at least +1 magnitude
	result := ModifySpellModifier(cmdID, -1, manager)
	if result != -2 {
		t.Errorf("expected modifier -2, got %d", result)
	}
}

func TestModifySpellModifier_NoPerks(t *testing.T) {
	manager := setupTestManager(t)
	cmdID := createTestCommander(manager, [3]string{"", "", ""})

	result := ModifySpellModifier(cmdID, 4, manager)
	if result != 4 {
		t.Errorf("expected modifier 4 unchanged, got %d", result)
	}
}

// ========================================
// EXCLUSIVITY TESTS
// ========================================

func TestExclusivity_EfficientCasting_Overcharge(t *testing.T) {
	manager := setupTestManager(t)
	entity := manager.World.NewEntity()
	entityID := entity.GetID()
	entity.AddComponent(CommanderPerkComponent, &CommanderPerkData{
		EquippedPerks: [3]string{"efficient_casting", "", ""},
	})

	reason := CanEquipPerk(entityID, "overcharge", 1, manager)
	if reason == "" {
		t.Error("expected exclusive conflict between efficient_casting and overcharge")
	}
}

func TestExclusivity_SpellMastery_PotentEnchantment(t *testing.T) {
	manager := setupTestManager(t)
	entity := manager.World.NewEntity()
	entityID := entity.GetID()
	entity.AddComponent(CommanderPerkComponent, &CommanderPerkData{
		EquippedPerks: [3]string{"spell_mastery", "", ""},
	})

	reason := CanEquipPerk(entityID, "potent_enchantment", 1, manager)
	if reason == "" {
		t.Error("expected exclusive conflict between spell_mastery and potent_enchantment")
	}
}
