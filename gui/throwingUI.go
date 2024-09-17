package gui

import (
	"fmt"
	"game_main/avatar"
	"game_main/gear"
	"game_main/graphics"
	"image/color"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

type ThrowingItemDisplay struct {
	ItmDisp                    ItemDisplay
	ItemsSelectedContainer     *widget.Container //Displays the items the user HAS selected for crafitng
	ItemsSelectedPropContainer *widget.Container //Container to hold the widget that displays the proeprties of the selected item
	ItemsSelectedPropTextArea  *widget.TextArea  //Displays the properties of the selected items
	ThrowableItemSelected      bool
	playerData                 *avatar.PlayerData
}

// Todo modify this to make it compatible with THrowable Display actions on list item click
func (throwingItemDisplay *ThrowingItemDisplay) CreateInventoryList(propFilters ...gear.StatusEffects) {

	inv := throwingItemDisplay.ItmDisp.GetInventory().GetInventoryForDisplay([]int{}, propFilters...)
	throwingItemDisplay.ItmDisp.InventoryDisplaylist = throwingItemDisplay.ItmDisp.GetInventoryListWidget(inv)

	throwingItemDisplay.ItmDisp.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		fmt.Print("Throwable Item Selected")

		a := args.(*widget.ListEntrySelectedEventArgs)
		entry := a.Entry.(gear.InventoryListEntry)

		it, err := throwingItemDisplay.ItmDisp.GetInventory().GetItem(entry.Index)

		//throwableComponentData := GetComponentStruct[*Item](it, ItemComponent)
		//	fmt.Println("Printing throwable ", throwableComponentData)

		if err == nil {
			throwingItemDisplay.playerData.PrepareThrowable(it, entry.Index)
		}

		throwingItemDisplay.ThrowableItemSelected = true

	})

	throwingItemDisplay.ItmDisp.ItemDisplayContainer.AddChild(throwingItemDisplay.ItmDisp.InventoryDisplaylist)

}

func (throwingItemDisplay *ThrowingItemDisplay) DisplayInventory() {

	//Passing a zero value throwable for the propFIlter

	s := graphics.NewTileSquare(0, 0, 0)

	//throwingItemDisplay.CreateInventoryList(&g.playerData, NewThrowable(0, 0, 0, NewTileSquare(0, 0, 0)))
	throwingItemDisplay.CreateInventoryList(gear.NewThrowable(0, 0, 0, s))

}

func (throwingItemDisplay *ThrowingItemDisplay) CreateContainers() {
	// Main container that will hold the container for available items and the items selected
	throwingItemDisplay.ItmDisp.RootContainer = widget.NewContainer(
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

	throwingItemDisplay.ItmDisp.ItemDisplayContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	throwingItemDisplay.ItmDisp.RootContainer.AddChild(throwingItemDisplay.ItmDisp.ItemDisplayContainer)
}
