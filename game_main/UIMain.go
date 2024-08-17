package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"strconv"

	"github.com/ebitenui/ebitenui"
	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

var face, _ = loadFont(20)
var buttonImage, _ = loadButtonImage()

/*
The rootContainer contains buttons that open different window.

The different windows show a filtered version of the inventory to the user

All windows use hte same Item Selected Containers
*/

type ItemUIWindowsState struct {
	craftingWindowOpen bool
	throwingWIndowOpen bool
}

type PlayerItemsUI struct {
	rootContainer              *widget.Container //The main for the inventory window
	rootCraftingWindow         *widget.Window
	rootThrowableWindow        *widget.Window
	ItemDisplayContainer       *widget.Container //Displays the items the user CAN select for crafting
	ItemsSelectedContainer     *widget.Container //Displays the items the user HAS selected for crafitng
	ItemsSelectedPropContainer *widget.Container //Container to hold the widget that displays the proeprties of the selected item
	ItemsSelectedPropTextArea  *widget.TextArea  //Displays the properties of the selected items

	ClearSelectedItemsButton *widget.Button
	CraftItemsButton         *widget.Button

	InventoryDisplaylist *widget.List
	ItemsSelectedList    *widget.List
	ItemsSelectedIndices []int //The indices in inventoryDisplayList of the items the user selected

	craftingWindowRemoveFunc  widget.RemoveWindowFunc
	throwableWindowRemoveFunc widget.RemoveWindowFunc

	windowState ItemUIWindowsState
}

// Called whenever the inventory is displayed to the user.
// This updates the GUI elements in PlayerCraftingUI
func (ui *PlayerItemsUI) UpdateCraftingInventory(g *Game, propFilters ...ItemProperty) {

	ui.UpdateInventoryDisplaylist(&g.playerData, propFilters...)

	g.craftingUI.ItemDisplayContainer.AddChild(ui.InventoryDisplaylist)

}

// Gets a list widget for displaying the inventory
func (PlayerCraftingUI *PlayerItemsUI) GetInventoryListWidget(entries []any) *widget.List {
	li := widget.NewList(

		// Set how wide the list should be

		widget.ListOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(150, 0),
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				StretchVertical:    true,
				Padding:            widget.NewInsetsSimple(50),
			}),
		)),

		// Set the entries in the list
		widget.ListOpts.Entries(entries),
		widget.ListOpts.ScrollContainerOpts(
			// Set the background images/color for the list
			widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
				Idle:     e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				Disabled: e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				Mask:     e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
			}),
		),
		widget.ListOpts.SliderOpts(
			// Set the background images/color for the background of the slider track
			widget.SliderOpts.Images(&widget.SliderTrackImage{
				Idle:  e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				Hover: e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
			}, buttonImage),
			widget.SliderOpts.MinHandleSize(5),
			// Set how wide the track should be
			widget.SliderOpts.TrackPadding(widget.NewInsetsSimple(2))),
		// Hide the horizontal slider
		widget.ListOpts.HideHorizontalSlider(),
		// Set the font for the list options
		widget.ListOpts.EntryFontFace(face),
		// Set the colors for the list
		widget.ListOpts.EntryColor(&widget.ListEntryColor{
			Selected:                   color.NRGBA{R: 0, G: 255, B: 0, A: 255},     // Foreground color for the unfocused selected entry
			Unselected:                 color.NRGBA{R: 254, G: 255, B: 255, A: 255}, // Foreground color for the unfocused unselected entry
			SelectedBackground:         color.NRGBA{R: 130, G: 130, B: 200, A: 255}, // Background color for the unfocused selected entry
			SelectingBackground:        color.NRGBA{R: 130, G: 130, B: 130, A: 255}, // Background color for the unfocused being selected entry
			SelectingFocusedBackground: color.NRGBA{R: 130, G: 140, B: 170, A: 255}, // Background color for the focused being selected entry
			SelectedFocusedBackground:  color.NRGBA{R: 130, G: 130, B: 170, A: 255}, // Background color for the focused selected entry
			FocusedBackground:          color.NRGBA{R: 170, G: 170, B: 180, A: 255}, // Background color for the focused unselected entry
			DisabledUnselected:         color.NRGBA{R: 100, G: 100, B: 100, A: 255}, // Foreground color for the disabled unselected entry
			DisabledSelected:           color.NRGBA{R: 100, G: 100, B: 100, A: 255}, // Foreground color for the disabled selected entry
			DisabledSelectedBackground: color.NRGBA{R: 100, G: 100, B: 100, A: 255}, // Background color for the disabled selected entry
		}),
		// This required function returns the string displayed in the list
		widget.ListOpts.EntryLabelFunc(func(e interface{}) string {
			return e.(InventoryListEntry).Name + " x" + strconv.Itoa(e.(InventoryListEntry).count)
		}),
		// Padding for each entry
		widget.ListOpts.EntryTextPadding(widget.NewInsetsSimple(5)),
		// Text position for each entry
		widget.ListOpts.EntryTextPosition(widget.TextPositionStart, widget.TextPositionCenter),
		// This handler defines what function to run when a list item is selected.

	)

	return li
}

// This updates the PlayerCraftingUI inventoryDisplayList with the slice passed as a parameter.
// A player can select an item by clicking it. It will then be added to the "Selected Item" container
func (playerCraftingUI *PlayerItemsUI) UpdateInventoryDisplaylist(playerData *PlayerData, propFilters ...ItemProperty) {

	// Nested function to add a selected item
	addSelectedItem := func(index int) {

		for _, itemIndex := range playerCraftingUI.ItemsSelectedIndices {
			if itemIndex == index {
				return
			}
		}
		playerCraftingUI.ItemsSelectedIndices = append(playerCraftingUI.ItemsSelectedIndices, index)
	}

	inv := playerData.GetPlayerInventory().GetInventoryForDisplay([]int{}, propFilters...)
	playerCraftingUI.InventoryDisplaylist = playerCraftingUI.GetInventoryListWidget(inv)

	playerCraftingUI.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		//So that we don't append to the container

		playerCraftingUI.ItemsSelectedContainer.RemoveChild(playerCraftingUI.ItemsSelectedList)

		a := args.(*widget.ListEntrySelectedEventArgs)
		entry := a.Entry.(InventoryListEntry)

		addSelectedItem(entry.index)

		names, _ := playerData.GetPlayerInventory().GetPropertyNames(entry.index)

		playerData.GetPlayerInventory().GetPropertyNames(100)

		sel := playerData.GetPlayerInventory().GetInventoryForDisplay(playerCraftingUI.ItemsSelectedIndices)

		playerCraftingUI.ItemsSelectedList = playerCraftingUI.GetInventoryListWidget(sel)

		if playerCraftingUI.ItemsSelectedList != nil {
			playerCraftingUI.ItemsSelectedContainer.AddChild(playerCraftingUI.ItemsSelectedList)

			names, _ := playerData.GetPlayerInventory().GetPropertyNames(entry.index)

			for _, n := range names {
				playerCraftingUI.ItemsSelectedPropTextArea.AppendText(n)

			}

		}

		fmt.Println("Entry Selected: ", entry)
		fmt.Println("Printing Item Properties: ", names)
		fmt.Println("Items Selected ", playerCraftingUI.ItemsSelectedIndices)
		//log.Print(playerData.GetPlayerInventory().GetInventoryForDispl

	})

}

// Creating the buttons that reside in the crafting menu.
func (playerCraftingUI *PlayerItemsUI) CreateCraftingMenuButtons(g *Game) {
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

			playerCraftingUI.ItemsSelectedContainer.RemoveChild(playerCraftingUI.ItemsSelectedList)
			playerCraftingUI.ItemsSelectedIndices = playerCraftingUI.ItemsSelectedIndices[:0]

		}),
	)

	playerCraftingUI.ClearSelectedItemsButton = button

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

	playerCraftingUI.CraftItemsButton = button

}

// Creates the main UI that allows the player to view the inventory, craft, and see equipment
func (g *Game) CreatePlayerUI() *ebitenui.UI {

	ui := ebitenui.UI{}

	// Main container that will hold the container for available items and the items selected
	g.craftingUI.rootContainer = widget.NewContainer(
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

	CreateItemManagementUI(g)

	//This creates the root container for this UI.
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.TrackHover(false)),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			// It is using a GridLayout with a single column
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{false}, []bool{false, false, false}),
			widget.GridLayoutOpts.Padding(widget.Insets{
				Top:    20,
				Bottom: 20,
			}),
			// Spacing defines how much space to put between each column and row
			widget.GridLayoutOpts.Spacing(0, 20))),
	)

	rootContainer.AddChild(CreateOpenCraftingButton(g, &ui))
	rootContainer.AddChild(CreateOpenThrowablesButton(g, &ui))

	g.craftingUI.windowState = ItemUIWindowsState{
		craftingWindowOpen: false,
		throwingWIndowOpen: false,
	}
	ui.Container = rootContainer

	return &ui

}

// Creating the button that opens the crafting menu.
func CreateOpenCraftingButton(g *Game, ui *ebitenui.UI) *widget.Button {
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
		widget.ButtonOpts.Text("Crafting", face, &widget.ButtonTextColor{
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
			x, y := g.craftingUI.rootCraftingWindow.Contents.PreferredSize()

			g.craftingUI.windowState.craftingWindowOpen = true
			g.craftingUI.windowState.throwingWIndowOpen = false

			/*
				g.craftingUI.ClearSelectedItemsButton.GetWidget().Visibility = widget.Visibility_Show
				g.craftingUI.ItemsSelectedPropContainer.GetWidget().Visibility = widget.Visibility_Show
				g.craftingUI.ItemsSelectedPropTextArea.GetWidget().Visibility = widget.Visibility_Show
				g.craftingUI.ItemsSelectedContainer.GetWidget().Visibility = widget.Visibility_Show

			*/

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 50})
			g.craftingUI.rootCraftingWindow.SetLocation(r)
			g.craftingUI.UpdateCraftingInventory(g)
			g.craftingUI.craftingWindowRemoveFunc = ui.AddWindow(g.craftingUI.rootCraftingWindow)

		}),
	)

	return button

}

// Creating the button that opens the crafting menu. Other buttons will be added
// Doing it inside a function makes the code easier to follow
func CreateOpenThrowablesButton(g *Game, ui *ebitenui.UI) *widget.Button {
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
		widget.ButtonOpts.Text("Throwables", face, &widget.ButtonTextColor{
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

			g.craftingUI.windowState.craftingWindowOpen = false
			g.craftingUI.windowState.throwingWIndowOpen = true

			/*
				g.craftingUI.ClearSelectedItemsButton.GetWidget().Visibility = widget.Visibility_Hide
				g.craftingUI.ItemsSelectedPropContainer.GetWidget().Visibility = widget.Visibility_Hide
				g.craftingUI.ItemsSelectedPropTextArea.GetWidget().Visibility = widget.Visibility_Hide
				g.craftingUI.ItemsSelectedContainer.GetWidget().Visibility = widget.Visibility_Hide
			*/

			x, y := g.craftingUI.rootThrowableWindow.Contents.PreferredSize()
			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{500, 100})
			g.craftingUI.rootThrowableWindow.SetLocation(r)

			g.craftingUI.UpdateCraftingInventory(g, NewThrowable(0, 0, 0)) //New Throwable params doesn't matter. Just need the type to search
			g.craftingUI.throwableWindowRemoveFunc = ui.AddWindow(g.craftingUI.rootThrowableWindow)

		}),
	)

	return button

}

// Text window to display the item properties of the selected items to the player
func CreateItemPropertyTextArea() *widget.TextArea {
	// construct a textarea
	textarea := widget.NewTextArea(
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

	return textarea
}

//For creating a window that the Item Display related containers are shown in

func CreateInventoryDisplayWindow(g *Game, title string) *widget.Window {

	g.craftingUI.ItemsSelectedIndices = make([]int, 0)
	// load button text font
	//face, _ := loadFont(20)
	// load the font for the window title
	titleFace, _ := loadFont(12)

	// Create the titlebar for the window
	titleContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{150, 150, 150, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	titleContainer.AddChild(widget.NewText(
		widget.TextOpts.Text(title, titleFace, color.NRGBA{254, 255, 255, 255}),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	))

	window := widget.NewWindow(

		widget.WindowOpts.Contents(g.craftingUI.rootContainer),

		widget.WindowOpts.TitleBar(titleContainer, 25),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.CLICK_OUT),
		widget.WindowOpts.Draggable(),
		widget.WindowOpts.Resizeable(),
		widget.WindowOpts.MinSize(200, 100),

		//widget.WindowOpts.MaxSize(300, 300),

		widget.WindowOpts.MoveHandler(func(args *widget.WindowChangedEventArgs) {
			fmt.Println("Window Moving")
		}),
		//Set the callback that triggers when a resize is complete
		widget.WindowOpts.ResizeHandler(func(args *widget.WindowChangedEventArgs) {
			fmt.Println("Window Resized")
		}),
	)

	return window

}

func CreateItemManagementUI(g *Game) {

	g.craftingUI.rootCraftingWindow = CreateInventoryDisplayWindow(g, "Crafting Window")
	g.craftingUI.CreateCraftingMenuButtons(g)

	g.craftingUI.rootThrowableWindow = CreateInventoryDisplayWindow(g, "Throwing Window")

	CreateItemContainers(g)

}

// Creates the containers that display the items.
// This is shared amongst all GUI elements that display the inventory
func CreateItemContainers(g *Game) {

	// Used for holding the items prior to selecting them for crafting
	g.craftingUI.ItemDisplayContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	//Holds the widget that displays the selected items to the player
	g.craftingUI.ItemsSelectedContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
		)))

	//Holds the window that will display item properties
	g.craftingUI.ItemsSelectedPropContainer = widget.NewContainer(

		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	g.craftingUI.rootContainer.AddChild(g.craftingUI.ItemDisplayContainer)
	g.craftingUI.rootContainer.AddChild(g.craftingUI.ItemsSelectedContainer)

	g.craftingUI.ItemsSelectedPropTextArea = CreateItemPropertyTextArea()
	g.craftingUI.ItemsSelectedPropContainer.AddChild(g.craftingUI.ItemsSelectedPropTextArea)
	g.craftingUI.ItemsSelectedContainer.AddChild(g.craftingUI.ClearSelectedItemsButton)
	g.craftingUI.ItemsSelectedPropContainer.AddChild(g.craftingUI.ItemsSelectedPropTextArea)
	g.craftingUI.ItemsSelectedPropContainer.AddChild(g.craftingUI.CraftItemsButton)
	g.craftingUI.rootContainer.AddChild(g.craftingUI.ItemsSelectedPropContainer)

}
