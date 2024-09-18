package input

import (
	"game_main/avatar"
	"game_main/combat"
	"game_main/common"
	"game_main/graphics"
	"game_main/timesystem"
	"game_main/worldmap"
)

// Need a way to to return the type of Action. Using enums to select which action to return.
// This will result in lots of case statements, but if it works, it works.
type PlayerAction int

const (
	PlayerSelRanged = iota
	PlayerPickupFromFloor
	PlayerMoveAction
)

// Also returns teh action
func GetSimplePlayerAction(act PlayerAction, pl *avatar.PlayerData, gm *worldmap.GameMap) (timesystem.ActionWrapper, int) {

	switch act {

	case PlayerSelRanged:
		return timesystem.NewSimplePlayerActions(PlayerSelectRangedTarget, pl, gm), 1
	case PlayerPickupFromFloor:
		return timesystem.NewSimplePlayerActions(PlayerPickupItem, pl, gm), 1
	default:
		return nil, 0
	}

}

// Although I could break this down further to separate between attacking and moving,
// The actions aren't complex enough to justify the modularity.
// Selects the action cost based on whether or nto to attack

func GetPlayerMoveAction(act PlayerAction, ecsmanager *common.EntityManager,
	pl *avatar.PlayerData,
	gm *worldmap.GameMap, xOffset, yOffset int) (timesystem.ActionWrapper, int) {

	attr := common.GetComponentType[*common.Attributes](pl.PlayerEntity, common.AttributeComponent)

	nextPosition := common.Position{
		X: pl.Pos.X + xOffset,
		Y: pl.Pos.Y + yOffset,
	}

	nextTile := gm.Tiles[graphics.IndexFromXY(nextPosition.X, nextPosition.Y)]

	actionCost := attr.TotalMovementSpeed
	if nextTile.Blocked {
		if combat.GetCreatureAtPosition(ecsmanager, &nextPosition) != nil {
			actionCost = attr.TotalAttackSpeed

		}

	}

	switch act {

	case PlayerMoveAction:

		return timesystem.NewPlayerMoveAction(MovePlayer, ecsmanager,
			pl, gm, xOffset, yOffset), actionCost

	default:
		return nil, 0
	}

}

func AddPlayerAction(simpleAction timesystem.ActionWrapper, pl *avatar.PlayerData, cost int, kindofAction timesystem.KindOfAction) {

	actionQueue := common.GetComponentType[*timesystem.ActionQueue](pl.PlayerEntity, timesystem.ActionQueueComponent)
	actionQueue.AddPlayerAction(simpleAction, cost, kindofAction)

}

// Perform the first action in the queue
func PerformPlayerAction(pl *avatar.PlayerData) {
	actionQueue := common.GetComponentType[*timesystem.ActionQueue](pl.PlayerEntity, timesystem.ActionQueueComponent)

	if len(actionQueue.AllActions) > 0 {
		//actionQueue.AllActions[0].Execute(actionQueue)
	}

}
