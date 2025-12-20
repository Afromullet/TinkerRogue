package gui

import (
	"fmt"
	"image/color"

	"game_main/gui/guicomponents"
	"game_main/gui/guiresources"
	"game_main/gui/widgets"
	"game_main/squads"

	"github.com/ebitenui/ebitenui/widget"
)

// UIComponentFactory builds complex UI components for all modes.
// This consolidates the functionality of CombatUIFactory and SquadBuilderUIFactory
// into a single factory pattern.
//
// Usage:
//   factory := NewUIComponentFactory(queries, panelBuilders, layout)
//   turnOrderPanel := factory.CreateCombatTurnOrderPanel()
//   gridPanel, buttons := factory.CreateSquadBuilderGridPanel(onCellClick)
type UIComponentFactory struct {
	queries       *guicomponents.GUIQueries
	panelBuilders *widgets.PanelBuilders
	layout        *widgets.LayoutConfig
	width, height int
}

// NewUIComponentFactory creates a new unified UI component factory
func NewUIComponentFactory(queries *guicomponents.GUIQueries, panelBuilders *widgets.PanelBuilders, layout *widgets.LayoutConfig) *UIComponentFactory {
	return &UIComponentFactory{
		queries:       queries,
		panelBuilders: panelBuilders,
		layout:        layout,
		width:         layout.ScreenWidth,
		height:        layout.ScreenHeight,
	}
}

// ============================================
// Combat UI Components
// ============================================

// CreateCombatTurnOrderPanel builds the turn order display panel
func (ucf *UIComponentFactory) CreateCombatTurnOrderPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(ucf.layout.ScreenWidth) * widgets.CombatTurnOrderWidth)
	panelHeight := int(float64(ucf.layout.ScreenHeight) * widgets.CombatTurnOrderHeight)

	// Create panel with horizontal row layout
	panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(NewResponsiveRowPadding(ucf.layout, widgets.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	topPad := int(float64(ucf.layout.ScreenHeight) * widgets.PaddingTight)
	panel.GetWidget().LayoutData = AnchorCenterStart(topPad)

	return panel
}

// CreateCombatFactionInfoPanel builds the faction information panel
func (ucf *UIComponentFactory) CreateCombatFactionInfoPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(ucf.layout.ScreenWidth) * widgets.CombatFactionInfoWidth)
	panelHeight := int(float64(ucf.layout.ScreenHeight) * widgets.CombatFactionInfoHeight)

	// Create panel with vertical row layout
	panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(NewResponsiveRowPadding(ucf.layout, widgets.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	leftPad := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)
	topPad := int(float64(ucf.layout.ScreenHeight) * widgets.PaddingTight)
	panel.GetWidget().LayoutData = AnchorStartStart(leftPad, topPad)

	return panel
}

// CreateCombatSquadListPanel builds the squad list panel
func (ucf *UIComponentFactory) CreateCombatSquadListPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(ucf.layout.ScreenWidth) * widgets.CombatSquadListWidth)
	panelHeight := int(float64(ucf.layout.ScreenHeight) * widgets.CombatSquadListHeight)

	// Create panel with vertical row layout
	panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(NewResponsiveRowPadding(ucf.layout, widgets.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	// Position below FactionInfo panel (which is 10% height + padding)
	leftPad := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)
	topOffset := int(float64(ucf.layout.ScreenHeight) * (widgets.CombatFactionInfoHeight + widgets.PaddingTight))
	panel.GetWidget().LayoutData = AnchorStartStart(leftPad, topOffset)

	// Add label
	listLabel := widgets.CreateSmallLabel("Your Squads:")
	panel.AddChild(listLabel)

	return panel
}

// CreateCombatSquadDetailPanel builds the squad detail panel
func (ucf *UIComponentFactory) CreateCombatSquadDetailPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(ucf.layout.ScreenWidth) * widgets.CombatSquadDetailWidth)
	panelHeight := int(float64(ucf.layout.ScreenHeight) * widgets.CombatSquadDetailHeight)

	// Create panel with vertical row layout
	panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(NewResponsiveRowPadding(ucf.layout, widgets.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	// Position below SquadList panel (FactionInfo 10% + SquadList 35% + 3 padding gaps)
	leftPad := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)
	topOffset := int(float64(ucf.layout.ScreenHeight) * (widgets.CombatFactionInfoHeight + widgets.CombatSquadListHeight + widgets.PaddingTight*3))
	panel.GetWidget().LayoutData = AnchorStartStart(leftPad, topOffset)

	return panel
}

// CreateCombatLogPanel builds the combat log panel using standard specification
func (ucf *UIComponentFactory) CreateCombatLogPanel() (*widget.Container, *widget.TextArea) {
	// Calculate responsive size
	panelWidth := int(float64(ucf.layout.ScreenWidth) * widgets.CombatLogWidth)
	panelHeight := int(float64(ucf.layout.ScreenHeight) * widgets.CombatLogHeight)

	// Create panel with anchor layout (to hold textarea)
	panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout:     widget.NewAnchorLayout(),
	})

	// Apply anchor layout positioning to panel
	// Position above action buttons (button height 8% + bottom offset 8% + padding)
	rightPad := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)
	bottomOffset := int(float64(ucf.layout.ScreenHeight) * (widgets.CombatActionButtonHeight + widgets.BottomButtonOffset + widgets.PaddingTight))
	panel.GetWidget().LayoutData = AnchorEndEnd(rightPad, bottomOffset)

	// Create textarea to fit within panel
	textArea := widgets.CreateTextAreaWithConfig(widgets.TextAreaConfig{
		MinWidth:  panelWidth - 20,
		MinHeight: panelHeight - 20,
		FontColor: color.White,
	})

	textArea.SetText("Combat started!\n")
	panel.AddChild(textArea)

	return panel, textArea
}

// CreateCombatActionButtons builds the action buttons container
func (ucf *UIComponentFactory) CreateCombatActionButtons(
	onAttack func(),
	onMove func(),
	onUndo func(),
	onRedo func(),
	onEndTurn func(),
	onFlee func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)

	// Create button group using widgets.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * widgets.BottomButtonOffset)
	anchorLayout := AnchorCenterEnd(bottomPad)

	buttonContainer := widgets.CreateButtonGroup(widgets.ButtonGroupConfig{
		Buttons: []widgets.ButtonSpec{
			{Text: "Attack (A)", OnClick: onAttack},
			{Text: "Move (M)", OnClick: onMove},
			{Text: "Undo (Ctrl+Z)", OnClick: onUndo},
			{Text: "Redo (Ctrl+Y)", OnClick: onRedo},
			{Text: "End Turn (Space)", OnClick: onEndTurn},
			{Text: "Flee (ESC)", OnClick: onFlee},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    NewResponsiveHorizontalPadding(ucf.layout, widgets.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateExplorationActionButtons builds the exploration mode buttons container (no panel wrapper, like combat mode)
func (ucf *UIComponentFactory) CreateExplorationActionButtons(
	onThrowables func(),
	onSquads func(),
	onInventory func(),
	onDeploy func(),
	onCombat func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)

	// Create button group using widgets.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * widgets.BottomButtonOffset)
	anchorLayout := AnchorCenterEnd(bottomPad)

	buttonContainer := widgets.CreateButtonGroup(widgets.ButtonGroupConfig{
		Buttons: []widgets.ButtonSpec{
			{Text: "Throwables", OnClick: onThrowables},
			{Text: "Squads (E)", OnClick: onSquads},
			{Text: "Inventory (I)", OnClick: onInventory},
			{Text: "Deploy (D)", OnClick: onDeploy},
			{Text: "Combat (C)", OnClick: onCombat},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    NewResponsiveHorizontalPadding(ucf.layout, widgets.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateSquadManagementActionButtons builds the squad management mode buttons container (no panel wrapper, like combat mode)
func (ucf *UIComponentFactory) CreateSquadManagementActionButtons(
	onBattleMap func(),
	onSquadBuilder func(),
	onFormation func(),
	onBuyUnits func(),
	onEditSquad func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)

	// Create button group using widgets.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * widgets.BottomButtonOffset)
	anchorLayout := AnchorCenterEnd(bottomPad)

	buttonContainer := widgets.CreateButtonGroup(widgets.ButtonGroupConfig{
		Buttons: []widgets.ButtonSpec{
			{Text: "Battle Map (ESC)", OnClick: onBattleMap},
			{Text: "Squad Builder (B)", OnClick: onSquadBuilder},
			{Text: "Formation (F)", OnClick: onFormation},
			{Text: "Buy Units (P)", OnClick: onBuyUnits},
			{Text: "Edit Squad (E)", OnClick: onEditSquad},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    NewResponsiveHorizontalPadding(ucf.layout, widgets.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateUnitPurchaseActionButtons builds the unit purchase mode buttons container (no panel wrapper, like combat mode)
func (ucf *UIComponentFactory) CreateUnitPurchaseActionButtons(
	onBuyUnit func(),
	onUndo func(),
	onRedo func(),
	onBack func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)

	// Create button group using widgets.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * widgets.BottomButtonOffset)
	anchorLayout := AnchorCenterEnd(bottomPad)

	buttonContainer := widgets.CreateButtonGroup(widgets.ButtonGroupConfig{
		Buttons: []widgets.ButtonSpec{
			{Text: "Buy Unit", OnClick: onBuyUnit},
			{Text: "Undo (Ctrl+Z)", OnClick: onUndo},
			{Text: "Redo (Ctrl+Y)", OnClick: onRedo},
			{Text: "Back (ESC)", OnClick: onBack},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    NewResponsiveHorizontalPadding(ucf.layout, widgets.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateSquadEditorActionButtons builds the squad editor mode buttons container (no panel wrapper, like combat mode)
func (ucf *UIComponentFactory) CreateSquadEditorActionButtons(
	onRenameSquad func(),
	onUndo func(),
	onRedo func(),
	onClose func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)

	// Create button group using widgets.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * widgets.BottomButtonOffset)
	anchorLayout := AnchorCenterEnd(bottomPad)

	buttonContainer := widgets.CreateButtonGroup(widgets.ButtonGroupConfig{
		Buttons: []widgets.ButtonSpec{
			{Text: "Rename Squad", OnClick: onRenameSquad},
			{Text: "Undo (Ctrl+Z)", OnClick: onUndo},
			{Text: "Redo (Ctrl+Y)", OnClick: onRedo},
			{Text: "Close (ESC)", OnClick: onClose},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    NewResponsiveHorizontalPadding(ucf.layout, widgets.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateSquadDeploymentActionButtons builds the squad deployment mode buttons container (no panel wrapper, like combat mode)
func (ucf *UIComponentFactory) CreateSquadDeploymentActionButtons(
	onClearAll func(),
	onStartCombat func(),
	onClose func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)

	// Create button group using widgets.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * widgets.BottomButtonOffset)
	anchorLayout := AnchorCenterEnd(bottomPad)

	buttonContainer := widgets.CreateButtonGroup(widgets.ButtonGroupConfig{
		Buttons: []widgets.ButtonSpec{
			{Text: "Clear All", OnClick: onClearAll},
			{Text: "Start Combat", OnClick: onStartCombat},
			{Text: "Close (ESC)", OnClick: onClose},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    NewResponsiveHorizontalPadding(ucf.layout, widgets.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// GetFormattedSquadDetails returns formatted squad details as string
func (ucf *UIComponentFactory) GetFormattedSquadDetails(squadID interface{}) string {
	// This is a helper that formats squad info for display
	// The actual formatting is delegated to the calling code
	return "Select a squad\nto view details"
}

// GetFormattedFactionInfo returns formatted faction info as string
func (ucf *UIComponentFactory) GetFormattedFactionInfo(factionInfo interface{}) string {
	// This is a helper that formats faction info for display
	if fi, ok := factionInfo.(*guicomponents.FactionInfo); ok {
		infoText := fmt.Sprintf("%s\n", fi.Name)
		infoText += fmt.Sprintf("Squads: %d/%d\n", fi.AliveSquadCount, len(fi.SquadIDs))
		infoText += fmt.Sprintf("Mana: %d/%d", fi.CurrentMana, fi.MaxMana)
		return infoText
	}
	return "Faction Info"
}

// ============================================
// Squad Builder UI Components
// ============================================

// CreateSquadBuilderGridPanel builds the 3x3 grid editor panel and returns button grid
func (ucf *UIComponentFactory) CreateSquadBuilderGridPanel(onCellClick func(row, col int)) (*widget.Container, [3][3]*widget.Button) {
	var buttons [3][3]*widget.Button

	// Calculate responsive padding
	padding := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)

	gridContainer, buttons := ucf.panelBuilders.BuildGridEditor(widgets.GridEditorConfig{
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
func (ucf *UIComponentFactory) CreateSquadBuilderRosterPalette(onEntrySelected func(interface{}), getRoster func() *squads.UnitRoster) *widget.List {
	listWidth := int(float64(ucf.layout.ScreenWidth) * widgets.SquadBuilderUnitListWidth)
	listHeight := int(float64(ucf.layout.ScreenHeight) * widgets.SquadBuilderUnitListHeight)

	// Calculate responsive padding
	hPadding := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingStandard)
	vPadding := int(float64(ucf.layout.ScreenHeight) * widgets.PaddingStandard)

	return widgets.CreateListWithConfig(widgets.ListConfig{
		Entries: []interface{}{}, // Will be populated dynamically
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
func (ucf *UIComponentFactory) CreateSquadBuilderPalette(onEntrySelected func(interface{})) *widget.List {
	listWidth := int(float64(ucf.layout.ScreenWidth) * widgets.SquadBuilderUnitListWidth)
	listHeight := int(float64(ucf.layout.ScreenHeight) * widgets.SquadBuilderUnitListHeight)

	// Build entries from squads.Units
	entries := make([]interface{}, len(squads.Units)+1)
	entries[0] = "[Remove Unit]"
	for i, unit := range squads.Units {
		entries[i+1] = fmt.Sprintf("%s (%s)", unit.Name, unit.Role.String())
	}

	// Calculate responsive padding
	hPadding := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingStandard)
	vPadding := int(float64(ucf.layout.ScreenHeight) * widgets.PaddingStandard)

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
				Left: hPadding,
				Top:  vPadding,
			},
		},
	})
}

// CreateSquadBuilderCapacityDisplay builds the capacity display panel
func (ucf *UIComponentFactory) CreateSquadBuilderCapacityDisplay() *widget.TextArea {
	displayWidth := int(float64(ucf.layout.ScreenWidth) * widgets.SquadBuilderInfoWidth)
	displayHeight := int(float64(ucf.layout.ScreenHeight) * widgets.SquadBuilderInfoHeight)

	config := widgets.TextAreaConfig{
		MinWidth:  displayWidth,
		MinHeight: displayHeight,
		FontColor: color.White,
	}

	capacityDisplay := widgets.CreateTextAreaWithConfig(config)
	capacityDisplay.SetText("Capacity: 0.0 / 6.0\n(No leader)")

	// Calculate responsive padding
	hPadding := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingStandard)
	vPadding := int(float64(ucf.layout.ScreenHeight) * widgets.PaddingStackedWidget)

	capacityDisplay.GetWidget().LayoutData = AnchorEndStart(hPadding, vPadding)

	return capacityDisplay
}

// CreateSquadBuilderDetailsPanel builds the unit details display panel
func (ucf *UIComponentFactory) CreateSquadBuilderDetailsPanel() *widget.TextArea {
	displayWidth := int(float64(ucf.layout.ScreenWidth) * widgets.SquadBuilderInfoWidth)
	displayHeight := int(float64(ucf.layout.ScreenHeight) * (widgets.SquadBuilderInfoHeight * 2))

	config := widgets.TextAreaConfig{
		MinWidth:  displayWidth,
		MinHeight: displayHeight,
		FontColor: color.White,
	}

	unitDetailsArea := widgets.CreateTextAreaWithConfig(config)
	unitDetailsArea.SetText("Select a unit to view details")

	// Calculate responsive padding
	hPadding := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingStandard)

	unitDetailsArea.GetWidget().LayoutData = AnchorEndCenter(hPadding)

	return unitDetailsArea
}

// CreateSquadBuilderNameInput builds the squad name input widget and container
func (ucf *UIComponentFactory) CreateSquadBuilderNameInput(onChanged func(string)) (*widget.Container, *widget.TextInput) {
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

	// Position at top center with responsive padding
	vPadding := int(float64(ucf.layout.ScreenHeight) * widgets.PaddingStandard)

	inputContainer.GetWidget().LayoutData = AnchorCenterStart(vPadding)

	return inputContainer, squadNameInput
}

// CreateSquadBuilderActionButtons builds the action buttons container
func (ucf *UIComponentFactory) CreateSquadBuilderActionButtons(
	onCreate func(),
	onClear func(),
	onToggleLeader func(),
	onClose func(),
) *widget.Container {
	// Calculate responsive spacing and padding
	spacing := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingTight)
	hPadding := int(float64(ucf.layout.ScreenWidth) * widgets.PaddingExtraSmall)

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
		Spacing:   spacing,
		Padding:   widget.Insets{Left: hPadding, Right: hPadding},
	})

	bottomPad := int(float64(ucf.layout.ScreenHeight) * widgets.BottomButtonOffset)
	anchorLayout := AnchorCenterEnd(bottomPad)
	buttonContainer.GetWidget().LayoutData = anchorLayout

	return buttonContainer
}
