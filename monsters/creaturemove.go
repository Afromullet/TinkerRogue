package monsters

import (
	"game_main/common"
	"game_main/graphics"
	"game_main/pathfinding"
	"game_main/randgen"
	"game_main/timesystem"
	"game_main/worldmap"
	"math"

	"github.com/bytearena/ecs"
)

/*
1) Create the component
2) Create the struct associated with the component
3) Implement the function that handles the movement.
4) In the Action package, create a wrapper for the function. See the comments in the actionmanager on how to do that.
5) Return the wrapper in the CreatureAttackSystem


*/
// movementAction is a map that will make it easier to call the movement functions.
// so that we no not have to add a conditional for every movement type in the MovementSystem
type MovementFunction func(ecsmanager *common.EntityManager, gm *worldmap.GameMap, mover *ecs.Entity)

var (
	SimpleWanderComp     *ecs.Component
	EntityFollowComp     *ecs.Component
	WithinRangeComponent *ecs.Component
	FleeComp             *ecs.Component

	MovementTypes []*ecs.Component
)

// Todo considering adding an interface with "BuildPath" m

type SimpleWander struct {
}

type EntityFollow struct {
	Target *ecs.Entity
}

// Todo need a better name for this. This is the kind of movement that does something in relation to the
// Target Entity, such as staying within a radius, fleeing from it, etc.
// Anything that uses DistanceToEntityMovement determines its movement in relation ot the target
type DistanceToEntityMovement struct {
	Target   *ecs.Entity
	Distance int
}

// Each Movement function implementation choses how to build a path.
// Must handle how the path updates every turn
// Be sure to call UpdatePosition

// Wander to a random position. Build a new path once arrived.
func SimpleWanderAction(ecsmanager *common.EntityManager, gm *worldmap.GameMap, mover *ecs.Entity) {

	creature := GetCreature(mover)
	creaturePosition := common.GetPosition(mover)

	randomPos := randgen.GetRandomBetween(0, len(worldmap.ValidPos.Pos))
	endPos := worldmap.ValidPos.Get(randomPos)

	//Only create a new path if one doesn't exist yet.
	if len(creature.Path) == 0 {

		astar := pathfinding.AStar{}
		creature.Path = astar.GetPath(*gm, creaturePosition, endPos, false)

	}

	creature.UpdatePosition(gm, creaturePosition)

}

func NoMoveAction(ecsmanager *common.EntityManager, gm *worldmap.GameMap, mover *ecs.Entity) {

}

func EntityFollowMoveAction(ecsmanager *common.EntityManager, gm *worldmap.GameMap, mover *ecs.Entity) {

	creature := GetCreature(mover)
	creaturePosition := common.GetPosition(mover)
	goToEnt := common.GetComponentType[*EntityFollow](mover, EntityFollowComp)

	if goToEnt.Target != nil {
		targetPos := common.GetComponentType[*common.Position](goToEnt.Target, common.PositionComponent)

		creature.Path = pathfinding.BuildPath(gm, creaturePosition, targetPos)

	}

	creature.UpdatePosition(gm, creaturePosition)

}

// Clears the path once within range
func WithinRangeMoveAction(ecsmanager *common.EntityManager, gm *worldmap.GameMap, mover *ecs.Entity) {

	creature := GetCreature(mover)
	creaturePosition := common.GetPosition(mover)
	within := common.GetComponentType[*DistanceToEntityMovement](mover, WithinRangeComponent)

	if within != nil && within.Target != nil {
		targetPos := common.GetComponentType[*common.Position](within.Target, common.PositionComponent)

		if targetPos.InRange(creaturePosition, within.Distance) {
			creature.Path = creature.Path[:0]

		} else {
			creature.Path = pathfinding.BuildPath(gm, creaturePosition, targetPos)

		}

	}

	creature.UpdatePosition(gm, creaturePosition)

}

// Also needs improvement
func FleeFromEntityMovementAction(ecsmanager *common.EntityManager, gm *worldmap.GameMap, mover *ecs.Entity) {
	fleeMov := common.GetComponentType[*DistanceToEntityMovement](mover, FleeComp)
	creature := GetCreature(mover)
	creaturePosition := common.GetPosition(mover)

	if fleeMov.Target != nil {
		targetPosition := common.GetComponentType[*common.Position](fleeMov.Target, common.PositionComponent)

		fleeVectorX := creaturePosition.X - targetPosition.X
		fleeVectorY := creaturePosition.Y - targetPosition.Y
		vectorLength := math.Sqrt(float64(fleeVectorX*fleeVectorX + fleeVectorY*fleeVectorY))

		normalizedX := float64(fleeVectorX) / vectorLength
		normalizedY := float64(fleeVectorY) / vectorLength

		// Try up to 3 times (initial + 3 retries)
		for attempt := 0; attempt < 3; attempt++ {
			// Scale the flee vector by fleeDistance and try different directions
			angleOffset := float64(attempt) * (math.Pi / 8.0) // 8 possible directions (22.5-degree steps)
			fleeTargetX := creaturePosition.X + int(normalizedX*float64(fleeMov.Distance)*math.Cos(angleOffset)) - int(normalizedY*float64(fleeMov.Distance)*math.Sin(angleOffset))
			fleeTargetY := creaturePosition.Y + int(normalizedX*float64(fleeMov.Distance)*math.Sin(angleOffset)) + int(normalizedY*float64(fleeMov.Distance)*math.Cos(angleOffset))

			fleePosition := common.Position{X: fleeTargetX, Y: fleeTargetY}

			if graphics.InBounds(fleeTargetX, fleeTargetY) {
				targetIndex := graphics.IndexFromLogicalXY(fleeTargetX, fleeTargetY)
				if !gm.Tiles[targetIndex].Blocked {
					// Use InRange to check if the flee position is within the desired range
					if creaturePosition.InRange(&fleePosition, fleeMov.Distance) {
						// Set the path to the flee destination
						path := pathfinding.BuildPath(gm, creaturePosition, &fleePosition)
						creature.Path = path

						return
					}
				}
			}

			// Adjust distance slightly for the next attempt if needed
			fleeMov.Distance = max(fleeMov.Distance-1, 1) // Ensure distance doesn't go below 1
		}

		// If no valid flee destination was found, you can add additional fallback logic here if necessary
	}

	creature.UpdatePosition(gm, creaturePosition)
}

func CreatureMovementSystem(ecsmanager *common.EntityManager, gm *worldmap.GameMap, c *ecs.QueryResult) timesystem.ActionWrapper {

	//var ok bool
	var ok bool

	// Todo need to avoid friendly fire

	if _, ok = c.Entity.GetComponentData(SimpleWanderComp); ok {
		return timesystem.NewEntityMover(SimpleWanderAction, ecsmanager, gm, c.Entity)
	}

	if _, ok = c.Entity.GetComponentData(EntityFollowComp); ok {
		return timesystem.NewEntityMover(EntityFollowMoveAction, ecsmanager, gm, c.Entity)

	}

	if _, ok = c.Entity.GetComponentData(WithinRangeComponent); ok {
		return timesystem.NewEntityMover(WithinRangeMoveAction, ecsmanager, gm, c.Entity)

	}

	if _, ok = c.Entity.GetComponentData(FleeComp); ok {
		return timesystem.NewEntityMover(FleeFromEntityMovementAction, ecsmanager, gm, c.Entity)

	}

	return nil

}

func InitializeMovementComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	SimpleWanderComp = manager.NewComponent()
	EntityFollowComp = manager.NewComponent()
	WithinRangeComponent = manager.NewComponent()
	FleeComp = manager.NewComponent()

	MovementTypes = append(MovementTypes, SimpleWanderComp, EntityFollowComp, WithinRangeComponent, FleeComp)

}
