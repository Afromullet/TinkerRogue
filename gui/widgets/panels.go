package widgets

import (
	"fmt"
	"image/color"

	"game_main/gui/core"
	"game_main/gui/guiresources"

	"github.com/ebitenui/ebitenui/widget"
)

// PanelBuilders provides high-level UI composition functions to reduce duplication
// across UI modes. Each builder encapsulates common UI patterns.
type PanelBuilders struct {
	Layout      *LayoutConfig        // Exported for external access
	modeManager *core.UIModeManager
}

// NewPanelBuilders creates a new PanelBuilders instance
func NewPanelBuilders(layout *LayoutConfig, modeManager *core.UIModeManager) *PanelBuilders {
	return &PanelBuilders{
		Layout:      layout,
		modeManager: modeManager,
	}
}

// NOTE: All old-style panel building methods have been removed and replaced by the
// BuildPanel() method with functional options. See panelconfig.go for the new approach.
// Old LayoutConfig methods (TopRightPanel, BottomRightPanel, etc.) have also been
// removed - use BuildPanel with position options instead (TopRight(), BottomCenter(), etc.).

// PanelType defines the type of panel to build with BuildTypedPanel
type PanelType int

const (
	PanelTypeSimple PanelType = iota // Container only, no children
	PanelTypeDetail                   // Container + TextArea (for detail views)
	PanelTypeList                     // Container + List widget
	PanelTypeGrid                     // Container + Grid layout (use BuildGridEditor instead)
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
// This consolidates all panel creation patterns into a single builder method.
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
//	    Padding:  Padding(PaddingStandard),
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

// GridEditorConfig provides configuration for 3x3 grid editors
type GridEditorConfig struct {
	CellTextFormat func(row, col int) string           // Function to generate cell text
	OnCellClick    func(row, col int)                  // Callback when cell is clicked
	Padding        widget.Insets                       // Padding for grid (default: 10 all sides)
}

// BuildGridEditor creates a centered 3x3 grid panel for squad formations
// Returns the grid container and a 3x3 array of the cell buttons
func (pb *PanelBuilders) BuildGridEditor(config GridEditorConfig) (*widget.Container, [3][3]*widget.Button) {
	// Calculate responsive padding
	defaultPadding := int(float64(pb.Layout.ScreenWidth) * PaddingExtraSmall)

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
	gridContainer := CreateStaticPanel(PanelConfig{
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
	cellPadding := int(float64(pb.Layout.ScreenWidth) * (PaddingExtraSmall * 0.8))

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
