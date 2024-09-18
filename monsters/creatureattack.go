package monsters

import (
	"fmt"

	"game_main/avatar"
	"game_main/combat"
	"game_main/common"
	"game_main/gear"
	"game_main/timesystem"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

/*
To Implement a new type of attack:

Actions are stored in a Queue. This requires some more steps after creating the component

1) Create the component
2) Create the struct associated with the component
3) Implement the function that handles the attack. Curently the function removes the existing movement component
And then adds a new one, which determines how to approach the creature that's attacking. The function needs to return an actionwrapper and the action cost
4) In the Action package, create a wrapper for the function. See the comments in the actionmanager on how to do that.
5) Return the wrapper in the CreatureAttackSystem

*/

var (
	ChargeAttackComp        *ecs.Component
	RangeAttackBehaviorComp *ecs.Component
)

type AttackBehavior struct {
}

// Used by Actions which perform a melee attack
func MeleeAttackHelper(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, target *ecs.Entity) {

	defenderPos := common.GetComponentType[*common.Position](target, common.PositionComponent)
	if common.DistanceBetween(c.Entity, target) == 1 {
		combat.MeleeAttackSystem(ecsmanger, pl, gm, common.GetPosition(c.Entity), defenderPos)

	}

}

// Used by Actions which perform a ranged attack.
func RangedAttackHelper(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, target *ecs.Entity) {

	RangedWeapon := common.GetComponentType[*gear.RangedWeapon](c.Entity, gear.RangedWeaponComponent)

	if RangedWeapon != nil {

		if common.GetPosition(c.Entity).InRange(common.GetPosition(target), RangedWeapon.ShootingRange) {
			combat.RangedAttackSystem(ecsmanger, pl, gm, common.GetPosition(c.Entity))

		}
	} else {
		fmt.Println("No ranged weapon")
	}

}

// This action keeps the entity at the weapons range and attacks once within that range.
// Does nothing if the target is more than 10 tiles away
func RangedAttackFromDistance(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, t *ecs.Entity) (timesystem.ActionWrapper, int) {

	wep := common.GetComponentType[*gear.RangedWeapon](c.Entity, gear.RangedWeaponComponent)
	cost := 0
	if wep == nil {
		return nil, cost
	}

	queue := common.GetComponentType[*timesystem.ActionQueue](c.Entity, timesystem.ActionQueueComponent)
	queue.ResetQueue() //Not resetting would result in the creature prioritizing attack every time.
	attr := common.GetComponentType[*common.Attributes](c.Entity, common.AttributeComponent)

	if common.DistanceBetween(c.Entity, t) < wep.ShootingRange {

		return timesystem.NewOneTargetAttack(RangedAttackHelper, ecsmanger, pl, gm, c, pl.PlayerEntity), attr.TotalAttackSpeed

	} else if common.DistanceBetween(c.Entity, t) > 10 {

		return timesystem.NewEntityMover(NoMoveAction, ecsmanger, gm, c.Entity), attr.TotalMovementSpeed
	} else {

		c.Entity.AddComponent(WithinRangeComponent, &DistanceToEntityMovement{Target: t, Distance: wep.ShootingRange})

		return timesystem.NewEntityMover(WithinRangeMoveAction, ecsmanger, gm, c.Entity), attr.TotalMovementSpeed
	}

}

// Action with which the entity charges at another entity and performs a melee attack
// Does nothing if the target is more than 10 tiles away
func ChargeAndAttack(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, t *ecs.Entity) (timesystem.ActionWrapper, int) {

	wep := common.GetComponentType[*gear.MeleeWeapon](c.Entity, gear.MeleeWeaponComponent)
	cost := 0

	if wep == nil {
		return nil, cost
	}

	queue := common.GetComponentType[*timesystem.ActionQueue](c.Entity, timesystem.ActionQueueComponent)
	queue.ResetQueue() //Not resetting would result in the creature prioritizing attack every time.
	attr := common.GetComponentType[*common.Attributes](c.Entity, common.AttributeComponent)

	if common.DistanceBetween(c.Entity, t) == 1 {

		return timesystem.NewOneTargetAttack(MeleeAttackHelper, ecsmanger, pl, gm, c, pl.PlayerEntity), attr.TotalAttackSpeed

	} else if common.DistanceBetween(c.Entity, t) > 10 {

		return timesystem.NewEntityMover(NoMoveAction, ecsmanger, gm, c.Entity), attr.TotalMovementSpeed

	} else {

		c.Entity.AddComponent(EntityFollowComp, &EntityFollow{Target: t})

		return timesystem.NewEntityMover(EntityFollowMoveAction, ecsmanger, gm, c.Entity), attr.TotalMovementSpeed
	}

}

// Gets called in the MonsterSystems loop
// Todo change logic to allow any entity to be targetted rather than just the player
func CreatureAttackSystem(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult) (timesystem.ActionWrapper, int) {

	var ok bool

	if _, ok = c.Entity.GetComponentData(RangeAttackBehaviorComp); ok {
		return RangedAttackFromDistance(ecsmanger, pl, gm, c, pl.PlayerEntity)

	}

	if _, ok = c.Entity.GetComponentData(ChargeAttackComp); ok {

		return ChargeAndAttack(ecsmanger, pl, gm, c, pl.PlayerEntity)
	}

	return nil, 0

}
