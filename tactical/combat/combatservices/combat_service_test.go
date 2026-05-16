package combatservices

import (
	"game_main/core/common"
	"game_main/tactical/combat/combattypes"
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

	if result.Status.Success {
		t.Error("Attack with non-existent squads should fail")
	}
}

// TestCombatResult_Structure tests that CombatResult struct is properly populated
func TestCombatResult_Structure(t *testing.T) {
	result := &combattypes.CombatResult{
		Status: combattypes.CombatStatus{
			Success:         true,
			ErrorReason:     "",
			TargetDestroyed: false,
		},
		Damage: &combattypes.DamageRecord{
			TotalDamage:   25,
			UnitsKilled:   []ecs.EntityID{},
			DamageByUnit:  make(map[ecs.EntityID]int),
			HealingByUnit: make(map[ecs.EntityID]int),
		},
	}

	if !result.Status.Success {
		t.Error("Success field should be true")
	}

	if result.Status.TargetDestroyed {
		t.Error("TargetDestroyed should be false")
	}

	if result.Damage.TotalDamage != 25 {
		t.Error("TotalDamage should be 25")
	}

	if result.Damage.UnitsKilled == nil {
		t.Error("UnitsKilled should not be nil")
	}

	if result.Damage.DamageByUnit == nil {
		t.Error("DamageByUnit should not be nil")
	}
}
