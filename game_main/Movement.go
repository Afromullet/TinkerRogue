package main

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// Using the MoveType enum so that we only have to query creatures once for the movement types
type MoveType int

const (
	SimpleWanderType = iota
	NoMovementType
	GoToPlayerMoveType
	InvalidMovementType
)

func GetMovementComponentType(e *ecs.Entity) MoveType {

	var ok bool

	if _, ok = e.GetComponentData(simpleWander); ok {
		return SimpleWanderType
	}

	if _, ok = e.GetComponentData(noMove); ok {
		return NoMovementType
	}

	if _, ok = e.GetComponentData(goToPlayer); ok {
		return GoToPlayerMoveType
	}

	return InvalidMovementType

}

// Select a random spot to wander to.
// Build a new path when arriving at the location
func SimpleWanderAction(g *Game, e *ecs.Entity) {

	creature := GetComponentStruct[*Creature](e, creature)

	creaturePosition := GetComponentStruct[*Position](e, position)

	randomPos := GetRandomBetween(0, len(validPositions.positions))
	endPos := validPositions.Get(randomPos)

	//Only create a new path if one doesn't exist yet.
	if len(creature.path) == 0 {

		astar := AStar{}
		creature.path = astar.GetPath(g.gameMap, creaturePosition, endPos)

	}

	creature.MoveToNextPosition(g, creaturePosition)

}

func NoMoveAction(g *Game, e *ecs.Entity) {

}

func GoToPlayerMoveAction(g *Game, e *ecs.Entity) {

	creature := GetComponentStruct[*Creature](e, creature)
	creaturePosition := GetComponentStruct[*Position](e, position)

	creature.BuildPathToPlayer(g, creaturePosition)

	creature.MoveToNextPosition(g, creaturePosition)

}

func MovementSystem(g *Game) {

	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		movType := GetMovementComponentType(c.Entity)

		switch movType {
		case SimpleWanderType:
			SimpleWanderAction(g, c.Entity)
		case NoMovementType:
			NoMoveAction(g, c.Entity)
		case GoToPlayerMoveType:
			GoToPlayerMoveAction(g, c.Entity)
		case InvalidMovementType:
			fmt.Print("Error Finding movement type")

		}

	}

}
