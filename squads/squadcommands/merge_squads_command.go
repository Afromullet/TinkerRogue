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
	savedSourceUnits     []mergedUnitState
	savedTargetCapacity  float64
}

// mergedUnitState captures unit data for undo
type mergedUnitState struct {
	unitID       ecs.EntityID
	template     squads.UnitTemplate
	gridRow      int
	gridCol      int
	isLeader     bool
	attributes   *common.Attributes
	gridPosition *squads.GridPositionData
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
	if cmd.sourceSquadID == 0 || cmd.targetSquadID == 0 {
		return fmt.Errorf("invalid squad IDs")
	}

	if cmd.sourceSquadID == cmd.targetSquadID {
		return fmt.Errorf("cannot merge squad with itself")
	}

	// Check if both squads exist
	sourceEntity := squads.GetSquadEntity(cmd.sourceSquadID, cmd.entityManager)
	if sourceEntity == nil {
		return fmt.Errorf("source squad does not exist")
	}

	targetEntity := squads.GetSquadEntity(cmd.targetSquadID, cmd.entityManager)
	if targetEntity == nil {
		return fmt.Errorf("target squad does not exist")
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
	sourceEntity := squads.GetSquadEntity(cmd.sourceSquadID, cmd.entityManager)
	if sourceEntity == nil {
		return fmt.Errorf("source squad not found")
	}

	targetEntity := squads.GetSquadEntity(cmd.targetSquadID, cmd.entityManager)
	if targetEntity == nil {
		return fmt.Errorf("target squad not found")
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
		unitEntity := common.FindEntityByIDWithTag(cmd.entityManager, unitID, squads.SquadMemberTag)
		if unitEntity == nil {
			continue
		}

		// Update squad membership
		memberData := common.GetComponentType[*squads.SquadMemberData](unitEntity, squads.SquadMemberComponent)
		if memberData != nil {
			memberData.SquadID = cmd.targetSquadID
		}

		// Update grid position to empty slot in target
		gridPos := common.GetComponentType[*squads.GridPositionData](unitEntity, squads.GridPositionComponent)
		if gridPos != nil && i < len(emptyPositions) {
			gridPos.AnchorRow = emptyPositions[i][0]
			gridPos.AnchorCol = emptyPositions[i][1]
		}
	}

	// Update target squad capacity
	squads.UpdateSquadCapacity(cmd.targetSquadID, cmd.entityManager)

	// Destroy source squad
	if sourceEntity.HasComponent(common.PositionComponent) {
		pos := common.GetComponentType[*coords.LogicalPosition](sourceEntity, common.PositionComponent)
		if pos != nil {
			common.GlobalPositionSystem.RemoveEntity(cmd.sourceSquadID, *pos)
		}
	}
	cmd.entityManager.World.DisposeEntities(sourceEntity)

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
		unitEntity := common.FindEntityByIDWithTag(cmd.entityManager, unitState.unitID, squads.SquadMemberTag)
		if unitEntity == nil {
			continue
		}

		// Update squad membership back to source
		memberData := common.GetComponentType[*squads.SquadMemberData](unitEntity, squads.SquadMemberComponent)
		if memberData != nil {
			memberData.SquadID = newSourceSquadID
		}

		// Restore original grid position
		gridPos := common.GetComponentType[*squads.GridPositionData](unitEntity, squads.GridPositionComponent)
		if gridPos != nil {
			gridPos.AnchorRow = unitState.gridRow
			gridPos.AnchorCol = unitState.gridCol
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
	sourceEntity := squads.GetSquadEntity(cmd.sourceSquadID, cmd.entityManager)
	if sourceEntity == nil {
		return fmt.Errorf("source squad not found")
	}

	// Save source squad data
	squadData := common.GetComponentType[*squads.SquadData](sourceEntity, squads.SquadComponent)
	if squadData == nil {
		return fmt.Errorf("source squad has no data component")
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

	// Save all unit states from source
	cmd.savedSourceUnits = make([]mergedUnitState, 0)
	sourceUnitIDs := squads.GetUnitIDsInSquad(cmd.sourceSquadID, cmd.entityManager)

	for _, unitID := range sourceUnitIDs {
		unitEntity := common.FindEntityByIDWithTag(cmd.entityManager, unitID, squads.SquadMemberTag)
		if unitEntity == nil {
			continue
		}

		unitState := mergedUnitState{
			unitID: unitID,
		}

		// Get attributes
		if unitEntity.HasComponent(common.AttributeComponent) {
			attr := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
			if attr != nil {
				unitState.attributes = &common.Attributes{
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
		}

		// Get grid position
		if unitEntity.HasComponent(squads.GridPositionComponent) {
			gridPos := common.GetComponentType[*squads.GridPositionData](unitEntity, squads.GridPositionComponent)
			if gridPos != nil {
				unitState.gridRow = gridPos.AnchorRow
				unitState.gridCol = gridPos.AnchorCol
				unitState.gridPosition = &squads.GridPositionData{
					AnchorRow: gridPos.AnchorRow,
					AnchorCol: gridPos.AnchorCol,
					Width:     gridPos.Width,
					Height:    gridPos.Height,
				}
			}
		}

		// Check if leader
		unitState.isLeader = unitEntity.HasComponent(squads.LeaderComponent)

		cmd.savedSourceUnits = append(cmd.savedSourceUnits, unitState)
	}

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
