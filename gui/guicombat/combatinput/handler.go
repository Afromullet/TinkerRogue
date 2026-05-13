package combatinput

import (
	"time"

	"game_main/core/coords"
	"game_main/gui/framework"
	"game_main/gui/guiartifacts"
	"game_main/gui/guicombat/combatbase"
	"game_main/gui/guicombat/combatvisualization"
	"game_main/gui/guiinspect"
	"game_main/gui/guispells"
	"game_main/tactical/combat/combatstate"

	"github.com/bytearena/ecs"
)

const doubleClickThreshold = 300 * time.Millisecond

// CombatInputHandler manages all input processing for combat mode.
type CombatInputHandler struct {
	actionHandler    *combatbase.CombatActionHandler
	deps             *combatbase.CombatModeDeps
	playerPos        *coords.LogicalPosition
	currentFactionID ecs.EntityID

	inDebugKillMode bool

	spellPanel    *guispells.SpellPanelController
	artifactPanel *guiartifacts.ArtifactPanelController

	visualization *combatvisualization.CombatVisualizationManager
	panels        *framework.PanelRegistry

	inspectPanel *guiinspect.InspectPanelController

	lastClickTime    time.Time
	lastClickSquadID ecs.EntityID
}

// NewCombatInputHandler creates a new combat input handler
func NewCombatInputHandler(actionHandler *combatbase.CombatActionHandler, deps *combatbase.CombatModeDeps) *CombatInputHandler {
	return &CombatInputHandler{
		actionHandler: actionHandler,
		deps:          deps,
	}
}

func (cih *CombatInputHandler) SetPlayerPosition(playerPos *coords.LogicalPosition) {
	cih.playerPos = playerPos
}

func (cih *CombatInputHandler) SetCurrentFactionID(factionID ecs.EntityID) {
	cih.currentFactionID = factionID
}

func (cih *CombatInputHandler) SetSpellPanel(panel *guispells.SpellPanelController) {
	cih.spellPanel = panel
}

func (cih *CombatInputHandler) SetArtifactPanel(panel *guiartifacts.ArtifactPanelController) {
	cih.artifactPanel = panel
}

func (cih *CombatInputHandler) SetInspectPanel(panel *guiinspect.InspectPanelController) {
	cih.inspectPanel = panel
}

func (cih *CombatInputHandler) SetVisualization(viz *combatvisualization.CombatVisualizationManager, panels *framework.PanelRegistry) {
	cih.visualization = viz
	cih.panels = panels
}

// HandleInput processes input and returns true if input was consumed
func (cih *CombatInputHandler) HandleInput(inputState *framework.InputState) bool {
	if cih.inDebugKillMode && inputState.ActionActive(framework.ActionCancel) {
		cih.inDebugKillMode = false
		return true
	}

	if cih.spellPanel != nil && cih.spellPanel.Handler().IsInSpellMode() {
		handler := cih.spellPanel.Handler()

		if inputState.ActionActive(framework.ActionCancel) {
			cih.spellPanel.OnCancelClicked()
			return true
		}

		if handler.HasSelectedSpell() {
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

			if inputState.ActionActive(framework.ActionMouseClick) {
				if handler.IsAoETargeting() {
					handler.HandleAoEConfirmClick(inputState.MouseX, inputState.MouseY)
				} else {
					handler.HandleSingleTargetClick(inputState.MouseX, inputState.MouseY)
				}
				return true
			}
		}
		return false
	}

	if cih.artifactPanel != nil && cih.artifactPanel.Handler().IsInArtifactMode() {
		handler := cih.artifactPanel.Handler()

		if inputState.ActionActive(framework.ActionCancel) {
			cih.artifactPanel.OnCancelClicked()
			return true
		}

		if handler.HasSelectedArtifact() {
			if inputState.ActionActive(framework.ActionMouseClick) {
				handler.HandleTargetClick(inputState.MouseX, inputState.MouseY)
				return true
			}
		}
		return false
	}

	if cih.deps.BattleState.InInspectMode {
		if inputState.ActionActive(framework.ActionCancel) {
			cih.exitInspectMode()
			return true
		}

		if inputState.ActionActive(framework.ActionMouseClick) {
			cih.handleInspectClick(inputState.MouseX, inputState.MouseY)
			return true
		}

		return false
	}

	if cih.deps.BattleState.InMoveMode && inputState.ActionActive(framework.ActionRightClick) {
		cih.actionHandler.ToggleMoveMode()
		return true
	}

	if inputState.ActionActive(framework.ActionMouseClick) {
		if cih.inDebugKillMode {
			defer func() { cih.inDebugKillMode = false }()
			cih.handleDebugKillClick(inputState.MouseX, inputState.MouseY)
			return true
		}

		if cih.deps.BattleState.InMoveMode {
			cih.handleMovementClick(inputState.MouseX, inputState.MouseY)
		} else {
			cih.handleSquadClick(inputState.MouseX, inputState.MouseY)
		}
		return true
	}

	if inputState.ActionActive(framework.ActionEndTurn) {
		return false
	}

	if inputState.ActionActive(framework.ActionSpellPanel) && cih.spellPanel != nil {
		cih.spellPanel.Toggle()
		return true
	}

	if inputState.ActionActive(framework.ActionArtifactPanel) && cih.artifactPanel != nil {
		cih.artifactPanel.Toggle()
		return true
	}

	if inputState.ActionActive(framework.ActionAttackMode) {
		cih.actionHandler.ToggleAttackMode()
		return true
	}

	if inputState.ActionActive(framework.ActionMoveMode) {
		cih.actionHandler.ToggleMoveMode()
		return true
	}

	if inputState.ActionActive(framework.ActionInspectMode) {
		if cih.spellPanel != nil && cih.spellPanel.Handler().IsInSpellMode() {
			cih.spellPanel.OnCancelClicked()
		}
		if cih.artifactPanel != nil && cih.artifactPanel.Handler().IsInArtifactMode() {
			cih.artifactPanel.OnCancelClicked()
		}
		cih.ToggleInspectMode()
		return true
	}

	if inputState.ActionActive(framework.ActionCycleSquad) {
		cih.actionHandler.CycleSquadSelection()
		return true
	}

	if inputState.ActionActive(framework.ActionUndoMove) {
		cih.actionHandler.UndoLastMove()
		return true
	}

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

	if cih.handleThreatToggle(inputState) {
		return true
	}
	if cih.handleHealthBarToggle(inputState) {
		return true
	}
	if cih.handleLayerToggle(inputState) {
		return true
	}

	if inputState.ActionActive(framework.ActionDebugKillAll) {
		println("[DEBUG] Calling killAllEnemySquads()")
		cih.killAllEnemySquads()
		return true
	}

	return false
}

func (cih *CombatInputHandler) handleMovementClick(mouseX, mouseY int) {
	if cih.playerPos == nil {
		return
	}

	clickedPos := coords.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)

	validTiles := cih.deps.CombatService.MovementSystem.GetValidMovementTiles(cih.deps.BattleState.SelectedSquadID)
	if validTiles == nil {
		validTiles = []coords.LogicalPosition{}
	}

	isValidTile := false
	for _, validPos := range validTiles {
		if validPos.X == clickedPos.X && validPos.Y == clickedPos.Y {
			isValidTile = true
			break
		}
	}
	if !isValidTile {
		return
	}

	selectedSquad := cih.deps.BattleState.SelectedSquadID
	if selectedSquad == 0 {
		return
	}

	cih.actionHandler.MoveSquad(selectedSquad, clickedPos)
}

func (cih *CombatInputHandler) handleSquadClick(mouseX, mouseY int) {
	if cih.playerPos == nil {
		return
	}

	clickedPos := coords.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)
	clickedSquadID := combatstate.GetSquadAtPosition(clickedPos, cih.deps.Queries.ECSManager)

	if clickedSquadID == 0 {
		cih.lastClickTime = time.Time{}
		cih.lastClickSquadID = 0
		return
	}

	squadInfo := cih.deps.Queries.GetSquadInfo(clickedSquadID)
	if squadInfo == nil {
		return
	}
	clickedFactionID := squadInfo.FactionID

	if cih.currentFactionID == 0 {
		return
	}

	factionData := cih.deps.Queries.CombatCache.FindFactionDataByID(cih.currentFactionID)
	if factionData != nil && factionData.IsPlayerControlled {
		if clickedFactionID == cih.currentFactionID {
			now := time.Now()
			isDoubleClick := clickedSquadID == cih.lastClickSquadID &&
				now.Sub(cih.lastClickTime) <= doubleClickThreshold

			cih.actionHandler.SelectSquad(clickedSquadID)

			if isDoubleClick {
				if !cih.deps.BattleState.InMoveMode {
					cih.actionHandler.ToggleMoveMode()
				}
				cih.lastClickTime = time.Time{}
				cih.lastClickSquadID = 0
			} else {
				cih.lastClickTime = now
				cih.lastClickSquadID = clickedSquadID
			}
			return
		}

		selectedSquad := cih.deps.BattleState.SelectedSquadID
		if selectedSquad != 0 && clickedFactionID != cih.currentFactionID {
			cih.deps.BattleState.SelectedTargetID = clickedSquadID
			cih.actionHandler.ExecuteAttack()
		}
		cih.lastClickTime = time.Time{}
		cih.lastClickSquadID = 0
	}
}
