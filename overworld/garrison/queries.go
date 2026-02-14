package garrison

import (
	"game_main/common"
	"game_main/overworld/core"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// GetGarrisonAtNode returns garrison data for a node, or nil if not garrisoned.
func GetGarrisonAtNode(manager *common.EntityManager, nodeID ecs.EntityID) *core.GarrisonData {
	return common.GetComponentTypeByID[*core.GarrisonData](manager, nodeID, core.GarrisonComponent)
}

// IsNodeGarrisoned returns true if the node has a garrison with at least one squad.
func IsNodeGarrisoned(manager *common.EntityManager, nodeID ecs.EntityID) bool {
	data := GetGarrisonAtNode(manager, nodeID)
	return data != nil && len(data.SquadIDs) > 0
}

// GetAvailableSquadsForGarrison returns player squads that are not garrisoned and not deployed in combat.
func GetAvailableSquadsForGarrison(manager *common.EntityManager, playerEntityID ecs.EntityID) []ecs.EntityID {
	roster := squads.GetPlayerSquadRoster(playerEntityID, manager)
	if roster == nil {
		return nil
	}

	available := make([]ecs.EntityID, 0)
	for _, squadID := range roster.OwnedSquads {
		squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
		if squadData == nil {
			continue
		}
		// Available if not garrisoned and not deployed in combat
		if squadData.GarrisonedAtNodeID == 0 && !squadData.IsDeployed {
			available = append(available, squadID)
		}
	}
	return available
}

// FindPlayerNodesNearFaction finds player-owned nodes adjacent to any tile in the given territory.
func FindPlayerNodesNearFaction(manager *common.EntityManager, factionTerritoryTiles []coords.LogicalPosition) []ecs.EntityID {
	// Build set of faction tiles for quick lookup
	factionTileSet := make(map[coords.LogicalPosition]bool, len(factionTerritoryTiles))
	for _, tile := range factionTerritoryTiles {
		factionTileSet[tile] = true
	}

	found := make([]ecs.EntityID, 0)
	seen := make(map[ecs.EntityID]bool)

	for _, result := range core.OverworldNodeView.Get() {
		entity := result.Entity
		nodeData := common.GetComponentType[*core.OverworldNodeData](entity, core.OverworldNodeComponent)
		if nodeData == nil || !core.IsFriendlyOwner(nodeData.OwnerID) {
			continue
		}

		pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		if pos == nil {
			continue
		}

		nodeID := entity.GetID()
		if seen[nodeID] {
			continue
		}

		// Check if any cardinal neighbor is a faction tile
		for _, adj := range core.GetCardinalNeighbors(*pos) {
			if factionTileSet[adj] {
				found = append(found, nodeID)
				seen[nodeID] = true
				break
			}
		}
	}

	return found
}
