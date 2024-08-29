package main

import (
	"log"
)

type Creature struct {
	path []Position
}

// Get the next position on the path and pops the position from the path.
// Passing currentPosition so we can stand in place when there is no path
// TODO needs to be improved. This will cause a creature to "teleport" if the path is blocked
// Since we're removing the position from the path without any conditions
func (c *Creature) UpdatePosition(g *Game, currentPosition *Position) {

	p := currentPosition

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

		currentPosition.X = p.X
		currentPosition.Y = p.Y
		nextTile.Blocked = true
		oldTile.Blocked = false

	}

}

// Build a path to the player. Accesses the position playerData
func (c *Creature) BuildPathToPlayer(g *Game, curPos *Position) {

	c.path = curPos.BuildPath(g, g.playerData.position)

}

// Currently only handles the movement
func MonsterActions(g *Game) {

	log.Print("Monster moving")
	MovementSystem(g)

	g.Turn = PlayerTurn

}
