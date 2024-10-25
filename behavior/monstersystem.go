package behavior

import (
	"fmt"
	"game_main/avatar"
	"game_main/common"
	"game_main/gear"
	"game_main/monsters"
	"game_main/timesystem"
	"game_main/worldmap"
)

// Currently executes all actions just as it did before, this time only doing it through the AllActions queue
// Will change later once the time system is implemented. Still want things to behave the same while implementing the time system
func MonsterSystems(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, ts *timesystem.GameTurn) {

	monsters.NumMonstersOnMap = 0
	actionCost := 0
	//TODO do I need to make sure the same action can't be added twice?
	for _, c := range ecsmanger.World.Query(ecsmanger.WorldTags["monsters"]) {

		if c.Entity != nil {
			actionQueue := common.GetComponentType[*timesystem.ActionQueue](c.Entity, timesystem.ActionQueueComponent)
			attr := common.GetComponentType[*common.Attributes](c.Entity, common.AttributeComponent)

			gear.UpdateEntityAttributes(c.Entity)
			monsters.ApplyStatusEffects(c)
			//gear.ConsumableEffectApplier(c.Entity)

			if actionQueue != nil && attr.CanAct {

				act := CreatureMovementSystem(ecsmanger, gm, c)
				if act != nil {

					actionQueue.AddMonsterAction(act, attr.TotalMovementSpeed, timesystem.MovementKind)
				}

				act, actionCost = CreatureAttackSystem(ecsmanger, pl, gm, c)

				if act != nil {

					actionQueue.AddMonsterAction(act, actionCost, timesystem.AttackKind)
				}

			} else {

				actionQueue.TotalActionPoints = 0
				fmt.Println("Can't act")
			}
		}

		monsters.NumMonstersOnMap++

	}

}
