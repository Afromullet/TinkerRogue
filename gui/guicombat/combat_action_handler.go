package guicombat

import (
	"game_main/gui/core"
	"game_main/gui/guicomponents"

	"fmt"
	"game_main/combat"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// CombatActionHandler manages combat actions and their execution
type CombatActionHandler struct {
	battleMapState *core.BattleMapState
	logManager     *CombatLogManager
	queries        *guicomponents.GUIQueries
	combatService  *combat.CombatService
	combatLogArea  *widget.TextArea
}

// NewCombatActionHandler creates a new combat action handler
func NewCombatActionHandler(
	battleMapState *core.BattleMapState,
	logManager *CombatLogManager,
	queries *guicomponents.GUIQueries,
	combatService *combat.CombatService,
	combatLogArea *widget.TextArea,
) *CombatActionHandler {
	return &CombatActionHandler{
		battleMapState: battleMapState,
		logManager:     logManager,
		queries:        queries,
		combatService:  combatService,
		combatLogArea:  combatLogArea,
	}
}

// SelectSquad selects a squad and logs the action
func (cah *CombatActionHandler) SelectSquad(squadID ecs.EntityID) {
	cah.battleMapState.SelectedSquadID = squadID

	// Get squad name and log
	squadName := cah.queries.GetSquadName(squadID)
	cah.addLog(fmt.Sprintf("Selected: %s", squadName))
}

// ToggleAttackMode enables/disables attack mode
func (cah *CombatActionHandler) ToggleAttackMode() {
	if cah.battleMapState.SelectedSquadID == 0 {
		cah.addLog("Select a squad first!")
		return
	}

	newAttackMode := !cah.battleMapState.InAttackMode
	cah.battleMapState.InAttackMode = newAttackMode

	if newAttackMode {
		cah.addLog("Attack mode: Press 1-3 to target enemy")
		cah.ShowAvailableTargets()
	} else {
		cah.addLog("Attack mode cancelled")
	}
}

func (cah *CombatActionHandler) ShowAvailableTargets() {
	currentFactionID := cah.combatService.GetCurrentFaction()
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
	if cah.battleMapState.SelectedSquadID == 0 {
		cah.addLog("Select a squad first!")
		return
	}

	newMoveMode := !cah.battleMapState.InMoveMode

	if newMoveMode {
		// Get valid movement tiles
		validTiles := cah.combatService.GetValidMovementTiles(cah.battleMapState.SelectedSquadID)

		if len(validTiles) == 0 {
			cah.addLog("No movement remaining!")
			return
		}

		cah.battleMapState.InMoveMode = true
		cah.battleMapState.ValidMoveTiles = validTiles
		cah.addLog(fmt.Sprintf("Move mode: Click a tile (%d tiles available)", len(validTiles)))
		cah.addLog("Click on the map to move, or press M to cancel")
	} else {
		cah.battleMapState.InMoveMode = false
		cah.battleMapState.ValidMoveTiles = []coords.LogicalPosition{}
		cah.addLog("Move mode cancelled")
	}
}

// SelectTarget selects a target squad for attack
func (cah *CombatActionHandler) SelectTarget(targetSquadID ecs.EntityID) {
	if !cah.battleMapState.InAttackMode {
		return
	}

	cah.battleMapState.SelectedTargetID = targetSquadID
	cah.ExecuteAttack()
}

// SelectEnemyTarget selects an enemy squad by index (0-2 for 1-3 keys)
func (cah *CombatActionHandler) SelectEnemyTarget(index int) {
	currentFactionID := cah.combatService.GetCurrentFaction()
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

func (cah *CombatActionHandler) ExecuteAttack() {
	selectedSquad := cah.battleMapState.SelectedSquadID
	selectedTarget := cah.battleMapState.SelectedTargetID

	if selectedSquad == 0 || selectedTarget == 0 {
		return
	}

	// Call service for all game logic
	result := cah.combatService.ExecuteSquadAttack(selectedSquad, selectedTarget)

	// Handle result - UI ONLY
	if !result.Success {
		cah.addLog(fmt.Sprintf("Cannot attack: %s", result.ErrorReason))
	} else {
		cah.addLog(fmt.Sprintf("%s attacked %s!", result.AttackerName, result.TargetName))
		if result.TargetDestroyed {
			cah.addLog(fmt.Sprintf("%s was destroyed!", result.TargetName))
		}
	}

	// Reset UI state
	cah.battleMapState.InAttackMode = false
}

// MoveSquad moves a squad to a new position
func (cah *CombatActionHandler) MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) error {
	// Execute movement
	result := cah.combatService.MoveSquad(squadID, newPos)
	if !result.Success {
		cah.addLog(fmt.Sprintf("Movement failed: %s", result.ErrorReason))
		return fmt.Errorf(result.ErrorReason)
	}

	// Update unit positions to match squad position using service
	cah.combatService.UpdateUnitPositions(squadID, newPos)

	cah.addLog(fmt.Sprintf("%s moved to (%d, %d)", result.SquadName, newPos.X, newPos.Y))

	// Exit move mode
	cah.battleMapState.InMoveMode = false
	cah.battleMapState.ValidMoveTiles = []coords.LogicalPosition{}

	return nil
}

// CycleSquadSelection selects the next squad in the faction
func (cah *CombatActionHandler) CycleSquadSelection() {
	currentFactionID := cah.combatService.GetCurrentFaction()
	if currentFactionID == 0 || !cah.queries.IsPlayerFaction(currentFactionID) {
		return
	}

	squadIDs := cah.combatService.GetFactionManager().GetFactionSquads(currentFactionID)

	// Filter out destroyed squads
	aliveSquads := []ecs.EntityID{}
	entityManager := cah.combatService.GetEntityManager()
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, entityManager) {
			aliveSquads = append(aliveSquads, squadID)
		}
	}

	if len(aliveSquads) == 0 {
		return
	}

	// Find current index
	currentIndex := -1
	selectedSquad := cah.battleMapState.SelectedSquadID
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

// addLog adds a message to the combat log
func (cah *CombatActionHandler) addLog(message string) {
	cah.logManager.UpdateTextArea(cah.combatLogArea, message)
}
