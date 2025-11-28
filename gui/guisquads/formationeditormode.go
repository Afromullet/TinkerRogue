package guisquads

import (
	"game_main/gui"

	"fmt"
	"game_main/common"
	"game_main/gui/core"
	"game_main/gui/widgets"
	"game_main/squads"
	"game_main/squads/squadcommands"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// FormationEditorMode provides 3x3 grid editing for squad formations
type FormationEditorMode struct {
	gui.BaseMode // Embed common mode infrastructure

	gridContainer  *widget.Container
	unitPalette    *widget.List
	actionButtons  *widget.Container
	commandHistory *gui.CommandHistory // Centralized undo/redo support
	statusLabel    *widget.Text        // Shows command results
	squadSelector  *widget.List        // Squad selection list
	currentSquadID ecs.EntityID        // Currently selected squad

	gridCells [3][3]*widget.Button // 3x3 grid of cells
}

func NewFormationEditorMode(modeManager *core.UIModeManager) *FormationEditorMode {
	mode := &FormationEditorMode{}
	mode.SetModeName("formation_editor")
	mode.ModeManager = modeManager
	return mode
}

func (fem *FormationEditorMode) Initialize(ctx *core.UIContext) error {
	// Initialize common mode infrastructure
	fem.InitializeBase(ctx)

	// Initialize command history with callbacks
	fem.commandHistory = gui.NewCommandHistory(
		fem.setStatus,
		fem.refreshAfterUndoRedo,
	)

	// Build squad selector on left side
	fem.buildSquadSelector()

	// Build 3x3 grid editor (center)
	fem.gridContainer, fem.gridCells = fem.PanelBuilders.BuildGridEditor(widgets.GridEditorConfig{
		OnCellClick: func(row, col int) {
			fem.onCellClicked(row, col)
		},
	})
	fem.RootContainer.AddChild(fem.gridContainer)

	fem.buildUnitPalette()
	fem.buildActionButtons()

	return nil
}

func (fem *FormationEditorMode) buildSquadSelector() {
	// Get all squads from ECS
	allSquadIDs := squads.FindAllSquads(fem.Context.ECSManager)

	// Build squad entries for list
	squadEntries := make([]interface{}, 0, len(allSquadIDs))
	for _, squadID := range allSquadIDs {
		squadName := squads.GetSquadName(squadID, fem.Context.ECSManager)
		squadEntries = append(squadEntries, squadName)
	}

	// Create squad selection list
	listWidth := int(float64(fem.Layout.ScreenWidth) * widgets.PanelWidthStandard)
	listHeight := int(float64(fem.Layout.ScreenHeight) * 0.3)

	fem.squadSelector = widgets.CreateListWithConfig(widgets.ListConfig{
		Entries:   squadEntries,
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
		OnEntrySelected: func(e interface{}) {
			// Find the selected squad ID
			selectedName := e.(string)
			for _, squadID := range allSquadIDs {
				if squads.GetSquadName(squadID, fem.Context.ECSManager) == selectedName {
					fem.currentSquadID = squadID
					fem.loadSquadFormation(squadID)
					fem.setStatus(fmt.Sprintf("Selected squad: %s", selectedName))
					break
				}
			}
		},
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
		},
	})

	fem.RootContainer.AddChild(fem.squadSelector)
}

func (fem *FormationEditorMode) buildUnitPalette() {
	// Left side unit palette
	listWidth := int(float64(fem.Layout.ScreenWidth) * widgets.PanelWidthStandard)
	listHeight := int(float64(fem.Layout.ScreenHeight) * widgets.PanelHeightExtraTall)

	// Unit types
	entries := []interface{}{
		"Tank",
		"DPS",
		"Support",
		"Remove Unit",
	}

	fem.unitPalette = widgets.CreateListWithConfig(widgets.ListConfig{
		Entries:   entries,
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		},
	})

	fem.RootContainer.AddChild(fem.unitPalette)
}

func (fem *FormationEditorMode) buildActionButtons() {
	// Build action buttons container using helper
	fem.actionButtons = gui.CreateBottomCenterButtonContainer(fem.PanelBuilders)

	// Apply Formation button (placeholder - would use ChangeFormationCommand)
	applyBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Apply Formation",
		OnClick: func() {
			fem.onApplyFormation()
		},
	})
	fem.actionButtons.AddChild(applyBtn)

	// Undo/Redo buttons from CommandHistory
	fem.actionButtons.AddChild(fem.commandHistory.CreateUndoButton())
	fem.actionButtons.AddChild(fem.commandHistory.CreateRedoButton())

	// Create close button to return to squad management (Overworld context)
	closeBtn := gui.CreateCloseButton(fem.ModeManager, "squad_management", "Close (ESC)")
	fem.actionButtons.AddChild(closeBtn)

	// Status label
	fem.statusLabel = widgets.CreateSmallLabel("")
	fem.RootContainer.AddChild(fem.statusLabel)

	fem.RootContainer.AddChild(fem.actionButtons)
}

func (fem *FormationEditorMode) onCellClicked(row, col int) {
	// Get selected unit type from palette
	selectedEntry := fem.unitPalette.SelectedEntry()
	if selectedEntry == nil {
		return
	}

	unitType := selectedEntry.(string)
	fmt.Printf("Placed %s at [%d,%d]\n", unitType, row, col)

	// Update button text to show placed unit
	fem.gridCells[row][col].Text().Label = unitType
}

func (fem *FormationEditorMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Formation Editor Mode")
	return nil
}

func (fem *FormationEditorMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Formation Editor Mode")
	return nil
}

func (fem *FormationEditorMode) Update(deltaTime float64) error {
	return nil
}

func (fem *FormationEditorMode) Render(screen *ebiten.Image) {
	// No custom rendering needed
}

func (fem *FormationEditorMode) HandleInput(inputState *core.InputState) bool {
	// Handle common input (ESC key)
	if fem.HandleCommonInput(inputState) {
		return true
	}

	// Handle undo/redo input (Ctrl+Z, Ctrl+Y)
	if fem.commandHistory.HandleInput(inputState) {
		return true
	}

	return false
}

// loadSquadFormation loads the current formation of a squad into the grid
func (fem *FormationEditorMode) loadSquadFormation(squadID ecs.EntityID) {
	// Clear grid first
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			fem.gridCells[row][col].Text().Label = ""
		}
	}

	// Get units in squad and their grid positions
	unitIDs := squads.GetUnitIDsInSquad(squadID, fem.Context.ECSManager)

	for _, unitID := range unitIDs {
		// Get grid position component
		entity := common.FindEntityByIDWithTag(fem.Context.ECSManager, unitID, squads.SquadMemberTag)
		if entity == nil {
			continue
		}

		gridPos := common.GetComponentType[*squads.GridPositionData](entity, squads.GridPositionComponent)
		if gridPos == nil {
			continue
		}

		// Get unit name
		nameStr := "Unit"
		if nameComp, ok := fem.Context.ECSManager.GetComponent(unitID, common.NameComponent); ok {
			if name := nameComp.(*common.Name); name != nil {
				nameStr = name.NameStr
			}
		}

		// Update grid cell
		if gridPos.AnchorRow >= 0 && gridPos.AnchorRow < 3 && gridPos.AnchorCol >= 0 && gridPos.AnchorCol < 3 {
			fem.gridCells[gridPos.AnchorRow][gridPos.AnchorCol].Text().Label = nameStr
		}
	}
}

// onApplyFormation applies the current formation using ChangeFormationCommand
func (fem *FormationEditorMode) onApplyFormation() {
	if fem.currentSquadID == 0 {
		fem.setStatus("No squad selected")
		return
	}

	squadName := squads.GetSquadName(fem.currentSquadID, fem.Context.ECSManager)

	// Show confirmation dialog
	dialog := widgets.CreateConfirmationDialog(widgets.DialogConfig{
		Title:   "Apply Formation",
		Message: fmt.Sprintf("Apply current formation to squad '%s'?\n\nThis will rearrange unit positions.\n\nYou can undo this action with Ctrl+Z.", squadName),
		OnConfirm: func() {
			// Build formation from current grid state
			formation, err := fem.buildFormationAssignments()
			if err != nil {
				fem.setStatus(fmt.Sprintf("✗ %s", err.Error()))
				return
			}

			// Create and execute command
			cmd := squadcommands.NewChangeFormationCommand(
				fem.Context.ECSManager,
				fem.currentSquadID,
				formation,
			)

			fem.commandHistory.Execute(cmd)
		},
		OnCancel: func() {
			fem.setStatus("Apply formation cancelled")
		},
	})

	fem.GetEbitenUI().AddWindow(dialog)
}

// buildFormationAssignments builds formation assignments from current grid state
func (fem *FormationEditorMode) buildFormationAssignments() ([]squadcommands.FormationAssignment, error) {
	if fem.currentSquadID == 0 {
		return nil, fmt.Errorf("no squad selected")
	}

	// Get all units in squad
	unitIDs := squads.GetUnitIDsInSquad(fem.currentSquadID, fem.Context.ECSManager)
	if len(unitIDs) == 0 {
		return nil, fmt.Errorf("squad has no units")
	}

	// Build map of unit names to unit IDs
	unitNameToID := make(map[string]ecs.EntityID)
	for _, unitID := range unitIDs {
		nameStr := "Unit"
		if nameComp, ok := fem.Context.ECSManager.GetComponent(unitID, common.NameComponent); ok {
			if name := nameComp.(*common.Name); name != nil {
				nameStr = name.NameStr
			}
		}
		unitNameToID[nameStr] = unitID
	}

	// Scan grid and build assignments
	assignments := make([]squadcommands.FormationAssignment, 0)
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cellLabel := fem.gridCells[row][col].Text().Label
			if cellLabel != "" && cellLabel != "Empty" {
				// Find unit ID for this label
				unitID, found := unitNameToID[cellLabel]
				if !found {
					// Skip units not in squad
					continue
				}

				// Add assignment
				assignment := squadcommands.FormationAssignment{
					UnitID:  unitID,
					GridRow: row,
					GridCol: col,
				}
				assignments = append(assignments, assignment)
			}
		}
	}

	if len(assignments) == 0 {
		return nil, fmt.Errorf("no units positioned in formation")
	}

	return assignments, nil
}

// refreshAfterUndoRedo is called after successful undo/redo operations
func (fem *FormationEditorMode) refreshAfterUndoRedo() {
	if fem.currentSquadID != 0 {
		fem.loadSquadFormation(fem.currentSquadID)
	}
}

// setStatus updates the status label with a message
func (fem *FormationEditorMode) setStatus(message string) {
	if fem.statusLabel != nil {
		fem.statusLabel.Label = message
	}
	fmt.Println(message) // Also log to console
}
