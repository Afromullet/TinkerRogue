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
}

// ToggleAttackMode enables/disables attack mode
func (cah *CombatActionHandler) ToggleAttackMode() {
	if cah.deps.BattleState.SelectedSquadID == 0 {
		return
	}

	cah.deps.BattleState.InAttackMode = !cah.deps.BattleState.InAttackMode
}

// ToggleMoveMode enables/disables move mode
func (cah *CombatActionHandler) ToggleMoveMode() {
	if cah.deps.BattleState.SelectedSquadID == 0 {
		return
	}

	newMoveMode := !cah.deps.BattleState.InMoveMode

	if newMoveMode {
		// Get valid movement tiles
		validTiles := cah.deps.CombatService.MovementSystem.GetValidMovementTiles(cah.deps.BattleState.SelectedSquadID)

		if len(validTiles) == 0 {
			return
		}

		cah.deps.BattleState.InMoveMode = true
	} else {
		cah.deps.BattleState.InMoveMode = false
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
		return
	}

	// Attack is valid - show animation then apply results
	if cah.deps.ModeManager != nil {
		if animMode, exists := cah.deps.ModeManager.GetMode("combat_animation"); exists {
			if caMode, ok := animMode.(*CombatAnimationMode); ok {
				caMode.SetCombatants(selectedSquad, selectedTarget)
				caMode.SetOnComplete(func() {
					// Animation complete - no-op (results already applied)
				})
				cah.deps.ModeManager.RequestTransition(animMode, "Combat animation")
				return
			}
		}
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
		return fmt.Errorf(result.Error)
	}

	// Cache invalidation is handled automatically by the onMoveComplete hook.

	// Exit move mode
	cah.deps.BattleState.InMoveMode = false

	return nil
}

// UndoLastMove undoes the last movement command
func (cah *CombatActionHandler) UndoLastMove() {
	if !cah.commandExecutor.CanUndo() {
		return
	}

	result := cah.commandExecutor.Undo()

	if result.Success {
		// Invalidate all squads since squad positions changed
		cah.deps.Queries.MarkAllSquadsDirty()
	}
}

// RedoLastMove redoes the last undone movement command
func (cah *CombatActionHandler) RedoLastMove() {
	if !cah.commandExecutor.CanRedo() {
		return
	}

	result := cah.commandExecutor.Redo()

	if result.Success {
		// Invalidate all squads since squad positions changed
		cah.deps.Queries.MarkAllSquadsDirty()
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
	// Clear selection if the killed squad was selected
	if cah.deps.BattleState.SelectedSquadID == squadID {
		cah.deps.BattleState.SelectedSquadID = 0
	}

	// Use the same cleanup path as normal combat death
	if err := combat.RemoveSquadFromMap(squadID, cah.deps.Queries.ECSManager); err != nil {
		return
	}

	// Invalidate cache for the removed squad
	cah.deps.Queries.InvalidateSquad(squadID)
}
