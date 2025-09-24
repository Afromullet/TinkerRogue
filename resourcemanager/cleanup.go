package resmanager

import (
	"game_main/common"
	"game_main/coords"
	"game_main/monsters"
	"game_main/trackers"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

func RemoveEntity(world *ecs.Manager, gm *worldmap.GameMap, e *ecs.Entity) {

	attr := common.GetComponentType[*common.Attributes](e, common.AttributeComponent)

	if attr.CurrentHealth > 0 {
		return
	}

	pos := common.GetPosition(e) //Todo replace pos with position from pos tracker

	trackers.CreatureTracker.Remove(e)

	logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
	ind := coords.CoordManager.LogicalToIndex(logicalPos)
	gm.Tiles[ind].Blocked = false

	world.DisposeEntity(e)
	monsters.NumMonstersOnMap--
	if monsters.NumMonstersOnMap == -1 {
		monsters.NumMonstersOnMap = 0
	}

}

func RemoveDeadEntities(ecsmanager *common.EntityManager, gm *worldmap.GameMap) {
	for _, c := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {

		RemoveEntity(ecsmanager.World, gm, c.Entity)
	}
}
