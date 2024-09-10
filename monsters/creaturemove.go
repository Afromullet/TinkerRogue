package monsters

import (
	"game_main/common"
	"game_main/graphics"
	"game_main/pathfinding"
	"game_main/randgen"
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
type MovementFunction func(ecsmanager *common.EntityManager, gm *worldmap.GameMap, mover *ecs.Entity)

var (
	SimpleWanderComp     *ecs.Component
	NoMoveComp           *ecs.Component
	EntityFollowComp     *ecs.Component
	WithinRadiusComp     *ecs.Component
	WithinRangeComponent *ecs.Component
	FleeComp             *ecs.Component

	MovementTypes []*ecs.Component

	MovementActions = map[*ecs.Component]MovementFunction{}
)

// Todo considering adding an interface with "BuildPath" m

type SimpleWander struct {
}

type NoMovement struct {
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
// They also handle walking on the path by calling reature.UpdatePosition
// Movement functions get called by MovementSystem which determines what movement component a creature has.
// Select a random spot to wander to and builds a new path when arriving at the position
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

}

// Sort of works but needs improvement
func WithinRadiusMoveAction(ecsmanager *common.EntityManager, gm *worldmap.GameMap, mover *ecs.Entity) {

	gd := graphics.NewScreenData()
	creature := GetCreature(mover)
	creaturePosition := common.GetPosition(mover)
	withinRange := common.GetComponentType[*DistanceToEntityMovement](mover, WithinRadiusComp)

	if withinRange.Target != nil {
		targetPos := common.GetComponentType[*common.Position](withinRange.Target, common.PositionComponent)

		pixelX, pixelY := common.PixelsFromPosition(targetPos, gd.TileWidth, gd.TileHeight)
		distance := withinRange.Distance

		var path []common.Position
		for distance >= 0 {
			indices := graphics.NewTileCircleOutline(pixelX, pixelY, distance).GetIndices()

			ind, ok := GetUnblockedTile(gm, indices)
			if ok {

				finalPos := common.PositionFromIndex(ind, gd.ScreenWidth)
				path = pathfinding.BuildPath(gm, creaturePosition, &finalPos)
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

	random_tile := randgen.GetRandomBetween(0, len(unblocked_tiles)-1)
	return unblocked_tiles[random_tile], true

}

// Clears the path once within range
func WithinRangeMoveAction(ecsmanager *common.EntityManager, gm *worldmap.GameMap, mover *ecs.Entity) {

	creature := GetCreature(mover)
	creaturePosition := common.GetPosition(mover)
	within := common.GetComponentType[*DistanceToEntityMovement](mover, WithinRangeComponent)

	if within.Target != nil {
		targetPos := common.GetComponentType[*common.Position](within.Target, common.PositionComponent)

		if targetPos.InRange(creaturePosition, within.Distance) {
			creature.Path = creature.Path[:0]

		} else {
			creature.Path = pathfinding.BuildPath(gm, creaturePosition, targetPos)

		}

	}

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
				targetIndex := graphics.IndexFromXY(fleeTargetX, fleeTargetY)
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
}

// Used for Stay Within Range movement. Selects a random unblocked tile to move to
// Gets called in MonsterSystems, which queries the ECS manager and returns query results containing all monsters
// A movmeent function builds a path for a creature to follow, and UpdatePosition lets a creature move on the path

func CreatureMovementSystem(ecsmanager *common.EntityManager, gm *worldmap.GameMap, c *ecs.QueryResult) MovementFunction {

	//var ok bool

	for _, comp := range MovementTypes {

		if c.Entity.HasComponent(comp) {

			creature := common.GetComponentType[*Creature](c.Entity, CreatureComponent)
			creaturePosition := common.GetComponentType[*common.Position](c.Entity, common.PositionComponent)
			if movementFunc, exists := MovementActions[comp]; exists {
				movementFunc(ecsmanager, gm, c.Entity) // Call the function
				creature.UpdatePosition(gm, creaturePosition)
				return movementFunc
			}
		}

	}

	return nil

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

	SimpleWanderComp = manager.NewComponent()
	NoMoveComp = manager.NewComponent()
	EntityFollowComp = manager.NewComponent()
	WithinRadiusComp = manager.NewComponent()
	WithinRangeComponent = manager.NewComponent()
	FleeComp = manager.NewComponent()

	MovementTypes = append(MovementTypes, SimpleWanderComp, NoMoveComp, EntityFollowComp, WithinRadiusComp, WithinRangeComponent, FleeComp)

	MovementActions[SimpleWanderComp] = SimpleWanderAction
	MovementActions[NoMoveComp] = NoMoveAction
	MovementActions[EntityFollowComp] = EntityFollowMoveAction
	MovementActions[WithinRadiusComp] = WithinRadiusMoveAction
	MovementActions[WithinRangeComponent] = WithinRangeMoveAction
	MovementActions[FleeComp] = FleeFromEntityMovementAction

}
