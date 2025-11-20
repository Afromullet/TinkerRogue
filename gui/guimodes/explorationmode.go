package guimodes

import (
	"fmt"
	"game_main/coords"
	"game_main/graphics"
	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/guicomponents"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ExplorationMode is the default UI mode during dungeon exploration
type ExplorationMode struct {
	gui.BaseMode // Embed common mode infrastructure

	initialized bool

	// UI Components (ebitenui widgets)
	statsPanel     *widget.Container
	statsTextArea  *widget.TextArea
	statsComponent *guicomponents.StatsDisplayComponent
	messageLog     *widget.TextArea
	quickInventory *widget.Container
}

func NewExplorationMode(modeManager *core.UIModeManager) *ExplorationMode {
	mode := &ExplorationMode{}
	mode.SetModeName("exploration")
	mode.ModeManager = modeManager
	return mode
}

func (em *ExplorationMode) Initialize(ctx *core.UIContext) error {
	// Initialize common mode infrastructure
	em.InitializeBase(ctx)

	// Register hotkeys for mode transitions
	em.RegisterHotkey(ebiten.KeyE, "squad_management")
	em.RegisterHotkey(ebiten.KeyI, "inventory")
	em.RegisterHotkey(ebiten.KeyB, "squad_builder")
	em.RegisterHotkey(ebiten.KeyC, "combat")

	// Build stats panel (top-right) using standard specification
	em.statsPanel, em.statsTextArea = gui.CreateStandardDetailPanel(
		em.PanelBuilders,
		em.Layout,
		"stats_panel",
		em.Context.PlayerData.PlayerAttributes(em.Context.ECSManager).DisplayString(),
	)
	em.RootContainer.AddChild(em.statsPanel)

	// Create stats display component to manage refresh logic
	em.statsComponent = guicomponents.NewStatsDisplayComponent(em.statsTextArea, nil)
	// Use default formatter which displays player attributes

	// Build message log (bottom-right) using standard specification
	logContainer, messageLog := gui.CreateStandardDetailPanel(
		em.PanelBuilders,
		em.Layout,
		"message_log",
		"",
	)
	em.messageLog = messageLog
	em.RootContainer.AddChild(logContainer)

	// Build exploration-specific UI layout
	em.buildQuickInventory()

	em.initialized = true
	return nil
}

func (em *ExplorationMode) buildQuickInventory() {
	// Use standard panel specification with custom runtime padding
	em.quickInventory = widgets.CreateStandardPanelWithOptions(
		em.PanelBuilders,
		"quick_inventory",
		widgets.CustomPadding(widget.Insets{
			Bottom: int(float64(em.Layout.ScreenHeight) * widgets.BottomButtonOffset),
		}),
	)

	// Throwables button
	throwableBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Throwables",
		OnClick: func() {
			// Transition to inventory mode
			if invMode, exists := em.ModeManager.GetMode("inventory"); exists {
				em.ModeManager.RequestTransition(invMode, "Open Throwables")
			}
		},
	})
	em.quickInventory.AddChild(throwableBtn)

	// Inventory button
	inventoryBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Inventory (I)",
		OnClick: func() {
			if invMode, exists := em.ModeManager.GetMode("inventory"); exists {
				em.ModeManager.RequestTransition(invMode, "Open Inventory")
			}
		},
	})
	em.quickInventory.AddChild(inventoryBtn)

	// Squad button
	squadBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Squads (E)",
		OnClick: func() {
			if squadMode, exists := em.ModeManager.GetMode("squad_management"); exists {
				em.ModeManager.RequestTransition(squadMode, "Open Squad Management")
			}
		},
	})
	em.quickInventory.AddChild(squadBtn)

	// Squad Builder button
	builderBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Builder (B)",
		OnClick: func() {
			if builderMode, exists := em.ModeManager.GetMode("squad_builder"); exists {
				em.ModeManager.RequestTransition(builderMode, "Open Squad Builder")
			}
		},
	})
	em.quickInventory.AddChild(builderBtn)

	// Squad Deployment button
	deployBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Deploy (D)",
		OnClick: func() {
			if deployMode, exists := em.ModeManager.GetMode("squad_deployment"); exists {
				em.ModeManager.RequestTransition(deployMode, "Open Squad Deployment")
			}
		},
	})
	em.quickInventory.AddChild(deployBtn)

	// Combat button
	combatBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Combat (C)",
		OnClick: func() {
			if combatMode, exists := em.ModeManager.GetMode("combat"); exists {
				em.ModeManager.RequestTransition(combatMode, "Enter Combat")
			}
		},
	})
	em.quickInventory.AddChild(combatBtn)

	em.RootContainer.AddChild(em.quickInventory)
}

func (em *ExplorationMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Exploration Mode")

	// Refresh player stats using component
	if em.statsComponent != nil && em.Context.PlayerData != nil {
		em.statsComponent.RefreshStats(em.Context.PlayerData, em.Context.ECSManager)
	}

	return nil
}

func (em *ExplorationMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Exploration Mode")
	return nil
}

func (em *ExplorationMode) Update(deltaTime float64) error {
	// Update message log if new messages
	// Update stats if player data changed
	// (Minimal updates - most updates happen in Enter/Exit)
	return nil
}

func (em *ExplorationMode) Render(screen *ebiten.Image) {
	// No custom rendering needed - ebitenui handles everything
	// Could add overlays here (threat ranges, movement paths, etc.)
}

func (em *ExplorationMode) HandleInput(inputState *core.InputState) bool {
	// Handle right-click to open info mode
	if inputState.MouseButton == ebiten.MouseButton2 && inputState.MousePressed {
		// Only open if not in other input modes
		if !inputState.PlayerInputStates.IsThrowing {
			// Convert mouse position to logical position
			playerPos := *em.Context.PlayerData.Pos
			manager := coords.NewCoordinateManager(graphics.ScreenInfo)
			viewport := coords.NewViewport(manager, playerPos)
			clickedPos := viewport.ScreenToLogical(inputState.MouseX, inputState.MouseY)

			// Transition to info mode with position
			if infoMode, exists := em.ModeManager.GetMode("info_inspect"); exists {
				if infoModeTyped, ok := infoMode.(*InfoMode); ok {
					infoModeTyped.SetInspectPosition(clickedPos)
					em.ModeManager.RequestTransition(infoMode, "Right-click inspection")
				}
			}
			return true
		}
	}

	// Mode transition hotkeys are now handled by gui.BaseMode.HandleCommonInput via RegisterHotkey
	return false
}
