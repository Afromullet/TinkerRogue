package gui

import (
	"fmt"
	"game_main/combat"
	"game_main/common"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// CombatActionValidator validates combat actions before execution.
// This separates validation logic from action execution, improving:
// - Testability (can test validation independently)
// - Reusability (validation can be used in multiple contexts)
// - Clarity (clear separation of concerns)
type CombatActionValidator struct {
	stateManager   *CombatStateManager
	queries        *GUIQueries
	turnManager    *combat.TurnManager
	entityManager  *common.EntityManager
	movementSystem *combat.MovementSystem
}

// NewCombatActionValidator creates a new action validator
func NewCombatActionValidator(
	stateManager *CombatStateManager,
	queries *GUIQueries,
	turnManager *combat.TurnManager,
	entityManager *common.EntityManager,
	movementSystem *combat.MovementSystem,
) *CombatActionValidator {
	return &CombatActionValidator{
		stateManager:   stateManager,
		queries:        queries,
		turnManager:    turnManager,
		entityManager:  entityManager,
		movementSystem: movementSystem,
	}
}

// ValidateSquadSelected checks if a squad is selected
// Returns (isValid, reason)
func (cav *CombatActionValidator) ValidateSquadSelected() (bool, string) {
	if cav.stateManager.GetSelectedSquad() == 0 {
		return false, "Select a squad first!"
	}
	return true, ""
}

// ValidateAttackModeToggle checks if attack mode can be toggled
// Returns (canToggle, reason)
func (cav *CombatActionValidator) ValidateAttackModeToggle() (bool, string) {
	if !cav.stateManager.IsAttackMode() {
		// Trying to enable attack mode
		isValid, reason := cav.ValidateSquadSelected()
		if !isValid {
			return false, reason
		}

		// Check if there are enemy targets
		currentFactionID := cav.turnManager.GetCurrentFaction()
		if currentFactionID == 0 {
			return false, "No active faction"
		}

		enemySquads := cav.queries.GetEnemySquads(currentFactionID)
		if len(enemySquads) == 0 {
			return false, "No enemy targets available!"
		}
	}

	return true, ""
}

// ValidateMoveModeToggle checks if move mode can be toggled
// Returns (canToggle, reason)
func (cav *CombatActionValidator) ValidateMoveModeToggle() (bool, string) {
	if !cav.stateManager.IsMoveMode() {
		// Trying to enable move mode
		isValid, reason := cav.ValidateSquadSelected()
		if !isValid {
			return false, reason
		}

		// Check if squad has movement remaining
		validTiles := cav.movementSystem.GetValidMovementTiles(cav.stateManager.GetSelectedSquad())
		if len(validTiles) == 0 {
			return false, "No movement remaining!"
		}
	}

	return true, ""
}

// ValidateAttack checks if an attack is valid
// Returns (canAttack, reason)
func (cav *CombatActionValidator) ValidateAttack(selectedSquad, selectedTarget ecs.EntityID) (bool, string) {
	if selectedSquad == 0 {
		return false, "No squad selected"
	}

	if selectedTarget == 0 {
		return false, "No target selected"
	}

	if !cav.stateManager.IsAttackMode() {
		return false, "Not in attack mode"
	}

	// Check with combat system
	combatSys := combat.NewCombatActionSystem(cav.entityManager)
	reason, canAttack := combatSys.CanSquadAttackWithReason(selectedSquad, selectedTarget)

	return canAttack, reason
}

// ValidateMovement checks if a movement is valid
// Returns (canMove, reason)
func (cav *CombatActionValidator) ValidateMovement(squadID ecs.EntityID) (bool, string) {
	if squadID == 0 {
		return false, "Invalid squad"
	}

	if squads.IsSquadDestroyed(squadID, cav.entityManager) {
		return false, "Squad is destroyed"
	}

	validTiles := cav.movementSystem.GetValidMovementTiles(squadID)
	if len(validTiles) == 0 {
		return false, "No movement remaining"
	}

	return true, ""
}

// ValidateEnemyTargetSelection checks if a target index is valid
// Returns (canSelect, reason, targetSquadID)
func (cav *CombatActionValidator) ValidateEnemyTargetSelection(index int) (bool, string, ecs.EntityID) {
	currentFactionID := cav.turnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return false, "No active faction", 0
	}

	enemySquads := cav.queries.GetEnemySquads(currentFactionID)
	if index < 0 || index >= len(enemySquads) {
		return false, fmt.Sprintf("No enemy squad at index %d", index+1), 0
	}

	return true, "", enemySquads[index]
}

// GetValidMovementTilesCount returns the number of valid movement tiles for a squad
func (cav *CombatActionValidator) GetValidMovementTilesCount(squadID ecs.EntityID) int {
	validTiles := cav.movementSystem.GetValidMovementTiles(squadID)
	return len(validTiles)
}

// GetAvailableEnemyTargets returns the list of available enemy targets
func (cav *CombatActionValidator) GetAvailableEnemyTargets() []ecs.EntityID {
	currentFactionID := cav.turnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return []ecs.EntityID{}
	}

	return cav.queries.GetEnemySquads(currentFactionID)
}
