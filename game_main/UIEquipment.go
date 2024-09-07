package main

import (
	"image/color"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

type EquipmentItemDisplay struct {
	itemDisplay                ItemDisplay
	ItemsSelectedContainer     *widget.Container //Displays the items the user HAS selected for crafitng
	ItemsSelectedPropContainer *widget.Container //Container to hold the widget that displays the proeprties of the selected item
	ItemsSelectedPropTextArea  *widget.TextArea  //Displays the properties of the selected items
	ThrowableItemSelected      bool
}

// Selects an item and adds it to the ItemsSelectedContainer container and ItemsSelectedPropContainer
// ItemSeleced container tells us which items we're crafting with
// ItemsSelectedPropContainer tells which properties the items have
func (equipmentDisplay *EquipmentItemDisplay) CreateInventoryList(playerData *PlayerData, propFilters ...StatusEffects) {

	inv := playerData.GetPlayerInventory().GetEquipmentForDisplay([]int{})
	equipmentDisplay.itemDisplay.InventoryDisplaylist = equipmentDisplay.itemDisplay.GetInventoryListWidget(inv)

	equipmentDisplay.itemDisplay.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		//Click Handler code goes here

	})

	equipmentDisplay.itemDisplay.ItemDisplayContainer.AddChild(equipmentDisplay.itemDisplay.InventoryDisplaylist)

}

func (equipmentDisplay *EquipmentItemDisplay) DisplayInventory(g *Game) {

	equipmentDisplay.CreateInventoryList(&g.playerData)

}

func (equipmentDisplay *EquipmentItemDisplay) CreateContainers() {
	// Main container that will hold the container for available items and the items selected
	equipmentDisplay.itemDisplay.rootContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			// It is using a GridLayout with a single column
			widget.GridLayoutOpts.Columns(3),

			widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true, true, true}),
			// Padding defines how much space to put around the outside of the grid.
			widget.GridLayoutOpts.Padding(widget.Insets{
				Top:    50,
				Bottom: 50,
			}),
			// Spacing defines how much space to put between each column and row
			widget.GridLayoutOpts.Spacing(0, 20))),
	)

	equipmentDisplay.itemDisplay.ItemDisplayContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	equipmentDisplay.itemDisplay.rootContainer.AddChild(equipmentDisplay.itemDisplay.ItemDisplayContainer)
}
