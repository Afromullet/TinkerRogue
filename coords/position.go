// Package coords provides unified coordinate system types and utilities.
// This package consolidates all position and coordinate management that was
// previously scattered across common and graphics packages.
package coords

import (
	"math"
)

// LogicalPosition represents a position in the game world (tile-based coordinates).
// This is the unified type that replaces both common.Position and graphics.LogicalPosition.
// All game entities, map coordinates, and movement targets use this type.
type LogicalPosition struct {
	X, Y int
}

// PixelPosition represents screen/pixel coordinates (only used for rendering).
// This is kept separate as it serves a different purpose from logical coordinates.
type PixelPosition struct {
	X, Y int
}

// IsEqual checks if two logical positions are identical.
// Moved from common.Position to maintain compatibility.
func (p *LogicalPosition) IsEqual(other *LogicalPosition) bool {
	return p.X == other.X && p.Y == other.Y
}

// ManhattanDistance calculates the Manhattan distance between two logical positions.
// Moved from common.Position to maintain compatibility.
func (p *LogicalPosition) ManhattanDistance(other *LogicalPosition) int {
	xDist := math.Abs(float64(p.X - other.X))
	yDist := math.Abs(float64(p.Y - other.Y))
	return int(xDist) + int(yDist)
}

// ChebyshevDistance calculates the Chebyshev distance between two logical positions.
// Moved from common.Position to maintain compatibility.
func (p *LogicalPosition) ChebyshevDistance(other *LogicalPosition) int {
	xDist := math.Abs(float64(p.X - other.X))
	yDist := math.Abs(float64(p.Y - other.Y))
	return int(math.Max(xDist, yDist))
}

// InRange checks if another position is within Manhattan distance range.
// Moved from common.Position to maintain compatibility.
func (p *LogicalPosition) InRange(other *LogicalPosition, distance int) bool {
	return p.ManhattanDistance(other) <= distance
}

// NewLogicalPosition creates a new logical position with the given coordinates.
func NewLogicalPosition(x, y int) LogicalPosition {
	return LogicalPosition{X: x, Y: y}
}