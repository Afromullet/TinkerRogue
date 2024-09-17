package gui

import (
	"fmt"
	"game_main/avatar"
	"game_main/gear"
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
It's painful to add a new container to display items in. Here are the steps:

1) Create a struct containing an ItemDisplay. This struct must also contain at least one container to hold the items to display
2) In that struct, create CreateInventoryList(playerData *avatar.PlayerData, propFilters ...StatusEffects) func. The propFilters are optional for filtering by a Status Effect
   CreateInventoryList also adds on click handler for what happens when the item is selected
3) Create the DisplayInventory(pl *avatar.PlayerData) function. This calls itemDisplay.CreateInventoryList(...) that was implemented in step 2
4) Create a  CreateContainers() function which creates all of the containers in the Item Display
5) Add the type to PlayerItemsUI
6) Create an CreateOpenxxxgButton function that creates the button and adds the window to the ItemDisplay for displaying the items
7) Create the containers and window in CreateItemManagementUI
8) Add the button to the root container in CreatePlayerUI
*/

// Every window that displays the inventory to teh user will be a struct that contains ItemDisplay
// And implements the ItemDisplayer interface
type ItemDisplayer interface {
	CreateContainers()                                     //For creating the containers
	CreateInventoryList(propFilters ...gear.StatusEffects) //For getting the inventory from the player and adding on click event handlers
	DisplayInventory()                                     //Really just there for calling CreateInventoryList with ItemProperty filters for the specific kind of window
}

// Anything that displays the inventory will have to use this struct through composition.
// Originally I ran into problems with having multiple windows due to the ItemDisplayCOntain
type ItemDisplay struct {
	inventory            *gear.Inventory   //Makes everything easier if every ItemDisplay has a pointer to the inventory
	RootContainer        *widget.Container //Holds all of the GUI elements
	RooWindow            *widget.Window    //Window to hold the root container content
	ItemDisplayContainer *widget.Container //Container that holds the items to be displayed
	InventoryDisplaylist *widget.List      //Holds all of the items
	ItemsSelectedList    *widget.List      //Holds only the items the player selects
	ItemsSelectedIndices []int             //The indices in inventoryDisplayList of the items the user selected

}

// Returns a widget.list containing the selected
func (itemDisplay *ItemDisplay) GetSelectedItems(index int, inv *gear.Inventory) *widget.List {

	for _, itemIndex := range itemDisplay.ItemsSelectedIndices {
		if itemIndex == index {
			return nil
		}
	}

	itemDisplay.ItemsSelectedIndices = append(itemDisplay.ItemsSelectedIndices, index)
	sel := inv.GetInventoryForDisplay(itemDisplay.ItemsSelectedIndices)

	return itemDisplay.GetInventoryListWidget(sel)

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
			return e.(gear.InventoryListEntry).Name + " x" + strconv.Itoa(e.(gear.InventoryListEntry).Count)
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

	itemDisplay.RooWindow = widget.NewWindow(

		widget.WindowOpts.Contents(itemDisplay.RootContainer),

		widget.WindowOpts.TitleBar(titleContainer, 25),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.CLICK_OUT),
		widget.WindowOpts.Draggable(),
		widget.WindowOpts.Resizeable(),
		widget.WindowOpts.MinSize(500, 500),

		widget.WindowOpts.MoveHandler(func(args *widget.WindowChangedEventArgs) {
			fmt.Println("Window Moving")
		}),
		//Set the callback that triggers when a resize is complete
		widget.WindowOpts.ResizeHandler(func(args *widget.WindowChangedEventArgs) {
			fmt.Println("Window Resized")
		}),
	)

}

func (itemDisplay *ItemDisplay) GetInventory() *gear.Inventory {
	return itemDisplay.inventory
}

type PlayerItemsUI struct {
	rootContainer *widget.Container //The main for the inventory window

	CraftingItemDisplay  CraftingItemDisplay
	ThrowableItemDisplay ThrowingItemDisplay
	EquipmentDisplay     EquipmentItemDisplay
	ConsumableDisplay    ConsumableItemDisplay
}

// Create the container that contains the widgets for displaying the different views of the inventory
func CreateItemContainers(playerUI *PlayerUI, inv *gear.Inventory, pl *avatar.PlayerData, ui *ebitenui.UI) *widget.Container {

	// Main container that will hold the container for available items and the items selected
	playerUI.ItemsUI.rootContainer = widget.NewContainer(
	//widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
	)

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

	itemDisplayOptionsContainer.AddChild(CreateOpenCraftingButton(playerUI, inv, ui))
	itemDisplayOptionsContainer.AddChild(CreateOpenThrowablesButton(playerUI, inv, ui))
	itemDisplayOptionsContainer.AddChild(CreateOpenEquipmentButton(playerUI, inv, ui))
	itemDisplayOptionsContainer.AddChild(CreateOpenConsumablesButton(playerUI, inv, ui))

	playerUI.ItemsUI.ThrowableItemDisplay.playerData = pl

	CreateItemManagementUI(playerUI, pl.Inv)

	return itemDisplayOptionsContainer

}

// Creates the main UI container
func CreatePlayerItemsUI(playerUI *PlayerUI, inv *gear.Inventory, pl *avatar.PlayerData) *ebitenui.UI {

	ui := ebitenui.UI{}

	// construct a new container that serves as the root of the UI hierarchy
	rootContainer := widget.NewContainer(

		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(30)),
		)),
	)

	inventoryAnchorContainer := widget.NewContainer(

		widget.ContainerOpts.WidgetOpts(

			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				StretchHorizontal:  true,
				StretchVertical:    true,
			}),
			widget.WidgetOpts.MinSize(100, 100),
		),
	)

	statsAnchorContainer := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				StretchHorizontal:  true,
				StretchVertical:    true,
			}),
			widget.WidgetOpts.MinSize(400, 300), // Adjust as necessary
		),
	)

	itemDisplayOptionsContainer := CreateItemContainers(playerUI, inv, pl, &ui)

	//playerUI.StatsUI.CreateStatsUI()
	inventoryAnchorContainer.AddChild(itemDisplayOptionsContainer)

	debugLabel := widget.NewText(
		widget.TextOpts.Text("Label 1 (NewText)", face, color.White),
	)
	//Set the

	statsAnchorContainer.AddChild(debugLabel) //debug why this does nt show up

	rootContainer.AddChild(inventoryAnchorContainer)
	rootContainer.AddChild(statsAnchorContainer)

	ui.Container = rootContainer

	return &ui

}

// Creating the button that opens the crafting menu.
func CreateOpenCraftingButton(playerUI *PlayerUI, inv *gear.Inventory, ui *ebitenui.UI) *widget.Button {
	// construct a button
	button := widget.NewButton(

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

			x, y := playerUI.ItemsUI.CraftingItemDisplay.ItmDisplay.RooWindow.Contents.PreferredSize()

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 50})
			playerUI.ItemsUI.CraftingItemDisplay.ItmDisplay.RooWindow.SetLocation(r)
			playerUI.ItemsUI.CraftingItemDisplay.DisplayInventory(inv)
			ui.AddWindow(playerUI.ItemsUI.CraftingItemDisplay.ItmDisplay.RooWindow)

		}),
	)

	return button

}

// Creating the button that opens the crafting menu. Other buttons will be added
// Doing it inside a function makes the code easier to follow
func CreateOpenThrowablesButton(playerUI *PlayerUI, inv *gear.Inventory, ui *ebitenui.UI) *widget.Button {
	// construct a button
	button := widget.NewButton(

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

			x, y := playerUI.ItemsUI.ThrowableItemDisplay.ItmDisp.RooWindow.Contents.PreferredSize()

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 200})
			playerUI.ItemsUI.ThrowableItemDisplay.ItmDisp.RooWindow.SetLocation(r)
			playerUI.ItemsUI.ThrowableItemDisplay.DisplayInventory()
			ui.AddWindow(playerUI.ItemsUI.ThrowableItemDisplay.ItmDisp.RooWindow)

		}),
	)

	return button

}

// Creating the button that opens the crafting menu. Other buttons will be added
// Doing it inside a function makes the code easier to follow
func CreateOpenEquipmentButton(playerUI *PlayerUI, inv *gear.Inventory, ui *ebitenui.UI) *widget.Button {
	// construct a button
	button := widget.NewButton(

		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Equipment", face, &widget.ButtonTextColor{
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

			x, y := playerUI.ItemsUI.EquipmentDisplay.ItmDisplay.RooWindow.Contents.PreferredSize()

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 50})
			playerUI.ItemsUI.EquipmentDisplay.ItmDisplay.RooWindow.SetLocation(r)
			playerUI.ItemsUI.EquipmentDisplay.DisplayInventory(inv)
			ui.AddWindow(playerUI.ItemsUI.EquipmentDisplay.ItmDisplay.RooWindow)

		}),
	)

	return button

}

// Creating the button that opens the crafting menu. Other buttons will be added
// Doing it inside a function makes the code easier to follow
func CreateOpenConsumablesButton(playerUI *PlayerUI, inv *gear.Inventory, ui *ebitenui.UI) *widget.Button {
	// construct a button
	button := widget.NewButton(

		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Consumables", face, &widget.ButtonTextColor{
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

			x, y := playerUI.ItemsUI.ConsumableDisplay.ItmDisplay.RooWindow.Contents.PreferredSize()

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 50})
			playerUI.ItemsUI.ConsumableDisplay.ItmDisplay.RooWindow.SetLocation(r)
			playerUI.ItemsUI.ConsumableDisplay.DisplayInventory(inv)
			ui.AddWindow(playerUI.ItemsUI.ConsumableDisplay.ItmDisplay.RooWindow)

		}),
	)

	return button

}

//For creating a window that the Item Display related containers are shown in

func CreateItemManagementUI(playerUI *PlayerUI, playerInventory *gear.Inventory) {

	playerUI.ItemsUI.CraftingItemDisplay.ItmDisplay.inventory = playerInventory
	playerUI.ItemsUI.ThrowableItemDisplay.ItmDisp.inventory = playerInventory
	playerUI.ItemsUI.EquipmentDisplay.ItmDisplay.inventory = playerInventory
	playerUI.ItemsUI.ConsumableDisplay.ItmDisplay.inventory = playerInventory

	playerUI.ItemsUI.CraftingItemDisplay.CreateContainers()
	playerUI.ItemsUI.CraftingItemDisplay.ItmDisplay.CreateInventoryDisplayWindow("Crafting Window")

	playerUI.ItemsUI.ThrowableItemDisplay.CreateContainers()
	playerUI.ItemsUI.ThrowableItemDisplay.ItmDisp.CreateInventoryDisplayWindow("Throwing Window")

	playerUI.ItemsUI.EquipmentDisplay.CreateContainers()
	playerUI.ItemsUI.EquipmentDisplay.ItmDisplay.CreateInventoryDisplayWindow("Equipment Window")

	playerUI.ItemsUI.ConsumableDisplay.CreateContainers()
	playerUI.ItemsUI.ConsumableDisplay.ItmDisplay.CreateInventoryDisplayWindow("Consumable Window")

}
