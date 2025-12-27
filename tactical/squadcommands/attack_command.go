package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// AttackCommand wraps attack execution for undo/redo capability
type AttackCommand struct {
	entityManager *common.EntityManager
	combatSystem  *combat.CombatActionSystem
	attackerID    ecs.EntityID
	defenderID    ecs.EntityID

	// Undo state
	attackerName      string
	defenderName      string
	oldActionState    combat.ActionStateData
	oldDefenderHealth map[ecs.EntityID]int // Unit HP before attack
	wasDestroyed      bool
	defenderPosition  coords.LogicalPosition
}

func NewAttackCommand(
	manager *common.EntityManager,
	combatSys *combat.CombatActionSystem,
	attackerID, defenderID ecs.EntityID,
) *AttackCommand {
	return &AttackCommand{
		entityManager: manager,
		combatSystem:  combatSys,
		attackerID:    attackerID,
		defenderID:    defenderID,
	}
}

func (cmd *AttackCommand) Validate() error {
	reason, canAttack := cmd.combatSystem.CanSquadAttackWithReason(cmd.attackerID, cmd.defenderID)
	if !canAttack {
		return fmt.Errorf(reason)
	}

	return nil
}

func (cmd *AttackCommand) Execute() error {
	// Capture state for undo
	cmd.captureState()

	// Delegate to CombatActionSystem (REUSE EXISTING)
	result := cmd.combatSystem.ExecuteAttackAction(cmd.attackerID, cmd.defenderID)
	if !result.Success {
		return fmt.Errorf("attack failed: %s", result.ErrorReason)
	}

	// Capture results for undo
	cmd.captureResults()

	return nil
}

func (cmd *AttackCommand) Undo() error {
	// Restore action state
	cmd.restoreActionState()

	// Restore defender health
	for unitID, oldHP := range cmd.oldDefenderHealth {
		entity := cmd.entityManager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr != nil {
			attr.CurrentHealth = oldHP
		}
	}

	// Restore squad destroyed status
	if cmd.wasDestroyed {
		squads.UpdateSquadDestroyedStatus(cmd.defenderID, cmd.entityManager)

		// Re-add to position system if no longer destroyed
		defenderSquad := cmd.entityManager.FindEntityByID(cmd.defenderID)
		if defenderSquad != nil {
			squadData := common.GetComponentType[*squads.SquadData](defenderSquad, squads.SquadComponent)
			if squadData != nil && !squadData.IsDestroyed {
				common.GlobalPositionSystem.AddEntity(cmd.defenderID, cmd.defenderPosition)
			}
		}
	}

	return nil
}

func (cmd *AttackCommand) Description() string {
	return fmt.Sprintf("%s attacks %s", cmd.attackerName, cmd.defenderName)
}

func (cmd *AttackCommand) captureState() {
	// Get names
	cmd.attackerName = getSquadName(cmd.attackerID, cmd.entityManager)
	cmd.defenderName = getSquadName(cmd.defenderID, cmd.entityManager)

	// Capture action state (use proper two-step pattern)
	actionStateEntity := combat.FindActionStateEntity(cmd.attackerID, cmd.entityManager)
	if actionStateEntity != nil {
		actionState := common.GetComponentType[*combat.ActionStateData](actionStateEntity, combat.ActionStateComponent)
		if actionState != nil {
			cmd.oldActionState = *actionState
		}
	}

	// Capture defender health
	cmd.oldDefenderHealth = make(map[ecs.EntityID]int)
	unitIDs := squads.GetUnitIDsInSquad(cmd.defenderID, cmd.entityManager)
	for _, unitID := range unitIDs {
		entity := cmd.entityManager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr != nil {
			cmd.oldDefenderHealth[unitID] = attr.CurrentHealth
		}
	}

	// Capture defender position
	defenderSquad := cmd.entityManager.FindEntityByID(cmd.defenderID)
	if defenderSquad != nil {
		posPtr := common.GetComponentType[*coords.LogicalPosition](defenderSquad, common.PositionComponent)
		if posPtr != nil {
			cmd.defenderPosition = *posPtr
		}
	}
}

func (cmd *AttackCommand) captureResults() {
	cmd.wasDestroyed = squads.IsSquadDestroyed(cmd.defenderID, cmd.entityManager)
}

func (cmd *AttackCommand) restoreActionState() {
	actionStateEntity := combat.FindActionStateEntity(cmd.attackerID, cmd.entityManager)
	if actionStateEntity != nil {
		actionState := common.GetComponentType[*combat.ActionStateData](actionStateEntity, combat.ActionStateComponent)
		if actionState != nil {
			*actionState = cmd.oldActionState
		}
	}
}

func getSquadName(squadID ecs.EntityID, manager *common.EntityManager) string {
	squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
	if squadData != nil {
		return squadData.Name
	}
	return "Unknown"
}
