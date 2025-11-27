package widgets

import (
	"image/color"

	"game_main/gui/guiresources"

	"github.com/ebitenui/ebitenui/widget"
)

// DialogConfig provides configuration for modal dialogs
type DialogConfig struct {
	Title      string
	Message    string
	OnConfirm  func()
	OnCancel   func()
	MinWidth   int
	MinHeight  int
}

// CreateConfirmationDialog creates a modal confirmation dialog with Yes/No buttons
func CreateConfirmationDialog(config DialogConfig) *widget.Window {
	// Apply defaults
	if config.MinWidth == 0 {
		config.MinWidth = 400
	}
	if config.MinHeight == 0 {
		config.MinHeight = 200
	}

	// Create container for dialog content
	contentContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(guiresources.PanelRes.Image),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 20, Right: 20, Top: 20, Bottom: 20,
			}),
		)),
	)

	// Title label
	if config.Title != "" {
		titleLabel := CreateLargeLabel(config.Title)
		contentContainer.AddChild(titleLabel)
	}

	// Message label
	if config.Message != "" {
		messageLabel := CreateSmallLabel(config.Message)
		contentContainer.AddChild(messageLabel)
	}

	// Button container
	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(15),
		)),
	)

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

	// Create window
	window = widget.NewWindow(
		widget.WindowOpts.Contents(contentContainer),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.MinSize(config.MinWidth, config.MinHeight),
	)

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

// CreateTextInputDialog creates a modal text input dialog with OK/Cancel buttons
func CreateTextInputDialog(config TextInputDialogConfig) *widget.Window {
	// Apply defaults
	if config.MinWidth == 0 {
		config.MinWidth = 500
	}
	if config.MinHeight == 0 {
		config.MinHeight = 250
	}

	// Create container for dialog content
	contentContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(guiresources.PanelRes.Image),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 20, Right: 20, Top: 20, Bottom: 20,
			}),
		)),
	)

	// Title label
	if config.Title != "" {
		titleLabel := CreateLargeLabel(config.Title)
		contentContainer.AddChild(titleLabel)
	}

	// Message label
	if config.Message != "" {
		messageLabel := CreateSmallLabel(config.Message)
		contentContainer.AddChild(messageLabel)
	}

	// Text input
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

	// Button container
	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(15),
		)),
	)

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

	// Create window
	window = widget.NewWindow(
		widget.WindowOpts.Contents(contentContainer),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.MinSize(config.MinWidth, config.MinHeight),
	)

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

// CreateMessageDialog creates a simple message dialog with OK button
func CreateMessageDialog(config MessageDialogConfig) *widget.Window {
	// Apply defaults
	if config.MinWidth == 0 {
		config.MinWidth = 400
	}
	if config.MinHeight == 0 {
		config.MinHeight = 200
	}

	// Create container for dialog content
	contentContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(guiresources.PanelRes.Image),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Left: 20, Right: 20, Top: 20, Bottom: 20,
			}),
		)),
	)

	// Title label
	if config.Title != "" {
		titleLabel := CreateLargeLabel(config.Title)
		contentContainer.AddChild(titleLabel)
	}

	// Message label
	if config.Message != "" {
		messageLabel := widget.NewText(
			widget.TextOpts.Text(config.Message, guiresources.SmallFace, color.White),
			widget.TextOpts.MaxWidth(350), // Wrap text
		)
		contentContainer.AddChild(messageLabel)
	}

	// Reference to window for closing
	var window *widget.Window

	// OK button
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

	// Create window
	window = widget.NewWindow(
		widget.WindowOpts.Contents(contentContainer),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.MinSize(config.MinWidth, config.MinHeight),
	)

	return window
}
