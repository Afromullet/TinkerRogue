package main

import (
	"github.com/bytearena/ecs"
)

var player *ecs.Component

type Player struct {
}

// There's only one player, so we can store frequently used component data in PlayerData
type PlayerData struct {
	playerEntity *ecs.Entity
	playerWeapon *ecs.Entity
	position     *Position
	inventory    *Inventory
}

// Helper function to make it less tedious to get the inventory
func (playerData *PlayerData) GetPlayerInventory() *Inventory {

	playerInventory := GetComponentStruct[*Inventory](playerData.playerEntity, InventoryComponent)

	return playerInventory
}

// Helper function to make it less tedious to get the inventory
func (playerData *PlayerData) GetPlayerWeapon() *Weapon {

	weapon := GetComponentStruct[*Weapon](playerData.playerWeapon, WeaponComponent)

	return weapon
}
