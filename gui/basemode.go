package gui

import (
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// BaseMode provides common mode infrastructure shared by all UI modes.
// Modes should embed this struct to inherit common fields and behavior.
type BaseMode struct {
	ui            *ebitenui.UI
	context       *UIContext
	layout        *LayoutConfig
	modeManager   *UIModeManager
	rootContainer *widget.Container
	panelBuilders *PanelBuilders
	queries       *GUIQueries // Unified ECS query service
	modeName      string
	returnMode    string // Mode to return to on ESC/close
}

// InitializeBase sets up common mode infrastructure.
// Call this from each mode's Initialize() method before building mode-specific UI.
// Note: modeName and returnMode should be set in the mode's constructor.
//
// Parameters:
//   - ctx: UIContext with ECS manager, player data, etc.
func (bm *BaseMode) InitializeBase(ctx *UIContext) {
	bm.context = ctx
	bm.layout = NewLayoutConfig(ctx)
	bm.panelBuilders = NewPanelBuilders(bm.layout, bm.modeManager)

	// Initialize unified ECS query service
	bm.queries = NewGUIQueries(ctx.ECSManager)

	// Create root ebitenui container
	bm.ui = &ebitenui.UI{}
	bm.rootContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	bm.ui.Container = bm.rootContainer
}

// HandleCommonInput processes standard input that's common across all modes.
// Currently handles ESC key to return to the designated return mode.
//
// Returns true if input was consumed (prevents further propagation).
// Modes should call this first in their HandleInput() method before processing
// mode-specific input.
func (bm *BaseMode) HandleCommonInput(inputState *InputState) bool {
	// ESC key - return to designated mode
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		if returnMode, exists := bm.modeManager.GetMode(bm.returnMode); exists {
			bm.modeManager.RequestTransition(returnMode, "ESC pressed")
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
