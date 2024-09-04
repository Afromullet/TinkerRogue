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
	Path           []Position
	EffectsToApply []Effects
}

// TODO stack the effects if they're of the same kind
// Add stuff together and keep the longest duration
func (c *Creature) AddEffects(effects *ecs.Entity) {

	e := AllEffects(effects)
	c.EffectsToApply = append(c.EffectsToApply, e...)

}

// The effect takes an ecs.QueryResult and chooses how the Effect
// impacts the Components. For example, the Burning Effect uses the query rtesult
// To get the creatures health and apply DOT damage.
func ApplyEffects(c *ecs.QueryResult) {

	creature := c.Components[creature].(*Creature)
	num_effects := len(creature.EffectsToApply)

	if num_effects == 0 {
		return
	}

	effects_to_keep := make([]Effects, 0)

	for _, eff := range creature.EffectsToApply {

		if eff.Duration() >= 1 {
			eff.ApplyToCreature(c)
		}

		//ApplyToCreature changes the duration, so we need to check again before
		//Deciding whether to keep the effect
		if eff.Duration() > 0 {
			effects_to_keep = append(effects_to_keep, eff)

		}

	}

	creature.EffectsToApply = effects_to_keep

}

// Get the next position on the path and pops the position from the path.
// Passing currentPosition so we can stand in place when there is no path
// TODO needs to be improved. This will cause a creature to "teleport" if the path is blocked
// Since we're removing the position from the path without any conditions
func (c *Creature) UpdatePosition(g *Game, currentPosition *Position) {

	p := currentPosition

	index := IndexFromXY(p.X, p.Y)
	oldTile := g.gameMap.Tiles[index]

	if len(c.Path) > 1 {
		p = &c.Path[1]
		c.Path = c.Path[2:]

	} else if len(c.Path) == 1 {

		//If there's just one entry left, then that's the current position
		c.Path = c.Path[:0]
	}

	index = IndexFromXY(p.X, p.Y)

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

		ApplyEffects(c)

		h := GetComponentType[*Attributes](c.Entity, attributeComponent)

		if h.CurrentHealth <= 0 {
			g.World.DisposeEntity(c.Entity)
		}

		MovementSystem(c, g)

	}

	g.Turn = PlayerTurn

}
