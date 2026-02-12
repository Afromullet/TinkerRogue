package guiexploration

import (
	"fmt"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for exploration mode
const (
	ExplorationPanelMessageLog     framework.PanelType = "exploration_message_log"
	ExplorationPanelActionButtons framework.PanelType = "exploration_action_buttons"
)

func init() {
	// Register message log panel
	framework.RegisterPanel(ExplorationPanelMessageLog, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			em := mode.(*ExplorationMode)

			// Use the typed panel builder for detail panels
			typedResult := em.PanelBuilders.BuildTypedPanel(builders.TypedPanelConfig{
				Type:       builders.PanelTypeDetail,
				SpecName:   "message_log",
				DetailText: "",
			})

			result.Container = typedResult.Panel
			result.Custom["messageLog"] = typedResult.TextArea

			return nil
		},
	})

	// Register action buttons panel
	framework.RegisterPanel(ExplorationPanelActionButtons, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			em := mode.(*ExplorationMode)
			layout := em.Layout

			// Calculate responsive spacing
			spacing := int(float64(layout.ScreenWidth) * specs.PaddingTight)

			// Create button group using builders.CreateButtonGroup with LayoutData
			bottomPad := int(float64(layout.ScreenHeight) * specs.BottomButtonOffset)
			anchorLayout := builders.AnchorCenterEnd(bottomPad)

			buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
				Buttons: []builders.ButtonSpec{
					{Text: "Overworld (O)", OnClick: func() {
						if em.Context.ModeCoordinator != nil {
							if err := em.Context.ModeCoordinator.ReturnToOverworld("overworld"); err != nil {
								fmt.Printf("ERROR: Failed to switch to overworld: %v\n", err)
							}
						}
					}},
				},
				Direction:  widget.DirectionHorizontal,
				Spacing:    spacing,
				Padding:    builders.NewResponsiveHorizontalPadding(layout, specs.PaddingExtraSmall),
				LayoutData: &anchorLayout,
			})

			result.Container = buttonContainer

			return nil
		},
	})
}

// Helper functions to retrieve widgets from panel registry

func GetExplorationMessageLog(panels *framework.PanelRegistry) *widget.TextArea {
	if result := panels.Get(ExplorationPanelMessageLog); result != nil {
		if textArea, ok := result.Custom["messageLog"].(*widget.TextArea); ok {
			return textArea
		}
	}
	return nil
}

