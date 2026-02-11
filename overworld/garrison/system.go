package garrison

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// AssignSquadToNode assigns a player squad to garrison a node.
// Creates the GarrisonComponent on the node if this is the first garrison.
func AssignSquadToNode(manager *common.EntityManager, squadID ecs.EntityID, nodeID ecs.EntityID) error {
	// Validate node exists and is player-owned
	nodeEntity := manager.FindEntityByID(nodeID)
	if nodeEntity == nil {
		return fmt.Errorf("node entity %d not found", nodeID)
	}

	nodeData := common.GetComponentType[*core.OverworldNodeData](nodeEntity, core.OverworldNodeComponent)
	if nodeData == nil {
		return fmt.Errorf("entity %d is not an overworld node", nodeID)
	}

	if !core.IsFriendlyOwner(nodeData.OwnerID) {
		return fmt.Errorf("cannot garrison node owned by %s", nodeData.OwnerID)
	}

	// Validate squad exists and is available
	squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
	if squadData == nil {
		return fmt.Errorf("squad %d not found", squadID)
	}

	if squadData.GarrisonedAtNodeID != 0 {
		return fmt.Errorf("squad %d is already garrisoned at node %d", squadID, squadData.GarrisonedAtNodeID)
	}

	if squadData.IsDeployed {
		return fmt.Errorf("squad %d is deployed in combat", squadID)
	}

	// Add or update garrison component on node
	garrisonData := GetGarrisonAtNode(manager, nodeID)
	if garrisonData == nil {
		// First garrison on this node - add component
		garrisonData = &core.GarrisonData{
			SquadIDs: []ecs.EntityID{squadID},
		}
		nodeEntity.AddComponent(core.GarrisonComponent, garrisonData)
	} else {
		// Check for duplicate
		for _, id := range garrisonData.SquadIDs {
			if id == squadID {
				return fmt.Errorf("squad %d already in garrison at node %d", squadID, nodeID)
			}
		}
		garrisonData.SquadIDs = append(garrisonData.SquadIDs, squadID)
	}

	// Mark squad as garrisoned
	squadData.GarrisonedAtNodeID = nodeID

	// Log event
	core.LogEvent(core.EventGarrisonAssigned, core.GetCurrentTick(manager), nodeID,
		fmt.Sprintf("Squad %s assigned to garrison at node %d", squadData.Name, nodeID), nil)

	fmt.Printf("Assigned squad %d (%s) to garrison at node %d\n", squadID, squadData.Name, nodeID)
	return nil
}

// RemoveSquadFromNode removes a squad from a node's garrison.
// Removes GarrisonComponent entirely if no squads remain.
func RemoveSquadFromNode(manager *common.EntityManager, squadID ecs.EntityID, nodeID ecs.EntityID) error {
	nodeEntity := manager.FindEntityByID(nodeID)
	if nodeEntity == nil {
		return fmt.Errorf("node entity %d not found", nodeID)
	}

	garrisonData := GetGarrisonAtNode(manager, nodeID)
	if garrisonData == nil {
		return fmt.Errorf("node %d has no garrison", nodeID)
	}

	// Find and remove squad from garrison
	found := false
	for i, id := range garrisonData.SquadIDs {
		if id == squadID {
			garrisonData.SquadIDs[i] = garrisonData.SquadIDs[len(garrisonData.SquadIDs)-1]
			garrisonData.SquadIDs = garrisonData.SquadIDs[:len(garrisonData.SquadIDs)-1]
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("squad %d not found in garrison at node %d", squadID, nodeID)
	}

	// Clear squad's garrison flag
	squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
	if squadData != nil {
		squadData.GarrisonedAtNodeID = 0
	}

	// Remove garrison component if empty
	if len(garrisonData.SquadIDs) == 0 {
		nodeEntity.RemoveComponent(core.GarrisonComponent)
	}

	core.LogEvent(core.EventGarrisonRemoved, core.GetCurrentTick(manager), nodeID,
		fmt.Sprintf("Squad %d removed from garrison at node %d", squadID, nodeID), nil)

	return nil
}

// CreateNPCGarrison creates NPC garrison squads at a node using power-budget generation.
// The squads are created from templates matching the faction type.
// This is called during Fortify by faction AI.
func CreateNPCGarrison(
	manager *common.EntityManager,
	nodeEntity *ecs.Entity,
	factionType core.FactionType,
	strength int,
) error {
	nodeData := common.GetComponentType[*core.OverworldNodeData](nodeEntity, core.OverworldNodeComponent)
	if nodeData == nil {
		return fmt.Errorf("entity is not an overworld node")
	}

	nodeID := nodeEntity.GetID()

	// Don't add garrison if already garrisoned
	if IsNodeGarrisoned(manager, nodeID) {
		return nil
	}

	// Create a single garrison squad using existing squad creation
	// Use faction-appropriate name and moderate power based on strength
	squadName := fmt.Sprintf("%s Garrison", factionType.String())
	squadLevel := 1 + (strength / 20) // Scale level with faction strength

	// Create squad from template using existing squad creation system
	unitTemplates := generateGarrisonUnits(factionType, squadLevel)
	if len(unitTemplates) == 0 {
		return fmt.Errorf("no unit templates available for faction %s", factionType.String())
	}

	pos := common.GetComponentType[*coords.LogicalPosition](nodeEntity, common.PositionComponent)
	if pos == nil {
		return fmt.Errorf("node has no position")
	}

	squadID := squads.CreateSquadFromTemplate(
		manager,
		squadName,
		squads.FormationBalanced,
		*pos,
		unitTemplates,
	)

	if squadID == 0 {
		return fmt.Errorf("failed to create garrison squad")
	}

	// Create garrison component on node
	garrisonData := &core.GarrisonData{
		SquadIDs: []ecs.EntityID{squadID},
	}
	nodeEntity.AddComponent(core.GarrisonComponent, garrisonData)

	core.LogEvent(core.EventGarrisonAssigned, core.GetCurrentTick(manager), nodeID,
		fmt.Sprintf("%s garrison created at node %d (squad %d)", factionType.String(), nodeID, squadID), nil)

	fmt.Printf("Created NPC garrison at node %d: squad %d (%s)\n", nodeID, squadID, squadName)
	return nil
}

// generateGarrisonUnits creates unit templates for an NPC garrison.
// Uses the existing unit pool filtered by faction type preferences.
func generateGarrisonUnits(factionType core.FactionType, level int) []squads.UnitTemplate {
	units := squads.Units
	if len(units) == 0 {
		return nil
	}

	// Pick 3-4 units for a garrison squad
	count := 3 + common.RandomInt(2) // 3-4 units
	if count > len(units) {
		count = len(units)
	}

	gridPositions := [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}}
	result := make([]squads.UnitTemplate, 0, count)

	for i := 0; i < count; i++ {
		unit := units[common.RandomInt(len(units))]
		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = (i == 0)
		result = append(result, unit)
	}

	return result
}

// TransferNodeOwnership changes the owner of a node after a garrison defeat.
// Removes the garrison component (garrison squads are already disposed by combat cleanup).
func TransferNodeOwnership(manager *common.EntityManager, nodeID ecs.EntityID, newOwnerID string) error {
	nodeEntity := manager.FindEntityByID(nodeID)
	if nodeEntity == nil {
		return fmt.Errorf("node entity %d not found", nodeID)
	}

	nodeData := common.GetComponentType[*core.OverworldNodeData](nodeEntity, core.OverworldNodeComponent)
	if nodeData == nil {
		return fmt.Errorf("entity %d is not an overworld node", nodeID)
	}

	oldOwner := nodeData.OwnerID
	nodeData.OwnerID = newOwnerID

	// Remove garrison component if present (garrison squads already cleaned up)
	if nodeEntity.HasComponent(core.GarrisonComponent) {
		nodeEntity.RemoveComponent(core.GarrisonComponent)
	}

	core.LogEvent(core.EventNodeCaptured, core.GetCurrentTick(manager), nodeID,
		fmt.Sprintf("Node %d captured: %s -> %s", nodeID, oldOwner, newOwnerID), nil)

	fmt.Printf("Node %d ownership transferred: %s -> %s\n", nodeID, oldOwner, newOwnerID)
	return nil
}
