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
		firstAction := am.EntityActions[0]
		am.EntityActions = append(am.EntityActions[1:], firstAction)
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
