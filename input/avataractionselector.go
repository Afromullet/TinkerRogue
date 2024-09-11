package input

import (
	"game_main/actionmanager"
	"game_main/avatar"
	"game_main/common"
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
func GetSimplePlayerAction(act PlayerAction, pl *avatar.PlayerData, gm *worldmap.GameMap) (actionmanager.ActionWrapper, int) {

	switch act {

	case PlayerSelRanged:
		return actionmanager.NewSimplePlayerActions(PlayerSelectRangedTarget, pl, gm), 1
	case PlayerPickupFromFloor:
		return actionmanager.NewSimplePlayerActions(PlayerPickupItem, pl, gm), 1
	default:
		return nil, 0
	}

}

func GetPlayerMoveAction(act PlayerAction, ecsmanager *common.EntityManager,
	pl *avatar.PlayerData,
	gm *worldmap.GameMap, xOffset, yOffset int) (actionmanager.ActionWrapper, int) {

	attr := common.GetComponentType[*common.Attributes](pl.PlayerEntity, common.AttributeComponent)

	switch act {

	case PlayerMoveAction:
		return actionmanager.NewPlayerMoveAction(MovePlayer, ecsmanager,
			pl, gm, xOffset, yOffset), attr.TotalMovementSpeed

	default:
		return nil, 0
	}

}

func AddPlayerAction(simpleAction actionmanager.ActionWrapper, pl *avatar.PlayerData, cost int, kindofAction actionmanager.KindOfAction) {

	actionQueue := common.GetComponentType[*actionmanager.ActionQueue](pl.PlayerEntity, actionmanager.ActionQueueComponent)
	actionQueue.AddAction(simpleAction, cost, kindofAction)

}

// Perform the first action in the queue
func PerformPlayerAction(pl *avatar.PlayerData) {
	actionQueue := common.GetComponentType[*actionmanager.ActionQueue](pl.PlayerEntity, actionmanager.ActionQueueComponent)

	if len(actionQueue.AllActions) > 0 {
		//actionQueue.AllActions[0].Execute(actionQueue)
	}

}
