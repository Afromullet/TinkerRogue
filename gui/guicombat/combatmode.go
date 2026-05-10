package guicombat

import (
	"fmt"

	"game_main/core/common"
	"game_main/core/config"
	"game_main/gui/framework"
	"game_main/gui/guiartifacts"
	"game_main/gui/guicombat/combatbase"
	"game_main/gui/guicombat/combatinput"
	"game_main/gui/guicombat/combatvisualization"
	"game_main/gui/guispells"
	"game_main/gui/guisquads"
	"game_main/gui/widgets"
	"game_main/mind/combatlifecycle"
	"game_main/mind/encounter"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combat/combatservices"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatMode provides focused UI for turn-based squad combat.
// Uses Panel Registry for declarative panel building and type-safe widget access.
type CombatMode struct {
	framework.BaseMode // Embed common mode infrastructure

	// Dependencies (consolidated for all handlers)
	deps *combatbase.CombatModeDeps

	// Managers
	actionHandler *combatbase.CombatActionHandler
	inputHandler  *combatinput.CombatInputHandler
	combatService *combatservices.CombatService

	// Update components (stored for refresh calls)
	squadDetailComponent *guisquads.DetailPanelComponent
	factionInfoComponent *guisquads.DetailPanelComponent
	turnOrderComponent   *widgets.TextDisplayComponent

	// Spell panel controller (owns spell selection UI + handler)
	spellPanel *guispells.SpellPanelController

	// Artifact panel controller (owns artifact activation UI + handler)
	artifactPanel *guiartifacts.ArtifactPanelController

	// Visualization systems
	visualization *combatvisualization.CombatVisualizationManager

	// Sub-menu controller (manages debug, spell, artifact, inspect sub-menu visibility)
	subMenus *framework.SubMenuController

	// Turn lifecycle management
	turnFlow *CombatTurnFlow

	// Input action map
	actionMap *framework.ActionMap

	// GUI's port onto EncounterService (stored until deps are created in Initialize)
	encounterController encounter.EncounterController

	// Factory for creating a fully-wired CombatService (with AI injected)
	serviceFactory func(*common.EntityManager) *combatservices.CombatService

	// State tracking for UI updates (GUI_PERFORMANCE_ANALYSIS.md)
	lastFactionID     ecs.EntityID
	lastSelectedSquad ecs.EntityID
}

func NewCombatMode(modeManager *framework.UIModeManager, encounterController encounter.EncounterController, serviceFactory func(*common.EntityManager) *combatservices.CombatService) *CombatMode {
	cm := &CombatMode{
		encounterController: encounterController,
		serviceFactory:      serviceFactory,
	}
	cm.SetModeName("combat")
	cm.SetReturnMode("exploration")
	cm.SetSelf(cm) // Enable panel registry building
	cm.ModeManager = modeManager
	return cm
}

// GetActionMap implements framework.ActionMapProvider.
func (cm *CombatMode) GetActionMap() *framework.ActionMap {
	return cm.actionMap
}

func (cm *CombatMode) Enter(fromMode framework.UIMode) error {
	isComingFromAnimation := fromMode != nil && fromMode.GetModeName() == "combat_animation"
	shouldInitialize := !isComingFromAnimation

	if shouldInitialize {
		// Reset stale caches from previous combat
		cm.Queries.ClearSquadCache()
		cm.visualization.ResetHighlightColors()

		// Re-register callbacks (cleared by previous TeardownCombat)
		cm.registerCombatCallbacks()

		// Refresh threat manager with newly created factions (must be after SpawnCombatEntities)
		cm.visualization.RefreshFactions(cm.Queries)

		// Start battle recording if enabled
		if config.ENABLE_COMBAT_LOG_EXPORT && cm.combatService.BattleRecorder != nil {
			cm.combatService.BattleRecorder.SetEnabled(true)
			cm.combatService.BattleRecorder.Start()
			cm.combatService.BattleRecorder.SetCurrentRound(1)
		}

		// Initialize combat factions
		factionIDs := cm.Queries.GetAllFactions()
		if len(factionIDs) > 0 {
			if err := cm.combatService.InitializeCombat(factionIDs); err != nil {
				return fmt.Errorf("error initializing combat factions: %w", err)
			}
		}
	}

	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	if currentFactionID != 0 {
		cm.turnOrderComponent.Refresh()
		cm.factionInfoComponent.ShowFaction(currentFactionID)
	}

	return nil
}

func (cm *CombatMode) Exit(toMode framework.UIMode) error {
	isToAnimation := toMode != nil && toMode.GetModeName() == "combat_animation"

	if !isToAnimation {
		victor := cm.combatService.GetExitResult()
		reason := combatlifecycle.DetermineExitReason(cm.combatService.IsFleeRequested(), victor.IsPlayerVictory)

		// Single call handles: overworld resolution, history recording, entity cleanup
		if cm.deps.Encounter != nil {
			cm.deps.Encounter.ExitCombat(reason,
				&combatlifecycle.EncounterOutcome{
					IsPlayerVictory:  victor.IsPlayerVictory,
					VictorFaction:    victor.VictorFaction,
					VictorName:       victor.VictorName,
					RoundsCompleted:  victor.RoundsCompleted,
					DefeatedFactions: victor.DefeatedFactions,
				},
				cm.combatService)
		}

		// Export battle log if enabled (GUI-only concern, stays here)
		if config.ENABLE_COMBAT_LOG_EXPORT && cm.combatService.BattleRecorder != nil && cm.combatService.BattleRecorder.IsEnabled() {
			victoryInfo := &battlelog.VictoryInfo{
				RoundsCompleted: victor.RoundsCompleted,
				VictorFaction:   victor.VictorFaction,
				VictorName:      victor.VictorName,
			}
			record := cm.combatService.BattleRecorder.Finalize(victoryInfo)
			if err := battlelog.ExportBattleJSON(record, config.COMBAT_LOG_EXPORT_DIR); err != nil {
				fmt.Printf("Error exporting battle log: %v\n", err)
			}
			cm.combatService.BattleRecorder.Clear()
			cm.combatService.BattleRecorder.SetEnabled(false)
		}

		// Clear cached victory/flee state
		cm.combatService.ClearExitState()
	}

	cm.visualization.ClearAllVisualizations()
	return nil
}

func (cm *CombatMode) Update(deltaTime float64) error {
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	if cm.lastFactionID != currentFactionID {
		cm.turnOrderComponent.Refresh()
		cm.lastFactionID = currentFactionID
		if cm.lastFactionID != 0 {
			cm.factionInfoComponent.ShowFaction(cm.lastFactionID)
		}
	}

	battleState := cm.Context.ModeCoordinator.GetTacticalState()
	if cm.lastSelectedSquad != battleState.SelectedSquadID {
		cm.lastSelectedSquad = battleState.SelectedSquadID
		if cm.lastSelectedSquad != 0 {
			cm.squadDetailComponent.ShowSquad(cm.lastSelectedSquad)
		}
	}

	currentRound := cm.combatService.TurnManager.GetCurrentRound()
	playerPos := *cm.Context.PlayerData.Pos
	viewportSize := 30

	cm.visualization.UpdateThreatVisualization(currentFactionID, currentRound, playerPos, viewportSize)

	// Update AoE spell targeting overlay each frame
	if cm.spellPanel != nil && cm.spellPanel.Handler().IsAoETargeting() {
		mouseX, mouseY := ebiten.CursorPosition()
		cm.spellPanel.Handler().HandleAoETargetingFrame(mouseX, mouseY)
	}

	return nil
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
	playerPos := *cm.Context.PlayerData.Pos
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	battleState := cm.Context.ModeCoordinator.GetTacticalState()
	cm.visualization.RenderAll(screen, playerPos, currentFactionID, battleState, cm.combatService)
}

func (cm *CombatMode) HandleInput(inputState *framework.InputState) bool {
	// Combat-specific input first (ESC cancels active modes like spell, artifact, inspect)
	cm.inputHandler.SetPlayerPosition(cm.Context.PlayerData.Pos)
	cm.inputHandler.SetCurrentFactionID(cm.combatService.TurnManager.GetCurrentFaction())
	if cm.inputHandler.HandleInput(inputState) {
		return true
	}

	// Block ESC from reaching HandleCommonInput — use the Flee button instead
	if inputState.ActionActive(framework.ActionCancel) {
		return true
	}

	// Common hotkeys (ESC already consumed above, won't trigger mode exit)
	if cm.HandleCommonInput(inputState) {
		return true
	}

	if inputState.ActionActive(framework.ActionEndTurn) {
		cm.turnFlow.HandleEndTurn()
		return true
	}

	return false
}
