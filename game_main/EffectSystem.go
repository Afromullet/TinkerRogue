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
	eff := creature.EffectsToApply

	ApplyEffects(creature)
	fmt.Println("Printing the effects ", eff)

	/*
		effects := creature.EffectsOnCreature
		health := c.Components[healthComponent].(*Health)
		fmt.Println(health)

		if creature.EffectsOnCreature == nil {
			return
		}

		if creature.EffectsOnCreature.HasComponent(BurningComponent) {
			CreatureBurningSystem(c, effects)

		}
	*/

}

func CreatureBurningSystem(c *ecs.QueryResult, effects *ecs.Entity) {

	fmt.Println("Creature is burning")

	health := c.Components[healthComponent].(*Health)

	//data, _ := effects.GetComponentData(BurningComponent)
	//d := data.(*Effects)
	//b := (*d).(Burning)

	//fmt.Println("Printing conv ", b)

	//b.MainProps.Duration -= 1

	///	fmt.Println("Printing duration", b.MainProps.Duration)

	//b := d.(Burning)

	//bur := GetComponentType[Effects](effects, BurningComponent)
	//fmt.Println(bur)

	//	if bur.MainProps.Duration > 0 {
	//	health.CurrentHealth -= bur.Temperature
	//	bur.MainProps.Duration -= 1

	//	}

	fmt.Println("Current health", health)

}
