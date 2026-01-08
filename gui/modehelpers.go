// Package gui provides UI and mode system for the game
package gui

import (
	"fmt"
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/core"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
)

// ============================================================================
// PANEL AND CONTAINER HELPERS
// ============================================================================

// CreateBottomCenterButtonContainer creates a standard bottom-center button container.
// Used by 4+ modes with identical layout (horizontal row, centered at bottom).
// Encapsulates repeated panel building code.
func CreateBottomCenterButtonContainer(panelBuilders *builders.PanelBuilders) *widget.Container {
	return panelBuilders.BuildPanel(
		builders.BottomCenter(),
		builders.HorizontalRowLayout(),
		builders.CustomPadding(widget.Insets{
			Bottom: int(float64(panelBuilders.Layout.ScreenHeight) * specs.BottomButtonOffset),
		}),
	)
}

// AddActionButton adds a button to an action button container with consistent styling.
// Reduces boilerplate when building action button collections.
// Each button is created with standard ButtonConfig and added to container.
func AddActionButton(container *widget.Container, text string, onClick func()) {
	btn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text:    text,
		OnClick: onClick,
	})
	container.AddChild(btn)
}

// CreateActionButtonGroup creates an action button group with the specified position and button specs.
// Consolidates the pattern of creating a positioned container and adding multiple action buttons.
// Eliminates repetitive code across 6+ modes that build bottom-center button containers.
// Parameters:
//   - panelBuilders: Used to build the panel with consistent styling
//   - position: Panel position (e.g., builders.BottomCenter(), builders.TopCenter())
//   - specs: Slice of button specifications (text + onClick handlers)
//
// Returns a container with all buttons added and positioned according to the layout.
// Uses standard action_buttons specification for consistent sizing (50% width, 8% height).
// Example:
//
//	buttons := []builders.ButtonSpec{
//		{Text: "Save", OnClick: func() { /* ... */ }},
//		{Text: "Cancel", OnClick: func() { /* ... */ }},
//	}
//	container := CreateActionButtonGroup(panelBuilders, builders.BottomCenter(), buttons)
func CreateActionButtonGroup(panelBuilders *builders.PanelBuilders, position builders.PanelOption, buttonSpecs []builders.ButtonSpec) *widget.Container {
	// Validate inputs
	if panelBuilders == nil {
		panic("CreateActionButtonGroup: panelBuilders is nil")
	}
	if panelBuilders.Layout == nil {
		panic("CreateActionButtonGroup: panelBuilders.Layout is nil")
	}
	if position == nil {
		panic("CreateActionButtonGroup: position is nil")
	}

	// Create positioned container with standard horizontal layout
	// Use narrower width (0.35 = 35%) to avoid overlap with LeftBottom (15%) and BottomRight (24%) panels
	// Centered at 35% width spans from 32.5% to 67.5%, leaving space for side panels
	container := panelBuilders.BuildPanel(
		position,
		builders.Size(0.35, 0.08), // Narrower than action_buttons spec to prevent overlap
		builders.HorizontalRowLayout(),
		builders.CustomPadding(widget.Insets{
			Bottom: int(float64(panelBuilders.Layout.ScreenHeight) * specs.BottomButtonOffset),
		}),
	)

	// Add all buttons from buttonSpecs
	for _, spec := range buttonSpecs {
		AddActionButton(container, spec.Text, spec.OnClick)
	}

	return container
}

// Panel Creation Patterns:
//
// 1. Simple Detail Panels (panel + single textarea):
//    Use CreateStandardDetailPanel() with a spec from builders.StandardPanels.
//    This ensures consistent sizing/positioning across the application.
//    Example: inventorymode, infomode, explorationmode, combat log
//
// 2. Complex/Custom Panels (multiple widgets, custom layout):
//    Use builders.CreatePanelWithConfig() directly when you need:
//    - Multiple textareas or widgets in one panel
//    - Custom layout (RowLayout instead of AnchorLayout)
//    - Dynamic panel creation (e.g., squadmanagementmode's per-squad panels)
//    Example: unitpurchasemode (2 textareas + button), squadmanagementmode
//
// 3. Standard Panels (no textarea, just container):
//    Use builders.CreateStandardPanel() with a spec from builders.StandardPanels.
//    Example: combat faction panel, squad list panel

// CreateFilterButtonContainer creates a filter button container with consistent styling.
// Eliminates repetitive panel building for filter buttons across multiple modes.
// Returns an empty container ready for buttons to be added to it.
// Parameters:
//   - panelBuilders: Used to build the panel with consistent styling
//   - alignment: Panel position (e.g., builders.TopLeft(), builders.TopRight())
func CreateFilterButtonContainer(panelBuilders *builders.PanelBuilders, alignment builders.PanelOption) *widget.Container {
	return panelBuilders.BuildPanel(
		alignment,
		builders.Padding(specs.PaddingStandard),
		builders.HorizontalRowLayout(),
	)
}

// CreateOptionsPanel creates a centered options panel using StandardPanels.
// Used by info/inspection mode for displaying selectable options.
// Returns an empty container ready for options to be added to it.
// Parameters:
//   - panelBuilders: Used to build the panel with consistent styling
func CreateOptionsPanel(panelBuilders *builders.PanelBuilders) *widget.Container {
	return builders.CreateStandardPanel(panelBuilders, "options_list")
}

// CreateStandardDetailPanel creates a detail panel with a text area using a StandardPanels specification.
// Combines panel specification lookup with detail panel functionality (AnchorLayout + TextArea).
// Parameters:
//   - panelBuilders: Used to build the panel with consistent styling
//   - layout: Screen layout config for calculating dimensions
//   - specName: Name of the panel specification in StandardPanels
//   - defaultText: Initial text to display in the textarea
func CreateStandardDetailPanel(
	panelBuilders *builders.PanelBuilders,
	layout *specs.LayoutConfig,
	specName string,
	defaultText string,
) (*widget.Container, *widget.TextArea) {
	spec, exists := builders.StandardPanels[specName]
	if !exists {
		return nil, nil
	}

	// Build the panel using the specification
	opts := []builders.PanelOption{
		spec.Position,
		builders.Size(spec.Width, spec.Height),
		builders.AnchorLayout(),
	}

	// Add padding option
	if spec.Custom != nil {
		opts = append(opts, builders.CustomPadding(*spec.Custom))
	} else {
		opts = append(opts, builders.Padding(spec.Padding))
	}

	panel := panelBuilders.BuildPanel(opts...)

	// Calculate textarea dimensions
	panelWidth := int(float64(layout.ScreenWidth) * spec.Width)
	panelHeight := int(float64(layout.ScreenHeight) * spec.Height)
	textArea := builders.CreateTextAreaWithConfig(builders.TextAreaConfig{
		MinWidth:  panelWidth - 20,
		MinHeight: panelHeight - 20,
		FontColor: color.White,
	})

	// Set initial text and add to panel
	textArea.SetText(defaultText)
	panel.AddChild(textArea)

	return panel, textArea
}

// ============================================================================
// NAVIGATION BUTTON HELPERS
// ============================================================================
// These helpers create ButtonSpec or Button widgets for common navigation patterns:
// - Mode transitions (within same context)
// - Context switches (Overworld ↔ BattleMap)
// - Command execution with confirmation dialogs

// ModeTransitionSpec creates a ButtonSpec for mode transitions.
// Use this within ButtonGroupSpec declarations in ModeBuilder.
//
// Example:
//
//	Buttons: []gui.ButtonGroupSpec{
//	    {
//	        Position: builders.BottomCenter(),
//	        Buttons: []builders.ButtonSpec{
//	            gui.ModeTransitionSpec(modeManager, "Close (ESC)", "squad_management"),
//	        },
//	    },
//	}
func ModeTransitionSpec(modeManager *core.UIModeManager, text, targetMode string) builders.ButtonSpec {
	return builders.ButtonSpec{
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
//	        Position: builders.BottomCenter(),
//	        Buttons: []builders.ButtonSpec{
//	            gui.ContextSwitchSpec(coordinator, "Battle Map (ESC)", "battlemap", "exploration"),
//	        },
//	    },
//	}
func ContextSwitchSpec(coordinator *core.GameModeCoordinator, text, targetContext, targetMode string) builders.ButtonSpec {
	return builders.ButtonSpec{
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
	return builders.CreateButtonWithConfig(builders.ButtonConfig{
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
	return builders.CreateButtonWithConfig(builders.ButtonConfig{
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
	return builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: config.Text,
		OnClick: func() {
			// If no confirmation needed, execute immediately
			if config.ConfirmTitle == "" {
				cmd := config.CreateCommand()
				config.CommandHistory.Execute(cmd)
				return
			}

			// Show confirmation dialog
			dialog := builders.CreateConfirmationDialog(builders.DialogConfig{
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
