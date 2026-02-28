package framework

import "github.com/hajimehoshi/ebiten/v2"

// ActionMap holds a set of input bindings and resolves them against the current frame's InputState.
type ActionMap struct {
	Name     string
	bindings []InputBinding
}

// NewActionMap creates a new empty ActionMap with the given name.
func NewActionMap(name string) *ActionMap {
	return &ActionMap{Name: name}
}

// Bind adds a just-pressed key binding with no modifiers.
func (am *ActionMap) Bind(key ebiten.Key, action InputAction) *ActionMap {
	am.bindings = append(am.bindings, InputBinding{
		Action:  action,
		Key:     key,
		Trigger: TriggerJustPressed,
	})
	return am
}

// BindMod adds a just-pressed key binding with modifier requirements.
func (am *ActionMap) BindMod(key ebiten.Key, mod ModifierMask, action InputAction) *ActionMap {
	am.bindings = append(am.bindings, InputBinding{
		Action:    action,
		Key:       key,
		Modifiers: mod,
		Trigger:   TriggerJustPressed,
	})
	return am
}

// BindMouse adds a just-pressed mouse button binding.
func (am *ActionMap) BindMouse(button ebiten.MouseButton, action InputAction) *ActionMap {
	am.bindings = append(am.bindings, InputBinding{
		Action:  action,
		Mouse:   button,
		IsMouse: true,
		Trigger: TriggerJustPressed,
	})
	return am
}

// BindRelease adds a key binding that fires on release.
func (am *ActionMap) BindRelease(key ebiten.Key, action InputAction) *ActionMap {
	am.bindings = append(am.bindings, InputBinding{
		Action:  action,
		Key:     key,
		Trigger: TriggerJustReleased,
	})
	return am
}

// BindHeld adds a key binding that fires every frame the key is held.
func (am *ActionMap) BindHeld(key ebiten.Key, action InputAction) *ActionMap {
	am.bindings = append(am.bindings, InputBinding{
		Action:  action,
		Key:     key,
		Trigger: TriggerHeld,
	})
	return am
}

// ResolveInto evaluates all bindings against the current InputState, writing active actions into dst.
// The caller must provide a pre-allocated map; it will be cleared before use.
// Modifier exclusivity: plain key bindings (ModNone) are rejected when any modifier is held,
// so Ctrl+Z won't also trigger a plain Z action.
func (am *ActionMap) ResolveInto(dst map[InputAction]bool, state *InputState) {
	clear(dst)

	anyModHeld := state.KeysPressed[ebiten.KeyControl] ||
		state.KeysPressed[ebiten.KeyMeta] ||
		state.KeysPressed[ebiten.KeyShift] ||
		state.KeysPressed[ebiten.KeyShiftLeft] ||
		state.KeysPressed[ebiten.KeyShiftRight] ||
		state.KeysPressed[ebiten.KeyAlt]

	for _, b := range am.bindings {
		if b.IsMouse {
			if am.resolveMouseBinding(b, state) {
				dst[b.Action] = true
			}
			continue
		}

		// Check modifier requirements
		if b.Modifiers == ModNone && anyModHeld {
			// Plain key binding but a modifier is held — skip
			continue
		}
		if b.Modifiers != ModNone && !am.modifiersMatch(b.Modifiers, state) {
			continue
		}

		// Check trigger type
		switch b.Trigger {
		case TriggerJustPressed:
			if state.KeysJustPressed[b.Key] {
				dst[b.Action] = true
			}
		case TriggerHeld:
			if state.KeysPressed[b.Key] {
				dst[b.Action] = true
			}
		case TriggerJustReleased:
			if state.KeysJustReleased[b.Key] {
				dst[b.Action] = true
			}
		}
	}
}

// modifiersMatch checks if the required modifier keys are pressed.
func (am *ActionMap) modifiersMatch(required ModifierMask, state *InputState) bool {
	if required&ModCtrl != 0 {
		if !state.KeysPressed[ebiten.KeyControl] && !state.KeysPressed[ebiten.KeyMeta] {
			return false
		}
	}
	if required&ModShift != 0 {
		if !state.KeysPressed[ebiten.KeyShift] &&
			!state.KeysPressed[ebiten.KeyShiftLeft] &&
			!state.KeysPressed[ebiten.KeyShiftRight] {
			return false
		}
	}
	if required&ModAlt != 0 {
		if !state.KeysPressed[ebiten.KeyAlt] {
			return false
		}
	}
	return true
}

// resolveMouseBinding checks if a mouse binding is active.
func (am *ActionMap) resolveMouseBinding(b InputBinding, state *InputState) bool {
	switch b.Trigger {
	case TriggerJustPressed:
		return state.MouseJustPressedButton(b.Mouse)
	case TriggerHeld:
		return state.MousePressed && state.MouseButton == b.Mouse
	default:
		return false
	}
}

// MergeActionMaps creates a new ActionMap by combining multiple maps.
// Bindings are evaluated in order (first map's bindings first).
func MergeActionMaps(name string, maps ...*ActionMap) *ActionMap {
	merged := NewActionMap(name)
	for _, m := range maps {
		merged.bindings = append(merged.bindings, m.bindings...)
	}
	return merged
}

// ActionMapProvider is an opt-in interface for modes that use the action map system.
// The UIModeManager checks if the current mode implements this interface and,
// if so, resolves actions automatically each frame.
type ActionMapProvider interface {
	GetActionMap() *ActionMap
}
