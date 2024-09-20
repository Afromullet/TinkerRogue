package timesystem

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

// Wrapper around functions which attack an entity. The *ecs.Entity is the target
type OneTargetAttack struct {
	Func   func(*common.EntityManager, *avatar.PlayerData, *worldmap.GameMap, *ecs.QueryResult, *ecs.Entity)
	Param1 *common.EntityManager
	Param2 *avatar.PlayerData
	Param3 *worldmap.GameMap
	Param4 *ecs.QueryResult
	Param5 *ecs.Entity
}

func NewOneTargetAttack(
	fn func(*common.EntityManager, *avatar.PlayerData, *worldmap.GameMap, *ecs.QueryResult, *ecs.Entity),
	param1 *common.EntityManager,
	param2 *avatar.PlayerData,
	param3 *worldmap.GameMap,
	param4 *ecs.QueryResult,
	param5 *ecs.Entity) *OneTargetAttack {

	return &OneTargetAttack{
		Func:   fn,
		Param1: param1,
		Param2: param2,
		Param3: param3,
		Param4: param4,
		Param5: param5}

}

// Removed the q.pop since we're not inserting things into the ActionQueue back in priority order.
func (a *OneTargetAttack) Execute(q *ActionQueue) {
	a.Func(a.Param1, a.Param2, a.Param3, a.Param4, a.Param5)
	//q.pop()
}

// Wrapper around functions which move the *ecs.entity
type EntityMover struct {
	Func   func(*common.EntityManager, *worldmap.GameMap, *ecs.Entity)
	Param1 *common.EntityManager
	Param2 *worldmap.GameMap
	Param3 *ecs.Entity
}

// NewEntityMover creates a new EntityMover with the provided parameters.
func NewEntityMover(
	fn func(*common.EntityManager, *worldmap.GameMap, *ecs.Entity),
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

// Removed the q.pop since we're not inserting things into the ActionQueue back in priority order.
func (a *EntityMover) Execute(q *ActionQueue) {
	a.Func(a.Param1, a.Param2, a.Param3)
	//q.pop()
}
