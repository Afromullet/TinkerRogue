// Package framework provides UI and mode system for the game
package framework

import (
	"fmt"

	"game_main/gui/builders"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// ============================================================================
// PANEL AND CONTAINER HELPERS
// ============================================================================

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
//	Buttons: []framework.ButtonGroupSpec{
//	    {
//	        Position: builders.BottomCenter(),
//	        Buttons: []builders.ButtonSpec{
//	            framework.ModeTransitionSpec(modeManager, "Close (ESC)", "squad_management"),
//	        },
//	    },
//	}
func ModeTransitionSpec(modeManager *UIModeManager, text, targetMode string) builders.ButtonSpec {
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
