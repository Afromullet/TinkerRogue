package main

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// TODO later replace this with the item information
type InventoryListEntry struct {
	index int
	Name  string
	count int
}

// An "Item" is anything with an Item component. An Item is any Entity with an Item Component.
// This means that we have to store a slice of entities, since Item isn't the only proprety
type Inventory struct {
	InventoryContent *[]ecs.Entity
}

var inventorySize = 20 //todo get rid of this

// AddItemToInventory adds an entity to the inventory
func (inventory *Inventory) AddItemToInventory(entityToAdd *ecs.Entity) {
	// Dereference the slice pointer and use append
	newItemName := GetComponentStruct[*Name](entityToAdd, nameComponent).NameStr
	exists := false

	for _, entity := range *inventory.InventoryContent {

		itemName := GetComponentStruct[*Name](&entity, nameComponent).NameStr

		if itemName == newItemName {
			exists = true
			GetComponentStruct[*Item](&entity, ItemComponent).IncrementCount()
			break
		}
	}

	if !exists {
		itemComp := GetComponentStruct[*Item](entityToAdd, ItemComponent)
		itemComp.count = 1
		*inventory.InventoryContent = append(*inventory.InventoryContent, *entityToAdd)

	}

}

// Returns a slice of strings of the Item Component names. These come from the CommonProperties
func (inv *Inventory) GetPropertyNames(index int) ([]string, error) {

	entity, err := inv.GetItemByIndex(index)

	if err != nil {
		return nil, fmt.Errorf("failed to get item by index: %w", err)
	}

	itemComp := GetComponentStruct[*Item](entity, ItemComponent)

	if itemComp == nil {
		return nil, fmt.Errorf("failed to get component data: %w", err)

	}

	return itemComp.GetPropertyNames(), nil

}

func (inv *Inventory) GetItemByIndex(index int) (*ecs.Entity, error) {
	if index < 0 || index >= len(*inv.InventoryContent) {
		return nil, fmt.Errorf("index out of range")
	}
	return &(*inv.InventoryContent)[index], nil
}

// Used for displaying the inventory to the player by returning a list for the ebitneui list widget
// Returns a slice of InventoryListEntries, which contain the inventory index, name, and item count
// If indicesSelected is empty, it returns the entire inventory. Otherwise it returns the items specified by the indicesToSelect slice
func (inventory *Inventory) GetInventoryForDisplay(indicesToSelect []int, itemPropertiesFilter ...ItemProperty) []any {

	inventoryItems := make([]any, 0, inventorySize)

	if len(indicesToSelect) == 0 {
		for index, entity := range *inventory.InventoryContent {

			itemName := GetComponentStruct[*Name](&entity, nameComponent)
			itemComp := GetComponentStruct[*Item](&entity, ItemComponent)

			if itemComp.HasAllProperties(itemPropertiesFilter...) {

				inventoryItems = append(inventoryItems, InventoryListEntry{
					index,
					itemName.NameStr,
					itemComp.count})
			}

		}
	} else {
		for _, index := range indicesToSelect {
			entity := (*inventory.InventoryContent)[index]
			itemName := GetComponentStruct[*Name](&entity, nameComponent)
			itemComp := GetComponentStruct[*Item](&entity, ItemComponent)

			if itemComp.HasAllProperties(itemPropertiesFilter...) {
				inventoryItems = append(inventoryItems, InventoryListEntry{
					index,
					itemName.NameStr,
					itemComp.count})
			}

		}

	}

	return inventoryItems

}
