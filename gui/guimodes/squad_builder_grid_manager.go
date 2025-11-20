package guimodes

import (
	"fmt"
	"game_main/common"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// GridEditorManager manages 3x3 grid state and all grid operations
type GridEditorManager struct {
	gridCells       [3][3]*GridCellButton
	currentLeaderID ecs.EntityID
	entityManager   *common.EntityManager
}

// NewGridEditorManager creates a new grid editor manager
func NewGridEditorManager(entityManager *common.EntityManager) *GridEditorManager {
	return &GridEditorManager{
		entityManager:   entityManager,
		currentLeaderID: 0,
	}
}

// SetGridCells assigns the grid cell button references
func (gem *GridEditorManager) SetGridCells(cells [3][3]*GridCellButton) {
	gem.gridCells = cells
}

// PlaceUnitInCell places a unit at a grid position and updates display
func (gem *GridEditorManager) PlaceUnitInCell(row, col, unitIndex int, squadID ecs.EntityID) error {
	unit := squads.Units[unitIndex]

	// Attempt to add unit to squad (this checks capacity constraints)
	err := squads.AddUnitToSquad(squadID, gem.entityManager, unit, row, col)
	if err != nil {
		return err
	}

	// Find the newly created unit entity
	unitIDs := squads.GetUnitIDsAtGridPosition(squadID, row, col, gem.entityManager)
	if len(unitIDs) == 0 {
		return fmt.Errorf("unit was not placed correctly")
	}

	unitID := unitIDs[0]

	// Get the unit's grid position component to find all occupied cells
	gridPosData := common.GetComponentTypeByIDWithTag[*squads.GridPositionData](gem.entityManager, unitID, squads.SquadMemberTag, squads.GridPositionComponent)
	if gridPosData == nil {
		return fmt.Errorf("unit has no grid position data")
	}

	// Update ALL occupied cells
	occupiedCells := gridPosData.GetOccupiedCells()
	totalCells := len(occupiedCells)

	for _, cellPos := range occupiedCells {
		cellRow, cellCol := cellPos[0], cellPos[1]
		if cellRow >= 0 && cellRow < 3 && cellCol >= 0 && cellCol < 3 {
			cell := gem.gridCells[cellRow][cellCol]
			cell.unitID = unitID
			cell.unitIndex = unitIndex

			// Mark cell as occupied - show if it's the anchor or a secondary cell
			if cellRow == row && cellCol == col {
				// Anchor cell - show unit name and size
				sizeInfo := ""
				if totalCells > 1 {
					sizeInfo = fmt.Sprintf(" (%dx%d)", unit.GridWidth, unit.GridHeight)
				}
				leaderMarker := ""
				if unitID == gem.currentLeaderID {
					leaderMarker = " ★"
				}
				cellText := fmt.Sprintf("%s%s%s\n%s\n[%d,%d]", unit.Name, sizeInfo, leaderMarker, unit.Role.String(), cellRow, cellCol)
				cell.button.Text().Label = cellText
			} else {
				// Secondary cell - show it's part of the unit with arrow pointing to anchor
				direction := ""
				if cellRow < row {
					direction += "↓"
				} else if cellRow > row {
					direction += "↑"
				}
				if cellCol < col {
					direction += "→"
				} else if cellCol > col {
					direction += "←"
				}
				leaderMarker := ""
				if unitID == gem.currentLeaderID {
					leaderMarker = " ★"
				}
				cellText := fmt.Sprintf("%s%s\n%s [%d,%d]\n[%d,%d]", unit.Name, leaderMarker, direction, row, col, cellRow, cellCol)
				cell.button.Text().Label = cellText
			}
		}
	}

	fmt.Printf("Placed %s (size %dx%d) at anchor [%d,%d]\n", unit.Name, unit.GridWidth, unit.GridHeight, row, col)
	return nil
}

// RemoveUnitFromCell removes a unit from a grid position and updates display
func (gem *GridEditorManager) RemoveUnitFromCell(row, col int) error {
	cell := gem.gridCells[row][col]

	if cell.unitID == 0 {
		return nil
	}

	unitID := cell.unitID

	// Get the unit's grid position to find all occupied cells BEFORE removing
	gridPosData := common.GetComponentTypeByIDWithTag[*squads.GridPositionData](gem.entityManager, unitID, squads.SquadMemberTag, squads.GridPositionComponent)
	if gridPosData == nil {
		return fmt.Errorf("could not find unit entity to remove")
	}

	var occupiedCells [][2]int
	occupiedCells = gridPosData.GetOccupiedCells()

	// Remove unit from squad
	err := squads.RemoveUnitFromSquad(unitID, gem.entityManager)
	if err != nil {
		return fmt.Errorf("failed to remove unit: %v", err)
	}

	// Clear ALL cells that were occupied by this unit
	if len(occupiedCells) > 0 {
		for _, cellPos := range occupiedCells {
			cellRow, cellCol := cellPos[0], cellPos[1]
			if cellRow >= 0 && cellRow < 3 && cellCol >= 0 && cellCol < 3 {
				targetCell := gem.gridCells[cellRow][cellCol]
				if targetCell.unitID == unitID { // Only clear if it was this unit
					targetCell.unitID = 0
					targetCell.unitIndex = -1
					targetCell.button.Text().Label = fmt.Sprintf("Empty\n[%d,%d]", cellRow, cellCol)
				}
			}
		}
	} else {
		// Fallback: only clear the clicked cell
		cell.unitID = 0
		cell.unitIndex = -1
		cell.button.Text().Label = fmt.Sprintf("Empty\n[%d,%d]", row, col)
	}

	// Clear leader if it was the removed unit
	if gem.currentLeaderID == unitID {
		gem.currentLeaderID = 0
	}

	fmt.Printf("Removed unit from [%d,%d]\n", row, col)
	return nil
}

// SetLeader sets a unit as the squad leader
func (gem *GridEditorManager) SetLeader(unitID ecs.EntityID) {
	gem.currentLeaderID = unitID
}

// GetLeader returns the current leader unit ID
func (gem *GridEditorManager) GetLeader() ecs.EntityID {
	return gem.currentLeaderID
}

// RefreshGridDisplay updates all grid cell displays (for leader markers, etc.)
func (gem *GridEditorManager) RefreshGridDisplay() {
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cell := gem.gridCells[row][col]
			if cell.unitID == 0 {
				cell.button.Text().Label = fmt.Sprintf("Empty\n[%d,%d]", row, col)
				continue
			}

			// Get unit info
			gridPosData := common.GetComponentTypeByIDWithTag[*squads.GridPositionData](gem.entityManager, cell.unitID, squads.SquadMemberTag, squads.GridPositionComponent)
			if gridPosData == nil {
				continue
			}

			// Check if this cell is the anchor
			isAnchor := (gridPosData.AnchorRow == row && gridPosData.AnchorCol == col)

			// Get unit template info
			if cell.unitIndex < 0 || cell.unitIndex >= len(squads.Units) {
				continue
			}
			unit := squads.Units[cell.unitIndex]

			leaderMarker := ""
			if cell.unitID == gem.currentLeaderID {
				leaderMarker = " ★"
			}

			if isAnchor {
				// Anchor cell
				sizeInfo := ""
				if gridPosData.Width > 1 || gridPosData.Height > 1 {
					sizeInfo = fmt.Sprintf(" (%dx%d)", gridPosData.Width, gridPosData.Height)
				}
				cellText := fmt.Sprintf("%s%s%s\n%s\n[%d,%d]", unit.Name, sizeInfo, leaderMarker, unit.Role.String(), row, col)
				cell.button.Text().Label = cellText
			} else {
				// Secondary cell
				direction := ""
				if row < gridPosData.AnchorRow {
					direction += "↓"
				} else if row > gridPosData.AnchorRow {
					direction += "↑"
				}
				if col < gridPosData.AnchorCol {
					direction += "→"
				} else if col > gridPosData.AnchorCol {
					direction += "←"
				}
				cellText := fmt.Sprintf("%s%s\n%s [%d,%d]\n[%d,%d]", unit.Name, leaderMarker, direction, gridPosData.AnchorRow, gridPosData.AnchorCol, row, col)
				cell.button.Text().Label = cellText
			}
		}
	}
}

// ClearGrid removes all units from the grid
func (gem *GridEditorManager) ClearGrid() {
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cell := gem.gridCells[row][col]
			cell.unitID = 0
			cell.unitIndex = -1
			cell.button.Text().Label = fmt.Sprintf("Empty\n[%d,%d]", row, col)
		}
	}
	gem.currentLeaderID = 0
}

// GetCellUnitID returns the unit ID at a specific grid cell
func (gem *GridEditorManager) GetCellUnitID(row, col int) ecs.EntityID {
	if row >= 0 && row < 3 && col >= 0 && col < 3 {
		return gem.gridCells[row][col].unitID
	}
	return 0
}
