package monsters

import (
	"fmt"
	"game_main/avatar"
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"
	"game_main/timesystem"
	"game_main/trackers"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

// Currently used for determining whether to spawn a new monster
var NumMonstersOnMap int
var CreatureComponent *ecs.Component

// EffectsToApply trigger every turn in MonsterSystems
// The Path is updated by a Movement Component
type Creature struct {
	Path []common.Position

	StatEffectTracker trackers.StatusEffectTracker
}

// This gets called so often that it might as well be a function
func GetCreature(e *ecs.Entity) *Creature {
	return common.GetComponentType[*Creature](e, CreatureComponent)
}

// TODO stack the effects if they're of the same kind
// Add stuff together and keep the longest duration
func (c *Creature) AddEffects(effects *ecs.Entity) {

	allEffects := gear.AllStatusEffects(effects)

	for _, e := range allEffects {
		c.StatEffectTracker.Add(e)
	}

	fmt.Println("Printing status effects ", c.StatEffectTracker)

}

/*
func (c Creature) GetEffect(effect *ecs.Entity) {

	for _, e := range c.EffectsToApply {

		if e.StatusEffectName() == effect

	}
}
*/

// Gets called once per turn. Applies all status effects to the creature
// Each effect implements the ApplyToCreature effect that determines...the kind of effect
func ApplyStatusEffects(c *ecs.QueryResult) {

	creature := c.Components[CreatureComponent].(*Creature)

	msg := common.GetComponentType[*common.UserMessage](c.Entity, common.UserMsgComponent)

	if len(creature.StatEffectTracker.ActiveEffects) == 0 {
		return
	}

	for key, eff := range creature.StatEffectTracker.ActiveEffects {

		if eff.Duration() >= 1 {
			eff.ApplyToCreature(c)

		}

		//ApplyTodCreature changes the duration, so we need to check again before
		//Deciding whether to keep the effect
		fmt.Println("Printing duration ", eff.Duration())
		if eff.Duration() == 0 {
			delete(creature.StatEffectTracker.ActiveEffects, key)

		}

	}

	msg.StatusEffectMessage = creature.StatEffectTracker.ActiveEffectNames()

	fmt.Println("Printing active effects")

	for _, eff := range creature.StatEffectTracker.ActiveEffects {

		fmt.Println(eff.StatusEffectName())

	}

}

// Get the next position on the path and pops the position from the path.
// Passing currentPosition so we can stand in place when there is no path
// TODO needs to be improved. This will cause a creature to "teleport" if the path is blocked
func (c *Creature) UpdatePosition(gm *worldmap.GameMap, currentPosition *common.Position) {

	p := currentPosition

	index := graphics.CoordTransformer.IndexFromLogicalXY(p.X, p.Y)
	oldTile := gm.Tiles[index]

	if len(c.Path) > 1 {
		p = &c.Path[1]
		c.Path = c.Path[2:]

	} else if len(c.Path) == 1 {

		//If there's just one entry left, then that's the current position
		c.Path = c.Path[:0]
	}

	index = graphics.CoordTransformer.IndexFromLogicalXY(p.X, p.Y)

	nextTile := gm.Tiles[index]

	if !nextTile.Blocked {

		currentPosition.X = p.X
		currentPosition.Y = p.Y
		nextTile.Blocked = true
		oldTile.Blocked = false

	}

}

// Returns a description on the entity to display to the player
func (c *Creature) DisplayString(e *ecs.Entity) string {

	attr := common.GetAttributes(e)
	name := common.GetComponentType[*common.Name](e, common.NameComponent)
	cr := common.GetComponentType[*Creature](e, CreatureComponent)

	result := fmt.Sprintln("Name ", name.NameStr)
	result += fmt.Sprintln("Health", attr.CurrentHealth, attr.MaxHealth)
	result += fmt.Sprintln(attr.DisplayString())
	result += fmt.Sprintln(cr.StatEffectTracker.ActiveEffectNames())

	return result
}

/*
// Returns a description on the entity to display to the player
func CreatureDescription(e *ecs.Entity) string {

	attr := common.GetAttributes(e)
	name := common.GetComponentType[*common.Name](e, common.NameComponent)
	cr := common.GetComponentType[*Creature](e, CreatureComponent)

	result := fmt.Sprintln("Name ", name.NameStr)
	result += fmt.Sprintln("Health", attr.CurrentHealth, attr.MaxHealth)
	result += fmt.Sprintln(attr.DisplayString())
	result += fmt.Sprintln(cr.StatEffectTracker.ActiveEffectNames())

	return result
}
*/

// Currently executes all actions just as it did before, this time only doing it through the AllActions queue
// Will change later once the time system is implemented. Still want things to behave the same while implementing the time system
func MonsterSystems(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, ts *timesystem.GameTurn) {

	NumMonstersOnMap = 0
	actionCost := 0
	//TODO do I need to make sure the same action can't be added twice?
	for _, c := range ecsmanger.World.Query(ecsmanger.WorldTags["monsters"]) {

		if c.Entity != nil {
			actionQueue := common.GetComponentType[*timesystem.ActionQueue](c.Entity, timesystem.ActionQueueComponent)
			attr := common.GetComponentType[*common.Attributes](c.Entity, common.AttributeComponent)
			gear.UpdateEntityAttributes(c.Entity)
			ApplyStatusEffects(c)
			//gear.ConsumableEffectApplier(c.Entity)

			if actionQueue != nil {

				act := CreatureMovementSystem(ecsmanger, gm, c)
				if act != nil {
					actionQueue.AddMonsterAction(act, attr.TotalMovementSpeed, timesystem.MovementKind)
				}

				act, actionCost = CreatureAttackSystem(ecsmanger, pl, gm, c)

				if act != nil {

					actionQueue.AddMonsterAction(act, actionCost, timesystem.AttackKind)
				}

			}
		}

		NumMonstersOnMap++

	}

}

// Todo clear action queues too
func ClearAllCreatures(ecsmanger *common.EntityManager) {
	for _, c := range ecsmanger.World.Query(ecsmanger.WorldTags["monsters"]) {

		ecsmanger.World.DisposeEntity(c.Entity)

	}
}

// Todo remove later. This is currently a duplicate - also in resourcemanager
// Here to avoid circular imports
func RemoveEntity(world *ecs.Manager, gm *worldmap.GameMap, e *ecs.Entity) {

	attr := common.GetComponentType[*common.Attributes](e, common.AttributeComponent)

	if attr.CurrentHealth > 0 {
		return
	}

	pos := common.GetPosition(e) //Todo replace pos with position from pos tracker

	fmt.Println("Starting length ", len(trackers.CreatureTracker.PosTracker))
	trackers.CreatureTracker.Remove(e)

	ind := graphics.CoordTransformer.IndexFromLogicalXY(pos.X, pos.Y)
	gm.Tiles[ind].Blocked = false

	timesystem.TurnManager.ActionDispatcher.RemoveActionQueueForEntity(e)
	world.DisposeEntity(e)
	NumMonstersOnMap--
	if NumMonstersOnMap == -1 {
		NumMonstersOnMap = 0
	}

	fmt.Println("Ending length ", len(trackers.CreatureTracker.PosTracker))

}
