package gui

import (
	"game_main/gear"
	"image/color"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

// EquipmentItemDisplay shows the currently equipped weapons and armor and lets the player equip other items
// Several TextAreas display the information the player
type EquipmentItemDisplay struct {
	ItmDisplay ItemDisplay

	CurrentlyEquippedContainer *widget.Container // Contains all the widgets related to what's currently equipped
	EquipmentSelectedContainer *widget.Container //Container the stats of the selected item
	EquipmentSelectedText      *widget.TextArea  //Displays the properties of the selected items
	MeleeWepText               *widget.TextArea  //Displays the properties of the selected items
	RangeWepText               *widget.TextArea  //Displays the properties of the selected items
	ArmorText                  *widget.TextArea  //Displays the properties of the selected items

}

func (equipmentDisplay *EquipmentItemDisplay) CreateInventoryList(propFilters ...gear.StatusEffects) {

	inv := equipmentDisplay.ItmDisplay.GetInventory().GetEquipmentForDisplay([]int{})
	equipmentDisplay.ItmDisplay.InventoryDisplaylist = equipmentDisplay.ItmDisplay.GetInventoryListWidget(inv)

	equipmentDisplay.ItmDisplay.InventoryDisplaylist.EntrySelectedEvent.AddHandler(func(args interface{}) {

		equipmentDisplay.ItmDisplay.ItemSelectedContainer.RemoveChild(equipmentDisplay.ItmDisplay.ItemsSelectedList)

		a := args.(*widget.ListEntrySelectedEventArgs)
		entry := a.Entry.(gear.InventoryListEntry)

		equipmentDisplay.ItmDisplay.ItemsSelectedList = equipmentDisplay.ItmDisplay.GetSelectedItems(entry.Index, equipmentDisplay.ItmDisplay.GetInventory())

		if equipmentDisplay.ItmDisplay.ItemsSelectedList != nil {
			equipmentDisplay.ItmDisplay.ItemSelectedContainer.AddChild(equipmentDisplay.ItmDisplay.ItemsSelectedList)

		}

	})

	equipmentDisplay.ItmDisplay.ItemDisplayContainer.AddChild(equipmentDisplay.ItmDisplay.InventoryDisplaylist)

}

func (equipmentDisplay *EquipmentItemDisplay) DisplayInventory(inventory *gear.Inventory) {

	equipmentDisplay.CreateInventoryList()

}

func (equipmentDisplay *EquipmentItemDisplay) CreateRootContainer() {

	equipmentDisplay.ItmDisplay.RootContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
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

func (equipmentDisplay *EquipmentItemDisplay) SetupCurrentlyEquippedContainer() {

	equipmentDisplay.CurrentlyEquippedContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255})),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			// It is using a GridLayout with a single column
			widget.GridLayoutOpts.Columns(3),

			widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{false, false, false}),

			//widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true, true, true}),
			// Padding defines how much space to put around the outside of the grid.
			widget.GridLayoutOpts.Padding(widget.Insets{
				Top:    15,
				Bottom: 15,
			}),
			// Spacing defines how much space to put between each column and row
			widget.GridLayoutOpts.Spacing(15, 15))),
	)

}

func (equipmentDisplay *EquipmentItemDisplay) SetupContainers() {

	// Main container that will hold the container for available items and the items selected

	// Holds the widget that displays the selected items to the player

	equipmentDisplay.SetupCurrentlyEquippedContainer()

	equipmentDisplay.MeleeWepText = CreateTextArea()
	equipmentDisplay.RangeWepText = CreateTextArea()
	equipmentDisplay.ArmorText = CreateTextArea()

	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.ItmDisplay.ItemDisplayContainer)
	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.ItmDisplay.ItemSelectedContainer)
	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.CurrentlyEquippedContainer)

	equipmentDisplay.CurrentlyEquippedContainer.AddChild(equipmentDisplay.MeleeWepText)
	equipmentDisplay.CurrentlyEquippedContainer.AddChild(equipmentDisplay.RangeWepText)
	equipmentDisplay.CurrentlyEquippedContainer.AddChild(equipmentDisplay.ArmorText)

	button := CreateButton("Remove Melee Weapon")

	button.Configure( // add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {

			equipmentDisplay.ItmDisplay.playerData.UnequipMeleeWeapon()
			equipmentDisplay.UpdateEquipmentDisplay()

		}))

	equipmentDisplay.CurrentlyEquippedContainer.AddChild(button)

	button = CreateButton("Remove Ranged Weapon")
	equipmentDisplay.CurrentlyEquippedContainer.AddChild(button)

	button = CreateButton("Remove Armor")
	equipmentDisplay.CurrentlyEquippedContainer.AddChild(button)

}

func (equipmentDisplay *EquipmentItemDisplay) UpdateEquipmentDisplay() {

	//pl := equipmentDisplay.ItmDisplay.playerEntity

	//armor := common.GetComponentType[*gear.Armor](pl, gear.ArmorComponent)

	playerEquipment := equipmentDisplay.ItmDisplay.playerData.Equipment

	equipmentDisplay.MeleeWepText.SetText("None")
	equipmentDisplay.RangeWepText.SetText("None")
	equipmentDisplay.ArmorText.SetText("None")

	if playerEquipment.PlayerMeleeWeapon != nil {

		equipmentDisplay.MeleeWepText.SetText(playerEquipment.GetPlayerMeleeWeapon().WeaponString())
	}

	if playerEquipment.PlayerRangedWeapon != nil {
		equipmentDisplay.RangeWepText.SetText(playerEquipment.GetPlayerRangedWeapon().WeaponString())
	}

	if playerEquipment.PlayerArmor != nil {
		equipmentDisplay.ArmorText.SetText(playerEquipment.PlayerArmor.ArmorString())
	}

}
