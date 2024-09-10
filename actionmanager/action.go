package actionmanager

import (
	"fmt"
	"game_main/avatar"
	"game_main/common"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

var ActionComponent *ecs.Component

type ActionFunc func()

type Actions struct {
	AllActions []ActionFunc
}

// All of the cases are with some specific funcitons in mind:
//
// case func(*common.EntityManager, *avatar.PlayerData, *worldmap.GameMap, *ecs.QueryResult, *ecs.Entity)
// --ApproachAndAttach, StayDistantRangedAttackAction

func (a *Actions) AddAttackAction(f interface{}, params ...interface{}) {
	actionWrapper := func() {
		switch fn := f.(type) {
		case func(*common.EntityManager, *avatar.PlayerData, *worldmap.GameMap, *ecs.QueryResult, *ecs.Entity):
			fmt.Println("Attack Action")
			fn(params[0].(*common.EntityManager),
				params[1].(*avatar.PlayerData),
				params[2].(*worldmap.GameMap),
				params[3].(*ecs.QueryResult),
				params[4].(*ecs.Entity))
		default:
			fmt.Println("Unsupported function signature")
		}
	}
	// Add the wrapped action (closure) to the actions slice
	a.AllActions = append(a.AllActions, actionWrapper)
}

// All of the cases are with some specific funcitons in mind:
//
// func(*common.EntityManager, *worldmap.GameMap, *ecs.Entity)
// --SimpleWanderAction,NoMoveAction,EntityFollowMoveAction
// --WithinRadiusMoveAction,WithinRangeMoveAction,FleeFromEntityMovementAction
func (a *Actions) AddMoveAction(f interface{}, params ...interface{}) {
	actionWrapper := func() {
		switch fn := f.(type) {
		case func(*common.EntityManager, *worldmap.GameMap, *ecs.Entity):
			fmt.Println("Movement Action")
			fn(params[0].(*common.EntityManager),
				params[1].(*worldmap.GameMap),
				params[2].(*ecs.Entity))

		default:
			fmt.Println("Unsupported function signature")
		}
	}
	// Add the wrapped action (closure) to the actions slice
	a.AllActions = append(a.AllActions, actionWrapper)
}
