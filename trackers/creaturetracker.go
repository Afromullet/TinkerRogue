package trackers

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// Todo add creatures to here when they are spawned
// Check whenever we create a creature entity and make sure it's added

var CreatureTracker = NewCreatureTracker()

// Used to quickly look up creature positions. Several parts of the code iterate
// Over all of the monsters, even when we just need a few
type PositionTracker struct {
	PosTracker map[*common.Position]*ecs.Entity
}

// Todo make sure the same entity can't be added twice
func (t *PositionTracker) Add(e *ecs.Entity) {

	if _, exists := t.PosTracker[common.GetPosition(e)]; exists {
		//Something really went wrong here
		panic("entity already in map")
	} else {
		//Something really went wrong here, so we want to throw a panic
		t.PosTracker[common.GetPosition(e)] = e
	}

}

func (t *PositionTracker) Remove(e *ecs.Entity) {
	for key, ent := range t.PosTracker {
		if ent == e {
			delete(t.PosTracker, key)
		}
	}
}

func NewCreatureTracker() PositionTracker {
	return PositionTracker{
		PosTracker: make(map[*common.Position]*ecs.Entity),
	}
}
