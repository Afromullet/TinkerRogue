package timesystem

import (
	"github.com/bytearena/ecs"
)

var InitiativeComponent *ecs.Component

// Simple initiative-based turn system
// Entities act in order, then go to the end of the queue

type InitiativeQueue struct {
	Entities []*ecs.Entity
}

func NewInitiativeQueue() *InitiativeQueue {
	return &InitiativeQueue{
		Entities: make([]*ecs.Entity, 0),
	}
}

func (iq *InitiativeQueue) AddEntity(entity *ecs.Entity) {
	if !iq.containsEntity(entity) {
		iq.Entities = append(iq.Entities, entity)
	}
}

func (iq *InitiativeQueue) RemoveEntity(entity *ecs.Entity) {
	for i, e := range iq.Entities {
		if e == entity {
			iq.Entities = append(iq.Entities[:i], iq.Entities[i+1:]...)
			return
		}
	}
}

func (iq *InitiativeQueue) GetCurrentEntity() *ecs.Entity {
	if len(iq.Entities) > 0 {
		return iq.Entities[0]
	}
	return nil
}

func (iq *InitiativeQueue) NextTurn() {
	if len(iq.Entities) > 0 {
		// Move first entity to end of queue
		first := iq.Entities[0]
		iq.Entities = append(iq.Entities[1:], first)
	}
}

func (iq *InitiativeQueue) containsEntity(entity *ecs.Entity) bool {
	for _, e := range iq.Entities {
		if e == entity {
			return true
		}
	}
	return false
}

func (iq *InitiativeQueue) IsEmpty() bool {
	return len(iq.Entities) == 0
}

func (iq *InitiativeQueue) Size() int {
	return len(iq.Entities)
}

// Clean up any nil entities
func (iq *InitiativeQueue) CleanUp() {
	validEntities := make([]*ecs.Entity, 0)
	for _, entity := range iq.Entities {
		if entity != nil {
			validEntities = append(validEntities, entity)
		}
	}
	iq.Entities = validEntities
}