package gui

import (
	"fmt"
	"game_main/combat"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// CombatActionHandler manages combat actions and their execution
type CombatActionHandler struct {
	stateManager    *CombatStateManager
	logManager      *CombatLogManager
	queries         *GUIQueries
	entityManager   *common.EntityManager
	turnManager     *combat.TurnManager
	factionManager  *combat.FactionManager
	movementSystem  *combat.MovementSystem
	combatLogArea   *widget.TextArea
}

// NewCombatActionHandler creates a new combat action handler
func NewCombatActionHandler(
	stateManager *CombatStateManager,
	logManager *CombatLogManager,
	queries *GUIQueries,
	entityManager *common.EntityManager,
	turnManager *combat.TurnManager,
	factionManager *combat.FactionManager,
	movementSystem *combat.MovementSystem,
	combatLogArea *widget.TextArea,
) *CombatActionHandler {
	return &CombatActionHandler{
		stateManager:   stateManager,
		logManager:     logManager,
		queries:        queries,
		entityManager:  entityManager,
		turnManager:    turnManager,
		factionManager: factionManager,
		movementSystem: movementSystem,
		combatLogArea:  combatLogArea,
	}
}

// SelectSquad selects a squad and logs the action
func (cah *CombatActionHandler) SelectSquad(squadID ecs.EntityID) {
	cah.stateManager.SetSelectedSquad(squadID)

	// Get squad name and log
	squadName := cah.queries.GetSquadName(squadID)
	cah.addLog(fmt.Sprintf("Selected: %s", squadName))
}

// ToggleAttackMode enables/disables attack mode
func (cah *CombatActionHandler) ToggleAttackMode() {
	if cah.stateManager.GetSelectedSquad() == 0 {
		cah.addLog("Select a squad first!")
		return
	}

	newAttackMode := !cah.stateManager.IsAttackMode()
	cah.stateManager.SetAttackMode(newAttackMode)

	if newAttackMode {
		cah.addLog("Attack mode: Press 1-3 to target enemy")
		cah.showAvailableTargets()
	} else {
		cah.addLog("Attack mode cancelled")
	}
}

// ShowAvailableTargets displays available enemy targets
func (cah *CombatActionHandler) ShowAvailableTargets() {
	cah.showAvailableTargets()
}

func (cah *CombatActionHandler) showAvailableTargets() {
	currentFactionID := cah.turnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return
	}

	// Get all enemy squads
	enemySquads := cah.queries.GetEnemySquads(currentFactionID)

	if len(enemySquads) == 0 {
		cah.addLog("No enemy targets available!")
		return
	}

	// Show up to 3 targets
	for i := 0; i < len(enemySquads) && i < 3; i++ {
		targetName := cah.queries.GetSquadName(enemySquads[i])
		cah.addLog(fmt.Sprintf("  [%d] %s", i+1, targetName))
	}
}

// ToggleMoveMode enables/disables move mode
func (cah *CombatActionHandler) ToggleMoveMode() {
	if cah.stateManager.GetSelectedSquad() == 0 {
		cah.addLog("Select a squad first!")
		return
	}

	newMoveMode := !cah.stateManager.IsMoveMode()

	if newMoveMode {
		// Get valid movement tiles
		validTiles := cah.movementSystem.GetValidMovementTiles(cah.stateManager.GetSelectedSquad())

		if len(validTiles) == 0 {
			cah.addLog("No movement remaining!")
			return
		}

		cah.stateManager.SetMoveMode(true, validTiles)
		cah.addLog(fmt.Sprintf("Move mode: Click a tile (%d tiles available)", len(validTiles)))
		cah.addLog("Click on the map to move, or press M to cancel")
	} else {
		cah.stateManager.SetMoveMode(false, nil)
		cah.addLog("Move mode cancelled")
	}
}

// SelectTarget selects a target squad for attack
func (cah *CombatActionHandler) SelectTarget(targetSquadID ecs.EntityID) {
	if !cah.stateManager.IsAttackMode() {
		return
	}

	cah.stateManager.SetSelectedTarget(targetSquadID)
	cah.executeAttack()
}

// SelectEnemyTarget selects an enemy squad by index (0-2 for 1-3 keys)
func (cah *CombatActionHandler) SelectEnemyTarget(index int) {
	currentFactionID := cah.turnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return
	}

	// Get all enemy squads
	enemySquads := cah.queries.GetEnemySquads(currentFactionID)

	if index < 0 || index >= len(enemySquads) {
		cah.addLog(fmt.Sprintf("No enemy squad at index %d", index+1))
		return
	}

	cah.SelectTarget(enemySquads[index])
}

// ExecuteAttack performs an attack action
func (cah *CombatActionHandler) ExecuteAttack() {
	cah.executeAttack()
}

func (cah *CombatActionHandler) executeAttack() {
	selectedSquad := cah.stateManager.GetSelectedSquad()
	selectedTarget := cah.stateManager.GetSelectedTarget()

	if selectedSquad == 0 || selectedTarget == 0 {
		return
	}

	// Create combat action system
	combatSys := combat.NewCombatActionSystem(cah.entityManager)

	// Check if attack is valid with detailed reason
	reason, canAttack := combatSys.CanSquadAttackWithReason(selectedSquad, selectedTarget)
	if !canAttack {
		cah.addLog(fmt.Sprintf("Cannot attack: %s", reason))
		cah.stateManager.SetAttackMode(false)
		return
	}

	// Execute attack
	attackerName := cah.queries.GetSquadName(selectedSquad)
	targetName := cah.queries.GetSquadName(selectedTarget)

	err := combatSys.ExecuteAttackAction(selectedSquad, selectedTarget)
	if err != nil {
		cah.addLog(fmt.Sprintf("Attack failed: %v", err))
	} else {
		cah.addLog(fmt.Sprintf("%s attacked %s!", attackerName, targetName))

		// Check if target destroyed
		if squads.IsSquadDestroyed(selectedTarget, cah.entityManager) {
			cah.addLog(fmt.Sprintf("%s was destroyed!", targetName))
		}
	}

	// Reset attack mode
	cah.stateManager.SetAttackMode(false)
}

// MoveSquad moves a squad to a new position
func (cah *CombatActionHandler) MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) error {
	// Execute movement
	err := cah.movementSystem.MoveSquad(squadID, newPos)
	if err != nil {
		cah.addLog(fmt.Sprintf("Movement failed: %v", err))
		return err
	}

	// Update unit positions to match squad position
	cah.updateUnitPositions(squadID, newPos)

	squadName := cah.queries.GetSquadName(squadID)
	cah.addLog(fmt.Sprintf("%s moved to (%d, %d)", squadName, newPos.X, newPos.Y))

	// Exit move mode
	cah.stateManager.SetMoveMode(false, nil)

	return nil
}

// CycleSquadSelection selects the next squad in the faction
func (cah *CombatActionHandler) CycleSquadSelection() {
	currentFactionID := cah.turnManager.GetCurrentFaction()
	if currentFactionID == 0 || !cah.queries.IsPlayerFaction(currentFactionID) {
		return
	}

	squadIDs := cah.factionManager.GetFactionSquads(currentFactionID)

	// Filter out destroyed squads
	aliveSquads := []ecs.EntityID{}
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, cah.entityManager) {
			aliveSquads = append(aliveSquads, squadID)
		}
	}

	if len(aliveSquads) == 0 {
		return
	}

	// Find current index
	currentIndex := -1
	selectedSquad := cah.stateManager.GetSelectedSquad()
	for i, squadID := range aliveSquads {
		if squadID == selectedSquad {
			currentIndex = i
			break
		}
	}

	// Select next squad
	nextIndex := (currentIndex + 1) % len(aliveSquads)
	cah.SelectSquad(aliveSquads[nextIndex])
}

// updateUnitPositions updates all unit positions in a squad
func (cah *CombatActionHandler) updateUnitPositions(squadID ecs.EntityID, newSquadPos coords.LogicalPosition) {
	// Get all units in the squad
	unitIDs := squads.GetUnitIDsInSquad(squadID, cah.entityManager)

	// Update each unit's position to match the squad's new position
	for _, unitID := range unitIDs {
		// Find the unit in the ECS world and update its position
		unitEntity := common.FindEntityByIDWithTag(cah.entityManager, unitID, squads.SquadMemberTag)
		if unitEntity != nil && unitEntity.HasComponent(common.PositionComponent) {
			posPtr := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
			if posPtr != nil {
				posPtr.X = newSquadPos.X
				posPtr.Y = newSquadPos.Y
			}
		}
	}
}

// addLog adds a message to the combat log
func (cah *CombatActionHandler) addLog(message string) {
	cah.logManager.UpdateTextArea(cah.combatLogArea, message)
}
