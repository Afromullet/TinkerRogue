package avatar

import (
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"

	"github.com/bytearena/ecs"
)

var PlayerComponent *ecs.Component

type Player struct {
}

// Tracks state for user input handling.
type PlayerInputStates struct {
	IsThrowing  bool
	IsShooting  bool
	HasKeyInput bool //Tells us whether the player pressed any key.
}

type PlayerEquipment struct {
	PlayerWeapon            *ecs.Entity
	PlayerRangedWeapon      *ecs.Entity
	RangedWeaponMaxDistance int
	RangedWeaponAOEShape    graphics.TileBasedShape
}

func (pl *PlayerEquipment) PrepareRangedAttack() {
	wep := common.GetComponentType[*gear.RangedWeapon](pl.PlayerRangedWeapon, gear.RangedWeaponComponent)
	pl.RangedWeaponAOEShape = wep.TargetArea
	pl.RangedWeaponMaxDistance = wep.ShootingRange

}

// Let's us modify the visual cue for the throwing weapons AOE and change the inventory once the item is thrown
type PlayerThrowable struct {
	SelectedThrowable  *ecs.Entity
	ThrowingAOEShape   graphics.TileBasedShape
	ThrowableItemIndex int
	ThrowableItem      *gear.Item
}

// Throwing items needs to both display information to the user and use the players inventory
func (pl *PlayerThrowable) PrepareThrowable(itemEntity *ecs.Entity, index int) {

	pl.SelectedThrowable = itemEntity

	item := gear.GetItem(pl.SelectedThrowable)
	pl.ThrowableItem = item

	t := item.ItemEffect(gear.THROWABLE_NAME).(*gear.Throwable)
	pl.ThrowableItemIndex = index

	pl.ThrowingAOEShape = t.Shape

}

func (pl *PlayerThrowable) RemoveThrownItem(inv *gear.Inventory) {

	inv.RemoveItem(pl.ThrowableItemIndex)

}

// All of the player information needs to be easily accessible.
// Everything that needs the PlayerData currently does it through a parameter rather than acecssing it globally.
// A global could probably work and reduce the number of parameters a lot of the functions take
// But for now, this works.
// Maybe it's a good use case for a Singleton, but I will worry about that later.
type PlayerData struct {
	PlayerEquipment
	PlayerThrowable
	PlayerInputStates

	PlayerEntity *ecs.Entity

	Pos *common.Position
	Inv *gear.Inventory
}

// Handles the conversions from the component type to the struct type
func (pl *PlayerData) GetPlayerInventory() *gear.Inventory {

	playerInventory := common.GetComponentType[*gear.Inventory](pl.PlayerEntity, gear.InventoryComponent)

	return playerInventory
}

// Handles the conversions from the component type to the struct type
func (pl *PlayerData) GetPlayerAttributes() *common.Attributes {

	attr := common.GetComponentType[*common.Attributes](pl.PlayerEntity, common.AttributeComponent)

	return attr
}

func (pl *PlayerData) GetPlayerRangedWeapon() *gear.RangedWeapon {

	weapon := common.GetComponentType[*gear.RangedWeapon](pl.PlayerRangedWeapon, gear.RangedWeaponComponent)

	return weapon
}

func (pl *PlayerEquipment) GetPlayerWeapon() *gear.MeleeWeapon {

	weapon := common.GetComponentType[*gear.MeleeWeapon](pl.PlayerWeapon, gear.MeleeWeaponComponent)

	return weapon
}
