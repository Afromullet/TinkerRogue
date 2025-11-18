package gui

import (
	"fmt"

	"github.com/ebitenui/ebitenui/widget"
)

// PanelBuilders provides high-level UI composition functions to reduce duplication
// across UI modes. Each builder encapsulates common UI patterns.
type PanelBuilders struct {
	layout      *LayoutConfig
	modeManager *UIModeManager
}

// NewPanelBuilders creates a new PanelBuilders instance
func NewPanelBuilders(layout *LayoutConfig, modeManager *UIModeManager) *PanelBuilders {
	return &PanelBuilders{
		layout:      layout,
		modeManager: modeManager,
	}
}

// NOTE: All old-style panel building methods have been removed and replaced by the
// BuildPanel() method with functional options. See panelconfig.go for the new approach.
// Old LayoutConfig methods (TopRightPanel, BottomRightPanel, etc.) have also been
// removed - use BuildPanel with position options instead (TopRight(), BottomCenter(), etc.).

// GridEditorConfig provides configuration for 3x3 grid editors
type GridEditorConfig struct {
	CellTextFormat func(row, col int) string           // Function to generate cell text
	OnCellClick    func(row, col int)                  // Callback when cell is clicked
	Padding        widget.Insets                       // Padding for grid (default: 10 all sides)
}

// BuildGridEditor creates a centered 3x3 grid panel for squad formations
// Returns the grid container and a 3x3 array of the cell buttons
func (pb *PanelBuilders) BuildGridEditor(config GridEditorConfig) (*widget.Container, [3][3]*widget.Button) {
	// Apply defaults
	if config.Padding.Left == 0 && config.Padding.Right == 0 {
		config.Padding = widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}
	}
	if config.CellTextFormat == nil {
		config.CellTextFormat = func(row, col int) string {
			return fmt.Sprintf("[%d,%d]", row, col)
		}
	}

	gridContainer := CreatePanelWithConfig(PanelConfig{
		Background: PanelRes.image,
		Layout: widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true, true, true}),
			widget.GridLayoutOpts.Spacing(5, 5),
			widget.GridLayoutOpts.Padding(config.Padding),
		),
	})

	var gridCells [3][3]*widget.Button

	// Create 3x3 grid buttons using ButtonConfig pattern for consistency
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cellRow, cellCol := row, col // Capture for closure
			cellText := config.CellTextFormat(row, col)

			cellBtn := CreateButtonWithConfig(ButtonConfig{
				Text:      cellText,
				FontFace:  SmallFace,
				MinWidth:  0, // Let grid layout handle sizing
				MinHeight: 0, // Let grid layout handle sizing
				Padding:   widget.Insets{Left: 8, Right: 8, Top: 8, Bottom: 8},
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
