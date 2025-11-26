package combat

import (
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// CombatService encapsulates all combat game logic and system ownership
type CombatService struct {
	entityManager  *common.EntityManager
	turnManager    *TurnManager
	factionManager *FactionManager
	movementSystem *MovementSystem
}

// NewCombatService creates a new combat service
func NewCombatService(manager *common.EntityManager) *CombatService {
	return &CombatService{
		entityManager:  manager,
		turnManager:    NewTurnManager(manager),
		factionManager: NewFactionManager(manager),
		movementSystem: NewMovementSystem(manager, common.GlobalPositionSystem),
	}
}

// AttackResult contains all information about an attack execution
type AttackResult struct {
	Success         bool
	ErrorReason     string
	AttackerName    string
	TargetName      string
	TargetDestroyed bool
	DamageDealt     int
}

// ExecuteSquadAttack performs a squad attack and returns detailed result
func (cs *CombatService) ExecuteSquadAttack(attackerID, targetID ecs.EntityID) *AttackResult {
	result := &AttackResult{}

	// Create combat action system
	combatSys := NewCombatActionSystem(cs.entityManager)

	// Validate attack
	reason, canAttack := combatSys.CanSquadAttackWithReason(attackerID, targetID)
	if !canAttack {
		result.Success = false
		result.ErrorReason = reason
		return result
	}

	// Get names for result
	result.AttackerName = getSquadNameByID(attackerID, cs.entityManager)
	result.TargetName = getSquadNameByID(targetID, cs.entityManager)

	// Execute attack
	err := combatSys.ExecuteAttackAction(attackerID, targetID)
	if err != nil {
		result.Success = false
		result.ErrorReason = err.Error()
		return result
	}

	result.Success = true
	result.TargetDestroyed = squads.IsSquadDestroyed(targetID, cs.entityManager)

	return result
}

// MoveSquadResult contains all information about a movement execution
type MoveSquadResult struct {
	Success       bool
	ErrorReason   string
	SquadName     string
	NewPosition   coords.LogicalPosition
	MovementCost  int
	RemainingAPs  int
}

// MoveSquad moves a squad to a new position and returns result
func (cs *CombatService) MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) *MoveSquadResult {
	result := &MoveSquadResult{
		NewPosition: newPos,
	}

	// Execute movement
	err := cs.movementSystem.MoveSquad(squadID, newPos)
	if err != nil {
		result.Success = false
		result.ErrorReason = err.Error()
		return result
	}

	result.Success = true
	result.SquadName = getSquadNameByID(squadID, cs.entityManager)

	// Update action state with remaining APs
	actionEntity := findActionStateEntity(squadID, cs.entityManager)
	if actionEntity != nil {
		actionState := common.GetComponentType[*ActionStateData](actionEntity, ActionStateComponent)
		if actionState != nil {
			result.RemainingAPs = actionState.MovementRemaining
		}
	}

	return result
}

// GetValidMovementTiles returns the list of tiles a squad can move to
func (cs *CombatService) GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition {
	return cs.movementSystem.GetValidMovementTiles(squadID)
}

// GetSquadsInRange returns all enemy squads within attack range
func (cs *CombatService) GetSquadsInRange(squadID ecs.EntityID) []ecs.EntityID {
	combatSys := NewCombatActionSystem(cs.entityManager)
	return combatSys.GetSquadsInRange(squadID)
}

// InitializeCombat initializes combat with the given factions
func (cs *CombatService) InitializeCombat(factionIDs []ecs.EntityID) error {
	return cs.turnManager.InitializeCombat(factionIDs)
}

// GetCurrentFaction returns the current faction's turn
func (cs *CombatService) GetCurrentFaction() ecs.EntityID {
	return cs.turnManager.GetCurrentFaction()
}

// ResetSquadActions resets action state for all squads in a faction
func (cs *CombatService) ResetSquadActions(factionID ecs.EntityID) error {
	return cs.turnManager.ResetSquadActions(factionID)
}

// GetTurnManager exposes turn manager for UI queries (read-only)
func (cs *CombatService) GetTurnManager() *TurnManager {
	return cs.turnManager
}

// GetFactionManager exposes faction manager for UI queries (read-only)
func (cs *CombatService) GetFactionManager() *FactionManager {
	return cs.factionManager
}

// GetMovementSystem exposes movement system for UI queries (read-only)
func (cs *CombatService) GetMovementSystem() *MovementSystem {
	return cs.movementSystem
}

// GetEntityManager exposes entity manager for UI queries (read-only)
func (cs *CombatService) GetEntityManager() *common.EntityManager {
	return cs.entityManager
}

// getSquadNameByID is a helper to get squad name from ID
func getSquadNameByID(squadID ecs.EntityID, manager *common.EntityManager) string {
	squadData := common.GetComponentTypeByIDWithTag[*squads.SquadData](manager, squadID, squads.SquadTag, squads.SquadComponent)
	if squadData != nil {
		return squadData.Name
	}
	return "Unknown"
}
