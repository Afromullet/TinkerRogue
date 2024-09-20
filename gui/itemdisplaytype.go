package gui

import (
	"fmt"
	"game_main/avatar"
	"game_main/gear"
	"image/color"
	_ "image/png"
	"strconv"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

/*
Any new window that displays a filtered inventory is a new type containing ItemDisplay and any additional windows needed.
I.E, Equipment, Throwables, and Consumables are all their own type with an ItemDisplay. The type implements the ItemDisplayer interface


CreatteRootContainer creates the root container that holds everything for the ItemDisplayer
SetupContainers creates any other containers unique to the implementation and adds them to root
CreateInventoryList decides how to filter the inventory for display
DisplayInventory just calls CreateInventoryList. Maybe change it later so it's not a method everyone has to implement, since it's the same for everyone
*/

// Every window that displays the inventory to teh user will be a struct that contains ItemDisplay
// And implements the ItemDisplayer interface
type ItemDisplayer interface {
	CreateRootContainer()                                  //Really just there for calling CreateInventoryList with ItemProperty filters for the specific kind of window
	SetupContainers()                                      //For creating the containers
	CreateInventoryList(propFilters ...gear.StatusEffects) //For getting the inventory from the player and adding on click event handlers
	DisplayInventory()
}

type ItemDisplay struct {
	playerData    *avatar.PlayerData
	RootContainer *widget.Container //Holds all of the GUI elements
	RooWindow     *widget.Window    //Window to hold the root container content

	ItemSelectedContainer *widget.Container
	ItemDisplayContainer  *widget.Container //Container that holds the items to be displayed
	InventoryDisplaylist  *widget.List      //Holds all of the items
	ItemsSelectedList     *widget.List      //Holds only the items the player selects
	ItemsSelectedIndices  []int             //The indices in inventoryDisplayList of the items the user selected

}

func (itemDisplay *ItemDisplay) createItemSelectedContainer() {

	// Holds the widget that displays the selected items to the player
	itemDisplay.ItemSelectedContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
		)))

}

func (itemDisplay *ItemDisplay) createItemDisplayContainer() {

	itemDisplay.ItemDisplayContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
}

func (itemDisplay *ItemDisplay) createInventoryDisplayWindow(title string) {

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

func (itemDisplay *ItemDisplay) GetInventory() *gear.Inventory {
	return itemDisplay.playerData.Inventory
}
