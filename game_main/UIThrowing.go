package main

import (
	"fmt"
	"image/color"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

type ThrowingItemDisplay struct {
	itemDisplay                ItemDisplay
	ItemsSelectedContainer     *widget.Container //Displays the items the user HAS selected for crafitng
	ItemsSelectedPropContainer *widget.Container //Container to hold the widget that displays the proeprties of the selected item
	ItemsSelectedPropTextArea  *widget.TextArea  //Displays the properties of the selected items
	throwableItemSelected      bool
}

// Todo modify this to make it compatible with THrowable Display actions on list item click
func (throwingItemDisplay *ThrowingItemDisplay) CreateInventoryList(playerData *PlayerData, propFilters ...ItemProperty) {

	inv := playerData.GetPlayerInventory().GetInventoryForDisplay([]int{}, propFilters...)
	throwingItemDisplay.itemDisplay.InventoryDisplaylist = throwingItemDisplay.itemDisplay.GetInventoryListWidget(inv)

	throwingItemDisplay.itemDisplay.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		fmt.Print("Throwable Item Selected")

		a := args.(*widget.ListEntrySelectedEventArgs)
		entry := a.Entry.(InventoryListEntry)

		it, err := playerData.inventory.GetItemByIndex(entry.index)

		//throwableComponentData := GetComponentStruct[*Item](it, ItemComponent)
		//	fmt.Println("Printing throwable ", throwableComponentData)

		if err == nil {
			playerData.PrepareThrowable(it)
		}

		throwingItemDisplay.throwableItemSelected = true

	})

	throwingItemDisplay.itemDisplay.ItemDisplayContainer.AddChild(throwingItemDisplay.itemDisplay.InventoryDisplaylist)

}

func (throwingItemDisplay *ThrowingItemDisplay) DisplayInventory(g *Game) {

	//Passing a zero value throwable for the propFIlter
	throwingItemDisplay.CreateInventoryList(&g.playerData, NewThrowable(0, 0, 0, NewSquareAtPixel(0, 0, 0)))

}

func (throwingItemDisplay *ThrowingItemDisplay) CreateContainers() {
	// Main container that will hold the container for available items and the items selected
	throwingItemDisplay.itemDisplay.rootContainer = widget.NewContainer(
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

	// Used for holding the items prior to selecting them for crafting
	throwingItemDisplay.itemDisplay.ItemDisplayContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	throwingItemDisplay.itemDisplay.rootContainer.AddChild(throwingItemDisplay.itemDisplay.ItemDisplayContainer)
}
