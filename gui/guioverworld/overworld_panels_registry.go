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
	OverworldPanelTickControls   framework.PanelType = "overworld_tick_controls"
	OverworldPanelThreatInfo     framework.PanelType = "overworld_threat_info"
	OverworldPanelTickStatus     framework.PanelType = "overworld_tick_status"
	OverworldPanelEventLog       framework.PanelType = "overworld_event_log"
	OverworldPanelThreatStats    framework.PanelType = "overworld_threat_stats"
	OverworldPanelDebugMenu      framework.PanelType = "overworld_debug_menu"
	OverworldPanelNodeMenu       framework.PanelType = "overworld_node_menu"
	OverworldPanelManagementMenu framework.PanelType = "overworld_management_menu"
	OverworldPanelResources      framework.PanelType = "overworld_resources"
)

// subMenuController manages sub-menu visibility. Only one sub-menu can be open at a time.
type subMenuController struct {
	menus  map[string]*widget.Container
	active string
}

func newSubMenuController() *subMenuController {
	return &subMenuController{
		menus: make(map[string]*widget.Container),
	}
}

func (sc *subMenuController) Register(name string, container *widget.Container) {
	sc.menus[name] = container
}

// Toggle returns a callback that toggles the named sub-menu.
// Opening one menu closes any other open menu.
func (sc *subMenuController) Toggle(name string) func() {
	return func() {
		if sc.active == name {
			sc.menus[name].GetWidget().Visibility = widget.Visibility_Hide
			sc.active = ""
			return
		}
		sc.CloseAll()
		if c, ok := sc.menus[name]; ok {
			c.GetWidget().Visibility = widget.Visibility_Show
			sc.active = name
		}
	}
}

func (sc *subMenuController) CloseAll() {
	for _, c := range sc.menus {
		c.GetWidget().Visibility = widget.Visibility_Hide
	}
	sc.active = ""
}

// createOverworldSubMenu creates a vertical sub-menu panel, registers it with the controller, and returns it.
func createOverworldSubMenu(om *OverworldMode, name string, buttons []builders.ButtonConfig) *widget.Container {
	spacing := int(float64(om.Layout.ScreenWidth) * specs.PaddingTight)
	subMenuBottomPad := int(float64(om.Layout.ScreenHeight) * (specs.BottomButtonOffset + 0.15))
	anchorLayout := builders.AnchorCenterEnd(subMenuBottomPad)

	panel := builders.CreatePanelWithConfig(builders.ContainerConfig{
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(spacing),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(10)),
		),
		LayoutData: anchorLayout,
	})
	for _, btn := range buttons {
		panel.AddChild(builders.CreateButtonWithConfig(btn))
	}
	panel.GetWidget().Visibility = widget.Visibility_Hide
	om.subMenus.Register(name, panel)
	return panel
}

// getOverworldTextArea retrieves a TextArea widget from the panel registry by panel type and custom key.
func getOverworldTextArea(panels *framework.PanelRegistry, panelType framework.PanelType, key string) *widget.TextArea {
	if result := panels.Get(panelType); result != nil {
		if textArea, ok := result.Custom[key].(*widget.TextArea); ok {
			return textArea
		}
	}
	return nil
}

func init() {
	// Register debug sub-menu (End Turn, Toggle Influence)
	framework.RegisterPanel(OverworldPanelDebugMenu, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			om := mode.(*OverworldMode)

			result.Container = createOverworldSubMenu(om, "debug", []builders.ButtonConfig{
				{Text: "End Turn (Space)", OnClick: func() {
					om.actionHandler.EndTurn()
					om.subMenus.CloseAll()
				}},
				{Text: "Toggle Influence (I)", OnClick: func() {
					om.actionHandler.ToggleInfluence()
					om.subMenus.CloseAll()
				}},
			})
			return nil
		},
	})

	// Register node sub-menu (Place Nodes, Garrison)
	framework.RegisterPanel(OverworldPanelNodeMenu, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			om := mode.(*OverworldMode)

			result.Container = createOverworldSubMenu(om, "node", []builders.ButtonConfig{
				{Text: "Place Nodes (N)", OnClick: func() {
					om.ModeManager.SetMode("node_placement")
					om.subMenus.CloseAll()
				}},
				{Text: "Garrison (G)", OnClick: func() {
					om.inputHandler.handleGarrison()
					om.subMenus.CloseAll()
				}},
			})
			return nil
		},
	})

	// Register management sub-menu (Squads, Inventory)
	framework.RegisterPanel(OverworldPanelManagementMenu, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			om := mode.(*OverworldMode)

			result.Container = createOverworldSubMenu(om, "management", []builders.ButtonConfig{
				{Text: "Squads", OnClick: func() {
					om.ModeManager.SetMode("squad_management")
					om.subMenus.CloseAll()
				}},
				{Text: "Inventory", OnClick: func() {
					om.ModeManager.SetMode("inventory")
					om.subMenus.CloseAll()
				}},
				{Text: "Recruit (R)", OnClick: func() {
					om.actionHandler.RecruitCommander()
					om.subMenus.CloseAll()
				}},
			})
			return nil
		},
	})

	// Register tick controls panel (main button bar)
	framework.RegisterPanel(OverworldPanelTickControls, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			om := mode.(*OverworldMode)
			layout := om.Layout

			// Calculate responsive spacing
			spacing := int(float64(layout.ScreenWidth) * specs.PaddingTight)
			bottomPad := int(float64(layout.ScreenHeight) * specs.BottomButtonOffset)
			anchorLayout := builders.AnchorCenterEnd(bottomPad)

			// Create button group with parent menu buttons + direct action buttons
			buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
				Buttons: []builders.ButtonSpec{
					{Text: "Debug", OnClick: om.subMenus.Toggle("debug")},
					{Text: "Node", OnClick: om.subMenus.Toggle("node")},
					{Text: "Management", OnClick: om.subMenus.Toggle("management")},
					{Text: "Move (M)", OnClick: func() {
						om.inputHandler.toggleMoveMode()
					}},
					{Text: "Engage (E)", OnClick: func() {
						om.actionHandler.EngageThreat(om.state.SelectedNodeID)
					}},
					{Text: "End Turn (Space)", OnClick: func() {
						om.actionHandler.EndTurn()
					}},
					{Text: "Return (ESC)", OnClick: func() {
						if om.Context.ModeCoordinator != nil {
							if err := om.Context.ModeCoordinator.EnterTactical("exploration"); err != nil {
								fmt.Printf("ERROR: Failed to return to tactical context: %v\n", err)
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

	// Register player resources panel (top-left, shows Gold/Iron/Wood/Stone)
	framework.RegisterPanel(OverworldPanelResources, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			om := mode.(*OverworldMode)

			typedResult := om.PanelBuilders.BuildTypedPanel(builders.TypedPanelConfig{
				Type:       builders.PanelTypeDetail,
				SpecName:   "player_resources",
				DetailText: "Resources: ---",
			})

			result.Container = typedResult.Panel
			result.Custom["resourcesText"] = typedResult.TextArea

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
	return getOverworldTextArea(panels, OverworldPanelThreatInfo, "threatInfoText")
}

func GetOverworldTickStatus(panels *framework.PanelRegistry) *widget.TextArea {
	return getOverworldTextArea(panels, OverworldPanelTickStatus, "tickStatusText")
}

func GetOverworldEventLog(panels *framework.PanelRegistry) *widget.TextArea {
	return getOverworldTextArea(panels, OverworldPanelEventLog, "eventLogText")
}

func GetOverworldThreatStats(panels *framework.PanelRegistry) *widget.TextArea {
	return getOverworldTextArea(panels, OverworldPanelThreatStats, "threatStatsText")
}

func GetOverworldResources(panels *framework.PanelRegistry) *widget.TextArea {
	return getOverworldTextArea(panels, OverworldPanelResources, "resourcesText")
}

func GetOverworldTickControls(panels *framework.PanelRegistry) *widget.Container {
	if result := panels.Get(OverworldPanelTickControls); result != nil {
		if container, ok := result.Custom["tickControls"].(*widget.Container); ok {
			return container
		}
	}
	return nil
}
