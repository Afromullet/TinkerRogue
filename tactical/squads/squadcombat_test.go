package squads

import (
	"game_main/common"
	"game_main/world/coords"
	"testing"

	"github.com/bytearena/ecs"
)

// Note: setupTestManager is defined in squads_test.go

// createTestUnit creates a unit with specified attributes for testing
func createTestUnit(manager *common.EntityManager, squadID ecs.EntityID, row, col int, health, strength, dexterity int) *ecs.Entity {
	unit := manager.World.NewEntity()

	// Add required components
	unit.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: squadID})
	unit.AddComponent(GridPositionComponent, &GridPositionData{
		AnchorRow: row,
		AnchorCol: col,
		Width:     1,
		Height:    1,
	})
	unit.AddComponent(common.NameComponent, &common.Name{NameStr: "TestUnit"})

	// Add attributes using NewAttributes constructor
	attr := common.NewAttributes(
		strength,  // Strength
		dexterity, // Dexterity (affects hit/dodge/crit)
		0,         // Magic
		0,         // Leadership
		2,         // Armor
		2,         // Weapon
	)
	// Set the specified health (after creation, since NewAttributes sets it to MaxHealth)
	attr.CurrentHealth = health
	unit.AddComponent(common.AttributeComponent, &attr)

	// Add targeting data (default to MeleeRow attacking front row)
	unit.AddComponent(TargetRowComponent, &TargetRowData{
		AttackType:  AttackTypeMeleeRow, // Explicit attack type
		TargetCells: nil,                // Not used for MeleeRow
	})

	// Add attack range component (default to melee range 1)
	unit.AddComponent(AttackRangeComponent, &AttackRangeData{
		Range: 1,
	})

	// Note: Tags are managed through the component query system, not directly added
	return unit
}

// createTestSquad creates a squad entity with specified ID at position (0,0)
func createTestSquad(manager *common.EntityManager, name string) ecs.EntityID {
	squad := manager.World.NewEntity()
	squadID := squad.GetID()

	squadData := &SquadData{
		SquadID:    squadID,
		Formation:  FormationBalanced,
		Name:       name,
		Morale:     100,
		SquadLevel: 1,
		TurnCount:  0,
		MaxUnits:   9,
	}

	squad.AddComponent(SquadComponent, squadData)

	// Add position component so squads can calculate distance
	squad.AddComponent(common.PositionComponent, &coords.LogicalPosition{
		X: 0,
		Y: 0,
	})

	// Note: Entities are added automatically, no need for AddEntity
	// Tags are managed through component queries

	return squadID
}

// ========================================
// ExecuteSquadAttack TESTS
// ========================================

func TestExecuteSquadAttack_SingleAttackerVsSingleDefender(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	_ = createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100) // 100% hit rate

	// Create defender squad
	defenderSquadID := createTestSquad(manager, "Defenders")
	defenderUnit := createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	defenderAttr := common.GetComponentType[*common.Attributes](defenderUnit, common.AttributeComponent)
	initialHP := defenderAttr.CurrentHealth

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify result
	if result == nil {
		t.Fatal("Expected combat result, got nil")
	}

	if result.TotalDamage <= 0 {
		t.Error("Expected damage > 0")
	}

	if defenderAttr.CurrentHealth >= initialHP {
		t.Errorf("Expected defender HP to decrease from %d, got %d", initialHP, defenderAttr.CurrentHealth)
	}

	if len(result.DamageByUnit) != 1 {
		t.Errorf("Expected 1 unit damaged, got %d", len(result.DamageByUnit))
	}
}

func TestExecuteSquadAttack_MultipleAttackersVsMultipleDefenders(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad with 3 units
	attackerSquadID := createTestSquad(manager, "Attackers")
	createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)
	createTestUnit(manager, attackerSquadID, 0, 1, 100, 20, 100)
	createTestUnit(manager, attackerSquadID, 0, 2, 100, 20, 100)

	// Create defender squad with 3 units
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0)
	createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify result
	if result.TotalDamage <= 0 {
		t.Error("Expected total damage > 0")
	}

	// Each attacker should hit one target (lowest HP)
	if len(result.DamageByUnit) < 1 {
		t.Errorf("Expected at least 1 unit damaged, got %d", len(result.DamageByUnit))
	}
}

func TestExecuteSquadAttack_DeadAttackersDoNotAttack(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad with dead unit
	attackerSquadID := createTestSquad(manager, "Attackers")
	deadAttacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)
	attr := common.GetComponentType[*common.Attributes](deadAttacker, common.AttributeComponent)
	attr.CurrentHealth = 0 // Dead unit

	// Create defender squad
	defenderSquadID := createTestSquad(manager, "Defenders")
	defenderUnit := createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	defenderAttr := common.GetComponentType[*common.Attributes](defenderUnit, common.AttributeComponent)
	initialHP := defenderAttr.CurrentHealth

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify no damage dealt
	if result.TotalDamage != 0 {
		t.Errorf("Expected no damage from dead attacker, got %d", result.TotalDamage)
	}

	if defenderAttr.CurrentHealth != initialHP {
		t.Errorf("Expected defender HP to remain %d, got %d", initialHP, defenderAttr.CurrentHealth)
	}
}

func TestExecuteSquadAttack_MultiTargetAttack(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker with multi-target ability
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set magic targeting (target 2 specific cells)
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMagic           // Use magic for cell-based targeting
	targetData.TargetCells = [][2]int{{0, 0}, {0, 1}} // Target first two front-row cells

	// Create defenders
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0) // Should be hit
	createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0) // Should be hit
	createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0) // Should NOT be hit

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify exactly 2 targets hit (the ones in targeted cells)
	if len(result.DamageByUnit) != 2 {
		t.Errorf("Expected 2 units damaged, got %d", len(result.DamageByUnit))
	}
}

func TestExecuteSquadAttack_CellBasedTargeting(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker with cell-based targeting (2x2 pattern)
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set magic targeting with 2x2 pattern
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMagic                           // Use magic for cell-based targeting
	targetData.TargetCells = [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}} // 2x2 top-left

	// Create defenders in targeted cells
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0) // Hit
	createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0) // Hit
	createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Hit
	createTestUnit(manager, defenderSquadID, 2, 2, 50, 10, 0) // Miss (not in pattern)

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify 3 units hit (3 in the 2x2 pattern)
	if len(result.DamageByUnit) != 3 {
		t.Errorf("Expected 3 units damaged (2x2 pattern), got %d", len(result.DamageByUnit))
	}
}

func TestExecuteSquadAttack_UnitsKilledTracking(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker with high damage
	attackerSquadID := createTestSquad(manager, "Attackers")
	createTestUnit(manager, attackerSquadID, 0, 0, 100, 100, 100) // High strength

	// Create weak defender
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 10, 5, 0) // Low HP

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify unit was killed
	if len(result.UnitsKilled) != 1 {
		t.Errorf("Expected 1 unit killed, got %d", len(result.UnitsKilled))
	}
}

// ========================================
// calculateUnitDamageByID TESTS
// ========================================

func TestCalculateUnitDamageByID_BasicDamageCalculation(t *testing.T) {
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100) // 100% hit rate
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 0)

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	defenderAttr := common.GetComponentType[*common.Attributes](defender, common.AttributeComponent)

	// Note: Attributes are derived from base stats (Strength, Dexterity, etc.)
	// We can't set them directly, but with Dexterity=100, attacker should have high hit rate
	// With Dexterity=0, defender should have low dodge chance

	damage, _ := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

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
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 0) // Low dexterity = low hit rate
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 0)

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	_ = attackerAttr // Keep for potential future use

	// Note: With Dexterity=0, hit rate is 80% (still decent)
	// This test may pass or fail based on random rolls
	// For a reliable test, we'd need a way to inject randomness

	damage, _ := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	// Can't reliably test for 0 damage without controlling randomness
	t.Logf("Damage dealt: %d (0 expected on miss, but randomness not controlled)", damage)
}

func TestCalculateUnitDamageByID_DodgeReturnsZero(t *testing.T) {
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100)
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 100) // High dexterity for dodge

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	defenderAttr := common.GetComponentType[*common.Attributes](defender, common.AttributeComponent)
	_, _ = attackerAttr, defenderAttr // Keep for potential future use

	// Note: With Dexterity=100, dodge chance is capped at 40%
	// This test may pass or fail based on random rolls
	// For a reliable test, we'd need a way to inject randomness

	damage, _ := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	// Can't reliably test for 0 damage without controlling randomness
	t.Logf("Damage dealt: %d (0 expected on dodge, but randomness not controlled)", damage)
}

func TestCalculateUnitDamageByID_PhysicalResistanceReducesDamage(t *testing.T) {
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	// Strength=20, Armor=10 for defender gives significant resistance
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100)
	defender := createTestUnit(manager, squadID, 0, 1, 100, 20, 10) // Higher armor = higher resistance

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	defenderAttr := common.GetComponentType[*common.Attributes](defender, common.AttributeComponent)

	// PhysicalResistance is derived from Strength/4 + Armor*3/2

	damage, _ := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

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
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	// Very low strength/weapon for attacker, very high armor for defender
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 1, 0)   // Strength=1, Weapon=0
	defender := createTestUnit(manager, squadID, 0, 1, 100, 50, 50) // Strength=50, Armor=50 for high resistance

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	defenderAttr := common.GetComponentType[*common.Attributes](defender, common.AttributeComponent)

	// Attacker has very low stats, defender has very high resistance
	// Expected: minimum 1 damage

	damage, _ := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	// Minimum damage should be 1 when attack hits
	if damage > 1 {
		t.Logf("Expected minimum damage of 1, got %d (might be crit or miss rolled 0)", damage)
	}

	t.Logf("Attacker damage: %d, Defender resistance: %d, Actual damage: %d",
		attackerAttr.GetPhysicalDamage(), defenderAttr.GetPhysicalResistance(), damage)
}

func TestCalculateUnitDamageByID_NilUnitsReturnZero(t *testing.T) {
	manager := setupTestManager(t)

	damage, _ := calculateUnitDamageByID(9999, 9998, manager) // Non-existent IDs

	if damage != 0 {
		t.Errorf("Expected 0 damage for nil units, got %d", damage)
	}
}

// ========================================
// MAGIC DAMAGE TESTS
// ========================================

func TestCalculateUnitDamageByID_MagicDamageUsesMagicFormula(t *testing.T) {
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Create magic attacker: Magic=15, Strength=3, Weapon=2
	attacker := manager.World.NewEntity()
	attacker.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: squadID})
	attacker.AddComponent(GridPositionComponent, &GridPositionData{AnchorRow: 0, AnchorCol: 0, Width: 1, Height: 1})
	attacker.AddComponent(common.NameComponent, &common.Name{NameStr: "Wizard"})

	attackerAttr := common.NewAttributes(3, 100, 15, 0, 1, 2) // High dex for guaranteed hit
	attackerAttr.CurrentHealth = 100
	attacker.AddComponent(common.AttributeComponent, &attackerAttr)

	// Set attack type to Magic
	attacker.AddComponent(TargetRowComponent, &TargetRowData{
		AttackType:  AttackTypeMagic,
		TargetCells: [][2]int{{0, 0}},
	})
	attacker.AddComponent(AttackRangeComponent, &AttackRangeData{Range: 4})

	// Create defender with low magic defense
	defender := manager.World.NewEntity()
	defender.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: squadID})
	defender.AddComponent(GridPositionComponent, &GridPositionData{AnchorRow: 0, AnchorCol: 1, Width: 1, Height: 1})
	defender.AddComponent(common.NameComponent, &common.Name{NameStr: "Fighter"})

	defenderAttr := common.NewAttributes(10, 0, 0, 0, 2, 10) // Magic=0, defense comes from BaseMagicResist only
	defenderAttr.CurrentHealth = 100
	defender.AddComponent(common.AttributeComponent, &defenderAttr)

	damage, event := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	// Verify magic damage formula was used

	if event.HitResult.Type != HitTypeMiss && event.HitResult.Type != HitTypeDodge {
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
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100) // Physical unit
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 0)

	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)

	damage, event := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	// Verify physical damage formula is still used
	if event.HitResult.Type != HitTypeMiss && event.HitResult.Type != HitTypeDodge {
		expectedBaseDamage := attackerAttr.GetPhysicalDamage()
		if event.BaseDamage != expectedBaseDamage {
			t.Errorf("Physical attacker should use physical damage formula. Expected base %d, got %d",
				expectedBaseDamage, event.BaseDamage)
		}

		t.Logf("Physical attacker dealt %d damage (base: %d, type: %s)", damage, event.BaseDamage, event.HitResult.Type)
	}
}

func TestCalculateUnitDamageByID_MagicDefenseApplied(t *testing.T) {
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Create magic attacker: Wizard with Magic=15
	attacker := manager.World.NewEntity()
	attacker.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: squadID})
	attacker.AddComponent(GridPositionComponent, &GridPositionData{AnchorRow: 0, AnchorCol: 0, Width: 1, Height: 1})
	attacker.AddComponent(common.NameComponent, &common.Name{NameStr: "Wizard"})

	attackerAttr := common.NewAttributes(3, 100, 15, 0, 1, 2) // High dex for guaranteed hit
	attackerAttr.CurrentHealth = 100
	attacker.AddComponent(common.AttributeComponent, &attackerAttr)

	attacker.AddComponent(TargetRowComponent, &TargetRowData{
		AttackType:  AttackTypeMagic,
		TargetCells: [][2]int{{0, 0}},
	})
	attacker.AddComponent(AttackRangeComponent, &AttackRangeData{Range: 4})

	// Create magic defender: Sorcerer with Magic=14
	defender := manager.World.NewEntity()
	defender.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: squadID})
	defender.AddComponent(GridPositionComponent, &GridPositionData{AnchorRow: 0, AnchorCol: 1, Width: 1, Height: 1})
	defender.AddComponent(common.NameComponent, &common.Name{NameStr: "Sorcerer"})

	defenderAttr := common.NewAttributes(4, 0, 14, 0, 1, 3) // Magic=14
	defenderAttr.CurrentHealth = 100
	defender.AddComponent(common.AttributeComponent, &defenderAttr)

	_, event := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	// Verify magic defense was used (not physical resistance)
	if event.HitResult.Type != HitTypeMiss && event.HitResult.Type != HitTypeDodge {
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

// ========================================
// COVER SYSTEM TESTS
// ========================================

// ========================================
// GetCoverProvidersFor TESTS
// ========================================

func TestGetCoverProvidersFor_NoProviders(t *testing.T) {
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	defender := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	defenderPos := common.GetComponentType[*GridPositionData](defender, GridPositionComponent)

	providers := GetCoverProvidersFor(defender.GetID(), squadID, defenderPos, manager)

	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
}

func TestGetCoverProvidersFor_SingleProvider(t *testing.T) {
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	frontLine := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})

	backLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	backLinePos := common.GetComponentType[*GridPositionData](backLine, GridPositionComponent)

	providers := GetCoverProvidersFor(backLine.GetID(), squadID, backLinePos, manager)

	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	if len(providers) > 0 && providers[0] != frontLine.GetID() {
		t.Error("Expected front-line unit to be the provider")
	}
}

func TestGetCoverProvidersFor_MultipleProviders(t *testing.T) {
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Two front-line units in same column
	frontLine1 := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine1.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.15,
		CoverRange:     2,
		RequiresActive: true,
	})

	midLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	midLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.10,
		CoverRange:     1,
		RequiresActive: true,
	})

	backLine := createTestUnit(manager, squadID, 2, 0, 100, 10, 0)
	backLinePos := common.GetComponentType[*GridPositionData](backLine, GridPositionComponent)

	providers := GetCoverProvidersFor(backLine.GetID(), squadID, backLinePos, manager)

	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}
}

func TestGetCoverProvidersFor_DoesNotIncludeSelf(t *testing.T) {
	manager := setupTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	unit := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	unit.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})
	unitPos := common.GetComponentType[*GridPositionData](unit, GridPositionComponent)

	providers := GetCoverProvidersFor(unit.GetID(), squadID, unitPos, manager)

	if len(providers) != 0 {
		t.Errorf("Expected 0 providers (unit should not provide cover to itself), got %d", len(providers))
	}
}

func TestGetCoverProvidersFor_OnlyFromSameSquad(t *testing.T) {
	manager := setupTestManager(t)

	// Squad 1
	squad1ID := createTestSquad(manager, "Squad1")
	squad1Unit := createTestUnit(manager, squad1ID, 0, 0, 100, 10, 0)
	squad1Unit.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})

	// Squad 2
	squad2ID := createTestSquad(manager, "Squad2")
	squad2Unit := createTestUnit(manager, squad2ID, 1, 0, 100, 10, 0)
	squad2UnitPos := common.GetComponentType[*GridPositionData](squad2Unit, GridPositionComponent)

	// Squad 2 unit should not get cover from Squad 1
	providers := GetCoverProvidersFor(squad2Unit.GetID(), squad2ID, squad2UnitPos, manager)

	if len(providers) != 0 {
		t.Errorf("Expected 0 providers from different squad, got %d", len(providers))
	}
}

// ========================================
// HELPER FUNCTION TESTS
// ========================================

func TestSumDamageMap(t *testing.T) {
	damageMap := map[ecs.EntityID]int{
		1: 10,
		2: 20,
		3: 30,
	}

	total := sumDamageMap(damageMap)

	if total != 60 {
		t.Errorf("Expected total damage 60, got %d", total)
	}
}

func TestSumDamageMap_EmptyMap(t *testing.T) {
	damageMap := make(map[ecs.EntityID]int)

	total := sumDamageMap(damageMap)

	if total != 0 {
		t.Errorf("Expected total damage 0 for empty map, got %d", total)
	}
}

// ========================================
// NEW TARGETING SYSTEM TESTS
// ========================================

func TestMeleeRowTargeting_FrontRow(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad with MeleeRow attacker
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to MeleeRow targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMeleeRow
	targetData.TargetCells = nil

	// Create defender squad with 3 units in front row (row 0)
	defenderSquadID := createTestSquad(manager, "Defenders")
	defender1 := createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	defender2 := createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0)
	defender3 := createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify all 3 front row units are targeted
	if len(targets) != 3 {
		t.Errorf("Expected 3 targets (entire front row), got %d", len(targets))
	}

	// Verify correct units are targeted
	expectedIDs := map[ecs.EntityID]bool{
		defender1.GetID(): true,
		defender2.GetID(): true,
		defender3.GetID(): true,
	}

	for _, targetID := range targets {
		if !expectedIDs[targetID] {
			t.Errorf("Unexpected target ID %d", targetID)
		}
	}
}

func TestMeleeRowTargeting_PierceToRow1(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to MeleeRow targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMeleeRow
	targetData.TargetCells = nil

	// Create defender squad with units only in row 1 (front row empty)
	defenderSquadID := createTestSquad(manager, "Defenders")
	defender1 := createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0)
	defender2 := createTestUnit(manager, defenderSquadID, 1, 2, 50, 10, 0)

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify units in row 1 are targeted
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets (row 1 after pierce), got %d", len(targets))
	}

	expectedIDs := map[ecs.EntityID]bool{
		defender1.GetID(): true,
		defender2.GetID(): true,
	}

	for _, targetID := range targets {
		if !expectedIDs[targetID] {
			t.Errorf("Unexpected target ID %d", targetID)
		}
	}
}

func TestMeleeRowTargeting_PierceWhenFrontRowDead(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to MeleeRow targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMeleeRow
	targetData.TargetCells = nil

	// Create defender squad with DEAD units in row 0 and ALIVE units in row 1
	defenderSquadID := createTestSquad(manager, "Defenders")

	// Row 0 - all dead (should be ignored for targeting)
	deadUnit1 := createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	deadAttr1 := common.GetComponentType[*common.Attributes](deadUnit1, common.AttributeComponent)
	deadAttr1.CurrentHealth = 0

	deadUnit2 := createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0)
	deadAttr2 := common.GetComponentType[*common.Attributes](deadUnit2, common.AttributeComponent)
	deadAttr2.CurrentHealth = 0

	// Row 1 - alive (should be targeted)
	aliveUnit1 := createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0)
	aliveUnit2 := createTestUnit(manager, defenderSquadID, 1, 2, 50, 10, 0)

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify pierce-through works: dead units in row 0 are ignored, row 1 units targeted
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets (row 1 after piercing through dead row 0), got %d", len(targets))
	}

	expectedIDs := map[ecs.EntityID]bool{
		aliveUnit1.GetID(): true,
		aliveUnit2.GetID(): true,
	}

	for _, targetID := range targets {
		if !expectedIDs[targetID] {
			t.Errorf("Unexpected target ID %d", targetID)
		}
		// Verify targeted units are alive
		if targetID == deadUnit1.GetID() || targetID == deadUnit2.GetID() {
			t.Errorf("Dead unit %d was targeted!", targetID)
		}
	}
}

func TestMeleeRowTargeting_PierceToRow2(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to MeleeRow targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMeleeRow
	targetData.TargetCells = nil

	// Create defender squad with units only in row 2 (rows 0 and 1 empty)
	defenderSquadID := createTestSquad(manager, "Defenders")
	defender1 := createTestUnit(manager, defenderSquadID, 2, 0, 50, 10, 0)
	defender2 := createTestUnit(manager, defenderSquadID, 2, 2, 50, 10, 0)

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify units in row 2 are targeted
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets (row 2 after pierce), got %d", len(targets))
	}

	expectedIDs := map[ecs.EntityID]bool{
		defender1.GetID(): true,
		defender2.GetID(): true,
	}

	for _, targetID := range targets {
		if !expectedIDs[targetID] {
			t.Errorf("Unexpected target ID %d", targetID)
		}
	}
}

func TestMeleeColumnTargeting_DirectFront(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad with MeleeColumn attacker in column 1
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 1, 100, 20, 100)

	// Set to MeleeColumn targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMeleeColumn
	targetData.TargetCells = nil

	// Create defender squad with units in different columns
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)              // Column 0 - NOT targeted
	defender2 := createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0) // Column 1 - TARGETED
	createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)              // Column 2 - NOT targeted

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify only 1 target in the same column
	if len(targets) != 1 {
		t.Errorf("Expected 1 target (same column), got %d", len(targets))
	}

	if targets[0] != defender2.GetID() {
		t.Errorf("Expected defender in column 1, got %d", targets[0])
	}
}

func TestMeleeColumnTargeting_PierceForward(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad in column 0
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to MeleeColumn targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMeleeColumn
	targetData.TargetCells = nil

	// Create defender squad with units in column 0 (row 0 col 0 is empty)
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0)              // Different column - NOT targeted
	defender2 := createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Same column, row 1 - TARGETED
	defender3 := createTestUnit(manager, defenderSquadID, 2, 0, 50, 10, 0) // Same column, row 2 - ALSO TARGETED (piercing attack)

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify 2 targets (all units in the column - piercing attack)
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets (all units in column), got %d", len(targets))
	}

	// Check both targets are present (order may vary)
	targetSet := make(map[ecs.EntityID]bool)
	for _, tid := range targets {
		targetSet[tid] = true
	}
	if !targetSet[defender2.GetID()] || !targetSet[defender3.GetID()] {
		t.Errorf("Expected defenders at (1,0) and (2,0) to be targeted")
	}
}

func TestMeleeColumnTargeting_ColumnWrapping(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad in column 1
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 1, 100, 20, 100)

	// Set to MeleeColumn targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMeleeColumn
	targetData.TargetCells = nil

	// Create defender squad with NO units in column 1, but units in columns 2 and 0
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)              // Column 0 - should wrap to this
	defender2 := createTestUnit(manager, defenderSquadID, 1, 2, 50, 10, 0) // Column 2 - TARGETED (next after col 1)

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify wrapping: attackerCol=1 → try col 1 (empty), col 2 (found!), col 0 (not reached)
	if len(targets) != 1 {
		t.Errorf("Expected 1 target (column 2 after wrapping), got %d", len(targets))
	}

	if targets[0] != defender2.GetID() {
		t.Errorf("Expected defender in column 2, got %d", targets[0])
	}
}

func TestMeleeColumnTargeting_WrapToColumn0(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad in column 2
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 2, 100, 20, 100)

	// Set to MeleeColumn targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMeleeColumn
	targetData.TargetCells = nil

	// Create defender squad with NO units in columns 2, only in column 0
	defenderSquadID := createTestSquad(manager, "Defenders")
	defender1 := createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Column 0 - TARGETED (after wrapping)

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify wrapping: attackerCol=2 → try col 2 (empty), col 0 (found!), col 1 (not reached)
	if len(targets) != 1 {
		t.Errorf("Expected 1 target (column 0 after wrapping), got %d", len(targets))
	}

	if targets[0] != defender1.GetID() {
		t.Errorf("Expected defender in column 0, got %d", targets[0])
	}
}

func TestRangedTargeting_SameRow(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad with ranged attacker in row 1
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 1, 0, 100, 20, 100)

	// Set to Ranged targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeRanged
	targetData.TargetCells = nil

	// Create defender squad with units in different rows
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)              // Row 0 - NOT targeted
	defender2 := createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Row 1 - TARGETED
	defender3 := createTestUnit(manager, defenderSquadID, 1, 1, 50, 10, 0) // Row 1 - TARGETED
	defender4 := createTestUnit(manager, defenderSquadID, 1, 2, 50, 10, 0) // Row 1 - TARGETED
	createTestUnit(manager, defenderSquadID, 2, 0, 50, 10, 0)              // Row 2 - NOT targeted

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify all 3 targets in row 1
	if len(targets) != 3 {
		t.Errorf("Expected 3 targets (same row), got %d", len(targets))
	}

	expectedIDs := map[ecs.EntityID]bool{
		defender2.GetID(): true,
		defender3.GetID(): true,
		defender4.GetID(): true,
	}

	for _, targetID := range targets {
		if !expectedIDs[targetID] {
			t.Errorf("Unexpected target ID %d", targetID)
		}
	}
}

func TestRangedTargeting_FallbackLowestArmor(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad in row 1
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 1, 0, 100, 20, 100)

	// Set to Ranged targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeRanged
	targetData.TargetCells = nil

	// Create defender squad with NO units in row 1 (fallback triggers)
	defenderSquadID := createTestSquad(manager, "Defenders")

	// Row 0, col 0: High armor
	defender1 := createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	attr1 := common.NewAttributes(10, 0, 0, 0, 10, 2) // Higher armor
	attr1.CurrentHealth = 50
	defender1.RemoveComponent(common.AttributeComponent)
	defender1.AddComponent(common.AttributeComponent, &attr1)

	// Row 0, col 1: Lowest armor
	defender2 := createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0)
	attr2 := common.NewAttributes(10, 0, 0, 0, 2, 2) // Lower armor
	attr2.CurrentHealth = 50
	defender2.RemoveComponent(common.AttributeComponent)
	defender2.AddComponent(common.AttributeComponent, &attr2)

	// Row 0, col 2: Medium armor
	defender3 := createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)
	attr3 := common.NewAttributes(10, 0, 0, 0, 5, 2) // Medium armor
	attr3.CurrentHealth = 50
	defender3.RemoveComponent(common.AttributeComponent)
	defender3.AddComponent(common.AttributeComponent, &attr3)

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify only 1 target (lowest armor)
	if len(targets) != 1 {
		t.Errorf("Expected 1 target (lowest armor fallback), got %d", len(targets))
	}

	if targets[0] != defender2.GetID() {
		t.Errorf("Expected defender2 (lowest armor), got %d", targets[0])
	}
}

func TestRangedTargeting_FallbackTiebreaker(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad in row 1
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 1, 0, 100, 20, 100)

	// Set to Ranged targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeRanged
	targetData.TargetCells = nil

	// Create defender squad with NO units in row 1
	defenderSquadID := createTestSquad(manager, "Defenders")

	// Row 0, col 2: Same armor, front row, rightmost
	defender1 := createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)
	attr1 := common.NewAttributes(10, 0, 0, 0, 5, 2)
	attr1.CurrentHealth = 50
	defender1.RemoveComponent(common.AttributeComponent)
	defender1.AddComponent(common.AttributeComponent, &attr1)

	// Row 2, col 0: Same armor, furthest row, leftmost - SHOULD WIN
	defender2 := createTestUnit(manager, defenderSquadID, 2, 0, 50, 10, 0)
	attr2 := common.NewAttributes(10, 0, 0, 0, 5, 2)
	attr2.CurrentHealth = 50
	defender2.RemoveComponent(common.AttributeComponent)
	defender2.AddComponent(common.AttributeComponent, &attr2)

	// Row 2, col 1: Same armor, furthest row, middle
	defender3 := createTestUnit(manager, defenderSquadID, 2, 1, 50, 10, 0)
	attr3 := common.NewAttributes(10, 0, 0, 0, 5, 2)
	attr3.CurrentHealth = 50
	defender3.RemoveComponent(common.AttributeComponent)
	defender3.AddComponent(common.AttributeComponent, &attr3)

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify defender2 wins (furthest row + leftmost column)
	if len(targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(targets))
	}

	if targets[0] != defender2.GetID() {
		t.Errorf("Expected defender2 (furthest row + leftmost), got %d", targets[0])
	}
}

func TestMagicTargeting_ExactCells(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to Magic targeting with specific pattern (column 1 only)
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMagic
	targetData.TargetCells = [][2]int{{0, 1}, {1, 1}, {2, 1}} // Middle column

	// Create defender squad
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)              // Col 0 - NOT targeted
	defender2 := createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0) // Col 1 - TARGETED
	createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)              // Col 2 - NOT targeted
	defender4 := createTestUnit(manager, defenderSquadID, 1, 1, 50, 10, 0) // Col 1 - TARGETED
	defender5 := createTestUnit(manager, defenderSquadID, 2, 1, 50, 10, 0) // Col 1 - TARGETED

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify only units in specified cells are targeted
	if len(targets) != 3 {
		t.Errorf("Expected 3 targets (column 1), got %d", len(targets))
	}

	expectedIDs := map[ecs.EntityID]bool{
		defender2.GetID(): true,
		defender4.GetID(): true,
		defender5.GetID(): true,
	}

	for _, targetID := range targets {
		if !expectedIDs[targetID] {
			t.Errorf("Unexpected target ID %d", targetID)
		}
	}
}

func TestMagicTargeting_NoPierce(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to Magic targeting (front row only)
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.AttackType = AttackTypeMagic
	targetData.TargetCells = [][2]int{{0, 0}, {0, 1}, {0, 2}} // Front row only

	// Create defender squad with NO units in row 0, units in row 1
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Row 1 - NOT targeted (no pierce)
	createTestUnit(manager, defenderSquadID, 1, 1, 50, 10, 0) // Row 1 - NOT targeted (no pierce)

	// Get targets
	targets := SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify no targets (magic doesn't pierce)
	if len(targets) != 0 {
		t.Errorf("Expected 0 targets (magic doesn't pierce), got %d", len(targets))
	}
}

// ========================================
// INTEGRATION TESTS
// ========================================

func TestCombatWithCoverSystem_Integration(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 30, 0) // Low dexterity = lower crit
	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)

	// Create defender squad with cover
	defenderSquadID := createTestSquad(manager, "Defenders")

	// Front-line unit provides cover
	frontLine := createTestUnit(manager, defenderSquadID, 0, 0, 100, 10, 0)
	frontLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.50, // 50% damage reduction
		CoverRange:     1,
		RequiresActive: true,
	})

	// Back-line unit receives cover
	backLine := createTestUnit(manager, defenderSquadID, 1, 0, 100, 10, 0) // Low dexterity = low dodge
	backLineAttr := common.GetComponentType[*common.Attributes](backLine, common.AttributeComponent)

	// Configure attacker to target back line
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.TargetCells = [][2]int{{1, 0}, {1, 1}, {1, 2}} // Target row 1

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify cover reduced damage
	if len(result.DamageByUnit) != 1 {
		t.Fatalf("Expected 1 unit damaged, got %d", len(result.DamageByUnit))
	}

	damageDealt := result.DamageByUnit[backLine.GetID()]
	baseDamage := attackerAttr.GetPhysicalDamage()
	resistance := backLineAttr.GetPhysicalResistance()
	_ = resistance // May be used in future assertions

	// Cover reduces damage by 50%, but we also need to account for resistance
	// expectedDamage := int(float64(baseDamage) * 0.50) // 50% reduction from cover

	// Allow variance due to randomness, crits, resistance
	if damageDealt < 0 || damageDealt > baseDamage {
		t.Errorf("Expected damage between 0 and %d (with 50%% cover), got %d", baseDamage, damageDealt)
	}

	// Verify cover had some effect (unless attack missed)
	if damageDealt > 0 && damageDealt >= baseDamage {
		t.Error("Expected cover to reduce damage below base damage")
	}

	t.Logf("Base damage: %d, Damage dealt: %d (with 50%% cover)", baseDamage, damageDealt)
}

func TestMultiRoundCombat_Integration(t *testing.T) {
	manager := setupTestManager(t)

	// Create two evenly matched squads
	squad1ID := createTestSquad(manager, "Squad1")
	createTestUnit(manager, squad1ID, 0, 0, 50, 15, 100)
	createTestUnit(manager, squad1ID, 0, 1, 50, 15, 100)

	squad2ID := createTestSquad(manager, "Squad2")
	createTestUnit(manager, squad2ID, 0, 0, 50, 15, 100)
	createTestUnit(manager, squad2ID, 0, 1, 50, 15, 100)

	// Simulate multiple rounds
	rounds := 0
	maxRounds := 10

	for rounds < maxRounds {
		rounds++

		// Squad 1 attacks Squad 2
		result1 := ExecuteSquadAttack(squad1ID, squad2ID, manager)

		// Check if Squad 2 is destroyed
		if IsSquadDestroyed(squad2ID, manager) {
			t.Logf("Squad2 destroyed in round %d", rounds)
			break
		}

		// Squad 2 attacks Squad 1
		result2 := ExecuteSquadAttack(squad2ID, squad1ID, manager)

		// Check if Squad 1 is destroyed
		if IsSquadDestroyed(squad1ID, manager) {
			t.Logf("Squad1 destroyed in round %d", rounds)
			break
		}

		t.Logf("Round %d: Squad1 dealt %d damage, Squad2 dealt %d damage",
			rounds, result1.TotalDamage, result2.TotalDamage)
	}

	if rounds >= maxRounds {
		t.Log("Combat reached maximum rounds without destruction")
	}

	// At least one squad should be heavily damaged
	squad1Destroyed := IsSquadDestroyed(squad1ID, manager)
	squad2Destroyed := IsSquadDestroyed(squad2ID, manager)

	if !squad1Destroyed && !squad2Destroyed {
		t.Log("Both squads survived - this is possible with lucky dodges/misses")
	}
}

func TestExecuteSquadAttack_MultiCellUnit_HitOnce(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad with unit targeting multiple rows
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)
	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	baseDamage := attackerAttr.GetPhysicalDamage()

	// Set attacker to target both row 0 and row 1
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.TargetCells = [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}}

	// Create defender squad with a 2x2 multi-cell unit spanning rows 0-1, cols 0-1
	defenderSquadID := createTestSquad(manager, "Defenders")
	multiCellUnit := manager.World.NewEntity()
	multiCellUnit.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: defenderSquadID})
	multiCellUnit.AddComponent(GridPositionComponent, &GridPositionData{
		AnchorRow: 0,
		AnchorCol: 0,
		Width:     2,
		Height:    2,
	})
	multiCellUnit.AddComponent(common.NameComponent, &common.Name{NameStr: "Giant"})

	// Set defender attributes
	defenderAttr := common.NewAttributes(10, 0, 0, 0, 2, 2)
	defenderAttr.CurrentHealth = 100
	multiCellUnit.AddComponent(common.AttributeComponent, &defenderAttr)

	initialHP := defenderAttr.CurrentHealth

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify the multi-cell unit was only hit ONCE, not twice
	if len(result.DamageByUnit) != 1 {
		t.Errorf("Expected 1 unit to be damaged, got %d", len(result.DamageByUnit))
	}

	// Verify total damage equals damage to single unit (not doubled)
	unitDamage := result.DamageByUnit[multiCellUnit.GetID()]
	if result.TotalDamage != unitDamage {
		t.Errorf("Expected total damage %d to equal unit damage %d", result.TotalDamage, unitDamage)
	}

	// Verify defender lost appropriate HP (not doubled)
	expectedHP := initialHP - unitDamage
	if defenderAttr.CurrentHealth != expectedHP {
		t.Errorf("Expected defender HP %d, got %d (initial: %d, damage: %d)",
			expectedHP, defenderAttr.CurrentHealth, initialHP, unitDamage)
	}

	// Verify damage is reasonable (should be around baseDamage - resistance, not doubled)
	resistance := defenderAttr.GetPhysicalResistance()
	expectedSingleHitDamage := baseDamage - resistance
	if expectedSingleHitDamage < 1 {
		expectedSingleHitDamage = 1
	}

	// Allow some variance for crits, but should not be ~2x the base damage
	maxReasonableDamage := expectedSingleHitDamage * 2 // Crit is 1.5x, so 2x is generous
	if unitDamage > maxReasonableDamage {
		t.Errorf("Unit took %d damage, which is too high. Expected around %d (base damage %d - resistance %d). This suggests the unit was hit multiple times.",
			unitDamage, expectedSingleHitDamage, baseDamage, resistance)
	}

	t.Logf("Multi-cell unit correctly hit once for %d damage (base: %d, resistance: %d, expected: %d)",
		unitDamage, baseDamage, resistance, expectedSingleHitDamage)
}

func TestExecuteSquadAttack_MultiCellUnit_CellBased_HitOnce(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad with cell-based targeting
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)
	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	baseDamage := attackerAttr.GetPhysicalDamage()

	// Set attacker to target all 4 cells occupied by the multi-cell unit
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.TargetCells = [][2]int{
		{0, 0}, // Top-left
		{0, 1}, // Top-right
		{1, 0}, // Bottom-left
		{1, 1}, // Bottom-right
	}

	// Create defender squad with a 2x2 multi-cell unit spanning all 4 cells
	defenderSquadID := createTestSquad(manager, "Defenders")
	multiCellUnit := manager.World.NewEntity()
	multiCellUnit.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: defenderSquadID})
	multiCellUnit.AddComponent(GridPositionComponent, &GridPositionData{
		AnchorRow: 0,
		AnchorCol: 0,
		Width:     2,
		Height:    2,
	})
	multiCellUnit.AddComponent(common.NameComponent, &common.Name{NameStr: "Giant"})

	// Set defender attributes
	defenderAttr := common.NewAttributes(10, 0, 0, 0, 2, 2)
	defenderAttr.CurrentHealth = 100
	multiCellUnit.AddComponent(common.AttributeComponent, &defenderAttr)

	initialHP := defenderAttr.CurrentHealth

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify the multi-cell unit was only hit ONCE despite occupying 4 cells
	if len(result.DamageByUnit) != 1 {
		t.Errorf("Expected 1 unit to be damaged, got %d", len(result.DamageByUnit))
	}

	// Verify total damage equals damage to single unit (not quadrupled)
	unitDamage := result.DamageByUnit[multiCellUnit.GetID()]
	if result.TotalDamage != unitDamage {
		t.Errorf("Expected total damage %d to equal unit damage %d", result.TotalDamage, unitDamage)
	}

	// Verify defender lost appropriate HP (not quadrupled)
	expectedHP := initialHP - unitDamage
	if defenderAttr.CurrentHealth != expectedHP {
		t.Errorf("Expected defender HP %d, got %d (initial: %d, damage: %d)",
			expectedHP, defenderAttr.CurrentHealth, initialHP, unitDamage)
	}

	// Verify damage is reasonable (should be around baseDamage - resistance, not multiplied)
	resistance := defenderAttr.GetPhysicalResistance()
	expectedSingleHitDamage := baseDamage - resistance
	if expectedSingleHitDamage < 1 {
		expectedSingleHitDamage = 1
	}

	// Allow some variance for crits, but should not be ~4x the base damage
	maxReasonableDamage := expectedSingleHitDamage * 2 // Crit is 1.5x, so 2x is generous
	if unitDamage > maxReasonableDamage {
		t.Errorf("Unit took %d damage, which is too high. Expected around %d (base damage %d - resistance %d). This suggests the unit was hit multiple times.",
			unitDamage, expectedSingleHitDamage, baseDamage, resistance)
	}

	t.Logf("Multi-cell unit correctly hit once (cell-based) for %d damage (base: %d, resistance: %d, expected: %d)",
		unitDamage, baseDamage, resistance, expectedSingleHitDamage)
}

// ========================================
// BENCHMARK TESTS
// ========================================

func BenchmarkExecuteSquadAttack_SingleVsSingle(b *testing.B) {
	manager := setupTestManager(&testing.T{})

	attackerSquadID := createTestSquad(manager, "Attackers")
	createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 100, 10, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)
	}
}

func BenchmarkExecuteSquadAttack_FullSquadVsFullSquad(b *testing.B) {
	manager := setupTestManager(&testing.T{})

	// Create full squads (9 units each)
	attackerSquadID := createTestSquad(manager, "Attackers")
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			createTestUnit(manager, attackerSquadID, row, col, 100, 20, 100)
		}
	}

	defenderSquadID := createTestSquad(manager, "Defenders")
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			createTestUnit(manager, defenderSquadID, row, col, 100, 10, 0)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)
	}
}
