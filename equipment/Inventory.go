package equipment

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
// This is used for displaying the item effects to the player.
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

// Gets all Melee Weapons, Ranged Weapons, and Armor for displaying
func (inv *Inventory) GetEquipmentForDisplay(indicesToSelect []int) []any {

	inventoryItems := make([]any, 0)

	for index, entity := range inv.InventoryContent {

		itemName := common.GetComponentType[*common.Name](entity, common.NameComponent)
		itemComp := GetItem(entity)

		if entity.HasComponent(ArmorComponent) {

			inventoryItems = append(inventoryItems, InventoryListEntry{
				index,
				itemName.NameStr,
				itemComp.Count})

		} else if entity.HasComponent(RangedWeaponComponent) {

			inventoryItems = append(inventoryItems, InventoryListEntry{
				index,
				itemName.NameStr,
				itemComp.Count})

		} else if entity.HasComponent(MeleeWeaponComponent) {

			inventoryItems = append(inventoryItems, InventoryListEntry{
				index,
				itemName.NameStr,
				itemComp.Count})

		}

	}

	return inventoryItems

}

// Used for displaying the inventory to the player. Returns a list the ebitenui list widgets expects
// The list contains the index in the inventory, the name, and the count of the item.
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
