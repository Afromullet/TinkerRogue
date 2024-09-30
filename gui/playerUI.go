package gui

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/graphics"

	"game_main/gear"
	"image"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
)

type PlayerUI struct {
	ItemsUI             PlayerItemsUI
	StatsUI             PlayerStatsUI
	MsgUI               PlayerMessageUI
	InformationUI       InfoUI
	MainPlayerInterface *ebitenui.UI
}

// Throwing an item will show a square to represent the AOE of the throwable.
// Right now it's a function of Game until I separate the UI more.
// Not going to try to generalize/abstract this until I figure out how I want to handle this
// The impression I get now is that this will take a "state machine" since the throwable window closes
// Once I click out of it
func (p *PlayerUI) IsThrowableItemSelected() bool {

	return p.ItemsUI.ThrowableItemDisplay.ThrowableItemSelected

}

func (p *PlayerUI) SetThrowableItemSelected(selected bool) {

	p.ItemsUI.ThrowableItemDisplay.ThrowableItemSelected = selected

}

// func CreatePlayerItemsUI(playerUI *PlayerUI, inv *gear.Inventory, pl *avatar.PlayerData)
func (playerUI *PlayerUI) CreateMainInterface(playerData *avatar.PlayerData, ecsmanager *common.EntityManager) {

	playerUI.MainPlayerInterface = CreatePlayerUI(playerUI, playerData.Inventory, playerData, ecsmanager)

}

// Creates the main UI container
func CreatePlayerUI(playerUI *PlayerUI, inv *gear.Inventory, pl *avatar.PlayerData, ecsmanager *common.EntityManager) *ebitenui.UI {

	ui := ebitenui.UI{}

	// construct a new container that serves as the root of the UI hierarchy
	rootContainer := widget.NewContainer(

	/*
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(30)),
		)),
	*/
	)

	itemDisplayOptionsContainer := CreateInventorySelectionContainer(playerUI, inv, pl, &ui) //Contains the buttons for opening inventory related windows

	playerUI.StatsUI.CreateStatsUI()
	playerUI.StatsUI.StatsTextArea.SetText(pl.GetPlayerAttributes().AttributeText())

	playerUI.MsgUI.CreatMsgUI()
	playerUI.MsgUI.msgTextArea.SetText("adadsdasdsa") //Placeholder

	rootContainer.AddChild(itemDisplayOptionsContainer)
	rootContainer.AddChild(playerUI.StatsUI.StatUIContainer)
	rootContainer.AddChild(playerUI.MsgUI.msgUIContainer)

	SetContainerLocation(itemDisplayOptionsContainer, 0, 0)
	SetContainerLocation(playerUI.StatsUI.StatUIContainer, graphics.ScreenInfo.GetCanvasWidth(), 0)
	SetContainerLocation(playerUI.MsgUI.msgUIContainer, graphics.ScreenInfo.GetCanvasWidth(), graphics.ScreenInfo.GetCanvasHeight()/4+graphics.ScreenInfo.TileHeight) //Placing it one tile under the Stats Container
	playerUI.InformationUI = CreateInfoUI(ecsmanager, &ui)

	ui.Container = rootContainer

	return &ui

}

// Creating the button that opens the crafting menu. Other buttons will be added
// Doing it inside a function makes the code easier to follow
func CreateOpenThrowablesButton(playerUI *PlayerUI, inv *gear.Inventory, ui *ebitenui.UI) *widget.Button {
	// construct a button
	button := CreateButton("Throwables")

	button.Configure( // add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {

			x, y := playerUI.ItemsUI.ThrowableItemDisplay.ItemDisplay.RootWindow.Contents.PreferredSize()

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 200})
			playerUI.ItemsUI.ThrowableItemDisplay.ItemDisplay.RootWindow.SetLocation(r)
			playerUI.ItemsUI.ThrowableItemDisplay.DisplayInventory()
			ui.AddWindow(playerUI.ItemsUI.ThrowableItemDisplay.ItemDisplay.RootWindow)
			//consumable.ApplyEffect(consDisplay.playerAttributes)

		}))

	return button

}

// Creating the button that opens the crafting menu. Other buttons will be added
// Doing it inside a function makes the code easier to follow
func CreateOpenEquipmentButton(playerUI *PlayerUI, inv *gear.Inventory, ui *ebitenui.UI) *widget.Button {
	// construct a button
	button := CreateButton("Equipment")

	button.Configure( // add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			x, y := playerUI.ItemsUI.EquipmentDisplay.ItmDisplay.RootWindow.Contents.PreferredSize()

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 50})
			playerUI.ItemsUI.EquipmentDisplay.ItmDisplay.RootWindow.SetLocation(r)
			playerUI.ItemsUI.EquipmentDisplay.DisplayInventory(inv)
			ui.AddWindow(playerUI.ItemsUI.EquipmentDisplay.ItmDisplay.RootWindow)
			playerUI.ItemsUI.EquipmentDisplay.UpdateEquipmentDisplay()
		}))

	return button

}

// Creating the button that opens the crafting menu. Other buttons will be added
// Doing it inside a function makes the code easier to follow
func CreateOpenConsumablesButton(playerUI *PlayerUI, inv *gear.Inventory, ui *ebitenui.UI) *widget.Button {
	// construct a button
	button := CreateButton("Consumables")

	button.Configure( // add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {

			x, y := playerUI.ItemsUI.ConsumableDisplay.ItmDisplay.RootWindow.Contents.PreferredSize()

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 50})
			playerUI.ItemsUI.ConsumableDisplay.ItmDisplay.RootWindow.SetLocation(r)
			playerUI.ItemsUI.ConsumableDisplay.DisplayInventory() //Remove the item from display
			ui.AddWindow(playerUI.ItemsUI.ConsumableDisplay.ItmDisplay.RootWindow)

		}))

	return button

}

// Creating the button that opens the crafting menu.
func CreateOpenCraftingButton(playerUI *PlayerUI, inv *gear.Inventory, ui *ebitenui.UI) *widget.Button {
	// construct a button
	button := CreateButton("Crafting")

	button.Configure( // add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {

			x, y := playerUI.ItemsUI.CraftingItemDisplay.ItmDisplay.RootWindow.Contents.PreferredSize()

			r := image.Rect(0, 0, x, y)
			r = r.Add(image.Point{200, 50})
			playerUI.ItemsUI.CraftingItemDisplay.ItmDisplay.RootWindow.SetLocation(r)
			playerUI.ItemsUI.CraftingItemDisplay.DisplayInventory(inv)
			ui.AddWindow(playerUI.ItemsUI.CraftingItemDisplay.ItmDisplay.RootWindow)

		}))

	return button

}
