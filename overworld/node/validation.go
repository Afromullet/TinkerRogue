package node

import (
	"math"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"
)

// PlacementResult indicates whether a placement is valid and why.
type PlacementResult struct {
	Valid  bool
	Reason string
}

// ValidatePlacement checks whether a node can be placed at the given position.
// Owner-agnostic: player passes their position + owned node positions as anchors.
// AI passes its faction's node positions as anchors.
func ValidatePlacement(manager *common.EntityManager, pos coords.LogicalPosition, ownerID string, anchorPositions []coords.LogicalPosition) PlacementResult {
	// Check walkable terrain
	if !core.IsTileWalkable(pos) {
		return PlacementResult{Valid: false, Reason: "Terrain is not walkable"}
	}

	// Check no existing node at position
	if IsAnyNodeAtPosition(manager, pos) {
		return PlacementResult{Valid: false, Reason: "Position already occupied by a node"}
	}

	// Check max node limit (uses same config for all owners)
	maxNodes := core.GetMaxPlayerNodes()
	if CountNodesByOwner(manager, ownerID) >= maxNodes {
		return PlacementResult{Valid: false, Reason: "Maximum node limit reached"}
	}

	// Check placement range from any anchor position
	maxRange := float64(core.GetMaxPlacementRange())
	inRange := false
	for _, anchor := range anchorPositions {
		dx := float64(pos.X - anchor.X)
		dy := float64(pos.Y - anchor.Y)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist <= maxRange {
			inRange = true
			break
		}
	}

	if !inRange {
		return PlacementResult{Valid: false, Reason: "Too far from anchor positions"}
	}

	return PlacementResult{Valid: true, Reason: ""}
}
