package main

import (
	"fmt"

	"github.com/bytearena/ecs"
)

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

	defenderPos := GetComponentType[*Position](target, PositionComponent)
	if DistanceBetween(c.Entity, target) == 1 {
		MeleeAttackSystem(g, GetPosition(c.Entity), defenderPos)

	}

}

// Stay within Ranged Attack Distance with the movement
func StayDistantRangedAttackAction(g *Game, c *ecs.QueryResult, target *ecs.Entity) {

	RemoveMovementComponent(c)

	RangedWeapon := GetComponentType[*RangedWeapon](c.Entity, RangedWeaponComponent)

	if RangedWeapon != nil {

		rangeToKeep := RangedWeapon.ShootingRange

		distance := GetPosition(c.Entity).ManhattanDistance(GetPosition(target))
		fmt.Println("Printing distance. Range On Attack", distance, RangedWeapon.ShootingRange)

		if GetPosition(c.Entity).InRange(GetPosition(target), RangedWeapon.ShootingRange) {
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
