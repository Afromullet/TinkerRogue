package main

import (
	"fmt"
	"game_main/common"
	"game_main/equipment"

	"github.com/bytearena/ecs"
)

/*
To Implement a new type of attack:

1) Create the component
2) Create the struct assocaited with the component
3) Implement the function that handles the attack. Curently the function removes the existing movement component
And then adds a new one, which determines how to approach the creature that's attacking
4) Call the function in CreatureAttackSystem

*/

var (
	approachAndAttack   *ecs.Component
	distanceRangeAttack *ecs.Component
)

type ApproachAndAttack struct {
}

type DistanceRangedAttack struct {
}

// Build a path to the player and attack once within range

func ApproachAndAttackAction(g *Game, c *ecs.QueryResult, target *ecs.Entity) {

	// Clear any existing movement if the creature attacks so there's no conflict
	RemoveMovementComponent(c)

	c.Entity.AddComponent(entityFollowComp, &EntityFollow{target: target})

	defenderPos := common.GetComponentType[*common.Position](target, common.PositionComponent)
	if common.DistanceBetween(c.Entity, target) == 1 {
		MeleeAttackSystem(g, common.GetPosition(c.Entity), defenderPos)

	}

}

// Stay within Ranged Attack Distance with the movement
func StayDistantRangedAttackAction(g *Game, c *ecs.QueryResult, target *ecs.Entity) {

	RemoveMovementComponent(c)

	RangedWeapon := common.GetComponentType[*RangedWeapon](c.Entity, equipment.RangedWeaponComponent)

	if RangedWeapon != nil {

		rangeToKeep := RangedWeapon.ShootingRange

		distance := common.GetPosition(c.Entity).ManhattanDistance(common.GetPosition(target))
		fmt.Println("Printing distance. Range On Attack", distance, RangedWeapon.ShootingRange)

		if common.GetPosition(c.Entity).InRange(common.GetPosition(target), RangedWeapon.ShootingRange) {
			RangedAttackSystem(g, common.GetPosition(c.Entity))
			fmt.Println("In range")
		} else {
			c.Entity.AddComponent(withinRadiusComp, &DistanceToEntityMovement{distance: rangeToKeep, target: target})

		}

	} else {
		fmt.Println("No ranged weapon")
	}

}

func SelectPlayerAsTarget(g *Game, c *ecs.QueryResult) {

}

// Gets called in the MonsterSystems loop
// Todo change logic to allow any entity to be targetted rather than just the player
func CreatureAttackSystem(c *ecs.QueryResult, g *Game) {

	var ok bool

	if _, ok = c.Entity.GetComponentData(approachAndAttack); ok {

		ApproachAndAttackAction(g, c, g.playerData.PlayerEntity)

	}

	// Todo need to avoid friendly fire
	if _, ok = c.Entity.GetComponentData(distanceRangeAttack); ok {
		StayDistantRangedAttackAction(g, c, g.playerData.PlayerEntity)
	}

}
