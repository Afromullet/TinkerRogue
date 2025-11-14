// Package gear manages all equipment, items, and inventory systems in the roguelike game.
// It handles consumables, item quality, stat effects, and player inventory.
// The package provides utilities for item creation, management, and effect application.
package gear

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

var itemTag = ecs.BuildTag(ItemComponent)

// FindItemEntityByID finds an item entity by its ID using ECS queries (ECS best practice)
// Returns nil if not found
// Note: This function expects an ecs.Manager directly (not EntityManager) for compatibility
// with existing call sites. The generic helper in common package works with EntityManager.
func FindItemEntityByID(manager *ecs.Manager, entityID ecs.EntityID) *ecs.Entity {
	for _, result := range manager.Query(itemTag) {
		if result.Entity.GetID() == entityID {
			return result.Entity
		}
	}
	return nil
}

// GetItemByID retrieves the Item component from an entity ID (ECS best practice)
// Returns nil if the entity doesn't exist or doesn't have an item component
func GetItemByID(manager *ecs.Manager, entityID ecs.EntityID) *Item {
	entity := FindItemEntityByID(manager, entityID)
	if entity == nil {
		return nil
	}
	return common.GetComponentType[*Item](entity, ItemComponent)
}

// ============================================================================
// Item Query/Helper Functions (ECS System-based)
// ============================================================================

// FindPropertiesEntityByID finds a properties entity by its ID using ECS queries
// Properties entities don't have a specific tag, so we query all entities
// Note: This function expects an ecs.Manager directly (not EntityManager) for compatibility
// with existing call sites. The generic helper in common package works with EntityManager.
func FindPropertiesEntityByID(manager *ecs.Manager, entityID ecs.EntityID) *ecs.Entity {
	if entityID == 0 {
		return nil
	}
	for _, result := range manager.Query(ecs.BuildTag()) {
		if result.Entity.GetID() == entityID {
			return result.Entity
		}
	}
	return nil
}

// GetItemEffectNames returns the names of all effects on an item (system function)
func GetItemEffectNames(manager *ecs.Manager, item *Item) []string {
	names := make([]string, 0)

	if item.Properties == 0 {
		return names
	}

	// Get the properties entity to check for effects
	propsEntity := FindPropertiesEntityByID(manager, item.Properties)
	if propsEntity == nil {
		return names
	}

	for _, c := range AllItemEffects {
		data, ok := propsEntity.GetComponentData(c)
		if ok {
			d := data.(*StatusEffects)
			names = append(names, StatusEffectName(d))
		}
	}
	return names
}

// HasAllEffects checks if an item has all specified effects (system function)
func HasAllEffects(manager *ecs.Manager, item *Item, effectsToCheck ...StatusEffects) bool {
	if len(effectsToCheck) == 0 {
		return true
	}

	for _, eff := range effectsToCheck {
		if !HasEffect(manager, item, eff) {
			return false
		}
	}

	return true
}

// HasEffect checks if an item has a specific effect (system function)
func HasEffect(manager *ecs.Manager, item *Item, effectToCheck StatusEffects) bool {
	names := GetItemEffectNames(manager, item)
	comp := effectToCheck.StatusEffectName()

	for _, n := range names {
		if n == comp {
			return true
		}
	}

	return false
}

// HasAction checks if an item has a specific action (system function)
func HasAction(item *Item, actionName string) bool {
	for _, action := range item.Actions {
		if action.ActionName() == actionName {
			return true
		}
	}
	return false
}
