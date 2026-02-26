package guisquads

import (
	"fmt"
	"game_main/common"
	"game_main/gui/guiinspect"
	"game_main/tactical/squadcommands"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// Grid interaction logic for SquadEditorMode

// onGridCellClicked handles clicking a grid cell
func (sem *SquadEditorMode) onGridCellClicked(row, col int) {
	if !sem.squadNav.HasSquads() {
		sem.SetStatus("No squad selected")
		return
	}

	currentSquadID := sem.squadNav.CurrentID()

	// Check if there's a unit at this position
	unitIDs := squads.GetUnitIDsAtGridPosition(currentSquadID, row, col, sem.Queries.ECSManager)

	if len(unitIDs) > 0 {
		// Unit exists - select it for moving and show units panel
		sem.selectedUnitID = unitIDs[0]
		sem.selectedGridCell = nil
		sem.subMenus.Show("units")

		nameStr := common.GetEntityName(sem.Queries.ECSManager, sem.selectedUnitID, "Unit")

		sem.SetStatus(fmt.Sprintf("Selected unit: %s. Click another cell to move", nameStr))
	} else if sem.selectedUnitID != 0 {
		// Empty cell clicked with unit selected - move unit here
		sem.moveSelectedUnitToCell(row, col)
		sem.selectedUnitID = 0
		sem.selectedGridCell = nil
	} else {
		// Empty cell clicked with no unit selected - show roster for placement
		sem.selectedGridCell = &GridCell{Row: row, Col: col}
		sem.subMenus.Show("roster")
		sem.SetStatus(fmt.Sprintf("Selected cell [%d,%d]. Click 'Add to Squad' to place a unit here", row, col))
	}
}

// moveSelectedUnitToCell moves the currently selected unit to the specified cell
func (sem *SquadEditorMode) moveSelectedUnitToCell(row, col int) {
	if sem.selectedUnitID == 0 {
		return
	}

	currentSquadID := sem.squadNav.CurrentID()

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

// loadSquadFormation loads squad units into the 3x3 grid display
func (sem *SquadEditorMode) loadSquadFormation(squadID ecs.EntityID) {
	// Clear grids first
	guiinspect.ClearGridCells(sem.gridCells)
	guiinspect.ClearGridCells(sem.attackGridCells)

	// Get units in squad and display them
	unitIDs := sem.Queries.SquadCache.GetUnitIDsInSquad(squadID)

	for _, unitID := range unitIDs {
		gridPos := common.GetComponentTypeByID[*squads.GridPositionData](
			sem.Queries.ECSManager, unitID, squads.GridPositionComponent)
		if gridPos == nil {
			continue
		}

		nameStr := common.GetEntityName(sem.Queries.ECSManager, unitID, "Unit")

		// Check if leader
		isLeader := sem.Queries.ECSManager.HasComponent(unitID, squads.LeaderComponent)
		if isLeader {
			nameStr = "[L] " + nameStr
		}

		// Update grid cell
		if gridPos.AnchorRow >= 0 && gridPos.AnchorRow < 3 && gridPos.AnchorCol >= 0 && gridPos.AnchorCol < 3 {
			sem.gridCells[gridPos.AnchorRow][gridPos.AnchorCol].Text().Label = nameStr
		}
	}

	sem.refreshAttackPattern()
}

// refreshAttackPattern updates the attack pattern grid if visible
func (sem *SquadEditorMode) refreshAttackPattern() {
	if !sem.showAttackPattern || !sem.squadNav.HasSquads() {
		return
	}
	pattern := squads.ComputeGenericAttackPattern(sem.squadNav.CurrentID(), sem.Queries.ECSManager)
	guiinspect.PopulateAttackGridCells(sem.attackGridCells, pattern)
}
