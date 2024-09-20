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
				Top:    50,
				Bottom: 50,
			}),
			// Spacing defines how much space to put between each column and row
			widget.GridLayoutOpts.Spacing(0, 20))),
	)

}

func (equipmentDisplay *EquipmentItemDisplay) SetupContainers() {

	// Main container that will hold the container for available items and the items selected

	// Holds the widget that displays the selected items to the player

	equipmentDisplay.MeleeWepText = CreateTextArea()
	equipmentDisplay.RangeWepText = CreateTextArea()
	equipmentDisplay.ArmorText = CreateTextArea()

	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.ItmDisplay.ItemDisplayContainer)
	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.ItmDisplay.ItemSelectedContainer)
	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.MeleeWepText)
	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.RangeWepText)
	equipmentDisplay.ItmDisplay.RootContainer.AddChild(equipmentDisplay.ArmorText)

}

func (equipmentDisplay *EquipmentItemDisplay) UpdateEquipmentDisplayText() {

	//pl := equipmentDisplay.ItmDisplay.playerEntity

	//armor := common.GetComponentType[*gear.Armor](pl, gear.ArmorComponent)

	playerEquipment := equipmentDisplay.ItmDisplay.playerData.Equipment

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
