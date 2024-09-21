package timesystem

import (
	"fmt"
	"game_main/avatar"
	"sort"

	"github.com/bytearena/ecs"
)

// The ActionManager contains a slice of all ActionQueues and executes the actions for all entities
// Pushes an action to the back of the slice once it's been performed. The action continues to be
// Performed unless there's a reason to stop doing it.

type ActionManager struct {
	EntityActions []*ActionQueue
}

// Runs through the queue once and performs the actions until we reach the players action
// This does not work as intended because it removes the action from the queue.
// It needs to execute the action and then place it in the correct place in the queue
func (am *ActionManager) ExecuteActionsUntilPlayer(pl *avatar.PlayerData) bool {

	executedActions := make([]*ActionQueue, 0)
	//playerFound := false
	for i, act := range am.EntityActions {

		if act.Entity == pl.PlayerEntity {

			am.EntityActions = append(am.EntityActions[i:], executedActions...)
			return true
		}

		act.ExecuteAction()

		executedActions = append(executedActions, act)

	}

	am.EntityActions = append(am.EntityActions[len(executedActions):], executedActions...)

	return false

}

// Does not work as intended. ExecuteActionsUntilPlayer executes each action only once until we reach the player
// This intends to fix it by inserting the queue back in priority order so that the time system works as intended
func (am *ActionManager) ExecuteActionsUntilPlayer2(pl *avatar.PlayerData) bool {
	for {
		if am.EntityActions[0].Entity == pl.PlayerEntity {
			return true
		}

		// Execute the action for the entity with the highest TotalActionPoints
		am.EntityActions[0].ExecuteActionWithoutRemoving()

		// Check if the entity has enough TotalActionPoints to continue acting
		if am.EntityActions[0].TotalActionPoints > 0 {
			// Move the action queue to the correct position
			act := am.EntityActions[0]
			am.EntityActions = am.EntityActions[1:]                 // Remove from the front
			am.EntityActions = insertInOrder(am.EntityActions, act) // Reinsert in the correct order
		} else {
			// No more action points, simply move to the back of the queue
			am.EntityActions = append(am.EntityActions[1:], am.EntityActions[0])
		}
	}
}

// Helper function to insert an ActionQueue back into the slice in the correct order
func insertInOrder(actions []*ActionQueue, act *ActionQueue) []*ActionQueue {
	index := sort.Search(len(actions), func(i int) bool {
		if actions[i].TotalActionPoints == act.TotalActionPoints {
			return i < len(actions)-1 // Ensures stable order for equal points
		}
		return actions[i].TotalActionPoints < act.TotalActionPoints
	})
	actions = append(actions[:index], append([]*ActionQueue{act}, actions[index:]...)...)
	return actions
}

func (am *ActionManager) AddActionQueue(aq *ActionQueue) {

	if !am.containsActionQueue(aq) {

		am.EntityActions = append(am.EntityActions, aq)
		am.ReorderActions()

	}
}

// Removes any ActionQueue without an entity
func (am *ActionManager) CleanController() {

	remainingActions := make([]*ActionQueue, 0)

	for _, act := range am.EntityActions {

		if act.Entity != nil {
			remainingActions = append(remainingActions, act)

		}

	}

	am.EntityActions = remainingActions

}

// Clears all actions
func (am *ActionManager) ResetActionManager() {
	for _, act := range am.EntityActions {
		act.ResetQueue()

	}

}

// Sorts the actionqueues in priority order.
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

func (am *ActionManager) ExecuteFirst() {

	if len(am.EntityActions) > 0 {

		am.EntityActions[0].ExecuteAction()
		//firstAction := am.EntityActions[0]
		//am.EntityActions = append(am.EntityActions[1:], firstAction)
	}

}

// Action points get reset every n number of turns.
func (am ActionManager) ResetActionPoints() {

	for _, q := range am.EntityActions {
		q.ResetActionPoints()
	}
}

func (am *ActionManager) RemoveActionQueueForEntity(entity *ecs.Entity) {
	for i, actionQueue := range am.EntityActions {
		if actionQueue.Entity == entity {

			am.EntityActions = append(am.EntityActions[:i], am.EntityActions[i+1:]...)
			return
		}
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
