package main

import (
	"fmt"
	"log"

	"github.com/bytearena/ecs"
)

var creature *ecs.Component
var simpleWander *ecs.Component
var noMove *ecs.Component
var goToPlayer *ecs.Component

type Creature struct {
	path []Position
}

// Get the next position on the path.
// Pops the position from the slice
// Passing currentPosition so we can stand in place when there is no path
func (c *Creature) GetNextPosition(currentPosition *Position) *Position {

	p := currentPosition
	if len(c.path) > 0 {
		p = &c.path[1]
		c.path = c.path[2:]

	}

	return p
}

// Get the next position on the path.
// Pops the position from the slice
// Passing currentPosition so we can stand in place when there is no path
func (c *Creature) MoveToNextPosition(g *Game, oldPosition *Position) {

	p := oldPosition

	index := GetIndexFromXY(p.X, p.Y)
	oldTile := g.gameMap.Tiles[index]

	if len(c.path) > 1 {
		p = &c.path[1]
		c.path = c.path[2:]

	} else if len(c.path) == 1 {

		log.Print("Resetting path")

		c.path = c.path[:0]
	}

	index = GetIndexFromXY(p.X, p.Y)

	nextTile := g.gameMap.Tiles[index]

	if !nextTile.Blocked {

		oldPosition.X = p.X
		oldPosition.Y = p.Y
		nextTile.Blocked = true

		oldTile.Blocked = false

	}

}

// Build a path to the player and clear the existing path.
func (c *Creature) BuildPathToPlayer(g *Game, curPos *Position) {

	c.path = curPos.BuildPath(g, g.playerData.position)

}

type SimpleWander struct {
}

type NoMovement struct {
}

type GoToPlayerMovement struct {
}

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

func SimpleWanderAction(g *Game, e *ecs.Entity) {

	pos, _ := e.GetComponentData(position)
	c, _ := e.GetComponentData(creature)

	creature := c.(*Creature)
	creaturePosition := pos.(*Position)

	randomPos := GetRandomBetween(0, len(validPositions.positions))
	endPos := validPositions.Get(randomPos)

	//Only create a new path if one doesn't exist yet.
	if len(creature.path) == 0 {
		log.Print("Building new path")
		astar := AStar{}
		creature.path = astar.GetPath(g.gameMap, creaturePosition, endPos)

	}

	creature.MoveToNextPosition(g, creaturePosition)

}

func NoMoveAction(g *Game, e *ecs.Entity) {

}

func GoToPlayerMoveAction(g *Game, e *ecs.Entity) {

	pos, _ := e.GetComponentData(position)
	c, _ := e.GetComponentData(creature)

	creature := c.(*Creature)
	creaturePosition := pos.(*Position)

	creature.BuildPathToPlayer(g, creaturePosition)

	creature.MoveToNextPosition(g, creaturePosition)

}

func HandleMovement(g *Game) {

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

func MonsterActions(g *Game) {

	log.Print("Monster moving")
	HandleMovement(g)

	/*
		for _, m := range g.World.Query(g.WorldTags["simpleWander"]) {

			pos, _ := m.Entity.GetComponentData(position)
			c, _ := m.Entity.GetComponentData(creature)

			creature := c.(*Creature)
			creaturePosition := pos.(*Position)

			randomPos := GetRandomBetween(0, len(validPositions.positions))
			endPos := validPositions.Get(randomPos)

			//Only create a new path if one doesn't exist yet.
			if len(creature.path) == 0 {
				astar := AStar{}
				creature.path = astar.GetPath(g.gameMap, creaturePosition, endPos)

			}

			p := creature.GetNextPosition(creaturePosition)

			creaturePosition.X = p.X
			creaturePosition.Y = p.Y

		}

	*/

	/*
		for i, m := range g.World.Query(g.WorldTags["noMove"]) {

			mov, _ := m.Entity.GetComponentData(noMove)
			pos, _ := m.Entity.GetComponentData(position)

			position := pos.(*Position)

			log.Print("___No Move___")
			log.Print(mov)
			log.Print(position)
			log.Print(i)
			log.Print(m)
		}
	*/

	g.Turn = PlayerTurn

}
