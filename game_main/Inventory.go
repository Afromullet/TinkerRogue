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
//todo, search the inventory to see if the item already exists. If it does, increase the count by 1

func (inventory *Inventory) AddItemToInventory(entityToAdd *ecs.Entity) {
	// Dereference the slice pointer and use append

	newItemName, _ := entityToAdd.GetComponentData(nameComponent)
	newItemName = newItemName.(*Name).NameStr
	exists := false

	for _, entity := range *inventory.InventoryContent {

		itemName, _ := entity.GetComponentData(nameComponent)
		itemName = itemName.(*Name).NameStr

		if itemName == newItemName {
			exists = true
			itemComp, _ := entity.GetComponentData(ItemComponent)
			itemComp.(*Item).IncrementCount()
			break
		}
	}

	if !exists {
		itemComp, _ := entityToAdd.GetComponentData(ItemComponent)
		itemComp.(*Item).count = 1

		*inventory.InventoryContent = append(*inventory.InventoryContent, *entityToAdd)

	}

}

// Returns a slice of strings of the Item Component names. These come from the CommonProperties
func (inv *Inventory) GetPropertyNames(index int) ([]string, error) {

	entity, err := inv.GetItemByIndex(index)

	if err != nil {
		return nil, fmt.Errorf("failed to get item by index: %w", err)
	}

	itemComp, ok := entity.GetComponentData(ItemComponent)

	if !ok {
		return nil, fmt.Errorf("failed to get component data: %w", err)

	}

	return itemComp.(*Item).GetPropertyNames(), nil

}

func (inv *Inventory) GetItemByIndex(index int) (*ecs.Entity, error) {
	if index < 0 || index >= len(*inv.InventoryContent) {
		return nil, fmt.Errorf("index out of range")
	}
	return &(*inv.InventoryContent)[index], nil
}

// Used for displaying the inventory to the player
// If indicesSelected is empty, it returns the entire inventory.
// Otherwise it returns the items specified by the indicesToSelect slice
// Figure out how to do this in an MVC way. Should this be handled by the controller?
func (inventory *Inventory) GetInventoryForDisplay(indicesToSelect []int) []any {

	inventoryItems := make([]any, 0, inventorySize)

	if len(indicesToSelect) == 0 {
		for index, entity := range *inventory.InventoryContent {
			itemName, _ := entity.GetComponentData(nameComponent)
			itemComp, _ := entity.GetComponentData(ItemComponent)

			inventoryItems = append(inventoryItems, InventoryListEntry{
				index,
				itemName.(*Name).NameStr,
				itemComp.(*Item).count})

		}
	} else {
		for _, index := range indicesToSelect {
			entity := (*inventory.InventoryContent)[index]
			itemName, _ := entity.GetComponentData(nameComponent)
			itemComp, _ := entity.GetComponentData(ItemComponent)
			inventoryItems = append(inventoryItems, InventoryListEntry{
				index,
				itemName.(*Name).NameStr,
				itemComp.(*Item).count})

		}

	}

	return inventoryItems

}
