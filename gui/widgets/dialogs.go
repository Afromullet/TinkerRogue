package widgets

import (
	"image/color"

	"game_main/gui/guiresources"

	"github.com/ebitenui/ebitenui/widget"
)

// Dialog System
//
// This file provides three specialized dialog functions:
// - CreateConfirmationDialog: Yes/No confirmation dialogs
// - CreateTextInputDialog: Text input dialogs with OK/Cancel
// - CreateMessageDialog: Simple message dialogs with OK
//
// Common Setup Extracted:
// Previously, these three functions duplicated 80% of their code (container setup,
// title/message labels, button containers, window creation). This has been refactored
// to use shared helper functions (createDialogContainer, addDialogHeader, etc.)
// while maintaining the same public API for backward compatibility.

// dialogBaseConfig contains common configuration for all dialog types.
// This struct extracts the duplicated fields from DialogConfig, TextInputDialogConfig, and MessageDialogConfig.
type dialogBaseConfig struct {
	Title     string
	Message   string
	MinWidth  int
	MinHeight int
}

// createDialogContainer creates the common dialog container with background and layout.
// Extracts the duplicated container setup code from all three dialog functions.
func createDialogContainer() *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(guiresources.PanelRes.Image),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 20, Right: 20, Top: 20, Bottom: 20,
			}),
		)),
	)
}

// addDialogHeader adds title and message labels to a dialog container.
// Extracts the duplicated title/message label creation from all three dialog functions.
func addDialogHeader(container *widget.Container, title, message string, wrapMessage bool) {
	// Title label
	if title != "" {
		titleLabel := CreateLargeLabel(title)
		container.AddChild(titleLabel)
	}

	// Message label
	if message != "" {
		if wrapMessage {
			// Use widget.NewText with MaxWidth for message dialogs
			messageLabel := widget.NewText(
				widget.TextOpts.Text(message, guiresources.SmallFace, color.White),
				widget.TextOpts.MaxWidth(350), // Wrap text
			)
			container.AddChild(messageLabel)
		} else {
			// Use simple label for other dialogs
			messageLabel := CreateSmallLabel(message)
			container.AddChild(messageLabel)
		}
	}
}

// createButtonContainer creates the common button container layout.
// Extracts the duplicated button container setup from all three dialog functions.
func createButtonContainer() *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(15),
		)),
	)
}

// createDialogWindow creates the final modal window.
// Extracts the duplicated window creation from all three dialog functions.
func createDialogWindow(container *widget.Container, minWidth, minHeight int) *widget.Window {
	return widget.NewWindow(
		widget.WindowOpts.Contents(container),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.MinSize(minWidth, minHeight),
	)
}

// DialogConfig provides configuration for modal dialogs
type DialogConfig struct {
	Title      string
	Message    string
	OnConfirm  func()
	OnCancel   func()
	MinWidth   int
	MinHeight  int
}

// CreateConfirmationDialog creates a modal confirmation dialog with Yes/No buttons.
// Uses extracted helper functions to reduce code duplication.
func CreateConfirmationDialog(config DialogConfig) *widget.Window {
	// Apply defaults
	if config.MinWidth == 0 {
		config.MinWidth = 400
	}
	if config.MinHeight == 0 {
		config.MinHeight = 200
	}

	// Create container using common helper
	contentContainer := createDialogContainer()

	// Add title and message using common helper
	addDialogHeader(contentContainer, config.Title, config.Message, false)

	// Create button container using common helper
	buttonContainer := createButtonContainer()

	// Reference to window for closing
	var window *widget.Window

	// Yes/Confirm button
	confirmBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Yes",
		OnClick: func() {
			if config.OnConfirm != nil {
				config.OnConfirm()
			}
			if window != nil {
				window.Close()
			}
		},
	})
	buttonContainer.AddChild(confirmBtn)

	// No/Cancel button
	cancelBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "No",
		OnClick: func() {
			if config.OnCancel != nil {
				config.OnCancel()
			}
			if window != nil {
				window.Close()
			}
		},
	})
	buttonContainer.AddChild(cancelBtn)

	contentContainer.AddChild(buttonContainer)

	// Create window using common helper
	window = createDialogWindow(contentContainer, config.MinWidth, config.MinHeight)

	return window
}

// TextInputDialogConfig provides configuration for text input dialogs
type TextInputDialogConfig struct {
	Title       string
	Message     string
	Placeholder string
	InitialText string
	OnConfirm   func(text string)
	OnCancel    func()
	MinWidth    int
	MinHeight   int
}

// CreateTextInputDialog creates a modal text input dialog with OK/Cancel buttons.
// Uses extracted helper functions to reduce code duplication.
func CreateTextInputDialog(config TextInputDialogConfig) *widget.Window {
	// Apply defaults
	if config.MinWidth == 0 {
		config.MinWidth = 500
	}
	if config.MinHeight == 0 {
		config.MinHeight = 250
	}

	// Create container using common helper
	contentContainer := createDialogContainer()

	// Add title and message using common helper
	addDialogHeader(contentContainer, config.Title, config.Message, false)

	// Text input (unique to this dialog type)
	var textInput *widget.TextInput
	textInput = CreateTextInputWithConfig(TextInputConfig{
		MinWidth:    400,
		MinHeight:   50,
		Placeholder: config.Placeholder,
	})

	// Set initial text if provided
	if config.InitialText != "" {
		textInput.SetText(config.InitialText)
	}

	contentContainer.AddChild(textInput)

	// Create button container using common helper
	buttonContainer := createButtonContainer()

	// Reference to window for closing
	var window *widget.Window

	// OK button
	okBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "OK",
		OnClick: func() {
			if config.OnConfirm != nil {
				config.OnConfirm(textInput.GetText())
			}
			if window != nil {
				window.Close()
			}
		},
	})
	buttonContainer.AddChild(okBtn)

	// Cancel button
	cancelBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Cancel",
		OnClick: func() {
			if config.OnCancel != nil {
				config.OnCancel()
			}
			if window != nil {
				window.Close()
			}
		},
	})
	buttonContainer.AddChild(cancelBtn)

	contentContainer.AddChild(buttonContainer)

	// Create window using common helper
	window = createDialogWindow(contentContainer, config.MinWidth, config.MinHeight)

	return window
}

// MessageDialogConfig provides configuration for simple message dialogs
type MessageDialogConfig struct {
	Title     string
	Message   string
	OnClose   func()
	MinWidth  int
	MinHeight int
}

// CreateMessageDialog creates a simple message dialog with OK button.
// Uses extracted helper functions to reduce code duplication.
func CreateMessageDialog(config MessageDialogConfig) *widget.Window {
	// Apply defaults
	if config.MinWidth == 0 {
		config.MinWidth = 400
	}
	if config.MinHeight == 0 {
		config.MinHeight = 200
	}

	// Create container using common helper
	contentContainer := createDialogContainer()

	// Add title and message using common helper (with text wrapping)
	addDialogHeader(contentContainer, config.Title, config.Message, true)

	// Reference to window for closing
	var window *widget.Window

	// OK button (unique to this dialog type - no button container needed)
	okBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "OK",
		OnClick: func() {
			if config.OnClose != nil {
				config.OnClose()
			}
			if window != nil {
				window.Close()
			}
		},
	})
	contentContainer.AddChild(okBtn)

	// Create window using common helper
	window = createDialogWindow(contentContainer, config.MinWidth, config.MinHeight)

	return window
}
