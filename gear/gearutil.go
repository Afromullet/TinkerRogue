// Package gear manages all equipment, items, and inventory systems in the roguelike game.
// It handles consumables, item quality, stat effects, and player inventory.
// The package provides utilities for item creation, management, and effect application.
package gear

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// GetItem retrieves the Item component from an entity.
// Returns nil if the entity doesn't have an item component.
func GetItem(e *ecs.Entity) *Item {
	return common.GetComponentType[*Item](e, ItemComponent)
}

// PreparePlayerThrowable sets up a throwable item for the player.
// This helper function bridges the gap between common.PlayerThrowable and gear.Item
// to avoid circular dependencies.
func PreparePlayerThrowable(throwable *common.PlayerThrowable, itemEntity *ecs.Entity, index int) {
	throwable.SelectedThrowable = itemEntity
	throwable.ThrowableItemEntity = itemEntity

	// Extract the Item component and set up the throwing shape
	item := GetItem(itemEntity)
	if item != nil {
		if t := item.GetThrowableAction(); t != nil {
			throwable.ThrowableItemIndex = index
			throwable.ThrowingAOEShape = t.Shape
		}
	}
}

// RemovePlayerThrownItem removes the thrown item from the inventory.
// This helper function allows common.PlayerThrowable to work without importing gear.
func RemovePlayerThrownItem(throwable *common.PlayerThrowable, inv *Inventory) {
	inv.RemoveItem(throwable.ThrowableItemIndex)
}

// GetThrowableItem retrieves the Item component from the player's throwable.
// Returns nil if no throwable is selected or the entity is invalid.
func GetThrowableItem(throwable *common.PlayerThrowable) *Item {
	if throwable.ThrowableItemEntity == nil {
		return nil
	}
	return GetItem(throwable.ThrowableItemEntity)
}
