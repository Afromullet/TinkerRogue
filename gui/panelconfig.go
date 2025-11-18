package gui

import (
	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

// PanelOption is a functional option for configuring panel creation.
// This pattern allows composable, self-documenting panel configuration.
type PanelOption func(*panelConfig)

// panelConfig holds the configuration for building a panel.
// This is an internal type that gets populated by PanelOptions.
type panelConfig struct {
	widthPercent   float64
	heightPercent  float64
	paddingPercent float64
	title          string
	background     *e_image.NineSlice
	layout         widget.Layouter
	layoutData     widget.AnchorLayoutData
	customPadding  *widget.Insets // For fine-grained control
}

// Predefined positioning options

// TopCenter positions a panel at the top-center of the screen.
func TopCenter() PanelOption {
	return func(pc *panelConfig) {
		pc.layoutData.HorizontalPosition = widget.AnchorLayoutPositionCenter
		pc.layoutData.VerticalPosition = widget.AnchorLayoutPositionStart
	}
}

// TopLeft positions a panel at the top-left of the screen.
func TopLeft() PanelOption {
	return func(pc *panelConfig) {
		pc.layoutData.HorizontalPosition = widget.AnchorLayoutPositionStart
		pc.layoutData.VerticalPosition = widget.AnchorLayoutPositionStart
	}
}

// TopRight positions a panel at the top-right of the screen.
func TopRight() PanelOption {
	return func(pc *panelConfig) {
		pc.layoutData.HorizontalPosition = widget.AnchorLayoutPositionEnd
		pc.layoutData.VerticalPosition = widget.AnchorLayoutPositionStart
	}
}

// LeftCenter positions a panel at the left-center of the screen.
func LeftCenter() PanelOption {
	return func(pc *panelConfig) {
		pc.layoutData.HorizontalPosition = widget.AnchorLayoutPositionStart
		pc.layoutData.VerticalPosition = widget.AnchorLayoutPositionCenter
	}
}

// LeftBottom positions a panel at the left-bottom of the screen.
func LeftBottom() PanelOption {
	return func(pc *panelConfig) {
		pc.layoutData.HorizontalPosition = widget.AnchorLayoutPositionStart
		pc.layoutData.VerticalPosition = widget.AnchorLayoutPositionEnd
	}
}

// RightCenter positions a panel at the right-center of the screen.
func RightCenter() PanelOption {
	return func(pc *panelConfig) {
		pc.layoutData.HorizontalPosition = widget.AnchorLayoutPositionEnd
		pc.layoutData.VerticalPosition = widget.AnchorLayoutPositionCenter
	}
}

// BottomCenter positions a panel at the bottom-center of the screen.
func BottomCenter() PanelOption {
	return func(pc *panelConfig) {
		pc.layoutData.HorizontalPosition = widget.AnchorLayoutPositionCenter
		pc.layoutData.VerticalPosition = widget.AnchorLayoutPositionEnd
	}
}

// BottomLeft positions a panel at the bottom-left of the screen.
func BottomLeft() PanelOption {
	return func(pc *panelConfig) {
		pc.layoutData.HorizontalPosition = widget.AnchorLayoutPositionStart
		pc.layoutData.VerticalPosition = widget.AnchorLayoutPositionEnd
	}
}

// BottomRight positions a panel at the bottom-right of the screen.
func BottomRight() PanelOption {
	return func(pc *panelConfig) {
		pc.layoutData.HorizontalPosition = widget.AnchorLayoutPositionEnd
		pc.layoutData.VerticalPosition = widget.AnchorLayoutPositionEnd
	}
}

// Center positions a panel at the center of the screen.
func Center() PanelOption {
	return func(pc *panelConfig) {
		pc.layoutData.HorizontalPosition = widget.AnchorLayoutPositionCenter
		pc.layoutData.VerticalPosition = widget.AnchorLayoutPositionCenter
	}
}

// Size configuration options

// Size sets the panel width and height as percentages of screen size.
// widthPercent and heightPercent should be values between 0.0 and 1.0.
//
// Example: Size(0.4, 0.08) creates a panel 40% of screen width, 8% of screen height.
func Size(widthPercent, heightPercent float64) PanelOption {
	return func(pc *panelConfig) {
		pc.widthPercent = widthPercent
		pc.heightPercent = heightPercent
	}
}

// Padding configuration options

// Padding sets uniform padding around the panel as a percentage of screen dimensions.
// The percentage is applied to both screen width (for left/right) and screen height (for top/bottom).
//
// Example: Padding(0.01) creates 1% screen padding on all sides.
func Padding(percent float64) PanelOption {
	return func(pc *panelConfig) {
		pc.paddingPercent = percent
	}
}

// CustomPadding sets specific padding values for each side.
// This overrides any padding set by Padding() option.
func CustomPadding(insets widget.Insets) PanelOption {
	return func(pc *panelConfig) {
		pc.customPadding = &insets
	}
}

// Layout configuration options

// RowLayout sets the panel to use a vertical row layout (stacks children vertically).
func RowLayout() PanelOption {
	return func(pc *panelConfig) {
		pc.layout = widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}),
		)
	}
}

// HorizontalRowLayout sets the panel to use a horizontal row layout (stacks children horizontally).
func HorizontalRowLayout() PanelOption {
	return func(pc *panelConfig) {
		pc.layout = widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}),
		)
	}
}

// AnchorLayout sets the panel to use an anchor layout (allows absolute positioning of children).
func AnchorLayout() PanelOption {
	return func(pc *panelConfig) {
		pc.layout = widget.NewAnchorLayout()
	}
}

// Content options

// WithTitle sets a title for the panel (currently stored but not rendered - for future use).
func WithTitle(title string) PanelOption {
	return func(pc *panelConfig) {
		pc.title = title
	}
}

// WithBackground sets a custom background for the panel.
func WithBackground(background *e_image.NineSlice) PanelOption {
	return func(pc *panelConfig) {
		pc.background = background
	}
}

// BuildPanel creates a panel with functional options.
// This replaces the multiple BuildXxxPanel functions with a single composable method.
//
// Example usage:
//
//	panel := pb.BuildPanel(
//	    TopCenter(),
//	    Size(0.4, 0.08),
//	    Padding(0.01),
//	    HorizontalRowLayout(),
//	)
func (pb *PanelBuilders) BuildPanel(opts ...PanelOption) *widget.Container {
	// Apply defaults
	config := panelConfig{
		widthPercent:  0.2,
		heightPercent: 0.3,
		background:    PanelRes.image,
		layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}),
		),
		layoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		},
	}

	// Apply all options
	for _, opt := range opts {
		opt(&config)
	}

	// Calculate actual dimensions from percentages
	width := int(float64(pb.layout.ScreenWidth) * config.widthPercent)
	height := int(float64(pb.layout.ScreenHeight) * config.heightPercent)

	// Apply padding to layoutData
	if config.customPadding != nil {
		// Use custom padding if provided
		config.layoutData.Padding = *config.customPadding
	} else if config.paddingPercent > 0 {
		// Calculate padding from percentage
		hPadding := int(float64(pb.layout.ScreenWidth) * config.paddingPercent)
		vPadding := int(float64(pb.layout.ScreenHeight) * config.paddingPercent)

		// Apply padding based on position
		insets := widget.Insets{}

		// Horizontal padding
		switch config.layoutData.HorizontalPosition {
		case widget.AnchorLayoutPositionStart:
			insets.Left = hPadding
		case widget.AnchorLayoutPositionEnd:
			insets.Right = hPadding
		case widget.AnchorLayoutPositionCenter:
			// No horizontal padding for centered
		}

		// Vertical padding
		switch config.layoutData.VerticalPosition {
		case widget.AnchorLayoutPositionStart:
			insets.Top = vPadding
		case widget.AnchorLayoutPositionEnd:
			insets.Bottom = vPadding
		case widget.AnchorLayoutPositionCenter:
			// No vertical padding for centered
		}

		config.layoutData.Padding = insets
	}

	// Build the panel
	panel := CreatePanelWithConfig(PanelConfig{
		MinWidth:   width,
		MinHeight:  height,
		Background: config.background,
		Layout:     config.layout,
		LayoutData: config.layoutData,
	})

	return panel
}
