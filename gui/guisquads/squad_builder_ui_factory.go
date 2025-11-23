package guisquads

import (
	"game_main/gui/guiresources"
	"game_main/gui/widgets"

	"fmt"
	"game_main/squads"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
)

// SquadBuilderUIFactory creates UI components for the squad builder
type SquadBuilderUIFactory struct {
	layout        *widgets.LayoutConfig
	panelBuilders *widgets.PanelBuilders
}

// NewSquadBuilderUIFactory creates a new UI factory for squad builder
func NewSquadBuilderUIFactory(layout *widgets.LayoutConfig, panelBuilders *widgets.PanelBuilders) *SquadBuilderUIFactory {
	return &SquadBuilderUIFactory{
		layout:        layout,
		panelBuilders: panelBuilders,
	}
}

// CreateGridPanel builds the 3x3 grid editor panel and returns button grid
func (sbuf *SquadBuilderUIFactory) CreateGridPanel(onCellClick func(row, col int)) (*widget.Container, [3][3]*widget.Button) {
	var buttons [3][3]*widget.Button
	gridContainer, buttons := sbuf.panelBuilders.BuildGridEditor(widgets.GridEditorConfig{
		CellTextFormat: func(row, col int) string {
			return fmt.Sprintf("Empty\n[%d,%d]", row, col)
		},
		OnCellClick: onCellClick,
		Padding:     widget.Insets{Left: 15, Right: 15, Top: 15, Bottom: 15},
	})
	return gridContainer, buttons
}

// CreateRosterPalettePanel builds the roster-based unit palette list
func (sbuf *SquadBuilderUIFactory) CreateRosterPalettePanel(onEntrySelected func(interface{})) *widget.List {
	listWidth := int(float64(sbuf.layout.ScreenWidth) * widgets.SquadBuilderUnitListWidth)
	listHeight := int(float64(sbuf.layout.ScreenHeight) * widgets.SquadBuilderUnitListHeight)

	return widgets.CreateListWithConfig(widgets.ListConfig{
		Entries:   []interface{}{}, // Will be populated dynamically
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			// Handle roster entries
			if rosterEntry, ok := e.(*squads.UnitRosterEntry); ok {
				return rosterEntry.TemplateName
			}
			// Handle string messages
			if str, ok := e.(string); ok {
				return str
			}
			return fmt.Sprintf("%v", e)
		},
		OnEntrySelected: onEntrySelected,
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: widget.Insets{
				Left: 20,
				Top:  20,
			},
		},
	})
}

// CreatePalettePanel builds the unit palette list (deprecated - kept for compatibility)
func (sbuf *SquadBuilderUIFactory) CreatePalettePanel(onEntrySelected func(interface{})) *widget.List {
	listWidth := int(float64(sbuf.layout.ScreenWidth) * widgets.SquadBuilderUnitListWidth)
	listHeight := int(float64(sbuf.layout.ScreenHeight) * widgets.SquadBuilderUnitListHeight)

	// Build entries from squads.Units
	entries := make([]interface{}, len(squads.Units)+1)
	entries[0] = "[Remove Unit]"
	for i, unit := range squads.Units {
		entries[i+1] = fmt.Sprintf("%s (%s)", unit.Name, unit.Role.String())
	}

	return widgets.CreateListWithConfig(widgets.ListConfig{
		Entries:   entries,
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
		OnEntrySelected: onEntrySelected,
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: widget.Insets{
				Left: 20,
				Top:  20,
			},
		},
	})
}

// CreateCapacityDisplay builds the capacity display panel
func (sbuf *SquadBuilderUIFactory) CreateCapacityDisplay() *widget.TextArea {
	displayWidth := int(float64(sbuf.layout.ScreenWidth) * widgets.SquadBuilderInfoWidth)
	displayHeight := int(float64(sbuf.layout.ScreenHeight) * widgets.SquadBuilderInfoHeight)

	config := widgets.TextAreaConfig{
		MinWidth:  displayWidth,
		MinHeight: displayHeight,
		FontColor: color.White,
	}

	capacityDisplay := widgets.CreateTextAreaWithConfig(config)
	capacityDisplay.SetText("Capacity: 0.0 / 6.0\n(No leader)")

	capacityDisplay.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding: widget.Insets{
			Right: 20,
			Top:   80,
		},
	}

	return capacityDisplay
}

// CreateDetailsPanel builds the unit details display panel
func (sbuf *SquadBuilderUIFactory) CreateDetailsPanel() *widget.TextArea {
	displayWidth := int(float64(sbuf.layout.ScreenWidth) * widgets.SquadBuilderInfoWidth)
	displayHeight := int(float64(sbuf.layout.ScreenHeight) * (widgets.SquadBuilderInfoHeight * 2))

	config := widgets.TextAreaConfig{
		MinWidth:  displayWidth,
		MinHeight: displayHeight,
		FontColor: color.White,
	}

	unitDetailsArea := widgets.CreateTextAreaWithConfig(config)
	unitDetailsArea.SetText("Select a unit to view details")

	unitDetailsArea.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		Padding: widget.Insets{
			Right: 20,
		},
	}

	return unitDetailsArea
}

// CreateSquadNameInput builds the squad name input widget and container
func (sbuf *SquadBuilderUIFactory) CreateSquadNameInput(onChanged func(string)) (*widget.Container, *widget.TextInput) {
	inputContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
		)),
	)

	// Label
	nameLabel := widgets.CreateLargeLabel("Squad Name:")
	inputContainer.AddChild(nameLabel)

	// Text input
	squadNameInput := widgets.CreateTextInputWithConfig(widgets.TextInputConfig{
		MinWidth:    300,
		MinHeight:   50,
		FontFace:    guiresources.SmallFace,
		Placeholder: "Enter squad name...",
		OnChanged:   onChanged,
	})
	inputContainer.AddChild(squadNameInput)

	// Position at top center
	inputContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding: widget.Insets{
			Top: 20,
		},
	}

	return inputContainer, squadNameInput
}

// CreateActionButtons builds the action buttons container
func (sbuf *SquadBuilderUIFactory) CreateActionButtons(
	onCreate func(),
	onClear func(),
	onToggleLeader func(),
	onClose func(),
) *widget.Container {
	// Create button group with squad builder actions
	buttonContainer := widgets.CreateButtonGroup(widgets.ButtonGroupConfig{
		Buttons: []widgets.ButtonSpec{
			{
				Text:    "Create Squad",
				OnClick: onCreate,
			},
			{
				Text:    "Clear Grid",
				OnClick: onClear,
			},
			{
				Text:    "Toggle Leader (L)",
				OnClick: onToggleLeader,
			},
			{
				Text:    "Close (ESC)",
				OnClick: onClose,
			},
		},
		Direction: widget.DirectionHorizontal,
		Spacing:   15,
		Padding:   widget.Insets{Left: 10, Right: 10},
		LayoutData: &widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionEnd,
			Padding: widget.Insets{
				Bottom: int(float64(sbuf.layout.ScreenHeight) * widgets.BottomButtonOffset),
			},
		},
	})

	return buttonContainer
}
