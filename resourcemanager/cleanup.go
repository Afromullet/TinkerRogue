package resmanager

import (
	"fmt"
	"game_main/common"
	"game_main/graphics"
	"game_main/monsters"
	"game_main/timesystem"
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

	fmt.Println("Starting length ", len(trackers.CreatureTracker.PosTracker))
	trackers.CreatureTracker.Remove(e)

	ind := graphics.CoordTransformer.IndexFromLogicalXY(pos.X, pos.Y)
	gm.Tiles[ind].Blocked = false

	timesystem.TurnManager.ActionDispatcher.RemoveActionQueueForEntity(e)
	world.DisposeEntity(e)
	monsters.NumMonstersOnMap--
	if monsters.NumMonstersOnMap == -1 {
		monsters.NumMonstersOnMap = 0
	}

	fmt.Println("Ending length ", len(trackers.CreatureTracker.PosTracker))

}

func RemoveDeadEntities(ecsmanager *common.EntityManager, gm *worldmap.GameMap) {
	for _, c := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {

		RemoveEntity(ecsmanager.World, gm, c.Entity)
	}
}
