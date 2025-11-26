package combatservices

import (
	"game_main/combat"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// CombatService encapsulates all combat game logic and system ownership
type CombatService struct {
	entityManager  *common.EntityManager
	turnManager    *combat.TurnManager
	factionManager *combat.FactionManager
	movementSystem *combat.MovementSystem
}

// NewCombatService creates a new combat service
func NewCombatService(manager *common.EntityManager) *CombatService {
	return &CombatService{
		entityManager:  manager,
		turnManager:    combat.NewTurnManager(manager),
		factionManager: combat.NewFactionManager(manager),
		movementSystem: combat.NewMovementSystem(manager, common.GlobalPositionSystem),
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
	combatSys := combat.NewCombatActionSystem(cs.entityManager)

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
	Success      bool
	ErrorReason  string
	SquadName    string
	NewPosition  coords.LogicalPosition
	MovementCost int
	RemainingAPs int
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
	actionEntity := combat.FindActionStateEntity(squadID, cs.entityManager)
	if actionEntity != nil {
		actionState := common.GetComponentType[*combat.ActionStateData](actionEntity, combat.ActionStateComponent)
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
	combatSys := combat.NewCombatActionSystem(cs.entityManager)
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

// GetMovementSystem exposes movement system for UI queries (read-only)
func (cs *CombatService) GetMovementSystem() *combat.MovementSystem {
	return cs.movementSystem
}

// GetCurrentRound returns the current combat round number
func (cs *CombatService) GetCurrentRound() int {
	return cs.turnManager.GetCurrentRound()
}

// GetAliveSquadsInFaction returns all alive squads for a faction
func (cs *CombatService) GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID {
	squadIDs := cs.factionManager.GetFactionSquads(factionID)
	result := []ecs.EntityID{}
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, cs.entityManager) {
			result = append(result, squadID)
		}
	}
	return result
}

// EndTurnResult contains information about turn transition
type EndTurnResult struct {
	Success         bool
	PreviousFaction ecs.EntityID
	NewFaction      ecs.EntityID
	NewRound        int
	Error           string
}

// EndTurn ends the current faction's turn and advances to next
func (cs *CombatService) EndTurn() *EndTurnResult {
	result := &EndTurnResult{
		PreviousFaction: cs.turnManager.GetCurrentFaction(),
	}

	err := cs.turnManager.EndTurn()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.NewFaction = cs.turnManager.GetCurrentFaction()
	result.NewRound = cs.turnManager.GetCurrentRound()
	return result
}

// VictoryCheckResult contains battle outcome information
type VictoryCheckResult struct {
	BattleOver       bool
	VictorFaction    ecs.EntityID
	VictorName       string
	DefeatedFactions []ecs.EntityID
	RoundsCompleted  int
}

// CheckVictoryCondition checks if battle has ended
func (cs *CombatService) CheckVictoryCondition() *VictoryCheckResult {
	result := &VictoryCheckResult{
		RoundsCompleted: cs.turnManager.GetCurrentRound(),
	}

	// Count alive squads per faction
	aliveByFaction := make(map[ecs.EntityID]int)

	for _, queryResult := range cs.entityManager.World.Query(combat.FactionTag) {
		entity := queryResult.Entity
		factionID := entity.GetID()
		aliveByFaction[factionID] = 0
	}

	// Count squads
	for _, queryResult := range cs.entityManager.World.Query(squads.SquadTag) {
		entity := queryResult.Entity
		squadData := common.GetComponentType[*squads.SquadData](entity, squads.SquadComponent)
		if squadData != nil && !squads.IsSquadDestroyed(entity.GetID(), cs.entityManager) {
			factionData := common.GetComponentTypeByIDWithTag[*combat.FactionData](
				cs.entityManager, entity.GetID(), squads.SquadTag, combat.FactionComponent)
			if factionData != nil {
				aliveByFaction[factionData.FactionID]++
			}
		}
	}

	// Check victory: only one faction with alive squads
	factionsWithSquads := 0
	var victorFaction ecs.EntityID
	for factionID, count := range aliveByFaction {
		if count > 0 {
			factionsWithSquads++
			victorFaction = factionID
		} else {
			result.DefeatedFactions = append(result.DefeatedFactions, factionID)
		}
	}

	if factionsWithSquads <= 1 {
		result.BattleOver = true
		result.VictorFaction = victorFaction
		result.VictorName = cs.factionManager.GetFactionName(victorFaction)
	}

	return result
}

// UpdateUnitPositions updates all unit positions in a squad to match the squad's new position
func (cs *CombatService) UpdateUnitPositions(squadID ecs.EntityID, newSquadPos coords.LogicalPosition) error {
	// Get all units in the squad
	unitIDs := squads.GetUnitIDsInSquad(squadID, cs.entityManager)

	// Update each unit's position to match the squad's new position
	for _, unitID := range unitIDs {
		// Find the unit in the ECS world and update its position
		unitEntity := common.FindEntityByIDWithTag(cs.entityManager, unitID, squads.SquadMemberTag)
		if unitEntity != nil && unitEntity.HasComponent(common.PositionComponent) {
			posPtr := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
			if posPtr != nil {
				posPtr.X = newSquadPos.X
				posPtr.Y = newSquadPos.Y
			}
		}
	}

	return nil
}

// getSquadNameByID is a helper to get squad name from ID
func getSquadNameByID(squadID ecs.EntityID, manager *common.EntityManager) string {
	squadData := common.GetComponentTypeByIDWithTag[*squads.SquadData](manager, squadID, squads.SquadTag, squads.SquadComponent)
	if squadData != nil {
		return squadData.Name
	}
	return "Unknown"
}
