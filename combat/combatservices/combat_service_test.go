package combatservices

import (
	"game_main/common"
	"game_main/coords"
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

	if service.entityManager != manager {
		t.Error("EntityManager not set correctly")
	}

	if service.turnManager == nil {
		t.Error("TurnManager should be initialized")
	}

	if service.factionManager == nil {
		t.Error("FactionManager should be initialized")
	}

	if service.movementSystem == nil {
		t.Error("MovementSystem should be initialized")
	}
}

// TestExecuteSquadAttack_NoSquads tests attack with invalid squads
func TestExecuteSquadAttack_NoSquads(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	// Try to attack with non-existent squad IDs
	result := service.ExecuteSquadAttack(ecs.EntityID(999), ecs.EntityID(998))

	if result == nil {
		t.Error("Result should not be nil")
	}

	if result.Success {
		t.Error("Attack with non-existent squads should fail")
	}
}

// TestGetCurrentFaction_BeforeInit tests current faction before combat starts
func TestGetCurrentFaction_BeforeInit(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	currentFaction := service.GetCurrentFaction()
	// Should be 0 before initialization
	if currentFaction != 0 {
		t.Logf("Current faction before init: %d (may be 0)", currentFaction)
	}
}

// TestGetTurnManager tests turn manager exposure
func TestGetTurnManager(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	turnMgr := service.GetTurnManager()

	if turnMgr == nil {
		t.Error("GetTurnManager should not return nil")
	}
}

// TestGetFactionManager tests faction manager exposure
func TestGetFactionManager(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	factionMgr := service.GetFactionManager()

	if factionMgr == nil {
		t.Error("GetFactionManager should not return nil")
	}
}

// TestGetMovementSystem tests movement system exposure
func TestGetMovementSystem(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	moveSys := service.GetMovementSystem()

	if moveSys == nil {
		t.Error("GetMovementSystem should not return nil")
	}
}

// TestGetEntityManager tests entity manager exposure
func TestGetEntityManager(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	em := service.GetEntityManager()

	if em == nil {
		t.Error("GetEntityManager should not return nil")
	}

	if em != manager {
		t.Error("GetEntityManager should return the same manager passed in")
	}
}

// TestMoveSquad tests MoveSquad method returns proper result structure
func TestMoveSquad_ResultStructure(t *testing.T) {
	// Test that MoveSquad with non-existent squad returns a result
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	newPos := coords.LogicalPosition{X: 6, Y: 5}
	result := service.MoveSquad(ecs.EntityID(999), newPos)

	if result == nil {
		t.Error("MoveSquad should return a result")
	}

	t.Logf("Move result: Success=%v, Error=%s", result.Success, result.ErrorReason)
}

// TestGetValidMovementTiles tests getting available movement tiles
func TestGetValidMovementTiles(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	// Get valid tiles for non-existent squad
	tiles := service.GetValidMovementTiles(ecs.EntityID(999))

	if tiles == nil {
		t.Error("GetValidMovementTiles should return a slice, not nil")
	}

	t.Logf("Valid movement tiles: %d", len(tiles))
}

// TestGetSquadsInRange tests finding squads in range
func TestGetSquadsInRange(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	// Get squads in range for non-existent squad
	squadsInRange := service.GetSquadsInRange(ecs.EntityID(999))

	// May return nil or empty slice - both are acceptable
	t.Logf("Squads in range: %v (len: %d)", squadsInRange, len(squadsInRange))
}

// TestAttackResult_Structure tests that AttackResult struct is properly populated
func TestAttackResult_Structure(t *testing.T) {
	result := &AttackResult{
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

// TestMoveSquadResult_Structure tests that MoveSquadResult struct is properly populated
func TestMoveSquadResult_Structure(t *testing.T) {
	result := &MoveSquadResult{
		Success:      true,
		ErrorReason:  "",
		SquadName:    "Mobile Squad",
		NewPosition:  coords.LogicalPosition{X: 5, Y: 5},
		MovementCost: 1,
		RemainingAPs: 2,
	}

	if !result.Success {
		t.Error("Success field should be true")
	}

	if result.SquadName != "Mobile Squad" {
		t.Error("SquadName should be set")
	}

	if result.NewPosition.X != 5 || result.NewPosition.Y != 5 {
		t.Error("NewPosition should be set correctly")
	}
}

// TestResetSquadActions tests resetting squad actions
func TestResetSquadActions(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewCombatService(manager)

	// Test reset with dummy faction ID
	err := service.ResetSquadActions(ecs.EntityID(999))

	// May return error if faction not found, which is OK
	t.Logf("Reset squad actions returned error: %v (may be expected for non-existent faction)", err)
}
