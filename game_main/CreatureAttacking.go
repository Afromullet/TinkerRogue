package main

import (
	"github.com/bytearena/ecs"
)

// TODO, when a creature is attacking, remove the Movement component so the movement and attacking don't conflict

var approachAndAttack *ecs.Component
var creatureRangedAttack *ecs.Component

type ApproachAndAttack struct {
}

type CreatureRangedAttack struct {
}

// Build a path to the player
// Then attack
func ApproachAndAttackAction(g *Game, c *ecs.QueryResult, target *ecs.Entity) {

	c.Entity.AddComponent(goToEntity, &GoToEntityMovement{})

}

func CreateRangedAttackAction(g *Game, c *ecs.QueryResult, target *ecs.Entity) {

}

func SelectPlayerAsTarget(g *Game, c *ecs.QueryResult) {

}

// Gets called in the MonsterSystems loop
func CreatureAttackSystem(c *ecs.QueryResult, g *Game) {

	var ok bool

	// Clear any existing movement if the creature attacks so there's no conflict
	if _, ok = c.Entity.GetComponentData(approachAndAttack); ok {
		RemoveMovementComponent(c)
		c.Entity.AddComponent(goToEntity, &GoToEntityMovement{entity: g.playerData.PlayerEntity})
		attackerPos := GetPosition(c.Entity)

		if DistanceBetween(c.Entity, g.playerData.PlayerEntity) == 1 {
			MeleeAttackSystem(g, attackerPos, g.playerData.position)

		}

	}

	if _, ok = c.Entity.GetComponentData(creatureRangedAttack); ok {
		//CreateRangedAttackAction(g, c.Entity)
	}

}
