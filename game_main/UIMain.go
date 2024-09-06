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

// Every window that displays the inventory to teh user will be a struct that contains ItemDisplay
// And implements the ItemDisplayer interface
type ItemDisplayer interface {
	CreateContainers()                                                        //For creating the containers
	CreateInventoryList(playerData *PlayerData, propFilters ...StatusEffects) //For getting the inventory from the player and adding on click event handlers
	DisplayInventory(g *Game)                                                 //Really just there for calling CreateInventoryList with ItemProperty filters for the specific kind of window
}

// Anything that displays the inventory will have to use this struct through composition.
// Originally I ran into problems with having multiple windows due to the ItemDisplayCOntain
type ItemDisplay struct {
	rootContainer        *widget.Container //Holds all of the GUI elements
	rootWindow           *widget.Window    //Window to hold the root container content
	ItemDisplayContainer *widget.Container //Container that holds the items to be displayed
	InventoryDisplaylist *widget.List      //Holds all of the items
	ItemsSelectedList    *widget.List      //Holds only the items the player selects
	ItemsSelectedIndices []int             //The indices in inventoryDisplayList of the items the user selected

}

// Gets a list which displays the inventory to the user.
// Simply gets the List. It does not tie it to the players inventory.
// That behavior is added by implementing CreateInventoryList from the ItemDisplayer interface
func (ItemDisplay *ItemDisplay) GetInventoryListWidget(entries []any) *widget.List {
	li := widget.NewList(

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

func (itemDisplay *ItemDisplay) CreateInventoryDisplayWindow(title string) {

	itemDisplay.ItemsSelectedIndices = make([]int, 0)

	titleFace, _ := loadFont(12)

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

	itemDisplay.rootWindow = widget.NewWindow(

		widget.WindowOpts.Contents(itemDisplay.rootContainer),

		widget.WindowOpts.TitleBar(titleContainer, 25),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.CLICK_OUT),
		widget.WindowOpts.Draggable(),
		widget.WindowOpts.Resizeable(),
		widget.WindowOpts.MinSize(200, 100),

		widget.WindowOpts.MoveHandler(func(args *widget.WindowChangedEventArgs) {
			fmt.Println("Window Moving")
		}),
		//Set the callback that triggers when a resize is complete
		widget.WindowOpts.ResizeHandler(func(args *widget.WindowChangedEventArgs) {
			fmt.Println("Window Resized")
		}),
	)

}

type PlayerItemsUI struct {
	rootContainer *widget.Container //The main for the inventory window

	craftingItemDisplay  CraftingItemDisplay
	throwableItemDisplay ThrowingItemDisplay
}

// Creates the main UI that allows the player to view the inventory, craft, and see equipment
func (g *Game) CreatePlayerUI() *ebitenui.UI {

	ui := ebitenui.UI{}

	// Main container that will hold the container for available items and the items selected
	g.itemsUI.rootContainer = widget.NewContainer(
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

	CreateItemManagementUI(g)

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

			x, y := g.itemsUI.craftingItemDisplay.itemDisplay.rootWindow.Contents.PreferredSize()

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 50})
			g.itemsUI.craftingItemDisplay.itemDisplay.rootWindow.SetLocation(r)
			g.itemsUI.craftingItemDisplay.DisplayInventory(g)
			ui.AddWindow(g.itemsUI.craftingItemDisplay.itemDisplay.rootWindow)

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

			x, y := g.itemsUI.throwableItemDisplay.itemDisplay.rootWindow.Contents.PreferredSize()

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 50})
			g.itemsUI.throwableItemDisplay.itemDisplay.rootWindow.SetLocation(r)
			g.itemsUI.throwableItemDisplay.DisplayInventory(g)
			ui.AddWindow(g.itemsUI.throwableItemDisplay.itemDisplay.rootWindow)

		}),
	)

	return button

}

//For creating a window that the Item Display related containers are shown in

func CreateItemManagementUI(g *Game) {

	g.itemsUI.craftingItemDisplay.CreateContainers()
	g.itemsUI.craftingItemDisplay.itemDisplay.CreateInventoryDisplayWindow("Crafting Window")

	g.itemsUI.throwableItemDisplay.CreateContainers()
	g.itemsUI.throwableItemDisplay.itemDisplay.CreateInventoryDisplayWindow("Throwing Window")

}
