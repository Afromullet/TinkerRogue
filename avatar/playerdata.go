package avatar

import (
	"game_main/common"
	"game_main/coords"
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
	HasKeyInput bool //Tells us whether the player pressed any key.
	InfoMeuOpen bool
}

// State and variable tracker for throwing items. Inventory, drawing, and input uses this.
type PlayerThrowable struct {
	SelectedThrowable  *ecs.Entity
	ThrowingAOEShape   graphics.TileBasedShape
	ThrowableItemIndex int
	ThrowableItem      *gear.Item
}

// Updates the throwable item with an item in the players inventory.
func (pl *PlayerThrowable) PrepareThrowable(itemEntity *ecs.Entity, index int) {

	pl.SelectedThrowable = itemEntity

	item := gear.GetItem(pl.SelectedThrowable)
	pl.ThrowableItem = item

	if t := item.GetThrowableAction(); t != nil {
		pl.ThrowableItemIndex = index
		pl.ThrowingAOEShape = t.Shape
	}

}

// Removes the item that the player threw.
// Called in avataractions.go
func (pl *PlayerThrowable) RemoveThrownItem(inv *gear.Inventory) {

	inv.RemoveItem(pl.ThrowableItemIndex)

}

// All of the player information needs to be easily accessible. This may work as a singleton. Todo
type PlayerData struct {
	Throwables   PlayerThrowable
	InputStates  PlayerInputStates
	PlayerEntity *ecs.Entity
	Pos          *coords.LogicalPosition
	Inventory    *gear.Inventory
}

// The inventory is a component of the player entity
func (pl *PlayerData) PlayerInventory() *gear.Inventory {

	playerInventory := common.GetComponentType[*gear.Inventory](pl.PlayerEntity, gear.InventoryComponent)

	return playerInventory
}

func (pl *PlayerData) PlayerAttributes() *common.Attributes {

	attr := &common.Attributes{}
	if data, ok := pl.PlayerEntity.GetComponentData(common.AttributeComponent); ok {
		attr = data.(*common.Attributes)

	}

	return attr
}
