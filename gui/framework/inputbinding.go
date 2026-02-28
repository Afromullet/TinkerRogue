package framework

import "github.com/hajimehoshi/ebiten/v2"

// InputTrigger controls when a binding fires relative to key state.
type InputTrigger int

const (
	TriggerJustPressed  InputTrigger = iota // Fires on the frame the key goes down (default)
	TriggerHeld                             // Fires every frame the key is held
	TriggerJustReleased                     // Fires on the frame the key goes up
)

// ModifierMask is a bitmask for modifier key requirements.
type ModifierMask uint8

const (
	ModNone  ModifierMask = 0
	ModCtrl  ModifierMask = 1 << iota // Control or Meta (for macOS)
	ModShift                          // Shift
	ModAlt                            // Alt
)

// InputBinding maps a physical key (or mouse button) to a semantic action.
type InputBinding struct {
	Action    InputAction
	Key       ebiten.Key
	Mouse     ebiten.MouseButton
	IsMouse   bool
	Modifiers ModifierMask
	Trigger   InputTrigger
}
