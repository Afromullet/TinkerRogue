package main

import (
	"log"
)

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

		//If there's just one entry left, then that's the current position
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

func MonsterActions(g *Game) {

	log.Print("Monster moving")
	MovementSystem(g)

	g.Turn = PlayerTurn

}
