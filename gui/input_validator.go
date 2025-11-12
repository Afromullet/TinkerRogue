package gui

import (
	"game_main/coords"
	"github.com/bytearena/ecs"
)

// InputValidator provides centralized validation for input handling across all modes.
// This prevents duplication of validation logic and ensures consistent error checking.
type InputValidator struct {
	queries *GUIQueries
}

// NewInputValidator creates a new input validator
func NewInputValidator(queries *GUIQueries) *InputValidator {
	return &InputValidator{
		queries: queries,
	}
}

// IsValidCoordinate checks if a logical position is within valid dungeon bounds
func (iv *InputValidator) IsValidCoordinate(pos coords.LogicalPosition) bool {
	if pos.X < 0 || pos.Y < 0 {
		return false
	}
	// Could add more sophisticated bounds checking here
	return true
}

// IsValidEntity checks if an entity ID exists and is valid
func (iv *InputValidator) IsValidEntity(entityID ecs.EntityID) bool {
	return entityID != 0
}

// IsValidSquad checks if a squad exists and retrieves its info
// Returns (isValid, squadInfo)
func (iv *InputValidator) IsValidSquad(squadID ecs.EntityID) (bool, *SquadInfo) {
	if !iv.IsValidEntity(squadID) {
		return false, nil
	}

	squadInfo := iv.queries.GetSquadInfo(squadID)
	return squadInfo != nil, squadInfo
}

// IsValidFaction checks if a faction exists
func (iv *InputValidator) IsValidFaction(factionID ecs.EntityID) bool {
	if !iv.IsValidEntity(factionID) {
		return false
	}

	factionInfo := iv.queries.GetFactionInfo(factionID)
	return factionInfo != nil
}

// IsClickInWorldBounds checks if a click is within the game world viewport
// Returns true if the coordinate is valid
func (iv *InputValidator) IsClickInWorldBounds(screenX, screenY int) bool {
	// Basic bounds check - can be extended with actual viewport bounds
	// For now, any screen coordinate is valid as the viewport handles clipping
	return screenX >= 0 && screenY >= 0
}

// ValidateSquadClick performs comprehensive validation for a squad click
// Returns (isValid, squadID, squadInfo, error message)
func (iv *InputValidator) ValidateSquadClick(squadID ecs.EntityID) (bool, *SquadInfo, string) {
	if !iv.IsValidEntity(squadID) {
		return false, nil, "No squad at clicked position"
	}

	isValid, squadInfo := iv.IsValidSquad(squadID)
	if !isValid {
		return false, nil, "Squad not found"
	}

	if squadInfo.IsDestroyed {
		return false, squadInfo, "Squad is destroyed"
	}

	return true, squadInfo, ""
}

// ValidatePlayerAction checks if player can perform an action
// Returns (canAct, reason)
func (iv *InputValidator) ValidatePlayerAction(isPlayerTurn bool, playerFaction ecs.EntityID) (bool, string) {
	if !isPlayerTurn {
		return false, "Not your turn"
	}

	if !iv.IsValidFaction(playerFaction) {
		return false, "Invalid faction"
	}

	return true, ""
}

// ValidateFactionRelationship checks the relationship between two factions
// Returns (isAllied, isEnemy, reason)
func (iv *InputValidator) ValidateFactionRelationship(playerFaction, targetFaction ecs.EntityID) (bool, bool, string) {
	if playerFaction == targetFaction {
		return true, false, "allied"
	}

	if playerFaction == 0 || targetFaction == 0 {
		return false, false, "invalid faction"
	}

	return false, true, "enemy"
}
