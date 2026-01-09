package guisquads

import (
	"fmt"
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/guiresources"
	"game_main/gui/specs"
	"game_main/tactical/squads"

	"github.com/ebitenui/ebitenui/widget"
)

// SquadBuilderPanelFactory builds squad builder UI panels and components
type SquadBuilderPanelFactory struct {
	panelBuilders *builders.PanelBuilders
	layout        *specs.LayoutConfig
}

// NewSquadBuilderPanelFactory creates a factory for squad builder UI components
func NewSquadBuilderPanelFactory(panelBuilders *builders.PanelBuilders, layout *specs.LayoutConfig) *SquadBuilderPanelFactory {
	return &SquadBuilderPanelFactory{
		panelBuilders: panelBuilders,
		layout:        layout,
	}
}

// CreateSquadBuilderGridPanel builds the 3x3 grid editor panel and returns button grid
func (sbpf *SquadBuilderPanelFactory) CreateSquadBuilderGridPanel(onCellClick func(row, col int)) (*widget.Container, [3][3]*widget.Button) {
	var buttons [3][3]*widget.Button

	// Calculate responsive padding
	padding := int(float64(sbpf.layout.ScreenWidth) * specs.PaddingTight)

	gridContainer, buttons := sbpf.panelBuilders.BuildGridEditor(builders.GridEditorConfig{
		CellTextFormat: func(row, col int) string {
			return fmt.Sprintf("Empty\n[%d,%d]", row, col)
		},
		OnCellClick: onCellClick,
		Padding:     widget.Insets{Left: padding, Right: padding, Top: padding, Bottom: padding},
	})
	return gridContainer, buttons
}

// CreateSquadBuilderRosterPalette builds the roster-based unit palette list
// Requires roster to be passed for displaying counts
func (sbpf *SquadBuilderPanelFactory) CreateSquadBuilderRosterPalette(onEntrySelected func(interface{}), getRoster func() *squads.UnitRoster) *widget.List {
	listWidth := int(float64(sbpf.layout.ScreenWidth) * specs.SquadBuilderUnitListWidth)
	listHeight := int(float64(sbpf.layout.ScreenHeight) * specs.SquadBuilderUnitListHeight)

	// Calculate responsive padding
	hPadding := int(float64(sbpf.layout.ScreenWidth) * specs.PaddingStandard)
	vPadding := int(float64(sbpf.layout.ScreenHeight) * specs.PaddingStandard)

	return builders.CreateListWithConfig(builders.ListConfig{
		Entries:   []interface{}{}, // Will be populated dynamically
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			// Handle roster entries
			if rosterEntry, ok := e.(*squads.UnitRosterEntry); ok {
				roster := getRoster()
				if roster != nil {
					available := roster.GetAvailableCount(rosterEntry.TemplateName)
					return fmt.Sprintf("%s (x%d)", rosterEntry.TemplateName, available)
				}
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
				Left: hPadding,
				Top:  vPadding,
			},
		},
	})
}

// CreateSquadBuilderPalette builds the unit palette list (deprecated - kept for compatibility)
func (sbpf *SquadBuilderPanelFactory) CreateSquadBuilderPalette(onEntrySelected func(interface{})) *widget.List {
	listWidth := int(float64(sbpf.layout.ScreenWidth) * specs.SquadBuilderUnitListWidth)
	listHeight := int(float64(sbpf.layout.ScreenHeight) * specs.SquadBuilderUnitListHeight)

	// Build entries from squads.Units
	entries := make([]interface{}, len(squads.Units)+1)
	entries[0] = "[Remove Unit]"
	for i, unit := range squads.Units {
		entries[i+1] = fmt.Sprintf("%s (%s)", unit.Name, unit.Role.String())
	}

	// Calculate responsive padding
	hPadding := int(float64(sbpf.layout.ScreenWidth) * specs.PaddingStandard)
	vPadding := int(float64(sbpf.layout.ScreenHeight) * specs.PaddingStandard)

	return builders.CreateListWithConfig(builders.ListConfig{
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
				Left: hPadding,
				Top:  vPadding,
			},
		},
	})
}

// CreateSquadBuilderCapacityDisplay builds the capacity display panel
func (sbpf *SquadBuilderPanelFactory) CreateSquadBuilderCapacityDisplay() *widget.TextArea {
	displayWidth := int(float64(sbpf.layout.ScreenWidth) * specs.SquadBuilderInfoWidth)
	displayHeight := int(float64(sbpf.layout.ScreenHeight) * specs.SquadBuilderInfoHeight)

	config := builders.TextAreaConfig{
		MinWidth:  displayWidth,
		MinHeight: displayHeight,
		FontColor: color.White,
	}

	capacityDisplay := builders.CreateTextAreaWithConfig(config)
	capacityDisplay.SetText("Capacity: 0.0 / 6.0\n(No leader)")

	// Calculate responsive padding
	hPadding := int(float64(sbpf.layout.ScreenWidth) * specs.PaddingStandard)
	vPadding := int(float64(sbpf.layout.ScreenHeight) * specs.PaddingStackedWidget)

	capacityDisplay.GetWidget().LayoutData = builders.AnchorEndStart(hPadding, vPadding)

	return capacityDisplay
}

// CreateSquadBuilderDetailsPanel builds the unit details display panel
func (sbpf *SquadBuilderPanelFactory) CreateSquadBuilderDetailsPanel() *widget.TextArea {
	displayWidth := int(float64(sbpf.layout.ScreenWidth) * specs.SquadBuilderInfoWidth)
	displayHeight := int(float64(sbpf.layout.ScreenHeight) * (specs.SquadBuilderInfoHeight * 2))

	config := builders.TextAreaConfig{
		MinWidth:  displayWidth,
		MinHeight: displayHeight,
		FontColor: color.White,
	}

	unitDetailsArea := builders.CreateTextAreaWithConfig(config)
	unitDetailsArea.SetText("Select a unit to view details")

	// Calculate responsive padding
	hPadding := int(float64(sbpf.layout.ScreenWidth) * specs.PaddingStandard)

	unitDetailsArea.GetWidget().LayoutData = builders.AnchorEndCenter(hPadding)

	return unitDetailsArea
}

// CreateSquadBuilderNameInput builds the squad name input widget and container
func (sbpf *SquadBuilderPanelFactory) CreateSquadBuilderNameInput(onChanged func(string)) (*widget.Container, *widget.TextInput) {
	inputContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
		)),
	)

	// Label
	nameLabel := builders.CreateLargeLabel("Squad Name:")
	inputContainer.AddChild(nameLabel)

	// Text input
	squadNameInput := builders.CreateTextInputWithConfig(builders.TextInputConfig{
		MinWidth:    300,
		MinHeight:   50,
		FontFace:    guiresources.SmallFace,
		Placeholder: "Enter squad name...",
		OnChanged:   onChanged,
	})
	inputContainer.AddChild(squadNameInput)

	// Position at top center with responsive padding
	vPadding := int(float64(sbpf.layout.ScreenHeight) * specs.PaddingStandard)

	inputContainer.GetWidget().LayoutData = builders.AnchorCenterStart(vPadding)

	return inputContainer, squadNameInput
}

// CreateSquadBuilderActionButtons builds the action buttons container
func (sbpf *SquadBuilderPanelFactory) CreateSquadBuilderActionButtons(
	onCreate func(),
	onClear func(),
	onToggleLeader func(),
	onClose func(),
) *widget.Container {
	// Calculate responsive spacing and padding
	spacing := int(float64(sbpf.layout.ScreenWidth) * specs.PaddingTight)
	hPadding := int(float64(sbpf.layout.ScreenWidth) * specs.PaddingExtraSmall)

	// Create button group with squad builder actions
	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
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
		Spacing:   spacing,
		Padding:   widget.Insets{Left: hPadding, Right: hPadding},
	})

	bottomPad := int(float64(sbpf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)
	buttonContainer.GetWidget().LayoutData = anchorLayout

	return buttonContainer
}
