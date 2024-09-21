package gui

import (
	"game_main/common"
	"game_main/gear"

	"github.com/ebitenui/ebitenui/widget"
)

// ConsumableItemDisplay allows the user to use consumables
// ConsumableEffectText displays the effects of the consumable when used
type ConsumableItemDisplay struct {
	ItmDisplay           ItemDisplay
	ConsumableEffectText *widget.TextArea //Displays the consumable ffects

}

func (consDisplay *ConsumableItemDisplay) CreateInventoryList(propFilters ...gear.StatusEffects) {

	inv := consDisplay.ItmDisplay.GetInventory().GetConsumablesForDisplay([]int{})
	consDisplay.ItmDisplay.InventoryDisplaylist = consDisplay.ItmDisplay.GetInventoryListWidget(inv)

	consDisplay.ItmDisplay.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		consDisplay.ItmDisplay.ItemSelectedContainer.RemoveChild(consDisplay.ItmDisplay.ItemsSelectedList)

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

			consDisplay.ConsumableEffectText.SetText("")
			consDisplay.ConsumableEffectText.AppendText(cons.ConsumableInfo())

		}

	})

	consDisplay.ItmDisplay.InventoryDisplayContainer.AddChild(consDisplay.ItmDisplay.InventoryDisplaylist)

}

func (consDisplay *ConsumableItemDisplay) DisplayInventory() {

	consDisplay.CreateInventoryList()

}

func (consDisplay *ConsumableItemDisplay) CreateRootContainer() {

	consDisplay.ItmDisplay.RootContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(defaultWidgetColor),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			// It is using a GridLayout with a single column
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, true}, []bool{true, true}),

			//widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true, true, true}),
			// Padding defines how much space to put around the outside of the grid.
			widget.GridLayoutOpts.Padding(widget.Insets{
				Top:    50,
				Bottom: 50,
			}),
			// Spacing defines how much space to put between each column and row
			widget.GridLayoutOpts.Spacing(0, 20))),
	)

}

func (consDisplay *ConsumableItemDisplay) SetupContainers() {

	consDisplay.ConsumableEffectText = CreateTextArea()
	consDisplay.ItmDisplay.RootContainer.AddChild(consDisplay.ItmDisplay.InventoryDisplayContainer)

	consDisplay.ItmDisplay.ItemSelectedContainer.AddChild(consDisplay.ConsumableEffectText)
	consDisplay.ItmDisplay.ItemSelectedContainer.AddChild(consDisplay.CreateUseConsumableButton())
	consDisplay.ItmDisplay.RootContainer.AddChild(consDisplay.ItmDisplay.ItemSelectedContainer)

}

// Creating the button that lets use a consumable
func (consDisplay *ConsumableItemDisplay) CreateUseConsumableButton() *widget.Button {

	// construct a button
	button := CreateButton("Use")

	button.Configure( // add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {

			inv := consDisplay.ItmDisplay.GetInventory()

			if len(consDisplay.ItmDisplay.ItemsSelectedIndices) > 0 {
				item, _ := inv.GetItem(consDisplay.ItmDisplay.ItemsSelectedIndices[0])
				consumable := common.GetComponentType[*gear.Consumable](item, gear.ConsumableComponent)
				gear.AddEffectToTracker(consDisplay.ItmDisplay.playerData.PlayerEntity, *consumable)

				inv.RemoveItem(consDisplay.ItmDisplay.ItemsSelectedIndices[0])
				consDisplay.DisplayInventory()
			}

			consDisplay.ItmDisplay.ItemsSelectedIndices = consDisplay.ItmDisplay.ItemsSelectedIndices[:0]

			//consumable.ApplyEffect(consDisplay.playerAttributes)

		}))

	return button

}
