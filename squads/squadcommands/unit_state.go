package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// CapturedUnitState represents complete unit state for undo operations
// Consolidates savedUnitState and mergedUnitState from various commands
type CapturedUnitState struct {
	UnitID       ecs.EntityID
	Template     squads.UnitTemplate
	GridRow      int
	GridCol      int
	IsLeader     bool
	Attributes   *common.Attributes
	Name         string
	GridPosition *squads.GridPositionData
}

// CaptureUnitState captures complete state of a single unit for undo operations
// Returns error if unit doesn't exist or is missing critical components
func CaptureUnitState(unitID ecs.EntityID, manager *common.EntityManager) (*CapturedUnitState, error) {
	unitEntity := common.FindEntityByIDWithTag(manager, unitID, squads.SquadMemberTag)
	if unitEntity == nil {
		return nil, fmt.Errorf("unit %d not found", unitID)
	}

	state := &CapturedUnitState{
		UnitID: unitID,
	}

	// Capture attributes
	if unitEntity.HasComponent(common.AttributeComponent) {
		attr := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
		if attr != nil {
			state.Attributes = copyAttributes(attr)
		}
	}

	// Capture name
	if unitEntity.HasComponent(common.NameComponent) {
		name := common.GetComponentType[*common.Name](unitEntity, common.NameComponent)
		if name != nil {
			state.Name = name.NameStr
			state.Template.Name = name.NameStr
		}
	}

	// Capture grid position
	if unitEntity.HasComponent(squads.GridPositionComponent) {
		gridPos := common.GetComponentType[*squads.GridPositionData](unitEntity, squads.GridPositionComponent)
		if gridPos != nil {
			state.GridRow = gridPos.AnchorRow
			state.GridCol = gridPos.AnchorCol
			state.GridPosition = copyGridPosition(gridPos)
		}
	}

	// Check if leader
	state.IsLeader = unitEntity.HasComponent(squads.LeaderComponent)

	// Capture unit role
	if unitEntity.HasComponent(squads.UnitRoleComponent) {
		roleData := common.GetComponentType[*squads.UnitRoleData](unitEntity, squads.UnitRoleComponent)
		if roleData != nil {
			state.Template.Role = roleData.Role
		}
	}

	// Build template from captured data
	if state.Attributes != nil {
		state.Template.Attributes = *state.Attributes
	}
	state.Template.GridRow = state.GridRow
	state.Template.GridCol = state.GridCol
	if state.GridPosition != nil {
		state.Template.GridWidth = state.GridPosition.Width
		state.Template.GridHeight = state.GridPosition.Height
	}

	return state, nil
}

// CaptureAllUnitsInSquad captures state of all units in a squad
// Returns slice of captured states and error if squad doesn't exist
func CaptureAllUnitsInSquad(squadID ecs.EntityID, manager *common.EntityManager) ([]CapturedUnitState, error) {
	// Verify squad exists
	squadEntity := squads.GetSquadEntity(squadID, manager)
	if squadEntity == nil {
		return nil, fmt.Errorf("squad %d not found", squadID)
	}

	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	capturedUnits := make([]CapturedUnitState, 0, len(unitIDs))

	for _, unitID := range unitIDs {
		state, err := CaptureUnitState(unitID, manager)
		if err != nil {
			// Skip units that fail to capture (shouldn't happen in normal flow)
			continue
		}
		capturedUnits = append(capturedUnits, *state)
	}

	return capturedUnits, nil
}

// RestoreUnitToSquad recreates a unit in a squad from captured state
// Returns the new unit's EntityID or error if restoration fails
func RestoreUnitToSquad(state *CapturedUnitState, squadID ecs.EntityID, manager *common.EntityManager) (ecs.EntityID, error) {
	// Create unit entity from template
	unitEntity, err := squads.CreateUnitEntity(manager, state.Template)
	if err != nil {
		return 0, fmt.Errorf("failed to create unit entity: %w", err)
	}

	newUnitID := unitEntity.GetID()

	// Add squad member component
	unitEntity.AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
		SquadID: squadID,
	})

	// Restore grid position
	gridPos := common.GetComponentType[*squads.GridPositionData](unitEntity, squads.GridPositionComponent)
	if gridPos != nil && state.GridPosition != nil {
		gridPos.AnchorRow = state.GridRow
		gridPos.AnchorCol = state.GridCol
		gridPos.Width = state.GridPosition.Width
		gridPos.Height = state.GridPosition.Height
	}

	// Restore leader status
	if state.IsLeader {
		addLeaderComponents(unitEntity)
	}

	return newUnitID, nil
}

// Copy Utilities
// These functions create deep copies of component data for state capture

// copyAttributes creates a deep copy of an Attributes struct
func copyAttributes(attr *common.Attributes) *common.Attributes {
	if attr == nil {
		return nil
	}
	return &common.Attributes{
		Strength:      attr.Strength,
		Dexterity:     attr.Dexterity,
		Magic:         attr.Magic,
		Leadership:    attr.Leadership,
		Armor:         attr.Armor,
		Weapon:        attr.Weapon,
		MovementSpeed: attr.MovementSpeed,
		AttackRange:   attr.AttackRange,
		CurrentHealth: attr.CurrentHealth,
		MaxHealth:     attr.MaxHealth,
		CanAct:        attr.CanAct,
	}
}

// copyGridPosition creates a deep copy of a GridPositionData struct
func copyGridPosition(gridPos *squads.GridPositionData) *squads.GridPositionData {
	if gridPos == nil {
		return nil
	}
	return &squads.GridPositionData{
		AnchorRow: gridPos.AnchorRow,
		AnchorCol: gridPos.AnchorCol,
		Width:     gridPos.Width,
		Height:    gridPos.Height,
	}
}
