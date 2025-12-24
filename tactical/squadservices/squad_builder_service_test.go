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

// TestRemoveUnitFromGrid tests removing a unit from grid
func TestRemoveUnitFromGrid_Success(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create player with roster
	playerID := setupTestPlayerWithRoster(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Remove Unit Test")

	// Create and place unit
	rosterUnitEntity := manager.World.NewEntity()
	rosterUnitID := rosterUnitEntity.GetID()
	roster := squads.GetPlayerRoster(playerID, manager)
	roster.AddUnit(rosterUnitID, "Warrior")

	unitTemplate := squads.UnitTemplate{
		Name:       "Warrior",
		GridWidth:  1,
		GridHeight: 1,
	}

	assignResult := service.AssignRosterUnitToSquad(playerID, squadID, rosterUnitID, unitTemplate, 0, 0)
	if !assignResult.Success {
		t.Fatalf("Failed to assign unit: %s", assignResult.Error)
	}

	// Remove unit
	remResult := service.RemoveUnitFromGrid(squadID, 0, 0)

	if !remResult.Success {
		t.Errorf("Should remove unit successfully, got error: %s", remResult.Error)
	}
}

// TestRemoveUnitFromGrid_EmptyPosition tests removing from empty position
func TestRemoveUnitFromGrid_EmptyPosition(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Empty Position Test")

	// Try to remove from empty position
	remResult := service.RemoveUnitFromGrid(squadID, 0, 0)

	if remResult.Success {
		t.Error("Should fail to remove from empty position")
	}

	if remResult.Error == "" {
		t.Error("Error message should be populated")
	}
}

// TestDesignateLeader tests designating a unit as leader
func TestDesignateLeader(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create player with roster
	playerID := setupTestPlayerWithRoster(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Leader Test")

	// Create and place unit
	rosterUnitEntity := manager.World.NewEntity()
	rosterUnitID := rosterUnitEntity.GetID()
	roster := squads.GetPlayerRoster(playerID, manager)
	roster.AddUnit(rosterUnitID, "Champion")

	unitTemplate := squads.UnitTemplate{
		Name:       "Champion",
		GridWidth:  1,
		GridHeight: 1,
	}

	assignResult := service.AssignRosterUnitToSquad(playerID, squadID, rosterUnitID, unitTemplate, 0, 0)
	if !assignResult.Success {
		t.Fatalf("Failed to assign unit: %s", assignResult.Error)
	}

	// Designate as leader
	leaderResult := service.DesignateLeader(assignResult.PlacedUnitID)

	if !leaderResult.Success {
		t.Errorf("Should designate leader successfully, got error: %s", leaderResult.Error)
	}
}

// TestGetCapacityInfo tests getting capacity information
func TestGetCapacityInfo(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Capacity Info Test")

	// Get capacity info
	info := service.GetCapacityInfo(squadID)

	if info == nil {
		t.Fatal("GetCapacityInfo returned nil")
	}

	if info.TotalCapacity != 6 {
		t.Errorf("Expected total capacity 6, got %d", info.TotalCapacity)
	}

	if info.UsedCapacity != 0 {
		t.Errorf("Expected used capacity 0, got %f", info.UsedCapacity)
	}

	if info.RemainingCapacity != 6 {
		t.Errorf("Expected remaining capacity 6, got %f", info.RemainingCapacity)
	}

	if info.HasLeader {
		t.Error("Empty squad should not have leader")
	}
}

// TestGetCapacityInfo_WithLeader tests capacity info when squad has leader
func TestGetCapacityInfo_WithLeader(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create player with roster
	playerID := setupTestPlayerWithRoster(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Leader Capacity Test")

	// Create and place unit
	rosterUnitEntity := manager.World.NewEntity()
	rosterUnitID := rosterUnitEntity.GetID()
	roster := squads.GetPlayerRoster(playerID, manager)
	roster.AddUnit(rosterUnitID, "Leader Unit")

	unitTemplate := squads.UnitTemplate{
		Name:       "Leader Unit",
		GridWidth:  1,
		GridHeight: 1,
	}

	assignResult := service.AssignRosterUnitToSquad(playerID, squadID, rosterUnitID, unitTemplate, 0, 0)
	if !assignResult.Success {
		t.Fatalf("Failed to assign unit: %s", assignResult.Error)
	}

	// Designate as leader
	service.DesignateLeader(assignResult.PlacedUnitID)

	// Get capacity info
	info := service.GetCapacityInfo(squadID)

	if !info.HasLeader {
		t.Error("Squad should have leader after designating")
	}
}

// TestValidateSquad tests squad validation
func TestValidateSquad_Empty(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create empty squad
	squadID := squads.CreateEmptySquad(manager, "Empty Squad")

	// Validate empty squad
	validation := service.ValidateSquad(squadID)

	if validation.Valid {
		t.Error("Empty squad should not be valid")
	}

	if validation.UnitCount != 0 {
		t.Errorf("Expected 0 units, got %d", validation.UnitCount)
	}

	if validation.ErrorMsg == "" {
		t.Error("Error message should be populated for empty squad")
	}
}

// TestValidateSquad_NoLeader tests validation fails without leader
func TestValidateSquad_NoLeader(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create player with roster
	playerID := setupTestPlayerWithRoster(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "No Leader Squad")

	// Add unit without designating as leader
	rosterUnitEntity := manager.World.NewEntity()
	rosterUnitID := rosterUnitEntity.GetID()
	roster := squads.GetPlayerRoster(playerID, manager)
	roster.AddUnit(rosterUnitID, "Regular Unit")

	unitTemplate := squads.UnitTemplate{
		Name:       "Regular Unit",
		GridWidth:  1,
		GridHeight: 1,
	}

	service.AssignRosterUnitToSquad(playerID, squadID, rosterUnitID, unitTemplate, 0, 0)

	// Validate (should fail - no leader)
	validation := service.ValidateSquad(squadID)

	if validation.Valid {
		t.Error("Squad without leader should not be valid")
	}

	if validation.UnitCount != 1 {
		t.Errorf("Expected 1 unit, got %d", validation.UnitCount)
	}

	if validation.HasLeader {
		t.Error("HasLeader should be false")
	}
}

// TestValidateSquad_Valid tests validation passes with units and leader
func TestValidateSquad_Valid(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create player with roster
	playerID := setupTestPlayerWithRoster(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Valid Squad")

	// Add unit
	rosterUnitEntity := manager.World.NewEntity()
	rosterUnitID := rosterUnitEntity.GetID()
	roster := squads.GetPlayerRoster(playerID, manager)
	roster.AddUnit(rosterUnitID, "Leader Unit")

	unitTemplate := squads.UnitTemplate{
		Name:       "Leader Unit",
		GridWidth:  1,
		GridHeight: 1,
	}

	assignResult := service.AssignRosterUnitToSquad(playerID, squadID, rosterUnitID, unitTemplate, 0, 0)
	if !assignResult.Success {
		t.Fatalf("Failed to assign unit: %s", assignResult.Error)
	}

	// Designate leader
	leaderResult := service.DesignateLeader(assignResult.PlacedUnitID)
	if !leaderResult.Success {
		t.Fatalf("Failed to designate leader: %s", leaderResult.Error)
	}

	// Validate (should pass)
	validation := service.ValidateSquad(squadID)

	if !validation.Valid {
		t.Errorf("Valid squad should pass validation: %s", validation.ErrorMsg)
	}

	if validation.UnitCount != 1 {
		t.Errorf("Expected 1 unit, got %d", validation.UnitCount)
	}

	if !validation.HasLeader {
		t.Error("HasLeader should be true")
	}
}

// TestUpdateSquadName tests updating squad name
func TestUpdateSquadName(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Original Name")

	// Update name
	success := service.UpdateSquadName(squadID, "New Name")

	if !success {
		t.Error("Should successfully update squad name")
	}

	// Verify name changed
	squadEntity := common.FindEntityByIDWithTag(manager, squadID, squads.SquadTag)
	if squadEntity == nil {
		t.Fatal("Squad entity not found")
	}

	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData == nil {
		t.Fatal("Squad data not found")
	}

	if squadData.Name != "New Name" {
		t.Errorf("Expected squad name 'New Name', got '%s'", squadData.Name)
	}
}

// TestUpdateSquadName_EmptyName tests updating with empty name
func TestUpdateSquadName_EmptyName(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Original Name")

	// Try to update with empty name
	success := service.UpdateSquadName(squadID, "")

	if success {
		t.Error("Should reject empty squad name")
	}
}

// TestSquadBuilderFlow tests complete squad building flow
func TestSquadBuilderFlow(t *testing.T) {
	manager := setupBuilderTestManager(t)
	service := NewSquadBuilderService(manager)

	// Create player with roster
	playerID := setupTestPlayerWithRoster(manager)

	// Create squad
	squadID := squads.CreateEmptySquad(manager, "Complete Squad")

	roster := squads.GetPlayerRoster(playerID, manager)

	// Add multiple units
	for i := 0; i < 3; i++ {
		rosterUnitEntity := manager.World.NewEntity()
		rosterUnitID := rosterUnitEntity.GetID()

		unitName := "Unit " + string(rune(i))
		roster.AddUnit(rosterUnitID, unitName)

		unitTemplate := squads.UnitTemplate{
			Name:       unitName,
			GridWidth:  1,
			GridHeight: 1,
		}

		col := i % 3
		row := i / 3

		result := service.AssignRosterUnitToSquad(playerID, squadID, rosterUnitID, unitTemplate, row, col)
		if !result.Success {
			t.Logf("Failed to assign unit %d: %s", i, result.Error)
		}

		// First unit becomes leader
		if i == 0 {
			leaderResult := service.DesignateLeader(result.PlacedUnitID)
			if !leaderResult.Success {
				t.Logf("Failed to designate leader: %s", leaderResult.Error)
			}
		}
	}

	// Update squad name
	service.UpdateSquadName(squadID, "Elite Squad")

	// Get capacity info
	info := service.GetCapacityInfo(squadID)
	if info == nil {
		t.Fatal("Capacity info is nil")
	}

	t.Logf("Final squad state - Units: 3, Used capacity: %f, Has leader: %v", info.UsedCapacity, info.HasLeader)

	// Validate squad
	validation := service.ValidateSquad(squadID)
	if !validation.Valid {
		t.Logf("Squad validation failed: %s", validation.ErrorMsg)
	} else {
		t.Log("Squad validated successfully")
	}
}
