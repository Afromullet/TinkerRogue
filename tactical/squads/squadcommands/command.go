package squadcommands

// SquadCommand represents an undoable squad operation
// Commands encapsulate validation, execution, and undo logic for squad management
type SquadCommand interface {
	// Validate checks if the command can be executed
	// Returns error if validation fails (e.g., squad doesn't exist, invalid state)
	Validate() error

	// Execute performs the command's operation
	// Should capture state needed for undo before making changes
	// Returns error if execution fails
	Execute() error

	// Undo reverses the command's operation
	// Uses captured state to restore previous state
	// Returns error if undo fails
	Undo() error

	// Description returns a human-readable description of the command
	// Used for logging and displaying command history
	Description() string
}
