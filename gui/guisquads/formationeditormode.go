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
	squadSelector  *widget.List // Squad selection list
	currentSquadID ecs.EntityID // Currently selected squad

	gridCells [3][3]*widget.Button // 3x3 grid of cells
}

func NewFormationEditorMode(modeManager *core.UIModeManager) *FormationEditorMode {
	mode := &FormationEditorMode{}
	mode.SetModeName("formation_editor")
	mode.SetReturnMode("squad_management") // ESC returns to squad management
	mode.ModeManager = modeManager
	return mode
}

func (fem *FormationEditorMode) Initialize(ctx *core.UIContext) error {
	return gui.NewModeBuilder(&fem.BaseMode, gui.ModeConfig{
		ModeName:   "formation_editor",
		ReturnMode: "squad_management",

		Panels: []gui.PanelSpec{
			{CustomBuild: fem.buildSquadSelector},
			{CustomBuild: fem.buildGridEditor},
			{CustomBuild: fem.buildUnitPalette},
		},

		Buttons: []gui.ButtonGroupSpec{
			{
				Position: widgets.BottomCenter(),
				Buttons: []widgets.ButtonSpec{
					{
						Text: "Apply Formation",
						OnClick: func() {
							fem.onApplyFormation()
						},
					},
					gui.ModeTransitionSpec(fem.ModeManager, "Close (ESC)", "squad_management"),
				},
			},
		},

		StatusLabel: true,
		Commands:    true,
		OnRefresh:   fem.refreshAfterUndoRedo,
	}).Build(ctx)
}

func (fem *FormationEditorMode) buildSquadSelector() *widget.Container {
	// Get all squads from ECS
	allSquadIDs := fem.Queries.SquadCache.FindAllSquads()

	// Create squad selection list using helper with formation-specific constants
	squadSelector := widgets.CreateSquadList(widgets.SquadListConfig{
		SquadIDs:      allSquadIDs,
		Manager:       fem.Queries.ECSManager,
		ScreenWidth:   fem.Layout.ScreenWidth,
		ScreenHeight:  fem.Layout.ScreenHeight,
		WidthPercent:  widgets.FormationSquadListWidth,
		HeightPercent: widgets.FormationSquadListHeight,
		OnSelect: func(squadID ecs.EntityID) {
			fem.currentSquadID = squadID
			fem.loadSquadFormation(squadID)
			squadName := fem.Queries.SquadCache.GetSquadName(squadID)
			fem.SetStatus(fmt.Sprintf("Selected squad: %s", squadName))
		},
	})
	leftPad := int(float64(fem.Layout.ScreenWidth) * widgets.PaddingStandard)
	topPad := int(float64(fem.Layout.ScreenHeight) * widgets.PaddingStandard)

	fem.squadSelector = squadSelector

	// Wrap in container with LayoutData
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(gui.AnchorStartStart(leftPad, topPad))),
	)
	container.AddChild(squadSelector)
	return container
}

func (fem *FormationEditorMode) buildGridEditor() *widget.Container {
	// Build 3x3 grid editor (center)
	gridContainer, gridCells := fem.PanelBuilders.BuildGridEditor(widgets.GridEditorConfig{
		OnCellClick: func(row, col int) {
			fem.onCellClicked(row, col)
		},
	})
	fem.gridContainer = gridContainer
	fem.gridCells = gridCells
	return gridContainer
}

func (fem *FormationEditorMode) buildUnitPalette() *widget.Container {
	// Unit type options
	unitTypes := []string{"Tank", "DPS", "Support", "Remove Unit"}

	// Create simple string list using helper with formation-specific constants
	unitPalette := widgets.CreateSimpleStringList(widgets.SimpleStringListConfig{
		Entries:       unitTypes,
		ScreenWidth:   fem.Layout.ScreenWidth,
		ScreenHeight:  fem.Layout.ScreenHeight,
		WidthPercent:  widgets.FormationPaletteWidth,
		HeightPercent: widgets.FormationPaletteHeight,
	})

	rightPad := int(float64(fem.Layout.ScreenWidth) * widgets.PaddingStandard)

	fem.unitPalette = unitPalette

	// Wrap in container with LayoutData
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(gui.AnchorEndCenter(rightPad))),
	)
	container.AddChild(unitPalette)
	return container
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
	if fem.CommandHistory.HandleInput(inputState) {
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
	unitIDs := fem.Queries.SquadCache.GetUnitIDsInSquad(squadID)

	for _, unitID := range unitIDs {
		// Get grid position component
		gridPos := common.GetComponentTypeByIDWithTag[*squads.GridPositionData](
			fem.Queries.ECSManager, unitID, squads.SquadMemberTag, squads.GridPositionComponent)
		if gridPos == nil {
			continue
		}

		// Get unit name
		nameStr := "Unit"
		if nameComp, ok := fem.Queries.ECSManager.GetComponent(unitID, common.NameComponent); ok {
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
		fem.SetStatus("No squad selected")
		return
	}

	squadName := fem.Queries.SquadCache.GetSquadName(fem.currentSquadID)

	// Show confirmation dialog
	dialog := widgets.CreateConfirmationDialog(widgets.DialogConfig{
		Title:   "Apply Formation",
		Message: fmt.Sprintf("Apply current formation to squad '%s'?\n\nThis will rearrange unit positions.\n\nYou can undo this action with Ctrl+Z.", squadName),
		OnConfirm: func() {
			// Build formation from current grid state
			formation, err := fem.buildFormationAssignments()
			if err != nil {
				fem.SetStatus(fmt.Sprintf("✗ %s", err.Error()))
				return
			}

			// Create and execute command
			cmd := squadcommands.NewChangeFormationCommand(
				fem.Queries.ECSManager,
				fem.currentSquadID,
				formation,
			)

			fem.CommandHistory.Execute(cmd)
		},
		OnCancel: func() {
			fem.SetStatus("Apply formation cancelled")
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
	unitIDs := fem.Queries.SquadCache.GetUnitIDsInSquad(fem.currentSquadID)
	if len(unitIDs) == 0 {
		return nil, fmt.Errorf("squad has no units")
	}

	// Build map of unit names to unit IDs
	unitNameToID := make(map[string]ecs.EntityID)
	for _, unitID := range unitIDs {
		nameStr := "Unit"
		if nameComp, ok := fem.Queries.ECSManager.GetComponent(unitID, common.NameComponent); ok {
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

