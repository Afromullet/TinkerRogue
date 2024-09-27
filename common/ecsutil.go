package common

import (
	"fmt"

	"github.com/bytearena/ecs"
)

var (
	PositionComponent  *ecs.Component
	NameComponent      *ecs.Component
	AttributeComponent *ecs.Component
)

// Wrapper around the ECS libraries manager and rags.
type EntityManager struct {
	World     *ecs.Manager
	WorldTags map[string]ecs.Tag
}

// A wrapper around the ECS libraries GetComponentData.
// It makes it a littel bit less tedious to get the struct assocaited with a component
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {

	defer func() {
		if r := recover(); r != nil {

			fmt.Println("Error in passing the component type. Component type must match struct.")

		}
	}()

	if c, ok := entity.GetComponentData(component); ok {
		return c.(T)

	} else {
		var nilValue T
		return nilValue
	}

}

// Calculate the Chebshev distance between two entities. Both entities need a position component
func DistanceBetween(e1 *ecs.Entity, e2 *ecs.Entity) int {

	pos1 := GetPosition(e1)
	pos2 := GetPosition(e2)

	return pos1.ChebyshevDistance(pos2)

}

// Getters for components which are referenced frequently.
func GetAttributes(e *ecs.Entity) *Attributes {
	return GetComponentType[*Attributes](e, AttributeComponent)
}

func GetPosition(e *ecs.Entity) *Position {
	return GetComponentType[*Position](e, PositionComponent)
}

// Todo need a better way to handle this rather than searching all monsters
// This also should not really be in attackingSystem
func GetCreatureAtPosition(ecsmnager *EntityManager, pos *Position) *ecs.Entity {

	var e *ecs.Entity = nil
	for _, c := range ecsmnager.World.Query(ecsmnager.WorldTags["monsters"]) {

		curPos := GetPosition(c.Entity)

		if pos.IsEqual(curPos) {
			e = c.Entity
			break
		}

	}

	return e

}
