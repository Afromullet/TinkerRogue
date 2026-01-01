package squads

import (
	"game_main/common"
	"testing"
)

// TestMagicDamageInRealCombat tests magic damage in a realistic combat scenario
func TestMagicDamageInRealCombat(t *testing.T) {
	manager := setupTestManager(t)

	// Create attacker squad with a Wizard
	attackerSquadID := createTestSquad(manager, "Magic Squad")
	wizard := manager.World.NewEntity()
	wizard.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: attackerSquadID})
	wizard.AddComponent(GridPositionComponent, &GridPositionData{AnchorRow: 2, AnchorCol: 1, Width: 1, Height: 1})
	wizard.AddComponent(common.NameComponent, &common.Name{NameStr: "Wizard"})

	// Wizard stats: Str=10, Magic=15
	wizardAttr := common.NewAttributes(10, 100, 15, 25, 3, 2) // High dex for guaranteed hit
	wizardAttr.CurrentHealth = 40 // HP = 20 + (10*2)
	wizard.AddComponent(common.AttributeComponent, &wizardAttr)

	// Add magic attack type
	wizard.AddComponent(TargetRowComponent, &TargetRowData{
		AttackType:  AttackTypeMagic,
		TargetCells: [][2]int{{0, 0}, {0, 1}, {0, 2}}, // Target front row
	})
	wizard.AddComponent(AttackRangeComponent, &AttackRangeData{Range: 4})

	// Create defender squad with a Fighter
	defenderSquadID := createTestSquad(manager, "Physical Squad")
	fighter := manager.World.NewEntity()
	fighter.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: defenderSquadID})
	fighter.AddComponent(GridPositionComponent, &GridPositionData{AnchorRow: 0, AnchorCol: 0, Width: 1, Height: 1})
	fighter.AddComponent(common.NameComponent, &common.Name{NameStr: "Fighter"})

	// Fighter stats
	fighterAttr := common.NewAttributes(15, 0, 0, 20, 8, 10) // Low dex to avoid dodge
	fighterAttr.CurrentHealth = 50
	fighter.AddComponent(common.AttributeComponent, &fighterAttr)

	fighter.AddComponent(TargetRowComponent, &TargetRowData{
		AttackType:  AttackTypeMeleeRow,
		TargetCells: nil,
	})
	fighter.AddComponent(AttackRangeComponent, &AttackRangeData{Range: 1})

	// Execute squad attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Check if wizard dealt damage
	t.Logf("=== MAGIC DAMAGE DEBUG ===")
	t.Logf("Total damage dealt: %d", result.TotalDamage)
	t.Logf("Number of attacks: %d", len(result.CombatLog.AttackEvents))

	expectedMagicDamage := wizardAttr.GetMagicDamage() // Should be 45
	t.Logf("Expected magic damage: %d (Magic=%d * 3)", expectedMagicDamage, wizardAttr.Magic)

	fighterMagicDefense := fighterAttr.GetMagicDefense() // Should be 0
	t.Logf("Fighter magic defense: %d (Magic=%d / 2)", fighterMagicDefense, fighterAttr.Magic)

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

	if result.TotalDamage < 30 {
		t.Errorf("Wizard should deal ~45 magic damage (minus resistance), but total damage was only %d", result.TotalDamage)
	}
}
