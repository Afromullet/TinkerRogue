package combatinput

import (
	"fmt"

	"game_main/gui/framework"
	"game_main/gui/guicombat/combatbase"
	"game_main/gui/guicombat/combatvisualization"

	"github.com/ebitenui/ebitenui/widget"
)

func (cih *CombatInputHandler) handleThreatToggle(inputState *framework.InputState) bool {
	threatViz := cih.visualization.GetThreatVisualizer()
	if threatViz == nil {
		return false
	}

	if inputState.ActionActive(framework.ActionThreatCycleFact) {
		threatViz.CycleFaction()
		cih.updateLayerStatusWidget()
		return true
	}

	if inputState.ActionActive(framework.ActionThreatToggle) {
		if !threatViz.IsActive() || threatViz.GetMode() != combatvisualization.VisualizerModeThreat {
			threatViz.SetMode(combatvisualization.VisualizerModeThreat)
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

func (cih *CombatInputHandler) handleHealthBarToggle(inputState *framework.InputState) bool {
	if !inputState.ActionActive(framework.ActionHealthBarToggle) {
		return false
	}

	battleState := cih.deps.BattleState
	battleState.ShowHealthBars = !battleState.ShowHealthBars
	return true
}

func (cih *CombatInputHandler) handleLayerToggle(inputState *framework.InputState) bool {
	threatViz := cih.visualization.GetThreatVisualizer()
	if threatViz == nil {
		return false
	}

	if inputState.ActionActive(framework.ActionLayerCycleMode) {
		threatViz.CycleLayerMode()
		cih.updateLayerStatusWidget()
		return true
	}

	if inputState.ActionActive(framework.ActionLayerToggle) {
		if !threatViz.IsActive() || threatViz.GetMode() != combatvisualization.VisualizerModeLayer {
			threatViz.SetMode(combatvisualization.VisualizerModeLayer)
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

func (cih *CombatInputHandler) updateLayerStatusWidget() {
	threatViz := cih.visualization.GetThreatVisualizer()

	result := cih.panels.Get(combatbase.CombatPanelLayerStatus)
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

	factionName := "Unknown"
	if factionInfo := cih.deps.Queries.GetFactionInfo(threatViz.GetViewFactionID()); factionInfo != nil {
		factionName = factionInfo.Name
	}

	var statusText string
	switch threatViz.GetMode() {
	case combatvisualization.VisualizerModeThreat:
		statusText = fmt.Sprintf("THREAT VIEW\nFaction: %s", factionName)
	case combatvisualization.VisualizerModeLayer:
		modeInfo := threatViz.GetLayerModeInfo()
		statusText = fmt.Sprintf("LAYER VIEW\nFaction: %s\n%s\n%s", factionName, modeInfo.Name, modeInfo.ColorKey)
	}
	layerStatusText.Label = statusText
	layerStatusPanel.GetWidget().Visibility = widget.Visibility_Show
}
