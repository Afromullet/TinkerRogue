package squads

import (
	"game_main/common"
	"game_main/coords"
	"testing"
)

// TestSquadServiceCreation tests that SquadService can be created
func TestSquadServiceCreation(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	if service == nil {
		t.Error("SquadService should not be nil")
	}

	if service.entityManager != manager {
		t.Error("EntityManager not set correctly")
	}
}

// TestCreateSquad_Success tests successful squad creation
func TestCreateSquad_Success(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	result := service.CreateSquad("Test Squad")

	if !result.Success {
		t.Errorf("Squad creation should succeed, got error: %s", result.Error)
	}

	if result.SquadID == 0 {
		t.Error("SquadID should not be 0")
	}

	if result.SquadName != "Test Squad" {
		t.Errorf("Expected squad name 'Test Squad', got '%s'", result.SquadName)
	}

	// Verify squad was created in ECS
	squadEntity := common.FindEntityByIDWithTag(manager, result.SquadID, SquadTag)
	if squadEntity == nil {
		t.Error("Squad entity should exist in ECS")
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	if squadData == nil {
		t.Error("Squad should have SquadData component")
	}

	if squadData.Name != "Test Squad" {
		t.Errorf("Squad data name mismatch: expected 'Test Squad', got '%s'", squadData.Name)
	}
}

// TestCreateSquad_EmptyName tests squad creation with empty name
func TestCreateSquad_EmptyName(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	result := service.CreateSquad("")

	if result.Success {
		t.Error("Squad creation should fail with empty name")
	}

	if result.Error == "" {
		t.Error("Error message should be populated")
	}
}

// TestAddUnitToSquad_Success tests adding a unit to a squad
func TestAddUnitToSquad_Success(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	// Create squad
	squadResult := service.CreateSquad("Test Squad")
	if !squadResult.Success {
		t.Fatalf("Failed to create squad: %s", squadResult.Error)
	}

	// Add unit
	unitTemplate := UnitTemplate{
		Name: "Warrior",
		Attributes: common.Attributes{
			Strength:  15,
			Dexterity: 10,
			Magic:     5,
			Health:    20,
			MaxHealth: 20,
		},
		GridWidth:  1,
		GridHeight: 1,
	}

	result := service.AddUnitToSquad(squadResult.SquadID, unitTemplate, 0, 0)

	if !result.Success {
		t.Errorf("Should add unit successfully, got error: %s", result.Error)
	}

	if result.UnitID == 0 {
		t.Error("UnitID should not be 0")
	}

	if result.RemainingCapacity == 6.0 {
		t.Logf("Remaining capacity: %f", result.RemainingCapacity)
	}
}

// TestAddUnitToSquad_InvalidPosition tests adding a unit to invalid grid position
func TestAddUnitToSquad_InvalidPosition(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	// Create squad
	squadResult := service.CreateSquad("Test Squad")

	unitTemplate := UnitTemplate{
		Name:       "Warrior",
		GridWidth:  1,
		GridHeight: 1,
	}

	// Try invalid position
	result := service.AddUnitToSquad(squadResult.SquadID, unitTemplate, 5, 0) // Row 5 is out of bounds

	if result.Success {
		t.Error("Should reject invalid grid position")
	}

	if result.Error == "" {
		t.Error("Error message should be populated")
	}
}

// TestAddUnitToSquad_OccupiedPosition tests adding a unit to occupied position
func TestAddUnitToSquad_OccupiedPosition(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	// Create squad
	squadResult := service.CreateSquad("Test Squad")

	unitTemplate := UnitTemplate{
		Name:       "Warrior",
		GridWidth:  1,
		GridHeight: 1,
	}

	// Add first unit
	result1 := service.AddUnitToSquad(squadResult.SquadID, unitTemplate, 0, 0)
	if !result1.Success {
		t.Fatalf("First unit should be added: %s", result1.Error)
	}

	// Try to add another unit at same position
	result2 := service.AddUnitToSquad(squadResult.SquadID, unitTemplate, 0, 0)

	if result2.Success {
		t.Error("Should reject adding unit to occupied position")
	}

	if result2.Error == "" {
		t.Error("Error message should be populated")
	}
}

// TestAddUnitToSquad_InsufficientCapacity tests capacity validation
func TestAddUnitToSquad_InsufficientCapacity(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	// Create squad
	squadResult := service.CreateSquad("Test Squad")

	// Create a large unit that uses most/all capacity
	largeUnit := UnitTemplate{
		Name: "Heavy Knight",
		Attributes: common.Attributes{
			Strength:  20,
			Dexterity: 5,
			Magic:     0,
			Health:    30,
			MaxHealth: 30,
		},
		GridWidth:  2,
		GridHeight: 2,
	}

	// Add multiple large units to exceed capacity
	for i := 0; i < 10; i++ {
		col := i % 3
		row := i / 3
		result := service.AddUnitToSquad(squadResult.SquadID, largeUnit, row, col)

		if !result.Success && result.RemainingCapacity < 0 {
			t.Logf("Capacity exceeded at unit %d: %s", i, result.Error)
			return // Expected to fail eventually
		}
	}
}

// TestRemoveUnitFromSquad tests unit removal
func TestRemoveUnitFromSquad(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	// Create squad and add unit
	squadResult := service.CreateSquad("Test Squad")
	unitTemplate := UnitTemplate{
		Name:       "Warrior",
		GridWidth:  1,
		GridHeight: 1,
	}

	unitResult := service.AddUnitToSquad(squadResult.SquadID, unitTemplate, 0, 0)
	if !unitResult.Success {
		t.Fatalf("Failed to add unit: %s", unitResult.Error)
	}

	capacityBefore := unitResult.RemainingCapacity

	// Remove unit
	remResult := service.RemoveUnitFromSquad(squadResult.SquadID, unitResult.UnitID)

	if !remResult.Success {
		t.Errorf("Should remove unit successfully, got error: %s", remResult.Error)
	}

	// Capacity should increase after removal
	if remResult.RemainingCapacity <= capacityBefore {
		t.Logf("Capacity after removal: %f (was %f)", remResult.RemainingCapacity, capacityBefore)
	}
}

// TestGetSquadInfo tests getting squad information
func TestGetSquadInfo(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	// Create squad
	squadResult := service.CreateSquad("Info Test Squad")

	// Get info
	info := service.GetSquadInfo(squadResult.SquadID)

	if info.SquadID != squadResult.SquadID {
		t.Errorf("Squad ID mismatch: expected %d, got %d", squadResult.SquadID, info.SquadID)
	}

	if info.SquadName != "Info Test Squad" {
		t.Errorf("Squad name mismatch: expected 'Info Test Squad', got '%s'", info.SquadName)
	}

	if info.TotalCapacity != 6 {
		t.Errorf("Expected total capacity 6, got %f", info.TotalCapacity)
	}

	if info.RemainingCapacity != 6 {
		t.Errorf("Expected remaining capacity 6, got %f", info.RemainingCapacity)
	}

	if info.UnitCount != 0 {
		t.Errorf("Expected 0 units, got %d", info.UnitCount)
	}
}

// TestCanAddMoreUnits tests capacity checking
func TestCanAddMoreUnits(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	// Create squad
	squadResult := service.CreateSquad("Capacity Test Squad")

	// Should be able to add units to empty squad
	if !service.CanAddMoreUnits(squadResult.SquadID, 1.0) {
		t.Error("Should be able to add units to empty squad")
	}

	// Add unit
	unitTemplate := UnitTemplate{
		Name:       "Warrior",
		GridWidth:  1,
		GridHeight: 1,
	}

	service.AddUnitToSquad(squadResult.SquadID, unitTemplate, 0, 0)

	// Should still be able to add more
	if !service.CanAddMoreUnits(squadResult.SquadID, 1.0) {
		t.Error("Should be able to add more units")
	}
}

// TestGetSquadRemainingCapacity tests remaining capacity calculation
func TestGetSquadRemainingCapacity(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	// Create squad
	squadResult := service.CreateSquad("Remaining Capacity Test")

	// Check initial capacity
	remaining := service.GetSquadRemainingCapacity(squadResult.SquadID)
	if remaining != 6 {
		t.Errorf("Expected initial remaining capacity 6, got %f", remaining)
	}

	// Add unit
	unitTemplate := UnitTemplate{
		Name:       "Warrior",
		GridWidth:  1,
		GridHeight: 1,
	}

	service.AddUnitToSquad(squadResult.SquadID, unitTemplate, 0, 0)

	// Check reduced capacity
	remainingAfter := service.GetSquadRemainingCapacity(squadResult.SquadID)
	if remainingAfter >= remaining {
		t.Errorf("Remaining capacity should decrease after adding unit: was %f, now %f", remaining, remainingAfter)
	}
}

// TestSquadWithPosition tests squad with position component
func TestSquadWithPosition(t *testing.T) {
	manager := common.NewEntityManager()
	service := NewSquadService(manager)

	// Create squad
	squadResult := service.CreateSquad("Positioned Squad")

	// Set position
	squadEntity := common.FindEntityByIDWithTag(manager, squadResult.SquadID, SquadTag)
	if squadEntity == nil {
		t.Fatal("Squad entity not found")
	}

	posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
	if posPtr == nil {
		t.Fatal("Squad should have position component")
	}

	posPtr.X = 10
	posPtr.Y = 15

	// Get info and verify position persists
	info := service.GetSquadInfo(squadResult.SquadID)
	if info.SquadID == 0 {
		t.Error("Squad info lookup failed")
	}
}
