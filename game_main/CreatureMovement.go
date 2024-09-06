package main

import (
	"fmt"
	"math"

	"github.com/bytearena/ecs"
)

var simpleWander *ecs.Component
var noMove *ecs.Component
var entityFollowComponent *ecs.Component
var stayWithinRangeComponent *ecs.Component
var fleeComponent *ecs.Component

// Todo considering adding an interface with "BuildPath" m

type SimpleWander struct {
}

type NoMovement struct {
}

type EntityFollow struct {
	target *ecs.Entity
}

type StayWithinRange struct {
	target   *ecs.Entity
	distance int
}

// Flee until greater than or equal to the distance
type FleeMovement struct {
	target   *ecs.Entity
	distance int
}

// Each Movement function implementation choses how to build a path.
// They also handle walking on the path by calling reature.UpdatePosition
// Movement functions get called by MovementSystem which determines what movement component a creature has.

// Select a random spot to wander to and builds a new path when arriving at the position
func SimpleWanderAction(g *Game, mover *ecs.Entity) {

	creature := GetComponentType[*Creature](mover, CreatureComponent)

	creaturePosition := GetPosition(mover)

	randomPos := GetRandomBetween(0, len(validPositions.positions))
	endPos := validPositions.Get(randomPos)

	//Only create a new path if one doesn't exist yet.
	if len(creature.Path) == 0 {

		astar := AStar{}
		creature.Path = astar.GetPath(g.gameMap, creaturePosition, endPos, false)

	}

	creature.UpdatePosition(g, creaturePosition)

}

func NoMoveAction(g *Game, mover *ecs.Entity) {

}

func GoToEntityMoveAction(g *Game, mover *ecs.Entity) {

	creature := GetComponentType[*Creature](mover, CreatureComponent)
	creaturePosition := GetPosition(mover)
	goToEnt := GetComponentType[*EntityFollow](mover, entityFollowComponent)

	if goToEnt.target != nil {
		targetPos := GetComponentType[*Position](goToEnt.target, PositionComponent)

		creature.Path = creaturePosition.BuildPath(g, targetPos)

	}

	creature.UpdatePosition(g, creaturePosition)

}

// Sort of works but needs improvement
func StayWithinRangeMoveAction(g *Game, mover *ecs.Entity) {

	creature := GetComponentType[*Creature](mover, CreatureComponent)
	creaturePosition := GetPosition(mover)
	withinRange := GetComponentType[*StayWithinRange](mover, stayWithinRangeComponent)

	if withinRange.target != nil {
		targetPos := GetComponentType[*Position](withinRange.target, PositionComponent)

		pixelX, pixelY := PixelsFromPosition(targetPos)
		distance := withinRange.distance

		var path []Position
		for distance >= 0 {
			indices := NewTileCircleOutline(pixelX, pixelY, distance).GetIndices()

			ind, ok := GetUnblockedTile(&g.gameMap, indices)
			if ok {

				finalPos := PositionFromIndex(ind)
				path = creaturePosition.BuildPath(g, &finalPos)
				break
			}
			// Decrease distance and try again
			distance--
		}

		creature.Path = path
		creature.UpdatePosition(g, creaturePosition)
		fmt.Println("Printing path ", creature.Path)
	}

	//fmt.Println(pos)

}
func GetUnblockedTile(gameMap *GameMap, indices []int) (int, bool) {

	unblocked_tiles := make([]int, 0)
	for _, ind := range indices {

		if !gameMap.Tiles[ind].Blocked {
			unblocked_tiles = append(unblocked_tiles, ind)
		}
	}

	if len(unblocked_tiles) == 0 {
		return -1, false
	}

	random_tile := GetRandomBetween(0, len(unblocked_tiles)-1)
	return unblocked_tiles[random_tile], true

}

// Also needs improvement
func FleeFromEntityMovementAction(g *Game, mover *ecs.Entity) {
	fleeMov := GetComponentType[*FleeMovement](mover, fleeComponent)
	creature := GetComponentType[*Creature](mover, CreatureComponent)
	creaturePosition := GetPosition(mover)

	if fleeMov.target != nil {
		targetPosition := GetComponentType[*Position](fleeMov.target, PositionComponent)

		fleeVectorX := creaturePosition.X - targetPosition.X
		fleeVectorY := creaturePosition.Y - targetPosition.Y
		vectorLength := math.Sqrt(float64(fleeVectorX*fleeVectorX + fleeVectorY*fleeVectorY))

		normalizedX := float64(fleeVectorX) / vectorLength
		normalizedY := float64(fleeVectorY) / vectorLength

		// Try up to 3 times (initial + 3 retries)
		for attempt := 0; attempt < 3; attempt++ {
			// Scale the flee vector by fleeDistance and try different directions
			angleOffset := float64(attempt) * (math.Pi / 8.0) // 8 possible directions (22.5-degree steps)
			fleeTargetX := creaturePosition.X + int(normalizedX*float64(fleeMov.distance)*math.Cos(angleOffset)) - int(normalizedY*float64(fleeMov.distance)*math.Sin(angleOffset))
			fleeTargetY := creaturePosition.Y + int(normalizedX*float64(fleeMov.distance)*math.Sin(angleOffset)) + int(normalizedY*float64(fleeMov.distance)*math.Cos(angleOffset))

			fleePosition := Position{X: fleeTargetX, Y: fleeTargetY}

			if InBounds(fleeTargetX, fleeTargetY) {
				targetIndex := IndexFromXY(fleeTargetX, fleeTargetY)
				if !g.gameMap.Tiles[targetIndex].Blocked {
					// Use InRange to check if the flee position is within the desired range
					if creaturePosition.InRange(&fleePosition, fleeMov.distance) {
						// Set the path to the flee destination
						path := creaturePosition.BuildPath(g, &fleePosition)
						creature.Path = path
						fmt.Println(path)
						creature.UpdatePosition(g, creaturePosition)
						return
					}
				}
			}

			// Adjust distance slightly for the next attempt if needed
			fleeMov.distance = max(fleeMov.distance-1, 1) // Ensure distance doesn't go below 1
		}

		// If no valid flee destination was found, you can add additional fallback logic here if necessary
	}
}

// Used for Stay Within Range movement. Selects a random unblocked tile to move to

// Gets called in MonsterSystems, which queries the ECS manager and returns query results containing all monsters

// Creature movement follows a path, which is a slice of Position Type. Each movement function calls
// UpdatePosition, which...updates the creatures position The movement type functions determine
// How a path is created
func MovementSystem(c *ecs.QueryResult, g *Game) {

	var ok bool

	if _, ok = c.Entity.GetComponentData(simpleWander); ok {
		SimpleWanderAction(g, c.Entity)
	}

	if _, ok = c.Entity.GetComponentData(noMove); ok {
		NoMoveAction(g, c.Entity)
	}

	if _, ok = c.Entity.GetComponentData(entityFollowComponent); ok {
		GoToEntityMoveAction(g, c.Entity)
	}

	if _, ok = c.Entity.GetComponentData(stayWithinRangeComponent); ok {
		StayWithinRangeMoveAction(g, c.Entity)
	}

	if _, ok = c.Entity.GetComponentData(fleeComponent); ok {
		FleeFromEntityMovementAction(g, c.Entity)
	}

}

func RemoveMovementComponent(c *ecs.QueryResult) {

	var ok bool

	if _, ok = c.Entity.GetComponentData(simpleWander); ok {
		c.Entity.RemoveComponent(simpleWander)

	}

	if _, ok = c.Entity.GetComponentData(noMove); ok {
		c.Entity.RemoveComponent(noMove)
	}

	if _, ok = c.Entity.GetComponentData(entityFollowComponent); ok {
		c.Entity.RemoveComponent(entityFollowComponent)
	}

	if _, ok = c.Entity.GetComponentData(stayWithinRangeComponent); ok {
		c.Entity.RemoveComponent(stayWithinRangeComponent)
	}

	if _, ok = c.Entity.GetComponentData(fleeComponent); ok {
		c.Entity.RemoveComponent(fleeComponent)
	}

}

func InitializeMovementComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	simpleWander = manager.NewComponent()
	noMove = manager.NewComponent()
	entityFollowComponent = manager.NewComponent()
	stayWithinRangeComponent = manager.NewComponent()
	fleeComponent = manager.NewComponent()

}
