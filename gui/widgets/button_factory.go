package widgets

import (
	"github.com/ebitenui/ebitenui/widget"
)

// ButtonSpec defines a single button in a button group
type ButtonSpec struct {
	Text    string
	OnClick func()
}

// ButtonGroupConfig defines configuration for creating a group of buttons
type ButtonGroupConfig struct {
	Buttons      []ButtonSpec           // Buttons to create
	Direction    widget.Direction       // Horizontal or Vertical
	Spacing      int                    // Space between buttons
	Padding      widget.Insets          // Container padding
	LayoutData   *widget.AnchorLayoutData // Optional positioning (for anchor layout)
}

// CreateButtonGroup creates a container with multiple buttons arranged according to config
func CreateButtonGroup(config ButtonGroupConfig) *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(config.Direction),
			widget.RowLayoutOpts.Spacing(config.Spacing),
			widget.RowLayoutOpts.Padding(config.Padding),
		)),
	)

	// Add all buttons
	for _, spec := range config.Buttons {
		button := CreateButtonWithConfig(ButtonConfig{
			Text:    spec.Text,
			OnClick: spec.OnClick,
		})
		container.AddChild(button)
	}

	// Apply layout data if provided (for anchor layout positioning)
	if config.LayoutData != nil {
		container.GetWidget().LayoutData = *config.LayoutData
	}

	return container
}

// HorizontalButtonGroup is a convenience helper for creating horizontal button groups
func HorizontalButtonGroup(buttons []ButtonSpec, spacing int, padding widget.Insets) *widget.Container {
	return CreateButtonGroup(ButtonGroupConfig{
		Buttons:   buttons,
		Direction: widget.DirectionHorizontal,
		Spacing:   spacing,
		Padding:   padding,
	})
}

// VerticalButtonGroup is a convenience helper for creating vertical button groups
func VerticalButtonGroup(buttons []ButtonSpec, spacing int, padding widget.Insets) *widget.Container {
	return CreateButtonGroup(ButtonGroupConfig{
		Buttons:   buttons,
		Direction: widget.DirectionVertical,
		Spacing:   spacing,
		Padding:   padding,
	})
}
