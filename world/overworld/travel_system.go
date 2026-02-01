package overworld

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"
	"math"

	"github.com/bytearena/ecs"
)

// CreateTravelStateEntity creates the singleton travel state entity
func CreateTravelStateEntity(manager *common.EntityManager) ecs.EntityID {
	entity := manager.World.NewEntity()

	travelData := &TravelStateData{
		IsTraveling: false,
	}

	entity.AddComponent(TravelStateComponent, travelData)

	return entity.GetID()
}

// GetTravelState retrieves the singleton travel state
func GetTravelState(manager *common.EntityManager) *TravelStateData {
	results := manager.World.Query(TravelStateTag)
	if len(results) == 0 {
		return nil
	}

	entity := results[0].Entity
	return common.GetComponentType[*TravelStateData](entity, TravelStateComponent)
}

// IsTraveling checks if player is currently traveling
func IsTraveling(manager *common.EntityManager) bool {
	travelState := GetTravelState(manager)
	if travelState == nil {
		return false
	}
	return travelState.IsTraveling
}

// calculateEuclideanDistance calculates straight-line distance between two points
func calculateEuclideanDistance(from, to coords.LogicalPosition) float64 {
	dx := float64(to.X - from.X)
	dy := float64(to.Y - from.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

// lerpPosition linearly interpolates between two positions based on progress (0.0 to 1.0)
func lerpPosition(from, to coords.LogicalPosition, progress float64) coords.LogicalPosition {
	// Clamp progress
	if progress < 0.0 {
		progress = 0.0
	}
	if progress > 1.0 {
		progress = 1.0
	}

	x := float64(from.X) + float64(to.X-from.X)*progress
	y := float64(from.Y) + float64(to.Y-from.Y)*progress

	return coords.LogicalPosition{
		X: int(math.Round(x)),
		Y: int(math.Round(y)),
	}
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

	// 3. Calculate Euclidean distance
	distance := calculateEuclideanDistance(*playerPos, destinationPos)

	// 4. Initialize travel state
	travelState.IsTraveling = true
	travelState.Origin = *playerPos
	travelState.Destination = destinationPos
	travelState.TotalDistance = distance
	travelState.RemainingDistance = distance
	travelState.TargetThreatID = targetThreatID
	travelState.TargetEncounterID = encounterID

	// 5. Log travel start
	fmt.Printf("Travel started: distance %.2f from (%d,%d) to (%d,%d)\n",
		distance, playerPos.X, playerPos.Y, destinationPos.X, destinationPos.Y)

	return nil
}

// AdvanceTravelTick moves player during travel (called by AdvanceTick)
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

	// 2. Get player entity and attributes
	playerEntity := manager.FindEntityByID(playerData.PlayerEntityID)
	if playerEntity == nil {
		return false, fmt.Errorf("player entity not found")
	}

	attributes := common.GetComponentType[*common.Attributes](playerEntity, common.AttributeComponent)
	if attributes == nil {
		return false, fmt.Errorf("player has no attributes component")
	}

	movementSpeed := float64(attributes.MovementSpeed)

	// 3. Reduce remaining distance
	travelState.RemainingDistance -= movementSpeed

	// 4. Update player position based on progress
	var newPos coords.LogicalPosition
	if travelState.RemainingDistance <= 0 {
		// Arrived - set exact destination
		newPos = travelState.Destination
	} else {
		// Still traveling - interpolate position
		progress := (travelState.TotalDistance - travelState.RemainingDistance) / travelState.TotalDistance
		newPos = lerpPosition(travelState.Origin, travelState.Destination, progress)
	}

	// Update player position if it changed
	currentPos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)
	if currentPos != nil && (currentPos.X != newPos.X || currentPos.Y != newPos.Y) {
		manager.MoveEntity(playerData.PlayerEntityID, playerEntity, *currentPos, newPos)
	}

	// 5. Check if arrived
	if travelState.RemainingDistance <= 0 {
		fmt.Printf("Travel completed: arrived at (%d,%d)\n", newPos.X, newPos.Y)
		// Clear travel state
		travelState.IsTraveling = false
		travelState.RemainingDistance = 0
		return true, nil
	}

	// 6. Log progress
	fmt.Printf("Travel progress: %.2f remaining (moved at speed %.1f)\n",
		travelState.RemainingDistance, movementSpeed)

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
	travelState.RemainingDistance = 0

	return nil
}
