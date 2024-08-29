package main

import (
	"github.com/bytearena/ecs"
)

var player *ecs.Component

type Player struct {
}

// There's only one player, so we can store frequently used component data in PlayerData
// Throwing items is a big part of the game, so we store the selected throwable item here
// Also storing the index so we can remove it from the inventory once it's thrown
type PlayerData struct {
	playerEntity       *ecs.Entity
	playerWeapon       *ecs.Entity
	position           *Position
	inventory          *Inventory
	selectedThrowable  *ecs.Entity
	shape              TileBasedShape
	throwableItemIndex int
}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerInventory() *Inventory {

	playerInventory := GetComponentStruct[*Inventory](pl.playerEntity, InventoryComponent)

	return playerInventory
}

func (pl *PlayerData) PrepareThrowable(itemEntity *ecs.Entity, index int) {

	pl.selectedThrowable = itemEntity
	item := GetComponentStruct[*Item](pl.selectedThrowable, ItemComponent)

	t := item.GetItemProperty(THROWABLE_NAME).(throwable)
	pl.throwableItemIndex = index

	pl.shape = t.shape

}

func (pl *PlayerData) ThrowPreparedItem() {

}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerWeapon() *Weapon {

	weapon := GetComponentStruct[*Weapon](pl.playerWeapon, WeaponComponent)

	return weapon
}

func (pl *PlayerData) GetPixelsFromPosition(gameMap *GameMap) []int {

	ind := GetIndexFromXY(pl.position.X, pl.position.Y)

	return []int{gameMap.Tiles[ind].PixelX, gameMap.Tiles[ind].PixelY}

}
