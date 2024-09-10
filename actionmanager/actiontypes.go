package actionmanager

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

/*
The timesystem requires actions to be stored so that they can be called in different orders.



Different actions have different parameters.

There will be a wrapper struct for each set of parameters that implements the Action interface

*/

/*
Here's an example for how it's done for attacking by adding a struct for attacking one target. We have:

ApproachAndAttackAction(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, target *ecs.Entity)
StayDistantRangedAttackAction(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, target *ecs.Entity)

The OneTargetAttack struct contains the function signature and one field for each parameter and implements execute.

type OneTargetAttack struct {
	fn      func(*common.EntityManager, *avatar.PlayerData, *worldmap.GameMap, *ecs.QueryResult, *ecs.Entity)
	params1 *common.EntityManager
	params2 *avatar.PlayerData
	params3 *worldmap.GameMap
	params4 *ecs.QueryResult
	params5 *ecs.Entity
}


func (a *OneTargetAttack) Execute() {
	a.fn(a.params1, a.params2, a.params3, a.params4, a.params5)
}


The system which calls these functions has to return actionmanager.Action, so we get this:

func CreatureAttackSystem(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult) actionmanager.Action

The CreatureAttackSystem returns the appropriate wrapper. I.Econst


	if _, ok = c.Entity.GetComponentData(ApproachAndAttackComp); ok {

		return &actionmanager.OneTargetAttack{
			Func:   ApproachAndAttackAction,
			Param1: ecsmanger,
			Param2: pl,
			Param3: gm,
			Param4: c,
			Param5: pl.PlayerEntity,
		}

	}


*/

var ActionComponent *ecs.Component

type Action interface {
	Execute()
}

// Wrapper around functions which attack an entity. The *ecs.Entity is the target
type OneTargetAttack struct {
	Func   func(*common.EntityManager, *avatar.PlayerData, *worldmap.GameMap, *ecs.QueryResult, *ecs.Entity)
	Param1 *common.EntityManager
	Param2 *avatar.PlayerData
	Param3 *worldmap.GameMap
	Param4 *ecs.QueryResult
	Param5 *ecs.Entity
}

func (a *OneTargetAttack) Execute() {
	a.Func(a.Param1, a.Param2, a.Param3, a.Param4, a.Param5)
}

// Wrapper around functions which move the *ecs.entity
type EntityMover struct {
	Func   func(*common.EntityManager, *worldmap.GameMap, *ecs.Entity)
	Param1 *common.EntityManager
	Param2 *worldmap.GameMap
	Param3 *ecs.Entity
}

// NewEntityMover creates a new EntityMover with the provided parameters.
func NewEntityMover(fn func(*common.EntityManager, *worldmap.GameMap, *ecs.Entity),
	param1 *common.EntityManager,
	param2 *worldmap.GameMap,
	param3 *ecs.Entity) *EntityMover {
	return &EntityMover{
		Func:   fn,
		Param1: param1,
		Param2: param2,
		Param3: param3,
	}
}

func (a *EntityMover) Execute() {
	a.Func(a.Param1, a.Param2, a.Param3)
}

type ActionQueue struct {
	AllActions []Action
}

func (a *ActionQueue) AddAction(action Action) {

	if action != nil {
		a.AllActions = append(a.AllActions, action)
	}
}
