package squads

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
	testfx "game_main/testing"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// TEST SETUP HELPERS
// ========================================

// setupTestManager creates a manager with squad system initialized.
// Uses the shared testfx.NewTestEntityManager() fixture and adds squad-specific init.
func setupTestManager(t *testing.T) *common.EntityManager {
	manager := testfx.NewTestEntityManager()
	if err := InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}
	return manager
}

// createTestJSONMonster creates a JSONMonster for testing
func createTestJSONMonster(name string, width, height int, role string) entitytemplates.JSONMonster {
	return entitytemplates.JSONMonster{
		Name:      name,
		ImageName: "test.png", // Not used in tests
		Attributes: entitytemplates.JSONAttributes{
			Strength:   10, // 40 HP (20 + 10*2), 7 damage (10/2 + 2*2), 6 resistance (10/4 + 2*2)
			Dexterity:  20, // 100% hit (80 + 20*2, capped), 10% crit (20/2), 6% dodge (20/3)
			Magic:      0,  // No magic abilities
			Leadership: 0,  // No squad leadership
			Armor:      2,  // Contributes to physical resistance
			Weapon:     2,  // Contributes to physical damage
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

// createLowCostTestMonster creates a unit with minimal capacity cost for visualization tests
// Cost = (Strength + Weapon + Armor) / 5 = (1 + 0 + 0) / 5 = 0.2 per unit
// This allows fitting multiple units in a single squad for visualization testing
func createLowCostTestMonster(name string, width, height int, role string) entitytemplates.JSONMonster {
	return entitytemplates.JSONMonster{
		Name:      name,
		ImageName: "test.png",
		Attributes: entitytemplates.JSONAttributes{
			Strength:   1,  // Minimal - only affects HP and damage
			Dexterity:  20, // Keep dexterity for combat mechanics
			Magic:      0,
			Leadership: 0,
			Armor:      0,  // No armor to minimize capacity cost
			Weapon:     0,  // No weapon to minimize capacity cost
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

// CreateHighCapacitySquad creates a squad with sufficient capacity for visualization tests
// This is needed because visualization tests try to add multiple units,
// and the default squad capacity (6) can only fit 2 units with normal stats
// Note: Capacity is based on leader's Leadership stat (GetUnitCapacity = 6 + Leadership/3, capped at 9)
// To work around this, we just add a dummy leader with max capacity
func CreateHighCapacitySquad(manager *common.EntityManager, squadName string, capacity int) ecs.EntityID {
	squadEntity := manager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(SquadComponent, &SquadData{
		SquadID:       squadID,
		Name:          squadName,
		Morale:        100,
		TurnCount:     0,
		MaxUnits:      9,
		UsedCapacity:  0.0,
		TotalCapacity: 9, // Will be recalculated based on leader
	})

	squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})

	// Create a dummy leader unit with max capacity to enable adding multiple units
	// Leadership of 9 gives capacity = 6 + (9/3) = 9 (capped)
	leaderEntity := manager.World.NewEntity()

	leaderEntity.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: squadID})
	leaderEntity.AddComponent(common.NameComponent, &common.Name{NameStr: "Leader"})

	// Create attributes for the leader with high Leadership
	leaderAttr := common.NewAttributes(
		1,  // Minimal strength
		0,  // No dexterity needed
		0,  // No magic
		9,  // Max leadership for capacity
		0,  // No armor
		0,  // No weapon
	)
	leaderEntity.AddComponent(common.AttributeComponent, &leaderAttr)

	// Add leader component with abilities
	leaderEntity.AddComponent(LeaderComponent, &LeaderData{
		Leadership: 9,
		Experience: 0,
	})
	leaderEntity.AddComponent(AbilitySlotComponent, &AbilitySlotData{Slots: [4]AbilitySlot{}})
	leaderEntity.AddComponent(CooldownTrackerComponent, &CooldownTrackerData{
		Cooldowns:    [4]int{0, 0, 0, 0},
		MaxCooldowns: [4]int{0, 0, 0, 0},
	})

	// Add GridPositionComponent at invalid position so it doesn't interfere with grid
	// and UnitRoleComponent for visualization compatibility
	leaderEntity.AddComponent(GridPositionComponent, &GridPositionData{
		AnchorRow: -1, // Invalid position - won't appear in 3x3 grid
		AnchorCol: -1,
		Width:     1,
		Height:    1,
	})
	leaderEntity.AddComponent(UnitRoleComponent, &UnitRoleData{
		Role: RoleTank, // Arbitrary role for the invisible leader
	})

	// Update squad capacity to reflect the leader
	UpdateSquadCapacity(squadID, manager)

	return squadID
}

// ========================================
// CreateUnitEntity TESTS
// ========================================

func TestCreateUnitEntity_SingleCell(t *testing.T) {
	manager := setupTestManager(t)

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
	manager := setupTestManager(t)

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
	manager := setupTestManager(t)

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
	manager := setupTestManager(t)

	// Create squad
	CreateEmptySquad(manager, "Test Squad")

	// Get the squad entity
	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
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
	manager := setupTestManager(t)

	// Use high capacity squad to fit 9 units (0.2 capacity each = 1.8 total)
	squadID := CreateHighCapacitySquad(manager, "Test Squad", 9)

	// Test adding units to all 9 positions
	expectedUnits := 1 // Start with 1 because CreateHighCapacitySquad adds a leader
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			// Use low-cost units to fit in capacity
			jsonMonster := createLowCostTestMonster("Unit", 1, 1, "DPS")
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
	for _, result := range manager.World.Query(SquadMemberTag) {
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
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
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
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
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
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
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
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
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
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
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
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
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
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
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
	manager := setupTestManager(t)

	// Use high capacity squad to fit multiple units
	squadID := CreateHighCapacitySquad(manager, "Test Squad", 9)

	// Add 2x2 unit at top-left (occupies [0,0], [0,1], [1,0], [1,1])
	// Use low-cost units to fit in capacity
	jsonGiant := createLowCostTestMonster("Giant", 2, 2, "Tank")
	giant, err := CreateUnitTemplates(jsonGiant)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, giant, 0, 0)
	if err != nil {
		t.Fatalf("Failed to add 2x2 unit: %v", err)
	}

	// Add 1x1 unit at (0,2) - top-right corner
	jsonArcher := createLowCostTestMonster("Archer", 1, 1, "DPS")
	archer, err := CreateUnitTemplates(jsonArcher)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, archer, 0, 2)
	if err != nil {
		t.Fatalf("Failed to add 1x1 unit at (0,2): %v", err)
	}

	// Add 1x1 unit at (2,0) - bottom-left corner
	jsonMage := createLowCostTestMonster("Mage", 1, 1, "Support")
	mage, err := CreateUnitTemplates(jsonMage)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, mage, 2, 0)
	if err != nil {
		t.Fatalf("Failed to add 1x1 unit at (2,0): %v", err)
	}

	// Verify all units present (including leader from CreateHighCapacitySquad)
	totalUnits := 0
	for _, result := range manager.World.Query(SquadMemberTag) {
		memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
		if memberData.SquadID == squadID {
			totalUnits++
		}
	}

	if totalUnits != 4 {
		t.Errorf("Expected 4 units (3 visible + 1 leader), got %d", totalUnits)
	}
}

func TestAddUnitToSquad_VerifySquadMemberComponent(t *testing.T) {
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
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
	for _, result := range manager.World.Query(SquadMemberTag) {
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

// ========================================
// VISUALIZATION TESTS
// ========================================

func TestVisualizeSquad_EmptySquad(t *testing.T) {
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Empty Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	if squadID == 0 {
		t.Fatal("Failed to create squad")
	}

	// Visualize empty squad
	output := VisualizeSquad(squadID, manager)

	// Verify output contains squad name
	if !contains(output, "Empty Squad") {
		t.Errorf("Expected squad name in output, got:\n%s", output)
	}

	// Verify output indicates no units
	if !contains(output, "No units in squad") {
		t.Errorf("Expected 'No units in squad' message, got:\n%s", output)
	}

	// Verify grid is present (should show all Empty cells)
	if !contains(output, "Empty") {
		t.Errorf("Expected empty cells in grid, got:\n%s", output)
	}

	t.Logf("Empty Squad Visualization:\n%s", output)
}

func TestVisualizeSquad_SingleUnit_1x1(t *testing.T) {
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Single Unit Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add single warrior at center
	jsonMonster := createTestJSONMonster("Warrior", 1, 1, "Tank")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit, 1, 1)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed: %v", err)
	}

	// Visualize squad
	output := VisualizeSquad(squadID, manager)

	// Verify unit appears in grid
	unitIDs := GetUnitIDsAtGridPosition(squadID, 1, 1, manager)
	if len(unitIDs) != 1 {
		t.Fatalf("Expected 1 unit at (1,1), got %d", len(unitIDs))
	}

	expectedID := fmt.Sprintf("%d", unitIDs[0])
	if !contains(output, expectedID) {
		t.Errorf("Expected unit ID %s in output, got:\n%s", expectedID, output)
	}

	// Verify unit details section
	if !contains(output, "Unit Details:") {
		t.Errorf("Expected 'Unit Details:' section, got:\n%s", output)
	}

	if !contains(output, "Tank") {
		t.Errorf("Expected role 'Tank' in unit details, got:\n%s", output)
	}

	t.Logf("Single Unit Visualization:\n%s", output)
}

func TestVisualizeSquad_MultiCell_2x2_Giant(t *testing.T) {
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Giant Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add 2x2 giant at top-left
	jsonMonster := createTestJSONMonster("Giant", 2, 2, "Tank")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit, 0, 0)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed: %v", err)
	}

	// Visualize squad
	output := VisualizeSquad(squadID, manager)

	// Get the unit ID
	unitIDs := GetUnitIDsAtGridPosition(squadID, 0, 0, manager)
	if len(unitIDs) != 1 {
		t.Fatalf("Expected 1 unit at (0,0), got %d", len(unitIDs))
	}

	expectedID := fmt.Sprintf("%d", unitIDs[0])

	// Verify same ID appears in all 4 cells (0,0), (0,1), (1,0), (1,1)
	expectedCells := [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}}
	for _, cell := range expectedCells {
		ids := GetUnitIDsAtGridPosition(squadID, cell[0], cell[1], manager)
		if len(ids) != 1 || fmt.Sprintf("%d", ids[0]) != expectedID {
			t.Errorf("Expected unit ID %s at cell (%d,%d)", expectedID, cell[0], cell[1])
		}
	}

	// Verify size info in unit details
	if !contains(output, "Size 2x2") {
		t.Errorf("Expected 'Size 2x2' in unit details, got:\n%s", output)
	}

	t.Logf("2x2 Giant Visualization:\n%s", output)
}

func TestVisualizeSquad_MultiCell_1x3_Cavalry(t *testing.T) {
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Cavalry Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add 1x3 cavalry (3 rows tall, 1 col wide)
	jsonMonster := createTestJSONMonster("Cavalry", 1, 3, "DPS")
	unit, err := CreateUnitTemplates(jsonMonster)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	err = AddUnitToSquad(squadID, manager, unit, 0, 0)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed: %v", err)
	}

	// Visualize squad
	output := VisualizeSquad(squadID, manager)

	// Verify unit appears in all 3 rows of left column
	expectedCells := [][2]int{{0, 0}, {1, 0}, {2, 0}}
	unitIDs := GetUnitIDsAtGridPosition(squadID, 0, 0, manager)
	if len(unitIDs) != 1 {
		t.Fatalf("Expected 1 unit, got %d", len(unitIDs))
	}

	expectedID := fmt.Sprintf("%d", unitIDs[0])
	for _, cell := range expectedCells {
		ids := GetUnitIDsAtGridPosition(squadID, cell[0], cell[1], manager)
		if len(ids) != 1 || fmt.Sprintf("%d", ids[0]) != expectedID {
			t.Errorf("Expected unit ID %s at cell (%d,%d)", expectedID, cell[0], cell[1])
		}
	}

	// Verify size info
	if !contains(output, "Size 1x3") {
		t.Errorf("Expected 'Size 1x3' in unit details, got:\n%s", output)
	}

	t.Logf("1x3 Cavalry Visualization:\n%s", output)
}

func TestVisualizeSquad_FullFormation_MixedUnits(t *testing.T) {
	manager := setupTestManager(t)

	// Use high capacity squad for multiple units (8 units * 2.8 capacity each = 22.4 needed)
	squadID := CreateHighCapacitySquad(manager, "Mixed Formation", 25)

	// Front row: Tank (0,0), Tank (0,1), Archer (0,2)
	tankJSON := createLowCostTestMonster("Tank", 1, 1, "Tank")
	tank1, _ := CreateUnitTemplates(tankJSON)
	AddUnitToSquad(squadID, manager, tank1, 0, 0)

	tank2, _ := CreateUnitTemplates(tankJSON)
	AddUnitToSquad(squadID, manager, tank2, 0, 1)

	archerJSON := createLowCostTestMonster("Archer", 1, 1, "DPS")
	archer1, _ := CreateUnitTemplates(archerJSON)
	AddUnitToSquad(squadID, manager, archer1, 0, 2)

	// Middle row: Warrior (1,0), empty, Warrior (1,2)
	warriorJSON := createLowCostTestMonster("Warrior", 1, 1, "DPS")
	warrior1, _ := CreateUnitTemplates(warriorJSON)
	AddUnitToSquad(squadID, manager, warrior1, 1, 0)

	warrior2, _ := CreateUnitTemplates(warriorJSON)
	AddUnitToSquad(squadID, manager, warrior2, 1, 2)

	// Back row: Mage (2,0), Mage (2,1), Mage (2,2)
	mageJSON := createLowCostTestMonster("Mage", 1, 1, "Support")
	mage1, _ := CreateUnitTemplates(mageJSON)
	AddUnitToSquad(squadID, manager, mage1, 2, 0)

	mage2, _ := CreateUnitTemplates(mageJSON)
	AddUnitToSquad(squadID, manager, mage2, 2, 1)

	mage3, _ := CreateUnitTemplates(mageJSON)
	AddUnitToSquad(squadID, manager, mage3, 2, 2)

	// Visualize squad
	output := VisualizeSquad(squadID, manager)

	// Verify 8 visible units present plus 1 invisible leader (1 empty cell at 1,1)
	totalUnits := len(GetUnitIDsInSquad(squadID, manager))
	if totalUnits != 9 {
		t.Errorf("Expected 9 units (8 visible + 1 leader), got %d", totalUnits)
	}

	// Verify empty cell at (1,1)
	emptyIDs := GetUnitIDsAtGridPosition(squadID, 1, 1, manager)
	if len(emptyIDs) != 0 {
		t.Errorf("Expected empty cell at (1,1), got %d units", len(emptyIDs))
	}

	// Verify all roles appear
	if !contains(output, "Tank") {
		t.Errorf("Expected Tank role in output")
	}
	if !contains(output, "DPS") {
		t.Errorf("Expected DPS role in output")
	}
	if !contains(output, "Support") {
		t.Errorf("Expected Support role in output")
	}

	t.Logf("Mixed Formation Visualization:\n%s", output)
}

func TestVisualizeSquad_ComplexFormation_MultiCellUnits(t *testing.T) {
	manager := setupTestManager(t)

	// Use high capacity squad for 4 units (4 * 2.8 = 11.2 needed)
	squadID := CreateHighCapacitySquad(manager, "Complex Formation", 15)

	// 2x2 giant at top-left (occupies [0,0], [0,1], [1,0], [1,1])
	giantJSON := createLowCostTestMonster("Giant", 2, 2, "Tank")
	giant, _ := CreateUnitTemplates(giantJSON)
	AddUnitToSquad(squadID, manager, giant, 0, 0)

	// 1x1 archer at top-right (0,2)
	archerJSON := createLowCostTestMonster("Archer", 1, 1, "DPS")
	archer, _ := CreateUnitTemplates(archerJSON)
	AddUnitToSquad(squadID, manager, archer, 0, 2)

	// 1x1 mage at middle-right (1,2)
	mageJSON := createLowCostTestMonster("Mage", 1, 1, "Support")
	mage, _ := CreateUnitTemplates(mageJSON)
	AddUnitToSquad(squadID, manager, mage, 1, 2)

	// 3x1 wall at bottom (occupies [2,0], [2,1], [2,2])
	wallJSON := createLowCostTestMonster("Wall", 3, 1, "Tank")
	wall, _ := CreateUnitTemplates(wallJSON)
	AddUnitToSquad(squadID, manager, wall, 2, 0)

	// Visualize squad
	output := VisualizeSquad(squadID, manager)

	// Verify 4 distinct units plus 1 invisible leader
	totalUnits := len(GetUnitIDsInSquad(squadID, manager))
	if totalUnits != 5 {
		t.Errorf("Expected 5 units (4 visible + 1 leader), got %d", totalUnits)
	}

	// Verify giant occupies 4 cells
	giantIDs := GetUnitIDsAtGridPosition(squadID, 0, 0, manager)
	if len(giantIDs) != 1 {
		t.Fatalf("Expected 1 giant unit")
	}
	giantID := giantIDs[0]
	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			ids := GetUnitIDsAtGridPosition(squadID, r, c, manager)
			if len(ids) != 1 || ids[0] != giantID {
				t.Errorf("Expected giant at (%d,%d)", r, c)
			}
		}
	}

	// Verify wall occupies 3 cells
	wallIDs := GetUnitIDsAtGridPosition(squadID, 2, 0, manager)
	if len(wallIDs) != 1 {
		t.Fatalf("Expected 1 wall unit")
	}
	wallID := wallIDs[0]
	for c := 0; c < 3; c++ {
		ids := GetUnitIDsAtGridPosition(squadID, 2, c, manager)
		if len(ids) != 1 || ids[0] != wallID {
			t.Errorf("Expected wall at (2,%d)", c)
		}
	}

	t.Logf("Complex Formation Visualization:\n%s", output)
}

func TestVisualizeSquad_NonExistentSquad(t *testing.T) {
	manager := setupTestManager(t)

	// Try to visualize squad that doesn't exist
	output := VisualizeSquad(999999, manager)

	if !contains(output, "not found") {
		t.Errorf("Expected 'not found' message for non-existent squad, got:\n%s", output)
	}

	t.Logf("Non-existent Squad Output:\n%s", output)
}

func TestVisualizeSquad_GridBoundaries(t *testing.T) {
	manager := setupTestManager(t)

	// Use high capacity squad for 4 corner units (4 * 2.8 = 11.2 needed)
	squadID := CreateHighCapacitySquad(manager, "Boundary Test", 15)

	// Test all corner positions
	corners := [][2]int{{0, 0}, {0, 2}, {2, 0}, {2, 2}}
	for i, corner := range corners {
		unitName := fmt.Sprintf("Corner%d", i)
		jsonMonster := createLowCostTestMonster(unitName, 1, 1, "DPS")
		unit, _ := CreateUnitTemplates(jsonMonster)
		err := AddUnitToSquad(squadID, manager, unit, corner[0], corner[1])
		if err != nil {
			t.Fatalf("Failed to add unit at corner (%d,%d): %v", corner[0], corner[1], err)
		}
	}

	// Visualize
	output := VisualizeSquad(squadID, manager)

	// Verify 4 visible units plus 1 invisible leader
	totalUnits := len(GetUnitIDsInSquad(squadID, manager))
	if totalUnits != 5 {
		t.Errorf("Expected 5 units (4 corners + 1 leader), got %d", totalUnits)
	}

	// Verify center is empty
	centerIDs := GetUnitIDsAtGridPosition(squadID, 1, 1, manager)
	if len(centerIDs) != 0 {
		t.Errorf("Expected empty center cell, got %d units", len(centerIDs))
	}

	t.Logf("Grid Boundaries Visualization:\n%s", output)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ========================================
// MOVEMENT SPEED TESTS
// ========================================

func TestGetSquadMovementSpeed_SingleUnit(t *testing.T) {
	manager := setupTestManager(t)
	CreateEmptySquad(manager, "Test Squad")

	// Get the squad entity
	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Create unit with movement speed 5
	jsonMonster := createTestJSONMonster("FastUnit", 1, 1, "DPS")
	jsonMonster.MovementSpeed = 5
	unit, _ := CreateUnitTemplates(jsonMonster)

	err := AddUnitToSquad(squadID, manager, unit, 0, 0)
	if err != nil {
		t.Fatalf("Failed to add unit: %v", err)
	}

	speed := GetSquadMovementSpeed(squadID, manager)
	if speed != 5 {
		t.Errorf("Expected squad speed 5, got %d", speed)
	}
}

func TestGetSquadMovementSpeed_MultipleUnits_ReturnsMinimum(t *testing.T) {
	manager := setupTestManager(t)
	CreateEmptySquad(manager, "Test Squad")

	// Get the squad entity
	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add units with different speeds (only 2 to avoid capacity issues)
	speeds := []int{5, 3}
	for i, speed := range speeds {
		unitName := fmt.Sprintf("Unit%d", i)
		jsonMonster := createTestJSONMonster(unitName, 1, 1, "DPS")
		jsonMonster.MovementSpeed = speed
		unit, _ := CreateUnitTemplates(jsonMonster)

		col := i
		err := AddUnitToSquad(squadID, manager, unit, 0, col)
		if err != nil {
			t.Fatalf("Failed to add unit %d: %v", i, err)
		}
	}

	// Squad should move at speed of slowest unit (3)
	speed := GetSquadMovementSpeed(squadID, manager)
	if speed != 3 {
		t.Errorf("Expected squad speed 3 (minimum), got %d", speed)
	}
}

func TestGetSquadMovementSpeed_DeadUnitsIgnored(t *testing.T) {
	manager := setupTestManager(t)
	CreateEmptySquad(manager, "Test Squad")

	// Get the squad entity
	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add two units: one slow (speed 2), one fast (speed 5)
	jsonMonster1 := createTestJSONMonster("SlowUnit", 1, 1, "DPS")
	jsonMonster1.MovementSpeed = 2
	unit1, _ := CreateUnitTemplates(jsonMonster1)
	AddUnitToSquad(squadID, manager, unit1, 0, 0)

	jsonMonster2 := createTestJSONMonster("FastUnit", 1, 1, "DPS")
	jsonMonster2.MovementSpeed = 5
	unit2, _ := CreateUnitTemplates(jsonMonster2)
	AddUnitToSquad(squadID, manager, unit2, 0, 1)

	// Initially, squad moves at speed 2 (slowest)
	speed := GetSquadMovementSpeed(squadID, manager)
	if speed != 2 {
		t.Errorf("Expected initial speed 2, got %d", speed)
	}

	// Kill the slow unit
	unitIDs := GetUnitIDsAtGridPosition(squadID, 0, 0, manager)
	if len(unitIDs) > 0 {
		slowUnit := common.FindEntityByIDWithTag(manager, unitIDs[0], SquadMemberTag)
		attr := common.GetAttributes(slowUnit)
		attr.CurrentHealth = 0
	}

	// Now squad should move at speed 5 (only alive unit)
	speed = GetSquadMovementSpeed(squadID, manager)
	if speed != 5 {
		t.Errorf("Expected speed 5 after slow unit died, got %d", speed)
	}
}

func TestGetSquadMovementSpeed_EmptySquad(t *testing.T) {
	manager := setupTestManager(t)
	CreateEmptySquad(manager, "Empty Squad")

	// Get the squad entity
	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Empty squad should have speed 0
	speed := GetSquadMovementSpeed(squadID, manager)
	if speed != 0 {
		t.Errorf("Expected speed 0 for empty squad, got %d", speed)
	}
}
