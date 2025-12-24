package combatservices

import (
	"game_main/common"
	"game_main/tactical/combat"
	"testing"

	"github.com/bytearena/ecs"
)

// TestCombatServiceCreation tests that CombatService can be created
func TestCombatServiceCreation(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	if service == nil {
		t.Error("CombatService should not be nil")
	}

	if service.EntityManager != manager {
		t.Error("EntityManager not set correctly")
	}

	if service.TurnManager == nil {
		t.Error("TurnManager should be initialized")
	}

	if service.FactionManager == nil {
		t.Error("FactionManager should be initialized")
	}

	if service.MovementSystem == nil {
		t.Error("MovementSystem should be initialized")
	}
}

// TestExecuteSquadAttack_NoSquads tests attack with invalid squads
func TestExecuteSquadAttack_NoSquads(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	// Try to attack with non-existent squad IDs
	result := service.CombatActSystem.ExecuteAttackAction(ecs.EntityID(999), ecs.EntityID(998))

	if result == nil {
		t.Error("Result should not be nil")
	}

	if result.Success {
		t.Error("Attack with non-existent squads should fail")
	}
}

// TestAttackResult_Structure tests that AttackResult struct is properly populated
func TestAttackResult_Structure(t *testing.T) {
	result := &combat.AttackResult{
		Success:         true,
		ErrorReason:     "",
		AttackerName:    "Attacker",
		TargetName:      "Defender",
		TargetDestroyed: false,
		DamageDealt:     10,
	}

	if !result.Success {
		t.Error("Success field should be true")
	}

	if result.AttackerName != "Attacker" {
		t.Error("AttackerName should be set")
	}

	if result.TargetName != "Defender" {
		t.Error("TargetName should be set")
	}

	if result.DamageDealt != 10 {
		t.Error("DamageDealt should be 10")
	}
}
