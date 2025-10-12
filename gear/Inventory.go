package gear

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// This is used to display the inventory inside of windows for the UI
// EbitenUI needs to build a list for display - this holds the list elements
type InventoryListEntry struct {
	Index int
	Name  string
	Count int
}

// Items are stored as entities so that we can add Status Effects/Properties
type Inventory struct {
	InventoryContent []*ecs.Entity
}

// If the item already exists, it increments the count by 1.
// Otherwise it just sets the count to one
func (inv *Inventory) AddItem(entityToAdd *ecs.Entity) {

	if entityToAdd == nil {
		return
	}
	newItemName := common.GetComponentType[*common.Name](entityToAdd, common.NameComponent).NameStr
	exists := false

	for _, entity := range inv.InventoryContent {

		itemName := common.GetComponentType[*common.Name](entity, common.NameComponent).NameStr

		if itemName == newItemName {
			exists = true

			GetItem(entity).IncrementCount()

			break
		}
	}

	if !exists {
		itemComp := GetItem(entityToAdd)
		itemComp.Count = 1
		inv.InventoryContent = append(inv.InventoryContent, entityToAdd)

	}

}

// Does not remove the item from the inventory, just returns the entity of the item
func (inv *Inventory) GetItem(index int) (*ecs.Entity, error) {
	if index < 0 || index >= len(inv.InventoryContent) {
		return nil, fmt.Errorf("index out of range")
	}
	return inv.InventoryContent[index], nil
}

// If there's more than one of an item, it decrements the Item Count
// Otherwise it removes the item from the inventory
func (inv *Inventory) RemoveItem(index int) {

	item, err := inv.GetItem(index)

	if err == nil {

		itemComp := GetItem(item)

		itemComp.DecrementCount()

		if itemComp.Count <= 0 {

			inv.InventoryContent = append(inv.InventoryContent[:index], inv.InventoryContent[index+1:]...)

		}
	}

}

// Returns the names of the Item Effects
// Used for displaying item effects in the GUI
func (inv *Inventory) EffectNames(index int) ([]string, error) {

	entity, err := inv.GetItem(index)

	if err != nil {
		return nil, fmt.Errorf("failed to get item by index: %w", err)
	}

	itemComp := GetItem(entity)

	if itemComp == nil {
		return nil, fmt.Errorf("failed to get component data: %w", err)

	}

	return itemComp.GetEffectNames(), nil

}

// Builds the list that's needed for displaying the inventory to the player
// itemPropertiesFilter StatusEffects lets us filter by status effects
// itemActionFilter string lets us filter by item actions (like "Throwable")
func (inv *Inventory) GetInventoryForDisplay(indicesToSelect []int, itemPropertiesFilter ...StatusEffects) []any {

	inventoryItems := make([]any, 0)

	if len(indicesToSelect) == 0 {
		for index, entity := range inv.InventoryContent {

			itemName := common.GetComponentType[*common.Name](entity, common.NameComponent)
			itemComp := GetItem(entity)

			if itemComp.HasAllEffects(itemPropertiesFilter...) {

				inventoryItems = append(inventoryItems, InventoryListEntry{
					index,
					itemName.NameStr,
					itemComp.Count})
			}

		}
	} else {
		for _, index := range indicesToSelect {
			entity := inv.InventoryContent[index]
			itemName := common.GetComponentType[*common.Name](entity, common.NameComponent)
			itemComp := GetItem(entity)

			if itemComp.HasAllEffects(itemPropertiesFilter...) {
				inventoryItems = append(inventoryItems, InventoryListEntry{
					index,
					itemName.NameStr,
					itemComp.Count})
			}

		}

	}

	return inventoryItems

}

// GetInventoryByAction filters inventory items by their ItemAction capabilities
// actionName is the name of the action to filter by (e.g., "Throwable")
func (inv *Inventory) GetInventoryByAction(indicesToSelect []int, actionName string) []any {
	inventoryItems := make([]any, 0)

	if len(indicesToSelect) == 0 {
		for index, entity := range inv.InventoryContent {
			itemName := common.GetComponentType[*common.Name](entity, common.NameComponent)
			itemComp := GetItem(entity)

			if itemComp.HasAction(actionName) {
				inventoryItems = append(inventoryItems, InventoryListEntry{
					index,
					itemName.NameStr,
					itemComp.Count})
			}
		}
	} else {
		for _, index := range indicesToSelect {
			entity := inv.InventoryContent[index]
			itemName := common.GetComponentType[*common.Name](entity, common.NameComponent)
			itemComp := GetItem(entity)

			if itemComp.HasAction(actionName) {
				inventoryItems = append(inventoryItems, InventoryListEntry{
					index,
					itemName.NameStr,
					itemComp.Count})
			}
		}
	}

	return inventoryItems
}

// GetThrowableItems returns all items that have throwable actions
func (inv *Inventory) GetThrowableItems(indicesToSelect []int) []any {
	return inv.GetInventoryByAction(indicesToSelect, THROWABLE_ACTION_NAME)
}

// HasItemsWithAction checks if the inventory contains any items with the specified action
func (inv *Inventory) HasItemsWithAction(actionName string) bool {
	for _, entity := range inv.InventoryContent {
		itemComp := GetItem(entity)
		if itemComp.HasAction(actionName) {
			return true
		}
	}
	return false
}

// HasThrowableItems checks if the inventory contains any throwable items
func (inv *Inventory) HasThrowableItems() bool {
	return inv.HasItemsWithAction(THROWABLE_ACTION_NAME)
}
