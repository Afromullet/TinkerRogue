package common

import "game_main/core/coords"

// GridMovement performs Chebyshev flood-fill over a tile grid. Callers supply
// the domain rule for whether a tile is enterable; everything else (bounding
// box iteration, distance check, exclusion of the origin tile) is shared.
//
// Extracted from CommanderMovementSystem and CombatMovementSystem to remove
// duplicated flood-fill logic.
type GridMovement struct {
	CanEnter func(targetPos coords.LogicalPosition) bool
}

// ValidTilesInRange returns every tile within Chebyshev distance `rng` of
// `from` that passes CanEnter. The origin tile itself (distance 0) is always
// excluded — neither caller wanted "you can move to where you already are."
func (g GridMovement) ValidTilesInRange(from coords.LogicalPosition, rng int) []coords.LogicalPosition {
	if rng <= 0 || g.CanEnter == nil {
		return nil
	}
	var validTiles []coords.LogicalPosition
	for x := from.X - rng; x <= from.X+rng; x++ {
		for y := from.Y - rng; y <= from.Y+rng; y++ {
			testPos := coords.LogicalPosition{X: x, Y: y}
			distance := from.ChebyshevDistance(&testPos)
			if distance == 0 || distance > rng {
				continue
			}
			if g.CanEnter(testPos) {
				validTiles = append(validTiles, testPos)
			}
		}
	}
	return validTiles
}
