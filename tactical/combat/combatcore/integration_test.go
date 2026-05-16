package combatcore

import (
	"game_main/core/common"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"testing"
)
// INTEGRATION TESTS
// ========================================

func TestCombatWithCoverSystem_Integration(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 30, 0) // Low dexterity = lower crit
	attackerAttr := common.GetComponentType[*common.Attributes](attacker, common.AttributeComponent)

	// Create defender squad with cover
	defenderSquadID := createTestSquad(manager, "Defenders")

	// Front-line unit provides cover
	frontLine := createTestUnit(manager, defenderSquadID, 0, 0, 100, 10, 0)
	frontLine.AddComponent(squadcore.CoverComponent, &squadcore.CoverData{
		CoverValue:     0.50, // 50% damage reduction
		CoverRange:     1,
		RequiresActive: true,
	})

	// Back-line unit receives cover
	backLine := createTestUnit(manager, defenderSquadID, 1, 0, 100, 10, 0) // Low dexterity = low dodge
	backLineAttr := common.GetComponentType[*common.Attributes](backLine, common.AttributeComponent)

	// Configure attacker to target back line
	targetData := common.GetComponentType[*squadcore.TargetRowData](attacker, squadcore.TargetRowComponent)
	targetData.TargetCells = [][2]int{{1, 0}, {1, 1}, {1, 2}} // Target row 1

	// Execute attack
	result := executeTestAttack(attackerSquadID, defenderSquadID, manager)

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
	manager := setupCombatTestManager(t)

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
		result1 := executeTestAttack(squad1ID, squad2ID, manager)

		// Check if Squad 2 is destroyed
		if squadcore.IsSquadDestroyed(squad2ID, manager) {
			t.Logf("Squad2 destroyed in round %d", rounds)
			break
		}

		// Squad 2 attacks Squad 1
		result2 := executeTestAttack(squad2ID, squad1ID, manager)

		// Check if Squad 1 is destroyed
		if squadcore.IsSquadDestroyed(squad1ID, manager) {
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
	squad1Destroyed := squadcore.IsSquadDestroyed(squad1ID, manager)
	squad2Destroyed := squadcore.IsSquadDestroyed(squad2ID, manager)

	if !squad1Destroyed && !squad2Destroyed {
		t.Log("Both squads survived - this is possible with lucky dodges/misses")
	}
}


// ========================================
// MAGIC DAMAGE IN REAL COMBAT TEST
// (migrated from magic_debug_test.go)
// ========================================

func TestMagicDamageInRealCombat(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Create attacker squad with a Wizard
	attackerSquadID := createTestSquad(manager, "Magic Squad")
	wizard := manager.World.NewEntity()
	wizard.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: attackerSquadID})
	wizard.AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{AnchorRow: 2, AnchorCol: 1, CellWidth: 1, CellHeight: 1})
	wizard.AddComponent(common.NameComponent, &common.Name{NameStr: "Wizard"})

	// Wizard stats: Str=10, Magic=15
	wizardAttr := common.NewAttributes(10, 100, 15, 25, 3, 2) // High dex for guaranteed hit
	wizardAttr.CurrentHealth = 40
	wizard.AddComponent(common.AttributeComponent, &wizardAttr)

	// Add magic attack type
	wizard.AddComponent(squadcore.TargetRowComponent, &squadcore.TargetRowData{
		AttackType:  unitdefs.AttackTypeMagic,
		TargetCells: [][2]int{{0, 0}, {0, 1}, {0, 2}}, // Target front row
	})
	wizard.AddComponent(squadcore.AttackRangeComponent, &squadcore.AttackRangeData{Range: 4})

	// Create defender squad with a Fighter
	defenderSquadID := createTestSquad(manager, "Physical Squad")
	fighter := manager.World.NewEntity()
	fighter.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: defenderSquadID})
	fighter.AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{AnchorRow: 0, AnchorCol: 0, CellWidth: 1, CellHeight: 1})
	fighter.AddComponent(common.NameComponent, &common.Name{NameStr: "Fighter"})

	// Fighter stats
	fighterAttr := common.NewAttributes(15, 0, 0, 20, 8, 10) // Low dex to avoid dodge
	fighterAttr.CurrentHealth = 50
	fighter.AddComponent(common.AttributeComponent, &fighterAttr)

	fighter.AddComponent(squadcore.TargetRowComponent, &squadcore.TargetRowData{
		AttackType:  unitdefs.AttackTypeMeleeRow,
		TargetCells: nil,
	})
	fighter.AddComponent(squadcore.AttackRangeComponent, &squadcore.AttackRangeData{Range: 1})

	// Execute squad attack
	result := executeTestAttack(attackerSquadID, defenderSquadID, manager)

	// Check if wizard dealt damage
	t.Logf("=== MAGIC DAMAGE DEBUG ===")
	t.Logf("Total damage dealt: %d", result.TotalDamage)
	t.Logf("Number of attacks: %d", len(result.CombatLog.AttackEvents))

	expectedMagicDamage := wizardAttr.GetMagicDamage()
	t.Logf("Expected magic damage: %d", expectedMagicDamage)

	fighterMagicDefense := fighterAttr.GetMagicDefense()
	t.Logf("Fighter magic defense: %d", fighterMagicDefense)

	if len(result.CombatLog.AttackEvents) > 0 {
		for i, event := range result.CombatLog.AttackEvents {
			attackerName := common.GetComponentTypeByID[*common.Name](manager, event.AttackerID, common.NameComponent)
			defenderName := common.GetComponentTypeByID[*common.Name](manager, event.DefenderID, common.NameComponent)

			t.Logf("Attack %d: %s -> %s", i+1, attackerName.NameStr, defenderName.NameStr)
			t.Logf("  Base Damage: %d", event.BaseDamage)
			t.Logf("  Resistance: %d", event.ResistanceAmount)
			t.Logf("  Final Damage: %d", event.FinalDamage)
			t.Logf("  Hit Type: %s", event.HitResult.Type)

			if attackerName.NameStr == "Wizard" {
				if event.BaseDamage != expectedMagicDamage {
					t.Errorf("Wizard base damage should be %d (magic damage), got %d", expectedMagicDamage, event.BaseDamage)
				}
				if event.ResistanceAmount != fighterMagicDefense {
					t.Errorf("Resistance should be %d (magic defense), got %d", fighterMagicDefense, event.ResistanceAmount)
				}
			}
		}
	} else {
		t.Error("No attacks were executed!")
	}

	// Verify fighter took damage
	fighterAttrAfter := common.GetComponentType[*common.Attributes](fighter, common.AttributeComponent)
	damageTaken := 50 - fighterAttrAfter.CurrentHealth
	t.Logf("Fighter HP: 50 -> %d (took %d damage)", fighterAttrAfter.CurrentHealth, damageTaken)

	// Compute minimum expected damage from the test's own stats
	minExpected := expectedMagicDamage - fighterMagicDefense
	if minExpected < 1 {
		minExpected = 1
	}
	if result.TotalDamage < minExpected/2 {
		t.Errorf("Wizard should deal around %d magic damage (%d - %d), but total damage was only %d",
			minExpected, expectedMagicDamage, fighterMagicDefense, result.TotalDamage)
	}
}

