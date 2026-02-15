package framework

import (
	"fmt"
	"game_main/tactical/squadcommands"

	"github.com/hajimehoshi/ebiten/v2"
)

// CommandHistory provides standardized command executor integration for UI modes.
// Handles undo/redo functionality with consistent UI patterns across modes.
type CommandHistory struct {
	executor       *squadcommands.CommandExecutor
	onStatusChange func(string) // Callback to display status messages
	onRefresh      func()       // Optional callback after successful undo/redo
}

// NewCommandHistory creates a new CommandHistory with the given callbacks
func NewCommandHistory(onStatusChange func(string), onRefresh func()) *CommandHistory {
	return &CommandHistory{
		executor:       squadcommands.NewCommandExecutor(),
		onStatusChange: onStatusChange,
		onRefresh:      onRefresh,
	}
}

// Execute validates and executes a command, showing status messages
func (ch *CommandHistory) Execute(cmd squadcommands.SquadCommand) bool {
	result := ch.executor.Execute(cmd)

	if result.Success {
		ch.onStatusChange(fmt.Sprintf("✓ %s", result.Description))
		if ch.onRefresh != nil {
			ch.onRefresh()
		}
	} else {
		ch.onStatusChange(fmt.Sprintf("✗ %s", result.Error))
	}

	return result.Success
}

// Undo reverses the last command
func (ch *CommandHistory) Undo() bool {
	if !ch.executor.CanUndo() {
		ch.onStatusChange("Nothing to undo")
		return false
	}

	result := ch.executor.Undo()

	if result.Success {
		ch.onStatusChange(fmt.Sprintf("⟲ %s", result.Description))
		if ch.onRefresh != nil {
			ch.onRefresh()
		}
	} else {
		ch.onStatusChange(fmt.Sprintf("✗ %s", result.Error))
	}

	return result.Success
}

// Redo re-executes the last undone command
func (ch *CommandHistory) Redo() bool {
	if !ch.executor.CanRedo() {
		ch.onStatusChange("Nothing to redo")
		return false
	}

	result := ch.executor.Redo()

	if result.Success {
		ch.onStatusChange(fmt.Sprintf("⟳ %s", result.Description))
		if ch.onRefresh != nil {
			ch.onRefresh()
		}
	} else {
		ch.onStatusChange(fmt.Sprintf("✗ %s", result.Error))
	}

	return result.Success
}

// Clear resets all command history and redo stack
func (ch *CommandHistory) Clear() {
	ch.executor.ClearHistory()
}

// HandleInput processes undo/redo keyboard shortcuts (Ctrl+Z, Ctrl+Y)
// Returns true if input was handled
func (ch *CommandHistory) HandleInput(inputState *InputState) bool {
	// Handle Ctrl+Z for Undo
	if inputState.KeysJustPressed[ebiten.KeyZ] && (inputState.KeysPressed[ebiten.KeyControl] || inputState.KeysPressed[ebiten.KeyMeta]) {
		ch.Undo()
		return true
	}

	// Handle Ctrl+Y for Redo
	if inputState.KeysJustPressed[ebiten.KeyY] && (inputState.KeysPressed[ebiten.KeyControl] || inputState.KeysPressed[ebiten.KeyMeta]) {
		ch.Redo()
		return true
	}

	return false
}

