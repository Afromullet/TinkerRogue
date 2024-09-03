package main

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// Todo find a better place for this.
// This applies all effects on the creature

// The thing we have to check to make sure doesn't happen:
// ItemProperty has CommonItemProperties which has a duration
// When decrementing the duration from the effect from the creature
// We want to make sure that we're not also decrementing the duration
// On the item that caused the effect
func CreatureEffectSystem(c *ecs.QueryResult) {

	creature := c.Components[creature].(*Creature)

	if creature.EffectsOnCreature == nil {
		return
	}

	if creature.EffectsOnCreature.HasComponent(BurningComponent) {
		CreatureBurningSystem(c)

	}

}

func CreatureBurningSystem(c *ecs.QueryResult) {

	fmt.Println("Creature is burning")

	//creature := c.Components[healthComponent].(*Health)

}
