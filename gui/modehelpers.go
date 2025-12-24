// Package gui provides UI and mode system for the game
package gui

import (
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

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
