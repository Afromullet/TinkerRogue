package squads

import (
	"game_main/common"
	"testing"

	"github.com/bytearena/ecs"
)

func TestUnitRoster_TemplateBasedCounting(t *testing.T) {
	manager := common.NewEntityManager()
	if err := InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}

	// Create roster
	roster := NewUnitRoster(10)

	// Create some unit entities
	unit1 := manager.World.NewEntity()
	unit2 := manager.World.NewEntity()
	unit3 := manager.World.NewEntity()

	// Add warriors
	err := roster.AddUnit(unit1.GetID(), "Warrior")
	if err != nil {
		t.Fatalf("Failed to add first warrior: %v", err)
	}

	err = roster.AddUnit(unit2.GetID(), "Warrior")
	if err != nil {
		t.Fatalf("Failed to add second warrior: %v", err)
	}

	// Add mage
	err = roster.AddUnit(unit3.GetID(), "Mage")
	if err != nil {
		t.Fatalf("Failed to add mage: %v", err)
	}

	// Check counts
	warriorEntry := roster.Units["Warrior"]
	if warriorEntry == nil {
		t.Fatal("Warrior entry not found")
	}
	if warriorEntry.TotalOwned != 2 {
		t.Errorf("Expected 2 warriors, got %d", warriorEntry.TotalOwned)
	}

	mageEntry := roster.Units["Mage"]
	if mageEntry == nil {
		t.Fatal("Mage entry not found")
	}
	if mageEntry.TotalOwned != 1 {
		t.Errorf("Expected 1 mage, got %d", mageEntry.TotalOwned)
	}

	// Check total count
	current, max := roster.GetUnitCount()
	if current != 3 {
		t.Errorf("Expected 3 total units, got %d", current)
	}
	if max != 10 {
		t.Errorf("Expected max 10 units, got %d", max)
	}
}

func TestUnitRoster_AvailableCount(t *testing.T) {
	manager := common.NewEntityManager()

	// Initialize components
	if err := InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}

	// Create roster
	roster := NewUnitRoster(10)

	// Add 3 warriors
	unit1 := manager.World.NewEntity().GetID()
	unit2 := manager.World.NewEntity().GetID()
	unit3 := manager.World.NewEntity().GetID()

	roster.AddUnit(unit1, "Warrior")
	roster.AddUnit(unit2, "Warrior")
	roster.AddUnit(unit3, "Warrior")

	// Check available count (all should be available)
	available := roster.GetAvailableCount("Warrior")
	if available != 3 {
		t.Errorf("Expected 3 available warriors, got %d", available)
	}

	// Mark one as in squad
	squadID := ecs.EntityID(999)
	err := roster.MarkUnitInSquad(unit1, squadID)
	if err != nil {
		t.Fatalf("Failed to mark unit in squad: %v", err)
	}

	// Check available count (should be 2 now)
	available = roster.GetAvailableCount("Warrior")
	if available != 2 {
		t.Errorf("Expected 2 available warriors after placing one, got %d", available)
	}

	// Mark another as in squad
	roster.MarkUnitInSquad(unit2, squadID)
	available = roster.GetAvailableCount("Warrior")
	if available != 1 {
		t.Errorf("Expected 1 available warrior after placing two, got %d", available)
	}

	// Mark unit back as available
	err = roster.MarkUnitAvailable(unit1)
	if err != nil {
		t.Fatalf("Failed to mark unit available: %v", err)
	}

	available = roster.GetAvailableCount("Warrior")
	if available != 2 {
		t.Errorf("Expected 2 available warriors after returning one, got %d", available)
	}
}

func TestUnitRoster_GetAvailableUnits(t *testing.T) {
	manager := common.NewEntityManager()

	// Initialize components
	if err := InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}

	// Create roster
	roster := NewUnitRoster(10)

	// Add units
	unit1 := manager.World.NewEntity().GetID()
	unit2 := manager.World.NewEntity().GetID()
	unit3 := manager.World.NewEntity().GetID()

	roster.AddUnit(unit1, "Warrior")
	roster.AddUnit(unit2, "Mage")
	roster.AddUnit(unit3, "Archer")

	// Get available units
	available := roster.GetAvailableUnits()
	if len(available) != 3 {
		t.Errorf("Expected 3 available unit types, got %d", len(available))
	}

	// Mark all warriors as in squad
	squadID := ecs.EntityID(999)
	roster.MarkUnitInSquad(unit1, squadID)

	// Should still have 2 available types (Mage and Archer)
	available = roster.GetAvailableUnits()
	if len(available) != 2 {
		t.Errorf("Expected 2 available unit types after placing warrior, got %d", len(available))
	}

	// Verify warrior is not in available list
	for _, entry := range available {
		if entry.TemplateName == "Warrior" {
			t.Error("Warrior should not be in available list")
		}
	}
}

func TestUnitRoster_GetUnitEntityForTemplate(t *testing.T) {
	manager := common.NewEntityManager()

	// Initialize components
	if err := InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}

	// Create roster
	roster := NewUnitRoster(10)

	// Add warriors
	unit1 := manager.World.NewEntity().GetID()
	unit2 := manager.World.NewEntity().GetID()

	roster.AddUnit(unit1, "Warrior")
	roster.AddUnit(unit2, "Warrior")

	// Get entity for template
	entityID := roster.GetUnitEntityForTemplate("Warrior")
	if entityID == 0 {
		t.Error("Expected valid entity ID for Warrior")
	}

	// Mark all warriors as in squad
	squadID := ecs.EntityID(999)
	roster.MarkUnitInSquad(unit1, squadID)
	roster.MarkUnitInSquad(unit2, squadID)

	// Should return 0 since no available warriors
	entityID = roster.GetUnitEntityForTemplate("Warrior")
	if entityID != 0 {
		t.Errorf("Expected 0 entity ID when no warriors available, got %d", entityID)
	}

	// Get entity for non-existent template
	entityID = roster.GetUnitEntityForTemplate("NonExistent")
	if entityID != 0 {
		t.Errorf("Expected 0 entity ID for non-existent template, got %d", entityID)
	}
}

func TestUnitRoster_RemoveUnit(t *testing.T) {
	manager := common.NewEntityManager()

	// Initialize components
	if err := InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}

	// Create roster
	roster := NewUnitRoster(10)

	// Add warriors
	unit1 := manager.World.NewEntity().GetID()
	unit2 := manager.World.NewEntity().GetID()

	roster.AddUnit(unit1, "Warrior")
	roster.AddUnit(unit2, "Warrior")

	// Remove one warrior
	removed := roster.RemoveUnit(unit1)
	if !removed {
		t.Error("Expected unit to be removed")
	}

	// Check count
	entry := roster.Units["Warrior"]
	if entry.TotalOwned != 1 {
		t.Errorf("Expected 1 warrior after removal, got %d", entry.TotalOwned)
	}

	// Remove last warrior
	roster.RemoveUnit(unit2)

	// Entry should be removed entirely
	_, exists := roster.Units["Warrior"]
	if exists {
		t.Error("Expected Warrior entry to be removed after last unit removed")
	}
}
