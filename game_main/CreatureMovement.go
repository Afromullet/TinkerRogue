package main

import (
	"github.com/bytearena/ecs"
)

// Select a random spot to wander to and builds a new path when arriving at the position
func SimpleWanderAction(g *Game, e *ecs.Entity) {

	creature := ComponentType[*Creature](e, creature)

	creaturePosition := ComponentType[*Position](e, position)

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

func GoToPlayerMoveAction(g *Game, e *ecs.Entity) {

	creature := ComponentType[*Creature](e, creature)
	creaturePosition := ComponentType[*Position](e, position)
	creature.Path = creaturePosition.BuildPath(g, g.playerData.position)
	creature.UpdatePosition(g, creaturePosition)

}

// Gets called in the Game Loop.
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

	if _, ok = c.Entity.GetComponentData(goToPlayer); ok {
		GoToPlayerMoveAction(g, c.Entity)
	}

}
