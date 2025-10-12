package resmanager

import (
	"game_main/common"
	"game_main/coords"

	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

func RemoveEntity(world *ecs.Manager, gm *worldmap.GameMap, e *ecs.Entity) {

	attr := common.GetComponentType[*common.Attributes](e, common.AttributeComponent)

	if attr.CurrentHealth > 0 {
		return
	}

	pos := common.GetPosition(e)

	logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
	ind := coords.CoordManager.LogicalToIndex(logicalPos)
	gm.Tiles[ind].Blocked = false

	// Remove from PositionSystem
	if common.GlobalPositionSystem != nil {
		common.GlobalPositionSystem.RemoveEntity(e.GetID(), logicalPos)
	}

	world.DisposeEntity(e)

}

func RemoveDeadEntities(ecsmanager *common.EntityManager, gm *worldmap.GameMap) {

	/*
		for _, c := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {

			RemoveEntity(ecsmanager.World, gm, c.Entity)
		}
	*/
}
