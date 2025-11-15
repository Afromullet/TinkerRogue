package squads

import (
	"fmt"
	"game_main/common"
	"strings"

	"github.com/bytearena/ecs"
)

// VisualizeSquad creates a text representation of a squad's 3x3 grid
// showing unit EntityIDs and their multi-cell occupancy
func VisualizeSquad(squadID ecs.EntityID, squadmanager *common.EntityManager) string {
	var output strings.Builder

	// Get squad info
	squadEntity := GetSquadEntity(squadID, squadmanager)
	if squadEntity == nil {
		return fmt.Sprintf("Squad %d not found\n", squadID)
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	// Header
	output.WriteString(fmt.Sprintf("=== Squad: %s (ID: %d) ===\n", squadData.Name, squadID))
	output.WriteString(fmt.Sprintf("Morale: %d | Turns: %d | Max Units: %d\n\n",
		squadData.Morale, squadData.TurnCount, squadData.MaxUnits))

	// Build the grid: 3x3 array of entity IDs
	// -1 means empty, otherwise stores the entity ID
	grid := [3][3]int64{}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			grid[r][c] = -1 // Initialize as empty
		}
	}

	// Get all units in the squad
	unitIDs := GetUnitIDsInSquad(squadID, squadmanager)

	// Fill the grid with unit IDs
	for _, unitID := range unitIDs {
		if !squadmanager.HasComponentByIDWithTag(unitID, SquadMemberTag, GridPositionComponent) {
			continue
		}

		gridPos := common.GetComponentTypeByIDWithTag[*GridPositionData](squadmanager, unitID, SquadMemberTag, GridPositionComponent)
		if gridPos == nil {
			continue
		}

		// Mark all occupied cells with this unit's ID
		occupiedCells := gridPos.GetOccupiedCells()
		for _, cell := range occupiedCells {
			row, col := cell[0], cell[1]
			if row >= 0 && row < 3 && col >= 0 && col < 3 {
				grid[row][col] = int64(unitID)
			}
		}
	}

	// Calculate column widths based on longest entity ID
	maxWidth := 5 // Minimum width for "Empty"
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if grid[r][c] != -1 {
				idStr := fmt.Sprintf("%d", grid[r][c])
				if len(idStr) > maxWidth {
					maxWidth = len(idStr)
				}
			}
		}
	}

	// Column header
	output.WriteString("     ") // Space for row label
	for c := 0; c < 3; c++ {
		output.WriteString(fmt.Sprintf("│ Col %d ", c))
		output.WriteString(strings.Repeat(" ", maxWidth-1))
	}
	output.WriteString("│\n")

	// Top border
	output.WriteString("─────")
	for c := 0; c < 3; c++ {
		output.WriteString("┼───────")
		output.WriteString(strings.Repeat("─", maxWidth-1))
	}
	output.WriteString("┤\n")

	// Grid rows
	for r := 0; r < 3; r++ {
		// Row label
		output.WriteString(fmt.Sprintf("Row %d", r))

		// Cells
		for c := 0; c < 3; c++ {
			output.WriteString(" │ ")

			if grid[r][c] == -1 {
				// Empty cell
				cellContent := "Empty"
				padding := maxWidth - len(cellContent)
				output.WriteString(cellContent)
				output.WriteString(strings.Repeat(" ", padding))
			} else {
				// Occupied cell - show entity ID
				cellContent := fmt.Sprintf("%d", grid[r][c])
				padding := maxWidth - len(cellContent)
				output.WriteString(cellContent)
				output.WriteString(strings.Repeat(" ", padding))
			}
		}
		output.WriteString(" │\n")

		// Row separator (except after last row)
		if r < 2 {
			output.WriteString("─────")
			for c := 0; c < 3; c++ {
				output.WriteString("┼───────")
				output.WriteString(strings.Repeat("─", maxWidth-1))
			}
			output.WriteString("┤\n")
		}
	}

	// Bottom border
	output.WriteString("─────")
	for c := 0; c < 3; c++ {
		output.WriteString("┴───────")
		output.WriteString(strings.Repeat("─", maxWidth-1))
	}
	output.WriteString("┘\n")

	// Unit details
	if len(unitIDs) > 0 {
		output.WriteString("\nUnit Details:\n")
		for _, unitID := range unitIDs {
			gridPos := common.GetComponentTypeByIDWithTag[*GridPositionData](squadmanager, unitID, SquadMemberTag, GridPositionComponent)
			role := common.GetComponentTypeByIDWithTag[*UnitRoleData](squadmanager, unitID, SquadMemberTag, UnitRoleComponent)

			if gridPos == nil || role == nil {
				continue
			}

			// Get health if available
			healthInfo := ""
			attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
			if attr != nil {
				healthInfo = fmt.Sprintf(" | HP: %d/%d", attr.CurrentHealth, attr.MaxHealth)
			}

			// Check if leader
			leaderStr := ""
			if squadmanager.HasComponentByIDWithTag(unitID, SquadMemberTag, LeaderComponent) {
				leaderStr = " [LEADER]"
			}

			output.WriteString(fmt.Sprintf("  • ID %d: Position (%d,%d) Size %dx%d | Role: %s%s%s\n",
				unitID,
				gridPos.AnchorRow, gridPos.AnchorCol,
				gridPos.Width, gridPos.Height,
				role.Role.String(),
				healthInfo,
				leaderStr))
		}
	} else {
		output.WriteString("\nNo units in squad\n")
	}

	return output.String()
}
