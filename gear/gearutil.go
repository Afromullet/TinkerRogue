// Package gear manages all equipment, items, and inventory systems in the roguelike game.
// It handles consumables, item quality, and player inventory.
// The package provides utilities for item creation and management.
package gear

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// GetItemByID retrieves the Item component from an entity ID (ECS best practice)
// Returns nil if the entity doesn't exist or doesn't have an item component
func GetItemByID(manager *common.EntityManager, entityID ecs.EntityID) *Item {
	entity := manager.FindEntityByID(entityID)
	if entity == nil {
		return nil
	}
	return common.GetComponentType[*Item](entity, ItemComponent)
}
