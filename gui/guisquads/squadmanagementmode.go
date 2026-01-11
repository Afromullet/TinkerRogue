package guisquads

import (
	"fmt"
	"game_main/gui/framework"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadManagementMode shows one squad at a time with navigation controls
type SquadManagementMode struct {
	framework.BaseMode // Embed common mode infrastructure

	commandContainer *widget.Container // Container for command buttons
}

func NewSquadManagementMode(modeManager *framework.UIModeManager) *SquadManagementMode {
	mode := &SquadManagementMode{}
	mode.SetModeName("squad_management")
	mode.ModeManager = modeManager
	mode.SetSelf(mode) // Required for panel registry building
	return mode
}

func (smm *SquadManagementMode) Initialize(ctx *framework.UIContext) error {
	// Build base UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&smm.BaseMode, framework.ModeConfig{
		ModeName:   "squad_management",
		ReturnMode: "", // Context switch handled separately

		Hotkeys: []framework.HotkeySpec{
			{Key: ebiten.KeyB, TargetMode: "squad_builder"},
			{Key: ebiten.KeyP, TargetMode: "unit_purchase"},
			{Key: ebiten.KeyE, TargetMode: "squad_editor"},
		},

		StatusLabel: true,
		Commands:    true,
		OnRefresh:   smm.refresh,
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Build panels from registry
	if err := smm.BuildPanels(SquadManagementPanelActionButtons); err != nil {
		return err
	}

	return nil
}

func (smm *SquadManagementMode) Enter(fromMode framework.UIMode) error {
	fmt.Println("Entering Squad Management Mode")

	return nil
}

func (smm *SquadManagementMode) Exit(toMode framework.UIMode) error {
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

func (smm *SquadManagementMode) HandleInput(inputState *framework.InputState) bool {
	// Handle common input (ESC key)
	if smm.HandleCommonInput(inputState) {
		return true
	}

	// Handle undo/redo input (Ctrl+Z, Ctrl+Y)
	if smm.CommandHistory.HandleInput(inputState) {
		return true
	}

	// E key hotkey is now handled by framework.BaseMode.HandleCommonInput via RegisterHotkey
	return false
}

// refreshAfterUndoRedo is called after successful undo/redo operations
func (smm *SquadManagementMode) refresh() {

}
