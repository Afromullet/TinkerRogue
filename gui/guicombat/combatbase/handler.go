package combatbase

import (
	"fmt"

	"game_main/core/coords"
	"game_main/gui/guicombat/combatanimation"
	"game_main/tactical/combat/combatdisposal"
	"game_main/tactical/squads/squadcommands"

	"github.com/bytearena/ecs"
)

// CombatActionHandler manages combat actions and their execution
type CombatActionHandler struct {
	deps            *CombatModeDeps
	commandExecutor *squadcommands.CommandExecutor
}

// NewCombatActionHandler creates a new combat action handler
func NewCombatActionHandler(deps *CombatModeDeps) *CombatActionHandler {
	return &CombatActionHandler{
		deps:            deps,
		commandExecutor: squadcommands.NewCommandExecutor(),
	}
}

func (cah *CombatActionHandler) SelectSquad(squadID ecs.EntityID) {
	cah.deps.BattleState.SelectedSquadID = squadID
}

func (cah *CombatActionHandler) ToggleInspectMode() {
	cah.deps.BattleState.InInspectMode = !cah.deps.BattleState.InInspectMode
}

func (cah *CombatActionHandler) ExitInspectMode() {
	cah.deps.BattleState.InInspectMode = false
}

func (cah *CombatActionHandler) ToggleAttackMode() {
	if cah.deps.BattleState.SelectedSquadID == 0 {
		return
	}
	cah.deps.BattleState.InAttackMode = !cah.deps.BattleState.InAttackMode
}

func (cah *CombatActionHandler) ToggleMoveMode() {
	if cah.deps.BattleState.SelectedSquadID == 0 {
		return
	}

	newMoveMode := !cah.deps.BattleState.InMoveMode

	if newMoveMode {
		validTiles := cah.deps.CombatService.MovementSystem.GetValidMovementTiles(cah.deps.BattleState.SelectedSquadID)
		if len(validTiles) == 0 {
			return
		}
		cah.deps.BattleState.InMoveMode = true
	} else {
		cah.deps.BattleState.InMoveMode = false
	}
}

func (cah *CombatActionHandler) SelectTarget(targetSquadID ecs.EntityID) {
	if !cah.deps.BattleState.InAttackMode {
		return
	}
	cah.deps.BattleState.SelectedTargetID = targetSquadID
	cah.ExecuteAttack()
}

func (cah *CombatActionHandler) SelectEnemyTarget(index int) {
	currentFactionID := cah.deps.CombatService.TurnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return
	}

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

	result := cah.deps.CombatService.CombatActSystem.ExecuteAttackAction(selectedSquad, selectedTarget)

	cah.deps.BattleState.InAttackMode = false

	if !result.Status.Success {
		return
	}

	cah.ClearMoveHistory()

	if cah.deps.ModeManager != nil {
		if animMode, exists := cah.deps.ModeManager.GetMode("combat_animation"); exists {
			if caMode, ok := animMode.(*combatanimation.CombatAnimationMode); ok {
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

func (cah *CombatActionHandler) MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) error {
	movementSystem := cah.deps.CombatService.MovementSystem
	combatCache := cah.deps.CombatService.CombatCache

	cmd := squadcommands.NewMoveSquadCommand(
		cah.deps.Queries.ECSManager,
		movementSystem,
		combatCache,
		squadID,
		newPos,
	)

	result := cah.commandExecutor.Execute(cmd)
	if !result.Success {
		return fmt.Errorf(result.Error)
	}

	cah.deps.BattleState.InMoveMode = false
	return nil
}

func (cah *CombatActionHandler) UndoLastMove() {
	if !cah.CanUndoMove() {
		return
	}
	result := cah.commandExecutor.Undo()
	if result.Success {
		cah.deps.Queries.MarkAllSquadsDirty()
	}
}

func (cah *CombatActionHandler) RedoLastMove() {
	if !cah.CanRedoMove() {
		return
	}
	result := cah.commandExecutor.Redo()
	if result.Success {
		cah.deps.Queries.MarkAllSquadsDirty()
	}
}

func (cah *CombatActionHandler) CanUndoMove() bool {
	return cah.commandExecutor.CanUndo() && !cah.isSelectedSquadActed()
}

func (cah *CombatActionHandler) CanRedoMove() bool {
	return cah.commandExecutor.CanRedo() && !cah.isSelectedSquadActed()
}

func (cah *CombatActionHandler) isSelectedSquadActed() bool {
	selectedSquadID := cah.deps.BattleState.SelectedSquadID
	if selectedSquadID == 0 {
		return false
	}
	actionState := cah.deps.CombatService.CombatCache.FindActionStateBySquadID(selectedSquadID)
	return actionState != nil && actionState.HasActed
}

func (cah *CombatActionHandler) ClearMoveHistory() {
	cah.commandExecutor.ClearHistory()
}

func (cah *CombatActionHandler) CycleSquadSelection() {
	currentFactionID := cah.deps.CombatService.TurnManager.GetCurrentFaction()
	factionData := cah.deps.Queries.CombatCache.FindFactionDataByID(currentFactionID)
	if currentFactionID == 0 || factionData == nil || !factionData.IsPlayerControlled {
		return
	}

	aliveSquads := cah.deps.CombatService.GetAliveSquadsInFaction(currentFactionID)
	if len(aliveSquads) == 0 {
		return
	}

	currentIndex := -1
	selectedSquad := cah.deps.BattleState.SelectedSquadID
	for i, squadID := range aliveSquads {
		if squadID == selectedSquad {
			currentIndex = i
			break
		}
	}

	nextIndex := (currentIndex + 1) % len(aliveSquads)
	cah.SelectSquad(aliveSquads[nextIndex])
}

func (cah *CombatActionHandler) DebugKillSquad(squadID ecs.EntityID) {
	if cah.deps.BattleState.SelectedSquadID == squadID {
		cah.deps.BattleState.SelectedSquadID = 0
	}

	if err := combatdisposal.RemoveSquadFromMap(squadID, cah.deps.Queries.ECSManager); err != nil {
		return
	}

	cah.deps.Queries.InvalidateSquad(squadID)
}
