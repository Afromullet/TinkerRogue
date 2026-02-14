package node

import (
	"fmt"
	"math"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/templates"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
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
	if core.IsAnyNodeAtPosition(manager, pos) {
		return PlacementResult{Valid: false, Reason: "Position already occupied by a node"}
	}

	// Check max node limit (uses same config for all owners)
	maxNodes := templates.OverworldConfigTemplate.PlayerNodes.MaxNodes
	if CountNodesByOwner(manager, ownerID) >= maxNodes {
		return PlacementResult{Valid: false, Reason: "Maximum node limit reached"}
	}

	// Check placement range from any anchor position
	maxRange := float64(templates.OverworldConfigTemplate.PlayerNodes.MaxPlacementRange)
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

// ValidatePlayerPlacement checks whether a player node can be placed at the given position.
// Uses commander positions and existing player node positions as anchors.
// commanderPos is the selected commander's position (placement range is measured from commanders, not the player avatar).
func ValidatePlayerPlacement(manager *common.EntityManager, pos coords.LogicalPosition, commanderPos *coords.LogicalPosition) PlacementResult {
	// Check walkable terrain
	if !core.IsTileWalkable(pos) {
		return PlacementResult{Valid: false, Reason: "Terrain is not walkable"}
	}

	// Check no existing node at position
	if core.IsAnyNodeAtPosition(manager, pos) {
		return PlacementResult{Valid: false, Reason: "Position already occupied by a node"}
	}

	// Check max node limit
	maxNodes := templates.OverworldConfigTemplate.PlayerNodes.MaxNodes
	if CountPlayerNodes(manager) >= maxNodes {
		return PlacementResult{Valid: false, Reason: "Maximum node limit reached"}
	}

	// Check placement range from selected commander or any existing player node
	maxRange := float64(templates.OverworldConfigTemplate.PlayerNodes.MaxPlacementRange)
	inRange := false

	// Check distance from selected commander's position
	if commanderPos != nil {
		dx := float64(pos.X - commanderPos.X)
		dy := float64(pos.Y - commanderPos.Y)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist <= maxRange {
			inRange = true
		}
	}

	// Check distance from any existing player node
	if !inRange {
		nearestDist := GetNearestPlayerNodeDistance(manager, pos)
		if nearestDist <= maxRange {
			inRange = true
		}
	}

	if !inRange {
		return PlacementResult{Valid: false, Reason: "Too far from commander or existing nodes"}
	}

	return PlacementResult{Valid: true, Reason: ""}
}

// ValidatePlayerPlacementWithCost extends ValidatePlayerPlacement with a resource affordability check.
// Returns invalid if the player cannot afford the node type's resource cost.
// commanderPos is the selected commander's position used for range validation.
func ValidatePlayerPlacementWithCost(manager *common.EntityManager, pos coords.LogicalPosition, commanderPos *coords.LogicalPosition, playerEntityID ecs.EntityID, nodeTypeID string) PlacementResult {
	// Run all existing placement checks first
	result := ValidatePlayerPlacement(manager, pos, commanderPos)
	if !result.Valid {
		return result
	}

	// Check resource affordability
	nodeDef := core.GetNodeRegistry().GetNodeByID(nodeTypeID)
	if nodeDef == nil {
		return PlacementResult{Valid: false, Reason: "Unknown node type"}
	}

	stockpile := common.GetResourceStockpile(playerEntityID, manager)
	if stockpile == nil {
		return PlacementResult{Valid: false, Reason: "No resource stockpile found"}
	}

	if !core.CanAfford(stockpile, nodeDef.Cost) {
		return PlacementResult{Valid: false, Reason: fmt.Sprintf("Insufficient resources (need %d/%d/%d Iron/Wood/Stone)",
			nodeDef.Cost.Iron, nodeDef.Cost.Wood, nodeDef.Cost.Stone)}
	}

	return PlacementResult{Valid: true, Reason: ""}
}
