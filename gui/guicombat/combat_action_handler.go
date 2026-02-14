package guicombat

import (
	"fmt"
	"game_main/tactical/combat"
	"game_main/tactical/squadcommands"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CombatActionHandler manages combat actions and their execution
type CombatActionHandler struct {
	deps            *CombatModeDeps
	commandExecutor *squadcommands.CommandExecutor // Command pattern for undo/redo
}

// NewCombatActionHandler creates a new combat action handler
func NewCombatActionHandler(deps *CombatModeDeps) *CombatActionHandler {
	return &CombatActionHandler{
		deps:            deps,
		commandExecutor: squadcommands.NewCommandExecutor(),
	}
}

// SelectSquad selects a squad and logs the action
func (cah *CombatActionHandler) SelectSquad(squadID ecs.EntityID) {
	cah.deps.BattleState.SelectedSquadID = squadID

	// Get squad name and log
	squadName := cah.deps.Queries.SquadCache.GetSquadName(squadID)
	cah.addLog(fmt.Sprintf("Selected: %s", squadName))
}

// ToggleAttackMode enables/disables attack mode
func (cah *CombatActionHandler) ToggleAttackMode() {
	if cah.deps.BattleState.SelectedSquadID == 0 {
		cah.addLog("Select a squad first!")
		return
	}

	newAttackMode := !cah.deps.BattleState.InAttackMode
	cah.deps.BattleState.InAttackMode = newAttackMode

	if newAttackMode {
		cah.addLog("Attack mode")

	} else {
		cah.addLog("Attack mode cancelled")
	}
}

// ToggleMoveMode enables/disables move mode
func (cah *CombatActionHandler) ToggleMoveMode() {
	if cah.deps.BattleState.SelectedSquadID == 0 {
		cah.addLog("Select a squad first!")
		return
	}

	newMoveMode := !cah.deps.BattleState.InMoveMode

	if newMoveMode {
		// Get valid movement tiles
		validTiles := cah.deps.CombatService.MovementSystem.GetValidMovementTiles(cah.deps.BattleState.SelectedSquadID)

		if len(validTiles) == 0 {
			cah.addLog("No movement remaining!")
			return
		}

		cah.deps.BattleState.InMoveMode = true
		cah.addLog(fmt.Sprintf("Move mode: Click a tile (%d tiles available)", len(validTiles)))
		cah.addLog("Click on the map to move, or press M to cancel")
	} else {
		cah.deps.BattleState.InMoveMode = false
		cah.addLog("Move mode cancelled")
	}
}

// SelectTarget selects a target squad for attack
func (cah *CombatActionHandler) SelectTarget(targetSquadID ecs.EntityID) {
	if !cah.deps.BattleState.InAttackMode {
		return
	}

	cah.deps.BattleState.SelectedTargetID = targetSquadID
	cah.ExecuteAttack()
}

// SelectEnemyTarget selects an enemy squad by index (0-2 for 1-3 keys)
func (cah *CombatActionHandler) SelectEnemyTarget(index int) {
	currentFactionID := cah.deps.CombatService.TurnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return
	}

	// Get all enemy squads
	enemySquads := cah.deps.Queries.GetEnemySquads(currentFactionID)

	if index < 0 || index >= len(enemySquads) {
		cah.addLog(fmt.Sprintf("No enemy squad at index %d", index+1))
		return
	}

	cah.SelectTarget(enemySquads[index])
}

func (cah *CombatActionHandler) ExecuteAttack() {
	selectedSquad := cah.deps.BattleState.SelectedSquadID
	selectedTarget := cah.deps.BattleState.SelectedTargetID

	if selectedSquad == 0 || selectedTarget == 0 {
		return
	}

	// Validate attack BEFORE triggering animation (prevents animation on invalid attacks)
	// Cache invalidation is handled automatically by the onAttackComplete hook.
	result := cah.deps.CombatService.CombatActSystem.ExecuteAttackAction(selectedSquad, selectedTarget)

	// Reset UI state
	cah.deps.BattleState.InAttackMode = false

	// Handle invalid attacks immediately (no animation)
	if !result.Success {
		cah.addLog(fmt.Sprintf("Cannot attack: %s", result.ErrorReason))
		return
	}

	// Attack is valid - show animation then apply results
	if cah.deps.ModeManager != nil {
		if animMode, exists := cah.deps.ModeManager.GetMode("combat_animation"); exists {
			if caMode, ok := animMode.(*CombatAnimationMode); ok {
				caMode.SetCombatants(selectedSquad, selectedTarget)
				caMode.SetOnComplete(func() {
					// Animation complete - show attack results
					attackerName := "Unknown"
					defenderName := "Unknown"
					if result.CombatLog != nil {
						attackerName = result.CombatLog.AttackerSquadName
						defenderName = result.CombatLog.DefenderSquadName
					}
					cah.addLog(fmt.Sprintf("%s attacked %s!", attackerName, defenderName))
					if result.TargetDestroyed {
						cah.addLog(fmt.Sprintf("%s was destroyed!", defenderName))
					}
				})
				cah.deps.ModeManager.RequestTransition(animMode, "Combat animation")
				return
			}
		}
	}

	// Fallback: no animation mode, just show results
	attackerName := "Unknown"
	defenderName := "Unknown"
	if result.CombatLog != nil {
		attackerName = result.CombatLog.AttackerSquadName
		defenderName = result.CombatLog.DefenderSquadName
	}
	cah.addLog(fmt.Sprintf("%s attacked %s!", attackerName, defenderName))
	if result.TargetDestroyed {
		cah.addLog(fmt.Sprintf("%s was destroyed!", defenderName))
	}
}

// MoveSquad moves a squad to a new position using command pattern for undo support
func (cah *CombatActionHandler) MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) error {
	// Get movement system and cache from combat service
	movementSystem := cah.deps.CombatService.MovementSystem
	combatCache := cah.deps.CombatService.CombatCache

	// Create move command with system reference
	cmd := squadcommands.NewMoveSquadCommand(
		cah.deps.Queries.ECSManager,
		movementSystem,
		combatCache,
		squadID,
		newPos,
	)

	// Execute via command executor
	result := cah.commandExecutor.Execute(cmd)

	if !result.Success {
		cah.addLog(fmt.Sprintf("Movement failed: %s", result.Error))
		return fmt.Errorf(result.Error)
	}

	// Cache invalidation is handled automatically by the onMoveComplete hook.
	cah.addLog(fmt.Sprintf("✓ %s", result.Description))

	// Exit move mode
	cah.deps.BattleState.InMoveMode = false

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
		cah.deps.Queries.MarkAllSquadsDirty()
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
		cah.deps.Queries.MarkAllSquadsDirty()
		cah.addLog(fmt.Sprintf("⟳ Redid: %s", result.Description))
	} else {
		cah.addLog(fmt.Sprintf("Redo failed: %s", result.Error))
	}
}

// CanUndoMove returns whether there are moves to undo
// TODO, add logic here.
func (cah *CombatActionHandler) CanUndoMove() bool {
	return cah.commandExecutor.CanUndo()
}

// CanRedoMove returns whether there are moves to redo
// // TODO, add logic here.
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
	currentFactionID := cah.deps.CombatService.TurnManager.GetCurrentFaction()
	factionData := cah.deps.Queries.CombatCache.FindFactionDataByID(currentFactionID)
	if currentFactionID == 0 || factionData == nil || !factionData.IsPlayerControlled {
		return
	}

	// Get alive squads using service
	aliveSquads := cah.deps.CombatService.GetAliveSquadsInFaction(currentFactionID)

	if len(aliveSquads) == 0 {
		return
	}

	// Find current index
	currentIndex := -1
	selectedSquad := cah.deps.BattleState.SelectedSquadID
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

// DebugKillSquad removes a squad from the combat map using the normal death cleanup flow.
func (cah *CombatActionHandler) DebugKillSquad(squadID ecs.EntityID) {
	// Get squad name before disposal
	squadName := cah.deps.Queries.SquadCache.GetSquadName(squadID)

	// Clear selection if the killed squad was selected
	if cah.deps.BattleState.SelectedSquadID == squadID {
		cah.deps.BattleState.SelectedSquadID = 0
	}

	// Use the same cleanup path as normal combat death
	if err := combat.RemoveSquadFromMap(squadID, cah.deps.Queries.ECSManager); err != nil {
		cah.addLog(fmt.Sprintf("[DEBUG] Kill failed: %v", err))
		return
	}

	// Invalidate cache for the removed squad
	cah.deps.Queries.InvalidateSquad(squadID)

	cah.addLog(fmt.Sprintf("[DEBUG] Killed squad: %s", squadName))
}

// addLog adds a message to the combat log
func (cah *CombatActionHandler) addLog(message string) {
	cah.deps.AddCombatLog(message)
}
