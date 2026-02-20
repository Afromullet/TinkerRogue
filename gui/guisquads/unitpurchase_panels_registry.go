package guisquads

import (
	"fmt"
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"
	"game_main/gui/widgets"
	"game_main/tactical/squads"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for unit purchase mode
const (
	UnitPurchasePanelResourceDisplay framework.PanelType = "unitpurchase_resource_display"
	UnitPurchasePanelUnitList        framework.PanelType = "unitpurchase_unit_list"
	UnitPurchasePanelDetailPanel     framework.PanelType = "unitpurchase_detail_panel"
	UnitPurchasePanelActionButtons   framework.PanelType = "unitpurchase_action_buttons"
)

func init() {
	// Register resource display panel
	framework.RegisterPanel(UnitPurchasePanelResourceDisplay, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			upm := mode.(*UnitPurchaseMode)
			layout := upm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * 0.25)
			panelHeight := int(float64(layout.ScreenHeight) * 0.08)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  panelWidth,
				MinHeight: panelHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingTight)),
				),
			})

			topPad := int(float64(layout.ScreenHeight) * 0.02)
			result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

			// Gold label
			goldLabel := builders.CreateSmallLabel("Gold: 0")
			result.Container.AddChild(goldLabel)

			// Roster label
			rosterLabel := builders.CreateSmallLabel("Roster: 0/0")
			result.Container.AddChild(rosterLabel)

			result.Custom["goldLabel"] = goldLabel
			result.Custom["rosterLabel"] = rosterLabel

			return nil
		},
	})

	// Register unit list panel
	framework.RegisterPanel(UnitPurchasePanelUnitList, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			upm := mode.(*UnitPurchaseMode)
			layout := upm.Layout

			listWidth := int(float64(layout.ScreenWidth) * specs.UnitPurchaseListWidth)
			listHeight := int(float64(layout.ScreenHeight) * specs.UnitPurchaseListHeight)

			baseList := builders.CreateListWithConfig(builders.ListConfig{
				Entries:   []interface{}{}, // Will be populated in Enter
				MinWidth:  listWidth,
				MinHeight: listHeight,
				EntryLabelFunc: func(e interface{}) string {
					if template, ok := e.(*squads.UnitTemplate); ok {
						totalOwned, available := upm.purchaseService.GetUnitOwnedCount(
							upm.Context.PlayerData.PlayerEntityID,
							template.UnitType,
						)
						if totalOwned > 0 {
							return fmt.Sprintf("%s (Owned: %d, Available: %d)", template.UnitType, totalOwned, available)
						}
						return fmt.Sprintf("%s (Owned: 0)", template.UnitType)
					}
					return fmt.Sprintf("%v", e)
				},
				OnEntrySelected: func(selectedEntry interface{}) {
					if template, ok := selectedEntry.(*squads.UnitTemplate); ok {
						upm.selectedTemplate = template
						upm.updateDetailPanel()
					}
				},
			})

			// Wrap with caching for performance
			unitList := widgets.NewCachedListWrapper(baseList)

			// Position below resource panel
			leftPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			topOffset := int(float64(layout.ScreenHeight) * (specs.UnitPurchaseResourceHeight + specs.PaddingStandard*2))

			result.Container = widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
				widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(builders.AnchorStartStart(leftPad, topOffset))),
			)
			result.Container.AddChild(baseList)

			result.Custom["unitList"] = unitList
			result.Custom["baseList"] = baseList

			return nil
		},
	})

	// Register detail panel
	framework.RegisterPanel(UnitPurchasePanelDetailPanel, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			upm := mode.(*UnitPurchaseMode)
			layout := upm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * 0.35)
			panelHeight := int(float64(layout.ScreenHeight) * 0.6)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  panelWidth,
				MinHeight: panelHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(10),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingTight)),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
			result.Container.GetWidget().LayoutData = builders.AnchorEndCenter(rightPad)

			// Basic info text area (cached)
			detailTextArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
				MinWidth:  panelWidth - 30,
				MinHeight: 100,
				FontColor: color.White,
			})
			detailTextArea.SetText("Select a unit to view details")
			result.Container.AddChild(detailTextArea)

			// View Stats button
			viewStatsButton := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "View Stats",
				OnClick: func() {
					upm.showStats()
				},
			})
			viewStatsButton.GetWidget().Disabled = true
			result.Container.AddChild(viewStatsButton)

			// Stats text area (hidden by default, cached)
			statsTextArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
				MinWidth:  panelWidth - 30,
				MinHeight: 300,
				FontColor: color.White,
			})
			statsTextArea.GetWidget().Visibility = widget.Visibility_Hide
			result.Container.AddChild(statsTextArea)

			result.Custom["detailTextArea"] = detailTextArea
			result.Custom["viewStatsButton"] = viewStatsButton
			result.Custom["statsTextArea"] = statsTextArea

			return nil
		},
	})

	// Register action buttons panel
	framework.RegisterPanel(UnitPurchasePanelActionButtons, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			upm := mode.(*UnitPurchaseMode)

			result.Container = builders.CreateBottomActionBar(upm.Layout, []builders.ButtonSpec{
				{Text: "Buy Unit", OnClick: func() { upm.purchaseUnit() }},
				{Text: "Undo (Ctrl+Z)", OnClick: func() { upm.CommandHistory.Undo() }},
				{Text: "Redo (Ctrl+Y)", OnClick: func() { upm.CommandHistory.Redo() }},
				{Text: "Back (ESC)", OnClick: func() {
					if mode, exists := upm.ModeManager.GetMode("squad_editor"); exists {
						upm.ModeManager.RequestTransition(mode, "Back button pressed")
					}
				}},
			})

			// Store buy button reference for enable/disable control
			if children := result.Container.Children(); len(children) > 0 {
				if btn, ok := children[0].(*widget.Button); ok {
					result.Custom["buyButton"] = btn
					btn.GetWidget().Disabled = true
				}
			}

			return nil
		},
	})
}

