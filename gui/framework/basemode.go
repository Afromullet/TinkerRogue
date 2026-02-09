package framework

import (
	"fmt"

	"game_main/gui/builders"
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
//
// Architecture:
//   - Panels: Registry of built panels with type-safe widget access
//   - Context: Access to ECS, PlayerData, ModeCoordinator
//   - Queries: Unified ECS query service for squad/unit lookups
type BaseMode struct {
	ui             *ebitenui.UI
	Context        *UIContext              // Exported for mode access
	Layout         *specs.LayoutConfig     // Exported for mode access
	ModeManager    *UIModeManager          // Exported for mode access
	RootContainer  *widget.Container       // Exported for mode access
	PanelBuilders  *builders.PanelBuilders // Exported for mode access
	Queries        *GUIQueries             // Unified ECS query service - exported for mode access
	StatusLabel    *widget.Text            // Optional status label for display and logging - set by modes that need it
	CommandHistory *CommandHistory         // Optional command history for undo/redo support
	Panels         *PanelRegistry          // Built panels with type-safe access

	modeName   string
	returnMode string                      // Mode to return to on ESC/close
	hotkeys    map[ebiten.Key]InputBinding // Registered hotkeys for mode transitions
	self       UIMode                      // Reference to concrete mode for panel building
}

// SetModeName sets the mode identifier
func (bm *BaseMode) SetModeName(name string) {
	bm.modeName = name
}

// SetReturnMode sets the mode to return to on ESC/close
func (bm *BaseMode) SetReturnMode(name string) {
	bm.returnMode = name
}

// SetSelf stores a reference to the concrete mode for panel building.
// Call this in the concrete mode's constructor after embedding BaseMode.
func (bm *BaseMode) SetSelf(mode UIMode) {
	bm.self = mode
}

// InitializeBase sets up common mode infrastructure.
// Call this from each mode's Initialize() method before building mode-specific UI.
// Note: modeName and returnMode should be set in the mode's constructor using SetModeName and SetReturnMode.
//
// Parameters:
//   - ctx: UIContext with ECS manager, player data, etc.
func (bm *BaseMode) InitializeBase(ctx *UIContext) {
	bm.Context = ctx
	bm.Layout = specs.NewLayoutConfig(ctx.ScreenWidth, ctx.ScreenHeight, ctx.TileSize)
	bm.PanelBuilders = builders.NewPanelBuilders(bm.Layout)

	// Use unified ECS query service from context
	bm.Queries = ctx.Queries

	// Initialize hotkeys map
	bm.hotkeys = make(map[ebiten.Key]InputBinding)

	// Initialize panel registry
	bm.Panels = NewPanelRegistry()

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

// SetStatus updates the status label with a message.
// Modes should assign their status label to StatusLabel in Initialize() to use this.
func (bm *BaseMode) SetStatus(message string) {
	if bm.StatusLabel != nil {
		bm.StatusLabel.Label = message
	}
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
func (bm *BaseMode) HandleCommonInput(inputState *InputState) bool {
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

// Default implementations for UIMode interface.
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
func (bm *BaseMode) Enter(fromMode UIMode) error {
	return nil
}

// Exit is called when switching FROM this mode to another.
// Default implementation does nothing. Override to clean up resources.
func (bm *BaseMode) Exit(toMode UIMode) error {
	return nil
}

// BuildPanels builds multiple panels from the global registry and adds them to the root container.
// Panels are built using the stored self reference (set via SetSelf) for OnCreate callbacks.
//
// Example:
//
//	cm.BuildPanels(CombatPanelTurnOrder, CombatPanelFactionInfo, CombatPanelSquadList)
func (bm *BaseMode) BuildPanels(panelTypes ...PanelType) error {
	for _, ptype := range panelTypes {
		result, err := BuildRegisteredPanel(ptype, bm.self, bm.PanelBuilders, bm.Layout)
		if err != nil {
			return fmt.Errorf("failed to build panel %s: %w", ptype, err)
		}
		bm.Panels.Add(result)
		if result.Container != nil {
			bm.RootContainer.AddChild(result.Container)
		}
	}
	return nil
}

// GetTextLabel returns the text label from a panel, or nil if not found.
// Use this for type-safe access to text panels.
func (bm *BaseMode) GetTextLabel(ptype PanelType) *widget.Text {
	if result := bm.Panels.Get(ptype); result != nil {
		return result.TextLabel
	}
	return nil
}

// GetPanelContainer returns the container for a panel, or nil if not found.
// Use this to add/remove children or access the panel directly.
func (bm *BaseMode) GetPanelContainer(ptype PanelType) *widget.Container {
	if result := bm.Panels.Get(ptype); result != nil {
		return result.Container
	}
	return nil
}
