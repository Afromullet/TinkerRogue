package gear

import (
	"game_main/common"
	testfx "game_main/testing"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// PURE DATA STRUCTURE TESTS
// ========================================

func TestNewArtifactInventory(t *testing.T) {
	inv := NewArtifactInventory(10)
	if inv == nil {
		t.Fatal("Expected non-nil inventory")
	}
	current, max := inv.GetArtifactCount()
	if current != 0 {
		t.Errorf("Expected 0 artifacts, got %d", current)
	}
	if max != 10 {
		t.Errorf("Expected max 10, got %d", max)
	}
}

func TestAddArtifact(t *testing.T) {
	inv := NewArtifactInventory(5)
	err := inv.AddArtifact("sword_of_dawn")
	if err != nil {
		t.Fatalf("Failed to add artifact: %v", err)
	}
	if !inv.OwnsArtifact("sword_of_dawn") {
		t.Error("Expected to own artifact after adding")
	}
	current, _ := inv.GetArtifactCount()
	if current != 1 {
		t.Errorf("Expected 1 artifact, got %d", current)
	}
}

func TestAddDuplicate(t *testing.T) {
	inv := NewArtifactInventory(5)
	inv.AddArtifact("sword_of_dawn")
	err := inv.AddArtifact("sword_of_dawn")
	if err != nil {
		t.Errorf("Adding duplicate should succeed (instance-based): %v", err)
	}
	current, _ := inv.GetArtifactCount()
	if current != 2 {
		t.Errorf("Expected 2 instances, got %d", current)
	}
}

func TestAddOverCapacity(t *testing.T) {
	inv := NewArtifactInventory(2)
	inv.AddArtifact("artifact_a")
	inv.AddArtifact("artifact_b")
	err := inv.AddArtifact("artifact_c")
	if err == nil {
		t.Error("Expected error when exceeding capacity")
	}
	current, _ := inv.GetArtifactCount()
	if current != 2 {
		t.Errorf("Expected 2 artifacts, got %d", current)
	}
}

func TestRemoveArtifact(t *testing.T) {
	inv := NewArtifactInventory(5)
	inv.AddArtifact("sword_of_dawn")

	err := inv.RemoveArtifact("sword_of_dawn")
	if err != nil {
		t.Fatalf("Failed to remove artifact: %v", err)
	}
	if inv.OwnsArtifact("sword_of_dawn") {
		t.Error("Should not own artifact after removal")
	}
}

func TestRemoveNonexistent(t *testing.T) {
	inv := NewArtifactInventory(5)
	err := inv.RemoveArtifact("nonexistent")
	if err == nil {
		t.Error("Expected error when removing nonexistent artifact")
	}
}

func TestRemoveEquippedFails(t *testing.T) {
	inv := NewArtifactInventory(5)
	inv.AddArtifact("sword_of_dawn")
	inv.MarkArtifactEquipped("sword_of_dawn", ecs.EntityID(42))

	err := inv.RemoveArtifact("sword_of_dawn")
	if err == nil {
		t.Error("Expected error when removing equipped artifact")
	}
	if !inv.OwnsArtifact("sword_of_dawn") {
		t.Error("Equipped artifact should still be owned after failed removal")
	}
}

func TestMarkEquipped(t *testing.T) {
	inv := NewArtifactInventory(5)
	inv.AddArtifact("sword_of_dawn")

	err := inv.MarkArtifactEquipped("sword_of_dawn", ecs.EntityID(10))
	if err != nil {
		t.Fatalf("Failed to mark equipped: %v", err)
	}
	if inv.IsArtifactAvailable("sword_of_dawn") {
		t.Error("Equipped artifact should not be available")
	}
	if inv.GetSquadWithArtifact("sword_of_dawn") != ecs.EntityID(10) {
		t.Errorf("Expected squad 10, got %d", inv.GetSquadWithArtifact("sword_of_dawn"))
	}
}

func TestMarkEquippedAlreadyEquipped(t *testing.T) {
	inv := NewArtifactInventory(5)
	inv.AddArtifact("sword_of_dawn")
	inv.MarkArtifactEquipped("sword_of_dawn", ecs.EntityID(10))

	err := inv.MarkArtifactEquipped("sword_of_dawn", ecs.EntityID(20))
	if err == nil {
		t.Error("Expected error when marking already-equipped artifact")
	}
}

func TestMarkAvailable(t *testing.T) {
	inv := NewArtifactInventory(5)
	inv.AddArtifact("sword_of_dawn")
	inv.MarkArtifactEquipped("sword_of_dawn", ecs.EntityID(10))

	err := inv.MarkArtifactAvailable("sword_of_dawn", ecs.EntityID(10))
	if err != nil {
		t.Fatalf("Failed to mark available: %v", err)
	}
	if !inv.IsArtifactAvailable("sword_of_dawn") {
		t.Error("Artifact should be available after marking available")
	}
}

func TestGetAvailableArtifacts(t *testing.T) {
	inv := NewArtifactInventory(5)
	inv.AddArtifact("artifact_a")
	inv.AddArtifact("artifact_b")
	inv.AddArtifact("artifact_c")
	inv.MarkArtifactEquipped("artifact_b", ecs.EntityID(10))

	available := inv.GetAvailableArtifacts()
	if len(available) != 2 {
		t.Errorf("Expected 2 available, got %d", len(available))
	}

	// Check that artifact_b is not in the available list
	for _, id := range available {
		if id == "artifact_b" {
			t.Error("Equipped artifact should not appear in available list")
		}
	}
}

func TestGetEquippedArtifacts(t *testing.T) {
	inv := NewArtifactInventory(5)
	inv.AddArtifact("artifact_a")
	inv.AddArtifact("artifact_b")
	inv.AddArtifact("artifact_c")
	inv.MarkArtifactEquipped("artifact_b", ecs.EntityID(10))

	equipped := inv.GetEquippedArtifacts()
	if len(equipped) != 1 {
		t.Errorf("Expected 1 equipped, got %d", len(equipped))
	}
	if len(equipped) > 0 && equipped[0] != "artifact_b" {
		t.Errorf("Expected artifact_b equipped, got %s", equipped[0])
	}
}

func TestOwnsArtifact(t *testing.T) {
	inv := NewArtifactInventory(5)
	if inv.OwnsArtifact("nonexistent") {
		t.Error("Should not own artifact that was never added")
	}
	inv.AddArtifact("sword_of_dawn")
	if !inv.OwnsArtifact("sword_of_dawn") {
		t.Error("Should own added artifact")
	}
	// Owns even when equipped
	inv.MarkArtifactEquipped("sword_of_dawn", ecs.EntityID(10))
	if !inv.OwnsArtifact("sword_of_dawn") {
		t.Error("Should still own equipped artifact")
	}
}

func TestIsArtifactAvailable(t *testing.T) {
	inv := NewArtifactInventory(5)
	if inv.IsArtifactAvailable("nonexistent") {
		t.Error("Nonexistent artifact should not be available")
	}
	inv.AddArtifact("sword_of_dawn")
	if !inv.IsArtifactAvailable("sword_of_dawn") {
		t.Error("Newly added artifact should be available")
	}
	inv.MarkArtifactEquipped("sword_of_dawn", ecs.EntityID(10))
	if inv.IsArtifactAvailable("sword_of_dawn") {
		t.Error("Equipped artifact should not be available")
	}
}

func TestCanAddArtifact(t *testing.T) {
	inv := NewArtifactInventory(1)
	if !inv.CanAddArtifact() {
		t.Error("Should be able to add to empty inventory")
	}
	inv.AddArtifact("artifact_a")
	if inv.CanAddArtifact() {
		t.Error("Should not be able to add to full inventory")
	}
}

// ========================================
// ECS INTEGRATION TEST
// ========================================

func TestGetPlayerArtifactInventory(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	playerEntity := manager.World.NewEntity().
		AddComponent(common.PlayerComponent, &common.Player{}).
		AddComponent(ArtifactInventoryComponent, NewArtifactInventory(10))
	playerID := playerEntity.GetID()

	inv := GetPlayerArtifactInventory(playerID, manager)
	if inv == nil {
		t.Fatal("Expected non-nil inventory from player entity")
	}
	_, max := inv.GetArtifactCount()
	if max != 10 {
		t.Errorf("Expected max 10, got %d", max)
	}
}

func TestGetPlayerArtifactInventory_NoComponent(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	entity := manager.World.NewEntity().
		AddComponent(common.PlayerComponent, &common.Player{})

	inv := GetPlayerArtifactInventory(entity.GetID(), manager)
	if inv != nil {
		t.Error("Expected nil inventory when component not present")
	}
}

// ========================================
// MULTI-INSTANCE TESTS
// ========================================

func TestAddMultipleCopies(t *testing.T) {
	inv := NewArtifactInventory(10)
	for i := 0; i < 3; i++ {
		if err := inv.AddArtifact("iron_bulwark"); err != nil {
			t.Fatalf("Failed to add copy %d: %v", i+1, err)
		}
	}
	current, _ := inv.GetArtifactCount()
	if current != 3 {
		t.Errorf("Expected 3 instances, got %d", current)
	}
	if inv.GetInstanceCount("iron_bulwark") != 3 {
		t.Errorf("Expected 3 instances of iron_bulwark, got %d", inv.GetInstanceCount("iron_bulwark"))
	}
}

func TestEquipMultipleInstances(t *testing.T) {
	inv := NewArtifactInventory(10)
	inv.AddArtifact("iron_bulwark")
	inv.AddArtifact("iron_bulwark")

	// Equip first copy on squad 10
	if err := inv.MarkArtifactEquipped("iron_bulwark", ecs.EntityID(10)); err != nil {
		t.Fatalf("Failed to equip on squad 10: %v", err)
	}
	// Equip second copy on squad 20
	if err := inv.MarkArtifactEquipped("iron_bulwark", ecs.EntityID(20)); err != nil {
		t.Fatalf("Failed to equip on squad 20: %v", err)
	}

	// No more available
	if inv.IsArtifactAvailable("iron_bulwark") {
		t.Error("All instances should be equipped")
	}

	// Unequip from squad 10
	if err := inv.MarkArtifactAvailable("iron_bulwark", ecs.EntityID(10)); err != nil {
		t.Fatalf("Failed to unequip from squad 10: %v", err)
	}

	// Now one available again
	if !inv.IsArtifactAvailable("iron_bulwark") {
		t.Error("Expected one available instance after unequip")
	}

	// Squad 20 still equipped
	if inv.GetSquadWithArtifact("iron_bulwark") != ecs.EntityID(20) {
		t.Errorf("Expected squad 20 to still have artifact, got %d", inv.GetSquadWithArtifact("iron_bulwark"))
	}
}

func TestGetAllInstances(t *testing.T) {
	inv := NewArtifactInventory(10)
	inv.AddArtifact("iron_bulwark")
	inv.AddArtifact("iron_bulwark")
	inv.AddArtifact("berserkers_torc")

	all := inv.GetAllInstances()
	if len(all) != 3 {
		t.Errorf("Expected 3 total instances, got %d", len(all))
	}

	bulwarkCount := 0
	torcCount := 0
	for _, info := range all {
		switch info.DefinitionID {
		case "iron_bulwark":
			bulwarkCount++
		case "berserkers_torc":
			torcCount++
		}
	}
	if bulwarkCount != 2 {
		t.Errorf("Expected 2 iron_bulwark instances, got %d", bulwarkCount)
	}
	if torcCount != 1 {
		t.Errorf("Expected 1 berserkers_torc instance, got %d", torcCount)
	}
}
