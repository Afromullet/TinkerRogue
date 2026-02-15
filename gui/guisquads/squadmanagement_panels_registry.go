package guisquads

import (
	"fmt"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for squad management mode
const (
	SquadManagementPanelActionButtons framework.PanelType = "squadmanagement_action_buttons"
)

func init() {
	// Register action buttons panel
	framework.RegisterPanel(SquadManagementPanelActionButtons, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			smm := mode.(*SquadManagementMode)
			layout := smm.Layout

			// Calculate responsive spacing
			spacing := int(float64(layout.ScreenWidth) * specs.PaddingTight)

			// Create button group using builders.CreateButtonGroup with LayoutData
			bottomPad := int(float64(layout.ScreenHeight) * specs.BottomButtonOffset)
			anchorLayout := builders.AnchorCenterEnd(bottomPad)

			// Determine return behavior based on context:
			// In overworld context, returnMode is "overworld" → return to it
			// In tactical context, returnMode is "" → context switch to tactical
			returnText := "Exploration (ESC)"
			if smm.GetReturnMode() != "" {
				returnText = "Overworld (ESC)"
			}

			result.Container = builders.CreateButtonGroup(builders.ButtonGroupConfig{
				Buttons: []builders.ButtonSpec{
					{Text: returnText, OnClick: func() {
						if returnMode, exists := smm.ModeManager.GetMode(smm.GetReturnMode()); exists {
							smm.ModeManager.RequestTransition(returnMode, "Return from squad management")
							return
						}
						if smm.Context.ModeCoordinator != nil {
							if err := smm.Context.ModeCoordinator.EnterTactical("exploration"); err != nil {
								fmt.Printf("ERROR: Failed to enter tactical context: %v\n", err)
							}
						}
					}},
					{Text: "Buy Units (P)", OnClick: func() {
						if mode, exists := smm.ModeManager.GetMode("unit_purchase"); exists {
							smm.ModeManager.RequestTransition(mode, "Buy Units clicked")
						}
					}},
					{Text: "Squads (E)", OnClick: func() {
						if mode, exists := smm.ModeManager.GetMode("squad_editor"); exists {
							smm.ModeManager.RequestTransition(mode, "Squads clicked")
						}
					}},
				},
				Direction:  widget.DirectionHorizontal,
				Spacing:    spacing,
				Padding:    builders.NewResponsiveHorizontalPadding(layout, specs.PaddingExtraSmall),
				LayoutData: &anchorLayout,
			})

			return nil
		},
	})
}
