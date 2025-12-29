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

// QuadraticFalloff decreases faster with distance (for support/proximity effects)
var QuadraticFalloff = func(distance, maxRange int) float64 {
	normalized := float64(distance) / float64(maxRange+1)
	return 1.0 - (normalized * normalized)
}

// PaintThreatToMap paints threat values onto a map with configurable falloff
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

			if distance <= radius {
				falloff := falloffFunc(distance, radius)
				threatMap[pos] += threatValue * falloff
			}
		}
	}
}

// PaintThreatToMapWithTracking paints threat and records which positions were painted
// Useful for tracking line-of-fire zones or areas of effect
func PaintThreatToMapWithTracking(
	threatMap map[coords.LogicalPosition]float64,
	center coords.LogicalPosition,
	radius int,
	threatValue float64,
	falloffFunc FalloffFunc,
) []coords.LogicalPosition {
	var paintedPositions []coords.LogicalPosition

	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			pos := coords.LogicalPosition{X: center.X + dx, Y: center.Y + dy}
			distance := center.ChebyshevDistance(&pos)

			if distance > 0 && distance <= radius {
				falloff := falloffFunc(distance, radius)
				threatMap[pos] += threatValue * falloff
				paintedPositions = append(paintedPositions, pos)
			}
		}
	}

	return paintedPositions
}
