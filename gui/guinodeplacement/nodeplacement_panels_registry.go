package guinodeplacement

import (
	"fmt"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for node placement mode
const (
	NodePlacementPanelNodeList framework.PanelType = "node_placement_node_list"
	NodePlacementPanelInfo     framework.PanelType = "node_placement_info"
	NodePlacementPanelControls framework.PanelType = "node_placement_controls"
)

func init() {
	// Register node type list panel (shows available node types)
	framework.RegisterPanel(NodePlacementPanelNodeList, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			npm := mode.(*NodePlacementMode)

			typedResult := npm.PanelBuilders.BuildTypedPanel(builders.TypedPanelConfig{
				Type:       builders.PanelTypeDetail,
				SpecName:   "node_list",
				DetailText: "Loading node types...",
			})

			result.Container = typedResult.Panel
			result.Custom["nodeListText"] = typedResult.TextArea

			return nil
		},
	})

	// Register placement info panel (shows selected type details + feedback)
	framework.RegisterPanel(NodePlacementPanelInfo, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			npm := mode.(*NodePlacementMode)

			typedResult := npm.PanelBuilders.BuildTypedPanel(builders.TypedPanelConfig{
				Type:       builders.PanelTypeDetail,
				SpecName:   "placement_info",
				DetailText: "Select a node type to place",
			})

			result.Container = typedResult.Panel
			result.Custom["placementInfoText"] = typedResult.TextArea

			return nil
		},
	})

	// Register controls panel (Place + Cancel buttons)
	framework.RegisterPanel(NodePlacementPanelControls, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			npm := mode.(*NodePlacementMode)
			layout := npm.Layout

			spacing := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			bottomPad := int(float64(layout.ScreenHeight) * specs.BottomButtonOffset)
			anchorLayout := builders.AnchorCenterEnd(bottomPad)

			buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
				Buttons: []builders.ButtonSpec{
					{Text: "Cancel (ESC)", OnClick: func() {
						if returnMode, exists := npm.ModeManager.GetMode("overworld"); exists {
							npm.ModeManager.RequestTransition(returnMode, "Cancel placement")
						}
					}},
				},
				Direction:  widget.DirectionHorizontal,
				Spacing:    spacing,
				Padding:    builders.NewResponsiveHorizontalPadding(layout, specs.PaddingExtraSmall),
				LayoutData: &anchorLayout,
			})

			result.Container = buttonContainer
			result.Custom["controls"] = buttonContainer

			return nil
		},
	})
}

// Helper functions to retrieve widgets from panel registry

func GetNodeListText(panels *framework.PanelRegistry) *widget.TextArea {
	if result := panels.Get(NodePlacementPanelNodeList); result != nil {
		if textArea, ok := result.Custom["nodeListText"].(*widget.TextArea); ok {
			return textArea
		}
	}
	fmt.Println("WARNING: Could not retrieve node list text area")
	return nil
}

func GetPlacementInfoText(panels *framework.PanelRegistry) *widget.TextArea {
	if result := panels.Get(NodePlacementPanelInfo); result != nil {
		if textArea, ok := result.Custom["placementInfoText"].(*widget.TextArea); ok {
			return textArea
		}
	}
	fmt.Println("WARNING: Could not retrieve placement info text area")
	return nil
}
