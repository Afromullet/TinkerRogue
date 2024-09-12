package common

import (
	"fmt"

	"github.com/bytearena/ecs"
)

var (
	PositionComponent   *ecs.Component
	NameComponent       *ecs.Component
	AttributeComponent  *ecs.Component
	RenderableComponent *ecs.Component //Putting this here for now rather than in graphics
	UsrMsg              *ecs.Component //I can probably remove this later
)

type EntityManager struct {
	World     *ecs.Manager
	WorldTags map[string]ecs.Tag
}

// A wrapper around the ECS libraries GetComponentData.
// Sometimes I make decisions to implement somethign to reduce typing, which isn't always a good idea as I've learned
// Using this until I realize it's a bad idea
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

// Calculate the Manhattan distance between two entities. Both entities need a position component
func DistanceBetween(e1 *ecs.Entity, e2 *ecs.Entity) int {

	pos1 := GetPosition(e1)
	pos2 := GetPosition(e2)

	return pos1.ManhattanDistance(pos2)

}

func GetAttributes(e *ecs.Entity) *Attributes {
	return GetComponentType[*Attributes](e, AttributeComponent)
}

func GetPosition(e *ecs.Entity) *Position {
	return GetComponentType[*Position](e, PositionComponent)
}
