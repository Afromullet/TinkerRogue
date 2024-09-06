package main

import (
	"github.com/bytearena/ecs"
)

// TODO, when a creature is attacking, remove the Movement component so the movement and attacking don't conflict

var approachAndAttack *ecs.Component
var distanceRangeAttack *ecs.Component

type ApproachAndAttack struct {
}

type DistanceRangedAttack struct {
	distance int
}

// Build a path to the player and attack once within range

func ApproachAndAttackAction(g *Game, c *ecs.QueryResult, target *ecs.Entity) {

	// Clear any existing movement if the creature attacks so there's no conflict
	RemoveMovementComponent(c)

	c.Entity.AddComponent(entityFollowComponent, &EntityFollow{target: target})

	defenderPos := GetComponentType[*Position](target, PositionComponent)
	if DistanceBetween(c.Entity, target) == 1 {
		MeleeAttackSystem(g, GetPosition(c.Entity), defenderPos)

	}

}

func StayDistantRangedAttackAction(g *Game, c *ecs.QueryResult, target *ecs.Entity) {

	RemoveMovementComponent(c)

	attackAction := GetComponentType[*DistanceRangedAttack](c.Entity, distanceRangeAttack)
	c.Entity.AddComponent(stayWithinRangeComponent, &StayWithinRange{distance: attackAction.distance, target: target})

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
