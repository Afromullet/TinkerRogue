package avatar

import (
	"fmt"
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"

	"github.com/bytearena/ecs"
)

var PlayerComponent *ecs.Component

type Player struct {
}

// Used in the input package to determine when button inputs are valid.
type PlayerInputStates struct {
	IsThrowing  bool
	IsShooting  bool
	HasKeyInput bool //Tells us whether the player pressed any key.
	InfoMeuOpen bool
}

type PlayerEquipment struct {
	EqMeleeWeapon           *ecs.Entity
	EqRangedWeapon          *ecs.Entity
	RangedWeaponMaxDistance int
	RangedWeaponAOEShape    graphics.TileBasedShape
	EqArmor                 *ecs.Entity //Will call this PlayerArmor soon. Replacing the gear.armor type gradually
}

func (pl *PlayerEquipment) RangedWeapon() *gear.RangedWeapon {

	return common.GetComponentType[*gear.RangedWeapon](pl.EqRangedWeapon, gear.RangedWeaponComponent)
}

func (pl *PlayerEquipment) MeleeWeapon() *gear.MeleeWeapon {

	return common.GetComponentType[*gear.MeleeWeapon](pl.EqMeleeWeapon, gear.MeleeWeaponComponent)
}

func (pl *PlayerEquipment) Armor() *gear.Armor {

	return common.GetComponentType[*gear.Armor](pl.EqArmor, gear.ArmorComponent)
}

// Does not remove an equipped item. Todo check if the item kind is already equipped, and if it is,
// Remove it. Low priority. Currently, the part of the GUI that handles this removes the item,
// But it's something that can be simplified
func (pl *PlayerEquipment) EquipItem(equipment *ecs.Entity, playerEntity *ecs.Entity) {

	switch gear.KindOfItem(equipment) {
	case gear.ArmorType:
		playerEntity.AddComponent(gear.ArmorComponent, equipment)
		pl.EqArmor = equipment
	case gear.MeleeWeaponType:
		playerEntity.AddComponent(gear.MeleeWeaponComponent, equipment)
		pl.EqMeleeWeapon = equipment
	case gear.RangedWeaponType:
		playerEntity.AddComponent(gear.RangedWeaponComponent, equipment)
		pl.EqRangedWeapon = equipment
	default:
		fmt.Println("Invalid item equipped")
	}

}

// Helper function for displaying the area of effect when the player is selecting a target for a ranged attack
func (pl *PlayerEquipment) PrepareRangedAttack() {
	wep := common.GetComponentType[*gear.RangedWeapon](pl.EqRangedWeapon, gear.RangedWeaponComponent)
	pl.RangedWeaponAOEShape = wep.TargetArea
	pl.RangedWeaponMaxDistance = wep.ShootingRange

}

// State and variable tracker for throwing items. Inventory, drawing, and input uses this.
type PlayerThrowable struct {
	SelectedThrowable  *ecs.Entity
	ThrowingAOEShape   graphics.TileBasedShape
	ThrowableItemIndex int
	ThrowableItem      *gear.Item
}

// Updates the throwable item with an item in the players inventory.
// Todo, need to add additional checks here to make sure that the inventory exists
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

// All of the player information needs to be easily accessible. This may work as a singleton. Todo
// Inventory and attributes are components, so there are helper functions to get them. It's for convenience
type PlayerData struct {
	Equipment       PlayerEquipment
	PlayerThrowable //Todo make this non embedded
	InputStates     PlayerInputStates

	PlayerEntity *ecs.Entity

	Pos       *common.Position
	Inventory *gear.Inventory
}

func (pl *PlayerData) UnequipMeleeWeapon() {

	pl.PlayerEntity.RemoveComponent(gear.MeleeWeaponComponent)
	pl.Inventory.AddItem(pl.Equipment.EqMeleeWeapon)
	pl.Equipment.EqMeleeWeapon = nil

}

func (pl *PlayerData) UnequipRangedWeapon() {

	pl.PlayerEntity.RemoveComponent(gear.RangedWeaponComponent)
	pl.Inventory.AddItem(pl.Equipment.EqRangedWeapon)
	pl.Equipment.EqRangedWeapon = nil

}

func (pl *PlayerData) UnequipArmor() {

	pl.PlayerEntity.RemoveComponent(gear.ArmorComponent)
	pl.Inventory.AddItem(pl.Equipment.EqArmor)
	pl.Equipment.EqArmor = nil

}

// Remvoes the item and adds it back to the inventory.
// The UnequipNNN functions add it back to the inventory
func (pl *PlayerData) RemoveItem(e *ecs.Entity) {

	switch gear.KindOfItem(e) {
	case gear.ArmorType:
		pl.UnequipArmor()
	case gear.MeleeWeaponType:
		pl.UnequipMeleeWeapon()
	case gear.RangedWeaponType:
		pl.UnequipRangedWeapon()
	default:
		fmt.Println("Invalid item equipped")
	}

}

// The inventory is a component of the player entity
func (pl *PlayerData) GetPlayerInventory() *gear.Inventory {

	playerInventory := common.GetComponentType[*gear.Inventory](pl.PlayerEntity, gear.InventoryComponent)

	return playerInventory
}

// Handles the conversions from the component type to the struct type
func (pl *PlayerData) GetPlayerAttributes() *common.Attributes {

	attr := &common.Attributes{}
	if data, ok := pl.PlayerEntity.GetComponentData(common.AttributeComponent); ok {
		attr = data.(*common.Attributes)

	}

	return attr
}

// Called as part of ManageTurn, which is called in the ebiten Update function.
// Effects can change the attributes, so we have a function dedicated to updating it.
func (pl *PlayerData) UpdatePlayerAttributes() {

	attr := pl.GetPlayerAttributes()

	ac := 0
	prot := 0
	dodge := float32(0.0)

	if pl.Equipment.EqArmor != nil {

		armor := common.GetComponentType[*gear.Armor](pl.Equipment.EqArmor, gear.ArmorComponent)
		ac = armor.ArmorClass
		prot = armor.Protection
		dodge = float32(armor.DodgeChance)
	}

	attr.TotalArmorClass = attr.BaseArmorClass + ac
	attr.TotalProtection = attr.BaseProtection + prot
	attr.TotalDodgeChance = attr.BaseDodgeChance + dodge

}
