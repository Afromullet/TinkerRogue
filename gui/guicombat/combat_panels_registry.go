package guicombat

import (
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"
	"game_main/gui/widgetresources"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for combat mode
const (
	CombatPanelTurnOrder   framework.PanelType = "combat_turn_order"
	CombatPanelFactionInfo framework.PanelType = "combat_faction_info"
	CombatPanelSquadDetail framework.PanelType = "combat_squad_detail"
	CombatPanelCombatLog   framework.PanelType = "combat_log"
	CombatPanelLayerStatus framework.PanelType = "combat_layer_status"
)

func init() {
	// Register all combat panels
	framework.RegisterPanel(CombatPanelTurnOrder, framework.PanelDescriptor{
		Content: framework.ContentText,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatTurnOrderWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatTurnOrderHeight)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(10),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
				),
			})

			topPad := int(float64(layout.ScreenHeight) * specs.PaddingTight)
			result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

			// Add label (stored in result for type-safe access)
			result.TextLabel = builders.CreateLargeLabel("Initializing combat...")
			result.Container.AddChild(result.TextLabel)

			return nil
		},
	})

	framework.RegisterPanel(CombatPanelFactionInfo, framework.PanelDescriptor{
		Content: framework.ContentText,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatFactionInfoWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatFactionInfoHeight)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
				),
			})

			leftPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			topPad := int(float64(layout.ScreenHeight) * specs.PaddingTight)
			result.Container.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topPad)

			result.TextLabel = builders.CreateSmallLabel("Faction Info")
			result.Container.AddChild(result.TextLabel)

			return nil
		},
	})

	framework.RegisterPanel(CombatPanelSquadDetail, framework.PanelDescriptor{
		Content: framework.ContentText,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatSquadDetailWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatSquadDetailHeight)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
					widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
				),
			})

			leftPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			topOffset := int(float64(layout.ScreenHeight) * (specs.CombatFactionInfoHeight + specs.PaddingTight))
			result.Container.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topOffset)

			result.TextLabel = builders.CreateSmallLabel("Select a squad\nto view details")
			result.Container.AddChild(result.TextLabel)

			return nil
		},
	})

	framework.RegisterPanel(CombatPanelCombatLog, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.CombatLogWidth)
			panelHeight := int(float64(layout.ScreenHeight) * specs.CombatLogHeight)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout:     widget.NewAnchorLayout(),
			})

			rightPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			bottomOffset := int(float64(layout.ScreenHeight) * (specs.CombatActionButtonHeight + specs.BottomButtonOffset + specs.PaddingTight))
			result.Container.GetWidget().LayoutData = builders.AnchorEndEnd(rightPad, bottomOffset)

			// Create cached textarea
			textArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
				MinWidth:  panelWidth - 20,
				MinHeight: panelHeight - 20,
				FontColor: color.White,
			})
			textArea.SetText("Combat started!\n")
			result.Container.AddChild(textArea)

			// Store in Custom map for retrieval
			result.Custom["textArea"] = textArea

			return nil
		},
	})

	framework.RegisterPanel(CombatPanelLayerStatus, framework.PanelDescriptor{
		Content: framework.ContentText,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			bm := mode.(*CombatMode)
			layout := bm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * 0.15)
			panelHeight := int(float64(layout.ScreenHeight) * 0.08)

			result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
				MinWidth:   panelWidth,
				MinHeight:  panelHeight,
				Background: widgetresources.PanelRes.Image,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(3),
					widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(5)),
				),
			})

			rightPad := int(float64(layout.ScreenWidth) * 0.01)
			topPad := int(float64(layout.ScreenHeight) * 0.01)
			result.Container.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topPad)

			result.TextLabel = builders.CreateSmallLabel("")
			result.Container.AddChild(result.TextLabel)

			// Hide initially
			result.Container.GetWidget().Visibility = widget.Visibility_Hide

			return nil
		},
	})
}

// GetCombatLogTextArea retrieves the combat log text area from panel registry
func GetCombatLogTextArea(panels *framework.PanelRegistry) *widgets.CachedTextAreaWrapper {
	if result := panels.Get(CombatPanelCombatLog); result != nil {
		if ta, ok := result.Custom["textArea"].(*widgets.CachedTextAreaWrapper); ok {
			return ta
		}
	}
	return nil
}
