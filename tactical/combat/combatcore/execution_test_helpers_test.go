package combatcore

import (
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combat/combatmath"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	testfx "game_main/testing"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// EXECUTION TEST HELPERS
// ========================================
// Used by the targeting / damage / cover / integration / benchmark tests
// extracted from the original combatexecution_test.go.
//
// Distinct from combat_test_helpers_test.go's CreateTest* helpers (which
// power the higher-level component/faction tests in combat_test.go).

// executeTestAttack replicates what ExecuteSquadAttack did using the extracted pipeline functions.
func executeTestAttack(attackerSquadID, defenderSquadID ecs.EntityID, manager *common.EntityManager) *combattypes.CombatResult {
	result := &combattypes.CombatResult{
		DamageByUnit:  make(map[ecs.EntityID]int),
		HealingByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:   []ecs.EntityID{},
	}

	combatLog := battlelog.InitializeCombatLog(attackerSquadID, defenderSquadID, manager)
	if combatLog.SquadDistance < 0 {
		result.CombatLog = combatLog
		return result
	}

	combatLog.AttackingUnits = battlelog.SnapshotAttackingUnits(attackerSquadID, combatLog.SquadDistance, manager)
	combatLog.DefendingUnits = battlelog.SnapshotAllUnits(defenderSquadID, manager)

	attackIndex := 0
	attackerUnitIDs := squadcore.GetUnitIDsInSquad(attackerSquadID, manager)

	for _, attackerID := range attackerUnitIDs {
		if !combatmath.CanUnitAttack(attackerID, combatLog.SquadDistance, manager) {
			continue
		}

		targetIDs := combatmath.SelectTargetUnits(attackerID, defenderSquadID, manager)
		attackIndex = ProcessAttackOnTargets(attackerID, defenderSquadID, targetIDs, result, combatLog, attackIndex, nil, manager)
	}

	combatmath.ApplyRecordedDamage(result, manager)
	battlelog.FinalizeCombatLog(result, combatLog, defenderSquadID, attackerSquadID, manager)
	result.CombatLog = combatLog
	return result
}

// calculateTestDamage replaces calculateUnitDamageByID with zero modifiers.
func calculateTestDamage(attackerID, defenderID ecs.EntityID, manager *common.EntityManager) (int, *combattypes.AttackEvent) {
	modifiers := combattypes.DamageModifiers{
		HitModifier:      0,
		DamageMultiplier: 1.0,
		IsCounterattack:  false,
	}
	return combatmath.CalculateDamage(attackerID, defenderID, modifiers, nil, manager)
}

// setupCombatTestManager creates a fully initialized EntityManager for combat tests.
func setupCombatTestManager(t *testing.T) *common.EntityManager {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	if err := squadcore.InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}
	return manager
}

// createTestSquad creates a squad entity with specified name at position (0,0).
func createTestSquad(manager *common.EntityManager, name string) ecs.EntityID {
	squad := manager.World.NewEntity()
	squadID := squad.GetID()

	squadData := &squadcore.SquadData{
		SquadID:    squadID,
		Formation:  squadcore.FormationBalanced,
		Name:       name,
		Morale:     100,
		SquadLevel: 1,
		TurnCount:  0,
		MaxUnits:   9,
	}

	squad.AddComponent(squadcore.SquadComponent, squadData)

	// Add position component so squads can calculate distance
	squad.AddComponent(common.PositionComponent, &coords.LogicalPosition{
		X: 0,
		Y: 0,
	})

	return squadID
}

// createTestUnit creates a unit with specified attributes for testing.
func createTestUnit(manager *common.EntityManager, squadID ecs.EntityID, row, col int, health, strength, dexterity int) *ecs.Entity {
	unit := manager.World.NewEntity()

	// Add required components
	unit.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: squadID})
	unit.AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{
		AnchorRow:  row,
		AnchorCol:  col,
		CellWidth:  1,
		CellHeight: 1,
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
	unit.AddComponent(squadcore.TargetRowComponent, &squadcore.TargetRowData{
		AttackType:  unitdefs.AttackTypeMeleeRow, // Explicit attack type
		TargetCells: nil,                         // Not used for MeleeRow
	})

	// Add attack range component (default to melee range 1)
	unit.AddComponent(squadcore.AttackRangeComponent, &squadcore.AttackRangeData{
		Range: 1,
	})

	return unit
}
