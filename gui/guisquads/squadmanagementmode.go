package guisquads

import (
	"fmt"
	"game_main/gui"
	"game_main/gui/core"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadManagementMode shows one squad at a time with navigation controls
type SquadManagementMode struct {
	gui.BaseMode // Embed common mode infrastructure

	commandContainer *widget.Container // Container for command buttons

}

func NewSquadManagementMode(modeManager *core.UIModeManager) *SquadManagementMode {
	mode := &SquadManagementMode{}
	mode.SetModeName("squad_management")
	mode.ModeManager = modeManager
	return mode
}

func (smm *SquadManagementMode) Initialize(ctx *core.UIContext) error {
	err := gui.NewModeBuilder(&smm.BaseMode, gui.ModeConfig{
		ModeName:   "squad_management",
		ReturnMode: "", // Context switch handled separately

		Hotkeys: []gui.HotkeySpec{
			{Key: ebiten.KeyB, TargetMode: "squad_builder"},
			{Key: ebiten.KeyP, TargetMode: "unit_purchase"},
			{Key: ebiten.KeyE, TargetMode: "squad_editor"},
		},

		Panels: []gui.PanelSpec{

			{CustomBuild: smm.buildActionButtons}, // Build after Context is available
		},

		StatusLabel: true,
		Commands:    true,
		OnRefresh:   smm.refresh,
	}).Build(ctx)

	return err
}

func (smm *SquadManagementMode) buildActionButtons() *widget.Container {
	// Create UI factory
	uiFactory := gui.NewUIComponentFactory(smm.Queries, smm.PanelBuilders, smm.Layout)

	// Create button callbacks (no panel wrapper - like combat mode)
	buttonContainer := uiFactory.CreateSquadManagementActionButtons(
		// Battle Map (ESC)
		func() {
			if smm.Context.ModeCoordinator != nil {
				if err := smm.Context.ModeCoordinator.EnterBattleMap("exploration"); err != nil {
					fmt.Printf("ERROR: Failed to enter battle map: %v\n", err)
				}
			}
		},
		// Squad Builder (B)
		func() {
			if mode, exists := smm.ModeManager.GetMode("squad_builder"); exists {
				smm.ModeManager.RequestTransition(mode, "Squad Builder clicked")
			}
		},
		// Buy Units (P)
		func() {
			if mode, exists := smm.ModeManager.GetMode("unit_purchase"); exists {
				smm.ModeManager.RequestTransition(mode, "Buy Units clicked")
			}
		},
		// Edit Squad (E)
		func() {
			if mode, exists := smm.ModeManager.GetMode("squad_editor"); exists {
				smm.ModeManager.RequestTransition(mode, "Edit Squad clicked")
			}
		},
	)

	return buttonContainer
}

func (smm *SquadManagementMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Squad Management Mode")

	return nil
}

func (smm *SquadManagementMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Squad Management Mode")

	return nil
}

func (smm *SquadManagementMode) Update(deltaTime float64) error {
	// Could refresh squad data periodically
	// For now, data is static until mode is re-entered
	return nil
}

func (smm *SquadManagementMode) Render(screen *ebiten.Image) {
	// No custom rendering - ebitenui draws everything
}

func (smm *SquadManagementMode) HandleInput(inputState *core.InputState) bool {
	// Handle common input (ESC key)
	if smm.HandleCommonInput(inputState) {
		return true
	}

	// Handle undo/redo input (Ctrl+Z, Ctrl+Y)
	if smm.CommandHistory.HandleInput(inputState) {
		return true
	}

	// E key hotkey is now handled by gui.BaseMode.HandleCommonInput via RegisterHotkey
	return false
}

// refreshAfterUndoRedo is called after successful undo/redo operations
func (smm *SquadManagementMode) refresh() {

}
