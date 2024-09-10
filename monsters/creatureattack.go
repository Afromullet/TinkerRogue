package monsters

import (
	"fmt"
	"game_main/actionmanager"
	"game_main/avatar"
	"game_main/combat"
	"game_main/common"
	"game_main/equipment"
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
	ApproachAndAttackComp   *ecs.Component
	DistanceRangeAttackComp *ecs.Component
)

type ApproachAndAttack struct {
}

type DistanceRangedAttack struct {
}

// Build a path to the player and attack once within range

func ApproachAndAttackAction(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, target *ecs.Entity) {

	// Clear any existing movement if the creature attacks so there's no conflict
	RemoveMovementComponent(c)

	c.Entity.AddComponent(EntityFollowComp, &EntityFollow{Target: target})

	defenderPos := common.GetComponentType[*common.Position](target, common.PositionComponent)
	if common.DistanceBetween(c.Entity, target) == 1 {
		combat.MeleeAttackSystem(ecsmanger, pl, gm, common.GetPosition(c.Entity), defenderPos)

	}

}

// Stay within Ranged Attack Distance with the movement
func StayDistantRangedAttackAction(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult, target *ecs.Entity) {

	RemoveMovementComponent(c)

	RangedWeapon := common.GetComponentType[*equipment.RangedWeapon](c.Entity, equipment.RangedWeaponComponent)

	if RangedWeapon != nil {

		rangeToKeep := RangedWeapon.ShootingRange

		distance := common.GetPosition(c.Entity).ManhattanDistance(common.GetPosition(target))
		fmt.Println("Printing distance. Range On Attack", distance, RangedWeapon.ShootingRange)

		if common.GetPosition(c.Entity).InRange(common.GetPosition(target), RangedWeapon.ShootingRange) {
			combat.RangedAttackSystem(ecsmanger, pl, gm, common.GetPosition(c.Entity))
			fmt.Println("In range")
		} else {
			c.Entity.AddComponent(WithinRadiusComp, &DistanceToEntityMovement{Distance: rangeToKeep, Target: target})

		}

	} else {
		fmt.Println("No ranged weapon")
	}

}

// Gets called in the MonsterSystems loop
// Todo change logic to allow any entity to be targetted rather than just the player
func CreatureAttackSystem(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, c *ecs.QueryResult) actionmanager.Action {

	var ok bool

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

	// Todo need to avoid friendly fire
	if _, ok = c.Entity.GetComponentData(DistanceRangeAttackComp); ok {

		return &actionmanager.OneTargetAttack{
			Func:   StayDistantRangedAttackAction,
			Param1: ecsmanger,
			Param2: pl,
			Param3: gm,
			Param4: c,
			Param5: pl.PlayerEntity,
		}
	}

	return nil

}
