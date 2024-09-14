package monsters

import (
	"fmt"

	"game_main/avatar"
	"game_main/combat"
	"game_main/common"
	"game_main/equipment"
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
And then adds a new one, which determines how to approach the creature that's attacking
4) In the Action package, create a wrapper for the function. See the comments in the actionmanager on how to do that.
5) Return the wrapper in the CreatureAttackSystem

*/

var (
	ChargeAttackComp        *ecs.Component
	RangeAttackBehaviorComp *ecs.Component
)

type AttackBehavior struct {
}

// Build a path to the player and attack once within range

func MeleeAttackAction(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, target *ecs.Entity) {

	defenderPos := common.GetComponentType[*common.Position](target, common.PositionComponent)
	if common.DistanceBetween(c.Entity, target) == 1 {
		combat.MeleeAttackSystem(ecsmanger, pl, gm, common.GetPosition(c.Entity), defenderPos)

	}

}

// Stay within Ranged Attack Distance with the movement
func RangedAttackAction(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, target *ecs.Entity) {

	RangedWeapon := common.GetComponentType[*equipment.RangedWeapon](c.Entity, equipment.RangedWeaponComponent)

	if RangedWeapon != nil {

		if common.GetPosition(c.Entity).InRange(common.GetPosition(target), RangedWeapon.ShootingRange) {
			combat.RangedAttackSystem(ecsmanger, pl, gm, common.GetPosition(c.Entity))

		}
	} else {
		fmt.Println("No ranged weapon")
	}

}

func RangedAttackFromDistance(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, t *ecs.Entity) timesystem.ActionWrapper {

	wep := common.GetComponentType[*equipment.RangedWeapon](c.Entity, equipment.RangedWeaponComponent)

	if wep == nil {
		return nil
	}

	if common.DistanceBetween(c.Entity, t) < wep.ShootingRange {

		return timesystem.NewOneTargetAttack(RangedAttackAction, ecsmanger, pl, gm, c, pl.PlayerEntity)

	} else {
		
		c.Entity.AddComponent(WithinRangeComponent, &DistanceToEntityMovement{Target: t, Distance: wep.ShootingRange})

		return timesystem.NewEntityMover(WithinRangeMoveAction, ecsmanger, gm, c.Entity)
	}

}

func ChargeAndAttack(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, t *ecs.Entity) timesystem.ActionWrapper {

	wep := common.GetComponentType[*equipment.MeleeWeapon](c.Entity, equipment.MeleeWeaponComponent)

	if wep == nil {
		return nil
	}

	if common.DistanceBetween(c.Entity, t) == 1 {

		return timesystem.NewOneTargetAttack(MeleeAttackAction, ecsmanger, pl, gm, c, pl.PlayerEntity)

	} else {

		c.Entity.AddComponent(EntityFollowComp, &EntityFollow{Target: t})

		return timesystem.NewEntityMover(EntityFollowMoveAction, ecsmanger, gm, c.Entity)
	}

}

// Gets called in the MonsterSystems loop
// Todo change logic to allow any entity to be targetted rather than just the player
func CreatureAttackSystem(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult) timesystem.ActionWrapper {

	var ok bool

	if _, ok = c.Entity.GetComponentData(RangeAttackBehaviorComp); ok {
		return RangedAttackFromDistance(ecsmanger, pl, gm, c, pl.PlayerEntity)

	}

	if _, ok = c.Entity.GetComponentData(ChargeAttackComp); ok {

		return ChargeAndAttack(ecsmanger, pl, gm, c, pl.PlayerEntity)
	}

	return nil

}
