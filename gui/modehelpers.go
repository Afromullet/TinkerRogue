// Package gui provides UI and mode system for the game
package gui

import (
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
