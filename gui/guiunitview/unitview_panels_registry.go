package guiunitview

import (
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

const (
	UnitViewPanelDetail framework.PanelType = "unitview_detail"
)

func init() {
	framework.RegisterPanel(UnitViewPanelDetail, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			uvm := mode.(*UnitViewMode)
			layout := uvm.Layout

			panelWidth := int(float64(layout.ScreenWidth) * specs.PanelWidthWide)
			panelHeight := int(float64(layout.ScreenHeight) * specs.PanelHeightTall)

			result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
				MinWidth:  panelWidth,
				MinHeight: panelHeight,
				Layout: widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(10),
					widget.RowLayoutOpts.Padding(widget.Insets{Left: 15, Right: 15, Top: 15, Bottom: 15}),
				),
			})

			result.Container.GetWidget().LayoutData = widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}

			// Title
			titleLabel := builders.CreateLargeLabel("Unit Details")
			result.Container.AddChild(titleLabel)

			// Text area for unit info
			detailText := builders.CreateTextAreaWithConfig(builders.TextAreaConfig{
				MinWidth:  panelWidth - 40,
				MinHeight: panelHeight - 150,
				FontColor: color.White,
			})
			result.Container.AddChild(detailText)
			result.Custom["detailText"] = detailText

			// Debug: Add XP button
			result.Container.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text:    "Add 100 XP",
				OnClick: func() { uvm.onAddXP() },
			}))

			// Back button
			result.Container.AddChild(builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: "Back (ESC)",
				OnClick: func() {
					if returnMode, exists := uvm.ModeManager.GetMode(uvm.GetReturnMode()); exists {
						uvm.ModeManager.RequestTransition(returnMode, "Back button clicked")
					}
				},
			}))

			return nil
		},
	})
}
