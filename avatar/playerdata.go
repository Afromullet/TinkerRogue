package avatar

import (
	"game_main/common"
	"game_main/equipment"
	"game_main/graphics"

	"github.com/bytearena/ecs"
)

var PlayerComponent *ecs.Component

type Player struct {
}

type PlayerEquipment struct {
	PlayerWeapon            *ecs.Entity
	PlayerRangedWeapon      *ecs.Entity
	RangedWeaponMaxDistance int
	RangedWeaponAOEShape    graphics.TileBasedShape
}

func (pl *PlayerEquipment) PrepareRangedAttack() {
	wep := common.GetComponentType[*equipment.RangedWeapon](pl.PlayerRangedWeapon, equipment.RangedWeaponComponent)
	pl.RangedWeaponAOEShape = wep.TargetArea
	pl.RangedWeaponMaxDistance = wep.ShootingRange

}

type PlayerThrowable struct {
	SelectedThrowable  *ecs.Entity
	ThrowingAOEShape   graphics.TileBasedShape
	ThrowableItemIndex int
	ThrowableItem      *equipment.Item
}

// Handles all conversions necessary for updating item throwing information
// The index lets us remove an item one it's thrown
// The shape lets us draw it on the screen
func (pl *PlayerThrowable) PrepareThrowable(itemEntity *ecs.Entity, index int) {

	pl.SelectedThrowable = itemEntity

	item := equipment.GetItem(pl.SelectedThrowable)
	pl.ThrowableItem = item

	t := item.ItemEffect(equipment.THROWABLE_NAME).(*equipment.Throwable)
	pl.ThrowableItemIndex = index

	pl.ThrowingAOEShape = t.Shape

}

func (pl *PlayerThrowable) ThrowPreparedItem(inv *equipment.Inventory) {

	inv.RemoveItem(pl.ThrowableItemIndex)

}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerEquipment) GetPlayerWeapon() *equipment.MeleeWeapon {

	weapon := common.GetComponentType[*equipment.MeleeWeapon](pl.PlayerWeapon, equipment.WeaponComponent)

	return weapon
}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerRangedWeapon() *equipment.RangedWeapon {

	weapon := common.GetComponentType[*equipment.RangedWeapon](pl.PlayerRangedWeapon, equipment.RangedWeaponComponent)

	return weapon
}

// Used to keep track of frequently accessed player information.
// Throwing items is an important part of the game, so we store additional information related
// ThrowingAOEShape is the shape that highlights the AOE of the thrown item
// isTargeting is a bool that indicates whether the player is currently selecting a ranged target
type PlayerData struct {
	PlayerEquipment
	PlayerThrowable

	PlayerEntity *ecs.Entity

	Pos *common.Position
	Inv *equipment.Inventory

	Targeting bool
}

func NewPlayerData() PlayerData {
	return PlayerData{}
}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerInventory() *equipment.Inventory {

	playerInventory := common.GetComponentType[*equipment.Inventory](pl.PlayerEntity, equipment.InventoryComponent)

	return playerInventory
}
