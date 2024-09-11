package actionmanager

import (
	"fmt"

	"github.com/bytearena/ecs"
)

/*
The timesystem requires actions to be stored so that they can be called in different orders.

Different actions have different parameters.

There will be a wrapper struct for each set of parameters that implements the Action interface
*/

type KindOfAction int

const (
	MovementKind = iota
	AttackKind
	MeleeAttackInd
	RangedAttackKind
	PickupItemKind
)

var ActionQueueComponent *ecs.Component

type Action struct {
	ActWrapper   ActionWrapper
	Cost         int
	kindOfAction KindOfAction
}

// Contains a slice of Actions the entity has queued up to perform.
// ExecuteAction subtracts the action cost from the TotalActionPoints
type ActionQueue struct {
	TotalActionPoints int
	AllActions        []Action
}

// Removes the first action in the queue.
func (a *ActionQueue) pop() {
	if len(a.AllActions) > 0 {

		a.AllActions = a.AllActions[1:]
	}
}

// The actionPointCost is how much the action...costs to perform
// Don't allow the same kind of action to be added twice
func (a *ActionQueue) AddAction(action ActionWrapper, actionPointCost int, kindOfAction KindOfAction) {

	for _, act := range a.AllActions {

		if act.kindOfAction == kindOfAction {
			return
		}

	}

	if actionPointCost <= 0 {
		panic("Action points must be greater than or equal to 0")
	}

	if len(a.AllActions) == 0 {
		fmt.Println("Breha ")
	}

	if action != nil {
		a.AllActions = append(a.AllActions, Action{ActWrapper: action, Cost: actionPointCost, kindOfAction: kindOfAction})
	}
}

// Executes the first action
func (a *ActionQueue) ExecuteAction() {
	if len(a.AllActions) > 0 {
		fmt.Println("Printing cost ", a.AllActions[0].Cost)
		a.TotalActionPoints -= a.AllActions[0].Cost
		a.AllActions[0].ActWrapper.Execute(a)
		a.pop()
	}
}

func (a *ActionQueue) NumOfActions() int {
	return len(a.AllActions)
}

type ActionWrapper interface {
	Execute(q *ActionQueue)
}
