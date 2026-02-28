package guicombat

import (
	"fmt"
	"game_main/common"
	"game_main/gui/framework"
	"game_main/gui/guiartifacts"
	"game_main/gui/guiinspect"
	"game_main/gui/guispells"
	"game_main/mind/behavior"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/visual/graphics"
	"game_main/world/coords"
	"time"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

const doubleClickThreshold = 300 * time.Millisecond

// CombatInputHandler manages all input processing for combat mode.
// Uses CombatModeDeps for shared state and services.
type CombatInputHandler struct {
	actionHandler    *CombatActionHandler
	deps             *CombatModeDeps
	playerPos        *coords.LogicalPosition
	currentFactionID ecs.EntityID

	// Debug kill mode
	inDebugKillMode bool

	// Spell panel controller (owns handler + panel UI)
	spellPanel *guispells.SpellPanelController

	// Artifact panel controller (owns handler + panel UI)
	artifactPanel *guiartifacts.ArtifactPanelController

	// Visualization input support
	visualization *CombatVisualizationManager
	panels        *framework.PanelRegistry

	// Inspect panel controller (owns formation display UI)
	inspectPanel *guiinspect.InspectPanelController

	// Double-click tracking
	lastClickTime    time.Time
	lastClickSquadID ecs.EntityID
}

// NewCombatInputHandler creates a new combat input handler
func NewCombatInputHandler(actionHandler *CombatActionHandler, deps *CombatModeDeps) *CombatInputHandler {
	return &CombatInputHandler{
		actionHandler: actionHandler,
		deps:          deps,
	}
}

// SetPlayerPosition sets the player position for viewport calculations
func (cih *CombatInputHandler) SetPlayerPosition(playerPos *coords.LogicalPosition) {
	cih.playerPos = playerPos
}

// SetCurrentFactionID sets the current faction ID for turn checking
func (cih *CombatInputHandler) SetCurrentFactionID(factionID ecs.EntityID) {
	cih.currentFactionID = factionID
}

// SetSpellPanel sets the spell panel controller for input delegation.
func (cih *CombatInputHandler) SetSpellPanel(panel *guispells.SpellPanelController) {
	cih.spellPanel = panel
}

// SetArtifactPanel sets the artifact panel controller for input delegation.
func (cih *CombatInputHandler) SetArtifactPanel(panel *guiartifacts.ArtifactPanelController) {
	cih.artifactPanel = panel
}

// SetInspectPanel sets the inspect panel controller for formation display.
func (cih *CombatInputHandler) SetInspectPanel(panel *guiinspect.InspectPanelController) {
	cih.inspectPanel = panel
}

// SetVisualization sets the visualization manager and related dependencies for keybinding handling
func (cih *CombatInputHandler) SetVisualization(viz *CombatVisualizationManager, panels *framework.PanelRegistry) {
	cih.visualization = viz
	cih.panels = panels
}

// EnterDebugKillMode activates click-to-kill mode for debug purposes.
func (cih *CombatInputHandler) EnterDebugKillMode() {
	cih.inDebugKillMode = true
}

// handleDebugKillClick processes a click while in debug kill mode.
func (cih *CombatInputHandler) handleDebugKillClick(mouseX, mouseY int) {
	if cih.playerPos == nil {
		return
	}

	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)
	clickedSquadID := combat.GetSquadAtPosition(clickedPos, cih.deps.Queries.ECSManager)

	if clickedSquadID == 0 {
		return
	}

	cih.actionHandler.DebugKillSquad(clickedSquadID)
}

// HandleInput processes input and returns true if input was consumed
func (cih *CombatInputHandler) HandleInput(inputState *framework.InputState) bool {
	// ESC cancels debug kill mode
	if cih.inDebugKillMode && inputState.ActionActive(framework.ActionCancel) {
		cih.inDebugKillMode = false
		return true
	}

	// Spell mode input handling (takes priority over normal clicks)
	if cih.spellPanel != nil && cih.spellPanel.Handler().IsInSpellMode() {
		handler := cih.spellPanel.Handler()

		// ESC cancels spell mode
		if inputState.ActionActive(framework.ActionCancel) {
			cih.spellPanel.OnCancelClicked()
			return true
		}

		if handler.HasSelectedSpell() {
			// Spell is selected - AoE shape rotation
			if handler.IsAoETargeting() {
				if inputState.ActionActive(framework.ActionAoERotateLeft) {
					handler.RotateShapeLeft()
					return true
				}
				if inputState.ActionActive(framework.ActionAoERotateRight) {
					handler.RotateShapeRight()
					return true
				}
			}

			// Click to cast
			if inputState.MouseJustPressedButton(ebiten.MouseButtonLeft) {
				if handler.IsAoETargeting() {
					handler.HandleAoEConfirmClick(inputState.MouseX, inputState.MouseY)
				} else {
					handler.HandleSingleTargetClick(inputState.MouseX, inputState.MouseY)
				}
				return true
			}
		}
		// Suppress other input while in spell mode
		return false
	}

	// Artifact mode input handling (takes priority over normal clicks)
	if cih.artifactPanel != nil && cih.artifactPanel.Handler().IsInArtifactMode() {
		handler := cih.artifactPanel.Handler()

		// ESC cancels artifact mode
		if inputState.ActionActive(framework.ActionCancel) {
			cih.artifactPanel.OnCancelClicked()
			return true
		}

		if handler.HasSelectedArtifact() {
			// Artifact selected and targeting - click to apply
			if inputState.MouseJustPressedButton(ebiten.MouseButtonLeft) {
				handler.HandleTargetClick(inputState.MouseX, inputState.MouseY)
				return true
			}
		}
		// Suppress other input while in artifact mode
		return false
	}

	// Inspect mode input handling (takes priority over normal clicks)
	if cih.deps.BattleState.InInspectMode {
		// ESC exits inspect mode
		if inputState.ActionActive(framework.ActionCancel) {
			cih.exitInspectMode()
			return true
		}

		// Left-click inspects squad at position
		if inputState.MouseJustPressedButton(ebiten.MouseButtonLeft) {
			cih.handleInspectClick(inputState.MouseX, inputState.MouseY)
			return true
		}

		// Suppress other input while in inspect mode
		return false
	}

	// Right-click exits move mode
	if cih.deps.BattleState.InMoveMode && inputState.MouseJustPressedButton(ebiten.MouseButtonRight) {
		cih.actionHandler.ToggleMoveMode()
		return true
	}

	// Handle mouse clicks (edge-detected: fires once per press)
	if inputState.MouseJustPressedButton(ebiten.MouseButtonLeft) {
		// Debug kill mode takes priority over all other click handling
		if cih.inDebugKillMode {
			defer func() { cih.inDebugKillMode = false }()
			cih.handleDebugKillClick(inputState.MouseX, inputState.MouseY)
			return true
		}

		if cih.deps.BattleState.InMoveMode {
			// In move mode: click to move squad
			cih.handleMovementClick(inputState.MouseX, inputState.MouseY)
		} else {
			// Not in move mode: click to select/attack squad
			cih.handleSquadClick(inputState.MouseX, inputState.MouseY)
		}
		return true
	}

	// Space to end turn - handled at CombatMode level
	if inputState.ActionActive(framework.ActionEndTurn) {
		return false
	}

	// S key to toggle spell panel
	if inputState.ActionActive(framework.ActionSpellPanel) && cih.spellPanel != nil {
		cih.spellPanel.Toggle()
		return true
	}

	// D key to toggle artifact panel
	if inputState.ActionActive(framework.ActionArtifactPanel) && cih.artifactPanel != nil {
		cih.artifactPanel.Toggle()
		return true
	}

	// A key to toggle attack mode
	if inputState.ActionActive(framework.ActionAttackMode) {
		cih.actionHandler.ToggleAttackMode()
		return true
	}

	// M key to toggle move mode
	if inputState.ActionActive(framework.ActionMoveMode) {
		cih.actionHandler.ToggleMoveMode()
		return true
	}

	// I key to toggle inspect mode
	if inputState.ActionActive(framework.ActionInspectMode) {
		// Close spell/artifact panels if active
		if cih.spellPanel != nil && cih.spellPanel.Handler().IsInSpellMode() {
			cih.spellPanel.OnCancelClicked()
		}
		if cih.artifactPanel != nil && cih.artifactPanel.Handler().IsInArtifactMode() {
			cih.artifactPanel.OnCancelClicked()
		}
		cih.toggleInspectMode()
		return true
	}

	// TAB to cycle through squads
	if inputState.ActionActive(framework.ActionCycleSquad) {
		cih.actionHandler.CycleSquadSelection()
		return true
	}

	// Ctrl+Z to undo last move
	if inputState.ActionActive(framework.ActionUndoMove) {
		cih.actionHandler.UndoLastMove()
		return true
	}

	// Number keys 1-3 to select enemy targets in attack mode
	if cih.deps.BattleState.InAttackMode {
		if inputState.ActionActive(framework.ActionSelectTarget1) {
			cih.actionHandler.SelectEnemyTarget(0)
			return true
		}
		if inputState.ActionActive(framework.ActionSelectTarget2) {
			cih.actionHandler.SelectEnemyTarget(1)
			return true
		}
		if inputState.ActionActive(framework.ActionSelectTarget3) {
			cih.actionHandler.SelectEnemyTarget(2)
			return true
		}
	}

	// Visualization keybindings
	if cih.handleThreatToggle(inputState) {
		return true
	}
	if cih.handleHealthBarToggle(inputState) {
		return true
	}
	if cih.handleLayerToggle(inputState) {
		return true
	}

	// Ctrl+K to kill all enemy squads (debug command)
	if inputState.ActionActive(framework.ActionDebugKillAll) {
		println("[DEBUG] Calling killAllEnemySquads()")
		cih.killAllEnemySquads()
		return true
	}

	return false
}

// handleMovementClick processes clicks when in move mode
func (cih *CombatInputHandler) handleMovementClick(mouseX, mouseY int) {
	// Convert mouse coordinates to tile coordinates
	if cih.playerPos == nil {
		return
	}

	// Convert screen coordinates to logical coordinates (handles both scrolling modes)
	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)

	// Compute valid tiles on-demand from movement system
	validTiles := cih.deps.CombatService.MovementSystem.GetValidMovementTiles(cih.deps.BattleState.SelectedSquadID)
	if validTiles == nil {
		validTiles = []coords.LogicalPosition{}
	}

	// Check if clicked position is in valid movement tiles
	isValidTile := false
	for _, validPos := range validTiles {
		if validPos.X == clickedPos.X && validPos.Y == clickedPos.Y {
			isValidTile = true
			break
		}
	}
	if !isValidTile {
		return // Invalid tile, do nothing
	}

	selectedSquad := cih.deps.BattleState.SelectedSquadID
	if selectedSquad == 0 {
		return
	}

	// Execute movement through action handler
	cih.actionHandler.MoveSquad(selectedSquad, clickedPos)
}

// handleSquadClick processes clicks when not in move mode
func (cih *CombatInputHandler) handleSquadClick(mouseX, mouseY int) {
	// Convert mouse coordinates to tile coordinates
	if cih.playerPos == nil {
		return
	}

	// Convert screen coordinates to logical coordinates (handles both scrolling modes)
	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)

	// Find if a squad is at the clicked position
	clickedSquadID := combat.GetSquadAtPosition(clickedPos, cih.deps.Queries.ECSManager)

	// If no squad was clicked, reset double-click tracking and do nothing
	if clickedSquadID == 0 {
		cih.lastClickTime = time.Time{}
		cih.lastClickSquadID = 0
		return
	}

	// Get faction info for the clicked squad
	squadInfo := cih.deps.Queries.GetSquadInfo(clickedSquadID)
	if squadInfo == nil {
		return
	}
	clickedFactionID := squadInfo.FactionID

	// If no faction currently active, do nothing
	if cih.currentFactionID == 0 {
		return
	}

	// If it's the player's turn
	factionData := cih.deps.Queries.CombatCache.FindFactionDataByID(cih.currentFactionID)
	if factionData != nil && factionData.IsPlayerControlled {
		// If clicking an allied squad: select it (+ double-click enters move mode)
		if clickedFactionID == cih.currentFactionID {
			now := time.Now()
			isDoubleClick := clickedSquadID == cih.lastClickSquadID &&
				now.Sub(cih.lastClickTime) <= doubleClickThreshold

			// Always select on single click
			cih.actionHandler.SelectSquad(clickedSquadID)

			if isDoubleClick {
				// Double-click: enter move mode (but don't toggle off if already in it)
				if !cih.deps.BattleState.InMoveMode {
					cih.actionHandler.ToggleMoveMode()
				}
				// Reset to prevent triple-click toggling
				cih.lastClickTime = time.Time{}
				cih.lastClickSquadID = 0
			} else {
				// Record for potential double-click
				cih.lastClickTime = now
				cih.lastClickSquadID = clickedSquadID
			}
			return
		}

		// If clicking an enemy squad and we have a selected squad: attack immediately
		selectedSquad := cih.deps.BattleState.SelectedSquadID
		if selectedSquad != 0 && clickedFactionID != cih.currentFactionID {
			cih.deps.BattleState.SelectedTargetID = clickedSquadID
			cih.actionHandler.ExecuteAttack()
		}
		// Reset double-click tracking on enemy clicks
		cih.lastClickTime = time.Time{}
		cih.lastClickSquadID = 0
	}
}

// killAllEnemySquads is a debug function that instantly kills all units in all enemy squads.
// This is useful for quickly testing victory conditions.
// Only affects squads in the current encounter (not all squads in the ECS).
func (cih *CombatInputHandler) killAllEnemySquads() {
	// Get current encounter ID to filter only this encounter's squads
	encounterID := cih.deps.EncounterService.GetCurrentEncounterID()
	if encounterID == 0 {
		println("[DEBUG] No current encounter, cannot kill enemies")
		return
	}

	// CRITICAL: Must use PLAYER faction ID, not current turn faction ID
	// If we use currentFactionID and it's the AI's turn, we'd kill player squads instead!
	playerFactionID := cih.getPlayerFactionID(encounterID)
	if playerFactionID == 0 {
		println("[DEBUG] No player faction found, cannot kill enemies")
		return
	}

	enemySquads := cih.deps.Queries.GetEnemySquadsForEncounter(playerFactionID, encounterID)
	println("[DEBUG] Ctrl+K pressed - Found", len(enemySquads), "enemy squads in encounter", encounterID, "to kill")

	if len(enemySquads) == 0 {
		return
	}

	// Kill all units in each enemy squad
	totalKilled := 0
	for _, squadID := range enemySquads {
		killed := cih.killAllUnitsInSquad(squadID)
		totalKilled += killed
		println("[DEBUG] Killed", killed, "units in squad", squadID)
	}

	println("[DEBUG] Total units killed:", totalKilled)

	// Note: Victory will be detected at next turn boundary (handleEndTurn or advanceAfterAITurn)
	// Debug cheat doesn't need immediate victory check since player must still end turn
}

// killAllUnitsInSquad sets all units in a squad to 0 health and returns count killed
func (cih *CombatInputHandler) killAllUnitsInSquad(squadID ecs.EntityID) int {
	// Get all units in the squad
	unitIDs := squads.GetUnitIDsInSquad(squadID, cih.deps.Queries.ECSManager)
	killed := 0

	// Set health to 0 for all units
	for _, unitID := range unitIDs {
		unitEntity := cih.deps.Queries.ECSManager.FindEntityByID(unitID)
		if unitEntity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
		if attr != nil {
			oldHealth := attr.CurrentHealth
			attr.CurrentHealth = 0
			killed++
			println("[DEBUG]   Unit", unitID, "health:", oldHealth, "-> 0")
		}
	}

	return killed
}

// getPlayerFactionID finds the player-controlled faction in the current encounter
func (cih *CombatInputHandler) getPlayerFactionID(encounterID ecs.EntityID) ecs.EntityID {
	encounterFactions := cih.deps.Queries.GetFactionsForEncounter(encounterID)
	for _, factionID := range encounterFactions {
		factionData := cih.deps.Queries.CombatCache.FindFactionDataByID(factionID)
		if factionData != nil && factionData.IsPlayerControlled {
			return factionID
		}
	}
	return 0
}

// toggleInspectMode toggles inspect mode and shows/hides the panel
func (cih *CombatInputHandler) toggleInspectMode() {
	cih.actionHandler.ToggleInspectMode()
	if !cih.deps.BattleState.InInspectMode && cih.inspectPanel != nil {
		cih.inspectPanel.Hide()
	}
}

// exitInspectMode exits inspect mode and hides the panel
func (cih *CombatInputHandler) exitInspectMode() {
	cih.actionHandler.ExitInspectMode()
	if cih.inspectPanel != nil {
		cih.inspectPanel.Hide()
	}
}

// handleInspectClick processes a click while in inspect mode
func (cih *CombatInputHandler) handleInspectClick(mouseX, mouseY int) {
	if cih.playerPos == nil || cih.inspectPanel == nil {
		return
	}

	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)
	clickedSquadID := combat.GetSquadAtPosition(clickedPos, cih.deps.Queries.ECSManager)

	if clickedSquadID == 0 {
		return
	}

	cih.inspectPanel.PopulateGrid(clickedSquadID)
}

// handleThreatToggle handles H key to toggle threat heat map
func (cih *CombatInputHandler) handleThreatToggle(inputState *framework.InputState) bool {
	threatViz := cih.visualization.GetThreatVisualizer()
	if threatViz == nil {
		return false
	}

	// Shift+H cycles faction
	if inputState.ActionActive(framework.ActionThreatCycleFact) {
		threatViz.CycleFaction()
		cih.updateLayerStatusWidget()
		return true
	}

	// Plain H toggles threat mode
	if inputState.ActionActive(framework.ActionThreatToggle) {
		if !threatViz.IsActive() || threatViz.GetMode() != behavior.VisualizerModeThreat {
			threatViz.SetMode(behavior.VisualizerModeThreat)
			if !threatViz.IsActive() {
				threatViz.Toggle()
			}
		} else {
			threatViz.Toggle()
		}
		cih.updateLayerStatusWidget()
		return true
	}

	return false
}

// handleHealthBarToggle handles CtrlRight key to toggle health bars
func (cih *CombatInputHandler) handleHealthBarToggle(inputState *framework.InputState) bool {
	if !inputState.ActionActive(framework.ActionHealthBarToggle) {
		return false
	}

	battleState := cih.deps.BattleState
	battleState.ShowHealthBars = !battleState.ShowHealthBars
	return true
}

// handleLayerToggle handles L key to toggle layer visualizer
func (cih *CombatInputHandler) handleLayerToggle(inputState *framework.InputState) bool {
	threatViz := cih.visualization.GetThreatVisualizer()
	if threatViz == nil {
		return false
	}

	// Shift+L cycles layer mode
	if inputState.ActionActive(framework.ActionLayerCycleMode) {
		threatViz.CycleLayerMode()
		cih.updateLayerStatusWidget()
		return true
	}

	// Plain L toggles layer visualizer
	if inputState.ActionActive(framework.ActionLayerToggle) {
		if !threatViz.IsActive() || threatViz.GetMode() != behavior.VisualizerModeLayer {
			threatViz.SetMode(behavior.VisualizerModeLayer)
			if !threatViz.IsActive() {
				threatViz.Toggle()
			}
		} else {
			threatViz.Toggle()
		}
		cih.updateLayerStatusWidget()
		return true
	}

	return false
}

// updateLayerStatusWidget updates the layer status panel visibility and text
func (cih *CombatInputHandler) updateLayerStatusWidget() {
	threatViz := cih.visualization.GetThreatVisualizer()

	result := cih.panels.Get(CombatPanelLayerStatus)
	if result == nil || threatViz == nil {
		return
	}
	layerStatusPanel := result.Container
	layerStatusText := result.TextLabel
	if layerStatusPanel == nil || layerStatusText == nil {
		return
	}

	if !threatViz.IsActive() {
		layerStatusPanel.GetWidget().Visibility = widget.Visibility_Hide
		return
	}

	// Get faction name for display
	factionName := "Unknown"
	if factionInfo := cih.deps.Queries.GetFactionInfo(threatViz.GetViewFactionID()); factionInfo != nil {
		factionName = factionInfo.Name
	}

	var statusText string
	switch threatViz.GetMode() {
	case behavior.VisualizerModeThreat:
		statusText = fmt.Sprintf("THREAT VIEW\nFaction: %s", factionName)
	case behavior.VisualizerModeLayer:
		modeInfo := threatViz.GetLayerModeInfo()
		statusText = fmt.Sprintf("LAYER VIEW\nFaction: %s\n%s\n%s", factionName, modeInfo.Name, modeInfo.ColorKey)
	}
	layerStatusText.Label = statusText
	layerStatusPanel.GetWidget().Visibility = widget.Visibility_Show
}
