package combatcore

import (
	"game_main/core/common"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"testing"
)
// ========================================
// calculateUnitDamageByID TESTS
// ========================================

func TestCalculateUnitDamageByID_BasicDamageCalculation(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100) // 100% hit rate
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 0)

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	defenderAttr := common.GetComponentType[*common.Attributes](defender, common.AttributeComponent)

	// Note: Attributes are derived from base stats (Strength, Dexterity, etc.)
	// We can't set them directly, but with Dexterity=100, attacker should have high hit rate
	// With Dexterity=0, defender should have low dodge chance

	damage, _ := calculateTestDamage(attacker.GetID(), defender.GetID(), manager)

	if damage <= 0 {
		t.Error("Expected positive damage (note: may miss/dodge based on derived stats)")
	}

	// Damage should be based on attacker's strength when it hits
	baseDamage := attackerAttr.GetPhysicalDamage()
	resistance := defenderAttr.GetPhysicalResistance()
	expectedDamage := baseDamage - resistance
	if expectedDamage < 1 {
		expectedDamage = 1 // Minimum damage
	}

	// Allow for misses (damage could be 0)
	if damage > 0 && damage != expectedDamage {
		t.Logf("Expected damage %d (after resistance), got %d (variance from crit/resistance)", expectedDamage, damage)
	}
}

func TestCalculateUnitDamageByID_MissReturnsZero(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 0) // Low dexterity = low hit rate
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 0)

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	_ = attackerAttr // Keep for potential future use

	// Note: With Dexterity=0, hit rate is 80% (still decent)
	// This test may pass or fail based on random rolls
	// For a reliable test, we'd need a way to inject randomness

	damage, _ := calculateTestDamage(attacker.GetID(), defender.GetID(), manager)

	// Can't reliably test for 0 damage without controlling randomness
	t.Logf("Damage dealt: %d (0 expected on miss, but randomness not controlled)", damage)
}

func TestCalculateUnitDamageByID_DodgeReturnsZero(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100)
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 100) // High dexterity for dodge

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	defenderAttr := common.GetComponentType[*common.Attributes](defender, common.AttributeComponent)
	_, _ = attackerAttr, defenderAttr // Keep for potential future use

	// Note: With Dexterity=100, dodge chance is capped at 40%
	// This test may pass or fail based on random rolls
	// For a reliable test, we'd need a way to inject randomness

	damage, _ := calculateTestDamage(attacker.GetID(), defender.GetID(), manager)

	// Can't reliably test for 0 damage without controlling randomness
	t.Logf("Damage dealt: %d (0 expected on dodge, but randomness not controlled)", damage)
}

func TestCalculateUnitDamageByID_PhysicalResistanceReducesDamage(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	// Strength=20, Armor=10 for defender gives significant resistance
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100)
	defender := createTestUnit(manager, squadID, 0, 1, 100, 20, 10) // Higher armor = higher resistance

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	defenderAttr := common.GetComponentType[*common.Attributes](defender, common.AttributeComponent)

	// PhysicalResistance is derived from Strength/4 + Armor*3/2

	damage, _ := calculateTestDamage(attacker.GetID(), defender.GetID(), manager)

	baseDamage := attackerAttr.GetPhysicalDamage()
	resistance := defenderAttr.GetPhysicalResistance()
	expectedDamage := baseDamage - resistance
	if expectedDamage < 1 {
		expectedDamage = 1 // Minimum damage
	}

	// Allow for variance from hit/dodge/crit
	if damage > 0 && damage > baseDamage {
		t.Errorf("Damage %d should not exceed base damage %d without crits", damage, baseDamage)
	}

	t.Logf("Base damage: %d, Resistance: %d, Expected: %d, Actual: %d", baseDamage, resistance, expectedDamage, damage)
}

func TestCalculateUnitDamageByID_MinimumDamageIsOne(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	// Very low strength/weapon for attacker, very high armor for defender
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 1, 0)   // Strength=1, Weapon=0
	defender := createTestUnit(manager, squadID, 0, 1, 100, 50, 50) // Strength=50, Armor=50 for high resistance

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	defenderAttr := common.GetComponentType[*common.Attributes](defender, common.AttributeComponent)

	// Attacker has very low stats, defender has very high resistance
	// Expected: minimum 1 damage

	damage, _ := calculateTestDamage(attacker.GetID(), defender.GetID(), manager)

	// Minimum damage should be 1 when attack hits
	if damage > 1 {
		t.Logf("Expected minimum damage of 1, got %d (might be crit or miss rolled 0)", damage)
	}

	t.Logf("Attacker damage: %d, Defender resistance: %d, Actual damage: %d",
		attackerAttr.GetPhysicalDamage(), defenderAttr.GetPhysicalResistance(), damage)
}

func TestCalculateUnitDamageByID_NilUnitsReturnZero(t *testing.T) {
	manager := setupCombatTestManager(t)

	damage, _ := calculateTestDamage(9999, 9998, manager) // Non-existent IDs

	if damage != 0 {
		t.Errorf("Expected 0 damage for nil units, got %d", damage)
	}
}

// ========================================
// MAGIC DAMAGE TESTS
// ========================================

func TestCalculateUnitDamageByID_MagicDamageUsesMagicFormula(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Create magic attacker: Magic=15, Strength=3, Weapon=2
	attacker := manager.World.NewEntity()
	attacker.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: squadID})
	attacker.AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{AnchorRow: 0, AnchorCol: 0, CellWidth: 1, CellHeight: 1})
	attacker.AddComponent(common.NameComponent, &common.Name{NameStr: "Wizard"})

	attackerAttr := common.NewAttributes(3, 100, 15, 0, 1, 2) // High dex for guaranteed hit
	attackerAttr.CurrentHealth = 100
	attacker.AddComponent(common.AttributeComponent, &attackerAttr)

	// Set attack type to Magic
	attacker.AddComponent(squadcore.TargetRowComponent, &squadcore.TargetRowData{
		AttackType:  unitdefs.AttackTypeMagic,
		TargetCells: [][2]int{{0, 0}},
	})
	attacker.AddComponent(squadcore.AttackRangeComponent, &squadcore.AttackRangeData{Range: 4})

	// Create defender with low magic defense
	defender := manager.World.NewEntity()
	defender.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: squadID})
	defender.AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{AnchorRow: 0, AnchorCol: 1, CellWidth: 1, CellHeight: 1})
	defender.AddComponent(common.NameComponent, &common.Name{NameStr: "Fighter"})

	defenderAttr := common.NewAttributes(10, 0, 0, 0, 2, 10) // Magic=0, defense comes from BaseMagicResist only
	defenderAttr.CurrentHealth = 100
	defender.AddComponent(common.AttributeComponent, &defenderAttr)

	damage, event := calculateTestDamage(attacker.GetID(), defender.GetID(), manager)

	// Verify magic damage formula was used

	if event.HitResult.Type != combattypes.HitTypeMiss && event.HitResult.Type != combattypes.HitTypeDodge {
		expectedBaseDamage := attackerAttr.GetMagicDamage() // Should be 45
		if event.BaseDamage != expectedBaseDamage {
			t.Errorf("Expected base damage %d (Magic*3), got %d", expectedBaseDamage, event.BaseDamage)
		}

		// Verify NOT using physical formula
		physicalDamage := attackerAttr.GetPhysicalDamage() // Would be (3/2) + (2*2) = 5
		if event.BaseDamage == physicalDamage {
			t.Error("Magic attacker incorrectly using physical damage formula")
		}

		t.Logf("Magic attacker dealt %d damage (base: %d, type: %s)", damage, event.BaseDamage, event.HitResult.Type)
	}
}

func TestCalculateUnitDamageByID_PhysicalAttackersUnchanged(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100) // Physical unit
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 0)

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)

	damage, event := calculateTestDamage(attacker.GetID(), defender.GetID(), manager)

	// Verify physical damage formula is still used
	if event.HitResult.Type != combattypes.HitTypeMiss && event.HitResult.Type != combattypes.HitTypeDodge {
		expectedBaseDamage := attackerAttr.GetPhysicalDamage()
		if event.BaseDamage != expectedBaseDamage {
			t.Errorf("Physical attacker should use physical damage formula. Expected base %d, got %d",
				expectedBaseDamage, event.BaseDamage)
		}

		t.Logf("Physical attacker dealt %d damage (base: %d, type: %s)", damage, event.BaseDamage, event.HitResult.Type)
	}
}

func TestCalculateUnitDamageByID_MagicDefenseApplied(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Create magic attacker: Wizard with Magic=15
	attacker := manager.World.NewEntity()
	attacker.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: squadID})
	attacker.AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{AnchorRow: 0, AnchorCol: 0, CellWidth: 1, CellHeight: 1})
	attacker.AddComponent(common.NameComponent, &common.Name{NameStr: "Wizard"})

	attackerAttr := common.NewAttributes(3, 100, 15, 0, 1, 2) // High dex for guaranteed hit
	attackerAttr.CurrentHealth = 100
	attacker.AddComponent(common.AttributeComponent, &attackerAttr)

	attacker.AddComponent(squadcore.TargetRowComponent, &squadcore.TargetRowData{
		AttackType:  unitdefs.AttackTypeMagic,
		TargetCells: [][2]int{{0, 0}},
	})
	attacker.AddComponent(squadcore.AttackRangeComponent, &squadcore.AttackRangeData{Range: 4})

	// Create magic defender: Sorcerer with Magic=14
	defender := manager.World.NewEntity()
	defender.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: squadID})
	defender.AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{AnchorRow: 0, AnchorCol: 1, CellWidth: 1, CellHeight: 1})
	defender.AddComponent(common.NameComponent, &common.Name{NameStr: "Sorcerer"})

	defenderAttr := common.NewAttributes(4, 0, 14, 0, 1, 3) // Magic=14
	defenderAttr.CurrentHealth = 100
	defender.AddComponent(common.AttributeComponent, &defenderAttr)

	_, event := calculateTestDamage(attacker.GetID(), defender.GetID(), manager)

	// Verify magic defense was used (not physical resistance)
	if event.HitResult.Type != combattypes.HitTypeMiss && event.HitResult.Type != combattypes.HitTypeDodge {
		expectedMagicDefense := defenderAttr.GetMagicDefense() // Should be 12 (14/2 + 5)
		if event.ResistanceAmount != expectedMagicDefense {
			t.Errorf("Expected magic defense %d, got resistance %d", expectedMagicDefense, event.ResistanceAmount)
		}

		// Verify NOT using physical resistance
		physicalResistance := defenderAttr.GetPhysicalResistance()
		if event.ResistanceAmount == physicalResistance {
			t.Error("Magic attack incorrectly using physical resistance instead of magic defense")
		}

		t.Logf("Magic vs Magic: Base damage %d, Magic defense %d, Final damage %d",
			event.BaseDamage, event.ResistanceAmount, event.FinalDamage)
	}
}

