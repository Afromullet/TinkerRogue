package gui

import (
	"game_main/common"
	"game_main/gear"
	"image/color"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

type ConsumableItemDisplay struct {
	playerAttributes          *common.Attributes
	ItmDisplay                ItemDisplay
	ItemSelectedContainer     *widget.Container //Displays the items the user HAS selected for crafitng
	ItemSelectedStatsTextArea *widget.TextArea  //Displays the properties of the selected items

}

// Selects an item and adds it to the ItemsSelectedContainer container and ItemsSelectedPropContainer
// ItemSeleced container tells us which items we're crafting with
// ItemsSelectedPropContainer tells which properties the items have
func (consDisplay *ConsumableItemDisplay) CreateInventoryList(propFilters ...gear.StatusEffects) {

	inv := consDisplay.ItmDisplay.GetInventory().GetConsumablesForDisplay([]int{})
	consDisplay.ItmDisplay.InventoryDisplaylist = consDisplay.ItmDisplay.GetInventoryListWidget(inv)

	consDisplay.ItmDisplay.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		consDisplay.ItemSelectedContainer.RemoveChild(consDisplay.ItmDisplay.ItemsSelectedList)

		a := args.(*widget.ListEntrySelectedEventArgs)
		entry := a.Entry.(gear.InventoryListEntry)

		//Only allow one item to be selected
		consDisplay.ItmDisplay.ItemsSelectedIndices = consDisplay.ItmDisplay.ItemsSelectedIndices[:0]
		consDisplay.ItmDisplay.ItemsSelectedList = consDisplay.ItmDisplay.GetSelectedItems(entry.Index, consDisplay.ItmDisplay.GetInventory())

		//Todo, don't add a list. Just set the text
		if consDisplay.ItmDisplay.ItemsSelectedList != nil {
			//consDisplay.ItemSelectedContainer.AddChild(consDisplay.ItmDisplay.ItemsSelectedList)

			item, _ := consDisplay.ItmDisplay.GetInventory().GetItem(entry.Index)
			cons := common.GetComponentType[*gear.Consumable](item, gear.ConsumableComponent)

			consDisplay.ItemSelectedStatsTextArea.SetText("")
			consDisplay.ItemSelectedStatsTextArea.AppendText(cons.ConsumableInfo())

		}

	})

	consDisplay.ItmDisplay.ItemDisplayContainer.AddChild(consDisplay.ItmDisplay.InventoryDisplaylist)

}

func (consDisplay *ConsumableItemDisplay) DisplayInventory() {

	consDisplay.CreateInventoryList()

}

func (consDisplay *ConsumableItemDisplay) CreateContainers() {

	consDisplay.CreateConsumableStatsTextArea()

	// Main container that will hold the container for available items and the items selected
	consDisplay.ItmDisplay.RootContainer = widget.NewContainer(
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
	consDisplay.ItemSelectedContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
		)))

	consDisplay.ItmDisplay.ItemDisplayContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	consDisplay.ItmDisplay.RootContainer.AddChild(consDisplay.ItmDisplay.ItemDisplayContainer)
	consDisplay.ItemSelectedContainer.AddChild(consDisplay.ItemSelectedStatsTextArea)

	button := consDisplay.CreateUseConsumableButton()
	consDisplay.ItemSelectedContainer.AddChild(button)
	consDisplay.ItmDisplay.RootContainer.AddChild(consDisplay.ItemSelectedContainer)

	//Holds the window that will display item properties

}

// Text window to display the item properties of the selected items to the player
func (consDisplay *ConsumableItemDisplay) CreateConsumableStatsTextArea() {
	// construct a textarea
	consDisplay.ItemSelectedStatsTextArea = widget.NewTextArea(
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

// Creating the button that opens the crafting menu.
func (consDisplay *ConsumableItemDisplay) CreateUseConsumableButton() *widget.Button {
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
		widget.ButtonOpts.Text("Use Consumable", face, &widget.ButtonTextColor{
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

			inv := consDisplay.ItmDisplay.GetInventory()

			item, _ := inv.GetItem(consDisplay.ItmDisplay.ItemsSelectedIndices[0])
			consumable := common.GetComponentType[*gear.Consumable](item, gear.ConsumableComponent)
			gear.AddEffectToTracker(consDisplay.ItmDisplay.playerEntity, *consumable)

			inv.RemoveItem(consDisplay.ItmDisplay.ItemsSelectedIndices[0])
			consDisplay.DisplayInventory()

			//consumable.ApplyEffect(consDisplay.playerAttributes)

		}),
	)

	return button

}
