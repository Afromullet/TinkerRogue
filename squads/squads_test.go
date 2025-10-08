package squads

import (
	"game_main/common"
	"game_main/entitytemplates"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// TEST SETUP HELPERS
// ========================================

// setupTestSquadManager creates a fresh SquadECSManager for testing
func setupTestSquadManager(t *testing.T) *SquadECSManager {
	manager := NewSquadECSManager()
	InitSquadComponents(*manager)
	InitSquadTags(*manager)

	// Initialize common.PositionComponent for CreateEmptySquad
	// (Squad entities need this to track their position on the world map)
	if common.PositionComponent == nil {
		common.PositionComponent = manager.Manager.NewComponent()
	}

	return manager
}

// createTestJSONMonster creates a JSONMonster for testing
func createTestJSONMonster(name string, width, height int, role string) entitytemplates.JSONMonster {
	return entitytemplates.JSONMonster{
		Name:      name,
		ImageName: "test.png", // Not used in tests
		Attributes: entitytemplates.JSONAttributes{
			MaxHealth:         100,
			AttackBonus:       5,
			BaseArmorClass:    10,
			BaseProtection:    5,
			BaseDodgeChance:   0.1,
			BaseMovementSpeed: 5,
			DamageBonus:       2,
		},
		Width:         width,
		Height:        height,
		Role:          role,
		TargetMode:    "row",
		TargetRows:    []int{0},
		IsMultiTarget: false,
		MaxTargets:    1,
		TargetCells:   nil,
	}
}

// ========================================
// CreateUnitEntity TESTS
// ========================================

func TestCreateUnitEntity_SingleCell(t *testing.T) {
	manager := setupTestSquadManager(t)

	jsonMonster := createTestJSONMonster("Warrior", 1, 1, "Tank")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	entity, err := CreateUnitEntity(manager, unit)
	if err != nil {
		t.Fatalf("CreateUnitEntity failed: %v", err)
	}

	if entity == nil {
		t.Fatal("Expected entity to be created, got nil")
	}

	// Verify GridPositionComponent
	if !entity.HasComponent(GridPositionComponent) {
		t.Fatal("Entity missing GridPositionComponent")
	}

	gridPos := common.GetComponentType[*GridPositionData](entity, GridPositionComponent)
	if gridPos.Width != 1 || gridPos.Height != 1 {
		t.Errorf("Expected 1x1 unit, got %dx%d", gridPos.Width, gridPos.Height)
	}

	// Verify UnitRoleComponent
	if !entity.HasComponent(UnitRoleComponent) {
		t.Fatal("Entity missing UnitRoleComponent")
	}

	roleData := common.GetComponentType[*UnitRoleData](entity, UnitRoleComponent)
	if roleData.Role != RoleTank {
		t.Errorf("Expected role Tank, got %v", roleData.Role)
	}

	// Verify TargetRowComponent
	if !entity.HasComponent(TargetRowComponent) {
		t.Fatal("Entity missing TargetRowComponent")
	}
}

func TestCreateUnitEntity_MultiCell_2x2(t *testing.T) {
	manager := setupTestSquadManager(t)

	jsonMonster := createTestJSONMonster("Giant", 2, 2, "Tank")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	entity, err := CreateUnitEntity(manager, unit)
	if err != nil {
		t.Fatalf("CreateUnitEntity failed for 2x2 unit: %v", err)
	}

	gridPos := common.GetComponentType[*GridPositionData](entity, GridPositionComponent)
	if gridPos.Width != 2 || gridPos.Height != 2 {
		t.Errorf("Expected 2x2 unit, got %dx%d", gridPos.Width, gridPos.Height)
	}

	// Verify it occupies 4 cells
	cells := gridPos.GetOccupiedCells()
	if len(cells) != 4 {
		t.Errorf("Expected 4 occupied cells for 2x2 unit, got %d", len(cells))
	}
}

func TestCreateUnitEntity_MultiCell_1x3(t *testing.T) {
	manager := setupTestSquadManager(t)

	jsonMonster := createTestJSONMonster("Cavalry", 1, 3, "DPS")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	entity, err := CreateUnitEntity(manager, unit)
	if err != nil {
		t.Fatalf("CreateUnitEntity failed for 1x3 unit: %v", err)
	}

	gridPos := common.GetComponentType[*GridPositionData](entity, GridPositionComponent)
	if gridPos.Width != 1 || gridPos.Height != 3 {
		t.Errorf("Expected 1x3 unit, got %dx%d", gridPos.Width, gridPos.Height)
	}

	// Verify it occupies 3 cells
	cells := gridPos.GetOccupiedCells()
	if len(cells) != 3 {
		t.Errorf("Expected 3 occupied cells for 1x3 unit, got %d", len(cells))
	}
}

// ========================================
// AddUnitToSquad TESTS
// ========================================

func TestAddUnitToSquad_SingleCell_ValidPosition(t *testing.T) {
	manager := setupTestSquadManager(t)

	// Create squad
	CreateEmptySquad(manager, "Test Squad")

	// Get the squad entity
	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	if squadID == 0 {
		t.Fatal("Failed to create squad")
	}

	// Create and add unit
	jsonMonster := createTestJSONMonster("Warrior", 1, 1, "Tank")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit, 0, 0)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed: %v", err)
	}

	// Verify unit was added
	unitIDs := GetUnitIDsAtGridPosition(squadID, 0, 0, manager)
	if len(unitIDs) != 1 {
		t.Errorf("Expected 1 unit at position (0,0), got %d", len(unitIDs))
	}
}

func TestAddUnitToSquad_SingleCell_AllPositions(t *testing.T) {
	manager := setupTestSquadManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Test adding units to all 9 positions
	expectedUnits := 0
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			jsonMonster := createTestJSONMonster("Unit", 1, 1, "DPS")
			unit, err := CreateUnitTemplates(jsonMonster)
			if err != nil {
				t.Fatalf("CreateUnitTemplates failed: %v", err)
			}

			err = AddUnitToSquad(squadID, manager, unit, row, col)
			if err != nil {
				t.Fatalf("AddUnitToSquad failed at (%d,%d): %v", row, col, err)
			}
			expectedUnits++

			// Verify unit exists at this position
			unitIDs := GetUnitIDsAtGridPosition(squadID, row, col, manager)
			if len(unitIDs) != 1 {
				t.Errorf("Expected 1 unit at (%d,%d), got %d", row, col, len(unitIDs))
			}
		}
	}

	// Verify total units
	totalUnits := 0
	for _, result := range manager.Manager.Query(SquadMemberTag) {
		memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
		if memberData.SquadID == squadID {
			totalUnits++
		}
	}

	if totalUnits != expectedUnits {
		t.Errorf("Expected %d total units, got %d", expectedUnits, totalUnits)
	}
}

func TestAddUnitToSquad_MultiCell_2x2_TopLeft(t *testing.T) {
	manager := setupTestSquadManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add 2x2 unit at top-left
	jsonMonster := createTestJSONMonster("Giant", 2, 2, "Tank")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit, 0, 0)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed for 2x2 unit: %v", err)
	}

	// Verify unit occupies all 4 cells
	expectedCells := [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}}
	for _, cell := range expectedCells {
		unitIDs := GetUnitIDsAtGridPosition(squadID, cell[0], cell[1], manager)
		if len(unitIDs) != 1 {
			t.Errorf("Expected 1 unit at (%d,%d), got %d", cell[0], cell[1], len(unitIDs))
		}
	}
}

func TestAddUnitToSquad_MultiCell_1x3_LeftColumn(t *testing.T) {
	manager := setupTestSquadManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add 1x3 unit (3 rows tall) at left column
	jsonMonster := createTestJSONMonster("Cavalry", 1, 3, "DPS")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit, 0, 0)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed for 1x3 unit: %v", err)
	}

	// Verify unit occupies all 3 cells vertically
	expectedCells := [][2]int{{0, 0}, {1, 0}, {2, 0}}
	for _, cell := range expectedCells {
		unitIDs := GetUnitIDsAtGridPosition(squadID, cell[0], cell[1], manager)
		if len(unitIDs) != 1 {
			t.Errorf("Expected 1 unit at (%d,%d), got %d", cell[0], cell[1], len(unitIDs))
		}
	}
}

func TestAddUnitToSquad_MultiCell_3x1_TopRow(t *testing.T) {
	manager := setupTestSquadManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add 3x1 unit (3 cols wide) at top row
	jsonMonster := createTestJSONMonster("Wall", 3, 1, "Tank")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit, 0, 0)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed for 3x1 unit: %v", err)
	}

	// Verify unit occupies all 3 cells horizontally
	expectedCells := [][2]int{{0, 0}, {0, 1}, {0, 2}}
	for _, cell := range expectedCells {
		unitIDs := GetUnitIDsAtGridPosition(squadID, cell[0], cell[1], manager)
		if len(unitIDs) != 1 {
			t.Errorf("Expected 1 unit at (%d,%d), got %d", cell[0], cell[1], len(unitIDs))
		}
	}
}

func TestAddUnitToSquad_Collision_SingleCellOverlap(t *testing.T) {
	manager := setupTestSquadManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add first unit
	jsonMonster1 := createTestJSONMonster("Warrior1", 1, 1, "Tank")
	unit1, err := CreateUnitTemplates(jsonMonster1)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit1, 1, 1)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed for first unit: %v", err)
	}

	// Try to add second unit at same position
	jsonMonster2 := createTestJSONMonster("Warrior2", 1, 1, "Tank")
	unit2, err := CreateUnitTemplates(jsonMonster2)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit2, 1, 1)
	if err == nil {
		t.Fatal("Expected error when adding unit to occupied position, got nil")
	}
}

func TestAddUnitToSquad_Collision_MultiCellOverlap(t *testing.T) {
	manager := setupTestSquadManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add 2x2 unit at top-left
	jsonMonster1 := createTestJSONMonster("Giant", 2, 2, "Tank")
	unit1, err := CreateUnitTemplates(jsonMonster1)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit1, 0, 0)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed for 2x2 unit: %v", err)
	}

	// Try to add 1x1 unit at (0,0) - should fail (overlaps giant)
	jsonMonster2 := createTestJSONMonster("Warrior", 1, 1, "DPS")
	unit2, err := CreateUnitTemplates(jsonMonster2)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit2, 0, 0)
	if err == nil {
		t.Fatal("Expected error when adding unit to position occupied by 2x2 unit, got nil")
	}

	// Try to add 1x1 unit at (1,1) - should fail (overlaps giant)
	err = AddUnitToSquad(squadID, manager, unit2, 1, 1)
	if err == nil {
		t.Fatal("Expected error when adding unit to position occupied by 2x2 unit, got nil")
	}
}

func TestAddUnitToSquad_InvalidPosition_RowTooLarge(t *testing.T) {
	manager := setupTestSquadManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	jsonMonster := createTestJSONMonster("Warrior", 1, 1, "Tank")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	// Pass invalid row as parameter
	err = AddUnitToSquad(squadID, manager, unit, 3, 0)
	if err == nil {
		t.Fatal("Expected error for row=3, got nil")
	}
}

func TestAddUnitToSquad_InvalidPosition_NegativeCol(t *testing.T) {
	manager := setupTestSquadManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	jsonMonster := createTestJSONMonster("Warrior", 1, 1, "Tank")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	// Pass invalid col as parameter
	err = AddUnitToSquad(squadID, manager, unit, 0, -1)
	if err == nil {
		t.Fatal("Expected error for negative col, got nil")
	}
}

func TestAddUnitToSquad_MixedSizes_NoOverlap(t *testing.T) {
	manager := setupTestSquadManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add 2x2 unit at top-left (occupies [0,0], [0,1], [1,0], [1,1])
	jsonGiant := createTestJSONMonster("Giant", 2, 2, "Tank")
	giant, err := CreateUnitTemplates(jsonGiant)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, giant, 0, 0)
	if err != nil {
		t.Fatalf("Failed to add 2x2 unit: %v", err)
	}

	// Add 1x1 unit at (0,2) - top-right corner
	jsonArcher := createTestJSONMonster("Archer", 1, 1, "DPS")
	archer, err := CreateUnitTemplates(jsonArcher)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, archer, 0, 2)
	if err != nil {
		t.Fatalf("Failed to add 1x1 unit at (0,2): %v", err)
	}

	// Add 1x1 unit at (2,0) - bottom-left corner
	jsonMage := createTestJSONMonster("Mage", 1, 1, "Support")
	mage, err := CreateUnitTemplates(jsonMage)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, mage, 2, 0)
	if err != nil {
		t.Fatalf("Failed to add 1x1 unit at (2,0): %v", err)
	}

	// Verify all units present
	totalUnits := 0
	for _, result := range manager.Manager.Query(SquadMemberTag) {
		memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
		if memberData.SquadID == squadID {
			totalUnits++
		}
	}

	if totalUnits != 3 {
		t.Errorf("Expected 3 units in squad, got %d", totalUnits)
	}
}

func TestAddUnitToSquad_VerifySquadMemberComponent(t *testing.T) {
	manager := setupTestSquadManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.Manager.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	jsonMonster := createTestJSONMonster("Warrior", 1, 1, "Tank")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit, 1, 1)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed: %v", err)
	}

	// Find the unit entity
	var unitEntity *ecs.Entity
	for _, result := range manager.Manager.Query(SquadMemberTag) {
		memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
		if memberData.SquadID == squadID {
			unitEntity = result.Entity
			break
		}
	}

	if unitEntity == nil {
		t.Fatal("Unit entity not found")
	}

	// Verify SquadMemberComponent
	if !unitEntity.HasComponent(SquadMemberComponent) {
		t.Fatal("Unit missing SquadMemberComponent")
	}

	memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
	if memberData.SquadID != squadID {
		t.Errorf("Expected SquadID %d, got %d", squadID, memberData.SquadID)
	}
}
