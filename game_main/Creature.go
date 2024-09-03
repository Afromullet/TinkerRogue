package main

import (
	"github.com/bytearena/ecs"
)

// OLD COMMENTS
// EffectsToApply are the "components" that will be applied to a creature
// Currently these are : Itemcomponents and damage

// NEW COMMENTS
// EffectsOnCreature is an Entity so that we can access the underlying
// Effect Components. The Effect Components are ItemProperties such as
// Burning, Sticky, Freezing, ETC
// Note that because we're copying over the ItemProperties
// We're also copying some properties that don't mean anything to the creature
// We will worry about that later
type Creature struct {
	path []Position

	EffectsToApply []Effects
}

func (c *Creature) AddEffects(effects *ecs.Entity) {

	e := GetEffect(effects)

	c.EffectsToApply = append(c.EffectsToApply, e.(Effects))

}

func ApplyEffects(c *Creature) {

	for _, eff := range c.EffectsToApply {

		eff.ApplyToCreature(c)

	}

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

func MonsterActions(g *Game) {

	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		MovementSystem(c, g)
		CreatureEffectSystem(c)

	}

	g.Turn = PlayerTurn

}
