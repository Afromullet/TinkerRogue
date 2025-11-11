package gui

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"golang.org/x/image/font"
)

func SetContainerLocation(w *widget.Container, x, y int) {

	r := image.Rect(0, 0, 0, 0)
	r = r.Add(image.Point{x, y})

	w.SetLocation(r)

}

type StringDisplay interface {
	DisplayString()
}

// TextConfig provides declarative text widget configuration
type TextConfig struct {
	Text       string
	FontFace   font.Face
	Color      color.Color
	LayoutData interface{}
}

// CreateTextWithConfig creates a text widget from config
func CreateTextWithConfig(config TextConfig) *widget.Text {
	// Apply defaults
	if config.FontFace == nil {
		config.FontFace = SmallFace
	}
	if config.Color == nil {
		config.Color = color.White
	}

	opts := []widget.TextOpt{
		widget.TextOpts.Text(config.Text, config.FontFace, config.Color),
	}

	// Add layout data if provided
	if config.LayoutData != nil {
		opts = append(opts, widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(config.LayoutData),
		))
	}

	return widget.NewText(opts...)
}

// TextAreaConfig provides configuration for creating text areas with responsive sizing
type TextAreaConfig struct {
	MinWidth  int
	MinHeight int
	FontColor color.Color
}

// CreateTextAreaWithConfig creates a textarea with custom configuration
func CreateTextAreaWithConfig(config TextAreaConfig) *widget.TextArea {
	return widget.NewTextArea(
		widget.TextAreaOpts.ContainerOpts(
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(config.MinWidth, config.MinHeight),
			),
		),
		widget.TextAreaOpts.ControlWidgetSpacing(2),
		widget.TextAreaOpts.ProcessBBCode(true),
		widget.TextAreaOpts.FontColor(config.FontColor),
		widget.TextAreaOpts.FontFace(TextAreaRes.face),
		widget.TextAreaOpts.TextPadding(TextAreaRes.entryPadding),
		widget.TextAreaOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(ListRes.image)),
		widget.TextAreaOpts.SliderOpts(
			widget.SliderOpts.Images(ListRes.track, ListRes.handle),
			widget.SliderOpts.MinHandleSize(ListRes.handleSize),
			widget.SliderOpts.TrackPadding(ListRes.trackPadding),
		),
	)
}

// ButtonConfig provides declarative button configuration
type ButtonConfig struct {
	Text       string
	MinWidth   int
	MinHeight  int
	FontFace   font.Face
	TextColor  *widget.ButtonTextColor
	Image      *widget.ButtonImage
	Padding    widget.Insets
	OnClick    func() // Simplified callback - no args needed in most cases
	LayoutData interface{} // For positioning
}

// CreateButtonWithConfig creates a button from config
func CreateButtonWithConfig(config ButtonConfig) *widget.Button {
	// Apply defaults
	if config.MinWidth == 0 {
		config.MinWidth = 100
	}
	if config.MinHeight == 0 {
		config.MinHeight = 100
	}
	if config.FontFace == nil {
		config.FontFace = largeFace
	}
	if config.TextColor == nil {
		config.TextColor = &widget.ButtonTextColor{
			Idle: color.NRGBA{0xdf, 0xf4, 0xff, 0xff},
		}
	}
	if config.Image == nil {
		config.Image = buttonImage
	}
	if config.Padding.Left == 0 {
		config.Padding = widget.Insets{Left: 30, Right: 30, Top: 30, Bottom: 30}
	}

	opts := []widget.ButtonOpt{
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(config.MinWidth, config.MinHeight),
		),
		widget.ButtonOpts.Image(config.Image),
		widget.ButtonOpts.Text(config.Text, config.FontFace, config.TextColor),
		widget.ButtonOpts.TextPadding(config.Padding),
	}

	// Add layout data if provided
	if config.LayoutData != nil {
		opts = append(opts, widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(config.LayoutData),
		))
	}

	// Add click handler if provided
	if config.OnClick != nil {
		opts = append(opts, widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			config.OnClick()
		}))
	}

	return widget.NewButton(opts...)
}

// ListConfig provides declarative list configuration
type ListConfig struct {
	Entries         []interface{}
	EntryLabelFunc  func(interface{}) string
	OnEntrySelected func(interface{}) // Simplified callback
	MinWidth        int
	MinHeight       int
	LayoutData      interface{}
}

// CreateListWithConfig creates a list from config
func CreateListWithConfig(config ListConfig) *widget.List {
	// Apply defaults
	if config.MinWidth == 0 {
		config.MinWidth = 150
	}
	if config.MinHeight == 0 {
		config.MinHeight = 300
	}
	if config.EntryLabelFunc == nil {
		config.EntryLabelFunc = func(e interface{}) string {
			return fmt.Sprintf("%v", e)
		}
	}

	opts := []widget.ListOpt{
		widget.ListOpts.Entries(config.Entries),
		widget.ListOpts.EntryLabelFunc(config.EntryLabelFunc),
		widget.ListOpts.ContainerOpts(
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(config.MinWidth, config.MinHeight),
			),
		),
		widget.ListOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(ListRes.image),
		),
		widget.ListOpts.SliderOpts(
			widget.SliderOpts.Images(ListRes.track, ListRes.handle),
			widget.SliderOpts.MinHandleSize(ListRes.handleSize),
			widget.SliderOpts.TrackPadding(ListRes.trackPadding),
		),
		widget.ListOpts.EntryColor(ListRes.entry),
		widget.ListOpts.EntryFontFace(ListRes.face),
	}

	// Add layout data if provided
	if config.LayoutData != nil {
		opts = append(opts, widget.ListOpts.ContainerOpts(
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(config.LayoutData),
			),
		))
	}

	list := widget.NewList(opts...)

	// Add selection handler if provided
	if config.OnEntrySelected != nil {
		list.EntrySelectedEvent.AddHandler(func(args interface{}) {
			a := args.(*widget.ListEntrySelectedEventArgs)
			config.OnEntrySelected(a.Entry)
		})
	}

	return list
}

// PanelConfig provides declarative panel configuration
type PanelConfig struct {
	Title      string
	MinWidth   int
	MinHeight  int
	Background *e_image.NineSlice
	Padding    widget.Insets
	Layout     widget.Layouter // Row, Grid, Anchor, etc.
	LayoutData interface{}
}

// CreatePanelWithConfig creates a container panel from config
func CreatePanelWithConfig(config PanelConfig) *widget.Container {
	// Apply defaults
	if config.Background == nil {
		config.Background = PanelRes.image
	}
	if config.Padding.Left == 0 {
		config.Padding = widget.Insets{Left: 15, Right: 15, Top: 15, Bottom: 15}
	}
	if config.Layout == nil {
		config.Layout = widget.NewAnchorLayout()
	}

	opts := []widget.ContainerOpt{
		widget.ContainerOpts.BackgroundImage(config.Background),
		widget.ContainerOpts.Layout(config.Layout),
	}

	if config.MinWidth > 0 || config.MinHeight > 0 {
		opts = append(opts, widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(config.MinWidth, config.MinHeight),
		))
	}

	if config.LayoutData != nil {
		opts = append(opts, widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(config.LayoutData),
		))
	}

	container := widget.NewContainer(opts...)

	// Add title if provided
	if config.Title != "" {
		titleLabel := widget.NewText(
			widget.TextOpts.Text(config.Title, LargeFace, color.White),
		)
		container.AddChild(titleLabel)
	}

	return container
}

// TextInputConfig provides declarative text input configuration
type TextInputConfig struct {
	MinWidth        int
	MinHeight       int
	FontFace        font.Face
	TextColor       *widget.TextInputColor
	Image           *widget.TextInputImage
	Placeholder     string
	Padding         widget.Insets
	CaretSize       int
	OnChanged       func(text string)
	LayoutData      interface{}
}

// CreateTextInputWithConfig creates a text input from config
func CreateTextInputWithConfig(config TextInputConfig) *widget.TextInput {
	// Apply defaults
	if config.MinWidth == 0 {
		config.MinWidth = 300
	}
	if config.MinHeight == 0 {
		config.MinHeight = 50
	}
	if config.FontFace == nil {
		config.FontFace = SmallFace
	}
	if config.Image == nil {
		config.Image = &widget.TextInputImage{
			Idle:     defaultWidgetColor,
			Disabled: defaultWidgetColor,
		}
	}
	if config.TextColor == nil {
		config.TextColor = &widget.TextInputColor{
			Idle:     color.White,
			Disabled: hexToColor(textDisabledColor),
			Caret:    color.White,
		}
	}
	if config.Padding.Left == 0 {
		config.Padding = widget.Insets{Left: 15, Right: 15, Top: 10, Bottom: 10}
	}
	if config.CaretSize == 0 {
		config.CaretSize = 2
	}

	opts := []widget.TextInputOpt{
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(config.MinWidth, config.MinHeight),
		),
		widget.TextInputOpts.Image(config.Image),
		widget.TextInputOpts.Face(config.FontFace),
		widget.TextInputOpts.Color(config.TextColor),
		widget.TextInputOpts.Padding(config.Padding),
		widget.TextInputOpts.CaretOpts(
			widget.CaretOpts.Size(config.FontFace, config.CaretSize),
		),
	}

	// Add placeholder if provided
	if config.Placeholder != "" {
		opts = append(opts, widget.TextInputOpts.Placeholder(config.Placeholder))
	}

	// Add layout data if provided
	if config.LayoutData != nil {
		opts = append(opts, widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(config.LayoutData),
		))
	}

	input := widget.NewTextInput(opts...)

	// Add change handler if provided
	if config.OnChanged != nil {
		input.ChangedEvent.AddHandler(func(args interface{}) {
			a := args.(*widget.TextInputChangedEventArgs)
			config.OnChanged(a.InputText)
		})
	}

	return input
}

// SliderConfig provides declarative slider configuration
// Note: Slider support depends on ebitenui version
// This is a placeholder for future use when slider widgets are needed
type SliderConfig struct {
	MinWidth   int
	MinHeight  int
	MinValue   int
	MaxValue   int
	LayoutData interface{}
}

// CheckboxConfig provides declarative checkbox configuration
// Note: Checkbox support depends on ebitenui version
// This is a placeholder for future use when checkbox widgets are needed
type CheckboxConfig struct {
	LayoutData interface{}
}
