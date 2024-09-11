package actionmanager

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/worldmap"
)

// Wrapper around player actions which require the PlayerData and Map
type SimplePlayerActions struct {
	Func   func(pl *avatar.PlayerData, gm *worldmap.GameMap)
	Param1 *avatar.PlayerData
	Param2 *worldmap.GameMap
}

// NewEntityMover creates a new EntityMover with the provided parameters.
func NewSimplePlayerActions(fn func(*avatar.PlayerData, *worldmap.GameMap),
	param1 *avatar.PlayerData,
	param2 *worldmap.GameMap) *SimplePlayerActions {
	return &SimplePlayerActions{
		Func:   fn,
		Param1: param1,
		Param2: param2,
	}
}

func (a *SimplePlayerActions) Execute(q *ActionQueue) {
	a.Func(a.Param1, a.Param2)
	q.pop()
}

// Wrapper around the action that lets the player move
type MovePlayerAction struct {
	Func   func(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, xOffset, yOffset int)
	Param1 *common.EntityManager
	Param2 *avatar.PlayerData
	Param3 *worldmap.GameMap
	Param4 int
	Param5 int
}

// NewEntityMover creates a new EntityMover with the provided parameters.
func NewPlayerMoveAction(fn func(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, xOffset, yOffset int),
	param1 *common.EntityManager,
	param2 *avatar.PlayerData,
	param3 *worldmap.GameMap,
	param4 int,
	param5 int) *MovePlayerAction {
	return &MovePlayerAction{
		Func:   fn,
		Param1: param1,
		Param2: param2,
		Param3: param3,
		Param4: param4,
		Param5: param5,
	}
}

func (a *MovePlayerAction) Execute(q *ActionQueue) {
	a.Func(a.Param1, a.Param2, a.Param3, a.Param4, a.Param5)
	q.pop()
}
