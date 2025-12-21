package guisquads

import (
	"fmt"
	"game_main/common"
	"game_main/gui"
	"game_main/gui/builders"
	"game_main/gui/core"
	"game_main/gui/specs"
	"game_main/squads"
	"game_main/squads/squadcommands"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadEditorMode provides comprehensive squad editing capabilities:
// - Select squads
// - Add units from roster
// - Remove units from squad
// - Change squad leader
// - Change unit positions in grid
type SquadEditorMode struct {
	gui.BaseMode // Embed common mode infrastructure

	// Squad navigation
	currentSquadIndex int
	allSquadIDs       []ecs.EntityID

	// UI containers
	squadSelectorContainer *widget.Container
	gridEditorContainer    *widget.Container
	unitListContainer      *widget.Container
	rosterListContainer    *widget.Container
	actionButtonsContainer *widget.Container
	navigationContainer    *widget.Container

	// UI widgets
	squadSelector     *widget.List   // Interactive - no caching
	gridCells         [3][3]*widget.Button
	unitList          *widget.List   // Interactive - no caching
	rosterList        *widget.List   // Interactive - no caching
	squadCounterLabel *widget.Text   // "Squad 1 of 3"
	prevButton        *widget.Button
	nextButton        *widget.Button

	// State
	selectedGridCell *GridCell      // Currently selected grid cell
	selectedUnitID   ecs.EntityID   // Currently selected unit in squad
}

// GridCell represents a selected cell in the 3x3 grid
type GridCell struct {
	Row int
	Col int
}

func NewSquadEditorMode(modeManager *core.UIModeManager) *SquadEditorMode {
	mode := &SquadEditorMode{
		currentSquadIndex: 0,
		allSquadIDs:       make([]ecs.EntityID, 0),
	}
	mode.SetModeName("squad_editor")
	mode.SetReturnMode("squad_management")
	mode.ModeManager = modeManager
	return mode
}

func (sem *SquadEditorMode) Initialize(ctx *core.UIContext) error {
	err := gui.NewModeBuilder(&sem.BaseMode, gui.ModeConfig{
		ModeName:   "squad_editor",
		ReturnMode: "squad_management",

		Panels: []gui.PanelSpec{
			{CustomBuild: sem.buildSquadNavigation},
			{CustomBuild: sem.buildSquadSelector},
			{CustomBuild: sem.buildGridEditor},
			{CustomBuild: sem.buildUnitList},
			{CustomBuild: sem.buildRosterList},
		},

		StatusLabel: true,
		Commands:    true,
		OnRefresh:   sem.refreshAfterUndoRedo,
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Add action buttons after ModeBuilder completes
	actionButtons := sem.buildActionButtons()
	sem.RootContainer.AddChild(actionButtons)

	return nil
}

// buildSquadNavigation creates Previous/Next buttons for squad navigation
func (sem *SquadEditorMode) buildSquadNavigation() *widget.Container {
	// Calculate responsive size
	navWidth := int(float64(sem.Layout.ScreenWidth) * 0.5)
	navHeight := int(float64(sem.Layout.ScreenHeight) * specs.SquadEditorNavHeight)

	sem.navigationContainer = builders.CreateStaticPanel(builders.PanelConfig{
		MinWidth:  navWidth,
		MinHeight: navHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(20),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(sem.Layout, specs.PaddingExtraSmall)),
		),
	})

	topPad := int(float64(sem.Layout.ScreenHeight) * specs.PaddingStandard)
	sem.navigationContainer.GetWidget().LayoutData = gui.AnchorCenterStart(topPad)

	// Previous button
	sem.prevButton = builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "< Previous",
		OnClick: func() {
			sem.showPreviousSquad()
		},
	})
	sem.navigationContainer.AddChild(sem.prevButton)

	// Squad counter label
	sem.squadCounterLabel = builders.CreateSmallLabel("Squad 1 of 1")
	sem.navigationContainer.AddChild(sem.squadCounterLabel)

	// Next button
	sem.nextButton = builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Next >",
		OnClick: func() {
			sem.showNextSquad()
		},
	})
	sem.navigationContainer.AddChild(sem.nextButton)

	return sem.navigationContainer
}

// buildSquadSelector creates the squad selection list (left side)
func (sem *SquadEditorMode) buildSquadSelector() *widget.Container {
	// Calculate responsive size
	listWidth := int(float64(sem.Layout.ScreenWidth) * specs.SquadEditorSquadListWidth)
	listHeight := int(float64(sem.Layout.ScreenHeight) * specs.SquadEditorSquadListHeight)

	sem.squadSelectorContainer = builders.CreateStaticPanel(builders.PanelConfig{
		MinWidth:  listWidth,
		MinHeight: listHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
		),
	})

	// Position below navigation, left side
	leftPad := int(float64(sem.Layout.ScreenWidth) * specs.PaddingStandard)
	topOffset := int(float64(sem.Layout.ScreenHeight) * (specs.SquadEditorNavHeight + specs.PaddingStandard*2))
	sem.squadSelectorContainer.GetWidget().LayoutData = gui.AnchorStartStart(leftPad, topOffset)

	titleLabel := builders.CreateSmallLabel("Select Squad:")
	sem.squadSelectorContainer.AddChild(titleLabel)

	// Squad list will be populated in Enter() - interactive, so no caching
	sem.squadSelector = builders.CreateSquadList(builders.SquadListConfig{
		SquadIDs:      []ecs.EntityID{},
		Manager:       sem.Context.ECSManager,
		ScreenWidth:   sem.Layout.ScreenWidth,
		ScreenHeight:  sem.Layout.ScreenHeight,
		WidthPercent:  0.2,
		HeightPercent: 0.4,
		OnSelect: func(squadID ecs.EntityID) {
			sem.onSquadSelected(squadID)
		},
	})
	sem.squadSelectorContainer.AddChild(sem.squadSelector)

	return sem.squadSelectorContainer
}

// buildGridEditor creates the 3x3 grid editor (center)
func (sem *SquadEditorMode) buildGridEditor() *widget.Container {
	sem.gridEditorContainer, sem.gridCells = sem.PanelBuilders.BuildGridEditor(builders.GridEditorConfig{
		OnCellClick: func(row, col int) {
			sem.onGridCellClicked(row, col)
		},
	})
	return sem.gridEditorContainer
}

// buildUnitList creates the unit list for the current squad (right side, top)
func (sem *SquadEditorMode) buildUnitList() *widget.Container {
	// Calculate responsive size
	listWidth := int(float64(sem.Layout.ScreenWidth) * specs.SquadEditorUnitListWidth)
	listHeight := int(float64(sem.Layout.ScreenHeight) * 0.35) // Half of vertical space

	sem.unitListContainer = builders.CreateStaticPanel(builders.PanelConfig{
		MinWidth:  listWidth,
		MinHeight: listHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
		),
	})

	// Position below navigation, right side
	rightPad := int(float64(sem.Layout.ScreenWidth) * specs.PaddingStandard)
	topOffset := int(float64(sem.Layout.ScreenHeight) * (specs.SquadEditorNavHeight + specs.PaddingStandard*2))
	sem.unitListContainer.GetWidget().LayoutData = gui.AnchorEndStart(rightPad, topOffset)

	titleLabel := builders.CreateSmallLabel("Squad Units:")
	sem.unitListContainer.AddChild(titleLabel)

	// Unit list will be populated when squad is selected - interactive, so no caching
	sem.unitList = builders.CreateUnitList(builders.UnitListConfig{
		UnitIDs:       []ecs.EntityID{},
		Manager:       sem.Context.ECSManager,
		ScreenWidth:   400,
		ScreenHeight:  300,
		WidthPercent:  1.0,
		HeightPercent: 1.0,
	})
	sem.unitListContainer.AddChild(sem.unitList)

	// Buttons for unit actions
	removeUnitBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Remove Selected Unit",
		OnClick: func() {
			sem.onRemoveUnit()
		},
	})
	sem.unitListContainer.AddChild(removeUnitBtn)

	makeLeaderBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Make Leader",
		OnClick: func() {
			sem.onMakeLeader()
		},
	})
	sem.unitListContainer.AddChild(makeLeaderBtn)

	return sem.unitListContainer
}

// buildRosterList creates the roster list showing available units (right side, bottom)
func (sem *SquadEditorMode) buildRosterList() *widget.Container {
	// Calculate responsive size
	listWidth := int(float64(sem.Layout.ScreenWidth) * specs.SquadEditorRosterListWidth)
	listHeight := int(float64(sem.Layout.ScreenHeight) * 0.35) // Half of vertical space

	sem.rosterListContainer = builders.CreateStaticPanel(builders.PanelConfig{
		MinWidth:  listWidth,
		MinHeight: listHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
		),
	})

	// Position below unitListContainer
	rightPad := int(float64(sem.Layout.ScreenWidth) * specs.PaddingStandard)
	topOffset := int(float64(sem.Layout.ScreenHeight) * (specs.SquadEditorNavHeight + 0.35 + specs.PaddingStandard*3))
	sem.rosterListContainer.GetWidget().LayoutData = gui.AnchorEndStart(rightPad, topOffset)

	titleLabel := builders.CreateSmallLabel("Available Units (Roster):")
	sem.rosterListContainer.AddChild(titleLabel)

	// Roster list will be populated in refreshRosterList() - interactive, so no caching
	sem.rosterList = builders.CreateSimpleStringList(builders.SimpleStringListConfig{
		Entries:       []string{},
		ScreenWidth:   400,
		ScreenHeight:  200,
		WidthPercent:  1.0,
		HeightPercent: 1.0,
	})
	sem.rosterListContainer.AddChild(sem.rosterList)

	addUnitBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
		Text: "Add to Squad",
		OnClick: func() {
			sem.onAddUnitFromRoster()
		},
	})
	sem.rosterListContainer.AddChild(addUnitBtn)

	return sem.rosterListContainer
}

// buildActionButtons creates bottom action buttons (called after Initialize completes)
func (sem *SquadEditorMode) buildActionButtons() *widget.Container {
	// Create UI factory
	uiFactory := gui.NewUIComponentFactory(sem.Queries, sem.PanelBuilders, sem.Layout)

	// Create button callbacks (no panel wrapper - like combat mode)
	sem.actionButtonsContainer = uiFactory.CreateSquadEditorActionButtons(
		// Rename Squad
		func() {
			sem.onRenameSquad()
		},
		// Undo
		func() {
			sem.CommandHistory.Undo()
		},
		// Redo
		func() {
			sem.CommandHistory.Redo()
		},
		// Close
		func() {
			if mode, exists := sem.ModeManager.GetMode("squad_management"); exists {
				sem.ModeManager.RequestTransition(mode, "Close button pressed")
			}
		},
	)

	return sem.actionButtonsContainer
}

func (sem *SquadEditorMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Squad Editor Mode")

	// Backfill roster with any existing squad units
	// This handles units created before roster tracking was implemented
	sem.backfillRosterWithSquadUnits()

	// Get all squad IDs
	sem.allSquadIDs = sem.Queries.SquadCache.FindAllSquads()

	// Refresh squad selector with current squads
	sem.refreshSquadSelector()

	// Reset to first squad if we have any
	if len(sem.allSquadIDs) > 0 {
		sem.currentSquadIndex = 0
		sem.refreshCurrentSquad()
	} else {
		sem.SetStatus("No squads available")
	}

	sem.updateNavigationButtons()
	sem.refreshRosterList()

	return nil
}

func (sem *SquadEditorMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Squad Editor Mode")
	sem.selectedGridCell = nil
	sem.selectedUnitID = 0
	return nil
}

func (sem *SquadEditorMode) Update(deltaTime float64) error {
	return nil
}

func (sem *SquadEditorMode) Render(screen *ebiten.Image) {
	// No custom rendering needed
}

func (sem *SquadEditorMode) HandleInput(inputState *core.InputState) bool {
	// Handle common input (ESC key)
	if sem.HandleCommonInput(inputState) {
		return true
	}

	// Handle undo/redo input (Ctrl+Z, Ctrl+Y)
	if sem.CommandHistory.HandleInput(inputState) {
		return true
	}

	return false
}

// refreshCurrentSquad loads the current squad's data into the UI
func (sem *SquadEditorMode) refreshCurrentSquad() {
	if len(sem.allSquadIDs) == 0 {
		return
	}

	currentSquadID := sem.allSquadIDs[sem.currentSquadIndex]

	// Update squad counter
	counterText := fmt.Sprintf("Squad %d of %d", sem.currentSquadIndex+1, len(sem.allSquadIDs))
	sem.squadCounterLabel.Label = counterText

	// Load squad formation into grid
	sem.loadSquadFormation(currentSquadID)

	// Refresh unit list
	sem.refreshUnitList(currentSquadID)

	// Update status
	squadName := sem.Queries.SquadCache.GetSquadName(currentSquadID)
	sem.SetStatus(fmt.Sprintf("Editing squad: %s", squadName))
}

// loadSquadFormation loads squad units into the 3x3 grid display
func (sem *SquadEditorMode) loadSquadFormation(squadID ecs.EntityID) {
	// Clear grid first
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			sem.gridCells[row][col].Text().Label = ""
		}
	}

	// Get units in squad and display them
	unitIDs := sem.Queries.SquadCache.GetUnitIDsInSquad(squadID)

	for _, unitID := range unitIDs {
		gridPos := common.GetComponentTypeByIDWithTag[*squads.GridPositionData](
			sem.Queries.ECSManager, unitID, squads.SquadMemberTag, squads.GridPositionComponent)
		if gridPos == nil {
			continue
		}

		// Get unit name
		nameStr := "Unit"
		if nameComp, ok := sem.Queries.ECSManager.GetComponent(unitID, common.NameComponent); ok {
			if name := nameComp.(*common.Name); name != nil {
				nameStr = name.NameStr
			}
		}

		// Check if leader
		isLeader := sem.Queries.ECSManager.HasComponentByIDWithTag(unitID, squads.SquadMemberTag, squads.LeaderComponent)
		if isLeader {
			nameStr = "[L] " + nameStr
		}

		// Update grid cell
		if gridPos.AnchorRow >= 0 && gridPos.AnchorRow < 3 && gridPos.AnchorCol >= 0 && gridPos.AnchorCol < 3 {
			sem.gridCells[gridPos.AnchorRow][gridPos.AnchorCol].Text().Label = nameStr
		}
	}
}

// refreshSquadSelector updates the squad selector list
func (sem *SquadEditorMode) refreshSquadSelector() {
	// Recreate squad list with current squads
	sem.squadSelectorContainer.RemoveChild(sem.squadSelector)
	sem.squadSelector = builders.CreateSquadList(builders.SquadListConfig{
		SquadIDs:      sem.allSquadIDs,
		Manager:       sem.Context.ECSManager,
		ScreenWidth:   sem.Layout.ScreenWidth,
		ScreenHeight:  sem.Layout.ScreenHeight,
		WidthPercent:  0.2,
		HeightPercent: 0.4,
		OnSelect: func(squadID ecs.EntityID) {
			sem.onSquadSelected(squadID)
		},
	})

	// Insert at position 1 (after title label)
	children := sem.squadSelectorContainer.Children()
	sem.squadSelectorContainer.RemoveChildren()
	sem.squadSelectorContainer.AddChild(children[0]) // Title label
	sem.squadSelectorContainer.AddChild(sem.squadSelector)
}

// refreshUnitList updates the unit list for the current squad
func (sem *SquadEditorMode) refreshUnitList(squadID ecs.EntityID) {
	unitIDs := sem.Queries.SquadCache.GetUnitIDsInSquad(squadID)

	// Recreate unit list with updated units
	sem.unitListContainer.RemoveChild(sem.unitList)
	sem.unitList = builders.CreateUnitList(builders.UnitListConfig{
		UnitIDs:       unitIDs,
		Manager:       sem.Queries.ECSManager,
		ScreenWidth:   400,
		ScreenHeight:  300,
		WidthPercent:  1.0,
		HeightPercent: 1.0,
	})

	// Insert at position 1 (after title label)
	children := sem.unitListContainer.Children()
	sem.unitListContainer.RemoveChildren()
	sem.unitListContainer.AddChild(children[0]) // Title label
	sem.unitListContainer.AddChild(sem.unitList)
	for i := 1; i < len(children); i++ {
		sem.unitListContainer.AddChild(children[i])
	}
}

// refreshRosterList updates the available units from player's roster
func (sem *SquadEditorMode) refreshRosterList() {
	roster := squads.GetPlayerRoster(sem.Context.PlayerData.PlayerEntityID, sem.Queries.ECSManager)
	if roster == nil {
		return
	}

	entries := make([]string, 0)
	for templateName := range roster.Units {
		availableCount := roster.GetAvailableCount(templateName)
		if availableCount > 0 {
			entries = append(entries, fmt.Sprintf("%s (x%d)", templateName, availableCount))
		}
	}

	if len(entries) == 0 {
		entries = append(entries, "No units available")
	}

	// Recreate roster list
	sem.rosterListContainer.RemoveChild(sem.rosterList)
	sem.rosterList = builders.CreateSimpleStringList(builders.SimpleStringListConfig{
		Entries:       entries,
		ScreenWidth:   400,
		ScreenHeight:  200,
		WidthPercent:  1.0,
		HeightPercent: 1.0,
	})

	// Insert at position 1 (after title label)
	children := sem.rosterListContainer.Children()
	sem.rosterListContainer.RemoveChildren()
	sem.rosterListContainer.AddChild(children[0]) // Title label
	sem.rosterListContainer.AddChild(sem.rosterList)
	for i := 1; i < len(children); i++ {
		sem.rosterListContainer.AddChild(children[i])
	}
}

// showPreviousSquad cycles to previous squad
func (sem *SquadEditorMode) showPreviousSquad() {
	if len(sem.allSquadIDs) == 0 {
		return
	}

	sem.currentSquadIndex--
	if sem.currentSquadIndex < 0 {
		sem.currentSquadIndex = len(sem.allSquadIDs) - 1
	}

	sem.refreshCurrentSquad()
	sem.updateNavigationButtons()
}

// showNextSquad cycles to next squad
func (sem *SquadEditorMode) showNextSquad() {
	if len(sem.allSquadIDs) == 0 {
		return
	}

	sem.currentSquadIndex++
	if sem.currentSquadIndex >= len(sem.allSquadIDs) {
		sem.currentSquadIndex = 0
	}

	sem.refreshCurrentSquad()
	sem.updateNavigationButtons()
}

// updateNavigationButtons enables/disables navigation based on squad count
func (sem *SquadEditorMode) updateNavigationButtons() {
	hasMultipleSquads := len(sem.allSquadIDs) > 1

	if sem.prevButton != nil {
		sem.prevButton.GetWidget().Disabled = !hasMultipleSquads
	}

	if sem.nextButton != nil {
		sem.nextButton.GetWidget().Disabled = !hasMultipleSquads
	}
}

// onSquadSelected handles squad selection from the list
func (sem *SquadEditorMode) onSquadSelected(squadID ecs.EntityID) {
	// Find index of selected squad
	for i, id := range sem.allSquadIDs {
		if id == squadID {
			sem.currentSquadIndex = i
			sem.refreshCurrentSquad()
			return
		}
	}
}

// onGridCellClicked handles clicking a grid cell
func (sem *SquadEditorMode) onGridCellClicked(row, col int) {
	if len(sem.allSquadIDs) == 0 {
		sem.SetStatus("No squad selected")
		return
	}

	currentSquadID := sem.allSquadIDs[sem.currentSquadIndex]

	// Check if there's a unit at this position
	unitIDs := squads.GetUnitIDsAtGridPosition(currentSquadID, row, col, sem.Queries.ECSManager)

	if len(unitIDs) > 0 {
		// Unit exists - select it for moving
		sem.selectedUnitID = unitIDs[0]
		sem.selectedGridCell = nil

		// Get unit name
		nameStr := "Unit"
		if nameComp, ok := sem.Queries.ECSManager.GetComponent(sem.selectedUnitID, common.NameComponent); ok {
			if name := nameComp.(*common.Name); name != nil {
				nameStr = name.NameStr
			}
		}

		sem.SetStatus(fmt.Sprintf("Selected unit: %s. Click another cell to move", nameStr))
	} else if sem.selectedUnitID != 0 {
		// Empty cell clicked with unit selected - move unit here
		sem.moveSelectedUnitToCell(row, col)
		sem.selectedUnitID = 0
		sem.selectedGridCell = nil
	} else {
		// Empty cell clicked with no unit selected - remember for adding units
		sem.selectedGridCell = &GridCell{Row: row, Col: col}
		sem.SetStatus(fmt.Sprintf("Selected cell [%d,%d]. Click 'Add to Squad' to place a unit here", row, col))
	}
}

// moveSelectedUnitToCell moves the currently selected unit to the specified cell
func (sem *SquadEditorMode) moveSelectedUnitToCell(row, col int) {
	if sem.selectedUnitID == 0 {
		return
	}

	currentSquadID := sem.allSquadIDs[sem.currentSquadIndex]

	// Create and execute move command
	cmd := squadcommands.NewMoveUnitCommand(
		sem.Queries.ECSManager,
		currentSquadID,
		sem.selectedUnitID,
		row,
		col,
	)

	sem.CommandHistory.Execute(cmd)
}

// onAddUnitFromRoster adds a unit from the roster to the squad
func (sem *SquadEditorMode) onAddUnitFromRoster() {
	if len(sem.allSquadIDs) == 0 {
		sem.SetStatus("No squad selected")
		return
	}

	selectedEntry := sem.rosterList.SelectedEntry()
	if selectedEntry == nil {
		sem.SetStatus("No unit selected from roster")
		return
	}

	if sem.selectedGridCell == nil {
		sem.SetStatus("No grid cell selected. Click an empty cell first")
		return
	}

	// Parse template name from entry (format: "TemplateName (xN)")
	entryStr := selectedEntry.(string)
	if entryStr == "No units available" {
		return
	}

	// Extract template name (everything before " (x")
	templateName := entryStr
	for i, c := range entryStr {
		if c == ' ' && i+1 < len(entryStr) && entryStr[i+1] == '(' {
			templateName = entryStr[:i]
			break
		}
	}

	currentSquadID := sem.allSquadIDs[sem.currentSquadIndex]

	// Create and execute add unit command
	cmd := squadcommands.NewAddUnitCommand(
		sem.Queries.ECSManager,
		sem.Context.PlayerData.PlayerEntityID,
		currentSquadID,
		templateName,
		sem.selectedGridCell.Row,
		sem.selectedGridCell.Col,
	)

	sem.CommandHistory.Execute(cmd)
	sem.selectedGridCell = nil
}

// onRemoveUnit removes the selected unit from the squad
func (sem *SquadEditorMode) onRemoveUnit() {
	if len(sem.allSquadIDs) == 0 {
		sem.SetStatus("No squad selected")
		return
	}

	selectedEntry := sem.unitList.SelectedEntry()
	if selectedEntry == nil {
		sem.SetStatus("No unit selected")
		return
	}

	// Get unit ID from selected entry (entry is the UnitIdentity)
	unitIdentity := selectedEntry.(squads.UnitIdentity)
	unitID := unitIdentity.ID

	// Check if this is the leader
	isLeader := sem.Queries.ECSManager.HasComponentByIDWithTag(unitID, squads.SquadMemberTag, squads.LeaderComponent)
	if isLeader {
		sem.SetStatus("Cannot remove leader. Make another unit leader first")
		return
	}

	currentSquadID := sem.allSquadIDs[sem.currentSquadIndex]

	// Calculate dialog position (center of screen, above the grid)
	dialogWidth := 400
	dialogHeight := 200
	centerX := sem.Layout.ScreenWidth / 2
	centerY := sem.Layout.ScreenHeight / 3 // Position in upper third of screen

	// Show confirmation dialog
	dialog := builders.CreateConfirmationDialog(builders.DialogConfig{
		Title:     "Confirm Remove Unit",
		Message:   fmt.Sprintf("Remove '%s' from squad?\n\nUnit will return to roster.\n\nYou can undo with Ctrl+Z.", unitIdentity.Name),
		MinWidth:  dialogWidth,
		MinHeight: dialogHeight,
		CenterX:   centerX,
		CenterY:   centerY,
		OnConfirm: func() {
			cmd := squadcommands.NewRemoveUnitCommand(
				sem.Queries.ECSManager,
				sem.Context.PlayerData.PlayerEntityID,
				currentSquadID,
				unitID,
			)

			sem.CommandHistory.Execute(cmd)
		},
		OnCancel: func() {
			sem.SetStatus("Remove cancelled")
		},
	})

	sem.GetEbitenUI().AddWindow(dialog)
}

// onMakeLeader changes the squad leader to the selected unit
func (sem *SquadEditorMode) onMakeLeader() {
	if len(sem.allSquadIDs) == 0 {
		sem.SetStatus("No squad selected")
		return
	}

	selectedEntry := sem.unitList.SelectedEntry()
	if selectedEntry == nil {
		sem.SetStatus("No unit selected")
		return
	}

	// Get unit ID from selected entry
	unitIdentity := selectedEntry.(squads.UnitIdentity)
	unitID := unitIdentity.ID

	// Check if already leader
	isLeader := sem.Queries.ECSManager.HasComponentByIDWithTag(unitID, squads.SquadMemberTag, squads.LeaderComponent)
	if isLeader {
		sem.SetStatus("Unit is already the leader")
		return
	}

	currentSquadID := sem.allSquadIDs[sem.currentSquadIndex]

	// Calculate dialog position (center of screen, above the grid)
	dialogWidth := 400
	dialogHeight := 200
	centerX := sem.Layout.ScreenWidth / 2
	centerY := sem.Layout.ScreenHeight / 3 // Position in upper third of screen

	// Show confirmation dialog
	dialog := builders.CreateConfirmationDialog(builders.DialogConfig{
		Title:     "Confirm Change Leader",
		Message:   fmt.Sprintf("Make '%s' the new squad leader?\n\nYou can undo with Ctrl+Z.", unitIdentity.Name),
		MinWidth:  dialogWidth,
		MinHeight: dialogHeight,
		CenterX:   centerX,
		CenterY:   centerY,
		OnConfirm: func() {
			cmd := squadcommands.NewChangeLeaderCommand(
				sem.Queries.ECSManager,
				currentSquadID,
				unitID,
			)

			sem.CommandHistory.Execute(cmd)
		},
		OnCancel: func() {
			sem.SetStatus("Change leader cancelled")
		},
	})

	sem.GetEbitenUI().AddWindow(dialog)
}

// onRenameSquad prompts for a new name and executes RenameSquadCommand
func (sem *SquadEditorMode) onRenameSquad() {
	if len(sem.allSquadIDs) == 0 {
		sem.SetStatus("No squad selected")
		return
	}

	currentSquadID := sem.allSquadIDs[sem.currentSquadIndex]
	currentName := sem.Queries.SquadCache.GetSquadName(currentSquadID)

	// Show text input dialog
	dialog := builders.CreateTextInputDialog(builders.TextInputDialogConfig{
		Title:       "Rename Squad",
		Message:     "Enter new squad name:",
		Placeholder: "Squad name",
		InitialText: currentName,
		OnConfirm: func(newName string) {
			if newName == "" || newName == currentName {
				sem.SetStatus("Rename cancelled")
				return
			}

			// Create and execute rename command
			cmd := squadcommands.NewRenameSquadCommand(
				sem.Queries.ECSManager,
				currentSquadID,
				newName,
			)

			sem.CommandHistory.Execute(cmd)
		},
		OnCancel: func() {
			sem.SetStatus("Rename cancelled")
		},
	})

	sem.GetEbitenUI().AddWindow(dialog)
}

// backfillRosterWithSquadUnits registers all existing squad units in the roster
// This is called when entering the mode to handle units created before roster tracking
func (sem *SquadEditorMode) backfillRosterWithSquadUnits() {
	roster := squads.GetPlayerRoster(sem.Context.PlayerData.PlayerEntityID, sem.Queries.ECSManager)
	if roster == nil {
		return
	}

	// Get all squads
	allSquads := sem.Queries.SquadCache.FindAllSquads()

	for _, squadID := range allSquads {
		// Get all units in this squad
		unitIDs := sem.Queries.SquadCache.GetUnitIDsInSquad(squadID)

		for _, unitID := range unitIDs {
			// Check if unit is already in roster
			alreadyRegistered := false
			for _, entry := range roster.Units {
				for _, existingID := range entry.UnitEntities {
					if existingID == unitID {
						alreadyRegistered = true
						break
					}
				}
				if alreadyRegistered {
					break
				}
			}

			// Register if not already in roster
			if !alreadyRegistered {
				err := squads.RegisterSquadUnitInRoster(roster, unitID, squadID, sem.Queries.ECSManager)
				if err != nil {
					fmt.Printf("Warning: Failed to register unit %d in roster: %v\n", unitID, err)
				}
			}
		}
	}
}

// refreshAfterUndoRedo is called after successful undo/redo operations
func (sem *SquadEditorMode) refreshAfterUndoRedo() {
	// Refresh squad list (squads might have been created/destroyed or renamed)
	sem.allSquadIDs = sem.Queries.SquadCache.FindAllSquads()

	// Adjust index if needed
	if sem.currentSquadIndex >= len(sem.allSquadIDs) && len(sem.allSquadIDs) > 0 {
		sem.currentSquadIndex = 0
	}

	// Refresh all UI elements (squad list, formation grid, status)
	sem.refreshSquadSelector()
	sem.refreshCurrentSquad()
	sem.refreshRosterList()
	sem.updateNavigationButtons()
}
