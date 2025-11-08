package gui

import (
	"fmt"
	"image/color"

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

// SquadListConfig provides configuration for squad list panels
type SquadListConfig struct {
	SquadNames      []string
	OnSelect        func(squadName string, squadIndex int)
	WidthPercent    float64 // Width as percentage of screen width (default 0.15)
	HeightPercent   float64 // Height as percentage of screen height (default 0.5)
	Label           string  // Optional label text (default "Squads:")
}

// BuildSquadListPanel creates a left-side squad list with selection callback
// Returns the panel container and list widget
func (pb *PanelBuilders) BuildSquadListPanel(config SquadListConfig) (*widget.Container, *widget.List) {
	// Apply defaults
	if config.WidthPercent == 0 {
		config.WidthPercent = 0.15
	}
	if config.HeightPercent == 0 {
		config.HeightPercent = 0.5
	}
	if config.Label == "" {
		config.Label = "Squads:"
	}

	width := int(float64(pb.layout.ScreenWidth) * config.WidthPercent)
	height := int(float64(pb.layout.ScreenHeight) * config.HeightPercent)

	panel := CreatePanelWithConfig(PanelConfig{
		MinWidth:   width,
		MinHeight:  height,
		Background: PanelRes.image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 5, Right: 5, Top: 10, Bottom: 10}),
		),
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: widget.Insets{
				Left: int(float64(pb.layout.ScreenWidth) * 0.01),
			},
		},
	})

	// Add label
	listLabel := widget.NewText(
		widget.TextOpts.Text(config.Label, SmallFace, color.White),
	)
	panel.AddChild(listLabel)

	// Convert squad names to entries
	entries := make([]interface{}, len(config.SquadNames))
	for i, name := range config.SquadNames {
		entries[i] = name
	}

	// Create list
	squadList := CreateListWithConfig(ListConfig{
		Entries: entries,
		EntryLabelFunc: func(e interface{}) string {
			if str, ok := e.(string); ok {
				return str
			}
			return fmt.Sprintf("%v", e)
		},
		OnEntrySelected: func(selectedEntry interface{}) {
			if config.OnSelect != nil {
				if name, ok := selectedEntry.(string); ok {
					// Find index of selected squad
					for i, squadName := range config.SquadNames {
						if squadName == name {
							config.OnSelect(name, i)
							break
						}
					}
				}
			}
		},
	})

	panel.AddChild(squadList)
	return panel, squadList
}

// DetailPanelConfig provides configuration for detail panels
type DetailPanelConfig struct {
	InitialText     string
	WidthPercent    float64 // Width as percentage of screen width (leave 0 for default 0.45)
	HeightPercent   float64 // Height as percentage of screen height (leave 0 for default 0.75)
	PaddingRight    float64 // Padding as percentage (leave 0 for default 0.02)
}

// BuildDetailPanel creates a detail panel (typically right-side) for showing selected item/squad info
// Returns the panel container and text area
func (pb *PanelBuilders) BuildDetailPanel(config DetailPanelConfig) (*widget.Container, *widget.TextArea) {
	// Apply defaults
	if config.WidthPercent == 0 {
		config.WidthPercent = 0.45
	}
	if config.HeightPercent == 0 {
		config.HeightPercent = 0.75
	}
	if config.PaddingRight == 0 {
		config.PaddingRight = 0.02
	}

	panelWidth := int(float64(pb.layout.ScreenWidth) * config.WidthPercent)
	panelHeight := int(float64(pb.layout.ScreenHeight) * config.HeightPercent)

	panel := CreatePanelWithConfig(PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: PanelRes.image,
		Layout:     widget.NewAnchorLayout(),
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionEnd,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: widget.Insets{
				Right: int(float64(pb.layout.ScreenWidth) * config.PaddingRight),
			},
		},
	})

	// Detail text area
	detailConfig := TextAreaConfig{
		MinWidth:  panelWidth - 20,
		MinHeight: panelHeight - 20,
		FontColor: color.White,
	}
	textArea := CreateTextAreaWithConfig(detailConfig)
	if config.InitialText != "" {
		textArea.SetText(config.InitialText)
	}

	panel.AddChild(textArea)
	return panel, textArea
}

// TopInstructionTextConfig provides configuration for top instruction text
type TopInstructionTextConfig struct {
	Text          string
	TopPadding    float64 // Padding as percentage (default 0.02)
}

// BuildTopInstructionText creates centered instruction text at the top of the screen
// Returns the text widget
func (pb *PanelBuilders) BuildTopInstructionText(config TopInstructionTextConfig) *widget.Text {
	if config.TopPadding == 0 {
		config.TopPadding = 0.02
	}

	instructionText := widget.NewText(
		widget.TextOpts.Text(config.Text, SmallFace, color.White),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				Padding: widget.Insets{
					Top: int(float64(pb.layout.ScreenHeight) * config.TopPadding),
				},
			}),
		),
	)

	return instructionText
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
