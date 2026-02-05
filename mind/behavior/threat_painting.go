package behavior

import "game_main/world/coords"

// FalloffFunc defines how threat decreases with distance
// Returns multiplier in range [0.0, 1.0]
type FalloffFunc func(distance, maxRange int) float64

// Common falloff functions

// LinearFalloff decreases linearly with distance
var LinearFalloff = func(distance, maxRange int) float64 {
	return 1.0 - (float64(distance) / float64(maxRange+1))
}

// NoFalloff maintains full threat at all ranges
var NoFalloff = func(distance, maxRange int) float64 {
	return 1.0
}

// PaintThreatToMap paints threat values onto a map with configurable falloff.
// If trackPositions is true, returns slice of painted positions (excludes center).
// If trackPositions is false, returns nil.
func PaintThreatToMap(
	threatMap map[coords.LogicalPosition]float64,
	center coords.LogicalPosition,
	radius int,
	threatValue float64,
	falloffFunc FalloffFunc,
	trackPositions bool,
) []coords.LogicalPosition {
	var paintedPositions []coords.LogicalPosition

	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			pos := coords.LogicalPosition{X: center.X + dx, Y: center.Y + dy}
			distance := center.ChebyshevDistance(&pos)

			// Skip center position (distance == 0) when tracking
			// Center is the squad's own position - not a valid threat target
			if distance > 0 && distance <= radius {
				falloff := falloffFunc(distance, radius)
				threatMap[pos] += threatValue * falloff
				if trackPositions {
					paintedPositions = append(paintedPositions, pos)
				}
			}
		}
	}

	return paintedPositions
}
