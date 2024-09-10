package monsters

import (
	"fmt"
	"game_main/actionmanager"
	"game_main/avatar"
	"game_main/common"
	"game_main/equipment"
	"game_main/graphics"
	"game_main/timesystem"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

var CreatureComponent *ecs.Component

// EffectsToApply trigger every turn in MonsterSystems
// The only thing applying an Effect Right now are throwable.
// The Path is updated by a Movement Component
type Creature struct {
	Path           []common.Position
	EffectsToApply []equipment.StatusEffects
}

// This gets called so often that it might as well be a function
func GetCreature(e *ecs.Entity) *Creature {
	return common.GetComponentType[*Creature](e, CreatureComponent)
}

// TODO stack the effects if they're of the same kind
// Add stuff together and keep the longest duration
func (c *Creature) AddEffects(effects *ecs.Entity) {

	e := equipment.AllStatusEffects(effects)
	c.EffectsToApply = append(c.EffectsToApply, e...)

}

// Gets called in MonsterSystems, which queries the ECS manager and returns query results containing all monsters
// Querying returns an ecs.queryResult, hence the parameter.
func ApplyEffects(c *ecs.QueryResult) {

	creature := c.Components[CreatureComponent].(*Creature)
	num_effects := len(creature.EffectsToApply)

	if num_effects == 0 {
		return
	}

	effects_to_keep := make([]equipment.StatusEffects, 0)

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
func (c *Creature) UpdatePosition(gm *worldmap.GameMap, currentPosition *common.Position) {

	p := currentPosition

	index := graphics.IndexFromXY(p.X, p.Y)
	oldTile := gm.Tiles[index]

	if len(c.Path) > 1 {
		p = &c.Path[1]
		c.Path = c.Path[2:]

	} else if len(c.Path) == 1 {

		//If there's just one entry left, then that's the current position
		c.Path = c.Path[:0]
	}

	index = graphics.IndexFromXY(p.X, p.Y)

	nextTile := gm.Tiles[index]

	if !nextTile.Blocked {

		currentPosition.X = p.X
		currentPosition.Y = p.Y
		nextTile.Blocked = true
		oldTile.Blocked = false

	}

}

// todo still need to remove game
func MonsterSystems(ecsmanger *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, ts *timesystem.GameTurn) {

	a := actionmanager.Actions{}
	fmt.Println(a)
	for _, c := range ecsmanger.World.Query(ecsmanger.WorldTags["monsters"]) {

		ApplyEffects(c)

		retFunc := CreatureAttackSystem(ecsmanger, pl, gm, c)

		if retFunc != nil {

			a.AddAttackAction(retFunc, ecsmanger, pl, gm, c, pl.PlayerEntity)
		}

		h := common.GetComponentType[*common.Attributes](c.Entity, common.AttributeComponent)

		if h.CurrentHealth <= 0 {
			ecsmanger.World.DisposeEntity(c.Entity)
		}

		CreatureMovementSystem(ecsmanger, gm, c)

	}

	for _, acts := range a.AllActions {

		acts()

	}

	ts.Turn = timesystem.PlayerTurn

}
