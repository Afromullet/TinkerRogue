package guisquads

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

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

// UpdateDisplayForPlacedUnit updates grid display after a unit has been placed (via service)
func (gem *GridEditorManager) UpdateDisplayForPlacedUnit(unitID ecs.EntityID, unitTemplate *squads.UnitTemplate, row, col int, rosterEntryID ecs.EntityID) error {
	// Get the unit's grid position component to find all occupied cells
	gridPosData := common.GetComponentTypeByID[*squads.GridPositionData](gem.entityManager, unitID, squads.GridPositionComponent)
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
			cell.rosterEntryID = rosterEntryID

			// Mark cell as occupied - show if it's the anchor or a secondary cell
			if cellRow == row && cellCol == col {
				// Anchor cell - show unit name and size
				sizeInfo := ""
				if totalCells > 1 {
					sizeInfo = fmt.Sprintf(" (%dx%d)", unitTemplate.GridWidth, unitTemplate.GridHeight)
				}
				leaderMarker := ""
				if unitID == gem.currentLeaderID {
					leaderMarker = " ★"
				}
				cellText := fmt.Sprintf("%s%s%s\n%s\n[%d,%d]", unitTemplate.Name, sizeInfo, leaderMarker, unitTemplate.Role.String(), cellRow, cellCol)
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
				cellText := fmt.Sprintf("%s%s\n%s [%d,%d]\n[%d,%d]", unitTemplate.Name, leaderMarker, direction, row, col, cellRow, cellCol)
				cell.button.Text().Label = cellText
			}
		}
	}

	fmt.Printf("Displayed %s (size %dx%d) at anchor [%d,%d]\n", unitTemplate.Name, unitTemplate.GridWidth, unitTemplate.GridHeight, row, col)
	return nil
}

// UpdateDisplayForRemovedUnit updates grid display after a unit has been removed (via service)
func (gem *GridEditorManager) UpdateDisplayForRemovedUnit(unitID ecs.EntityID) {
	// Clear all cells that contained this unit
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cell := gem.gridCells[row][col]
			if cell.unitID == unitID {
				cell.unitID = 0
				cell.rosterEntryID = 0
				cell.button.Text().Label = fmt.Sprintf("Empty\n[%d,%d]", row, col)
			}
		}
	}

	// Clear leader if it was the removed unit
	if gem.currentLeaderID == unitID {
		gem.currentLeaderID = 0
	}

	fmt.Printf("Updated display for removed unit %d\n", unitID)
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

			// This replaces 3 separate GetComponentTypeByID calls with just 1
			entity := gem.entityManager.FindEntityByID(cell.unitID)
			if entity == nil {
				continue
			}

			// Get unit grid position
			gridPosData := common.GetComponentType[*squads.GridPositionData](entity, squads.GridPositionComponent)
			if gridPosData == nil {
				continue
			}

			// Check if this cell is the anchor
			isAnchor := (gridPosData.AnchorRow == row && gridPosData.AnchorCol == col)

			// Get unit name from entity
			nameData := common.GetComponentType[*common.Name](entity, common.NameComponent)
			unitName := "Unknown"
			if nameData != nil {
				unitName = nameData.NameStr
			}

			// Get unit role from entity
			roleData := common.GetComponentType[*squads.UnitRoleData](entity, squads.UnitRoleComponent)
			roleStr := "Unknown"
			if roleData != nil {
				roleStr = roleData.Role.String()
			}

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
				cellText := fmt.Sprintf("%s%s%s\n%s\n[%d,%d]", unitName, sizeInfo, leaderMarker, roleStr, row, col)
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
				cellText := fmt.Sprintf("%s%s\n%s [%d,%d]\n[%d,%d]", unitName, leaderMarker, direction, gridPosData.AnchorRow, gridPosData.AnchorCol, row, col)
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
			cell.rosterEntryID = 0
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
