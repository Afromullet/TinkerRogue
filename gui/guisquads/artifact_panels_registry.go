package guisquads

import (
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for artifact mode
const (
	ArtifactPanelSquadSelector framework.PanelType = "artifact_squad_selector"
	ArtifactPanelContent       framework.PanelType = "artifact_content"
)

func init() {
	// Register squad selector panel (prev/next + counter label)
	framework.RegisterPanel(ArtifactPanelSquadSelector, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			am := mode.(*ArtifactMode)
			layout := am.Layout

			selectorWidth := int(float64(layout.ScreenWidth) * 0.4)
			selectorHeight := int(float64(layout.ScreenHeight) * specs.CommanderSelectorHeight)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  selectorWidth,
				MinHeight: selectorHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(15),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
				),
			})

			topPad := int(float64(layout.ScreenHeight) * specs.PaddingExtraSmall)
			result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

			prevBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "< Prev Squad",
				OnClick: func() {
					am.cycleSquad(-1)
				},
			})
			result.Container.AddChild(prevBtn)

			counterLabel := builders.CreateSmallLabel("Squad 1 of 1")
			result.Container.AddChild(counterLabel)

			nextBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Next Squad >",
				OnClick: func() {
					am.cycleSquad(1)
				},
			})
			result.Container.AddChild(nextBtn)

			result.Custom["prevButton"] = prevBtn
			result.Custom["nextButton"] = nextBtn
			result.Custom["counterLabel"] = counterLabel

			return nil
		},
	})

	// Register main content panel with inventory + equipment tabs
	framework.RegisterPanel(ArtifactPanelContent, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			am := mode.(*ArtifactMode)
			layout := am.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.SquadEditorUnitPanelWidth)
			listHeight := int(float64(layout.ScreenHeight) * specs.SquadEditorUnitPanelHeight)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  listWidth,
				MinHeight: listHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				),
			})

			topOffset := int(float64(layout.ScreenHeight) * (specs.CommanderSelectorHeight + specs.PaddingStandard))
			result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(topOffset)

			// Tab row
			tabRow := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)
			tabRow.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Inventory (I)",
				OnClick: func() { am.switchTab("inventory") },
			}))
			tabRow.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Equipment (E)",
				OnClick: func() { am.switchTab("equipment") },
			}))
			result.Container.AddChild(tabRow)

			// === Inventory content sub-container ===
			inventoryContent := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)

			inventoryTitle := builders.CreateSmallLabel("Artifacts (0/20)")
			inventoryContent.AddChild(inventoryTitle)

			inventoryList := builders.CreateSimpleStringList(builders.SimpleStringListConfig{
				Entries:       []string{"No artifacts"},
				ScreenWidth:   400,
				ScreenHeight:  200,
				WidthPercent:  1.0,
				HeightPercent: 0.5,
			})
			inventoryContent.AddChild(inventoryList)

			inventoryDetail := builders.CreateTextAreaWithConfig(builders.TextAreaConfig{
				MinWidth:  380,
				MinHeight: 100,
				FontColor: color.White,
			})
			inventoryDetail.SetText("Select an artifact to view details.")
			inventoryContent.AddChild(inventoryDetail)

			inventoryButton := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Equip on Squad",
				OnClick: func() { am.onInventoryEquipAction() },
			})
			inventoryContent.AddChild(inventoryButton)

			result.Container.AddChild(inventoryContent)

			// === Equipment content sub-container (starts hidden) ===
			equipmentContent := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
				)),
			)

			equipmentTitle := builders.CreateSmallLabel("Equipment (0/3)")
			equipmentContent.AddChild(equipmentTitle)

			equipmentList := builders.CreateSimpleStringList(builders.SimpleStringListConfig{
				Entries:       []string{"No artifacts equipped"},
				ScreenWidth:   400,
				ScreenHeight:  200,
				WidthPercent:  1.0,
				HeightPercent: 0.5,
			})
			equipmentContent.AddChild(equipmentList)

			equipmentDetail := builders.CreateTextAreaWithConfig(builders.TextAreaConfig{
				MinWidth:  380,
				MinHeight: 100,
				FontColor: color.White,
			})
			equipmentDetail.SetText("Select an artifact to view details.")
			equipmentContent.AddChild(equipmentDetail)

			equipmentButton := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Unequip",
				OnClick: func() { am.onEquipmentAction() },
			})
			equipmentContent.AddChild(equipmentButton)

			equipmentContent.GetWidget().Visibility = widget.Visibility_Hide
			result.Container.AddChild(equipmentContent)

			// Store all references
			result.Custom["inventoryContent"] = inventoryContent
			result.Custom["inventoryList"] = inventoryList
			result.Custom["inventoryTitle"] = inventoryTitle
			result.Custom["inventoryDetail"] = inventoryDetail
			result.Custom["inventoryButton"] = inventoryButton
			result.Custom["equipmentContent"] = equipmentContent
			result.Custom["equipmentList"] = equipmentList
			result.Custom["equipmentTitle"] = equipmentTitle
			result.Custom["equipmentDetail"] = equipmentDetail
			result.Custom["equipmentButton"] = equipmentButton

			return nil
		},
	})
}
