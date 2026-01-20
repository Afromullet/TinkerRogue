package gear

import (
	"game_main/common"
	"game_main/world/coords"
	"game_main/visual/graphics"
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

// createTestThrowableItem creates a throwable item for testing (without image loading)
func createTestThrowableItem(manager *ecs.Manager, name string) *ecs.Entity {
	// Create a basic throwable action
	throwableAction := NewThrowableAction(
		1,  // duration
		5,  // range
		10, // damage
		graphics.NewCircle(0, 0, graphics.MediumShape),
	)

	// Create properties entity (empty for testing)
	propsEntity := manager.NewEntity()

	// Create item component with throwable action
	item := &Item{
		Count:      1,
		Properties: propsEntity.GetID(),
		Actions:    []ItemAction{throwableAction},
	}

	// Create item entity
	itemEntity := manager.NewEntity().
		AddComponent(common.PositionComponent, &coords.LogicalPosition{X: 0, Y: 0}).
		AddComponent(common.NameComponent, &common.Name{NameStr: name}).
		AddComponent(ItemComponent, item)

	return itemEntity
}

// createTestNonThrowableItem creates a non-throwable item for testing (without image loading)
func createTestNonThrowableItem(manager *ecs.Manager, name string) *ecs.Entity {
	// Create properties entity (empty for testing)
	propsEntity := manager.NewEntity()

	// Create item component without actions
	item := &Item{
		Count:      1,
		Properties: propsEntity.GetID(),
		Actions:    make([]ItemAction, 0),
	}

	// Create item entity
	itemEntity := manager.NewEntity().
		AddComponent(common.PositionComponent, &coords.LogicalPosition{X: 0, Y: 0}).
		AddComponent(common.NameComponent, &common.Name{NameStr: name}).
		AddComponent(ItemComponent, item)

	return itemEntity
}

func TestSelectThrowable_Success(t *testing.T) {
	service, ecsManager, playerID := setupTestInventoryService()

	// Get player inventory
	inv := common.GetComponentTypeByID[*Inventory](ecsManager, playerID, InventoryComponent)
	if inv == nil {
		t.Fatal("Player inventory not found")
	}

	// Create and add a throwable item
	throwableItem := createTestThrowableItem(ecsManager.World, "Fire Bomb")
	AddItem(ecsManager, inv, throwableItem.GetID())

	// Test selecting the throwable
	result := service.SelectThrowable(playerID, 0)

	// Verify success
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	if result.ItemName != "Fire Bomb" {
		t.Errorf("Expected item name 'Fire Bomb', got '%s'", result.ItemName)
	}

	if result.ItemEntityID == 0 {
		t.Error("Expected valid item entity ID")
	}

	if result.ItemIndex != 0 {
		t.Errorf("Expected item index 0, got %d", result.ItemIndex)
	}
}

func TestSelectThrowable_InvalidIndex(t *testing.T) {
	service, _, playerID := setupTestInventoryService()

	// Try to select item at invalid index (inventory is empty)
	result := service.SelectThrowable(playerID, 0)

	// Verify failure
	if result.Success {
		t.Error("Expected failure for invalid index, got success")
	}

	if result.Error == "" {
		t.Error("Expected error message for invalid index")
	}
}

func TestSelectThrowable_NonThrowableItem(t *testing.T) {
	service, ecsManager, playerID := setupTestInventoryService()

	// Get player inventory
	inv := common.GetComponentTypeByID[*Inventory](ecsManager, playerID, InventoryComponent)
	if inv == nil {
		t.Fatal("Player inventory not found")
	}

	// Create and add a non-throwable item
	nonThrowableItem := createTestNonThrowableItem(ecsManager.World, "Potion")
	AddItem(ecsManager, inv, nonThrowableItem.GetID())

	// Try to select it as throwable
	result := service.SelectThrowable(playerID, 0)

	// Verify failure
	if result.Success {
		t.Error("Expected failure for non-throwable item, got success")
	}

	if result.Error != "Item is not throwable" {
		t.Errorf("Expected 'Item is not throwable' error, got '%s'", result.Error)
	}
}

func TestSelectThrowable_NoInventory(t *testing.T) {
	// Create service without adding inventory to player
	ecsManager := common.NewEntityManager()

	// Initialize required components
	InventoryComponent = ecsManager.World.NewComponent()

	playerEntity := ecsManager.World.NewEntity()

	service := NewInventoryService(ecsManager)

	// Try to select throwable
	result := service.SelectThrowable(playerEntity.GetID(), 0)

	// Verify failure
	if result.Success {
		t.Error("Expected failure for player without inventory, got success")
	}

	if result.Error != "Player has no inventory" {
		t.Errorf("Expected 'Player has no inventory' error, got '%s'", result.Error)
	}
}

func TestGetInventoryItemInfo_Success(t *testing.T) {
	service, ecsManager, playerID := setupTestInventoryService()

	// Get player inventory
	inv := common.GetComponentTypeByID[*Inventory](ecsManager, playerID, InventoryComponent)
	if inv == nil {
		t.Fatal("Player inventory not found")
	}

	// Create and add a throwable item
	throwableItem := createTestThrowableItem(ecsManager.World, "Ice Bomb")
	AddItem(ecsManager, inv, throwableItem.GetID())

	// Get item info
	info, err := service.GetInventoryItemInfo(playerID, 0)

	// Verify success
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if info.Name != "Ice Bomb" {
		t.Errorf("Expected name 'Ice Bomb', got '%s'", info.Name)
	}

	if !info.IsThrowable {
		t.Error("Expected item to be throwable")
	}

	if info.Count != 1 {
		t.Errorf("Expected count 1, got %d", info.Count)
	}

	if info.Index != 0 {
		t.Errorf("Expected index 0, got %d", info.Index)
	}
}
