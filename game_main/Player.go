package main

import (
	"github.com/bytearena/ecs"
)

var player *ecs.Component

type Player struct {
}

// Used to keep track of frequently accessed player information.
// Throwing items is an important part of the game, so we store additional information related
// TO throwing
type PlayerData struct {
	PlayerEntity       *ecs.Entity
	PlayerWeapon       *ecs.Entity
	position           *Position
	inventory          *Inventory
	SelectedThrowable  *ecs.Entity
	Shape              TileBasedShape
	ThrowableItemIndex int
	ThrowableItem      *Item
}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerInventory() *Inventory {

	playerInventory := ComponentType[*Inventory](pl.PlayerEntity, InventoryComponent)

	return playerInventory
}

// Handles all conversions necessary for updating item throwing information
// The index lets us remove an item one it's thrown
// The shape lets us draw it on the screen
func (pl *PlayerData) PrepareThrowable(itemEntity *ecs.Entity, index int) {

	pl.SelectedThrowable = itemEntity
	item := ComponentType[*Item](pl.SelectedThrowable, ItemComponent)
	pl.ThrowableItem = item

	t := item.ItemEffect(THROWABLE_NAME).(*Throwable)
	pl.ThrowableItemIndex = index

	pl.Shape = t.Shape

}

func (pl *PlayerData) ThrowPreparedItem() {

	pl.inventory.RemoveItem(pl.ThrowableItemIndex)

}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerWeapon() *Weapon {

	weapon := ComponentType[*Weapon](pl.PlayerWeapon, WeaponComponent)

	return weapon
}
