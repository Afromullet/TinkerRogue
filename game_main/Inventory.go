package main

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// This is used to display the inventory inside of windows for the UI
// TODO later replace this with the item information
type InventoryListEntry struct {
	index int
	Name  string
	count int
}

var inventorySize = 20 //todo get rid of this

type Inventory struct {
	InventoryContent []*ecs.Entity
}

// the Item type stores a "count" which is incremented if the item exists in the inventory
func (inv *Inventory) AddItem(entityToAdd *ecs.Entity) {
	// Dereference the slice pointer and use append
	newItemName := ComponentType[*Name](entityToAdd, nameComponent).NameStr
	exists := false

	for _, entity := range inv.InventoryContent {

		itemName := ComponentType[*Name](entity, nameComponent).NameStr

		if itemName == newItemName {
			exists = true
			ComponentType[*Item](entity, ItemComponent).IncrementCount()
			break
		}
	}

	if !exists {
		itemComp := ComponentType[*Item](entityToAdd, ItemComponent)
		itemComp.Count = 1
		inv.InventoryContent = append(inv.InventoryContent, entityToAdd)

	}

}

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

		itemComp := ComponentType[*Item](item, ItemComponent)

		itemComp.DecrementCount()

		if itemComp.Count <= 0 {

			inv.InventoryContent = append(inv.InventoryContent[:index], inv.InventoryContent[index+1:]...)

		}
	}

}

// Returns the names of the Item Effects
// This is used for displaying the item effects to the player.
func (inv *Inventory) EffectNames(index int) ([]string, error) {

	entity, err := inv.GetItem(index)

	if err != nil {
		return nil, fmt.Errorf("failed to get item by index: %w", err)
	}

	itemComp := ComponentType[*Item](entity, ItemComponent)

	if itemComp == nil {
		return nil, fmt.Errorf("failed to get component data: %w", err)

	}

	return itemComp.GetEffectNames(), nil

}

// Used for displaying the inventory to the player. Returns a list the ebitenui list widgets expects
// The list contains the index in the inventory, the name, and the count of the item.
func (inv *Inventory) GetInventoryForDisplay(indicesToSelect []int, itemPropertiesFilter ...Effects) []any {

	inventoryItems := make([]any, 0, inventorySize)

	if len(indicesToSelect) == 0 {
		for index, entity := range inv.InventoryContent {

			itemName := ComponentType[*Name](entity, nameComponent)
			itemComp := ComponentType[*Item](entity, ItemComponent)

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
			itemName := ComponentType[*Name](entity, nameComponent)
			itemComp := ComponentType[*Item](entity, ItemComponent)

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
