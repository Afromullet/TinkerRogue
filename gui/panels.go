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

// BuildCloseButton creates a bottom-center close button that transitions to a target mode
// Returns the container holding the close button
func (pb *PanelBuilders) BuildCloseButton(targetModeName string, buttonText string) *widget.Container {
	if buttonText == "" {
		buttonText = "Close (ESC)"
	}

	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	closeBtn := CreateButtonWithConfig(ButtonConfig{
		Text: buttonText,
		OnClick: func() {
			if targetMode, exists := pb.modeManager.GetMode(targetModeName); exists {
				pb.modeManager.RequestTransition(targetMode, "Close button pressed")
			}
		},
	})

	buttonContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionEnd,
		Padding: widget.Insets{
			Bottom: int(float64(pb.layout.ScreenHeight) * 0.08),
		},
	}

	buttonContainer.AddChild(closeBtn)
	return buttonContainer
}

// BuildStatsPanel creates a top-right stats panel with initial content
// Returns the panel container and text area for dynamic updates
func (pb *PanelBuilders) BuildStatsPanel(content string) (*widget.Container, *widget.TextArea) {
	x, y, width, height := pb.layout.TopRightPanel()

	panel := CreatePanelWithConfig(PanelConfig{
		MinWidth:   width,
		MinHeight:  height,
		Background: PanelRes.image,
		Layout: widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.Insets{
				Left: 10, Right: 10, Top: 10, Bottom: 10,
			}),
		),
	})

	textArea := CreateTextAreaWithConfig(TextAreaConfig{
		MinWidth:  width - 20,
		MinHeight: height - 20,
		FontColor: color.White,
	})
	textArea.SetText(content)

	panel.AddChild(textArea)
	SetContainerLocation(panel, x, y)

	return panel, textArea
}

// BuildMessageLog creates a bottom-right message log panel
// Returns the panel container and text area for appending messages
func (pb *PanelBuilders) BuildMessageLog() (*widget.Container, *widget.TextArea) {
	x, y, width, height := pb.layout.BottomRightPanel()

	logConfig := TextAreaConfig{
		MinWidth:  width - 20,
		MinHeight: height - 20,
		FontColor: color.White,
	}
	messageLog := CreateTextAreaWithConfig(logConfig)

	logContainer := CreatePanelWithConfig(PanelConfig{
		MinWidth:   width,
		MinHeight:  height,
		Background: PanelRes.image,
		Layout:     widget.NewAnchorLayout(),
	})
	logContainer.AddChild(messageLog)

	SetContainerLocation(logContainer, x, y)

	return logContainer, messageLog
}

// BuildRightSidePanel creates a right-side panel for logs or info
// Returns the panel container and text area
func (pb *PanelBuilders) BuildRightSidePanel(initialText string) (*widget.Container, *widget.TextArea) {
	_, _, width, height := pb.layout.RightSidePanel()

	logConfig := TextAreaConfig{
		MinWidth:  width - 20,
		MinHeight: height - 20,
		FontColor: color.White,
	}
	textArea := CreateTextAreaWithConfig(logConfig)
	textArea.SetText(initialText)

	container := CreatePanelWithConfig(PanelConfig{
		MinWidth:   width,
		MinHeight:  height,
		Background: PanelRes.image,
		Layout:     widget.NewAnchorLayout(),
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionEnd,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: widget.Insets{
				Right: int(float64(pb.layout.ScreenWidth) * 0.01),
			},
		},
	})
	container.AddChild(textArea)

	return container, textArea
}

// BuildActionButtons creates a bottom-center button row from provided buttons
// Returns the container holding all buttons
func (pb *PanelBuilders) BuildActionButtons(buttons []*widget.Button) *widget.Container {
	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				Padding: widget.Insets{
					Bottom: int(float64(pb.layout.ScreenHeight) * 0.08),
				},
			}),
		),
	)

	for _, btn := range buttons {
		buttonContainer.AddChild(btn)
	}

	return buttonContainer
}

// BuildBottomCenterButtons creates a bottom-center button row container (without pre-populated buttons)
// Returns empty container positioned at bottom-center for manual button addition
func (pb *PanelBuilders) BuildBottomCenterButtons() *widget.Container {
	x, y := pb.layout.BottomCenterButtons()

	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10}),
		)),
	)

	SetContainerLocation(buttonContainer, x, y)
	return buttonContainer
}

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

// BuildTopCenterPanel creates a top-center panel with specified dimensions
// Returns the panel container
func (pb *PanelBuilders) BuildTopCenterPanel(widthPercent, heightPercent, topPadding float64) *widget.Container {
	width := int(float64(pb.layout.ScreenWidth) * widthPercent)
	height := int(float64(pb.layout.ScreenHeight) * heightPercent)

	panel := CreatePanelWithConfig(PanelConfig{
		MinWidth:   width,
		MinHeight:  height,
		Background: PanelRes.image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}),
		),
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
			Padding: widget.Insets{
				Top: int(float64(pb.layout.ScreenHeight) * topPadding),
			},
		},
	})

	return panel
}

// BuildTopLeftPanel creates a top-left panel with specified dimensions
// Returns the panel container
func (pb *PanelBuilders) BuildTopLeftPanel(widthPercent, heightPercent, topPadding, leftPadding float64) *widget.Container {
	width := int(float64(pb.layout.ScreenWidth) * widthPercent)
	height := int(float64(pb.layout.ScreenHeight) * heightPercent)

	panel := CreatePanelWithConfig(PanelConfig{
		MinWidth:   width,
		MinHeight:  height,
		Background: PanelRes.image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}),
		),
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
			Padding: widget.Insets{
				Top:  int(float64(pb.layout.ScreenHeight) * topPadding),
				Left: int(float64(pb.layout.ScreenWidth) * leftPadding),
			},
		},
	})

	return panel
}

// BuildLeftSidePanel creates a left-side panel with specified dimensions and positioning
// Returns the panel container
func (pb *PanelBuilders) BuildLeftSidePanel(widthPercent, heightPercent, leftPadding float64, verticalPos widget.AnchorLayoutPosition) *widget.Container {
	width := int(float64(pb.layout.ScreenWidth) * widthPercent)
	height := int(float64(pb.layout.ScreenHeight) * heightPercent)

	panel := CreatePanelWithConfig(PanelConfig{
		MinWidth:   width,
		MinHeight:  height,
		Background: PanelRes.image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}),
		),
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   verticalPos,
			Padding: widget.Insets{
				Left: int(float64(pb.layout.ScreenWidth) * leftPadding),
			},
		},
	})

	return panel
}

// BuildLeftBottomPanel creates a left-bottom panel with specified dimensions
// Returns the panel container
func (pb *PanelBuilders) BuildLeftBottomPanel(widthPercent, heightPercent, leftPadding, bottomPadding float64) *widget.Container {
	width := int(float64(pb.layout.ScreenWidth) * widthPercent)
	height := int(float64(pb.layout.ScreenHeight) * heightPercent)

	panel := CreatePanelWithConfig(PanelConfig{
		MinWidth:   width,
		MinHeight:  height,
		Background: PanelRes.image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}),
		),
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionEnd,
			Padding: widget.Insets{
				Left:   int(float64(pb.layout.ScreenWidth) * leftPadding),
				Bottom: int(float64(pb.layout.ScreenHeight) * bottomPadding),
			},
		},
	})

	return panel
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

	// Create 3x3 grid buttons
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cellRow, cellCol := row, col // Capture for closure
			cellText := config.CellTextFormat(row, col)

			cellBtn := widget.NewButton(
				widget.ButtonOpts.Image(buttonImage),
				widget.ButtonOpts.Text(cellText, SmallFace, &widget.ButtonTextColor{
					Idle: color.NRGBA{0xdf, 0xf4, 0xff, 0xff},
				}),
				widget.ButtonOpts.TextPadding(widget.Insets{
					Left: 8, Right: 8, Top: 8, Bottom: 8,
				}),
				widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
					if config.OnCellClick != nil {
						config.OnCellClick(cellRow, cellCol)
					}
				}),
			)

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
