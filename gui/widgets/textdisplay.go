package widgets

import (
	"github.com/ebitenui/ebitenui/widget"
)

// TextDisplayComponent manages a text widget with periodic updates
type TextDisplayComponent struct {
	textWidget *widget.Text
	formatter  TextDisplayFormatter
}

// TextDisplayFormatter converts data to display text
type TextDisplayFormatter func() string

// NewTextDisplayComponent creates a text display updater
func NewTextDisplayComponent(
	textWidget *widget.Text,
	formatter TextDisplayFormatter,
) *TextDisplayComponent {
	return &TextDisplayComponent{
		textWidget: textWidget,
		formatter:  formatter,
	}
}

// Refresh updates the text display
func (tdc *TextDisplayComponent) Refresh() {
	if tdc.textWidget == nil || tdc.formatter == nil {
		return
	}
	tdc.textWidget.Label = tdc.formatter()
}

// SetText sets arbitrary text directly
func (tdc *TextDisplayComponent) SetText(text string) {
	if tdc.textWidget != nil {
		tdc.textWidget.Label = text
	}
}
