package node

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CreateNodeParams contains all parameters needed to create a unified overworld node.
type CreateNodeParams struct {
	Position         coords.LogicalPosition
	NodeTypeID       string
	OwnerID          string
	InitialIntensity int
	EncounterID      string
	CurrentTick      int64
}

// CreateNode creates a unified overworld node entity with position, node data, and influence.
// Returns the EntityID of the new node, or an error if the node type is unknown.
func CreateNode(manager *common.EntityManager, params CreateNodeParams) (ecs.EntityID, error) {
	nodeDef := core.GetNodeRegistry().GetNodeByID(params.NodeTypeID)
	if nodeDef == nil {
		return 0, fmt.Errorf("unknown node type ID: %q", params.NodeTypeID)
	}

	entity := manager.World.NewEntity()

	// Add position component
	entity.AddComponent(common.PositionComponent, &coords.LogicalPosition{
		X: params.Position.X,
		Y: params.Position.Y,
	})

	// Determine growth fields from node definition (zero for non-threats)
	growthRate := 0.0
	if nodeDef.Category == core.NodeCategoryThreat {
		growthRate = nodeDef.BaseGrowthRate
	}

	// Add unified node data
	entity.AddComponent(core.OverworldNodeComponent, &core.OverworldNodeData{
		NodeID:         entity.GetID(),
		NodeTypeID:     params.NodeTypeID,
		Category:       nodeDef.Category,
		OwnerID:        params.OwnerID,
		EncounterID:    params.EncounterID,
		Intensity:      params.InitialIntensity,
		GrowthProgress: 0.0,
		GrowthRate:     growthRate,
		IsContained:    false,
		CreatedTick:    params.CurrentTick,
	})

	// Add influence component
	radius := nodeDef.BaseRadius
	magnitude := core.GetDefaultPlayerNodeMagnitude()
	if nodeDef.Category == core.NodeCategoryThreat {
		radius += params.InitialIntensity
		magnitude = core.CalculateBaseMagnitude(params.InitialIntensity)
	}

	entity.AddComponent(core.InfluenceComponent, &core.InfluenceData{
		Radius:        radius,
		BaseMagnitude: magnitude,
	})

	// Register in position system
	common.GlobalPositionSystem.AddEntity(entity.GetID(), params.Position)

	return entity.GetID(), nil
}

// DestroyNode removes a unified overworld node from the world.
func DestroyNode(manager *common.EntityManager, entity *ecs.Entity) {
	pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)

	if pos != nil {
		manager.CleanDisposeEntity(entity, pos)
	} else {
		manager.World.DisposeEntities(entity)
	}
}
