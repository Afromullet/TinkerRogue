package gear

import (
	"game_main/common"
	testfx "game_main/testing"
	"testing"

	"github.com/bytearena/ecs"
)

// --- Test-only query helpers ---

func getAvailableArtifacts(inv *ArtifactInventoryData) []string {
	var result []string
	for id, instances := range inv.OwnedArtifacts {
		for _, inst := range instances {
			if inst.EquippedOn == 0 {
				result = append(result, id)
				break
			}
		}
	}
	return result
}

func getEquippedArtifacts(inv *ArtifactInventoryData) []string {
	var result []string
	for id, instances := range inv.OwnedArtifacts {
		for _, inst := range instances {
			if inst.EquippedOn != 0 {
				result = append(result, id)
				break
			}
		}
	}
	return result
}

func getFirstSquadWithArtifact(inv *ArtifactInventoryData, artifactID string) ecs.EntityID {
	instances, exists := inv.OwnedArtifacts[artifactID]
	if !exists {
		return 0
	}
	for _, inst := range instances {
		if inst.EquippedOn != 0 {
			return inst.EquippedOn
		}
	}
	return 0
}

// ========================================
// PURE DATA STRUCTURE TESTS
// ========================================

func TestNewArtifactInventory(t *testing.T) {
	inv := NewArtifactInventory(10)
	if inv == nil {
		t.Fatal("Expected non-nil inventory")
	}
	current, max := GetArtifactCount(inv)
	if current != 0 {
		t.Errorf("Expected 0 artifacts, got %d", current)
	}
	if max != 10 {
		t.Errorf("Expected max 10, got %d", max)
	}
}

func TestAddArtifact(t *testing.T) {
	inv := NewArtifactInventory(5)
	err := AddArtifactToInventory(inv, "sword_of_dawn")
	if err != nil {
		t.Fatalf("Failed to add artifact: %v", err)
	}
	if !OwnsArtifact(inv, "sword_of_dawn") {
		t.Error("Expected to own artifact after adding")
	}
	current, _ := GetArtifactCount(inv)
	if current != 1 {
		t.Errorf("Expected 1 artifact, got %d", current)
	}
}

func TestAddDuplicate(t *testing.T) {
	inv := NewArtifactInventory(5)
	AddArtifactToInventory(inv, "sword_of_dawn")
	err := AddArtifactToInventory(inv, "sword_of_dawn")
	if err != nil {
		t.Errorf("Adding duplicate should succeed (instance-based): %v", err)
	}
	current, _ := GetArtifactCount(inv)
	if current != 2 {
		t.Errorf("Expected 2 instances, got %d", current)
	}
}

func TestAddOverCapacity(t *testing.T) {
	inv := NewArtifactInventory(2)
	AddArtifactToInventory(inv, "artifact_a")
	AddArtifactToInventory(inv, "artifact_b")
	err := AddArtifactToInventory(inv, "artifact_c")
	if err == nil {
		t.Error("Expected error when exceeding capacity")
	}
	current, _ := GetArtifactCount(inv)
	if current != 2 {
		t.Errorf("Expected 2 artifacts, got %d", current)
	}
}

func TestRemoveArtifact(t *testing.T) {
	inv := NewArtifactInventory(5)
	AddArtifactToInventory(inv, "sword_of_dawn")

	err := RemoveArtifactFromInventory(inv, "sword_of_dawn")
	if err != nil {
		t.Fatalf("Failed to remove artifact: %v", err)
	}
	if OwnsArtifact(inv, "sword_of_dawn") {
		t.Error("Should not own artifact after removal")
	}
}

func TestRemoveNonexistent(t *testing.T) {
	inv := NewArtifactInventory(5)
	err := RemoveArtifactFromInventory(inv, "nonexistent")
	if err == nil {
		t.Error("Expected error when removing nonexistent artifact")
	}
}

func TestRemoveEquippedFails(t *testing.T) {
	inv := NewArtifactInventory(5)
	AddArtifactToInventory(inv, "sword_of_dawn")
	MarkArtifactEquipped(inv, "sword_of_dawn", ecs.EntityID(42))

	err := RemoveArtifactFromInventory(inv, "sword_of_dawn")
	if err == nil {
		t.Error("Expected error when removing equipped artifact")
	}
	if !OwnsArtifact(inv, "sword_of_dawn") {
		t.Error("Equipped artifact should still be owned after failed removal")
	}
}

func TestMarkEquipped(t *testing.T) {
	inv := NewArtifactInventory(5)
	AddArtifactToInventory(inv, "sword_of_dawn")

	err := MarkArtifactEquipped(inv, "sword_of_dawn", ecs.EntityID(10))
	if err != nil {
		t.Fatalf("Failed to mark equipped: %v", err)
	}
	if IsArtifactAvailable(inv, "sword_of_dawn") {
		t.Error("Equipped artifact should not be available")
	}
	if getFirstSquadWithArtifact(inv, "sword_of_dawn") != ecs.EntityID(10) {
		t.Errorf("Expected squad 10, got %d", getFirstSquadWithArtifact(inv, "sword_of_dawn"))
	}
}

func TestMarkEquippedAlreadyEquipped(t *testing.T) {
	inv := NewArtifactInventory(5)
	AddArtifactToInventory(inv, "sword_of_dawn")
	MarkArtifactEquipped(inv, "sword_of_dawn", ecs.EntityID(10))

	err := MarkArtifactEquipped(inv, "sword_of_dawn", ecs.EntityID(20))
	if err == nil {
		t.Error("Expected error when marking already-equipped artifact")
	}
}

func TestMarkAvailable(t *testing.T) {
	inv := NewArtifactInventory(5)
	AddArtifactToInventory(inv, "sword_of_dawn")
	MarkArtifactEquipped(inv, "sword_of_dawn", ecs.EntityID(10))

	err := MarkArtifactAvailable(inv, "sword_of_dawn", ecs.EntityID(10))
	if err != nil {
		t.Fatalf("Failed to mark available: %v", err)
	}
	if !IsArtifactAvailable(inv, "sword_of_dawn") {
		t.Error("Artifact should be available after marking available")
	}
}

func TestgetAvailableArtifacts(t *testing.T) {
	inv := NewArtifactInventory(5)
	AddArtifactToInventory(inv, "artifact_a")
	AddArtifactToInventory(inv, "artifact_b")
	AddArtifactToInventory(inv, "artifact_c")
	MarkArtifactEquipped(inv, "artifact_b", ecs.EntityID(10))

	available := getAvailableArtifacts(inv)
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

func TestgetEquippedArtifacts(t *testing.T) {
	inv := NewArtifactInventory(5)
	AddArtifactToInventory(inv, "artifact_a")
	AddArtifactToInventory(inv, "artifact_b")
	AddArtifactToInventory(inv, "artifact_c")
	MarkArtifactEquipped(inv, "artifact_b", ecs.EntityID(10))

	equipped := getEquippedArtifacts(inv)
	if len(equipped) != 1 {
		t.Errorf("Expected 1 equipped, got %d", len(equipped))
	}
	if len(equipped) > 0 && equipped[0] != "artifact_b" {
		t.Errorf("Expected artifact_b equipped, got %s", equipped[0])
	}
}

func TestOwnsArtifact(t *testing.T) {
	inv := NewArtifactInventory(5)
	if OwnsArtifact(inv, "nonexistent") {
		t.Error("Should not own artifact that was never added")
	}
	AddArtifactToInventory(inv, "sword_of_dawn")
	if !OwnsArtifact(inv, "sword_of_dawn") {
		t.Error("Should own added artifact")
	}
	// Owns even when equipped
	MarkArtifactEquipped(inv, "sword_of_dawn", ecs.EntityID(10))
	if !OwnsArtifact(inv, "sword_of_dawn") {
		t.Error("Should still own equipped artifact")
	}
}

func TestIsArtifactAvailable(t *testing.T) {
	inv := NewArtifactInventory(5)
	if IsArtifactAvailable(inv, "nonexistent") {
		t.Error("Nonexistent artifact should not be available")
	}
	AddArtifactToInventory(inv, "sword_of_dawn")
	if !IsArtifactAvailable(inv, "sword_of_dawn") {
		t.Error("Newly added artifact should be available")
	}
	MarkArtifactEquipped(inv, "sword_of_dawn", ecs.EntityID(10))
	if IsArtifactAvailable(inv, "sword_of_dawn") {
		t.Error("Equipped artifact should not be available")
	}
}

func TestCanAddArtifact(t *testing.T) {
	inv := NewArtifactInventory(1)
	if !CanAddArtifact(inv) {
		t.Error("Should be able to add to empty inventory")
	}
	AddArtifactToInventory(inv, "artifact_a")
	if CanAddArtifact(inv) {
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
	_, max := GetArtifactCount(inv)
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
		if err := AddArtifactToInventory(inv, "iron_bulwark"); err != nil {
			t.Fatalf("Failed to add copy %d: %v", i+1, err)
		}
	}
	current, _ := GetArtifactCount(inv)
	if current != 3 {
		t.Errorf("Expected 3 instances, got %d", current)
	}
	if GetInstanceCount(inv, "iron_bulwark") != 3 {
		t.Errorf("Expected 3 instances of iron_bulwark, got %d", GetInstanceCount(inv, "iron_bulwark"))
	}
}

func TestEquipMultipleInstances(t *testing.T) {
	inv := NewArtifactInventory(10)
	AddArtifactToInventory(inv, "iron_bulwark")
	AddArtifactToInventory(inv, "iron_bulwark")

	// Equip first copy on squad 10
	if err := MarkArtifactEquipped(inv, "iron_bulwark", ecs.EntityID(10)); err != nil {
		t.Fatalf("Failed to equip on squad 10: %v", err)
	}
	// Equip second copy on squad 20
	if err := MarkArtifactEquipped(inv, "iron_bulwark", ecs.EntityID(20)); err != nil {
		t.Fatalf("Failed to equip on squad 20: %v", err)
	}

	// No more available
	if IsArtifactAvailable(inv, "iron_bulwark") {
		t.Error("All instances should be equipped")
	}

	// Unequip from squad 10
	if err := MarkArtifactAvailable(inv, "iron_bulwark", ecs.EntityID(10)); err != nil {
		t.Fatalf("Failed to unequip from squad 10: %v", err)
	}

	// Now one available again
	if !IsArtifactAvailable(inv, "iron_bulwark") {
		t.Error("Expected one available instance after unequip")
	}

	// Squad 20 still equipped
	if getFirstSquadWithArtifact(inv, "iron_bulwark") != ecs.EntityID(20) {
		t.Errorf("Expected squad 20 to still have artifact, got %d", getFirstSquadWithArtifact(inv, "iron_bulwark"))
	}
}

func TestGetAllInstances(t *testing.T) {
	inv := NewArtifactInventory(10)
	AddArtifactToInventory(inv, "iron_bulwark")
	AddArtifactToInventory(inv, "iron_bulwark")
	AddArtifactToInventory(inv, "berserkers_torc")

	all := GetAllInstances(inv)
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
