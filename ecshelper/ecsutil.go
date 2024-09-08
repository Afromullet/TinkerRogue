package ecshelper

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
