package squads

import (
	"fmt"
	"game_main/common"
	"game_main/coords"

	"github.com/bytearena/ecs"
)

// SquadBuilderService encapsulates squad building game logic
type SquadBuilderService struct {
	entityManager *common.EntityManager
}

// NewSquadBuilderService creates a new squad builder service
func NewSquadBuilderService(manager *common.EntityManager) *SquadBuilderService {
	return &SquadBuilderService{
		entityManager: manager,
	}
}

// CreateSquadResult contains information about squad creation
type SquadBuilderSquadResult struct {
	Success   bool
	SquadID   ecs.EntityID
	SquadName string
	Error     string
}

// CreateSquad creates a new empty squad for building
func (sbs *SquadBuilderService) CreateSquad(squadName string) *SquadBuilderSquadResult {
	result := &SquadBuilderSquadResult{
		SquadName: squadName,
	}

	if squadName == "" {
		squadName = "New Squad"
	}

	squadEntity := sbs.entityManager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(SquadComponent, &SquadData{
		SquadID:       squadID,
		Name:          squadName,
		Morale:        100,
		TurnCount:     0,
		MaxUnits:      9,
		UsedCapacity:  0.0,
		TotalCapacity: 6, // Default capacity (no leader yet)
	})

	squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})

	result.Success = true
	result.SquadID = squadID

	return result
}

// PlaceUnitResult contains information about unit placement
type PlaceUnitResult struct {
	Success           bool
	UnitID            ecs.EntityID
	UnitName          string
	Error             string
	RemainingCapacity float64
}

// PlaceUnit places a unit from roster into a squad at grid position
func (sbs *SquadBuilderService) PlaceUnit(
	squadID ecs.EntityID,
	rosterUnitID ecs.EntityID,
	unit UnitTemplate,
	gridRow, gridCol int,
) *PlaceUnitResult {
	result := &PlaceUnitResult{
		UnitName: unit.Name,
	}

	// Validate position
	if gridRow < 0 || gridRow > 2 || gridCol < 0 || gridCol > 2 {
		result.Error = fmt.Sprintf("invalid grid position (%d, %d)", gridRow, gridCol)
		return result
	}

	// Check if position occupied
	existingUnitIDs := GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, sbs.entityManager)
	if len(existingUnitIDs) > 0 {
		result.Error = fmt.Sprintf("grid position (%d, %d) already occupied", gridRow, gridCol)
		return result
	}

	// Check capacity
	unitCapacityCost := unit.Attributes.GetCapacityCost()
	if !CanAddUnitToSquad(squadID, unitCapacityCost, sbs.entityManager) {
		remaining := GetSquadRemainingCapacity(squadID, sbs.entityManager)
		result.RemainingCapacity = remaining
		result.Error = fmt.Sprintf("insufficient squad capacity: need %.2f, have %.2f remaining", unitCapacityCost, remaining)
		return result
	}

	// Create unit entity
	unitEntity, err := CreateUnitEntity(sbs.entityManager, unit)
	if err != nil {
		result.Error = fmt.Sprintf("invalid unit %s: %v", unit.Name, err)
		return result
	}

	unitID := unitEntity.GetID()

	// Add SquadMemberComponent to link unit to squad
	unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{
		SquadID: squadID,
	})

	// Update GridPositionComponent with grid position
	gridPos := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)
	gridPos.AnchorRow = gridRow
	gridPos.AnchorCol = gridCol

	// Update squad capacity
	UpdateSquadCapacity(squadID, sbs.entityManager)

	result.Success = true
	result.UnitID = unitID
	result.RemainingCapacity = GetSquadRemainingCapacity(squadID, sbs.entityManager)

	return result
}

// RemoveUnitResult contains information about unit removal
type RemoveUnitFromGridResult struct {
	Success           bool
	Error             string
	RemainingCapacity float64
}

// RemoveUnitFromGrid removes a unit from squad at grid position
func (sbs *SquadBuilderService) RemoveUnitFromGrid(
	squadID ecs.EntityID,
	gridRow, gridCol int,
) *RemoveUnitFromGridResult {
	result := &RemoveUnitFromGridResult{}

	// Get unit at position
	unitIDs := GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, sbs.entityManager)
	if len(unitIDs) == 0 {
		result.Error = fmt.Sprintf("no unit at position (%d, %d)", gridRow, gridCol)
		return result
	}

	// Remove first unit at this position
	unitID := unitIDs[0]
	unitEntity := common.FindEntityByIDWithTag(sbs.entityManager, unitID, SquadMemberTag)
	if unitEntity == nil {
		result.Error = fmt.Sprintf("unit %d not found", unitID)
		return result
	}

	sbs.entityManager.World.DisposeEntities(unitEntity)

	// Update squad capacity
	UpdateSquadCapacity(squadID, sbs.entityManager)

	result.Success = true
	result.RemainingCapacity = GetSquadRemainingCapacity(squadID, sbs.entityManager)

	return result
}

// DesignateLeaderResult contains information about leader designation
type DesignateLeaderResult struct {
	Success bool
	Error   string
}

// DesignateLeader designates a unit as squad leader
func (sbs *SquadBuilderService) DesignateLeader(unitID ecs.EntityID) *DesignateLeaderResult {
	result := &DesignateLeaderResult{}

	// Find the unit entity
	unitEntity := common.FindEntityByIDWithTag(sbs.entityManager, unitID, SquadMemberTag)
	if unitEntity == nil {
		result.Error = fmt.Sprintf("unit %d not found", unitID)
		return result
	}

	// Add LeaderComponent
	unitEntity.AddComponent(LeaderComponent, &LeaderData{})

	result.Success = true
	return result
}

// GetSquadCapacityInfo returns capacity information for a squad
type SquadCapacityInfo struct {
	UsedCapacity      float64
	TotalCapacity     int
	RemainingCapacity float64
	HasLeader         bool
}

// GetCapacityInfo returns capacity information for the squad
func (sbs *SquadBuilderService) GetCapacityInfo(squadID ecs.EntityID) *SquadCapacityInfo {
	info := &SquadCapacityInfo{}

	squadData := common.GetComponentTypeByIDWithTag[*SquadData](sbs.entityManager, squadID, SquadTag, SquadComponent)
	if squadData == nil {
		return info
	}

	info.UsedCapacity = squadData.UsedCapacity
	info.TotalCapacity = squadData.TotalCapacity
	info.RemainingCapacity = float64(squadData.TotalCapacity) - squadData.UsedCapacity

	// Check for leader
	unitIDs := GetUnitIDsInSquad(squadID, sbs.entityManager)
	for _, unitID := range unitIDs {
		if sbs.entityManager.HasComponentByIDWithTag(unitID, SquadMemberTag, LeaderComponent) {
			info.HasLeader = true
			break
		}
	}

	return info
}

// ValidateSquadForCreation checks if squad is valid for final creation
type ValidateSquadResult struct {
	Valid     bool
	ErrorMsg  string
	UnitCount int
	HasLeader bool
}

// ValidateSquad validates that a squad is ready for final creation
func (sbs *SquadBuilderService) ValidateSquad(squadID ecs.EntityID) *ValidateSquadResult {
	result := &ValidateSquadResult{}

	unitIDs := GetUnitIDsInSquad(squadID, sbs.entityManager)
	result.UnitCount = len(unitIDs)

	if result.UnitCount == 0 {
		result.ErrorMsg = "Squad must have at least one unit"
		return result
	}

	// Check for leader
	for _, unitID := range unitIDs {
		if sbs.entityManager.HasComponentByIDWithTag(unitID, SquadMemberTag, LeaderComponent) {
			result.HasLeader = true
			break
		}
	}

	if !result.HasLeader {
		result.ErrorMsg = "Squad must have a designated leader"
		return result
	}

	// Check squad name
	squadData := common.GetComponentTypeByIDWithTag[*SquadData](sbs.entityManager, squadID, SquadTag, SquadComponent)
	if squadData == nil || squadData.Name == "" {
		result.ErrorMsg = "Squad must have a name"
		return result
	}

	result.Valid = true
	return result
}

// UpdateSquadName updates the name of a squad
func (sbs *SquadBuilderService) UpdateSquadName(squadID ecs.EntityID, newName string) bool {
	if newName == "" {
		return false
	}

	squadData := common.GetComponentTypeByIDWithTag[*SquadData](sbs.entityManager, squadID, SquadTag, SquadComponent)
	if squadData == nil {
		return false
	}

	squadData.Name = newName
	return true
}

// FinalizeSquadResult contains information about squad finalization
type FinalizeSquadResult struct {
	Success   bool
	SquadID   ecs.EntityID
	SquadName string
	UnitCount int
	Error     string
}

// FinalizeSquad validates and finalizes a squad, making it ready for deployment/combat
func (sbs *SquadBuilderService) FinalizeSquad(squadID ecs.EntityID) *FinalizeSquadResult {
	result := &FinalizeSquadResult{
		SquadID: squadID,
	}

	// Validate the squad first
	validation := sbs.ValidateSquad(squadID)
	if !validation.Valid {
		result.Error = validation.ErrorMsg
		return result
	}

	// Get squad data
	squadData := common.GetComponentTypeByIDWithTag[*SquadData](sbs.entityManager, squadID, SquadTag, SquadComponent)
	if squadData == nil {
		result.Error = "squad not found"
		return result
	}

	result.SquadName = squadData.Name
	result.UnitCount = validation.UnitCount
	result.Success = true

	return result
}
