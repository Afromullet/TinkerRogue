package squads

import (
	"fmt"
	"game_main/common"
	"game_main/templates"
	testfx "game_main/testing"
	"game_main/world/coords"
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
	// Initialize all registered subsystems (including squads via init())
	common.InitializeSubsystems(manager)
	if err := InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}
	return manager
}

// createTestJSONMonster creates a JSONMonster for testing
func createTestJSONMonster(name string, width, height int, role string) templates.JSONMonster {
	return templates.JSONMonster{
		Name:      name,
		ImageName: "test.png", // Not used in tests
		Attributes: templates.JSONAttributes{
			Strength:   10, // 40 HP (20 + 10*2), 7 damage (10/2 + 2*2), 6 resistance (10/4 + 2*2)
			Dexterity:  20, // 100% hit (80 + 20*2, capped), 10% crit (20/2), 6% dodge (20/3)
			Magic:      0,  // No magic abilities
			Leadership: 0,  // No squad leadership
			Armor:      2,  // Contributes to physical resistance
			Weapon:     2,  // Contributes to physical damage
		},
		Width:       width,
		Height:      height,
		Role:        role,
		AttackType:  "MeleeRow", // Explicitly set attack type for test clarity
		TargetCells: nil,        // Not used for MeleeRow
	}
}

// createLowCostTestMonster creates a unit with minimal capacity cost for visualization tests
// Cost = (Strength + Weapon + Armor) / 5 = (1 + 0 + 0) / 5 = 0.2 per unit
// This allows fitting multiple units in a single squad for visualization testing
func createLowCostTestMonster(name string, width, height int, role string) templates.JSONMonster {
	return templates.JSONMonster{
		Name:      name,
		ImageName: "test.png",
		Attributes: templates.JSONAttributes{
			Strength:   1,  // Minimal - only affects HP and damage
			Dexterity:  20, // Keep dexterity for combat mechanics
			Magic:      0,
			Leadership: 0,
			Armor:      0, // No armor to minimize capacity cost
			Weapon:     0, // No weapon to minimize capacity cost
		},
		Width:       width,
		Height:      height,
		Role:        role,
		AttackType:  "MeleeRow", // Explicitly set attack type for test clarity
		TargetCells: nil,        // Not used for MeleeRow
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
		SquadID:  squadID,
		Name:     squadName,
		Morale:   100,
		TurnCount: 0,
		MaxUnits: 9,
	})

	squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})

	// Create a dummy leader unit with max capacity to enable adding multiple units
	// Leadership of 9 gives capacity = 6 + (9/3) = 9 (capped)
	leaderEntity := manager.World.NewEntity()

	leaderEntity.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: squadID})
	leaderEntity.AddComponent(common.NameComponent, &common.Name{NameStr: "Leader"})

	// Create attributes for the leader with high Leadership
	leaderAttr := common.NewAttributes(
		1, // Minimal strength
		0, // No dexterity needed
		0, // No magic
		9, // Max leadership for capacity
		0, // No armor
		0, // No weapon
	)
	leaderEntity.AddComponent(common.AttributeComponent, &leaderAttr)

	// Add leader components (LeaderComponent, AbilitySlotComponent, CooldownTrackerComponent)
	AddLeaderComponents(leaderEntity)

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

	_, err = AddUnitToSquad(squadID, manager, unit, 0, 0)
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

			_, err = AddUnitToSquad(squadID, manager, unit, row, col)
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

	_, err = AddUnitToSquad(squadID, manager, unit, 0, 0)
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

	_, err = AddUnitToSquad(squadID, manager, unit, 0, 0)
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

	_, err = AddUnitToSquad(squadID, manager, unit, 0, 0)
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

	_, err = AddUnitToSquad(squadID, manager, unit1, 1, 1)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed for first unit: %v", err)
	}

	// Try to add second unit at same position
	jsonMonster2 := createTestJSONMonster("Warrior2", 1, 1, "Tank")
	unit2, err := CreateUnitTemplates(jsonMonster2)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	_, err = AddUnitToSquad(squadID, manager, unit2, 1, 1)
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

	_, err = AddUnitToSquad(squadID, manager, unit1, 0, 0)
	if err != nil {
		t.Fatalf("AddUnitToSquad failed for 2x2 unit: %v", err)
	}

	// Try to add 1x1 unit at (0,0) - should fail (overlaps giant)
	jsonMonster2 := createTestJSONMonster("Warrior", 1, 1, "DPS")
	unit2, err := CreateUnitTemplates(jsonMonster2)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	_, err = AddUnitToSquad(squadID, manager, unit2, 0, 0)
	if err == nil {
		t.Fatal("Expected error when adding unit to position occupied by 2x2 unit, got nil")
	}

	// Try to add 1x1 unit at (1,1) - should fail (overlaps giant)
	_, err = AddUnitToSquad(squadID, manager, unit2, 1, 1)
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
	_, err = AddUnitToSquad(squadID, manager, unit, 3, 0)
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
	_, err = AddUnitToSquad(squadID, manager, unit, 0, -1)
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

	_, err = AddUnitToSquad(squadID, manager, giant, 0, 0)
	if err != nil {
		t.Fatalf("Failed to add 2x2 unit: %v", err)
	}

	// Add 1x1 unit at (0,2) - top-right corner
	jsonArcher := createLowCostTestMonster("Archer", 1, 1, "DPS")
	archer, err := CreateUnitTemplates(jsonArcher)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	_, err = AddUnitToSquad(squadID, manager, archer, 0, 2)
	if err != nil {
		t.Fatalf("Failed to add 1x1 unit at (0,2): %v", err)
	}

	// Add 1x1 unit at (2,0) - bottom-left corner
	jsonMage := createLowCostTestMonster("Mage", 1, 1, "Support")
	mage, err := CreateUnitTemplates(jsonMage)
	if err != nil {
		t.Fatalf("CreateUnitTemplates failed: %v", err)
	}

	_, err = AddUnitToSquad(squadID, manager, mage, 2, 0)
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

	_, err = AddUnitToSquad(squadID, manager, unit, 1, 1)
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

	_, err := AddUnitToSquad(squadID, manager, unit, 0, 0)
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
		_, err := AddUnitToSquad(squadID, manager, unit, 0, col)
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
	_, _ = AddUnitToSquad(squadID, manager, unit1, 0, 0)

	jsonMonster2 := createTestJSONMonster("FastUnit", 1, 1, "DPS")
	jsonMonster2.MovementSpeed = 5
	unit2, _ := CreateUnitTemplates(jsonMonster2)
	_, _ = AddUnitToSquad(squadID, manager, unit2, 0, 1)

	// Initially, squad moves at speed 2 (slowest)
	speed := GetSquadMovementSpeed(squadID, manager)
	if speed != 2 {
		t.Errorf("Expected initial speed 2, got %d", speed)
	}

	// Kill the slow unit
	unitIDs := GetUnitIDsAtGridPosition(squadID, 0, 0, manager)
	if len(unitIDs) > 0 {
		slowUnit := manager.FindEntityByID(unitIDs[0])
		attr := common.GetComponentType[*common.Attributes](slowUnit, common.AttributeComponent)
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
