package timesystem

import (
	"github.com/bytearena/ecs"
)

// Todo add comments on why and how the ActionWrapper is used
type ActionWrapper interface {
	Execute(q *ActionQueue)
}

// KindOfAction is used so that an action can be added to the ActionQueue only once.
// There's probably a better way to do it, but for now we just need something that works
type KindOfAction int

const (
	MovementKind = iota
	AttackKind
	MeleeAttackInd
	RangedAttackKind
	PickupItemKind
)

// Only monsters use priority
func GetActionPriority(kind KindOfAction) int {

	switch kind {
	case AttackKind:
		return 5
	case MeleeAttackInd:
		return 4
	case RangedAttackKind:
		return 2
	case MovementKind:
		return 2
	case PickupItemKind:
		return 1
	default:
		return 1
	}

}

var ActionQueueComponent *ecs.Component

// Cost is substracted from the ActionQueues TotalEnergy when an entity peforms an action
type Action struct {
	ActWrapper   ActionWrapper
	Cost         int
	kindOfAction KindOfAction
	priority     int
}

// Every entity has an ActionQueue. Whenever an entity wants to perform an action, it's added to its ActionQueue.
// The ActionManager handles the queues of all entities
// ExecuteAction performs the first action in the queue and removes it.
type ActionQueue struct {
	TotalActionPoints int
	AllActions        []Action
	Entity            *ecs.Entity //Entity associated with the queue
}

// Removes the first action in the queue. Used after executing the action.
func (a *ActionQueue) pop() {
	if len(a.AllActions) > 0 {
		a.AllActions = a.AllActions[1:]
	}
}

func (a *ActionQueue) ResetActionPoints() {
	a.TotalActionPoints = 100
}

// Player Queue does not use priority
// Only allow the same kind of action to be added once.
func (a *ActionQueue) AddPlayerAction(action ActionWrapper, actionPointCost int, kindOfAction KindOfAction) {

	if actionPointCost == 0 {
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

// Monster actions have a priority, so that's why it's a separate function
func (a *ActionQueue) AddMonsterAction(action ActionWrapper, actionPointCost int, kindOfAction KindOfAction) {

	if actionPointCost <= 0 {
		panic("Action points must be greater than or equal to 0")
	}

	// Ensure the action is unique (no duplicate actions of the same type)
	for _, act := range a.AllActions {
		if act.kindOfAction == kindOfAction {
			return
		}
	}

	// Create the new action
	newAction := Action{ActWrapper: action, Cost: actionPointCost, kindOfAction: kindOfAction}
	newActionPriority := GetActionPriority(kindOfAction)

	// Insert the new action in priority order
	inserted := false
	for i, act := range a.AllActions {
		existingActionPriority := GetActionPriority(act.kindOfAction)
		if newActionPriority > existingActionPriority {
			// Insert new action at the correct position
			a.AllActions = append(a.AllActions[:i], append([]Action{newAction}, a.AllActions[i:]...)...)
			inserted = true
			break
		}
	}

	// If the new action has the lowest priority, append it to the end of the queue
	if !inserted {
		a.AllActions = append(a.AllActions, newAction)
	}
}

// Executes the first action and removes it from the queue
func (a *ActionQueue) ExecuteAction() {
	if len(a.AllActions) > 0 {

		a.TotalActionPoints -= a.AllActions[0].Cost
		a.AllActions[0].ActWrapper.Execute(a)

		a.pop()
	}
}

// Executes the first action without removing it from the queue.

func (a *ActionQueue) ExecuteActionWithoutRemoving() {
	if len(a.AllActions) > 0 {

		a.TotalActionPoints -= a.AllActions[0].Cost
		a.AllActions[0].ActWrapper.Execute(a)

	}

}

func (a *ActionQueue) NumOfActions() int {
	return len(a.AllActions)
}

func (a *ActionQueue) ResetQueue() {
	a.AllActions = nil
	a.AllActions = make([]Action, 0)
}
