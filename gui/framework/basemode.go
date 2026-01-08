package framework

import (
	"fmt"

	"game_main/gui/builders"
	"game_main/gui/core"
	"game_main/gui/guicomponents"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// InputBinding maps a key to a mode transition
type InputBinding struct {
	Key        ebiten.Key
	TargetMode string
	Reason     string
}

// BaseMode provides common mode infrastructure shared by all UI modes.
// Modes should embed this struct to inherit common fields and behavior.
type BaseMode struct {
	ui             *ebitenui.UI
	Context        *core.UIContext           // Exported for mode access
	Layout         *specs.LayoutConfig       // Exported for mode access
	ModeManager    *core.UIModeManager       // Exported for mode access
	RootContainer  *widget.Container         // Exported for mode access
	PanelBuilders  *builders.PanelBuilders   // Exported for mode access
	Queries        *guicomponents.GUIQueries // Unified ECS query service - exported for mode access
	StatusLabel    *widget.Text              // Optional status label for display and logging - set by modes that need it
	CommandHistory *CommandHistory           // Optional command history for undo/redo support
	PanelWidgets   map[string]interface{}    // Stores typed panel widgets by SpecName (TextArea, List, etc.)
	modeName       string
	returnMode     string                      // Mode to return to on ESC/close
	hotkeys        map[ebiten.Key]InputBinding // Registered hotkeys for mode transitions
}

// SetModeName sets the mode identifier
func (bm *BaseMode) SetModeName(name string) {
	bm.modeName = name
}

// SetReturnMode sets the mode to return to on ESC/close
func (bm *BaseMode) SetReturnMode(name string) {
	bm.returnMode = name
}

// InitializeBase sets up common mode infrastructure.
// Call this from each mode's Initialize() method before building mode-specific UI.
// Note: modeName and returnMode should be set in the mode's constructor using SetModeName and SetReturnMode.
//
// Parameters:
//   - ctx: UIContext with ECS manager, player data, etc.
func (bm *BaseMode) InitializeBase(ctx *core.UIContext) {
	bm.Context = ctx
	bm.Layout = specs.NewLayoutConfig(ctx)
	bm.PanelBuilders = builders.NewPanelBuilders(bm.Layout, bm.ModeManager)

	// Initialize unified ECS query service
	bm.Queries = guicomponents.NewGUIQueries(ctx.ECSManager)

	// Initialize hotkeys map
	bm.hotkeys = make(map[ebiten.Key]InputBinding)

	// Initialize panel widgets map
	bm.PanelWidgets = make(map[string]interface{})

	// Create root ebitenui container
	bm.ui = &ebitenui.UI{}
	bm.RootContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	bm.ui.Container = bm.RootContainer
}

// RegisterHotkey registers a key binding that transitions to a target mode.
// This provides a declarative way to set up mode navigation without duplicating
// hotkey handling code across modes.
//
// Parameters:
//   - key: The ebiten key to bind
//   - targetMode: The name of the mode to transition to
//
// Example:
//
//	bm.RegisterHotkey(ebiten.KeyE, "squad_management")
func (bm *BaseMode) RegisterHotkey(key ebiten.Key, targetMode string) {
	if bm.hotkeys == nil {
		bm.hotkeys = make(map[ebiten.Key]InputBinding)
	}

	bm.hotkeys[key] = InputBinding{
		Key:        key,
		TargetMode: targetMode,
		Reason:     fmt.Sprintf("%s key pressed", key.String()),
	}
}

// SetStatus updates the status label with a message and logs to console.
// Modes should assign their status label to StatusLabel in Initialize() to use this.
func (bm *BaseMode) SetStatus(message string) {
	if bm.StatusLabel != nil {
		bm.StatusLabel.Label = message
	}
	fmt.Println(message) // Also log to console
}

// InitializeCommandHistory creates and initializes CommandHistory for this mode.
// The onRefresh callback is called after successful undo/redo operations.
// Automatically uses BaseMode.SetStatus for status updates.
//
// Example usage in mode's Initialize():
//
//	bm.InitializeCommandHistory(func() {
//		bm.refreshCurrentSquad()
//	})
func (bm *BaseMode) InitializeCommandHistory(onRefresh func()) {
	bm.CommandHistory = NewCommandHistory(bm.SetStatus, onRefresh)
}

// HandleCommonInput processes standard input that's common across all modes.
// Checks registered hotkeys and handles ESC key to return to the designated return mode.
//
// Returns true if input was consumed (prevents further propagation).
// Modes should call this first in their HandleInput() method before processing
// mode-specific input.
func (bm *BaseMode) HandleCommonInput(inputState *core.InputState) bool {
	// Check registered hotkeys for mode transitions
	for key, binding := range bm.hotkeys {
		if inputState.KeysJustPressed[key] {
			if targetMode, exists := bm.ModeManager.GetMode(binding.TargetMode); exists {
				bm.ModeManager.RequestTransition(targetMode, binding.Reason)
				return true
			}
		}
	}

	// ESC key - return to designated mode
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		if returnMode, exists := bm.ModeManager.GetMode(bm.returnMode); exists {
			bm.ModeManager.RequestTransition(returnMode, "ESC pressed")
			return true
		}
	}
	return false
}

// Default implementations for core.UIMode interface.
// Modes can override these as needed.

// GetEbitenUI returns the root ebitenui.UI for this mode.
func (bm *BaseMode) GetEbitenUI() *ebitenui.UI {
	return bm.ui
}

// GetModeName returns identifier for this mode (for debugging/logging).
func (bm *BaseMode) GetModeName() string {
	return bm.modeName
}

// Update is called every frame while mode is active.
// Default implementation does nothing. Override if mode needs per-frame updates.
func (bm *BaseMode) Update(deltaTime float64) error {
	return nil
}

// Render is called to draw this mode's UI.
// Default implementation does nothing (ebitenui handles rendering automatically).
// Override if mode needs custom rendering (e.g., highlighting tiles, drawing overlays).
func (bm *BaseMode) Render(screen *ebiten.Image) {
	// No custom rendering by default - ebitenui handles everything
}

// Enter is called when switching TO this mode.
// Default implementation does nothing. Override to refresh UI state.
func (bm *BaseMode) Enter(fromMode core.UIMode) error {
	return nil
}

// Exit is called when switching FROM this mode to another.
// Default implementation does nothing. Override to clean up resources.
func (bm *BaseMode) Exit(toMode core.UIMode) error {
	return nil
}
