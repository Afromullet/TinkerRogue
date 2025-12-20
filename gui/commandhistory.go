package gui

import (
	"fmt"
	"game_main/gui/builders"
	"game_main/gui/core"
	"game_main/squads/squadcommands"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CommandHistory provides standardized command executor integration for UI modes.
// Handles undo/redo functionality with consistent UI patterns across modes.
type CommandHistory struct {
	executor       *squadcommands.CommandExecutor
	onStatusChange func(string) // Callback to display status messages
	onRefresh      func()        // Optional callback after successful undo/redo
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

// HandleInput processes undo/redo keyboard shortcuts (Ctrl+Z, Ctrl+Y)
// Returns true if input was handled
func (ch *CommandHistory) HandleInput(inputState *core.InputState) bool {
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

// CreateUndoButton creates a standard undo button
func (ch *CommandHistory) CreateUndoButton() *widget.Button {
	return builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Undo (Ctrl+Z)",
		OnClick: func() {
			ch.Undo()
		},
	})
}

// CreateRedoButton creates a standard redo button
func (ch *CommandHistory) CreateRedoButton() *widget.Button {
	return builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Redo (Ctrl+Y)",
		OnClick: func() {
			ch.Redo()
		},
	})
}

// CanUndo returns whether there are commands available to undo
func (ch *CommandHistory) CanUndo() bool {
	return ch.executor.CanUndo()
}

// CanRedo returns whether there are commands available to redo
func (ch *CommandHistory) CanRedo() bool {
	return ch.executor.CanRedo()
}
