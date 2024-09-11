package timesystem

import (
	"fmt"
	"sort"
)

/*
Each entity has an ActionQueue, which contains the Total Energy and a list of actions the entity has queued up.

The ActionManager is a slice of all entity queues. It's sorted from highest to lowest Total Energy, meaning that the creature
with the highest energy is in front of the queue.

Whenever an entity performs an action, the action is added to its own queue. After the player and the entities act, the actions begin to execute.
The actions only execute after the player and all monsters submitted their action.

The part I don't have yet is energy recovery. I need to break the time management system down into more discrete units, so that after x amount of time
every entity recovers energy. I don't want to base this on "actual" time - it has to be every so many action cycles.


*/

var ActionDispatcher ActionManager

// Each Queue are all the queued actions by an Entity
type ActionManager struct {
	EntityActions []*ActionQueue
}

func (am *ActionManager) AddActionQueue(aq *ActionQueue) {

	if !am.containsActionQueue(aq) {

		am.EntityActions = append(am.EntityActions, aq)
		am.ReorderActions()

	}
}

// Removes any ActionQueue without actions
func (am *ActionManager) CleanController() {

	remainingActions := make([]*ActionQueue, 0)

	for _, act := range am.EntityActions {

		if act.NumOfActions() > 0 {
			remainingActions = append(remainingActions, act)

		}

	}

	am.EntityActions = remainingActions

}

func (am *ActionManager) ReorderActions() {
	sort.Slice(am.EntityActions, func(i, j int) bool {
		// Sort by TotalActionPoints in descending order
		if am.EntityActions[i].TotalActionPoints == am.EntityActions[j].TotalActionPoints {
			// If they have the same points, move the one further down (higher index) ahead
			return i > j
		}
		return am.EntityActions[i].TotalActionPoints > am.EntityActions[j].TotalActionPoints
	})
}

// Todo handle case where there ius no action in the first Action queue
func (am *ActionManager) ExecuteFirst() {

	if len(am.EntityActions) > 0 {

		am.EntityActions[0].ExecuteAction()

	}

}

func (am *ActionManager) containsActionQueue(aq *ActionQueue) bool {
	for _, existingAQ := range am.EntityActions {
		if existingAQ == aq {
			return true
		}
	}
	return false
}

func (am *ActionManager) DebugOutput() {
	for i, q := range am.EntityActions {
		fmt.Println("Printing Action Cost ", i, q.TotalActionPoints, " ")

	}

}
