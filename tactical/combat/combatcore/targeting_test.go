package combatcore

import (
	"game_main/core/common"
	"game_main/tactical/combat/combatmath"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"testing"

	"github.com/bytearena/ecs"
)
// NEW TARGETING SYSTEM TESTS
// ========================================

func TestMeleeRowTargeting_FrontRow(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker squad with MeleeRow attacker
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to MeleeRow targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMeleeRow
	targetData.TargetCells = nil

	// Create defender squad with 3 units in front row (row 0)
	defenderSquadID := createTestSquad(manager, "Defenders")
	defender1 := createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	defender2 := createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0)
	defender3 := createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)

	// Get targets
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to MeleeRow targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMeleeRow
	targetData.TargetCells = nil

	// Create defender squad with units only in row 1 (front row empty)
	defenderSquadID := createTestSquad(manager, "Defenders")
	defender1 := createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0)
	defender2 := createTestUnit(manager, defenderSquadID, 1, 2, 50, 10, 0)

	// Get targets
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to MeleeRow targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMeleeRow
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
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to MeleeRow targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMeleeRow
	targetData.TargetCells = nil

	// Create defender squad with units only in row 2 (rows 0 and 1 empty)
	defenderSquadID := createTestSquad(manager, "Defenders")
	defender1 := createTestUnit(manager, defenderSquadID, 2, 0, 50, 10, 0)
	defender2 := createTestUnit(manager, defenderSquadID, 2, 2, 50, 10, 0)

	// Get targets
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

	// Create attacker squad with MeleeColumn attacker in column 1
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 1, 100, 20, 100)

	// Set to MeleeColumn targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMeleeColumn
	targetData.TargetCells = nil

	// Create defender squad with units in different columns
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)              // Column 0 - NOT targeted
	defender2 := createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0) // Column 1 - TARGETED
	createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)              // Column 2 - NOT targeted

	// Get targets
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify only 1 target in the same column
	if len(targets) != 1 {
		t.Errorf("Expected 1 target (same column), got %d", len(targets))
	}

	if targets[0] != defender2.GetID() {
		t.Errorf("Expected defender in column 1, got %d", targets[0])
	}
}

func TestMeleeColumnTargeting_PierceForward(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker squad in column 0
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to MeleeColumn targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMeleeColumn
	targetData.TargetCells = nil

	// Create defender squad with units in column 0 (row 0 col 0 is empty)
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0)              // Different column - NOT targeted
	defender2 := createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Same column, row 1 - TARGETED
	defender3 := createTestUnit(manager, defenderSquadID, 2, 0, 50, 10, 0) // Same column, row 2 - ALSO TARGETED (piercing attack)

	// Get targets
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

	// Create attacker squad in column 1
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 1, 100, 20, 100)

	// Set to MeleeColumn targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMeleeColumn
	targetData.TargetCells = nil

	// Create defender squad with NO units in column 1, but units in columns 2 and 0
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)              // Column 0 - should wrap to this
	defender2 := createTestUnit(manager, defenderSquadID, 1, 2, 50, 10, 0) // Column 2 - TARGETED (next after col 1)

	// Get targets
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify wrapping: attackerCol=1 -> try col 1 (empty), col 2 (found!), col 0 (not reached)
	if len(targets) != 1 {
		t.Errorf("Expected 1 target (column 2 after wrapping), got %d", len(targets))
	}

	if targets[0] != defender2.GetID() {
		t.Errorf("Expected defender in column 2, got %d", targets[0])
	}
}

func TestMeleeColumnTargeting_WrapToColumn0(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker squad in column 2
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 2, 100, 20, 100)

	// Set to MeleeColumn targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMeleeColumn
	targetData.TargetCells = nil

	// Create defender squad with NO units in columns 2, only in column 0
	defenderSquadID := createTestSquad(manager, "Defenders")
	defender1 := createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Column 0 - TARGETED (after wrapping)

	// Get targets
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify wrapping: attackerCol=2 -> try col 2 (empty), col 0 (found!), col 1 (not reached)
	if len(targets) != 1 {
		t.Errorf("Expected 1 target (column 0 after wrapping), got %d", len(targets))
	}

	if targets[0] != defender1.GetID() {
		t.Errorf("Expected defender in column 0, got %d", targets[0])
	}
}

func TestRangedTargeting_SameRow(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker squad with ranged attacker in row 1
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 1, 0, 100, 20, 100)

	// Set to Ranged targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeRanged
	targetData.TargetCells = nil

	// Create defender squad with units in different rows
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)              // Row 0 - NOT targeted
	defender2 := createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Row 1 - TARGETED
	defender3 := createTestUnit(manager, defenderSquadID, 1, 1, 50, 10, 0) // Row 1 - TARGETED
	defender4 := createTestUnit(manager, defenderSquadID, 1, 2, 50, 10, 0) // Row 1 - TARGETED
	createTestUnit(manager, defenderSquadID, 2, 0, 50, 10, 0)              // Row 2 - NOT targeted

	// Get targets
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

	// Create attacker squad in row 1
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 1, 0, 100, 20, 100)

	// Set to Ranged targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeRanged
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
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify only 1 target (lowest armor)
	if len(targets) != 1 {
		t.Errorf("Expected 1 target (lowest armor fallback), got %d", len(targets))
	}

	if targets[0] != defender2.GetID() {
		t.Errorf("Expected defender2 (lowest armor), got %d", targets[0])
	}
}

func TestRangedTargeting_FallbackTiebreaker(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker squad in row 1
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 1, 0, 100, 20, 100)

	// Set to Ranged targeting
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeRanged
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
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify defender2 wins (furthest row + leftmost column)
	if len(targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(targets))
	}

	if targets[0] != defender2.GetID() {
		t.Errorf("Expected defender2 (furthest row + leftmost), got %d", targets[0])
	}
}

func TestMagicTargeting_ExactCells(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to Magic targeting with specific pattern (column 1 only)
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMagic
	targetData.TargetCells = [][2]int{{0, 1}, {1, 1}, {2, 1}} // Middle column

	// Create defender squad
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)              // Col 0 - NOT targeted
	defender2 := createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0) // Col 1 - TARGETED
	createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)              // Col 2 - NOT targeted
	defender4 := createTestUnit(manager, defenderSquadID, 1, 1, 50, 10, 0) // Col 1 - TARGETED
	defender5 := createTestUnit(manager, defenderSquadID, 2, 1, 50, 10, 0) // Col 1 - TARGETED

	// Get targets
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set to Magic targeting (front row only)
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.AttackType = unitdefs.AttackTypeMagic
	targetData.TargetCells = [][2]int{{0, 0}, {0, 1}, {0, 2}} // Front row only

	// Create defender squad with NO units in row 0, units in row 1
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Row 1 - NOT targeted (no pierce)
	createTestUnit(manager, defenderSquadID, 1, 1, 50, 10, 0) // Row 1 - NOT targeted (no pierce)

	// Get targets
	targets := combatmath.SelectTargetUnits(attacker.GetID(), defenderSquadID, manager)

	// Verify no targets (magic doesn't pierce)
	if len(targets) != 0 {
		t.Errorf("Expected 0 targets (magic doesn't pierce), got %d", len(targets))
	}
}

