package squadservices

import (
	"game_main/common"
	"game_main/tactical/squads"
	testfx "game_main/testing"
	"testing"

	"github.com/bytearena/ecs"
)

// setupTestManager creates a manager with squad system initialized
func setupBuilderTestManager(t *testing.T) *common.EntityManager {
	manager := testfx.NewTestEntityManager()
	if err := squads.InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}
	return manager
}

// setupTestPlayerWithRoster creates a player entity with an empty roster for testing
func setupTestPlayerWithRoster(manager *common.EntityManager) ecs.EntityID {
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	// Add roster component
	roster := squads.NewUnitRoster(50) // Test capacity
	playerEntity.AddComponent(squads.UnitRosterComponent, roster)

	return playerID
}

// TestSquadBuilderServiceCreation tests that SquadBuilderService can be created
func TestSquadBuilderServiceCreation(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	if service == nil {
		t.Error("SquadBuilderService should not be nil")
	}

	if service.entityManager != manager {
		t.Error("EntityManager not set correctly")
	}
}

// TestAssignRosterUnitToSquad_Success tests assigning a roster unit to a squad
func TestAssignRosterUnitToSquad_Success(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create player with roster
	playerID := setupTestPlayerWithRoster(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Place Unit Test")

	// Create roster unit
	rosterUnitEntity := manager.World.NewEntity()
	rosterUnitID := rosterUnitEntity.GetID()
	roster := squads.GetPlayerRoster(playerID, manager)
	roster.AddUnit(rosterUnitID, "Test Warrior")

	unitTemplate := squads.UnitTemplate{
		Name: "Test Warrior",
		Attributes: common.Attributes{
			Strength:  12,
			Dexterity: 10,
			Magic:     5,

			MaxHealth: 15,
		},
		GridWidth:  1,
		GridHeight: 1,
	}

	// Assign unit to squad
	result := service.AssignRosterUnitToSquad(playerID, squadID, rosterUnitID, unitTemplate, 0, 0)

	if !result.Success {
		t.Errorf("Should assign unit successfully, got error: %s", result.Error)
	}

	if result.PlacedUnitID == 0 {
		t.Error("PlacedUnitID should not be 0")
	}
}

// TestAssignRosterUnitToSquad_InvalidPosition tests assigning unit at invalid position
func TestAssignRosterUnitToSquad_InvalidPosition(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create player with roster
	playerID := setupTestPlayerWithRoster(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Invalid Position Test")

	rosterUnitEntity := manager.World.NewEntity()
	rosterUnitID := rosterUnitEntity.GetID()
	roster := squads.GetPlayerRoster(playerID, manager)
	roster.AddUnit(rosterUnitID, "Warrior")

	unitTemplate := squads.UnitTemplate{
		Name:       "Warrior",
		GridWidth:  1,
		GridHeight: 1,
	}

	// Try invalid position (row 5 is out of bounds for 3x3 grid)
	result := service.AssignRosterUnitToSquad(playerID, squadID, rosterUnitID, unitTemplate, 5, 0)

	if result.Success {
		t.Error("Should reject invalid grid position")
	}

	if result.Error == "" {
		t.Error("Error message should be populated")
	}
}
