package main

import (
	"fmt"

	"github.com/bytearena/ecs"
)

var simpleWander *ecs.Component
var noMove *ecs.Component
var goToEntity *ecs.Component

type SimpleWander struct {
}

type NoMovement struct {
}

type GoToEntityMovement struct {
	entity *ecs.Entity
}

// Each Movement function implementation choses how to build a path.
// They also handle walking on the path by calling reature.UpdatePosition
// Movement functions get called by MovementSystem which determines what movement component a creature has.

// Select a random spot to wander to and builds a new path when arriving at the position
func SimpleWanderAction(g *Game, e *ecs.Entity) {

	creature := GetComponentType[*Creature](e, CreatureComponent)

	creaturePosition := GetPosition(e)

	randomPos := GetRandomBetween(0, len(validPositions.positions))
	endPos := validPositions.Get(randomPos)

	//Only create a new path if one doesn't exist yet.
	if len(creature.Path) == 0 {

		astar := AStar{}
		creature.Path = astar.GetPath(g.gameMap, creaturePosition, endPos, false)

	}

	creature.UpdatePosition(g, creaturePosition)

}

func NoMoveAction(g *Game, e *ecs.Entity) {

}

// todo Not building the path correctly
func GoToEntityMoveAction(g *Game, e *ecs.Entity) {

	creature := GetComponentType[*Creature](e, CreatureComponent)
	creaturePosition := GetPosition(e)
	goToEnt := GetComponentType[*GoToEntityMovement](e, goToEntity)

	if goToEnt.entity != nil {
		targetPos := GetComponentType[*Position](goToEnt.entity, PositionComponent)

		creature.Path = creaturePosition.BuildPath(g, targetPos)
		fmt.Println("Printing path ", creature.Path)
	}

	creature.UpdatePosition(g, creaturePosition)

}

// Gets called in MonsterSystems, which queries the ECS manager and returns query results containing all monsters

// Creature movement follows a path, which is a slice of Position Type. Each movement function calls
// UpdatePosition, which...updates the creatures position The movement type functions determine
// How a path is created
func MovementSystem(c *ecs.QueryResult, g *Game) {

	var ok bool

	if _, ok = c.Entity.GetComponentData(simpleWander); ok {
		SimpleWanderAction(g, c.Entity)
	}

	if _, ok = c.Entity.GetComponentData(noMove); ok {
		NoMoveAction(g, c.Entity)
	}

	if _, ok = c.Entity.GetComponentData(goToEntity); ok {
		GoToEntityMoveAction(g, c.Entity)
	}

}

func RemoveMovementComponent(c *ecs.QueryResult) {

	var ok bool

	if _, ok = c.Entity.GetComponentData(simpleWander); ok {
		c.Entity.RemoveComponent(simpleWander)

	}

	if _, ok = c.Entity.GetComponentData(noMove); ok {
		c.Entity.RemoveComponent(noMove)
	}

	if _, ok = c.Entity.GetComponentData(goToEntity); ok {
		c.Entity.RemoveComponent(goToEntity)
	}

}
