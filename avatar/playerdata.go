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

type PlayerInputStates struct {
	IsThrowing  bool
	IsShooting  bool
	HasKeyInput bool //Tells us whether the player pressed a key in the avatarmovement
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

// Let's us modify the visual cue for the throwing weapons AOE and change the inventory once the item is thrown
type PlayerThrowable struct {
	SelectedThrowable  *ecs.Entity
	ThrowingAOEShape   graphics.TileBasedShape
	ThrowableItemIndex int
	ThrowableItem      *equipment.Item
}

// Throwing items needs to both display information to the user and use the players inventory
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
	Inv *equipment.Inventory
}

func NewPlayerData() PlayerData {
	return PlayerData{}
}

func (pl *PlayerData) GetPlayerInventory() *equipment.Inventory {

	playerInventory := common.GetComponentType[*equipment.Inventory](pl.PlayerEntity, equipment.InventoryComponent)

	return playerInventory
}

func (pl *PlayerData) GetPlayerRangedWeapon() *equipment.RangedWeapon {

	weapon := common.GetComponentType[*equipment.RangedWeapon](pl.PlayerRangedWeapon, equipment.RangedWeaponComponent)

	return weapon
}

func (pl *PlayerEquipment) GetPlayerWeapon() *equipment.MeleeWeapon {

	weapon := common.GetComponentType[*equipment.MeleeWeapon](pl.PlayerWeapon, equipment.MeleeWeaponComponent)

	return weapon
}
