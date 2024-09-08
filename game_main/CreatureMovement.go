package main

import (
	"game_main/ecshelper"
	"game_main/graphics"
	"game_main/worldmap"
	"math"

	"github.com/bytearena/ecs"
)

/*
To create a new Movement Type do the following:

1) Create the movement component. See where simpleWanderComp, noMoveComp and the rest are declare.d
2) Created the associated struct as we do for the ECS library we're using.
3) Create the function that handles the movement. The function updates the path of the entity that's moving
4) In InitializeMovementComponents, initialize the component created in step 1
5) In InitializeMovementComponents, append the component to MovementTypes
6) Add the function to the MovementActions map

*/
// movementAction is a map that will make it easier to call the movement functions.
// so that we no not have to add a conditional for every movement type in the MovementSystem
type MovementFunction func(g *Game, mover *ecs.Entity)

var (
	simpleWanderComp     *ecs.Component
	noMoveComp           *ecs.Component
	entityFollowComp     *ecs.Component
	withinRadiusComp     *ecs.Component
	withinRangeComponent *ecs.Component
	fleeComp             *ecs.Component

	MovementTypes []*ecs.Component

	MovementActions = map[*ecs.Component]MovementFunction{}
)

// Todo considering adding an interface with "BuildPath" m

type SimpleWander struct {
}

type NoMovement struct {
}

type EntityFollow struct {
	target *ecs.Entity
}

// Todo need a better name for this. This is the kind of movement that does something in relation to the
// Target Entity, such as staying within a radius, fleeing from it, etc.
// Anything that uses DistanceToEntityMovement determines its movement in relation ot the target
type DistanceToEntityMovement struct {
	target   *ecs.Entity
	distance int
}

// Each Movement function implementation choses how to build a path.
// They also handle walking on the path by calling reature.UpdatePosition
// Movement functions get called by MovementSystem which determines what movement component a creature has.
// Select a random spot to wander to and builds a new path when arriving at the position
func SimpleWanderAction(g *Game, mover *ecs.Entity) {

	creature := GetCreature(mover)
	creaturePosition := ecshelper.GetPosition(mover)

	randomPos := GetRandomBetween(0, len(worldmap.ValidPos.Pos))
	endPos := worldmap.ValidPos.Get(randomPos)

	//Only create a new path if one doesn't exist yet.
	if len(creature.Path) == 0 {

		astar := AStar{}
		creature.Path = astar.GetPath(g.gameMap, creaturePosition, endPos, false)

	}

}

func NoMoveAction(g *Game, mover *ecs.Entity) {

}

func EntityFollowMoveAction(g *Game, mover *ecs.Entity) {

	creature := GetCreature(mover)
	creaturePosition := ecshelper.GetPosition(mover)
	goToEnt := ecshelper.GetComponentType[*EntityFollow](mover, entityFollowComp)

	if goToEnt.target != nil {
		targetPos := ecshelper.GetComponentType[*ecshelper.Position](goToEnt.target, ecshelper.PositionComponent)

		creature.Path = BuildPath(g, creaturePosition, targetPos)

	}

}

// Sort of works but needs improvement
func WithinRadiusMoveAction(g *Game, mover *ecs.Entity) {

	gd := graphics.NewScreenData()
	creature := GetCreature(mover)
	creaturePosition := ecshelper.GetPosition(mover)
	withinRange := ecshelper.GetComponentType[*DistanceToEntityMovement](mover, withinRadiusComp)

	if withinRange.target != nil {
		targetPos := ecshelper.GetComponentType[*ecshelper.Position](withinRange.target, ecshelper.PositionComponent)

		pixelX, pixelY := ecshelper.PixelsFromPosition(targetPos, gd.TileWidth, gd.TileHeight)
		distance := withinRange.distance

		var path []ecshelper.Position
		for distance >= 0 {
			indices := graphics.NewTileCircleOutline(pixelX, pixelY, distance).GetIndices()

			ind, ok := GetUnblockedTile(&g.gameMap, indices)
			if ok {

				finalPos := ecshelper.PositionFromIndex(ind, gd.ScreenWidth, gd.ScreenHeight)
				path = BuildPath(g, creaturePosition, &finalPos)
				break
			}
			// Decrease distance and try again
			distance--
		}

		creature.Path = path

	}

	//fmt.Println(pos)

}
func GetUnblockedTile(gameMap *worldmap.GameMap, indices []int) (int, bool) {

	unblocked_tiles := make([]int, 0)
	for i, ind := range indices {

		if i < len(indices) {
			if !gameMap.Tiles[ind].Blocked {
				unblocked_tiles = append(unblocked_tiles, ind)
			}
		}
	}

	if len(unblocked_tiles) == 0 {
		return -1, false
	}

	random_tile := GetRandomBetween(0, len(unblocked_tiles)-1)
	return unblocked_tiles[random_tile], true

}

// Clears the path once within range
func WithinRangeMoveAction(g *Game, mover *ecs.Entity) {

	creature := GetCreature(mover)
	creaturePosition := ecshelper.GetPosition(mover)
	within := ecshelper.GetComponentType[*DistanceToEntityMovement](mover, withinRangeComponent)

	if within.target != nil {
		targetPos := ecshelper.GetComponentType[*ecshelper.Position](within.target, ecshelper.PositionComponent)

		if targetPos.InRange(creaturePosition, within.distance) {
			creature.Path = creature.Path[:0]

		} else {
			creature.Path = BuildPath(g, creaturePosition, targetPos)

		}

	}

}

// Also needs improvement
func FleeFromEntityMovementAction(g *Game, mover *ecs.Entity) {
	fleeMov := ecshelper.GetComponentType[*DistanceToEntityMovement](mover, fleeComp)
	creature := GetCreature(mover)
	creaturePosition := ecshelper.GetPosition(mover)

	if fleeMov.target != nil {
		targetPosition := ecshelper.GetComponentType[*ecshelper.Position](fleeMov.target, ecshelper.PositionComponent)

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

			fleePosition := ecshelper.Position{X: fleeTargetX, Y: fleeTargetY}

			if graphics.InBounds(fleeTargetX, fleeTargetY) {
				targetIndex := graphics.IndexFromXY(fleeTargetX, fleeTargetY)
				if !g.gameMap.Tiles[targetIndex].Blocked {
					// Use InRange to check if the flee position is within the desired range
					if creaturePosition.InRange(&fleePosition, fleeMov.distance) {
						// Set the path to the flee destination
						path := BuildPath(g, creaturePosition, &fleePosition)
						creature.Path = path

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
// A movmeent function builds a path for a creature to follow, and UpdatePosition lets a creature move on the path

func MovementSystem(c *ecs.QueryResult, g *Game) {

	//var ok bool

	for _, comp := range MovementTypes {

		if c.Entity.HasComponent(comp) {

			creature := ecshelper.GetComponentType[*Creature](c.Entity, CreatureComponent)
			creaturePosition := ecshelper.GetComponentType[*ecshelper.Position](c.Entity, ecshelper.PositionComponent)
			if movementFunc, exists := MovementActions[comp]; exists {
				movementFunc(g, c.Entity) // Call the function
				creature.UpdatePosition(g, creaturePosition)
			}
		}

	}

}

func RemoveMovementComponent(c *ecs.QueryResult) {

	var ok bool

	for _, m := range MovementTypes {
		if _, ok = c.Entity.GetComponentData(m); ok {
			c.Entity.RemoveComponent(m)

		}

	}

}

func InitializeMovementComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	simpleWanderComp = manager.NewComponent()
	noMoveComp = manager.NewComponent()
	entityFollowComp = manager.NewComponent()
	withinRadiusComp = manager.NewComponent()
	withinRangeComponent = manager.NewComponent()
	fleeComp = manager.NewComponent()

	MovementTypes = append(MovementTypes, simpleWanderComp, noMoveComp, entityFollowComp, withinRadiusComp, withinRangeComponent, fleeComp)

	MovementActions[simpleWanderComp] = SimpleWanderAction
	MovementActions[noMoveComp] = NoMoveAction
	MovementActions[entityFollowComp] = EntityFollowMoveAction
	MovementActions[withinRadiusComp] = WithinRadiusMoveAction
	MovementActions[withinRangeComponent] = WithinRangeMoveAction
	MovementActions[fleeComp] = FleeFromEntityMovementAction

}
