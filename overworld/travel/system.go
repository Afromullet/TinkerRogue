package travel

import (
	"fmt"
	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"
	"math"

	"github.com/bytearena/ecs"
)

// CreateTravelStateEntity creates the singleton travel state entity
func CreateTravelStateEntity(manager *common.EntityManager) ecs.EntityID {
	entity := manager.World.NewEntity()

	travelData := &core.TravelStateData{
		IsTraveling: false,
	}

	entity.AddComponent(core.TravelStateComponent, travelData)

	return entity.GetID()
}

// GetTravelState retrieves the singleton travel state
func GetTravelState(manager *common.EntityManager) *core.TravelStateData {
	results := manager.World.Query(core.TravelStateTag)
	if len(results) == 0 {
		return nil
	}

	entity := results[0].Entity
	return common.GetComponentType[*core.TravelStateData](entity, core.TravelStateComponent)
}

// IsTraveling checks if player is currently traveling
func IsTraveling(manager *common.EntityManager) bool {
	travelState := GetTravelState(manager)
	if travelState == nil {
		return false
	}
	return travelState.IsTraveling
}

// manhattanDistance returns the Manhattan distance between two positions
func manhattanDistance(from, to coords.LogicalPosition) int {
	dx := from.X - to.X
	if dx < 0 {
		dx = -dx
	}
	dy := from.Y - to.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// StartTravel initiates travel to a threat node
func StartTravel(
	manager *common.EntityManager,
	playerData *common.PlayerData,
	destinationPos coords.LogicalPosition,
	targetThreatID ecs.EntityID,
	encounterID ecs.EntityID,
) error {
	// 1. Validate not already traveling
	travelState := GetTravelState(manager)
	if travelState == nil {
		return fmt.Errorf("travel state not initialized")
	}

	if travelState.IsTraveling {
		return fmt.Errorf("already traveling - cannot start new travel")
	}

	// 2. Get player position
	playerEntity := manager.FindEntityByID(playerData.PlayerEntityID)
	if playerEntity == nil {
		return fmt.Errorf("player entity not found")
	}

	playerPos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)
	if playerPos == nil {
		return fmt.Errorf("player has no position component")
	}

	// 3. Get movement speed
	attributes := common.GetComponentType[*common.Attributes](playerEntity, common.AttributeComponent)
	movementSpeed := 1
	if attributes != nil && attributes.MovementSpeed > 0 {
		movementSpeed = attributes.MovementSpeed
	}

	// 4. Calculate ticks needed (Manhattan distance / movement speed, rounded up)
	distance := manhattanDistance(*playerPos, destinationPos)
	ticksNeeded := int(math.Ceil(float64(distance) / float64(movementSpeed)))

	// 5. Initialize travel state
	travelState.IsTraveling = true
	travelState.Origin = *playerPos
	travelState.Destination = destinationPos
	travelState.TicksRemaining = ticksNeeded
	travelState.TargetThreatID = targetThreatID
	travelState.TargetEncounterID = encounterID

	// 6. Log travel start
	fmt.Printf("Travel started: %d ticks from (%d,%d) to (%d,%d)\n",
		ticksNeeded, playerPos.X, playerPos.Y, destinationPos.X, destinationPos.Y)

	return nil
}

// AdvanceTravelTick moves player during travel (called by manager.AdvanceTick)
// Returns true if travel completed this tick
func AdvanceTravelTick(
	manager *common.EntityManager,
	playerData *common.PlayerData,
) (bool, error) {
	// 1. Check if traveling
	travelState := GetTravelState(manager)
	if travelState == nil || !travelState.IsTraveling {
		return false, nil
	}

	// 2. Decrement tick counter
	travelState.TicksRemaining--

	// 3. Check if arrived
	if travelState.TicksRemaining <= 0 {
		// Move player to exact destination
		playerEntity := manager.FindEntityByID(playerData.PlayerEntityID)
		if playerEntity == nil {
			return false, fmt.Errorf("player entity not found")
		}

		currentPos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)
		if currentPos != nil && (currentPos.X != travelState.Destination.X || currentPos.Y != travelState.Destination.Y) {
			manager.MoveEntity(playerData.PlayerEntityID, playerEntity, *currentPos, travelState.Destination)
		}

		fmt.Printf("Travel completed: arrived at (%d,%d)\n",
			travelState.Destination.X, travelState.Destination.Y)

		travelState.IsTraveling = false
		travelState.TicksRemaining = 0
		return true, nil
	}

	// 4. Log progress
	fmt.Printf("Travel progress: %d ticks remaining\n", travelState.TicksRemaining)

	return false, nil
}

// CancelTravel stops active travel, returns to origin
func CancelTravel(
	manager *common.EntityManager,
	playerData *common.PlayerData,
) error {
	// 1. Check if traveling
	travelState := GetTravelState(manager)
	if travelState == nil || !travelState.IsTraveling {
		return nil // Not traveling, nothing to cancel
	}

	// 2. Get player entity
	playerEntity := manager.FindEntityByID(playerData.PlayerEntityID)
	if playerEntity == nil {
		return fmt.Errorf("player entity not found")
	}

	// 3. Dispose encounter entity
	encounterEntity := manager.FindEntityByID(travelState.TargetEncounterID)
	if encounterEntity != nil {
		// Check if encounter has position (for proper cleanup)
		encounterPos := common.GetComponentType[*coords.LogicalPosition](encounterEntity, common.PositionComponent)
		if encounterPos != nil {
			manager.CleanDisposeEntity(encounterEntity, encounterPos)
		} else {
			manager.World.DisposeEntities(encounterEntity)
		}
	}

	// 4. Reset player position to origin
	currentPos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)
	if currentPos != nil {
		manager.MoveEntity(playerData.PlayerEntityID, playerEntity, *currentPos, travelState.Origin)
	}

	// 5. Clear travel state
	fmt.Printf("Travel cancelled: returned to origin (%d,%d)\n",
		travelState.Origin.X, travelState.Origin.Y)
	travelState.IsTraveling = false
	travelState.TicksRemaining = 0

	return nil
}
