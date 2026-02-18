package guiexploration

import (
	"fmt"

	"game_main/common"
	"game_main/config"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"
	"game_main/overworld/core"
	"game_main/world/worldmap"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for exploration mode
const (
	ExplorationPanelMessageLog    framework.PanelType = "exploration_message_log"
	ExplorationPanelActionButtons framework.PanelType = "exploration_action_buttons"
	ExplorationPanelDebugMenu     framework.PanelType = "exploration_debug_menu"
)

// createExplorationSubMenu creates a vertical sub-menu panel, registers it with the controller, and returns it.
func createExplorationSubMenu(em *ExplorationMode, name string, buttons []builders.ButtonConfig) *widget.Container {
	spacing := int(float64(em.Layout.ScreenWidth) * specs.PaddingTight)
	subMenuBottomPad := int(float64(em.Layout.ScreenHeight) * (specs.BottomButtonOffset + 0.15))
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
	em.subMenus.Register(name, panel)
	return panel
}

// regenerateMap generates a new map using the named generator and repositions the player.
func regenerateMap(em *ExplorationMode, generatorName string) {
	// 1. Generate new map (replaces tile slice, rooms, valid positions, etc.)
	newMap := worldmap.NewGameMap(generatorName)
	*em.Context.GameMap = newMap

	// 2. Rebuild walkable grid
	core.InitWalkableGrid(config.DefaultMapWidth, config.DefaultMapHeight)
	for _, pos := range em.Context.GameMap.ValidPositions {
		core.SetTileWalkable(pos, true)
	}

	// 3. Reposition player
	startPos := em.Context.GameMap.StartingPosition()
	oldPos := *em.Context.PlayerData.Pos
	em.Context.PlayerData.Pos.X = startPos.X
	em.Context.PlayerData.Pos.Y = startPos.Y

	// 4. Update spatial index
	common.GlobalPositionSystem.MoveEntity(em.Context.PlayerData.PlayerEntityID, oldPos, startPos)

	// 5. Force tile re-render
	em.Context.GameMap.TileColorsDirty = true

	fmt.Printf("DEBUG: Regenerated map with generator '%s'\n", generatorName)
}

func init() {
	// Register debug sub-menu (map generator options)
	framework.RegisterPanel(ExplorationPanelDebugMenu, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
			em := mode.(*ExplorationMode)

			result.Container = createExplorationSubMenu(em, "debug", []builders.ButtonConfig{
				{Text: "Cavern", OnClick: func() {
					regenerateMap(em, "cavern")
					em.subMenus.CloseAll()
				}},
				{Text: "Rooms & Corridors", OnClick: func() {
					regenerateMap(em, "rooms_corridors")
					em.subMenus.CloseAll()
				}},
				{Text: "Tactical Biome", OnClick: func() {
					regenerateMap(em, "tactical_biome")
					em.subMenus.CloseAll()
				}},
				{Text: "Overworld", OnClick: func() {
					regenerateMap(em, "overworld")
					em.subMenus.CloseAll()
				}},
			})
			return nil
		},
	})

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

			// Detect whether squad_editor is in the current (tactical) mode manager.
			// When it is (roguelike mode), only show "Squad" and route within tactical context.
			// When it isn't (overworld mode), show both buttons as today.
			_, hasSquadInTactical := em.ModeManager.GetMode("squad_editor")

			buttons := []builders.ButtonSpec{}

			// Only show Overworld button when squad_editor is NOT in tactical (i.e., overworld mode)
			if !hasSquadInTactical {
				buttons = append(buttons, builders.ButtonSpec{
					Text: "Overworld (O)", OnClick: func() {
						if em.Context.ModeCoordinator != nil {
							if err := em.Context.ModeCoordinator.ReturnToOverworld("overworld"); err != nil {
								fmt.Printf("ERROR: Failed to switch to overworld: %v\n", err)
							}
						}
					},
				})
			}

			// Squad button: route based on whether squad_editor is a local tactical mode
			buttons = append(buttons, builders.ButtonSpec{
				Text: "Squad", OnClick: func() {
					if targetMode, exists := em.ModeManager.GetMode("squad_editor"); exists {
						em.ModeManager.RequestTransition(targetMode, "Squad button pressed")
					} else if em.Context.ModeCoordinator != nil {
						em.Context.ModeCoordinator.ReturnToOverworld("squad_editor")
					}
				},
			})

			// Debug button: only show in roguelike mode (when squad_editor is in tactical context)
			if hasSquadInTactical {
				buttons = append(buttons, builders.ButtonSpec{
					Text: "Debug", OnClick: em.subMenus.Toggle("debug"),
				})
			}

			buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
				Buttons:    buttons,
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
