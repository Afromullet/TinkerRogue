package gui

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/gear"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
)

type PlayerItemsUI struct {
	ThrowableItemDisplay ThrowingItemDisplay
	EquipmentDisplay     EquipmentItemDisplay
	ConsumableDisplay    ConsumableItemDisplay
}

// Create the container that contains the widgets for displaying the different views of the inventory
func CreateInventorySelectionContainer(playerUI *PlayerUI, inv *gear.Inventory, pl *avatar.PlayerData, ui *ebitenui.UI) *widget.Container {

	//This creates the root container for this UI.
	itemDisplayOptionsContainer := widget.NewContainer(

		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.TrackHover(false)),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			// It is using a GridLayout with a single column
			widget.GridLayoutOpts.Columns(4),
			widget.GridLayoutOpts.Stretch([]bool{false}, []bool{false, false, false}),
			widget.GridLayoutOpts.Padding(widget.Insets{
				Top:    20,
				Bottom: 20,
			}),
			// Spacing defines how much space to put between each column and row
			widget.GridLayoutOpts.Spacing(0, 20))),
	)

	itemDisplayOptionsContainer.AddChild(CreateOpenThrowablesButton(playerUI, inv, ui))
	itemDisplayOptionsContainer.AddChild(CreateOpenEquipmentButton(playerUI, inv, ui))
	itemDisplayOptionsContainer.AddChild(CreateOpenConsumablesButton(playerUI, inv, ui))

	CreateItemManagementUI(playerUI, pl, pl.Inventory, pl.PlayerAttributes(), pl.PlayerEntity)

	return itemDisplayOptionsContainer

}

func (itemDisp *ItemDisplay) SetupItemDisplay(windowName string, pl *avatar.PlayerData) {
	itemDisp.createItemDisplayContainer()
	itemDisp.createInventoryDisplayWindow(windowName)
	itemDisp.createItemSelectedContainer()
	itemDisp.playerData = pl

}

// This could probably be simplified
func CreateItemManagementUI(playerUI *PlayerUI, playerData *avatar.PlayerData, playerInventory *gear.Inventory, playerAttributes *common.Attributes, playerEnt *ecs.Entity) {

	playerUI.ItemsUI.ConsumableDisplay.CreateRootContainer()
	playerUI.ItemsUI.ConsumableDisplay.ItmDisplay.SetupItemDisplay("Consumables", playerData)
	playerUI.ItemsUI.ConsumableDisplay.SetupContainers()

	playerUI.ItemsUI.EquipmentDisplay.CreateRootContainer()
	playerUI.ItemsUI.EquipmentDisplay.ItmDisplay.SetupItemDisplay("Equipment Window", playerData)
	playerUI.ItemsUI.EquipmentDisplay.SetupContainers()

	playerUI.ItemsUI.ThrowableItemDisplay.CreateRootContainer()
	playerUI.ItemsUI.ThrowableItemDisplay.ItemDisplay.SetupItemDisplay("Throwable Window", playerData)
	playerUI.ItemsUI.ThrowableItemDisplay.SetupContainers()


}
