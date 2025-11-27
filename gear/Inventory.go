package gear

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

var (
	InventoryComponent *ecs.Component
)

// This is used to display the inventory inside of windows for the UI
// EbitenUI needs to build a list for display - this holds the list elements
type InventoryListEntry struct {
	Index int
	Name  string
	Count int
}

// Inventory is a pure data component storing item entity IDs
// Use InventorySystem functions for all logic operations
type Inventory struct {
	ItemEntityIDs []ecs.EntityID // ECS best practice: use EntityID, not pointers
}

// InventorySystem - System-based functions for inventory management
// Following ECS best practices: all logic in systems, not component methods

// AddItem adds an item to the inventory (system function)
// If the item already exists, it increments the count by 1.
// Otherwise it sets the count to 1 and adds it to the inventory.
func AddItem(manager *ecs.Manager, inv *Inventory, itemEntityID ecs.EntityID) {
	itemEntity := common.FindEntityByIDInManager(manager, itemEntityID)
	if itemEntity == nil {
		return
	}

	newItemName := common.GetComponentType[*common.Name](itemEntity, common.NameComponent).NameStr
	exists := false

	// Check if item already exists in inventory
	for _, existingID := range inv.ItemEntityIDs {
		existingEntity := common.FindEntityByIDInManager(manager, existingID)
		if existingEntity == nil {
			continue
		}

		existingName := common.GetComponentType[*common.Name](existingEntity, common.NameComponent).NameStr
		if existingName == newItemName {
			exists = true
			// Increment count on existing item
			itemComp := GetItemByID(manager, existingID)
			if itemComp != nil {
				itemComp.Count++
			}
			break
		}
	}

	if !exists {
		// Add new item to inventory
		itemComp := GetItemByID(manager, itemEntityID)
		if itemComp != nil {
			itemComp.Count = 1
		}
		inv.ItemEntityIDs = append(inv.ItemEntityIDs, itemEntityID)
	}
}

// GetItemEntityID returns the entity ID at the specified inventory index (system function)
// Returns 0 and error if index is out of range
func GetItemEntityID(inv *Inventory, index int) (ecs.EntityID, error) {
	if index < 0 || index >= len(inv.ItemEntityIDs) {
		return 0, fmt.Errorf("index out of range")
	}
	return inv.ItemEntityIDs[index], nil
}

// RemoveItem removes an item from inventory (system function)
// If there's more than one of an item, it decrements the Item Count
// Otherwise it removes the item from the inventory
func RemoveItem(manager *ecs.Manager, inv *Inventory, index int) {
	itemID, err := GetItemEntityID(inv, index)
	if err != nil {
		return
	}

	itemComp := GetItemByID(manager, itemID)
	if itemComp == nil {
		return
	}

	itemComp.Count--

	if itemComp.Count <= 0 {
		// Remove from inventory slice
		inv.ItemEntityIDs = append(inv.ItemEntityIDs[:index], inv.ItemEntityIDs[index+1:]...)
	}
}

// GetInventoryForDisplay builds the list needed for displaying the inventory to the player (system function)
// itemPropertiesFilter StatusEffects lets us filter by status effects
func GetInventoryForDisplay(manager *ecs.Manager, inv *Inventory, indicesToSelect []int, itemPropertiesFilter ...StatusEffects) []any {
	inventoryItems := make([]any, 0)

	if len(indicesToSelect) == 0 {
		// Show all items
		for index, itemID := range inv.ItemEntityIDs {
			itemEntity := common.FindEntityByIDInManager(manager, itemID)
			if itemEntity == nil {
				continue
			}

			itemName := common.GetComponentType[*common.Name](itemEntity, common.NameComponent)
			itemComp := GetItemByID(manager, itemID)

			if itemComp != nil && HasAllEffects(manager, itemComp, itemPropertiesFilter...) {
				inventoryItems = append(inventoryItems, InventoryListEntry{
					Index: index,
					Name:  itemName.NameStr,
					Count: itemComp.Count,
				})
			}
		}
	} else {
		// Show selected indices only
		for _, index := range indicesToSelect {
			if index < 0 || index >= len(inv.ItemEntityIDs) {
				continue
			}

			itemID := inv.ItemEntityIDs[index]
			itemEntity := common.FindEntityByIDInManager(manager, itemID)
			if itemEntity == nil {
				continue
			}

			itemName := common.GetComponentType[*common.Name](itemEntity, common.NameComponent)
			itemComp := GetItemByID(manager, itemID)

			if itemComp != nil && HasAllEffects(manager, itemComp, itemPropertiesFilter...) {
				inventoryItems = append(inventoryItems, InventoryListEntry{
					Index: index,
					Name:  itemName.NameStr,
					Count: itemComp.Count,
				})
			}
		}
	}

	return inventoryItems
}

// GetInventoryByAction filters inventory items by their ItemAction capabilities (system function)
// actionName is the name of the action to filter by (e.g., "Throwable")
func GetInventoryByAction(manager *ecs.Manager, inv *Inventory, indicesToSelect []int, actionName string) []any {
	inventoryItems := make([]any, 0)

	if len(indicesToSelect) == 0 {
		// Show all items with the specified action
		for index, itemID := range inv.ItemEntityIDs {
			itemEntity := common.FindEntityByIDInManager(manager, itemID)
			if itemEntity == nil {
				continue
			}

			itemName := common.GetComponentType[*common.Name](itemEntity, common.NameComponent)
			itemComp := GetItemByID(manager, itemID)

			if itemComp != nil && HasAction(itemComp, actionName) {
				inventoryItems = append(inventoryItems, InventoryListEntry{
					Index: index,
					Name:  itemName.NameStr,
					Count: itemComp.Count,
				})
			}
		}
	} else {
		// Show selected indices with the specified action
		for _, index := range indicesToSelect {
			if index < 0 || index >= len(inv.ItemEntityIDs) {
				continue
			}

			itemID := inv.ItemEntityIDs[index]
			itemEntity := common.FindEntityByIDInManager(manager, itemID)
			if itemEntity == nil {
				continue
			}

			itemName := common.GetComponentType[*common.Name](itemEntity, common.NameComponent)
			itemComp := GetItemByID(manager, itemID)

			if itemComp != nil && HasAction(itemComp, actionName) {
				inventoryItems = append(inventoryItems, InventoryListEntry{
					Index: index,
					Name:  itemName.NameStr,
					Count: itemComp.Count,
				})
			}
		}
	}

	return inventoryItems
}

// GetThrowableItems returns all items that have throwable actions (system function)
func GetThrowableItems(manager *ecs.Manager, inv *Inventory, indicesToSelect []int) []any {
	return GetInventoryByAction(manager, inv, indicesToSelect, THROWABLE_ACTION_NAME)
}

// HasItemsWithAction checks if the inventory contains any items with the specified action (system function)
func HasItemsWithAction(manager *ecs.Manager, inv *Inventory, actionName string) bool {
	for _, itemID := range inv.ItemEntityIDs {
		itemComp := GetItemByID(manager, itemID)
		if itemComp != nil && HasAction(itemComp, actionName) {
			return true
		}
	}
	return false
}

// HasThrowableItems checks if the inventory contains any throwable items (system function)
func HasThrowableItems(manager *ecs.Manager, inv *Inventory) bool {
	return HasItemsWithAction(manager, inv, THROWABLE_ACTION_NAME)
}
