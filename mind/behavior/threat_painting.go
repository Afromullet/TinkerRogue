package behavior

import "game_main/core/coords"

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
func PaintThreatToMap(
	threatMap map[coords.LogicalPosition]float64,
	center coords.LogicalPosition,
	radius int,
	threatValue float64,
	falloffFunc FalloffFunc,
) {
	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			pos := coords.LogicalPosition{X: center.X + dx, Y: center.Y + dy}
			distance := center.ChebyshevDistance(&pos)

			// Skip the center position (distance == 0): it is the squad's own tile, not a threat target.
			if distance > 0 && distance <= radius {
				falloff := falloffFunc(distance, radius)
				threatMap[pos] += threatValue * falloff
			}
		}
	}
}
