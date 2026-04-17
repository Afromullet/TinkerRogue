package guicombat

import (
	"game_main/gui/framework"
	"game_main/gui/guisquads"
	"game_main/gui/widgets"
	"game_main/tactical/combat/combatservices"
)

// CombatTurnFlow manages the turn lifecycle: ending turns, executing AI turns,
// chaining AI attack animations, and checking victory conditions.
type CombatTurnFlow struct {
	combatService *combatservices.CombatService
	visualization *CombatVisualizationManager
	actionHandler *CombatActionHandler
	queries       *framework.GUIQueries
	modeManager   *framework.UIModeManager
	panels        *framework.PanelRegistry
	context       *framework.UIContext

	// UI components for refresh
	turnOrderComponent   *widgets.TextDisplayComponent
	factionInfoComponent *guisquads.DetailPanelComponent
	squadDetailComponent *guisquads.DetailPanelComponent
}

// NewCombatTurnFlow creates a new turn flow manager
func NewCombatTurnFlow(
	combatService *combatservices.CombatService,
	visualization *CombatVisualizationManager,
	actionHandler *CombatActionHandler,
	queries *framework.GUIQueries,
	modeManager *framework.UIModeManager,
	panels *framework.PanelRegistry,
	context *framework.UIContext,
) *CombatTurnFlow {
	return &CombatTurnFlow{
		combatService: combatService,
		visualization: visualization,
		actionHandler: actionHandler,
		queries:       queries,
		modeManager:   modeManager,
		panels:        panels,
		context:       context,
	}
}

// SetUIComponents sets the UI component references needed for refreshing panels
func (tf *CombatTurnFlow) SetUIComponents(
	turnOrder *widgets.TextDisplayComponent,
	factionInfo *guisquads.DetailPanelComponent,
	squadDetail *guisquads.DetailPanelComponent,
) {
	tf.turnOrderComponent = turnOrder
	tf.factionInfoComponent = factionInfo
	tf.squadDetailComponent = squadDetail
}

// HandleEndTurn ends the current player turn and advances to the next faction
func (tf *CombatTurnFlow) HandleEndTurn() {
	tf.actionHandler.ClearMoveHistory()

	if !tf.completeTurn() {
		return
	}

	tf.context.ModeCoordinator.GetTacticalState().Reset()
	tf.squadDetailComponent.SetText("Select a squad\nto view details")

	tf.executeAITurnIfNeeded()
}

// HandleFlee handles the player requesting to flee from combat
func (tf *CombatTurnFlow) HandleFlee() {
	rounds := tf.combatService.TurnManager.GetCurrentRound()
	tf.combatService.MarkFleeRequested()
	tf.combatService.CacheVictoryResult(&combatservices.VictoryCheckResult{
		BattleOver:      true,
		IsPlayerVictory: false,
		VictorName:      "Retreat",
		RoundsCompleted: rounds,
	})

	returnModeName := tf.getPostCombatReturnMode()
	if returnMode, exists := tf.modeManager.GetMode(returnModeName); exists {
		tf.modeManager.RequestTransition(returnMode, "Fled from combat")
	}
}

// CheckAndHandleVictory checks if combat has ended and handles the transition.
// Returns true if combat ended (victory or defeat), false if combat continues.
// Caches the victory result to avoid redundant checks during cleanup.
func (tf *CombatTurnFlow) CheckAndHandleVictory() bool {
	result := tf.combatService.CheckVictoryCondition()

	if !result.BattleOver {
		return false
	}

	// Cache the result for use in Exit() to avoid redundant checks
	tf.combatService.CacheVictoryResult(result)

	// Transition to post-combat mode (raid or exploration)
	returnModeName := tf.getPostCombatReturnMode()
	if returnMode, exists := tf.modeManager.GetMode(returnModeName); exists {
		tf.modeManager.RequestTransition(returnMode, "Combat ended - "+result.VictorName+" victorious")
	}

	return true
}

// executeAITurnIfNeeded checks if current faction is AI-controlled and executes its turn
func (tf *CombatTurnFlow) executeAITurnIfNeeded() {
	currentFactionID := tf.combatService.TurnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return
	}

	factionData := tf.queries.CombatCache.FindFactionDataByID(currentFactionID)
	if factionData == nil || factionData.IsPlayerControlled {
		return
	}

	aiController := tf.combatService.GetAIController()
	// Cache invalidation for destroyed squads is handled automatically by the onAttackComplete hook.
	aiController.DecideFactionTurn(currentFactionID)

	if aiController.HasQueuedAttacks() {
		tf.playAIAttackAnimations(aiController)
		return
	}

	tf.advanceAfterAITurn()
}

// playAIAttackAnimations plays all queued AI attack animations sequentially
func (tf *CombatTurnFlow) playAIAttackAnimations(aiController combatservices.AITurnController) {
	attacks := aiController.GetQueuedAttacks()

	if len(attacks) == 0 {
		tf.advanceAfterAITurn()
		return
	}

	tf.playNextAIAttack(attacks, 0, aiController)
}

// playNextAIAttack plays a single AI attack animation and chains to the next
func (tf *CombatTurnFlow) playNextAIAttack(attacks []combatservices.QueuedAttack, index int, aiController combatservices.AITurnController) {
	if index >= len(attacks) {
		aiController.ClearAttackQueue()

		if combatMode, exists := tf.modeManager.GetMode("combat"); exists {
			tf.modeManager.RequestTransition(combatMode, "AI attacks complete")
		}

		tf.advanceAfterAITurn()
		return
	}

	attack := attacks[index]
	isFirstAttack := (index == 0)

	if animMode, exists := tf.modeManager.GetMode("combat_animation"); exists {
		if caMode, ok := animMode.(*CombatAnimationMode); ok {
			caMode.SetCombatants(attack.AttackerID, attack.DefenderID)
			caMode.SetAutoPlay(true)

			caMode.SetOnComplete(func() {
				caMode.ResetForNextAttack()
				tf.playNextAIAttack(attacks, index+1, aiController)
			})

			if isFirstAttack {
				tf.modeManager.RequestTransition(animMode, "AI Attack Animation")
			}
		} else {
			aiController.ClearAttackQueue()
			tf.advanceAfterAITurn()
		}
	} else {
		aiController.ClearAttackQueue()
		tf.advanceAfterAITurn()
	}
}

// advanceAfterAITurn advances to next turn after AI completes
func (tf *CombatTurnFlow) advanceAfterAITurn() {
	if !tf.completeTurn() {
		return
	}
	tf.executeAITurnIfNeeded()
}

// completeTurn ends the current turn, checks victory, and refreshes UI.
// Returns false if the turn could not end or combat ended (victory/defeat).
func (tf *CombatTurnFlow) completeTurn() bool {
	if err := tf.combatService.TurnManager.EndTurn(); err != nil {
		return false
	}

	if tf.CheckAndHandleVictory() {
		return false
	}

	currentFactionID := tf.combatService.TurnManager.GetCurrentFaction()
	round := tf.combatService.TurnManager.GetCurrentRound()

	if tf.combatService.BattleRecorder != nil && tf.combatService.BattleRecorder.IsEnabled() {
		tf.combatService.BattleRecorder.SetCurrentRound(round)
	}

	tf.turnOrderComponent.Refresh()
	tf.factionInfoComponent.ShowFaction(currentFactionID)

	return true
}

// getPostCombatReturnMode returns the mode to transition to after combat ends.
// If PostCombatReturnMode is set on TacticalState (e.g., "raid"), uses that.
// Otherwise defaults to "exploration".
func (tf *CombatTurnFlow) getPostCombatReturnMode() string {
	if tf.context != nil && tf.context.ModeCoordinator != nil {
		tacticalState := tf.context.ModeCoordinator.GetTacticalState()
		if tacticalState.PostCombatReturnMode != "" {
			return tacticalState.PostCombatReturnMode
		}
	}
	return "exploration"
}
