package squadcommands

import "fmt"

// CommandResult represents the result of a command execution, undo, or redo
type CommandResult struct {
	Success     bool   // Whether the operation succeeded
	Error       string // Error message if operation failed
	Description string // Human-readable description of what happened
}

// CommandExecutor manages command history and undo/redo stacks
// Provides centralized command execution with history tracking
type CommandExecutor struct {
	history    []SquadCommand // Stack of executed commands (most recent at end)
	redoStack  []SquadCommand // Stack of undone commands available for redo
	maxHistory int            // Maximum number of commands to keep in history
}

// NewCommandExecutor creates a new command executor with a maximum history size
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{
		history:    make([]SquadCommand, 0),
		redoStack:  make([]SquadCommand, 0),
		maxHistory: 20, // Keep last 20 commands
	}
}

// Execute validates and executes a command, adding it to history
// Clears the redo stack since new actions invalidate previously undone commands
func (ce *CommandExecutor) Execute(cmd SquadCommand) *CommandResult {
	result := &CommandResult{
		Description: cmd.Description(),
	}

	// Validate before executing
	if err := cmd.Validate(); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Validation failed: %v", err)
		return result
	}

	// Execute the command
	if err := cmd.Execute(); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Execution failed: %v", err)
		return result
	}

	// Add to history
	ce.history = append(ce.history, cmd)

	// Trim history if it exceeds max size
	if len(ce.history) > ce.maxHistory {
		ce.history = ce.history[1:]
	}

	// Clear redo stack - new action invalidates redo history
	ce.redoStack = ce.redoStack[:0]

	result.Success = true
	return result
}

// Undo reverses the last executed command
// Moves the command to the redo stack
func (ce *CommandExecutor) Undo() *CommandResult {
	if len(ce.history) == 0 {
		return &CommandResult{
			Success: false,
			Error:   "Nothing to undo",
		}
	}

	// Pop last command from history
	lastCmd := ce.history[len(ce.history)-1]
	ce.history = ce.history[:len(ce.history)-1]

	// Execute undo
	if err := lastCmd.Undo(); err != nil {
		// Undo failed - put command back in history
		ce.history = append(ce.history, lastCmd)
		return &CommandResult{
			Success: false,
			Error:   fmt.Sprintf("Undo failed: %v", err),
		}
	}

	// Add to redo stack
	ce.redoStack = append(ce.redoStack, lastCmd)

	return &CommandResult{
		Success:     true,
		Description: fmt.Sprintf("Undid: %s", lastCmd.Description()),
	}
}

// Redo re-executes a previously undone command
// Moves the command back to the history stack
func (ce *CommandExecutor) Redo() *CommandResult {
	if len(ce.redoStack) == 0 {
		return &CommandResult{
			Success: false,
			Error:   "Nothing to redo",
		}
	}

	// Pop last undone command from redo stack
	lastUndone := ce.redoStack[len(ce.redoStack)-1]
	ce.redoStack = ce.redoStack[:len(ce.redoStack)-1]

	// Re-execute the command
	if err := lastUndone.Execute(); err != nil {
		// Re-execution failed - put command back in redo stack
		ce.redoStack = append(ce.redoStack, lastUndone)
		return &CommandResult{
			Success: false,
			Error:   fmt.Sprintf("Redo failed: %v", err),
		}
	}

	// Add back to history
	ce.history = append(ce.history, lastUndone)

	// Trim history if needed
	if len(ce.history) > ce.maxHistory {
		ce.history = ce.history[1:]
	}

	return &CommandResult{
		Success:     true,
		Description: fmt.Sprintf("Redid: %s", lastUndone.Description()),
	}
}

// CanUndo returns whether there are commands available to undo
func (ce *CommandExecutor) CanUndo() bool {
	return len(ce.history) > 0
}

// CanRedo returns whether there are commands available to redo
func (ce *CommandExecutor) CanRedo() bool {
	return len(ce.redoStack) > 0
}

// GetHistoryCount returns the number of commands in history
func (ce *CommandExecutor) GetHistoryCount() int {
	return len(ce.history)
}

// GetRedoCount returns the number of commands available for redo
func (ce *CommandExecutor) GetRedoCount() int {
	return len(ce.redoStack)
}

// ClearHistory clears all command history and redo stack
func (ce *CommandExecutor) ClearHistory() {
	ce.history = ce.history[:0]
	ce.redoStack = ce.redoStack[:0]
}
