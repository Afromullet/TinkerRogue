package gear

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// InventoryService provides business logic for inventory operations
// Separates game logic from UI presentation layer
type InventoryService struct {
	ecsManager *common.EntityManager
}

// NewInventoryService creates a new inventory service
func NewInventoryService(ecsManager *common.EntityManager) *InventoryService {
	return &InventoryService{
		ecsManager: ecsManager,
	}
}

// InventoryItemInfo contains display information for a specific inventory item
type InventoryItemInfo struct {
	ItemEntityID ecs.EntityID
	Name         string
	Count        int
	Index        int
}

// GetInventoryItemInfo retrieves detailed information about an inventory item
func (svc *InventoryService) GetInventoryItemInfo(playerEntityID ecs.EntityID, itemIndex int) (*InventoryItemInfo, error) {
	// Get player's inventory
	inv := common.GetComponentTypeByID[*Inventory](svc.ecsManager, playerEntityID, InventoryComponent)
	if inv == nil {
		return nil, fmt.Errorf("player has no inventory")
	}

	// Validate item index
	itemEntityID, err := GetItemEntityID(inv, itemIndex)
	if err != nil {
		return nil, err
	}

	// Get item component
	item := GetItemByID(svc.ecsManager, itemEntityID)
	if item == nil {
		return nil, fmt.Errorf("item not found")
	}

	// Get item name
	itemName := "Unknown Item"
	if nameRaw, ok := svc.ecsManager.GetComponent(itemEntityID, common.NameComponent); ok {
		if name, ok := nameRaw.(*common.Name); ok {
			itemName = name.NameStr
		}
	}

	return &InventoryItemInfo{
		ItemEntityID: itemEntityID,
		Name:         itemName,
		Count:        item.Count,
		Index:        itemIndex,
	}, nil
}
