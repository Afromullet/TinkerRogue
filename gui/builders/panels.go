package builders

import (
	"fmt"
	"image/color"

	"game_main/gui/guiresources"
	"game_main/gui/specs"

	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

// ============================================
// PANEL OPTION FUNCTIONAL CONFIGURATION
// ============================================

// PanelOption is a functional option for configuring panel creation.
// This pattern allows composable, self-documenting panel configuration.
type PanelOption func(*panelConfig)

// panelConfig holds the configuration for building a panel.
// This is an internal type that gets populated by PanelOptions.
type panelConfig struct {
	widthPercent         float64
	heightPercent        float64
	paddingPercent       float64
	title                string
	background           *e_image.NineSlice
	layout               widget.Layouter
	layoutData           widget.AnchorLayoutData
	customPadding        *widget.Insets   // For fine-grained control
	useResponsivePadding bool             // Flag to apply responsive padding to row layouts
	rowLayoutDirection   widget.Direction // Direction for row layouts when using responsive padding
	enableCaching        bool             // Whether to use cached background rendering (default: true for all panels)
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
// Note: Padding will be calculated responsively by BuildPanel based on screen size.
func RowLayout() PanelOption {
	return func(pc *panelConfig) {
		// Mark that we want responsive padding to be applied by BuildPanel
		pc.useResponsivePadding = true
		pc.rowLayoutDirection = widget.DirectionVertical
		pc.layout = widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			// Padding will be set by BuildPanel using PaddingExtraSmall
		)
	}
}

// HorizontalRowLayout sets the panel to use a horizontal row layout (stacks children horizontally).
// Note: Padding will be calculated responsively by BuildPanel based on screen size.
func HorizontalRowLayout() PanelOption {
	return func(pc *panelConfig) {
		// Mark that we want responsive padding to be applied by BuildPanel
		pc.useResponsivePadding = true
		pc.rowLayoutDirection = widget.DirectionHorizontal
		pc.layout = widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			// Padding will be set by BuildPanel using PaddingExtraSmall
		)
	}
}

// AnchorLayout sets the panel to use an anchor layout (allows absolute positioning of children).
func AnchorLayout() PanelOption {
	return func(pc *panelConfig) {
		pc.layout = widget.NewAnchorLayout()
	}
}

// ============================================
// PANEL BUILDERS
// ============================================

// PanelBuilders provides high-level UI composition functions to reduce duplication
// across UI modes. Each builder encapsulates common UI patterns.
type PanelBuilders struct {
	Layout *specs.LayoutConfig // Exported for external access
}

// NewPanelBuilders creates a new PanelBuilders instance
func NewPanelBuilders(layout *specs.LayoutConfig) *PanelBuilders {
	return &PanelBuilders{
		Layout: layout,
	}
}

// BuildPanel creates a panel with functional options.

// Example usage:
//
//	panel := pb.BuildPanel(
//	    TopCenter(),
//	    Size(0.4, 0.08),
//	    Padding(0.01),
//	    HorizontalRowLayout(),
//	)
func (pb *PanelBuilders) BuildPanel(opts ...PanelOption) *widget.Container {
	// Calculate default responsive padding
	defaultPadding := int(float64(pb.Layout.ScreenWidth) * specs.PaddingExtraSmall)

	// Apply defaults
	config := panelConfig{
		widthPercent:  0.2,
		heightPercent: 0.3,
		background:    guiresources.PanelRes.Image,
		layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: defaultPadding, Right: defaultPadding, Top: defaultPadding, Bottom: defaultPadding}),
		),
		layoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		},
		enableCaching: true, // Enable caching by default for all panels (reduces NineSlice allocations)
	}

	// Apply all options
	for _, opt := range opts {
		opt(&config)
	}

	// Apply responsive padding to row layouts if requested
	if config.useResponsivePadding {
		padding := int(float64(pb.Layout.ScreenWidth) * specs.PaddingExtraSmall)

		// Recreate row layout with responsive padding using stored direction
		if _, ok := config.layout.(*widget.RowLayout); ok {
			// Determine spacing based on direction
			spacing := 5
			if config.rowLayoutDirection == widget.DirectionHorizontal {
				spacing = 10
			}

			config.layout = widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(config.rowLayoutDirection),
				widget.RowLayoutOpts.Spacing(spacing),
				widget.RowLayoutOpts.Padding(widget.Insets{Left: padding, Right: padding, Top: padding, Bottom: padding}),
			)
		}
	}

	// Calculate actual dimensions from percentages
	width := int(float64(pb.Layout.ScreenWidth) * config.widthPercent)
	height := int(float64(pb.Layout.ScreenHeight) * config.heightPercent)

	// Apply padding to layoutData
	if config.customPadding != nil {
		// Use custom padding if provided
		config.layoutData.Padding = *config.customPadding
	} else if config.paddingPercent > 0 {
		// Calculate padding from percentage
		hPadding := int(float64(pb.Layout.ScreenWidth) * config.paddingPercent)
		vPadding := int(float64(pb.Layout.ScreenHeight) * config.paddingPercent)

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

	// Pre-cache background if caching is enabled
	// This warms the cache so the first render is faster
	if config.enableCaching && config.background != nil {
		_ = guiresources.GetPanelBackground(width, height)
	}

	// Build the panel
	panel := CreatePanelWithConfig(ContainerConfig{
		MinWidth:   width,
		MinHeight:  height,
		Background: config.background,
		Layout:     config.layout,
		LayoutData: config.layoutData,
	})

	return panel
}

// ============================================
// TYPED PANEL BUILDING
// ============================================

// PanelType defines the type of panel to build with BuildTypedPanel
type PanelType int

const (
	PanelTypeSimple PanelType = iota // Container only, no children
	PanelTypeDetail                  // Container + TextArea (for detail views)
	PanelTypeList                    // Container + List widget
	PanelTypeGrid                    // Container + Grid layout (use BuildGridEditor instead)
)

// TypedPanelConfig provides configuration for BuildTypedPanel
type TypedPanelConfig struct {
	// Panel type determines what content is added
	Type PanelType

	// SpecName from StandardPanels (optional - if empty, use manual options)
	SpecName string

	// Manual panel options (when SpecName is empty)
	Position PanelOption
	Size     PanelOption
	Layout   PanelOption
	Padding  PanelOption

	// Detail panel options (for PanelTypeDetail)
	DetailText string // Initial text for detail panels

	// List panel options (for PanelTypeList)
	ListConfig *ListConfig // Complete list configuration
}

// TypedPanelResult contains the panel and any created widgets
type TypedPanelResult struct {
	Panel    *widget.Container
	TextArea *widget.TextArea // For PanelTypeDetail
	List     *widget.List     // For PanelTypeList
}

// BuildTypedPanel creates a panel with standard content based on panel type.
//
// Example usage:
//
//	// Simple panel from spec
//	result := pb.BuildTypedPanel(TypedPanelConfig{
//	    Type:     PanelTypeSimple,
//	    SpecName: "turn_order",
//	})
//
//	// Detail panel from spec
//	result := pb.BuildTypedPanel(TypedPanelConfig{
//	    Type:       PanelTypeDetail,
//	    SpecName:   "inventory_detail",
//	    DetailText: "Select an item",
//	})
//	textArea := result.TextArea
//
//	// Manual panel construction
//	result := pb.BuildTypedPanel(TypedPanelConfig{
//	    Type:     PanelTypeSimple,
//	    Position: TopCenter(),
//	    Size:     Size(0.5, 0.3),
//	    Layout:   HorizontalRowLayout(),
//	    Padding:  Padding(specs.PaddingStandard),
//	})
func (pb *PanelBuilders) BuildTypedPanel(config TypedPanelConfig) TypedPanelResult {
	var opts []PanelOption

	// Build options from spec or manual configuration
	if config.SpecName != "" {
		// Use StandardPanels spec
		spec, exists := StandardPanels[config.SpecName]
		if !exists {
			return TypedPanelResult{Panel: nil}
		}

		opts = []PanelOption{
			spec.Position,
			Size(spec.Width, spec.Height),
			spec.Layout,
		}

		// Add padding option
		if spec.Custom != nil {
			opts = append(opts, CustomPadding(*spec.Custom))
		} else {
			opts = append(opts, Padding(spec.Padding))
		}
	} else {
		// Use manual options
		if config.Position != nil {
			opts = append(opts, config.Position)
		}
		if config.Size != nil {
			opts = append(opts, config.Size)
		}
		if config.Layout != nil {
			opts = append(opts, config.Layout)
		}
		if config.Padding != nil {
			opts = append(opts, config.Padding)
		}
	}

	// Build the base panel
	panel := pb.BuildPanel(opts...)

	result := TypedPanelResult{Panel: panel}

	// Add typed content based on panel type
	switch config.Type {
	case PanelTypeSimple:
		// No additional content needed

	case PanelTypeDetail:
		// Add TextArea to panel
		spec := StandardPanels[config.SpecName]
		panelWidth := int(float64(pb.Layout.ScreenWidth) * spec.Width)
		panelHeight := int(float64(pb.Layout.ScreenHeight) * spec.Height)

		textArea := CreateTextAreaWithConfig(TextAreaConfig{
			MinWidth:  panelWidth - 20,
			MinHeight: panelHeight - 20,
			FontColor: color.White,
		})
		textArea.SetText(config.DetailText)
		panel.AddChild(textArea)
		result.TextArea = textArea

	case PanelTypeList:
		// Add List to panel
		if config.ListConfig != nil {
			list := CreateListWithConfig(*config.ListConfig)
			panel.AddChild(list)
			result.List = list
		}

	case PanelTypeGrid:
		// Grid panels should use BuildGridEditor instead
		// This is just a fallback - no content added
	}

	return result
}

// ============================================
// GRID EDITOR
// ============================================

// GridEditorConfig provides configuration for 3x3 grid editors
type GridEditorConfig struct {
	CellTextFormat func(row, col int) string // Function to generate cell text
	OnCellClick    func(row, col int)        // Callback when cell is clicked
	Padding        widget.Insets             // Padding for grid (default: 10 all sides)
}

// BuildGridEditor creates a centered 3x3 grid panel for squad formations
// Returns the grid container and a 3x3 array of the cell buttons
func (pb *PanelBuilders) BuildGridEditor(config GridEditorConfig) (*widget.Container, [3][3]*widget.Button) {
	// Calculate responsive padding
	defaultPadding := int(float64(pb.Layout.ScreenWidth) * specs.PaddingExtraSmall)

	// Apply defaults
	if config.Padding.Left == 0 && config.Padding.Right == 0 {
		config.Padding = widget.Insets{Left: defaultPadding, Right: defaultPadding, Top: defaultPadding, Bottom: defaultPadding}
	}
	if config.CellTextFormat == nil {
		config.CellTextFormat = func(row, col int) string {
			return fmt.Sprintf("[%d,%d]", row, col)
		}
	}

	// Grid editors are static - enable caching
	gridContainer := CreateStaticPanel(ContainerConfig{
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true, true, true}),
			widget.GridLayoutOpts.Spacing(5, 5),
			widget.GridLayoutOpts.Padding(config.Padding),
		),
	})

	var gridCells [3][3]*widget.Button

	// Calculate responsive cell padding (slightly smaller than container padding)
	cellPadding := int(float64(pb.Layout.ScreenWidth) * (specs.PaddingExtraSmall * 0.8))

	// Create 3x3 grid buttons using ButtonConfig pattern for consistency
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cellRow, cellCol := row, col // Capture for closure
			cellText := config.CellTextFormat(row, col)

			cellBtn := CreateButtonWithConfig(ButtonConfig{
				Text:      cellText,
				FontFace:  guiresources.SmallFace,
				MinWidth:  0, // Let grid layout handle sizing
				MinHeight: 0, // Let grid layout handle sizing
				Padding:   widget.Insets{Left: cellPadding, Right: cellPadding, Top: cellPadding, Bottom: cellPadding},
				OnClick: func() {
					if config.OnCellClick != nil {
						config.OnCellClick(cellRow, cellCol)
					}
				},
			})

			gridCells[row][col] = cellBtn
			gridContainer.AddChild(cellBtn)
		}
	}

	// Center positioning
	gridContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}

	return gridContainer, gridCells
}

// CreateStaticPanel creates a panel optimized for static content (enables caching and pre-rendering).
// Use this for panels that:
// - Have fixed dimensions
// - Don't change size frequently
// - Are visible most of the time
//
// Examples: squad management panels, combat UI panels, stats displays
func CreateStaticPanel(config ContainerConfig) *widget.Container {
	config.EnableCaching = true
	return CreatePanelWithConfig(config)
}
