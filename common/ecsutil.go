package common

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// A wrapper around the ECS libraries GetComponentData.
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

// This gets called so often that it might as well be a function
func GetAttributes(e *ecs.Entity) *Attributes {
	return GetComponentType[*Attributes](e, AttributeComponent)
}

func DistanceBetween(e1 *ecs.Entity, e2 *ecs.Entity) int {

	pos1 := GetPosition(e1)
	pos2 := GetPosition(e2)

	return pos1.ManhattanDistance(pos2)

}

type EntityManager struct {
	World     *ecs.Manager
	WorldTags map[string]ecs.Tag
}
