package guimodes

import (
	"fmt"

	"game_main/graphics"
	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ExplorationMode is the default UI mode during dungeon exploration
type ExplorationMode struct {
	gui.BaseMode // Embed common mode infrastructure

	initialized bool

	// UI Components (ebitenui widgets)

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
	// Use ModeBuilder for declarative initialization (reduces 60+ lines to ~30)
	err := gui.NewModeBuilder(&em.BaseMode, gui.ModeConfig{
		ModeName:   "exploration",
		ReturnMode: "", // No return mode - exploration is the main mode

		// Register hotkeys for mode transitions (Battle Map context only)
		Hotkeys: []gui.HotkeySpec{
			{Key: ebiten.KeyI, TargetMode: "inventory"},
			{Key: ebiten.KeyC, TargetMode: "combat"},
			{Key: ebiten.KeyD, TargetMode: "squad_deployment"},
			// Note: 'E' key for squads requires context switch - handled in button
		},

		// Build panels
		Panels: []gui.PanelSpec{
			{
				// Message log panel (bottom-right) - now uses typed panel
				PanelType:  widgets.PanelTypeDetail,
				SpecName:   "message_log",
				DetailText: "",
			},
			{
				// Quick inventory panel (custom build)
				CustomBuild: em.buildQuickInventory,
			},
		},
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Get reference to message log TextArea from typed panel
	if w, ok := em.PanelWidgets["message_log"]; ok {
		if textArea, ok := w.(*widget.TextArea); ok {
			em.messageLog = textArea
		}
	}

	em.initialized = true
	return nil
}

func (em *ExplorationMode) buildQuickInventory() *widget.Container {
	// Use standard panel specification with custom runtime padding
	quickInventory := widgets.CreateStandardPanelWithOptions(
		em.PanelBuilders,
		"quick_inventory",
		widgets.CustomPadding(widget.Insets{
			Bottom: int(float64(em.Layout.ScreenHeight) * widgets.BottomButtonOffset),
		}),
	)

	// Throwables button
	throwableBtn := gui.ModeTransitionButton(em.ModeManager, "Throwables", "inventory")
	quickInventory.AddChild(throwableBtn)

	// Squads button (switches to Overworld context)
	squadsBtn := gui.ContextSwitchButton(em.Context.ModeCoordinator, "Squads (E)", "overworld", "squad_management")
	quickInventory.AddChild(squadsBtn)

	// Inventory button (Battle Map context)
	inventoryBtn := gui.ModeTransitionButton(em.ModeManager, "Inventory (I)", "inventory")
	quickInventory.AddChild(inventoryBtn)

	// Squad Deployment button (Battle Map context)
	deployBtn := gui.ModeTransitionButton(em.ModeManager, "Deploy (D)", "squad_deployment")
	quickInventory.AddChild(deployBtn)

	// Combat button (Battle Map context)
	combatBtn := gui.ModeTransitionButton(em.ModeManager, "Combat (C)", "combat")
	quickInventory.AddChild(combatBtn)

	// Store reference and return
	em.quickInventory = quickInventory
	return quickInventory
}

func (em *ExplorationMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Exploration Mode")

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
	// Handle common input first (ESC key, registered hotkeys like I/C/D)
	if em.HandleCommonInput(inputState) {
		return true
	}

	// Handle right-click to open info mode
	if inputState.MouseButton == ebiten.MouseButton2 && inputState.MousePressed {
		// Only open if not in other input modes
		if !inputState.PlayerInputStates.IsThrowing {
			// Convert mouse position to logical position (handles both scrolling modes)
			playerPos := *em.Context.PlayerData.Pos
			clickedPos := graphics.MouseToLogicalPosition(inputState.MouseX, inputState.MouseY, playerPos)

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

	return false
}
