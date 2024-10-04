package gui

import (
	"game_main/gear"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// EquipmentItemDisplay shows the currently equipped weapons and armor and lets the player equip other items
// Several TextAreas display the information the player
type EquipmentItemDisplay struct {
	ItmDisplay ItemDisplay

	EquipmentSelectedText *widget.TextArea //Displays the properties of the selected items that can be equipped

	currentItem                *ecs.Entity
	currentItemIndex           int               // Will make it easier to handle updatng the inventory display list.
	CurrentlyEquippedContainer *widget.Container // Contains all the widgets related to what's currently equipped
	MeleeWepText               *widget.TextArea  // Shows the stats of the currently equipped melee weapon
	RangeWepText               *widget.TextArea  // Shows the stats of the currently equipped ranged weapon
	ArmorText                  *widget.TextArea  // Shows the stats of the currently equipped armor

}

func (equipmentDisplay *EquipmentItemDisplay) CreateInventoryList(propFilters ...gear.StatusEffects) {

	inv := equipmentDisplay.ItmDisplay.GetInventory().GetEquipmentForDisplay([]int{})
	equipmentDisplay.ItmDisplay.InventoryDisplaylist = equipmentDisplay.ItmDisplay.GetInventoryListWidget(inv)

	equipmentDisplay.ItmDisplay.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		equipmentDisplay.ItmDisplay.ItemSelectedContainer.RemoveChild(equipmentDisplay.ItmDisplay.ItemsSelectedList)

		a := args.(*widget.ListEntrySelectedEventArgs)
		entry := a.Entry.(gear.InventoryListEntry)

		//Only allow one item to be selected
		equipmentDisplay.ItmDisplay.ItemsSelectedIndices = equipmentDisplay.ItmDisplay.ItemsSelectedIndices[:0]
		equipmentDisplay.ItmDisplay.ItemsSelectedList = equipmentDisplay.ItmDisplay.GetSelectedItems(entry.Index, equipmentDisplay.ItmDisplay.GetInventory())
		equipmentDisplay.EquipmentSelectedText.SetText("")

		if equipmentDisplay.ItmDisplay.ItemsSelectedList != nil {

			equipmentDisplay.currentItem, _ = equipmentDisplay.ItmDisplay.playerData.Inventory.GetItem(entry.Index)
			equipmentDisplay.EquipmentSelectedText.SetText(gear.ItemStats(equipmentDisplay.currentItem))
			equipmentDisplay.currentItemIndex = entry.Index

		}

	})

	equipmentDisplay.ItmDisplay.InventoryDisplayContainer.AddChild(equipmentDisplay.ItmDisplay.InventoryDisplaylist)

}

func (equipmentDisplay *EquipmentItemDisplay) DisplayInventory(inventory *gear.Inventory) {

	equipmentDisplay.CreateInventoryList()

}

func (equipmentDisplay *EquipmentItemDisplay) CreateRootContainer() {

	equipmentDisplay.ItmDisplay.RootContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(defaultWidgetColor),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			// It is using a GridLayout with a single column
			widget.GridLayoutOpts.Columns(5),
			widget.GridLayoutOpts.Stretch([]bool{true, true, true, true, true}, []bool{true, true, true, true, true}),

			//widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true, true, true}),
			// Padding defines how much space to put around the outside of the grid.
			widget.GridLayoutOpts.Padding(widget.Insets{
				Top:    10,
				Bottom: 10,
			}),
			// Spacing defines how much space to put between each column and row
			widget.GridLayoutOpts.Spacing(0, 20))),
	)

}

// Contains three columns - melee weapon, ranged weapon, and armor
// Shows the stats of what's currently equipped
func (equipmentDisplay *EquipmentItemDisplay) SetupCurrentlyEquippedContainer() {

	equipmentDisplay.CurrentlyEquippedContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(defaultWidgetColor),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			// It is using a GridLayout with a single column
			widget.GridLayoutOpts.Columns(3),

			widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{false, false, false}),

			//widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true, true, true}),
			// Padding defines how much space to put around the outside of the grid.
			widget.GridLayoutOpts.Padding(widget.Insets{
				Top:    150,
				Bottom: 150,
			}),
			// Spacing defines how much space to put between each column and row
			widget.GridLayoutOpts.Spacing(50, 15))),
	)

	equipmentDisplay.MeleeWepText = CreateTextArea(300, 300)

	equipmentDisplay.RangeWepText = CreateTextArea(300, 300)
	equipmentDisplay.ArmorText = CreateTextArea(300, 300)

}

func (equipmentDisplay *EquipmentItemDisplay) SetupContainers() {

	// Main container that will hold the container for available items and the items selected

	// Holds the widget that displays the selected items to the player

	equipmentDisplay.ItmDisplay.ItemSelectedContainer = widget.NewContainer(

		widget.ContainerOpts.BackgroundImage(defaultWidgetColor),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			// It is using a GridLayout with a single column
			widget.GridLayoutOpts.Columns(1),
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

	equipmentDisplay.EquipmentSelectedText = CreateTextArea(300, 300)
	equipmentDisplay.ItmDisplay.ItemSelectedContainer.AddChild(equipmentDisplay.EquipmentSelectedText)
	equipmentDisplay.EquipmentSelectedText.SetText("asdasdsa")

	button := CreateButton("Equip")

	button.Configure( // add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if equipmentDisplay.currentItem != nil {

				//Remove the old item and add it back to the inventory. Then add the new item to equipped and remove it from the inventory
				equipmentDisplay.ItmDisplay.playerData.RemoveItem(equipmentDisplay.currentItem)

				equipmentDisplay.ItmDisplay.playerData.Equipment.EquipItem(equipmentDisplay.currentItem, equipmentDisplay.ItmDisplay.playerData.PlayerEntity)
				equipmentDisplay.ItmDisplay.playerData.Inventory.RemoveItem(equipmentDisplay.currentItemIndex)
				equipmentDisplay.DisplayInventory(equipmentDisplay.ItmDisplay.playerData.Inventory)
				equipmentDisplay.UpdateEquipmentDisplay()
			}
		}))

	equipmentDisplay.ItmDisplay.ItemSelectedContainer.AddChild(button)

	//Todo add the next to display
	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.ItmDisplay.InventoryDisplayContainer)

	equipmentDisplay.SetupCurrentlyEquippedContainer()

	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.ItmDisplay.InventoryDisplayContainer)
	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.ItmDisplay.ItemSelectedContainer)
	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.CurrentlyEquippedContainer)

	//This adds it to the next row
	equipmentDisplay.CurrentlyEquippedContainer.AddChild(equipmentDisplay.MeleeWepText)
	equipmentDisplay.CurrentlyEquippedContainer.AddChild(equipmentDisplay.RangeWepText)
	equipmentDisplay.CurrentlyEquippedContainer.AddChild(equipmentDisplay.ArmorText)

	button = CreateButton("Remove\nMelee")

	button.Configure( // add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {

			equipmentDisplay.ItmDisplay.playerData.UnequipMeleeWeapon()
			equipmentDisplay.UpdateEquipmentDisplay()

		}))

	equipmentDisplay.CurrentlyEquippedContainer.AddChild(button)

	button = CreateButton("Remove\nRanged")

	button.Configure( // add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {

			equipmentDisplay.ItmDisplay.playerData.UnequipRangedWeapon()
			equipmentDisplay.UpdateEquipmentDisplay()

		}))

	equipmentDisplay.CurrentlyEquippedContainer.AddChild(button)

	button = CreateButton("Remove\nArmor")

	button.Configure( // add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {

			equipmentDisplay.ItmDisplay.playerData.UnequipArmor()
			equipmentDisplay.UpdateEquipmentDisplay()

		}))

	equipmentDisplay.CurrentlyEquippedContainer.AddChild(button)

}

func (equipmentDisplay *EquipmentItemDisplay) UpdateEquipmentDisplay() {

	playerEquipment := equipmentDisplay.ItmDisplay.playerData.Equipment

	equipmentDisplay.MeleeWepText.SetText("None")
	equipmentDisplay.RangeWepText.SetText("None")
	equipmentDisplay.ArmorText.SetText("None")

	if playerEquipment.EqMeleeWeapon != nil {

		equipmentDisplay.MeleeWepText.SetText(gear.ItemStats(playerEquipment.EqMeleeWeapon))

	}

	if playerEquipment.EqRangedWeapon != nil {
		equipmentDisplay.RangeWepText.SetText(gear.ItemStats(playerEquipment.EqRangedWeapon))
	}

	if playerEquipment.EqArmor != nil {
		equipmentDisplay.ArmorText.SetText(gear.ItemStats(playerEquipment.EqArmor))
	}

	equipmentDisplay.DisplayInventory(equipmentDisplay.ItmDisplay.playerData.Inventory) //To show the added item

}
