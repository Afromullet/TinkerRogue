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
