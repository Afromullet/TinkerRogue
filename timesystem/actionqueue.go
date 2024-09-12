package timesystem

import (
	"github.com/bytearena/ecs"
)

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

// Every entity has an ActionQueue. Whenever an entity wants to perform an action, it's added to its ActionQueue.
// An action needs an ActionWrapper to add to the Queue.
// ExecuteAction performs the first action in the queue and removes it.
type ActionQueue struct {
	TotalActionPoints int
	AllActions        []Action
	Entity            *ecs.Entity //Entity associated with the queue
}

// Removes the first action in the queue.
func (a *ActionQueue) pop() {
	if len(a.AllActions) > 0 {
		a.AllActions = a.AllActions[1:]
	}
}

func (a *ActionQueue) ResetActionPoints() {
	a.TotalActionPoints = 100
}

// Only allow the same kind of action to be added once.
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

func (a *ActionQueue) ResetQueue() {
	a.AllActions = nil
	a.AllActions = make([]Action, 0)
}
