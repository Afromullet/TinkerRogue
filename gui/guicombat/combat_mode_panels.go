package guicombat

import (
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/guicombat/combatbase"
	"game_main/gui/guiinspect"

	"github.com/ebitenui/ebitenui/widget"
)

// buildPanelsFromRegistry builds all combat panels using the Panel Registry
func (cm *CombatMode) buildPanelsFromRegistry() error {
	// Build sub-menu panels first (they register with subMenus controller)
	// Then build standard panels
	panels := []framework.PanelType{
		combatbase.CombatPanelDebugMenu,
		combatbase.CombatPanelMagicMenu,
		combatbase.CombatPanelSpellSelection,
		combatbase.CombatPanelArtifactSelection,
		guiinspect.InspectPanelType,
		combatbase.CombatPanelTurnOrder,
		combatbase.CombatPanelFactionInfo,
		combatbase.CombatPanelSquadDetail,
		combatbase.CombatPanelLayerStatus,
	}

	return cm.BuildPanels(panels...)
}

// buildContextActions creates bottom-left action buttons for combat actions
func (cm *CombatMode) buildContextActions() *widget.Container {
	return builders.CreateLeftActionBar(cm.Layout, []builders.ButtonSpec{
		{Text: "Attack (A)", OnClick: cm.handleAttackClick},
		{Text: "Move (M)", OnClick: cm.handleMoveClick},
		{Text: "Inspect (I)", OnClick: cm.handleInspectClick},
		{Text: "Magic", OnClick: cm.subMenus.Toggle("magic")},
	})
}

// buildNavigationActions creates bottom-right action buttons for turn/mode navigation
func (cm *CombatMode) buildNavigationActions() *widget.Container {
	return builders.CreateRightActionBar(cm.Layout, []builders.ButtonSpec{
		{Text: "Undo (Ctrl+Z)", OnClick: cm.handleUndoMove},
		{Text: "End Turn (Space)", OnClick: cm.handleEndTurnClick},
		{Text: "Debug", OnClick: cm.subMenus.Toggle("debug")},
	})
}

// Button click handlers that delegate to action handler
func (cm *CombatMode) handleAttackClick() {
	cm.actionHandler.ToggleAttackMode()
}

func (cm *CombatMode) handleMoveClick() {
	cm.actionHandler.ToggleMoveMode()
}

func (cm *CombatMode) handleSpellClick() {
	cm.spellPanel.Toggle()
}

func (cm *CombatMode) handleArtifactClick() {
	cm.artifactPanel.Toggle()
}

func (cm *CombatMode) handleInspectClick() {
	cm.inputHandler.ToggleInspectMode()
}

func (cm *CombatMode) handleUndoMove() {
	cm.actionHandler.UndoLastMove()
}

func (cm *CombatMode) handleEndTurnClick() {
	cm.turnFlow.HandleEndTurn()
}
