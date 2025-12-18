package guicombat

import (
	"fmt"
	"game_main/combat/combatservices"
	"game_main/coords"
	"game_main/gui/core"
	"game_main/gui/guicomponents"
	"game_main/squads/squadcommands"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// CombatActionHandler manages combat actions and their execution
type CombatActionHandler struct {
	battleMapState  *core.BattleMapState
	logManager      *CombatLogManager
	queries         *guicomponents.GUIQueries
	combatService   *combatservices.CombatService
	combatLogArea   *widget.TextArea
	commandExecutor *squadcommands.CommandExecutor // Command pattern for undo/redo
	modeManager     *core.UIModeManager            // For triggering combat animation mode
}

// NewCombatActionHandler creates a new combat action handler
func NewCombatActionHandler(
	battleMapState *core.BattleMapState,
	logManager *CombatLogManager,
	queries *guicomponents.GUIQueries,
	combatService *combatservices.CombatService,
	combatLogArea *widget.TextArea,
	modeManager *core.UIModeManager,
) *CombatActionHandler {
	return &CombatActionHandler{
		battleMapState:  battleMapState,
		logManager:      logManager,
		queries:         queries,
		combatService:   combatService,
		combatLogArea:   combatLogArea,
		commandExecutor: squadcommands.NewCommandExecutor(),
		modeManager:     modeManager,
	}
}

// SelectSquad selects a squad and logs the action
func (cah *CombatActionHandler) SelectSquad(squadID ecs.EntityID) {
	cah.battleMapState.SelectedSquadID = squadID

	// Get squad name and log
	squadName := cah.queries.SquadCache.GetSquadName(squadID)
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
		targetName := cah.queries.SquadCache.GetSquadName(enemySquads[i])
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

	// Validate attack BEFORE triggering animation (prevents animation on invalid attacks)
	result := cah.combatService.ExecuteSquadAttack(selectedSquad, selectedTarget)

	// Invalidate cache for both squads (attacker and defender) since their HP/status changed
	cah.queries.MarkSquadDirty(selectedSquad)
	cah.queries.MarkSquadDirty(selectedTarget)

	// Reset UI state
	cah.battleMapState.InAttackMode = false

	// Handle invalid attacks immediately (no animation)
	if !result.Success {
		cah.addLog(fmt.Sprintf("Cannot attack: %s", result.ErrorReason))
		return
	}

	// Attack is valid - show animation then apply results
	if cah.modeManager != nil {
		if animMode, exists := cah.modeManager.GetMode("combat_animation"); exists {
			if caMode, ok := animMode.(*CombatAnimationMode); ok {
				caMode.SetCombatants(selectedSquad, selectedTarget)
				caMode.SetOnComplete(func() {
					// Animation complete - show attack results
					cah.addLog(fmt.Sprintf("%s attacked %s!", result.AttackerName, result.TargetName))
					if result.TargetDestroyed {
						cah.addLog(fmt.Sprintf("%s was destroyed!", result.TargetName))
					}
				})
				cah.modeManager.RequestTransition(animMode, "Combat animation")
				return
			}
		}
	}

	// Fallback: no animation mode, just show results
	cah.addLog(fmt.Sprintf("%s attacked %s!", result.AttackerName, result.TargetName))
	if result.TargetDestroyed {
		cah.addLog(fmt.Sprintf("%s was destroyed!", result.TargetName))
	}
}


// MoveSquad moves a squad to a new position using command pattern for undo support
func (cah *CombatActionHandler) MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) error {
	// Get movement system from combat service
	movementSystem := cah.combatService.GetMovementSystem()

	// Create move command with system reference
	cmd := squadcommands.NewMoveSquadCommand(
		cah.queries.ECSManager,
		movementSystem,
		squadID,
		newPos,
	)

	// Execute via command executor
	result := cah.commandExecutor.Execute(cmd)

	if !result.Success {
		cah.addLog(fmt.Sprintf("Movement failed: %s", result.Error))
		return fmt.Errorf(result.Error)
	}

	// Invalidate cache for the moved squad (position and movement remaining changed)
	cah.queries.MarkSquadDirty(squadID)

	cah.addLog(fmt.Sprintf("✓ %s", result.Description))

	// Exit move mode
	cah.battleMapState.InMoveMode = false
	cah.battleMapState.ValidMoveTiles = []coords.LogicalPosition{}

	return nil
}

// UndoLastMove undoes the last movement command
func (cah *CombatActionHandler) UndoLastMove() {
	if !cah.commandExecutor.CanUndo() {
		cah.addLog("Nothing to undo")
		return
	}

	result := cah.commandExecutor.Undo()

	if result.Success {
		// Invalidate all squads since squad positions changed
		cah.queries.MarkAllSquadsDirty()
		cah.addLog(fmt.Sprintf("⟲ Undid: %s", result.Description))
	} else {
		cah.addLog(fmt.Sprintf("Undo failed: %s", result.Error))
	}
}

// RedoLastMove redoes the last undone movement command
func (cah *CombatActionHandler) RedoLastMove() {
	if !cah.commandExecutor.CanRedo() {
		cah.addLog("Nothing to redo")
		return
	}

	result := cah.commandExecutor.Redo()

	if result.Success {
		// Invalidate all squads since squad positions changed
		cah.queries.MarkAllSquadsDirty()
		cah.addLog(fmt.Sprintf("⟳ Redid: %s", result.Description))
	} else {
		cah.addLog(fmt.Sprintf("Redo failed: %s", result.Error))
	}
}

// CanUndoMove returns whether there are moves to undo
func (cah *CombatActionHandler) CanUndoMove() bool {
	return cah.commandExecutor.CanUndo()
}

// CanRedoMove returns whether there are moves to redo
func (cah *CombatActionHandler) CanRedoMove() bool {
	return cah.commandExecutor.CanRedo()
}

// ClearMoveHistory clears all movement history (called when ending turn)
func (cah *CombatActionHandler) ClearMoveHistory() {
	cah.commandExecutor.ClearHistory()
	cah.addLog("Movement history cleared")
}

// CycleSquadSelection selects the next squad in the faction
func (cah *CombatActionHandler) CycleSquadSelection() {
	currentFactionID := cah.combatService.GetCurrentFaction()
	factionData := cah.queries.CombatCache.FindFactionDataByID(currentFactionID, cah.queries.ECSManager)
	if currentFactionID == 0 || factionData == nil || !factionData.IsPlayerControlled {
		return
	}

	// Get alive squads using service
	aliveSquads := cah.combatService.GetAliveSquadsInFaction(currentFactionID)

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
