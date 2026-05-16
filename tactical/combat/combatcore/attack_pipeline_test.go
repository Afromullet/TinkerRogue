package combatcore

import (
	"game_main/core/common"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"testing"
)
// ========================================
// ExecuteSquadAttack TESTS
// ========================================

func TestExecuteSquadAttack_SingleAttackerVsSingleDefender(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	_ = createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100) // 100% hit rate

	// Create defender squad
	defenderSquadID := createTestSquad(manager, "Defenders")
	defenderUnit := createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	defenderAttr := common.GetComponentType[*common.Attributes](defenderUnit, common.AttributeComponent)
	initialHP := defenderAttr.CurrentHealth

	// Execute attack
	result := executeTestAttack(attackerSquadID, defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

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
	result := executeTestAttack(attackerSquadID, defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

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
	result := executeTestAttack(attackerSquadID, defenderSquadID, manager)

	// Verify no damage dealt
	if result.TotalDamage != 0 {
		t.Errorf("Expected no damage from dead attacker, got %d", result.TotalDamage)
	}

	if defenderAttr.CurrentHealth != initialHP {
		t.Errorf("Expected defender HP to remain %d, got %d", initialHP, defenderAttr.CurrentHealth)
	}
}

func TestExecuteSquadAttack_MultiTargetAttack(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker with multi-target ability
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set magic targeting (target 2 specific cells)
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMagic  // Use magic for cell-based targeting
	targetData.TargetCells = [][2]int{{0, 0}, {0, 1}} // Target first two front-row cells

	// Create defenders
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0) // Should be hit
	createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0) // Should be hit
	createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0) // Should NOT be hit

	// Execute attack
	result := executeTestAttack(attackerSquadID, defenderSquadID, manager)

	// Verify exactly 2 targets hit (the ones in targeted cells)
	if len(result.DamageByUnit) != 2 {
		t.Errorf("Expected 2 units damaged, got %d", len(result.DamageByUnit))
	}
}

func TestExecuteSquadAttack_CellBasedTargeting(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker with cell-based targeting (2x2 pattern)
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set magic targeting with 2x2 pattern
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMagic                  // Use magic for cell-based targeting
	targetData.TargetCells = [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}} // 2x2 top-left

	// Create defenders in targeted cells
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0) // Hit
	createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0) // Hit
	createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Hit
	createTestUnit(manager, defenderSquadID, 2, 2, 50, 10, 0) // Miss (not in pattern)

	// Execute attack
	result := executeTestAttack(attackerSquadID, defenderSquadID, manager)

	// Verify 3 units hit (3 in the 2x2 pattern)
	if len(result.DamageByUnit) != 3 {
		t.Errorf("Expected 3 units damaged (2x2 pattern), got %d", len(result.DamageByUnit))
	}
}

func TestExecuteSquadAttack_UnitsKilledTracking(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker with high damage
	attackerSquadID := createTestSquad(manager, "Attackers")
	createTestUnit(manager, attackerSquadID, 0, 0, 100, 100, 100) // High strength

	// Create weak defender
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 10, 5, 0) // Low HP

	// Execute attack
	result := executeTestAttack(attackerSquadID, defenderSquadID, manager)

	// Verify unit was killed
	if len(result.UnitsKilled) != 1 {
		t.Errorf("Expected 1 unit killed, got %d", len(result.UnitsKilled))
	}
}


func TestExecuteSquadAttack_MultiCellUnit_HitOnce(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker squad with unit targeting multiple rows
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)
	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	baseDamage := attackerAttr.GetPhysicalDamage()

	// Set attacker to target both row 0 and row 1
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.TargetCells = [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}}

	// Create defender squad with a 2x2 multi-cell unit spanning rows 0-1, cols 0-1
	defenderSquadID := createTestSquad(manager, "Defenders")
	multiCellUnit := manager.World.NewEntity()
	multiCellUnit.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: defenderSquadID})
	multiCellUnit.AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{
		AnchorRow:  0,
		AnchorCol:  0,
		CellWidth:  2,
		CellHeight: 2,
	})
	multiCellUnit.AddComponent(common.NameComponent, &common.Name{NameStr: "Giant"})

	// Set defender attributes
	defenderAttr := common.NewAttributes(10, 0, 0, 0, 2, 2)
	defenderAttr.CurrentHealth = 100
	multiCellUnit.AddComponent(common.AttributeComponent, &defenderAttr)

	initialHP := defenderAttr.CurrentHealth

	// Execute attack
	result := executeTestAttack(attackerSquadID, defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

	// Create attacker squad with cell-based targeting
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)
	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)
	baseDamage := attackerAttr.GetPhysicalDamage()

	// Set attacker to target all 4 cells occupied by the multi-cell unit
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.TargetCells = [][2]int{
		{0, 0}, // Top-left
		{0, 1}, // Top-right
		{1, 0}, // Bottom-left
		{1, 1}, // Bottom-right
	}

	// Create defender squad with a 2x2 multi-cell unit spanning all 4 cells
	defenderSquadID := createTestSquad(manager, "Defenders")
	multiCellUnit := manager.World.NewEntity()
	multiCellUnit.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: defenderSquadID})
	multiCellUnit.AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{
		AnchorRow:  0,
		AnchorCol:  0,
		CellWidth:  2,
		CellHeight: 2,
	})
	multiCellUnit.AddComponent(common.NameComponent, &common.Name{NameStr: "Giant"})

	// Set defender attributes
	defenderAttr := common.NewAttributes(10, 0, 0, 0, 2, 2)
	defenderAttr.CurrentHealth = 100
	multiCellUnit.AddComponent(common.AttributeComponent, &defenderAttr)

	initialHP := defenderAttr.CurrentHealth

	// Execute attack
	result := executeTestAttack(attackerSquadID, defenderSquadID, manager)

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

