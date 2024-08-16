package main

import (
	"fmt"

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
	inv, err := playerData.playerEntity.GetComponentData(InventoryComponent)

	if !err {
		// Handle the error (e.g., log it, return nil, or handle accordingly)
		fmt.Println("Error getting player inventory:", err)
		return nil // Return nil if you can't get the inventory
	}

	playerInventory := inv.(*Inventory)
	return playerInventory
}
