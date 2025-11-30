package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// DisbandSquadCommand disbands a squad and returns all units to the roster
// Captures full squad state for undo
type DisbandSquadCommand struct {
	entityManager *common.EntityManager
	playerID      ecs.EntityID
	squadID       ecs.EntityID

	// Captured state for undo
	savedSquadData *squads.SquadData
	savedPosition  *coords.LogicalPosition
	savedUnits     []savedUnitState
}

// savedUnitState captures all data needed to recreate a unit
type savedUnitState struct {
	unitID       ecs.EntityID
	template     squads.UnitTemplate
	gridRow      int
	gridCol      int
	isLeader     bool
	attributes   *common.Attributes
	name         *common.Name
	gridPosition *squads.GridPositionData
}

// NewDisbandSquadCommand creates a new disband squad command
func NewDisbandSquadCommand(
	manager *common.EntityManager,
	playerID ecs.EntityID,
	squadID ecs.EntityID,
) *DisbandSquadCommand {
	return &DisbandSquadCommand{
		entityManager: manager,
		playerID:      playerID,
		squadID:       squadID,
	}
}

// Validate checks if the squad can be disbanded
func (cmd *DisbandSquadCommand) Validate() error {
	if cmd.squadID == 0 {
		return fmt.Errorf("invalid squad ID")
	}

	// Check if squad exists
	squadEntity := squads.GetSquadEntity(cmd.squadID, cmd.entityManager)
	if squadEntity == nil {
		return fmt.Errorf("squad does not exist")
	}

	// Get squad data for validation
	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData == nil {
		return fmt.Errorf("squad has no data component")
	}

	// Note: Additional validation could check if squad is in combat or deployed
	// For now, we allow disbanding any squad
	// Future: Add checks like:
	// if combat.IsSquadInCombat(cmd.squadID, cmd.entityManager) {
	//     return fmt.Errorf("cannot disband squad in combat")
	// }

	return nil
}

// Execute disbands the squad and returns units to roster
func (cmd *DisbandSquadCommand) Execute() error {
	// Capture state for undo BEFORE making changes
	if err := cmd.captureState(); err != nil {
		return fmt.Errorf("failed to capture squad state: %w", err)
	}

	// Get player roster
	roster := squads.GetPlayerRoster(cmd.playerID, cmd.entityManager)
	if roster == nil {
		return fmt.Errorf("player has no roster")
	}

	// Return all units to roster
	for _, unitState := range cmd.savedUnits {
		// Mark unit as available in roster
		if err := roster.MarkUnitAvailable(unitState.unitID); err != nil {
			// If marking available fails, the unit might not have been in roster
			// This is OK - we'll continue with other units
			continue
		}
	}

	// Remove all units from squad (dispose entities)
	unitIDs := squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)
	for _, unitID := range unitIDs {
		unitEntity := common.FindEntityByIDWithTag(cmd.entityManager, unitID, squads.SquadMemberTag)
		if unitEntity != nil {
			// Get position component for cleanup
			pos := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
			// Use CleanDisposeEntity to remove from both ECS World and GlobalPositionSystem
			cmd.entityManager.CleanDisposeEntity(unitEntity, pos)
		}
	}

	// Remove squad entity
	squadEntity := squads.GetSquadEntity(cmd.squadID, cmd.entityManager)
	if squadEntity != nil {
		// Get position component for cleanup
		pos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
		// Use CleanDisposeEntity to remove from both ECS World and GlobalPositionSystem
		cmd.entityManager.CleanDisposeEntity(squadEntity, pos)
	}

	return nil
}

// Undo recreates the squad from saved state
func (cmd *DisbandSquadCommand) Undo() error {
	if cmd.savedSquadData == nil {
		return fmt.Errorf("no saved state available for undo")
	}

	// Create new squad entity with saved ID
	squadEntity := cmd.entityManager.World.NewEntity()
	newSquadID := squadEntity.GetID()

	// Restore squad data (using new squad ID)
	restoredSquadData := &squads.SquadData{
		SquadID:       newSquadID,
		Formation:     cmd.savedSquadData.Formation,
		Name:          cmd.savedSquadData.Name,
		Morale:        cmd.savedSquadData.Morale,
		SquadLevel:    cmd.savedSquadData.SquadLevel,
		TurnCount:     cmd.savedSquadData.TurnCount,
		MaxUnits:      cmd.savedSquadData.MaxUnits,
		UsedCapacity:  cmd.savedSquadData.UsedCapacity,
		TotalCapacity: cmd.savedSquadData.TotalCapacity,
	}

	squadEntity.AddComponent(squads.SquadComponent, restoredSquadData)

	// Restore squad position
	if cmd.savedPosition != nil {
		squadEntity.AddComponent(common.PositionComponent, cmd.savedPosition)
		common.GlobalPositionSystem.AddEntity(newSquadID, *cmd.savedPosition)
	} else {
		squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})
	}

	// Get player roster
	roster := squads.GetPlayerRoster(cmd.playerID, cmd.entityManager)
	if roster == nil {
		return fmt.Errorf("player has no roster")
	}

	// Recreate all units
	for _, unitState := range cmd.savedUnits {
		// Create unit entity from template
		unitEntity, err := squads.CreateUnitEntity(cmd.entityManager, unitState.template)
		if err != nil {
			return fmt.Errorf("failed to recreate unit %s: %w", unitState.template.Name, err)
		}

		newUnitID := unitEntity.GetID()

		// Add squad member component
		unitEntity.AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
			SquadID: newSquadID,
		})

		// Restore grid position
		gridPos := common.GetComponentType[*squads.GridPositionData](unitEntity, squads.GridPositionComponent)
		gridPos.AnchorRow = unitState.gridRow
		gridPos.AnchorCol = unitState.gridCol
		gridPos.Width = unitState.gridPosition.Width
		gridPos.Height = unitState.gridPosition.Height

		// Restore leader status
		if unitState.isLeader {
			leaderData := &squads.LeaderData{
				Leadership: 10, // Default leadership value
				Experience: 0,
			}
			unitEntity.AddComponent(squads.LeaderComponent, leaderData)
		}

		// Mark unit as in squad in roster
		if err := roster.MarkUnitInSquad(newUnitID, newSquadID); err != nil {
			// If marking fails, continue with other units
			continue
		}
	}

	// Update squad capacity
	squads.UpdateSquadCapacity(newSquadID, cmd.entityManager)

	// Update the command's squad ID to the new one (for potential re-undo)
	cmd.squadID = newSquadID

	return nil
}

// Description returns a human-readable description
func (cmd *DisbandSquadCommand) Description() string {
	if cmd.savedSquadData != nil {
		return fmt.Sprintf("Disband squad '%s'", cmd.savedSquadData.Name)
	}
	return "Disband squad"
}

// captureState saves all squad state before disbanding
func (cmd *DisbandSquadCommand) captureState() error {
	// Get squad entity
	squadEntity := squads.GetSquadEntity(cmd.squadID, cmd.entityManager)
	if squadEntity == nil {
		return fmt.Errorf("squad not found")
	}

	// Save squad data
	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData == nil {
		return fmt.Errorf("squad has no data component")
	}

	cmd.savedSquadData = &squads.SquadData{
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

	// Save squad position
	if squadEntity.HasComponent(common.PositionComponent) {
		pos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
		if pos != nil {
			cmd.savedPosition = &coords.LogicalPosition{
				X: pos.X,
				Y: pos.Y,
			}
		}
	}

	// Save all unit states
	cmd.savedUnits = make([]savedUnitState, 0)
	unitIDs := squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)

	for _, unitID := range unitIDs {
		unitEntity := common.FindEntityByIDWithTag(cmd.entityManager, unitID, squads.SquadMemberTag)
		if unitEntity == nil {
			continue
		}

		// Get unit template (reconstruct from current state)
		unitState := savedUnitState{
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

		// Get name
		if unitEntity.HasComponent(common.NameComponent) {
			name := common.GetComponentType[*common.Name](unitEntity, common.NameComponent)
			if name != nil {
				unitState.name = &common.Name{
					NameStr: name.NameStr,
				}
				unitState.template.Name = name.NameStr
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

		// Get unit role
		if unitEntity.HasComponent(squads.UnitRoleComponent) {
			roleData := common.GetComponentType[*squads.UnitRoleData](unitEntity, squads.UnitRoleComponent)
			if roleData != nil {
				unitState.template.Role = roleData.Role
			}
		}

		// Build template from captured data
		if unitState.attributes != nil {
			unitState.template.Attributes = *unitState.attributes
		}

		cmd.savedUnits = append(cmd.savedUnits, unitState)
	}

	return nil
}
