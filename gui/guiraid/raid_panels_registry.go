package guiraid

import (
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/widgetresources"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for the raid mode.
const (
	RaidPanelFloorMap framework.PanelType = "raid_floor_map"
	RaidPanelDeploy   framework.PanelType = "raid_deploy"
	RaidPanelSummary  framework.PanelType = "raid_summary"
)

func init() {
	// Floor map panel — shows room DAG, alert level, room selection
	framework.RegisterPanel(RaidPanelFloorMap, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*RaidMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * 0.8)
			panelHeight := int(float64(layout.ScreenHeight) * 0.85)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(8),
					widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(10)),
				),
			})
			result.Container.GetWidget().LayoutData = widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}

			// Title label
			titleLabel := builders.CreateLargeLabel("Garrison Raid — Floor Map")
			result.Container.AddChild(titleLabel)

			// Room list label (updated dynamically)
			roomListLabel := builders.CreateSmallLabel("Loading rooms...")
			result.Container.AddChild(roomListLabel)

			// Alert level label
			alertLabel := builders.CreateSmallLabel("Alert: Unaware")
			result.Container.AddChild(alertLabel)

			// Store custom widgets for dynamic access
			result.Custom = map[string]interface{}{
				"titleLabel":    titleLabel,
				"roomListLabel": roomListLabel,
				"alertLabel":    alertLabel,
			}

			// Button row container
			buttonRow := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(10),
				)),
			)

			retreatBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Retreat",
				OnClick: func() {},
			})
			buttonRow.AddChild(retreatBtn)

			result.Container.AddChild(buttonRow)
			result.Custom["retreatBtn"] = retreatBtn

			return nil
		},
	})

	// Deploy panel — pre-encounter squad selection
	framework.RegisterPanel(RaidPanelDeploy, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*RaidMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * 0.7)
			panelHeight := int(float64(layout.ScreenHeight) * 0.75)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(8),
					widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(10)),
				),
			})
			result.Container.GetWidget().LayoutData = widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}
			result.Container.GetWidget().Visibility = widget.Visibility_Hide_Blocking

			titleLabel := builders.CreateLargeLabel("Deploy Squads")
			result.Container.AddChild(titleLabel)

			squadListLabel := builders.CreateSmallLabel("Select squads to deploy...")
			result.Container.AddChild(squadListLabel)

			// Button row
			buttonRow := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(10),
				)),
			)

			autoDeployBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Auto Deploy",
				OnClick: func() {},
			})
			buttonRow.AddChild(autoDeployBtn)

			startBattleBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Start Battle",
				OnClick: func() {},
			})
			buttonRow.AddChild(startBattleBtn)

			backBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Back",
				OnClick: func() {},
			})
			buttonRow.AddChild(backBtn)

			result.Container.AddChild(buttonRow)

			result.Custom = map[string]interface{}{
				"titleLabel":     titleLabel,
				"squadListLabel": squadListLabel,
				"autoDeployBtn":  autoDeployBtn,
				"startBattleBtn": startBattleBtn,
				"backBtn":        backBtn,
			}

			return nil
		},
	})

	// Summary panel — post-encounter results
	framework.RegisterPanel(RaidPanelSummary, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*RaidMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * 0.6)
			panelHeight := int(float64(layout.ScreenHeight) * 0.65)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(8),
					widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(10)),
				),
			})
			result.Container.GetWidget().LayoutData = widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}
			result.Container.GetWidget().Visibility = widget.Visibility_Hide_Blocking

			titleLabel := builders.CreateLargeLabel("Encounter Summary")
			result.Container.AddChild(titleLabel)

			summaryText := builders.CreateSmallLabel("No encounter data.")
			result.Container.AddChild(summaryText)

			continueBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Continue",
				OnClick: func() {},
			})
			result.Container.AddChild(continueBtn)

			result.Custom = map[string]interface{}{
				"titleLabel":  titleLabel,
				"summaryText": summaryText,
				"continueBtn": continueBtn,
			}

			return nil
		},
	})
}
