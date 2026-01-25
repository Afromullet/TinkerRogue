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
	ExplorationPanelQuickInventory framework.PanelType = "exploration_quick_inventory"
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

	// Register quick inventory (action buttons) panel
	framework.RegisterPanel(ExplorationPanelQuickInventory, framework.PanelDescriptor{
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
					{Text: "Throwables", OnClick: func() {
						if mode, exists := em.ModeManager.GetMode("inventory"); exists {
							em.ModeManager.RequestTransition(mode, "Throwables clicked")
						}
					}},
					{Text: "Overworld (O)", OnClick: func() {
						if em.Context.ModeCoordinator != nil {
							if err := em.Context.ModeCoordinator.ReturnToOverworld("overworld"); err != nil {
								fmt.Printf("ERROR: Failed to switch to overworld: %v\n", err)
							}
						}
					}},
					{Text: "Squads (E)", OnClick: func() {
						if em.Context.ModeCoordinator != nil {
							if err := em.Context.ModeCoordinator.ReturnToOverworld("squad_management"); err != nil {
								fmt.Printf("ERROR: Failed to return to overworld: %v\n", err)
							}
						}
					}},
					{Text: "Inventory (I)", OnClick: func() {
						if mode, exists := em.ModeManager.GetMode("inventory"); exists {
							em.ModeManager.RequestTransition(mode, "Inventory clicked")
						}
					}},
					{Text: "Deploy (D)", OnClick: func() {
						if mode, exists := em.ModeManager.GetMode("squad_deployment"); exists {
							em.ModeManager.RequestTransition(mode, "Deploy clicked")
						}
					}},
					{Text: "Combat (C)", OnClick: func() {
						if mode, exists := em.ModeManager.GetMode("combat"); exists {
							em.ModeManager.RequestTransition(mode, "Combat clicked")
						}
					}},
				},
				Direction:  widget.DirectionHorizontal,
				Spacing:    spacing,
				Padding:    builders.NewResponsiveHorizontalPadding(layout, specs.PaddingExtraSmall),
				LayoutData: &anchorLayout,
			})

			result.Container = buttonContainer
			result.Custom["quickInventory"] = buttonContainer

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

func GetExplorationQuickInventory(panels *framework.PanelRegistry) *widget.Container {
	if result := panels.Get(ExplorationPanelQuickInventory); result != nil {
		if container, ok := result.Custom["quickInventory"].(*widget.Container); ok {
			return container
		}
	}
	return nil
}
