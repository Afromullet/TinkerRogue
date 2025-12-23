package core

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// UIModeManager coordinates switching between UI modes
type UIModeManager struct {
	currentMode       UIMode
	modes             map[string]UIMode // Registry of all available modes
	context           *UIContext
	pendingTransition *ModeTransition
	inputState        *InputState
}

func NewUIModeManager(ctx *UIContext) *UIModeManager {
	return &UIModeManager{
		modes:   make(map[string]UIMode),
		context: ctx,
		inputState: &InputState{
			KeysPressed:       make(map[ebiten.Key]bool),
			KeysJustPressed:   make(map[ebiten.Key]bool),
			PlayerInputStates: &ctx.PlayerData.InputStates,
		},
	}
}

// RegisterMode adds a mode to the available modes
func (umm *UIModeManager) RegisterMode(mode UIMode) error {
	name := mode.GetModeName()
	if _, exists := umm.modes[name]; exists {
		return fmt.Errorf("mode %s already registered", name)
	}

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

// Render renders the current mode
func (umm *UIModeManager) Render(screen *ebiten.Image) {
	if umm.currentMode != nil {
		// Render mode-specific UI
		umm.currentMode.Render(screen)

		// Draw the ebitenui widgets
		umm.currentMode.GetEbitenUI().Draw(screen)
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
	umm.inputState.MouseReleased = !umm.inputState.MousePressed

	// Keyboard (example keys - expand as needed)
	keysToTrack := []ebiten.Key{
		ebiten.KeyE, ebiten.KeyC, ebiten.KeyF, ebiten.KeyEscape,
		ebiten.KeyI, ebiten.KeyTab, ebiten.KeySpace, ebiten.KeyB,
		ebiten.KeyP,
		// Combat mode keys
		ebiten.KeyH, ebiten.KeyA, ebiten.KeyM, ebiten.KeyZ, ebiten.KeyY,
		ebiten.Key1, ebiten.Key2, ebiten.Key3,
		// Modifier keys for shortcuts (Ctrl+Z, Ctrl+Y, Shift+H)
		ebiten.KeyControl, ebiten.KeyMeta, ebiten.KeyShift,
		ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
	}

	prevPressed := make(map[ebiten.Key]bool)
	for k, v := range umm.inputState.KeysPressed {
		prevPressed[k] = v
	}

	for _, key := range keysToTrack {
		isPressed := ebiten.IsKeyPressed(key)
		umm.inputState.KeysPressed[key] = isPressed

		// Just pressed = pressed now but not last frame
		wasPressed := prevPressed[key]
		umm.inputState.KeysJustPressed[key] = isPressed && !wasPressed
	}

	// Sync with PlayerInputStates (bridge to existing system)
	umm.inputState.PlayerInputStates = &umm.context.PlayerData.InputStates
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
