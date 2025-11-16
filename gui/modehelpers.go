// Package gui provides UI and mode system for the game
package gui

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
)

// CreateCloseButton creates a standard close button that transitions to a target mode.
// Used consistently across all modes to provide ESC-like functionality.
// All modes use this same pattern - centralized here for consistency.
func CreateCloseButton(modeManager *UIModeManager, targetModeName, buttonText string) *widget.Button {
	return CreateButtonWithConfig(ButtonConfig{
		Text: buttonText,
		OnClick: func() {
			if targetMode, exists := modeManager.GetMode(targetModeName); exists {
				modeManager.RequestTransition(targetMode, "Close button pressed")
			}
		},
	})
}

// CreateBottomCenterButtonContainer creates a standard bottom-center button container.
// Used by 4+ modes with identical layout (horizontal row, centered at bottom).
// Encapsulates repeated panel building code.
func CreateBottomCenterButtonContainer(panelBuilders *PanelBuilders) *widget.Container {
	return panelBuilders.BuildPanel(
		BottomCenter(),
		HorizontalRowLayout(),
		CustomPadding(widget.Insets{
			Bottom: int(float64(panelBuilders.layout.ScreenHeight) * 0.08),
		}),
	)
}

// AddActionButton adds a button to an action button container with consistent styling.
// Reduces boilerplate when building action button collections.
// Each button is created with standard ButtonConfig and added to container.
func AddActionButton(container *widget.Container, text string, onClick func()) {
	btn := CreateButtonWithConfig(ButtonConfig{
		Text:    text,
		OnClick: onClick,
	})
	container.AddChild(btn)
}

// CreateDetailPanel creates a detail panel with a text area inside.
// Eliminates repetitive panel+textarea initialization code across multiple modes.
// Returns both the panel container and text area for flexibility.
// Parameters:
//   - panelBuilders: Used to build the panel with consistent styling
//   - layout: Screen layout config for calculating dimensions
//   - position: Panel position (e.g., RightCenter(), TopRight())
//   - widthPct: Panel width as percentage of screen width (0-1)
//   - heightPct: Panel height as percentage of screen height (0-1)
//   - paddingPct: Panel padding as percentage of screen (0-1)
//   - defaultText: Initial text to display in the textarea
func CreateDetailPanel(
	panelBuilders *PanelBuilders,
	layout *LayoutConfig,
	position PanelOption,
	widthPct, heightPct, paddingPct float64,
	defaultText string,
) (*widget.Container, *widget.TextArea) {
	// Build the panel with specified position, size, and padding
	panel := panelBuilders.BuildPanel(
		position,
		Size(widthPct, heightPct),
		Padding(paddingPct),
		AnchorLayout(),
	)

	// Calculate textarea dimensions (same as panel size minus padding)
	panelWidth := int(float64(layout.ScreenWidth) * widthPct)
	panelHeight := int(float64(layout.ScreenHeight) * heightPct)
	textArea := CreateTextAreaWithConfig(TextAreaConfig{
		MinWidth:  panelWidth - 20,
		MinHeight: panelHeight - 20,
		FontColor: color.White,
	})

	// Set initial text and add to panel
	textArea.SetText(defaultText)
	panel.AddChild(textArea)

	return panel, textArea
}

// CreateFilterButtonContainer creates a filter button container with consistent styling.
// Eliminates repetitive panel building for filter buttons across multiple modes.
// Returns an empty container ready for buttons to be added to it.
// Parameters:
//   - panelBuilders: Used to build the panel with consistent styling
//   - alignment: Panel position (e.g., TopLeft(), TopRight())
func CreateFilterButtonContainer(panelBuilders *PanelBuilders, alignment PanelOption) *widget.Container {
	return panelBuilders.BuildPanel(
		alignment,
		Padding(0.02),
		HorizontalRowLayout(),
	)
}

// CreateOptionsPanel creates a centered options panel using StandardPanels.
// Used by info/inspection mode for displaying selectable options.
// Returns an empty container ready for options to be added to it.
// Parameters:
//   - panelBuilders: Used to build the panel with consistent styling
func CreateOptionsPanel(panelBuilders *PanelBuilders) *widget.Container {
	return CreateStandardPanel(panelBuilders, "options_list")
}

// CreateStandardDetailPanel creates a detail panel with a text area using a StandardPanels specification.
// Combines panel specification lookup with detail panel functionality (AnchorLayout + TextArea).
// Parameters:
//   - panelBuilders: Used to build the panel with consistent styling
//   - layout: Screen layout config for calculating dimensions
//   - specName: Name of the panel specification in StandardPanels
//   - defaultText: Initial text to display in the textarea
func CreateStandardDetailPanel(
	panelBuilders *PanelBuilders,
	layout *LayoutConfig,
	specName string,
	defaultText string,
) (*widget.Container, *widget.TextArea) {
	spec, exists := StandardPanels[specName]
	if !exists {
		return nil, nil
	}

	// Build the panel using the specification
	opts := []PanelOption{
		spec.Position,
		Size(spec.Width, spec.Height),
		AnchorLayout(),
	}

	// Add padding option
	if spec.Custom != nil {
		opts = append(opts, CustomPadding(*spec.Custom))
	} else {
		opts = append(opts, Padding(spec.Padding))
	}

	panel := panelBuilders.BuildPanel(opts...)

	// Calculate textarea dimensions
	panelWidth := int(float64(layout.ScreenWidth) * spec.Width)
	panelHeight := int(float64(layout.ScreenHeight) * spec.Height)
	textArea := CreateTextAreaWithConfig(TextAreaConfig{
		MinWidth:  panelWidth - 20,
		MinHeight: panelHeight - 20,
		FontColor: color.White,
	})

	// Set initial text and add to panel
	textArea.SetText(defaultText)
	panel.AddChild(textArea)

	return panel, textArea
}
