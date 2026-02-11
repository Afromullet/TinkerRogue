package guioverworld

import (
	"fmt"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for overworld mode
const (
	OverworldPanelTickControls framework.PanelType = "overworld_tick_controls"
	OverworldPanelThreatInfo   framework.PanelType = "overworld_threat_info"
	OverworldPanelTickStatus   framework.PanelType = "overworld_tick_status"
	OverworldPanelEventLog     framework.PanelType = "overworld_event_log"
	OverworldPanelThreatStats  framework.PanelType = "overworld_threat_stats"
)

func init() {
	// Register tick controls panel (manual tick button + pause/resume)
	framework.RegisterPanel(OverworldPanelTickControls, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			om := mode.(*OverworldMode)
			layout := om.Layout

			// Calculate responsive spacing
			spacing := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			bottomPad := int(float64(layout.ScreenHeight) * specs.BottomButtonOffset)
			anchorLayout := builders.AnchorCenterEnd(bottomPad)

			// Create button group
			buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
				Buttons: []builders.ButtonSpec{
					{Text: "Advance Tick (Space)", OnClick: func() {
						om.actionHandler.AdvanceTick()
					}},
					{Text: "Auto-Travel (A)", OnClick: func() {
						om.actionHandler.ToggleAutoTravel()
					}},
					{Text: "Toggle Influence (I)", OnClick: func() {
						om.actionHandler.ToggleInfluence()
					}},
					{Text: "Engage Threat (E)", OnClick: func() {
						om.actionHandler.EngageThreat(om.state.SelectedNodeID)
					}},
					{Text: "Garrison (G)", OnClick: func() {
						om.inputHandler.handleGarrison()
					}},
					{Text: "Place Nodes (N)", OnClick: func() {
						om.ModeManager.SetMode("node_placement")
					}},
					{Text: "Return (ESC)", OnClick: func() {
						if om.Context.ModeCoordinator != nil {
							if err := om.Context.ModeCoordinator.EnterBattleMap("exploration"); err != nil {
								fmt.Printf("ERROR: Failed to return to battle map: %v\n", err)
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
			result.Custom["tickControls"] = buttonContainer

			return nil
		},
	})

	// Register threat info panel (shows selected threat details)
	framework.RegisterPanel(OverworldPanelThreatInfo, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			om := mode.(*OverworldMode)

			typedResult := om.PanelBuilders.BuildTypedPanel(builders.TypedPanelConfig{
				Type:       builders.PanelTypeDetail,
				SpecName:   "threat_info",
				DetailText: "Select a threat to view details",
			})

			result.Container = typedResult.Panel
			result.Custom["threatInfoText"] = typedResult.TextArea

			return nil
		},
	})

	// Register tick status panel (current tick, tick rate, pause state)
	framework.RegisterPanel(OverworldPanelTickStatus, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			om := mode.(*OverworldMode)

			typedResult := om.PanelBuilders.BuildTypedPanel(builders.TypedPanelConfig{
				Type:       builders.PanelTypeDetail,
				SpecName:   "tick_status",
				DetailText: "Tick: 0 | Status: Paused",
			})

			result.Container = typedResult.Panel
			result.Custom["tickStatusText"] = typedResult.TextArea

			return nil
		},
	})

	// Register event log panel (tick events, threat evolution, etc.)
	framework.RegisterPanel(OverworldPanelEventLog, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			om := mode.(*OverworldMode)

			typedResult := om.PanelBuilders.BuildTypedPanel(builders.TypedPanelConfig{
				Type:       builders.PanelTypeDetail,
				SpecName:   "event_log",
				DetailText: "Overworld initialized",
			})

			result.Container = typedResult.Panel
			result.Custom["eventLogText"] = typedResult.TextArea

			return nil
		},
	})

	// Register threat statistics panel (total threats, average intensity)
	framework.RegisterPanel(OverworldPanelThreatStats, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			om := mode.(*OverworldMode)

			typedResult := om.PanelBuilders.BuildTypedPanel(builders.TypedPanelConfig{
				Type:       builders.PanelTypeDetail,
				SpecName:   "threat_stats",
				DetailText: "Threats: 0 | Avg Intensity: 0.0",
			})

			result.Container = typedResult.Panel
			result.Custom["threatStatsText"] = typedResult.TextArea

			return nil
		},
	})
}

// Helper functions to retrieve widgets from panel registry

func GetOverworldThreatInfo(panels *framework.PanelRegistry) *widget.TextArea {
	if result := panels.Get(OverworldPanelThreatInfo); result != nil {
		if textArea, ok := result.Custom["threatInfoText"].(*widget.TextArea); ok {
			return textArea
		}
	}
	return nil
}

func GetOverworldTickStatus(panels *framework.PanelRegistry) *widget.TextArea {
	if result := panels.Get(OverworldPanelTickStatus); result != nil {
		if textArea, ok := result.Custom["tickStatusText"].(*widget.TextArea); ok {
			return textArea
		}
	}
	return nil
}

func GetOverworldEventLog(panels *framework.PanelRegistry) *widget.TextArea {
	if result := panels.Get(OverworldPanelEventLog); result != nil {
		if textArea, ok := result.Custom["eventLogText"].(*widget.TextArea); ok {
			return textArea
		}
	}
	return nil
}

func GetOverworldThreatStats(panels *framework.PanelRegistry) *widget.TextArea {
	if result := panels.Get(OverworldPanelThreatStats); result != nil {
		if textArea, ok := result.Custom["threatStatsText"].(*widget.TextArea); ok {
			return textArea
		}
	}
	return nil
}

func GetOverworldTickControls(panels *framework.PanelRegistry) *widget.Container {
	if result := panels.Get(OverworldPanelTickControls); result != nil {
		if container, ok := result.Custom["tickControls"].(*widget.Container); ok {
			return container
		}
	}
	fmt.Println("WARNING: Could not retrieve tick controls container")
	return nil
}
