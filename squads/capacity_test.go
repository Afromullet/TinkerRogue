package squads

import (
	"game_main/common"
	"game_main/entitytemplates"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// CAPACITY SYSTEM TESTS
// ========================================

func TestCapacitySystem_BasicCalculations(t *testing.T) {
	manager := setupTestManager(t)

	// Create squad with no leader (default capacity = 6)
	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Test GetSquadTotalCapacity (should be 6 with no leader)
	totalCapacity := GetSquadTotalCapacity(squadID, manager)
	if totalCapacity != 6 {
		t.Errorf("Expected default capacity 6, got %d", totalCapacity)
	}

	// Test GetSquadUsedCapacity (should be 0 for empty squad)
	usedCapacity := GetSquadUsedCapacity(squadID, manager)
	if usedCapacity != 0 {
		t.Errorf("Expected 0 used capacity for empty squad, got %.2f", usedCapacity)
	}

	// Test GetSquadRemainingCapacity (should be 6.0)
	remaining := GetSquadRemainingCapacity(squadID, manager)
	if remaining != 6.0 {
		t.Errorf("Expected 6.0 remaining capacity, got %.2f", remaining)
	}
}

func TestCapacitySystem_UnitCapacityCost(t *testing.T) {
	// Test unit with Strength=10, Weapon=2, Armor=2
	// Expected cost: (10+2+2)/5 = 2.8
	attr := common.NewAttributes(10, 20, 0, 0, 2, 2)
	cost := attr.GetCapacityCost()
	expected := 2.8
	if cost != expected {
		t.Errorf("Expected capacity cost %.2f, got %.2f", expected, cost)
	}

	// Test stronger unit: Strength=18, Weapon=8, Armor=9
	// Expected cost: (18+8+9)/5 = 7.0
	strongAttr := common.NewAttributes(18, 6, 0, 0, 9, 8)
	strongCost := strongAttr.GetCapacityCost()
	expectedStrong := 7.0
	if strongCost != expectedStrong {
		t.Errorf("Expected strong unit cost %.2f, got %.2f", expectedStrong, strongCost)
	}
}

func TestCapacitySystem_EnforceLimitWithoutLeader(t *testing.T) {
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Create weak units (cost 1.0 each: Strength=3, Weapon=1, Armor=1 → (3+1+1)/5 = 1.0)
	jsonMonster := entitytemplates.JSONMonster{
		Name:      "Weak Unit",
		ImageName: "test.png",
		Attributes: entitytemplates.JSONAttributes{
			Strength:   3,
			Dexterity:  20,
			Magic:      0,
			Leadership: 0,
			Armor:      1,
			Weapon:     1,
		},
		Width:       1,
		Height:      1,
		Role:        "DPS",
		AttackType:  "MeleeRow",
		TargetCells: nil,
	}

	// With default capacity of 6, we should be able to add 6 units (6 * 1.0 = 6.0)
	for i := 0; i < 6; i++ {
		unit, err := CreateUnitTemplates(jsonMonster)
		if err != nil {
			t.Fatalf("CreateUnitTemplates failed: %v", err)
		}

		row := i / 3
		col := i % 3
		err = AddUnitToSquad(squadID, manager, unit, row, col)
		if err != nil {
			t.Errorf("Failed to add unit %d: %v", i, err)
		}
	}

	// Verify we have 6 units
	unitCount := len(GetUnitIDsInSquad(squadID, manager))
	if unitCount != 6 {
		t.Errorf("Expected 6 units, got %d", unitCount)
	}

	// Try to add a 7th unit (should fail - exceeds capacity)
	unit, _ := CreateUnitTemplates(jsonMonster)
	err := AddUnitToSquad(squadID, manager, unit, 2, 0)
	if err == nil {
		t.Error("Expected error when exceeding capacity, got nil")
	}
}

func TestCapacitySystem_WithLeader(t *testing.T) {
	// Test the Leadership attribute's effect on capacity calculation
	// Leadership=9 → capacity = 6 + 9/3 = 9
	attr := common.NewAttributes(5, 20, 0, 9, 1, 1)
	capacity := attr.GetUnitCapacity()

	expected := 9
	if capacity != expected {
		t.Errorf("Expected capacity %d with Leadership=9, got %d", expected, capacity)
	}

	// Test max capacity cap (Leadership=15 should still cap at 9)
	highLeaderAttr := common.NewAttributes(5, 20, 0, 15, 1, 1)
	highCapacity := highLeaderAttr.GetUnitCapacity()

	if highCapacity != 9 {
		t.Errorf("Expected max capacity 9 even with Leadership=15, got %d", highCapacity)
	}

	t.Logf("Leadership=9 → Capacity: %d", capacity)
	t.Logf("Leadership=15 → Capacity: %d (capped)", highCapacity)
}

func TestCapacitySystem_IsSquadOverCapacity(t *testing.T) {
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Initially not over capacity
	if IsSquadOverCapacity(squadID, manager) {
		t.Error("Empty squad should not be over capacity")
	}

	// Add units up to capacity (6.0 total)
	jsonMonster := entitytemplates.JSONMonster{
		Name:      "Unit",
		ImageName: "test.png",
		Attributes: entitytemplates.JSONAttributes{
			Strength:   3,
			Dexterity:  20,
			Magic:      0,
			Leadership: 0,
			Armor:      1,
			Weapon:     1,
		},
		Width:       1,
		Height:      1,
		Role:        "DPS",
		AttackType:  "MeleeRow",
		TargetCells: nil,
	}

	// Add units (capacity cost = 1.0 each)
	for i := 0; i < 6; i++ {
		unit, _ := CreateUnitTemplates(jsonMonster)
		row := i / 3
		col := i % 3
		AddUnitToSquad(squadID, manager, unit, row, col)
	}

	// Should be at capacity but not over
	remaining := GetSquadRemainingCapacity(squadID, manager)
	if remaining != 0.0 {
		t.Errorf("Expected 0 remaining capacity, got %.2f", remaining)
	}

	if IsSquadOverCapacity(squadID, manager) {
		t.Error("Squad at full capacity should not report as over capacity")
	}
}

func TestCapacitySystem_CanAddUnitToSquad(t *testing.T) {
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Can add unit with cost 2.0 to empty squad (capacity 6)
	if !CanAddUnitToSquad(squadID, 2.0, manager) {
		t.Error("Should be able to add unit with cost 2.0 to empty squad")
	}

	// Cannot add unit with cost 7.0 to empty squad (exceeds capacity 6)
	if CanAddUnitToSquad(squadID, 7.0, manager) {
		t.Error("Should not be able to add unit with cost 7.0 to squad with capacity 6")
	}
}

func TestCapacitySystem_UpdateSquadCapacity(t *testing.T) {
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add a unit
	jsonMonster := entitytemplates.JSONMonster{
		Name:      "Unit",
		ImageName: "test.png",
		Attributes: entitytemplates.JSONAttributes{
			Strength:   3,
			Dexterity:  20,
			Magic:      0,
			Leadership: 0,
			Armor:      1,
			Weapon:     1,
		},
		Width:       1,
		Height:      1,
		Role:        "DPS",
		AttackType:  "MeleeRow",
		TargetCells: nil,
	}

	unit, _ := CreateUnitTemplates(jsonMonster)
	AddUnitToSquad(squadID, manager, unit, 0, 0)

	// Check SquadData fields were updated
	squadEntity := GetSquadEntity(squadID, manager)
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	if squadData.TotalCapacity != 6 {
		t.Errorf("Expected TotalCapacity 6, got %d", squadData.TotalCapacity)
	}

	expectedUsed := 1.0 // (3+1+1)/5
	if squadData.UsedCapacity != expectedUsed {
		t.Errorf("Expected UsedCapacity %.2f, got %.2f", expectedUsed, squadData.UsedCapacity)
	}
}
