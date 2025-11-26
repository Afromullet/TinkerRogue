package squads

import (
	"fmt"
	"game_main/common"
	"game_main/coords"

	"github.com/bytearena/ecs"
)

// SquadService encapsulates all squad game logic
type SquadService struct {
	entityManager *common.EntityManager
}

// NewSquadService creates a new squad service
func NewSquadService(manager *common.EntityManager) *SquadService {
	return &SquadService{
		entityManager: manager,
	}
}

// CreateSquadResult contains information about squad creation
type CreateSquadResult struct {
	Success   bool
	SquadID   ecs.EntityID
	SquadName string
	Error     string
}

// CreateSquad creates a new empty squad
func (ss *SquadService) CreateSquad(squadName string) *CreateSquadResult {
	result := &CreateSquadResult{
		SquadName: squadName,
	}

	if squadName == "" {
		result.Error = "squad name cannot be empty"
		return result
	}

	squadEntity := ss.entityManager.World.NewEntity()
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

// AddUnitResult contains information about unit addition
type AddUnitResult struct {
	Success           bool
	UnitID            ecs.EntityID
	UnitName          string
	Error             string
	RemainingCapacity float64
}

// AddUnitToSquad adds a unit to a squad at the specified grid position
func (ss *SquadService) AddUnitToSquad(
	squadID ecs.EntityID,
	unit UnitTemplate,
	gridRow, gridCol int,
) *AddUnitResult {
	result := &AddUnitResult{
		UnitName: unit.Name,
	}

	// Validate position
	if gridRow < 0 || gridRow > 2 || gridCol < 0 || gridCol > 2 {
		result.Error = fmt.Sprintf("invalid grid position (%d, %d)", gridRow, gridCol)
		return result
	}

	// Check if position occupied
	existingUnitIDs := GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, ss.entityManager)
	if len(existingUnitIDs) > 0 {
		result.Error = fmt.Sprintf("grid position (%d, %d) already occupied", gridRow, gridCol)
		return result
	}

	// Check capacity before adding unit
	unitCapacityCost := unit.Attributes.GetCapacityCost()
	if !CanAddUnitToSquad(squadID, unitCapacityCost, ss.entityManager) {
		remaining := GetSquadRemainingCapacity(squadID, ss.entityManager)
		result.RemainingCapacity = remaining
		result.Error = fmt.Sprintf("insufficient squad capacity: need %.2f, have %.2f remaining", unitCapacityCost, remaining)
		return result
	}

	// Create unit entity
	unitEntity, err := CreateUnitEntity(ss.entityManager, unit)
	if err != nil {
		result.Error = fmt.Sprintf("invalid unit %s: %v", unit.Name, err)
		return result
	}

	unitID := unitEntity.GetID()

	// Add SquadMemberComponent to link unit to squad
	unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{
		SquadID: squadID,
	})

	// Update GridPositionComponent with actual grid position
	gridPos := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)
	gridPos.AnchorRow = gridRow
	gridPos.AnchorCol = gridCol

	// Update squad capacity tracking
	UpdateSquadCapacity(squadID, ss.entityManager)

	result.Success = true
	result.UnitID = unitID
	result.RemainingCapacity = GetSquadRemainingCapacity(squadID, ss.entityManager)

	return result
}

// RemoveUnitResult contains information about unit removal
type RemoveUnitResult struct {
	Success           bool
	Error             string
	RemainingCapacity float64
	RemovedUnitCount  int
}

// RemoveUnitFromSquad removes a unit from a squad
func (ss *SquadService) RemoveUnitFromSquad(
	squadID ecs.EntityID,
	unitID ecs.EntityID,
) *RemoveUnitResult {
	result := &RemoveUnitResult{}

	// Find the unit entity
	unitEntity := common.FindEntityByIDWithTag(ss.entityManager, unitID, SquadMemberTag)
	if unitEntity == nil {
		result.Error = fmt.Sprintf("unit %d not found in squad", unitID)
		return result
	}

	// Verify it belongs to the specified squad
	squadMember := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
	if squadMember == nil || squadMember.SquadID != squadID {
		result.Error = fmt.Sprintf("unit %d does not belong to squad %d", unitID, squadID)
		return result
	}

	// Remove the unit
	ss.entityManager.World.DisposeEntities(unitEntity)

	// Update squad capacity
	UpdateSquadCapacity(squadID, ss.entityManager)

	result.Success = true
	result.RemainingCapacity = GetSquadRemainingCapacity(squadID, ss.entityManager)
	result.RemovedUnitCount = 1

	return result
}

// GetSquadInfoResult contains squad information
type GetSquadInfoResult struct {
	SquadID           ecs.EntityID
	SquadName         string
	TotalCapacity     float64
	UsedCapacity      float64
	RemainingCapacity float64
	UnitCount         int
	IsDestroyed       bool
}

// GetSquadInfo returns information about a squad
func (ss *SquadService) GetSquadInfo(squadID ecs.EntityID) *GetSquadInfoResult {
	result := &GetSquadInfoResult{
		SquadID: squadID,
	}

	squadData := common.GetComponentTypeByIDWithTag[*SquadData](ss.entityManager, squadID, SquadTag, SquadComponent)
	if squadData == nil {
		return result
	}

	result.SquadName = squadData.Name
	result.TotalCapacity = float64(squadData.TotalCapacity)
	result.UsedCapacity = squadData.UsedCapacity
	result.RemainingCapacity = float64(squadData.TotalCapacity) - squadData.UsedCapacity
	result.UnitCount = len(GetUnitIDsInSquad(squadID, ss.entityManager))
	result.IsDestroyed = IsSquadDestroyed(squadID, ss.entityManager)

	return result
}

// CanAddMoreUnits checks if more units can be added to a squad
func (ss *SquadService) CanAddMoreUnits(squadID ecs.EntityID, unitCapacityCost float64) bool {
	return CanAddUnitToSquad(squadID, unitCapacityCost, ss.entityManager)
}

// GetSquadRemainingCapacity returns the remaining capacity of a squad
func (ss *SquadService) GetSquadRemainingCapacity(squadID ecs.EntityID) float64 {
	return GetSquadRemainingCapacity(squadID, ss.entityManager)
}

// DestroySquadResult contains information about squad destruction
type DestroySquadResult struct {
	Success   bool
	SquadID   ecs.EntityID
	SquadName string
	Error     string
}

// DestroySquad removes a squad from play
func (ss *SquadService) DestroySquad(squadID ecs.EntityID) *DestroySquadResult {
	result := &DestroySquadResult{
		SquadID: squadID,
	}

	// Find and mark squad as destroyed
	squadEntity := common.FindEntityByIDWithTag(ss.entityManager, squadID, SquadTag)
	if squadEntity == nil {
		result.Error = fmt.Sprintf("squad %d not found", squadID)
		return result
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	if squadData != nil {
		result.SquadName = squadData.Name
		squadData.Morale = 0 // Mark as destroyed
	}

	result.Success = true
	return result
}

// ApplyDamageResult contains information about damage application
type ApplyDamageResult struct {
	Success        bool
	SquadName      string
	DamageApplied  int
	RemainingHP    int
	SquadDestroyed bool
	Error          string
}

// ApplyDamageToSquad applies damage to a squad (affects units and morale)
func (ss *SquadService) ApplyDamageToSquad(squadID ecs.EntityID, damageAmount int) *ApplyDamageResult {
	result := &ApplyDamageResult{
		DamageApplied: damageAmount,
	}

	squadEntity := common.FindEntityByIDWithTag(ss.entityManager, squadID, SquadTag)
	if squadEntity == nil {
		result.Error = fmt.Sprintf("squad %d not found", squadID)
		return result
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	if squadData == nil {
		result.Error = "squad has no squad data component"
		return result
	}

	result.SquadName = squadData.Name

	// Apply morale damage (morale represents squad integrity)
	moraleReduction := (damageAmount / 5) // Rough estimate
	if moraleReduction < 1 {
		moraleReduction = 1
	}
	squadData.Morale -= moraleReduction
	if squadData.Morale < 0 {
		squadData.Morale = 0
		result.SquadDestroyed = true
	}

	result.RemainingHP = squadData.Morale
	result.Success = true
	return result
}
