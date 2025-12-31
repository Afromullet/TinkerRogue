package main

import (
	"fmt"
	"strings"
)

// RenderSquadGrid renders a 3x3 squad formation grid with box-drawing characters.
// prefix is "A" for attacker or "D" for defender.
func RenderSquadGrid(units []UnitSnapshot, squadName string, prefix string) string {
	// Create 3x3 grid (9 cells), each cell has 2 lines
	type Cell struct {
		line1 string // Unit ID (e.g., "A71")
		line2 string // Abbreviated name (e.g., "Mrk")
	}

	grid := make([]Cell, 9)

	// Initialize all cells as empty
	for i := range grid {
		grid[i] = Cell{line1: "", line2: ""}
	}

	// Place units in grid based on GridRow, GridCol
	for _, unit := range units {
		idx := unit.GridRow*3 + unit.GridCol
		if idx >= 0 && idx < 9 {
			grid[idx].line1 = fmt.Sprintf("%s%d", prefix, unit.UnitID)
			grid[idx].line2 = abbreviateName(unit.UnitName)
		}
	}

	var sb strings.Builder

	// Top border: ┌─────┬─────┬─────┐
	sb.WriteString("┌─────┬─────┬─────┐")
	if squadName != "" {
		sb.WriteString("  ")
		sb.WriteString(squadName)
	}
	sb.WriteString("\n")

	// Render 3 rows of cells
	for row := 0; row < 3; row++ {
		// Line 1 of cells (unit IDs)
		sb.WriteString("│")
		for col := 0; col < 3; col++ {
			idx := row*3 + col
			cell := grid[idx]
			sb.WriteString(padCenter(cell.line1, 5))
			sb.WriteString("│")
		}
		sb.WriteString("\n")

		// Line 2 of cells (unit names)
		sb.WriteString("│")
		for col := 0; col < 3; col++ {
			idx := row*3 + col
			cell := grid[idx]
			sb.WriteString(padCenter(cell.line2, 5))
			sb.WriteString("│")
		}
		sb.WriteString("\n")

		// Horizontal separator (except after last row)
		if row < 2 {
			sb.WriteString("├─────┼─────┼─────┤\n")
		}
	}

	// Bottom border: └─────┴─────┴─────┘
	sb.WriteString("└─────┴─────┴─────┘\n")

	return sb.String()
}

// abbreviateName returns abbreviated unit name.
// If name is 4 chars or less, returns as-is.
// Otherwise, returns first 3-4 characters.
func abbreviateName(name string) string {
	if len(name) <= 4 {
		return name
	}
	// Return first 4 chars for consistency with cell width
	if len(name) >= 4 {
		return name[:4]
	}
	return name
}

// padCenter centers text within a field of given width.
// If text is longer than width, truncates from right.
func padCenter(text string, width int) string {
	textLen := len(text)

	if textLen >= width {
		// Truncate if too long
		return text[:width]
	}

	// Calculate padding
	totalPad := width - textLen
	leftPad := totalPad / 2
	rightPad := totalPad - leftPad

	return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
}
