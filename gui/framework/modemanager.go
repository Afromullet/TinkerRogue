package framework

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// UIModeManager coordinates switching between UI modes
type UIModeManager struct {
	currentMode       UIMode
	modes             map[string]UIMode // Registry of all available modes
	context           *UIContext
	pendingTransition *ModeTransition
	inputState        *InputState

	// Reusable buffers for ebiten key queries (avoids per-frame allocation)
	pressedBuf      []ebiten.Key
	justPressedBuf  []ebiten.Key
	justReleasedBuf []ebiten.Key
}

func NewUIModeManager(ctx *UIContext) *UIModeManager {
	return &UIModeManager{
		modes:   make(map[string]UIMode),
		context: ctx,
		inputState: &InputState{
			KeysPressed:            make(map[ebiten.Key]bool),
			KeysJustPressed:        make(map[ebiten.Key]bool),
			KeysJustReleased:       make(map[ebiten.Key]bool),
			PlayerInputStates:      &ctx.PlayerData.InputStates,
			mouseJustPressedButtons: make(map[ebiten.MouseButton]bool),
			ActionsActive:          make(map[InputAction]bool),
		},
	}
}

// RegisterMode adds a mode to the available modes.
// Re-registering an existing mode replaces it (re-runs Initialize).
func (umm *UIModeManager) RegisterMode(mode UIMode) error {
	name := mode.GetModeName()

	// Initialize the mode
	if err := mode.Initialize(umm.context); err != nil {
		return fmt.Errorf("failed to initialize mode %s: %w", name, err)
	}

	umm.modes[name] = mode
	return nil
}

// SetMode switches to the specified mode
func (umm *UIModeManager) SetMode(modeName string) error {
	newMode, exists := umm.modes[modeName]
	if !exists {
		return fmt.Errorf("mode %s not registered", modeName)
	}

	return umm.transitionToMode(newMode, fmt.Sprintf("SetMode(%s)", modeName))
}

// RequestTransition queues a mode transition (happens at end of frame)
func (umm *UIModeManager) RequestTransition(toMode UIMode, reason string) {
	umm.pendingTransition = &ModeTransition{
		ToMode: toMode,
		Reason: reason,
	}
}

// transitionToMode performs the actual mode switch
func (umm *UIModeManager) transitionToMode(toMode UIMode, reason string) error {
	// Exit current mode
	if umm.currentMode != nil {
		if err := umm.currentMode.Exit(toMode); err != nil {
			return fmt.Errorf("failed to exit mode %s: %w", umm.currentMode.GetModeName(), err)
		}
	}

	// Enter new mode
	if err := toMode.Enter(umm.currentMode); err != nil {
		return fmt.Errorf("failed to enter mode %s: %w", toMode.GetModeName(), err)
	}

	umm.currentMode = toMode
	fmt.Printf("UI Mode Transition: %s\n", reason)
	return nil
}

// Update updates the current mode and processes transitions
func (umm *UIModeManager) Update(deltaTime float64) error {
	// Update input state
	umm.updateInputState()

	// Handle pending transition
	if umm.pendingTransition != nil {
		if err := umm.transitionToMode(umm.pendingTransition.ToMode, umm.pendingTransition.Reason); err != nil {
			return err
		}
		umm.pendingTransition = nil
	}

	// Update current mode
	if umm.currentMode != nil {
		// Let mode handle input first
		umm.currentMode.HandleInput(umm.inputState)

		// Update mode logic
		if err := umm.currentMode.Update(deltaTime); err != nil {
			return err
		}

		// Update the ebitenui.UI (processes widget interactions)
		umm.currentMode.GetEbitenUI().Update()
	}

	return nil
}

// OverlayRenderer is an optional interface for modes that need to draw
// custom content on top of ebitenui widgets (after UI.Draw).
type OverlayRenderer interface {
	RenderOverlay(screen *ebiten.Image)
}

// Render renders the current mode
func (umm *UIModeManager) Render(screen *ebiten.Image) {
	if umm.currentMode != nil {
		// Render mode-specific UI
		umm.currentMode.Render(screen)

		// Draw the ebitenui widgets
		umm.currentMode.GetEbitenUI().Draw(screen)

		// Post-UI overlay (cards, custom graphics on top of panel backgrounds)
		if overlay, ok := umm.currentMode.(OverlayRenderer); ok {
			overlay.RenderOverlay(screen)
		}
	}
}

// updateInputState captures current frame's input
func (umm *UIModeManager) updateInputState() {
	// Mouse position
	umm.inputState.MouseX, umm.inputState.MouseY = ebiten.CursorPosition()

	// Mouse buttons (track which button pressed)
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		umm.inputState.MousePressed = true
		umm.inputState.MouseButton = ebiten.MouseButtonLeft
	} else if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		umm.inputState.MousePressed = true
		umm.inputState.MouseButton = ebiten.MouseButtonRight
	} else {
		umm.inputState.MousePressed = false
	}

	// Edge-detected mouse press (true only on the frame the button goes down)
	umm.inputState.MouseJustPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)

	// Per-button just-pressed tracking
	umm.inputState.mouseJustPressedButtons[ebiten.MouseButtonLeft] = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	umm.inputState.mouseJustPressedButtons[ebiten.MouseButtonRight] = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)

	clear(umm.inputState.KeysPressed)
	clear(umm.inputState.KeysJustPressed)
	clear(umm.inputState.KeysJustReleased)

	umm.pressedBuf = inpututil.AppendPressedKeys(umm.pressedBuf[:0])
	for _, key := range umm.pressedBuf {
		umm.inputState.KeysPressed[key] = true
	}

	umm.justPressedBuf = inpututil.AppendJustPressedKeys(umm.justPressedBuf[:0])
	for _, key := range umm.justPressedBuf {
		umm.inputState.KeysJustPressed[key] = true
	}

	umm.justReleasedBuf = inpututil.AppendJustReleasedKeys(umm.justReleasedBuf[:0])
	for _, key := range umm.justReleasedBuf {
		umm.inputState.KeysJustReleased[key] = true
	}

	// Sync with PlayerInputStates (bridge to existing system)
	umm.inputState.PlayerInputStates = &umm.context.PlayerData.InputStates

	// Resolve semantic actions if current mode provides an ActionMap
	clear(umm.inputState.ActionsActive)
	if umm.currentMode != nil {
		if provider, ok := umm.currentMode.(ActionMapProvider); ok {
			if actionMap := provider.GetActionMap(); actionMap != nil {
				actionMap.ResolveInto(umm.inputState.ActionsActive, umm.inputState)
			}
		}
	}
}

// GetCurrentMode returns the active mode
func (umm *UIModeManager) GetCurrentMode() UIMode {
	return umm.currentMode
}

// GetMode retrieves a registered mode by name
func (umm *UIModeManager) GetMode(name string) (UIMode, bool) {
	mode, exists := umm.modes[name]
	return mode, exists
}

// GetInputState returns the current frame's input state.
// Useful for systems like CameraController that process input outside the mode manager.
func (umm *UIModeManager) GetInputState() *InputState {
	return umm.inputState
}
