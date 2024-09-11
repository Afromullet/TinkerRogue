package actionmanager

import (
	"fmt"
	"sort"
)

var ActionDispatcher ActionController

// Each Queue are all the queued actions by an Entity
type ActionController struct {
	EntityActions []*ActionQueue
}

func (am *ActionController) AddActionQueue(aq *ActionQueue) {

	if !am.containsActionQueue(aq) {

		am.EntityActions = append(am.EntityActions, aq)
		am.reorder()

	}
}

// Removes any ActionQueue without actions
func (am *ActionController) CleanController() {

	remainingActions := make([]*ActionQueue, 0)

	for _, act := range am.EntityActions {

		if act.NumOfActions() > 0 {
			remainingActions = append(remainingActions, act)

		}

	}

	am.EntityActions = remainingActions

}

func (am *ActionController) reorder() {
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
func (am *ActionController) ExecuteFirst() {

	if len(am.EntityActions) > 0 {

		am.EntityActions[0].ExecuteAction()

	}

	am.reorder()

}

func (am *ActionController) containsActionQueue(aq *ActionQueue) bool {
	for _, existingAQ := range am.EntityActions {
		if existingAQ == aq {
			return true
		}
	}
	return false
}

func (am *ActionController) DebugOutput() {
	for i, q := range am.EntityActions {
		fmt.Println("Printing Action Cost ", i, q.TotalActionPoints, " ")

	}

}
