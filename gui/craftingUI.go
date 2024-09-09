package gui

import (
	"game_main/avatar"
	"game_main/equipment"
	"image/color"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

// The CraftingItemDisplay tracks the items selected for crafting and their properties.
// Each uses a
type CraftingItemDisplay struct {
	ItmDisplay                 ItemDisplay
	ItemsSelectedContainer     *widget.Container //Displays the items the user HAS selected for crafitng
	ItemsSelectedPropContainer *widget.Container //Container to hold the widget that displays the proeprties of the selected item
	ItemsSelectedPropTextArea  *widget.TextArea  //Displays the properties of the selected items
	CraftItemsButton           *widget.Button    //Craft with the selected items
	ClearItemsButton           *widget.Button    //Clear the selected items

}

// Selects an item and adds it to the ItemsSelectedContainer container and ItemsSelectedPropContainer
// ItemSeleced container tells us which items we're crafting with
// ItemsSelectedPropContainer tells which properties the items have
func (craftingItemDisplay *CraftingItemDisplay) CreateInventoryList(playerData *avatar.PlayerData, propFilters ...equipment.StatusEffects) {

	inv := playerData.GetPlayerInventory().GetInventoryForDisplay([]int{}, propFilters...)
	craftingItemDisplay.ItmDisplay.InventoryDisplaylist = craftingItemDisplay.ItmDisplay.GetInventoryListWidget(inv)

	craftingItemDisplay.ItmDisplay.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		craftingItemDisplay.ItemsSelectedContainer.RemoveChild(craftingItemDisplay.ItmDisplay.ItemsSelectedList)

		a := args.(*widget.ListEntrySelectedEventArgs)
		entry := a.Entry.(equipment.InventoryListEntry)

		craftingItemDisplay.ItmDisplay.ItemsSelectedList = craftingItemDisplay.ItmDisplay.GetSelectedItems(entry.Index, playerData)

		if craftingItemDisplay.ItmDisplay.ItemsSelectedList != nil {
			craftingItemDisplay.ItemsSelectedContainer.AddChild(craftingItemDisplay.ItmDisplay.ItemsSelectedList)

			names, _ := playerData.GetPlayerInventory().EffectNames(entry.Index)

			for _, n := range names {
				craftingItemDisplay.ItemsSelectedPropTextArea.AppendText(n)

			}

		}

	})

	craftingItemDisplay.ItmDisplay.ItemDisplayContainer.AddChild(craftingItemDisplay.ItmDisplay.InventoryDisplaylist)

}

// Used by the Clicked Handler of the Crafting Button. Displays the inventory
func (craftingItemDisplay *CraftingItemDisplay) DisplayInventory(pl *avatar.PlayerData) {

	craftingItemDisplay.CreateInventoryList(pl)

}

func (craftingItemDisplay *CraftingItemDisplay) CreateContainers() {

	craftingItemDisplay.CreateCraftingMenuButtons()
	craftingItemDisplay.CreateItemPropertyTextArea()

	// Main container that will hold the container for available items and the items selected
	craftingItemDisplay.ItmDisplay.RootContainer = widget.NewContainer(
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
	craftingItemDisplay.ItmDisplay.ItemDisplayContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	craftingItemDisplay.ItmDisplay.RootContainer.AddChild(craftingItemDisplay.ItmDisplay.ItemDisplayContainer)

	// Holds the widget that displays the selected items to the player
	craftingItemDisplay.ItemsSelectedContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
		)))

	//Holds the window that will display item properties
	craftingItemDisplay.ItemsSelectedPropContainer = widget.NewContainer(

		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	craftingItemDisplay.ItmDisplay.RootContainer.AddChild(craftingItemDisplay.ItemsSelectedContainer)

	craftingItemDisplay.ItemsSelectedPropContainer.AddChild(craftingItemDisplay.ItemsSelectedPropTextArea)
	craftingItemDisplay.ItemsSelectedContainer.AddChild(craftingItemDisplay.ClearItemsButton)
	craftingItemDisplay.ItemsSelectedPropContainer.AddChild(craftingItemDisplay.ItemsSelectedPropTextArea)
	craftingItemDisplay.ItemsSelectedPropContainer.AddChild(craftingItemDisplay.CraftItemsButton)
	craftingItemDisplay.ItmDisplay.RootContainer.AddChild(craftingItemDisplay.ItemsSelectedPropContainer)

}

// Creating the buttons that reside in the crafting menu.
func (craftingItemDisplay *CraftingItemDisplay) CreateCraftingMenuButtons() {
	// construct a button
	button := widget.NewButton(
		// set general widget options
		widget.ButtonOpts.WidgetOpts(
			// instruct the container's anchor layout to center the button both horizontally and vertically
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),

		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Clear Items", face, &widget.ButtonTextColor{
			Idle: color.NRGBA{0xdf, 0xf4, 0xff, 0xff},
		}),

		widget.ButtonOpts.TextPadding(widget.Insets{
			Left:   30,
			Right:  30,
			Top:    5,
			Bottom: 5,
		}),

		// add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {

			//playerCraftingUI.ItemsSelectedContainer.RemoveChild(playerCraftingUI.ItemsSelectedList)
			//playerCraftingUI.ItemsSelectedIndices = playerCraftingUI.ItemsSelectedIndices[:0]

		}),
	)

	craftingItemDisplay.ClearItemsButton = button

	// construct a button
	button = widget.NewButton(
		// set general widget options
		widget.ButtonOpts.WidgetOpts(
			// instruct the container's anchor layout to center the button both horizontally and vertically
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),

		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Craft", face, &widget.ButtonTextColor{
			Idle: color.NRGBA{0xdf, 0xf4, 0xff, 0xff},
		}),

		widget.ButtonOpts.TextPadding(widget.Insets{
			Left:   30,
			Right:  30,
			Top:    5,
			Bottom: 5,
		}),

		// add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {

		}),
	)

	craftingItemDisplay.CraftItemsButton = button

}

// Text window to display the item properties of the selected items to the player
func (craftingItemDisplay *CraftingItemDisplay) CreateItemPropertyTextArea() {
	// construct a textarea
	craftingItemDisplay.ItemsSelectedPropTextArea = widget.NewTextArea(
		widget.TextAreaOpts.ContainerOpts(
			widget.ContainerOpts.WidgetOpts(
				//Set the layout data for the textarea
				//including a max height to ensure the scroll bar is visible
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					//Position: widget.RowLayoutPositionCenter,
					Stretch: true,
				}),
				//Set the minimum size for the widget
				//widget.WidgetOpts.MinSize(300, 100),
			),
		),
		//Set gap between scrollbar and text
		widget.TextAreaOpts.ControlWidgetSpacing(2),
		//Tell the textarea to display bbcodes
		widget.TextAreaOpts.ProcessBBCode(true),
		//Set the font color
		widget.TextAreaOpts.FontColor(color.Black),
		//Set the font face (size) to use
		widget.TextAreaOpts.FontFace(face),

		//Tell the TextArea to show the vertical scrollbar
		widget.TextAreaOpts.ShowVerticalScrollbar(),
		//Set padding between edge of the widget and where the text is drawn
		widget.TextAreaOpts.TextPadding(widget.NewInsetsSimple(10)),
		//This sets the background images for the scroll container
		widget.TextAreaOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
				Idle: e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				Mask: e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
			}),
		),
		//This sets the images to use for the sliders
		widget.TextAreaOpts.SliderOpts(
			widget.SliderOpts.Images(
				// Set the track images
				&widget.SliderTrackImage{
					Idle:  e_image.NewNineSliceColor(color.NRGBA{200, 200, 200, 255}),
					Hover: e_image.NewNineSliceColor(color.NRGBA{200, 200, 200, 255}),
				},
				// Set the handle images
				&widget.ButtonImage{
					Idle:    e_image.NewNineSliceColor(color.NRGBA{255, 100, 100, 255}),
					Hover:   e_image.NewNineSliceColor(color.NRGBA{255, 100, 100, 255}),
					Pressed: e_image.NewNineSliceColor(color.NRGBA{255, 100, 100, 255}),
				},
			),
		),
	)

}
