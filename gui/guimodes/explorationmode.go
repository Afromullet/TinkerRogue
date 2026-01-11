package guimodes

import (
	"fmt"

	"game_main/gui/framework"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ExplorationMode is the default UI mode during dungeon exploration
type ExplorationMode struct {
	framework.BaseMode // Embed common mode infrastructure

	initialized bool

	// Interactive widget references (stored here for refresh/access)
	// These are populated from panel registry after BuildPanels()
	messageLog     *widget.TextArea
	quickInventory *widget.Container
}

func NewExplorationMode(modeManager *framework.UIModeManager) *ExplorationMode {
	mode := &ExplorationMode{}
	mode.SetModeName("exploration")
	mode.SetReturnMode("") // No return mode - exploration is the main mode
	mode.ModeManager = modeManager
	mode.SetSelf(mode) // Required for panel registry building
	return mode
}

func (em *ExplorationMode) Initialize(ctx *framework.UIContext) error {
	// Build base UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&em.BaseMode, framework.ModeConfig{
		ModeName:   "exploration",
		ReturnMode: "", // No return mode - exploration is the main mode

		// Register hotkeys for mode transitions (Battle Map context only)
		Hotkeys: []framework.HotkeySpec{
			{Key: ebiten.KeyI, TargetMode: "inventory"},
			{Key: ebiten.KeyC, TargetMode: "combat"},
			{Key: ebiten.KeyD, TargetMode: "squad_deployment"},
			// Note: 'E' key for squads requires context switch - handled in button
		},
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Build panels from registry
	if err := em.BuildPanels(
		ExplorationPanelMessageLog,
		ExplorationPanelQuickInventory,
	); err != nil {
		return err
	}

	// Initialize widget references from registry
	em.initializeWidgetReferences()

	em.initialized = true
	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (em *ExplorationMode) initializeWidgetReferences() {
	em.messageLog = GetExplorationMessageLog(em.Panels)
	em.quickInventory = GetExplorationQuickInventory(em.Panels)
}

func (em *ExplorationMode) Enter(fromMode framework.UIMode) error {
	fmt.Println("Entering Exploration Mode")

	return nil
}

func (em *ExplorationMode) Exit(toMode framework.UIMode) error {
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

func (em *ExplorationMode) HandleInput(inputState *framework.InputState) bool {
	// Handle common input first (ESC key, registered hotkeys like I/C/D)
	if em.HandleCommonInput(inputState) {
		return true
	}

	return false
}
