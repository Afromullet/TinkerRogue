package tracker

import (
	"game_main/common"
	"game_main/monsters"

	"github.com/bytearena/ecs"
)

// Todo add creatures to here when they are spawned
// Check whenever we create a creature entity and make sure it's added

var CreatureTracker = NewCreatureTracker()

// Used to quickly look up creature positions. Several parts of the code iterate
// Over all of the monsters, even when we just need a few
type CreaturePositionTracker struct {
	tracker map[*common.Position]*ecs.Entity
}

func (t *CreaturePositionTracker) Add(e *ecs.Entity) {

	if !e.HasComponent(monsters.CreatureComponent) {

		panic("entity does not have CreatureComponent")
	}

	if _, exists := t.tracker[common.GetPosition(e)]; exists {
		//Something really went wrong here
		panic("entity already in map")
	} else {
		//Something really went wrong here, so we want to throw a panic
		t.tracker[common.GetPosition(e)] = e
	}

}

func (t CreaturePositionTracker) Get(p *common.Position) *ecs.Entity {

	return t.tracker[p]

}

func NewCreatureTracker() CreaturePositionTracker {
	return CreaturePositionTracker{
		tracker: make(map[*common.Position]*ecs.Entity),
	}
}
