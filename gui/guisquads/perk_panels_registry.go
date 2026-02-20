package guisquads

import (
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for perk mode
const (
	PerkPanelSelector framework.PanelType = "perk_selector"
	PerkPanelContent  framework.PanelType = "perk_content"
)

func init() {
	// Register selector panel (squad nav + level buttons + unit nav)
	framework.RegisterPanel(PerkPanelSelector, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			pm := mode.(*PerkMode)
			layout := pm.Layout

			selectorWidth := int(float64(layout.ScreenWidth) * 0.4)
			// Taller than artifact selector (3 rows vs 1) but still compact
			selectorHeight := int(float64(layout.ScreenHeight) * specs.CommanderSelectorHeight * 1.8)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  selectorWidth,
				MinHeight: selectorHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(3),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
				),
			})

			topPad := int(float64(layout.ScreenHeight) * specs.PaddingExtraSmall)
			result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

			// Row 1: Squad navigation
			squadNavRow := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(10),
				)),
			)

			prevSquadBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "< Prev Squad",
				OnClick: func() { pm.cycleSquad(-1) },
			})
			squadNavRow.AddChild(prevSquadBtn)

			squadCounterLabel := builders.CreateSmallLabel("Squad 1 of 1")
			squadNavRow.AddChild(squadCounterLabel)

			nextSquadBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Next Squad >",
				OnClick: func() { pm.cycleSquad(1) },
			})
			squadNavRow.AddChild(nextSquadBtn)

			result.Container.AddChild(squadNavRow)

			// Row 2: Level buttons
			levelRow := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)

			squadLevelBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Squad Perks",
				OnClick: func() { pm.switchLevel("squad") },
			})
			levelRow.AddChild(squadLevelBtn)

			unitLevelBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Unit Perks",
				OnClick: func() { pm.switchLevel("unit") },
			})
			levelRow.AddChild(unitLevelBtn)

			commanderLevelBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Commander Perks",
				OnClick: func() { pm.switchLevel("commander") },
			})
			levelRow.AddChild(commanderLevelBtn)

			result.Container.AddChild(levelRow)

			// Row 3: Unit navigation (hidden by default)
			unitNavContainer := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(10),
				)),
			)

			prevUnitBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "< Prev Unit",
				OnClick: func() { pm.cycleUnit(-1) },
			})
			unitNavContainer.AddChild(prevUnitBtn)

			unitNameLabel := builders.CreateSmallLabel("Unit 1 of 1")
			unitNavContainer.AddChild(unitNameLabel)

			nextUnitBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Next Unit >",
				OnClick: func() { pm.cycleUnit(1) },
			})
			unitNavContainer.AddChild(nextUnitBtn)

			unitNavContainer.GetWidget().Visibility = widget.Visibility_Hide
			result.Container.AddChild(unitNavContainer)

			// Store references
			result.Custom["prevSquadButton"] = prevSquadBtn
			result.Custom["nextSquadButton"] = nextSquadBtn
			result.Custom["squadCounterLabel"] = squadCounterLabel
			result.Custom["squadLevelButton"] = squadLevelBtn
			result.Custom["unitLevelButton"] = unitLevelBtn
			result.Custom["commanderLevelButton"] = commanderLevelBtn
			result.Custom["unitNavContainer"] = unitNavContainer
			result.Custom["unitNameLabel"] = unitNameLabel
			result.Custom["prevUnitButton"] = prevUnitBtn
			result.Custom["nextUnitButton"] = nextUnitBtn

			return nil
		},
	})

	// Register main content panel with available + equipped tabs
	framework.RegisterPanel(PerkPanelContent, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			pm := mode.(*PerkMode)
			layout := pm.Layout

			// Match artifact panel width, reduce height to fit below taller selector
			listWidth := int(float64(layout.ScreenWidth) * specs.SquadEditorUnitListWidth)
			listHeight := int(float64(layout.ScreenHeight) * 0.55)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  listWidth,
				MinHeight: listHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				),
			})

			// Position well below the selector panel (selector has 2-3 rows of buttons)
			topOffset := int(float64(layout.ScreenHeight) * 0.20)
			result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(topOffset)

			// Tab row
			tabRow := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)
			tabRow.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Available",
				OnClick: func() { pm.switchTab("available") },
			}))
			tabRow.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Equipped",
				OnClick: func() { pm.switchTab("equipped") },
			}))
			tabRow.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Back (ESC)",
				OnClick: func() {
					if returnMode, exists := pm.ModeManager.GetMode(pm.GetReturnMode()); exists {
						pm.ModeManager.RequestTransition(returnMode, "Back button pressed")
					}
				},
			}))
			result.Container.AddChild(tabRow)

			// === Available content sub-container ===
			availableContent := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)

			availableTitle := builders.CreateSmallLabel("Available Perks (0)")
			availableContent.AddChild(availableTitle)

			availableList := builders.CreateSimpleStringList(builders.SimpleStringListConfig{
				Entries:       []string{"No perks available"},
				ScreenWidth:   400,
				ScreenHeight:  200,
				WidthPercent:  1.0,
				HeightPercent: 0.5,
			})
			availableContent.AddChild(availableList)

			availableDetail := builders.CreateTextAreaWithConfig(builders.TextAreaConfig{
				MinWidth:  380,
				MinHeight: 100,
				FontColor: color.White,
			})
			availableDetail.SetText("Select a perk to view details.")
			availableContent.AddChild(availableDetail)

			// Slot buttons row (dynamic based on level)
			slotButtonRow := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)
			// Default slot buttons for squad (3 slots)
			for i := 0; i < 3; i++ {
				slotIdx := i
				slotButtonRow.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
					Text:    "Equip Slot " + string(rune('1'+slotIdx)),
					OnClick: func() { pm.onEquipAction(slotIdx) },
				}))
			}
			availableContent.AddChild(slotButtonRow)

			result.Container.AddChild(availableContent)

			// === Equipped content sub-container (starts hidden) ===
			equippedContent := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)

			equippedTitle := builders.CreateSmallLabel("Equipped (0/3)")
			equippedContent.AddChild(equippedTitle)

			equippedList := builders.CreateSimpleStringList(builders.SimpleStringListConfig{
				Entries:       []string{"No perks equipped"},
				ScreenWidth:   400,
				ScreenHeight:  200,
				WidthPercent:  1.0,
				HeightPercent: 0.5,
			})
			equippedContent.AddChild(equippedList)

			equippedDetail := builders.CreateTextAreaWithConfig(builders.TextAreaConfig{
				MinWidth:  380,
				MinHeight: 100,
				FontColor: color.White,
			})
			equippedDetail.SetText("Select a slot to view details.")
			equippedContent.AddChild(equippedDetail)

			unequipBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Unequip",
				OnClick: func() { pm.onUnequipAction() },
			})
			unequipBtn.GetWidget().Disabled = true
			equippedContent.AddChild(unequipBtn)

			equippedContent.GetWidget().Visibility = widget.Visibility_Hide
			result.Container.AddChild(equippedContent)

			// Store all references
			result.Custom["availableContent"] = availableContent
			result.Custom["availableList"] = availableList
			result.Custom["availableTitle"] = availableTitle
			result.Custom["availableDetail"] = availableDetail
			result.Custom["slotButtonRow"] = slotButtonRow
			result.Custom["equippedContent"] = equippedContent
			result.Custom["equippedList"] = equippedList
			result.Custom["equippedTitle"] = equippedTitle
			result.Custom["equippedDetail"] = equippedDetail
			result.Custom["unequipButton"] = unequipBtn

			return nil
		},
	})
}
