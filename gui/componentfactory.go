package gui

import (
	"fmt"
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/guicomponents"
	"game_main/gui/guiresources"
	"game_main/gui/specs"
	"game_main/gui/widgets"
	"game_main/tactical/squads"

	"github.com/ebitenui/ebitenui/widget"
)

// UIComponentFactory builds complex UI components for all modes.
// This consolidates the functionality of CombatUIFactory and SquadBuilderUIFactory
// into a single factory pattern.
//
// Usage:
//
//	factory := NewUIComponentFactory(queries, panelBuilders, layout)
//	turnOrderPanel := factory.CreateCombatTurnOrderPanel()
//	gridPanel, buttons := factory.CreateSquadBuilderGridPanel(onCellClick)
type UIComponentFactory struct {
	queries       *guicomponents.GUIQueries
	panelBuilders *builders.PanelBuilders
	layout        *specs.LayoutConfig
	width, height int
}

// NewUIComponentFactory creates a new unified UI component factory
func NewUIComponentFactory(queries *guicomponents.GUIQueries, panelBuilders *builders.PanelBuilders, layout *specs.LayoutConfig) *UIComponentFactory {
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
	panelWidth := int(float64(ucf.layout.ScreenWidth) * specs.CombatTurnOrderWidth)
	panelHeight := int(float64(ucf.layout.ScreenHeight) * specs.CombatTurnOrderHeight)

	// Create panel with horizontal row layout
	panel := builders.CreatePanelWithConfig(builders.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(ucf.layout, specs.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	topPad := int(float64(ucf.layout.ScreenHeight) * specs.PaddingTight)
	panel.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

	return panel
}

// CreateCombatFactionInfoPanel builds the faction information panel
func (ucf *UIComponentFactory) CreateCombatFactionInfoPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(ucf.layout.ScreenWidth) * specs.CombatFactionInfoWidth)
	panelHeight := int(float64(ucf.layout.ScreenHeight) * specs.CombatFactionInfoHeight)

	// Create panel with vertical row layout
	panel := builders.CreatePanelWithConfig(builders.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(ucf.layout, specs.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	leftPad := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)
	topPad := int(float64(ucf.layout.ScreenHeight) * specs.PaddingTight)
	panel.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topPad)

	return panel
}

// CreateCombatSquadListPanel builds the squad list panel
func (ucf *UIComponentFactory) CreateCombatSquadListPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(ucf.layout.ScreenWidth) * specs.CombatSquadListWidth)
	panelHeight := int(float64(ucf.layout.ScreenHeight) * specs.CombatSquadListHeight)

	// Create panel with vertical row layout
	panel := builders.CreatePanelWithConfig(builders.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(ucf.layout, specs.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	// Position below FactionInfo panel (which is 10% height + padding)
	leftPad := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)
	topOffset := int(float64(ucf.layout.ScreenHeight) * (specs.CombatFactionInfoHeight + specs.PaddingTight))
	panel.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topOffset)

	// Add label
	listLabel := builders.CreateSmallLabel("Your Squads:")
	panel.AddChild(listLabel)

	return panel
}

// CreateCombatSquadDetailPanel builds the squad detail panel
func (ucf *UIComponentFactory) CreateCombatSquadDetailPanel() *widget.Container {
	// Calculate responsive size
	panelWidth := int(float64(ucf.layout.ScreenWidth) * specs.CombatSquadDetailWidth)
	panelHeight := int(float64(ucf.layout.ScreenHeight) * specs.CombatSquadDetailHeight)

	// Create panel with vertical row layout
	panel := builders.CreatePanelWithConfig(builders.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(ucf.layout, specs.PaddingExtraSmall)),
		),
	})

	// Apply anchor layout positioning
	// Position below SquadList panel (FactionInfo 10% + SquadList 35% + 3 padding gaps)
	leftPad := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)
	topOffset := int(float64(ucf.layout.ScreenHeight) * (specs.CombatFactionInfoHeight + specs.CombatSquadListHeight + specs.PaddingTight*3))
	panel.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topOffset)

	return panel
}

// CreateCombatLogPanel builds the combat log panel using standard specification
func (ucf *UIComponentFactory) CreateCombatLogPanel() (*widget.Container, *widgets.CachedTextAreaWrapper) {
	// Calculate responsive size
	panelWidth := int(float64(ucf.layout.ScreenWidth) * specs.CombatLogWidth)
	panelHeight := int(float64(ucf.layout.ScreenHeight) * specs.CombatLogHeight)

	// Create panel with anchor layout (to hold textarea)
	panel := builders.CreatePanelWithConfig(builders.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout:     widget.NewAnchorLayout(),
	})

	// Apply anchor layout positioning to panel
	// Position above action buttons (button height 8% + bottom offset 8% + padding)
	rightPad := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)
	bottomOffset := int(float64(ucf.layout.ScreenHeight) * (specs.CombatActionButtonHeight + specs.BottomButtonOffset + specs.PaddingTight))
	panel.GetWidget().LayoutData = builders.AnchorEndEnd(rightPad, bottomOffset)

	// Create cached textarea to fit within panel - only re-renders when combat log updates
	textArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
		MinWidth:  panelWidth - 20,
		MinHeight: panelHeight - 20,
		FontColor: color.White,
	})

	textArea.SetText("Combat started!\n") // SetText calls MarkDirty() internally
	panel.AddChild(textArea)              // The wrapper implements the necessary widget interfaces

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
	spacing := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Attack (A)", OnClick: onAttack},
			{Text: "Move (M)", OnClick: onMove},
			{Text: "Undo (Ctrl+Z)", OnClick: onUndo},
			{Text: "Redo (Ctrl+Y)", OnClick: onRedo},
			{Text: "End Turn (Space)", OnClick: onEndTurn},
			{Text: "Flee (ESC)", OnClick: onFlee},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(ucf.layout, specs.PaddingExtraSmall),
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
	spacing := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Throwables", OnClick: onThrowables},
			{Text: "Squads (E)", OnClick: onSquads},
			{Text: "Inventory (I)", OnClick: onInventory},
			{Text: "Deploy (D)", OnClick: onDeploy},
			{Text: "Combat (C)", OnClick: onCombat},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(ucf.layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

// CreateSquadManagementActionButtons builds the squad management mode buttons container (no panel wrapper, like combat mode)
func (ucf *UIComponentFactory) CreateSquadManagementActionButtons(
	onBattleMap func(),
	onSquadBuilder func(),
	onBuyUnits func(),
	onEditSquad func(),
) *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Battle Map (ESC)", OnClick: onBattleMap},
			{Text: "Squad Builder (B)", OnClick: onSquadBuilder},
			{Text: "Buy Units (P)", OnClick: onBuyUnits},
			{Text: "Edit Squad (E)", OnClick: onEditSquad},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(ucf.layout, specs.PaddingExtraSmall),
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
	spacing := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Buy Unit", OnClick: onBuyUnit},
			{Text: "Undo (Ctrl+Z)", OnClick: onUndo},
			{Text: "Redo (Ctrl+Y)", OnClick: onRedo},
			{Text: "Back (ESC)", OnClick: onBack},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(ucf.layout, specs.PaddingExtraSmall),
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
	spacing := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Rename Squad", OnClick: onRenameSquad},
			{Text: "Undo (Ctrl+Z)", OnClick: onUndo},
			{Text: "Redo (Ctrl+Y)", OnClick: onRedo},
			{Text: "Close (ESC)", OnClick: onClose},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(ucf.layout, specs.PaddingExtraSmall),
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
	spacing := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(ucf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Clear All", OnClick: onClearAll},
			{Text: "Start Combat", OnClick: onStartCombat},
			{Text: "Close (ESC)", OnClick: onClose},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(ucf.layout, specs.PaddingExtraSmall),
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
	padding := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)

	gridContainer, buttons := ucf.panelBuilders.BuildGridEditor(builders.GridEditorConfig{
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
	listWidth := int(float64(ucf.layout.ScreenWidth) * specs.SquadBuilderUnitListWidth)
	listHeight := int(float64(ucf.layout.ScreenHeight) * specs.SquadBuilderUnitListHeight)

	// Calculate responsive padding
	hPadding := int(float64(ucf.layout.ScreenWidth) * specs.PaddingStandard)
	vPadding := int(float64(ucf.layout.ScreenHeight) * specs.PaddingStandard)

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
func (ucf *UIComponentFactory) CreateSquadBuilderPalette(onEntrySelected func(interface{})) *widget.List {
	listWidth := int(float64(ucf.layout.ScreenWidth) * specs.SquadBuilderUnitListWidth)
	listHeight := int(float64(ucf.layout.ScreenHeight) * specs.SquadBuilderUnitListHeight)

	// Build entries from squads.Units
	entries := make([]interface{}, len(squads.Units)+1)
	entries[0] = "[Remove Unit]"
	for i, unit := range squads.Units {
		entries[i+1] = fmt.Sprintf("%s (%s)", unit.Name, unit.Role.String())
	}

	// Calculate responsive padding
	hPadding := int(float64(ucf.layout.ScreenWidth) * specs.PaddingStandard)
	vPadding := int(float64(ucf.layout.ScreenHeight) * specs.PaddingStandard)

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
func (ucf *UIComponentFactory) CreateSquadBuilderCapacityDisplay() *widget.TextArea {
	displayWidth := int(float64(ucf.layout.ScreenWidth) * specs.SquadBuilderInfoWidth)
	displayHeight := int(float64(ucf.layout.ScreenHeight) * specs.SquadBuilderInfoHeight)

	config := builders.TextAreaConfig{
		MinWidth:  displayWidth,
		MinHeight: displayHeight,
		FontColor: color.White,
	}

	capacityDisplay := builders.CreateTextAreaWithConfig(config)
	capacityDisplay.SetText("Capacity: 0.0 / 6.0\n(No leader)")

	// Calculate responsive padding
	hPadding := int(float64(ucf.layout.ScreenWidth) * specs.PaddingStandard)
	vPadding := int(float64(ucf.layout.ScreenHeight) * specs.PaddingStackedWidget)

	capacityDisplay.GetWidget().LayoutData = builders.AnchorEndStart(hPadding, vPadding)

	return capacityDisplay
}

// CreateSquadBuilderDetailsPanel builds the unit details display panel
func (ucf *UIComponentFactory) CreateSquadBuilderDetailsPanel() *widget.TextArea {
	displayWidth := int(float64(ucf.layout.ScreenWidth) * specs.SquadBuilderInfoWidth)
	displayHeight := int(float64(ucf.layout.ScreenHeight) * (specs.SquadBuilderInfoHeight * 2))

	config := builders.TextAreaConfig{
		MinWidth:  displayWidth,
		MinHeight: displayHeight,
		FontColor: color.White,
	}

	unitDetailsArea := builders.CreateTextAreaWithConfig(config)
	unitDetailsArea.SetText("Select a unit to view details")

	// Calculate responsive padding
	hPadding := int(float64(ucf.layout.ScreenWidth) * specs.PaddingStandard)

	unitDetailsArea.GetWidget().LayoutData = builders.AnchorEndCenter(hPadding)

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
	vPadding := int(float64(ucf.layout.ScreenHeight) * specs.PaddingStandard)

	inputContainer.GetWidget().LayoutData = builders.AnchorCenterStart(vPadding)

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
	spacing := int(float64(ucf.layout.ScreenWidth) * specs.PaddingTight)
	hPadding := int(float64(ucf.layout.ScreenWidth) * specs.PaddingExtraSmall)

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

	bottomPad := int(float64(ucf.layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)
	buttonContainer.GetWidget().LayoutData = anchorLayout

	return buttonContainer
}
