package gui

import (
	"fmt"
	"game_main/gear"
	"game_main/graphics"

	"github.com/ebitenui/ebitenui/widget"
)

// Todo add additional widgets to display the properties of the selected item
type ThrowingItemDisplay struct {
	ItemDisplay ItemDisplay

	ThrowableItemText     *widget.TextArea //Displays the properties of the selected items
	ThrowableItemSelected bool
}

// Todo modify this to make it compatible with THrowable Display actions on list item click
func (throwingItemDisplay *ThrowingItemDisplay) CreateInventoryList(propFilters ...gear.StatusEffects) {

	inv := throwingItemDisplay.ItemDisplay.GetInventory().GetInventoryForDisplay([]int{}, propFilters...)
	throwingItemDisplay.ItemDisplay.InventoryDisplaylist = throwingItemDisplay.ItemDisplay.GetInventoryListWidget(inv)

	throwingItemDisplay.ItemDisplay.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		fmt.Print("Throwable Item Selected")

		a := args.(*widget.ListEntrySelectedEventArgs)
		entry := a.Entry.(gear.InventoryListEntry)

		it, err := throwingItemDisplay.ItemDisplay.GetInventory().GetItem(entry.Index)

		//throwableComponentData := GetComponentStruct[*Item](it, ItemComponent)
		//	fmt.Println("Printing throwable ", throwableComponentData)

		if err == nil {

			throwingItemDisplay.ItemDisplay.playerData.PrepareThrowable(it, entry.Index)
		}

		throwingItemDisplay.ThrowableItemSelected = true

	})

	throwingItemDisplay.ItemDisplay.InventoryDisplayContainer.AddChild(throwingItemDisplay.ItemDisplay.InventoryDisplaylist)

}

func (throwingItemDisplay *ThrowingItemDisplay) DisplayInventory() {

	//Passing a zero value throwable for the propFIlter

	s := graphics.NewTileSquare(0, 0, 0)

	//throwingItemDisplay.CreateInventoryList(&g.playerData, NewThrowable(0, 0, 0, NewTileSquare(0, 0, 0)))
	throwingItemDisplay.CreateInventoryList(gear.NewThrowable(0, 0, 0, s))

}

func (throwingItemDisplay *ThrowingItemDisplay) CreateRootContainer() {

	throwingItemDisplay.ItemDisplay.RootContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(defaultWidgetColor),
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

}

func (throwingItemDisplay *ThrowingItemDisplay) SetupContainers() {

	// Main container that will hold the container for available items and the items selected

	throwingItemDisplay.ItemDisplay.InventoryDisplayContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(defaultWidgetColor),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	throwingItemDisplay.ItemDisplay.RootContainer.AddChild(throwingItemDisplay.ItemDisplay.InventoryDisplayContainer)
}
