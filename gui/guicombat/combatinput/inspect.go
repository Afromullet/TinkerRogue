package combatinput

import (
	"game_main/core/coords"
	"game_main/tactical/combat/combatstate"
)

func (cih *CombatInputHandler) ToggleInspectMode() {
	cih.actionHandler.ToggleInspectMode()
	if !cih.deps.BattleState.InInspectMode && cih.inspectPanel != nil {
		cih.inspectPanel.Hide()
	}
}

func (cih *CombatInputHandler) exitInspectMode() {
	cih.actionHandler.ExitInspectMode()
	if cih.inspectPanel != nil {
		cih.inspectPanel.Hide()
	}
}

func (cih *CombatInputHandler) handleInspectClick(mouseX, mouseY int) {
	if cih.playerPos == nil || cih.inspectPanel == nil {
		return
	}

	clickedPos := coords.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)
	clickedSquadID := combatstate.GetSquadAtPosition(clickedPos, cih.deps.Queries.ECSManager)

	if clickedSquadID == 0 {
		return
	}

	cih.inspectPanel.PopulateGrid(clickedSquadID)
}
