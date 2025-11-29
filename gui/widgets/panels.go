package widgets

import (
	"fmt"

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

	gridContainer := CreatePanelWithConfig(PanelConfig{
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
