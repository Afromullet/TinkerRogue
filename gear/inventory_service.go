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

// SelectThrowableResult contains the result of selecting a throwable item
type SelectThrowableResult struct {
	Success              bool
	ItemEntityID         ecs.EntityID
	ItemName             string
	ItemIndex            int
	EffectDescriptions   []string
	Error                string
}

// SelectThrowable prepares a throwable item for use
// Validates the item exists, has throwable action, and returns all necessary data
// This replaces direct ECS manipulation in the UI layer
func (svc *InventoryService) SelectThrowable(playerEntityID ecs.EntityID, itemIndex int) SelectThrowableResult {
	result := SelectThrowableResult{
		Success: false,
	}

	// Get player's inventory
	inv := common.GetComponentTypeByID[*Inventory](svc.ecsManager, playerEntityID, InventoryComponent)
	if inv == nil {
		result.Error = "Player has no inventory"
		return result
	}

	// Validate item index
	itemEntityID, err := GetItemEntityID(inv, itemIndex)
	if err != nil {
		result.Error = fmt.Sprintf("Invalid item index: %v", err)
		return result
	}

	// Get item component
	item := GetItemByID(svc.ecsManager.World, itemEntityID)
	if item == nil {
		result.Error = "Item not found"
		return result
	}

	// Verify item has throwable action
	throwableAction := item.GetThrowableAction()
	if throwableAction == nil {
		result.Error = "Item is not throwable"
		return result
	}

	// Get item name
	itemName := "Unknown Item"
	if nameRaw, ok := svc.ecsManager.GetComponent(itemEntityID, common.NameComponent); ok {
		if name, ok := nameRaw.(*common.Name); ok {
			itemName = name.NameStr
		}
	}

	// Get effect descriptions
	effectNames := GetItemEffectNames(svc.ecsManager.World, item)

	// Build successful result
	result.Success = true
	result.ItemEntityID = itemEntityID
	result.ItemName = itemName
	result.ItemIndex = itemIndex
	result.EffectDescriptions = effectNames

	return result
}

// GetInventoryInfo returns display information for a specific inventory item
type InventoryItemInfo struct {
	ItemEntityID ecs.EntityID
	Name         string
	Count        int
	Index        int
	IsThrowable  bool
	Effects      []string
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
	item := GetItemByID(svc.ecsManager.World, itemEntityID)
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

	// Get effect descriptions
	effectNames := GetItemEffectNames(svc.ecsManager.World, item)

	return &InventoryItemInfo{
		ItemEntityID: itemEntityID,
		Name:         itemName,
		Count:        item.Count,
		Index:        itemIndex,
		IsThrowable:  item.HasThrowableAction(),
		Effects:      effectNames,
	}, nil
}

// FilterInventoryByAction returns inventory entries filtered by action type
// This is a service-level wrapper around the existing inventory functions
func (svc *InventoryService) FilterInventoryByAction(playerEntityID ecs.EntityID, actionName string) []InventoryListEntry {
	inv := common.GetComponentTypeByID[*Inventory](svc.ecsManager, playerEntityID, InventoryComponent)
	if inv == nil {
		return []InventoryListEntry{}
	}

	// Use existing inventory system functions
	entries := GetInventoryByAction(svc.ecsManager.World, inv, nil, actionName)

	// Convert []any to []InventoryListEntry
	result := make([]InventoryListEntry, 0, len(entries))
	for _, entry := range entries {
		if listEntry, ok := entry.(InventoryListEntry); ok {
			result = append(result, listEntry)
		}
	}

	return result
}

// GetThrowableItems returns all throwable items in the player's inventory
func (svc *InventoryService) GetThrowableItems(playerEntityID ecs.EntityID) []InventoryListEntry {
	return svc.FilterInventoryByAction(playerEntityID, THROWABLE_ACTION_NAME)
}

// HasThrowableItems checks if the player has any throwable items
func (svc *InventoryService) HasThrowableItems(playerEntityID ecs.EntityID) bool {
	inv := common.GetComponentTypeByID[*Inventory](svc.ecsManager, playerEntityID, InventoryComponent)
	if inv == nil {
		return false
	}

	return HasThrowableItems(svc.ecsManager.World, inv)
}
