package squads

import (
	"game_main/common"
	"game_main/config"
	"game_main/templates"

	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// CAPACITY SYSTEM TESTS
// ========================================

func TestCapacitySystem_BasicCalculations(t *testing.T) {
	manager := setupTestManager(t)

	// Create squad with no leader (default capacity = config.DefaultBaseCapacity)
	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Test GetSquadTotalCapacity (should be DefaultBaseCapacity with no leader)
	totalCapacity := GetSquadTotalCapacity(squadID, manager)
	if totalCapacity != config.DefaultBaseCapacity {
		t.Errorf("Expected default capacity %d, got %d", config.DefaultBaseCapacity, totalCapacity)
	}

	// Test GetSquadUsedCapacity (should be 0 for empty squad)
	usedCapacity := GetSquadUsedCapacity(squadID, manager)
	if usedCapacity != 0 {
		t.Errorf("Expected 0 used capacity for empty squad, got %.2f", usedCapacity)
	}

	// Test GetSquadRemainingCapacity (should be DefaultBaseCapacity)
	remaining := GetSquadRemainingCapacity(squadID, manager)
	if remaining != float64(config.DefaultBaseCapacity) {
		t.Errorf("Expected %.2f remaining capacity, got %.2f", float64(config.DefaultBaseCapacity), remaining)
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

	// Create units (cost 10.0 each: Strength=30, Weapon=10, Armor=10 → (30+10+10)/5 = 10.0)
	jsonMonster := templates.JSONMonster{
		UnitType:  "Unit",
		ImageName: "",
		Attributes: templates.JSONAttributes{
			Strength:   30,
			Dexterity:  20,
			Magic:      0,
			Leadership: 0,
			Armor:      10,
			Weapon:     10,
		},
		Width:       1,
		Height:      1,
		Role:        "DPS",
		AttackType:  "MeleeRow",
		TargetCells: nil,
	}

	// With default capacity of 60, we should be able to add 6 units (6 * 10.0 = 60.0)
	for i := 0; i < 6; i++ {
		unit, err := CreateUnitTemplates(jsonMonster)
		if err != nil {
			t.Fatalf("CreateUnitTemplates failed: %v", err)
		}

		row := i / 3
		col := i % 3
		_, err = AddUnitToSquad(squadID, manager, unit, row, col)
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
	_, err := AddUnitToSquad(squadID, manager, unit, 2, 0)
	if err == nil {
		t.Error("Expected error when exceeding capacity, got nil")
	}
}

func TestCapacitySystem_WithLeader(t *testing.T) {
	// Test the Leadership attribute's effect on capacity calculation
	// Leadership=9 → capacity = DefaultBaseCapacity + 9/3 = 60 + 3 = 63
	attr := common.NewAttributes(5, 20, 0, 9, 1, 1)
	capacity := attr.GetUnitCapacity()

	expected := config.DefaultBaseCapacity + 9/3
	if capacity != expected {
		t.Errorf("Expected capacity %d with Leadership=9, got %d", expected, capacity)
	}

	// Test max capacity cap (Leadership=300 → 60 + 100 = 160, capped at DefaultMaxCapacity)
	highLeaderAttr := common.NewAttributes(5, 20, 0, 300, 1, 1)
	highCapacity := highLeaderAttr.GetUnitCapacity()

	if highCapacity != config.DefaultMaxCapacity {
		t.Errorf("Expected max capacity %d even with Leadership=300, got %d", config.DefaultMaxCapacity, highCapacity)
	}

	t.Logf("Leadership=9 → Capacity: %d", capacity)
	t.Logf("Leadership=300 → Capacity: %d (capped)", highCapacity)
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

	// Add units up to capacity (60.0 total)
	jsonMonster := templates.JSONMonster{
		UnitType:  "Unit",
		ImageName: "",
		Attributes: templates.JSONAttributes{
			Strength:   30,
			Dexterity:  20,
			Magic:      0,
			Leadership: 0,
			Armor:      10,
			Weapon:     10,
		},
		Width:       1,
		Height:      1,
		Role:        "DPS",
		AttackType:  "MeleeRow",
		TargetCells: nil,
	}

	// Add units (capacity cost = 10.0 each)
	for i := 0; i < 6; i++ {
		unit, _ := CreateUnitTemplates(jsonMonster)
		row := i / 3
		col := i % 3
		_, _ = AddUnitToSquad(squadID, manager, unit, row, col)
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

	// Can add unit with cost 2.0 to empty squad
	if !CanAddUnitToSquad(squadID, 2.0, manager) {
		t.Error("Should be able to add unit with cost 2.0 to empty squad")
	}

	// Cannot add unit with cost exceeding capacity
	overCost := float64(config.DefaultBaseCapacity) + 1.0
	if CanAddUnitToSquad(squadID, overCost, manager) {
		t.Errorf("Should not be able to add unit with cost %.1f to squad with capacity %d", overCost, config.DefaultBaseCapacity)
	}
}

func TestCapacitySystem_ComputedCapacityAfterAddingUnit(t *testing.T) {
	manager := setupTestManager(t)

	CreateEmptySquad(manager, "Test Squad")

	var squadID ecs.EntityID
	for _, result := range manager.World.Query(SquadTag) {
		squadID = result.Entity.GetID()
		break
	}

	// Add a unit
	jsonMonster := templates.JSONMonster{
		UnitType:  "Unit",
		ImageName: "",
		Attributes: templates.JSONAttributes{
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
	_, _ = AddUnitToSquad(squadID, manager, unit, 0, 0)

	// Check computed capacity functions
	totalCapacity := GetSquadTotalCapacity(squadID, manager)
	if totalCapacity != config.DefaultBaseCapacity {
		t.Errorf("Expected TotalCapacity %d, got %d", config.DefaultBaseCapacity, totalCapacity)
	}

	expectedUsed := 1.0 // (3+1+1)/5
	usedCapacity := GetSquadUsedCapacity(squadID, manager)
	if usedCapacity != expectedUsed {
		t.Errorf("Expected UsedCapacity %.2f, got %.2f", expectedUsed, usedCapacity)
	}
}
