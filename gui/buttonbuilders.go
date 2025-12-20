package gui

import (
	"fmt"
	"game_main/gui/core"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
)

// Button Spec Helpers - for use with ButtonGroupSpec in ModeBuilder
// These create ButtonSpec structs that can be used in ButtonGroupSpec declarations

// ModeTransitionSpec creates a ButtonSpec for mode transitions.
// Use this within ButtonGroupSpec declarations in ModeBuilder.
//
// Example:
//
//	Buttons: []gui.ButtonGroupSpec{
//	    {
//	        Position: widgets.BottomCenter(),
//	        Buttons: []widgets.ButtonSpec{
//	            gui.ModeTransitionSpec(modeManager, "Close (ESC)", "squad_management"),
//	        },
//	    },
//	}
func ModeTransitionSpec(modeManager *core.UIModeManager, text, targetMode string) widgets.ButtonSpec {
	return widgets.ButtonSpec{
		Text: text,
		OnClick: func() {
			if mode, exists := modeManager.GetMode(targetMode); exists {
				modeManager.RequestTransition(mode, fmt.Sprintf("%s clicked", text))
			} else {
				fmt.Printf("WARNING: Mode %s not registered\n", targetMode)
			}
		},
	}
}

// ContextSwitchSpec creates a ButtonSpec for context switches.
// Use this within ButtonGroupSpec declarations in ModeBuilder.
//
// Example:
//
//	Buttons: []gui.ButtonGroupSpec{
//	    {
//	        Position: widgets.BottomCenter(),
//	        Buttons: []widgets.ButtonSpec{
//	            gui.ContextSwitchSpec(coordinator, "Battle Map (ESC)", "battlemap", "exploration"),
//	        },
//	    },
//	}
func ContextSwitchSpec(coordinator *core.GameModeCoordinator, text, targetContext, targetMode string) widgets.ButtonSpec {
	return widgets.ButtonSpec{
		Text: text,
		OnClick: func() {
			if coordinator != nil {
				switch targetContext {
				case "battlemap":
					if err := coordinator.EnterBattleMap(targetMode); err != nil {
						fmt.Printf("ERROR: Failed to enter battle map: %v\n", err)
					}
				case "overworld":
					if err := coordinator.ReturnToOverworld(targetMode); err != nil {
						fmt.Printf("ERROR: Failed to return to overworld: %v\n", err)
					}
				default:
					fmt.Printf("ERROR: Invalid context %s (use 'battlemap' or 'overworld')\n", targetContext)
				}
			} else {
				fmt.Printf("WARNING: GameModeCoordinator is nil\n")
			}
		},
	}
}

// ModeTransitionButton creates a button that transitions to another mode within the same context.
// This builder eliminates 8 lines of boilerplate code for each mode transition button.
//
// Example usage:
//
//	btn := gui.ModeTransitionButton(modeManager, "Squad Management (E)", "squad_management")
func ModeTransitionButton(modeManager *core.UIModeManager, text, targetMode string) *widget.Button {
	return widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: text,
		OnClick: func() {
			if mode, exists := modeManager.GetMode(targetMode); exists {
				modeManager.RequestTransition(mode, fmt.Sprintf("%s clicked", text))
			} else {
				fmt.Printf("WARNING: Mode %s not registered\n", targetMode)
			}
		},
	})
}

// ContextSwitchButton creates a button that switches between game contexts (Overworld ↔ BattleMap).
// This builder handles the common pattern of switching contexts with a specific mode.
//
// targetContext should be either "battlemap" or "overworld"
// targetMode is the mode to enter in the target context (e.g., "exploration", "squad_management")
//
// Example usage:
//
//	btn := gui.ContextSwitchButton(coordinator, "Battle Map (ESC)", "battlemap", "exploration")
//	btn := gui.ContextSwitchButton(coordinator, "Squads (E)", "overworld", "squad_management")
func ContextSwitchButton(coordinator *core.GameModeCoordinator, text, targetContext, targetMode string) *widget.Button {
	return widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: text,
		OnClick: func() {
			if coordinator != nil {
				switch targetContext {
				case "battlemap":
					if err := coordinator.EnterBattleMap(targetMode); err != nil {
						fmt.Printf("ERROR: Failed to enter battle map: %v\n", err)
					}
				case "overworld":
					if err := coordinator.ReturnToOverworld(targetMode); err != nil {
						fmt.Printf("ERROR: Failed to return to overworld: %v\n", err)
					}
				default:
					fmt.Printf("ERROR: Invalid context %s (use 'battlemap' or 'overworld')\n", targetContext)
				}
			} else {
				fmt.Printf("WARNING: GameModeCoordinator is nil\n")
			}
		},
	})
}

// CommandButtonConfig provides configuration for creating command execution buttons.
// Command buttons typically show a confirmation dialog before executing the command.
type CommandButtonConfig struct {
	Text           string                   // Button text
	ConfirmTitle   string                   // Confirmation dialog title
	ConfirmMessage string                   // Confirmation dialog message
	CreateCommand  func() interface{}       // Function that creates the command to execute
	CommandHistory CommandExecutor          // Interface for command execution
	OnCancel       func()                   // Optional: Called when user cancels
	UI             *ebitenui.UI             // Required: ebitenui UI for adding dialog window
}

// CommandExecutor is a minimal interface for command execution.
// This allows CommandButton to work with any command history implementation.
type CommandExecutor interface {
	Execute(cmd interface{}) bool
}

// CommandButton creates a button that executes a command with optional confirmation.
// This builder handles the common pattern of showing a confirmation dialog before
// executing a command (like disbanding squads, applying formations, etc.)
//
// If ConfirmTitle is empty, the command executes immediately without confirmation.
// Otherwise, a confirmation dialog is shown first.
//
// Example usage:
//
//	btn := gui.CommandButton(gui.CommandButtonConfig{
//	    Text:           "Disband Squad",
//	    ConfirmTitle:   "Confirm Disband",
//	    ConfirmMessage: "Disband squad? This will return all units to the roster.",
//	    CreateCommand:  func() interface{} {
//	        return squadcommands.NewDisbandSquadCommand(manager, playerID, squadID)
//	    },
//	    CommandHistory: commandHistory,
//	    OnCancel:       func() { setStatus("Cancelled") },
//	    UI:             ebitenUI,
//	})
func CommandButton(config CommandButtonConfig) *widget.Button {
	return widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: config.Text,
		OnClick: func() {
			// If no confirmation needed, execute immediately
			if config.ConfirmTitle == "" {
				cmd := config.CreateCommand()
				config.CommandHistory.Execute(cmd)
				return
			}

			// Show confirmation dialog
			dialog := widgets.CreateConfirmationDialog(widgets.DialogConfig{
				Title:   config.ConfirmTitle,
				Message: config.ConfirmMessage,
				OnConfirm: func() {
					cmd := config.CreateCommand()
					config.CommandHistory.Execute(cmd)
				},
				OnCancel: config.OnCancel,
			})

			if config.UI != nil {
				config.UI.AddWindow(dialog)
			} else {
				fmt.Printf("WARNING: UI is nil, cannot show confirmation dialog\n")
			}
		},
	})
}
