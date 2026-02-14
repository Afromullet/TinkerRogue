package gear

import (
	"game_main/common"
	"game_main/world/coords"
	"testing"

	"github.com/bytearena/ecs"
)

// setupTestInventoryService creates a test environment with inventory service
func setupTestInventoryService() (*InventoryService, *common.EntityManager, ecs.EntityID) {
	// Create entity manager
	ecsManager := common.NewEntityManager()

	// Initialize required components for testing
	common.PositionComponent = ecsManager.World.NewComponent()
	common.NameComponent = ecsManager.World.NewComponent()
	InventoryComponent = ecsManager.World.NewComponent()
	ItemComponent = ecsManager.World.NewComponent()

	// Create player entity with inventory
	playerEntity := ecsManager.World.NewEntity()
	playerEntity.AddComponent(InventoryComponent, &Inventory{
		ItemEntityIDs: make([]ecs.EntityID, 0),
	})

	// Create inventory service
	service := NewInventoryService(ecsManager)

	return service, ecsManager, playerEntity.GetID()
}

// createTestItem creates a simple item for testing (without image loading)
func createTestItem(manager *ecs.Manager, name string) *ecs.Entity {
	item := &Item{
		Count: 1,
	}

	itemEntity := manager.NewEntity().
		AddComponent(common.PositionComponent, &coords.LogicalPosition{X: 0, Y: 0}).
		AddComponent(common.NameComponent, &common.Name{NameStr: name}).
		AddComponent(ItemComponent, item)

	return itemEntity
}

func TestGetInventoryItemInfo_Success(t *testing.T) {
	service, ecsManager, playerID := setupTestInventoryService()

	// Get player inventory
	inv := common.GetComponentTypeByID[*Inventory](ecsManager, playerID, InventoryComponent)
	if inv == nil {
		t.Fatal("Player inventory not found")
	}

	// Create and add an item
	testItem := createTestItem(ecsManager.World, "Health Potion")
	AddItem(ecsManager, inv, testItem.GetID())

	// Get item info
	info, err := service.GetInventoryItemInfo(playerID, 0)

	// Verify success
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if info.Name != "Health Potion" {
		t.Errorf("Expected name 'Health Potion', got '%s'", info.Name)
	}

	if info.Count != 1 {
		t.Errorf("Expected count 1, got %d", info.Count)
	}

	if info.Index != 0 {
		t.Errorf("Expected index 0, got %d", info.Index)
	}
}

func TestGetInventoryItemInfo_InvalidIndex(t *testing.T) {
	service, _, playerID := setupTestInventoryService()

	// Try to get item at invalid index (inventory is empty)
	_, err := service.GetInventoryItemInfo(playerID, 0)

	if err == nil {
		t.Error("Expected error for invalid index, got nil")
	}
}

func TestGetInventoryItemInfo_NoInventory(t *testing.T) {
	// Create service without adding inventory to player
	ecsManager := common.NewEntityManager()

	// Initialize required components
	InventoryComponent = ecsManager.World.NewComponent()

	playerEntity := ecsManager.World.NewEntity()

	service := NewInventoryService(ecsManager)

	// Try to get item info
	_, err := service.GetInventoryItemInfo(playerEntity.GetID(), 0)

	if err == nil {
		t.Error("Expected error for player without inventory, got nil")
	}

	if err.Error() != "player has no inventory" {
		t.Errorf("Expected 'player has no inventory' error, got '%s'", err.Error())
	}
}
