package playernode

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CreatePlayerNode creates a new player-owned node at the given position.
// Returns the EntityID of the new node, or an error if nodeTypeID is unknown.
func CreatePlayerNode(manager *common.EntityManager, pos coords.LogicalPosition, nodeTypeID core.NodeTypeID, currentTick int64) (ecs.EntityID, error) {
	if !core.GetNodeRegistry().HasNode(string(nodeTypeID)) {
		return 0, fmt.Errorf("unknown node type ID: %q", nodeTypeID)
	}

	entity := manager.World.NewEntity()

	// Add position component
	entity.AddComponent(common.PositionComponent, &coords.LogicalPosition{
		X: pos.X,
		Y: pos.Y,
	})

	// Add player node data
	entity.AddComponent(core.PlayerNodeComponent, &core.PlayerNodeData{
		NodeID:     entity.GetID(),
		NodeTypeID: nodeTypeID,
		PlacedTick: currentTick,
	})

	// Add influence component using baseRadius from NodeRegistry
	nodeDef := core.GetNodeRegistry().GetNodeByID(string(nodeTypeID))
	baseRadius := core.GetDefaultPlayerNodeRadius()
	if nodeDef != nil {
		baseRadius = nodeDef.BaseRadius
	}
	entity.AddComponent(core.InfluenceComponent, &core.InfluenceData{
		Radius:        baseRadius,
		BaseMagnitude: core.GetDefaultPlayerNodeMagnitude(),
	})

	// Register in position system
	common.GlobalPositionSystem.AddEntity(entity.GetID(), pos)

	// Log event
	core.LogEvent(core.EventPlayerNodePlaced, currentTick, entity.GetID(),
		core.FormatEventString("Player node '%s' placed at (%d, %d)",
			string(nodeTypeID), pos.X, pos.Y), nil)

	return entity.GetID(), nil
}

// DestroyPlayerNode removes a player node from the overworld.
func DestroyPlayerNode(manager *common.EntityManager, entity *ecs.Entity) {
	pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)

	if pos != nil {
		manager.CleanDisposeEntity(entity, pos)
	} else {
		manager.World.DisposeEntities(entity)
	}
}
