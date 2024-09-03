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
	playerEntity       *ecs.Entity
	playerWeapon       *ecs.Entity
	position           *Position
	inventory          *Inventory
	selectedThrowable  *ecs.Entity
	shape              TileBasedShape
	throwableItemIndex int
	throwableItem      *Item
}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerInventory() *Inventory {

	playerInventory := GetComponentType[*Inventory](pl.playerEntity, InventoryComponent)

	return playerInventory
}

// Handles all conversions necessary for updating item throwing information
// The index lets us remove an item one it's thrown
// The shape lets us draw it on the screen
func (pl *PlayerData) PrepareThrowable(itemEntity *ecs.Entity, index int) {

	pl.selectedThrowable = itemEntity
	item := GetComponentType[*Item](pl.selectedThrowable, ItemComponent)
	pl.throwableItem = item

	t := item.GetItemEffect(THROWABLE_NAME).(Throwable)
	pl.throwableItemIndex = index

	pl.shape = t.shape

}

func (pl *PlayerData) ThrowPreparedItem() {

	pl.inventory.RemoveItem(pl.throwableItemIndex)

}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerWeapon() *Weapon {

	weapon := GetComponentType[*Weapon](pl.playerWeapon, WeaponComponent)

	return weapon
}
