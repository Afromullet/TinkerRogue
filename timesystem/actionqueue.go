package timesystem

import (
	"github.com/bytearena/ecs"
)

/*
The timesystem requires actions to be stored so that they can be called in different orders.

Different actions have different parameters.

There will be a wrapper struct for each set of parameters that implements the Action interface
*/

// Anything that adds an action to an ActionQueue does it through the ActionWrapper.
// Todo add comments on why and how the ActionWrapper is used
type ActionWrapper interface {
	Execute(q *ActionQueue)
}

// KindOfAction is used so that an action can be added to the ActionQueue only once
type KindOfAction int

const (
	MovementKind = iota
	AttackKind
	MeleeAttackInd
	RangedAttackKind
	PickupItemKind
)

var ActionQueueComponent *ecs.Component

// Cost is substracted from the ActionQueues TotalEnergy when an entity peforms an action
type Action struct {
	ActWrapper   ActionWrapper
	Cost         int
	kindOfAction KindOfAction
}

// Contains a slice of Actions an entity has queued up to perform.
// The MonsterSystem and Player Input handler adds actions to the queue.
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
// Don't want allow the same kind of action to be added twice, which is why there's a kindOfAction parameter
// Does not distinguish between melee and ranged attacks yet
func (a *ActionQueue) AddAction(action ActionWrapper, actionPointCost int, kindOfAction KindOfAction) {

	if actionPointCost <= 0 {
		panic("Action points must be greater than or equal to 0")
	}

	for _, act := range a.AllActions {

		if act.kindOfAction == kindOfAction {
			return
		}

	}

	if action != nil {
		a.AllActions = append(a.AllActions, Action{ActWrapper: action, Cost: actionPointCost, kindOfAction: kindOfAction})
	}
}

// Executes the first action
func (a *ActionQueue) ExecuteAction() {
	if len(a.AllActions) > 0 {

		a.TotalActionPoints -= a.AllActions[0].Cost
		a.AllActions[0].ActWrapper.Execute(a)
		a.pop()
	}
}

func (a *ActionQueue) NumOfActions() int {
	return len(a.AllActions)
}
