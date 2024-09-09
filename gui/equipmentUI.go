package gui

import (
	"game_main/equipment"
	"image/color"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

type EquipmentItemDisplay struct {
	ItmDisplay                ItemDisplay
	ItemSelectedContainer     *widget.Container //Displays the items the user HAS selected for crafitng
	ItemSelectedStats         *widget.Container //Container the stats of the selected item
	ItemsSelectedPropTextArea *widget.TextArea  //Displays the properties of the selected items
	ThrowableItemSelected     bool
}

// Selects an item and adds it to the ItemsSelectedContainer container and ItemsSelectedPropContainer
// ItemSeleced container tells us which items we're crafting with
// ItemsSelectedPropContainer tells which properties the items have
func (equipmentDisplay *EquipmentItemDisplay) CreateInventoryList(inventory *equipment.Inventory, propFilters ...equipment.StatusEffects) {

	inv := inventory.GetEquipmentForDisplay([]int{})
	equipmentDisplay.ItmDisplay.InventoryDisplaylist = equipmentDisplay.ItmDisplay.GetInventoryListWidget(inv)

	equipmentDisplay.ItmDisplay.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		equipmentDisplay.ItemSelectedContainer.RemoveChild(equipmentDisplay.ItmDisplay.ItemsSelectedList)

		a := args.(*widget.ListEntrySelectedEventArgs)
		entry := a.Entry.(equipment.InventoryListEntry)

		equipmentDisplay.ItmDisplay.ItemsSelectedList = equipmentDisplay.ItmDisplay.GetSelectedItems(entry.Index, inventory)

		if equipmentDisplay.ItmDisplay.ItemsSelectedList != nil {
			equipmentDisplay.ItemSelectedContainer.AddChild(equipmentDisplay.ItmDisplay.ItemsSelectedList)

		}

	})

	equipmentDisplay.ItmDisplay.ItemDisplayContainer.AddChild(equipmentDisplay.ItmDisplay.InventoryDisplaylist)

}

func (equipmentDisplay *EquipmentItemDisplay) DisplayInventory(inventory *equipment.Inventory) {

	equipmentDisplay.CreateInventoryList(inventory)

}

func (equipmentDisplay *EquipmentItemDisplay) CreateContainers() {
	// Main container that will hold the container for available items and the items selected
	equipmentDisplay.ItmDisplay.RootContainer = widget.NewContainer(
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

	// Holds the widget that displays the selected items to the player
	equipmentDisplay.ItemSelectedContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
		)))

	equipmentDisplay.ItmDisplay.ItemDisplayContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.ItmDisplay.ItemDisplayContainer)
	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.ItemSelectedContainer)
}
