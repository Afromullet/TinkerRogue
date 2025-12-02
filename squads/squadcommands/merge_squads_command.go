package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// MergeSquadsCommand merges a source squad into a target squad
// All units from source are moved to target, then source is destroyed
type MergeSquadsCommand struct {
	entityManager *common.EntityManager
	playerID      ecs.EntityID
	sourceSquadID ecs.EntityID
	targetSquadID ecs.EntityID

	// Captured state for undo
	savedSourceSquadData *squads.SquadData
	savedSourcePosition  *coords.LogicalPosition
	savedSourceUnits     []CapturedUnitState
	savedTargetCapacity  float64
}

// NewMergeSquadsCommand creates a new merge squads command
func NewMergeSquadsCommand(
	manager *common.EntityManager,
	playerID ecs.EntityID,
	sourceSquadID ecs.EntityID,
	targetSquadID ecs.EntityID,
) *MergeSquadsCommand {
	return &MergeSquadsCommand{
		entityManager: manager,
		playerID:      playerID,
		sourceSquadID: sourceSquadID,
		targetSquadID: targetSquadID,
	}
}

// Validate checks if the squads can be merged
func (cmd *MergeSquadsCommand) Validate() error {
	if cmd.sourceSquadID == cmd.targetSquadID {
		return fmt.Errorf("cannot merge squad with itself")
	}

	// Check if both squads exist
	if err := validateSquadExists(cmd.sourceSquadID, cmd.entityManager); err != nil {
		return fmt.Errorf("source squad: %w", err)
	}

	if err := validateSquadExists(cmd.targetSquadID, cmd.entityManager); err != nil {
		return fmt.Errorf("target squad: %w", err)
	}

	// Check if target has space for source units
	sourceUnitIDs := squads.GetUnitIDsInSquad(cmd.sourceSquadID, cmd.entityManager)
	sourceCapacityUsed := squads.GetSquadUsedCapacity(cmd.sourceSquadID, cmd.entityManager)
	targetCapacityRemaining := squads.GetSquadRemainingCapacity(cmd.targetSquadID, cmd.entityManager)

	if sourceCapacityUsed > targetCapacityRemaining {
		return fmt.Errorf("target squad does not have enough capacity (needs %.2f, has %.2f remaining)",
			sourceCapacityUsed, targetCapacityRemaining)
	}

	// Check grid space (simplified - just check if target has enough empty cells)
	targetUnitIDs := squads.GetUnitIDsInSquad(cmd.targetSquadID, cmd.entityManager)
	if len(sourceUnitIDs)+len(targetUnitIDs) > 9 {
		return fmt.Errorf("target squad does not have enough grid space")
	}

	return nil
}

// Execute merges source squad into target squad
func (cmd *MergeSquadsCommand) Execute() error {
	// Capture state for undo
	if err := cmd.captureState(); err != nil {
		return fmt.Errorf("failed to capture state: %w", err)
	}

	// Get source and target squads
	sourceEntity, err := getSquadOrError(cmd.sourceSquadID, cmd.entityManager)
	if err != nil {
		return fmt.Errorf("source squad: %w", err)
	}

	_, err = getSquadOrError(cmd.targetSquadID, cmd.entityManager)
	if err != nil {
		return fmt.Errorf("target squad: %w", err)
	}

	// Move all units from source to target
	sourceUnitIDs := squads.GetUnitIDsInSquad(cmd.sourceSquadID, cmd.entityManager)

	// Find empty grid positions in target
	emptyPositions := cmd.findEmptyPositions(cmd.targetSquadID)
	if len(emptyPositions) < len(sourceUnitIDs) {
		return fmt.Errorf("not enough empty grid positions in target squad")
	}

	// Move each unit
	for i, unitID := range sourceUnitIDs {
		// Update squad membership
		memberData := common.GetComponentTypeByIDWithTag[*squads.SquadMemberData](
			cmd.entityManager, unitID, squads.SquadMemberTag, squads.SquadMemberComponent)
		if memberData != nil {
			memberData.SquadID = cmd.targetSquadID
		}

		// Update grid position to empty slot in target
		gridPos := common.GetComponentTypeByIDWithTag[*squads.GridPositionData](
			cmd.entityManager, unitID, squads.SquadMemberTag, squads.GridPositionComponent)
		if gridPos != nil && i < len(emptyPositions) {
			gridPos.AnchorRow = emptyPositions[i][0]
			gridPos.AnchorCol = emptyPositions[i][1]
		}
	}

	// Update target squad capacity
	squads.UpdateSquadCapacity(cmd.targetSquadID, cmd.entityManager)

	// Destroy source squad
	pos := common.GetComponentType[*coords.LogicalPosition](sourceEntity, common.PositionComponent)
	// Use CleanDisposeEntity to remove from both ECS World and GlobalPositionSystem
	cmd.entityManager.CleanDisposeEntity(sourceEntity, pos)

	return nil
}

// Undo recreates the source squad and moves units back
func (cmd *MergeSquadsCommand) Undo() error {
	if cmd.savedSourceSquadData == nil {
		return fmt.Errorf("no saved state available for undo")
	}

	// Create new source squad entity
	squadEntity := cmd.entityManager.World.NewEntity()
	newSourceSquadID := squadEntity.GetID()

	// Restore squad data
	restoredSquadData := &squads.SquadData{
		SquadID:       newSourceSquadID,
		Formation:     cmd.savedSourceSquadData.Formation,
		Name:          cmd.savedSourceSquadData.Name,
		Morale:        cmd.savedSourceSquadData.Morale,
		SquadLevel:    cmd.savedSourceSquadData.SquadLevel,
		TurnCount:     cmd.savedSourceSquadData.TurnCount,
		MaxUnits:      cmd.savedSourceSquadData.MaxUnits,
		UsedCapacity:  cmd.savedSourceSquadData.UsedCapacity,
		TotalCapacity: cmd.savedSourceSquadData.TotalCapacity,
	}

	squadEntity.AddComponent(squads.SquadComponent, restoredSquadData)

	// Restore squad position
	if cmd.savedSourcePosition != nil {
		squadEntity.AddComponent(common.PositionComponent, cmd.savedSourcePosition)
		common.GlobalPositionSystem.AddEntity(newSourceSquadID, *cmd.savedSourcePosition)
	} else {
		squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})
	}

	// Move units back to source squad
	for _, unitState := range cmd.savedSourceUnits {
		// Update squad membership back to source
		memberData := common.GetComponentTypeByIDWithTag[*squads.SquadMemberData](
			cmd.entityManager, unitState.UnitID, squads.SquadMemberTag, squads.SquadMemberComponent)
		if memberData != nil {
			memberData.SquadID = newSourceSquadID
		}

		// Restore original grid position
		gridPos := common.GetComponentTypeByIDWithTag[*squads.GridPositionData](
			cmd.entityManager, unitState.UnitID, squads.SquadMemberTag, squads.GridPositionComponent)
		if gridPos != nil {
			gridPos.AnchorRow = unitState.GridRow
			gridPos.AnchorCol = unitState.GridCol
		}
	}

	// Update capacities
	squads.UpdateSquadCapacity(newSourceSquadID, cmd.entityManager)
	squads.UpdateSquadCapacity(cmd.targetSquadID, cmd.entityManager)

	// Update command's source squad ID for potential re-execution
	cmd.sourceSquadID = newSourceSquadID

	return nil
}

// Description returns a human-readable description
func (cmd *MergeSquadsCommand) Description() string {
	if cmd.savedSourceSquadData != nil {
		sourceName := cmd.savedSourceSquadData.Name
		targetName := squads.GetSquadName(cmd.targetSquadID, cmd.entityManager)
		return fmt.Sprintf("Merge squad '%s' into '%s'", sourceName, targetName)
	}
	return "Merge squads"
}

// captureState saves source squad state before merging
func (cmd *MergeSquadsCommand) captureState() error {
	// Get source squad
	sourceEntity, err := getSquadOrError(cmd.sourceSquadID, cmd.entityManager)
	if err != nil {
		return fmt.Errorf("source squad: %w", err)
	}

	// Save source squad data
	squadData, err := getSquadDataOrError(sourceEntity)
	if err != nil {
		return fmt.Errorf("source squad: %w", err)
	}

	cmd.savedSourceSquadData = &squads.SquadData{
		SquadID:       squadData.SquadID,
		Formation:     squadData.Formation,
		Name:          squadData.Name,
		Morale:        squadData.Morale,
		SquadLevel:    squadData.SquadLevel,
		TurnCount:     squadData.TurnCount,
		MaxUnits:      squadData.MaxUnits,
		UsedCapacity:  squadData.UsedCapacity,
		TotalCapacity: squadData.TotalCapacity,
	}

	// Save source position
	if sourceEntity.HasComponent(common.PositionComponent) {
		pos := common.GetComponentType[*coords.LogicalPosition](sourceEntity, common.PositionComponent)
		if pos != nil {
			cmd.savedSourcePosition = &coords.LogicalPosition{
				X: pos.X,
				Y: pos.Y,
			}
		}
	}

	// Save target capacity before merge
	cmd.savedTargetCapacity = squads.GetSquadUsedCapacity(cmd.targetSquadID, cmd.entityManager)

	// Capture all unit states from source squad using shared helper
	capturedUnits, err := CaptureAllUnitsInSquad(cmd.sourceSquadID, cmd.entityManager)
	if err != nil {
		return fmt.Errorf("failed to capture source unit states: %w", err)
	}
	cmd.savedSourceUnits = capturedUnits

	return nil
}

// findEmptyPositions finds empty grid positions in target squad
func (cmd *MergeSquadsCommand) findEmptyPositions(squadID ecs.EntityID) [][2]int {
	emptyPositions := make([][2]int, 0)

	// Check each grid cell
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			unitIDs := squads.GetUnitIDsAtGridPosition(squadID, row, col, cmd.entityManager)
			if len(unitIDs) == 0 {
				emptyPositions = append(emptyPositions, [2]int{row, col})
			}
		}
	}

	return emptyPositions
}
