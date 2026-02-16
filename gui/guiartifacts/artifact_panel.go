package guiartifacts

import (
	"fmt"
	"game_main/gui/framework"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
)

// ArtifactPanelDeps holds injected dependencies for the artifact panel controller.
type ArtifactPanelDeps struct {
	Handler      *ArtifactActivationHandler
	BattleState  *framework.TacticalState
	AddCombatLog func(string)
	ShowSubmenu  func() // injected from CombatMode's subMenus.Show("artifact")
	CloseSubmenu func() // injected from CombatMode's subMenus.CloseAll()
}

// ArtifactPanelController manages artifact selection panel state and interactions.
type ArtifactPanelController struct {
	deps             *ArtifactPanelDeps
	artifactList     *widgets.CachedListWrapper
	detailArea       *widgets.CachedTextAreaWrapper
	activateButton   *widget.Button
	selectedArtifact *ArtifactOption
}

// NewArtifactPanelController creates a new artifact panel controller.
func NewArtifactPanelController(deps *ArtifactPanelDeps) *ArtifactPanelController {
	return &ArtifactPanelController{
		deps: deps,
	}
}

// SetWidgets sets widget references after panel construction.
func (ap *ArtifactPanelController) SetWidgets(list *widgets.CachedListWrapper, detail *widgets.CachedTextAreaWrapper, activate *widget.Button) {
	ap.artifactList = list
	ap.detailArea = detail
	ap.activateButton = activate
}

// Handler returns the underlying ArtifactActivationHandler.
func (ap *ArtifactPanelController) Handler() *ArtifactActivationHandler {
	return ap.deps.Handler
}

// OnArtifactSelected is the list click callback - updates detail area and activate button.
func (ap *ArtifactPanelController) OnArtifactSelected(option *ArtifactOption) {
	ap.selectedArtifact = option
	ap.UpdateDetailPanel()
}

// UpdateDetailPanel shows artifact details and checks charge availability.
func (ap *ArtifactPanelController) UpdateDetailPanel() {
	option := ap.selectedArtifact
	if option == nil || ap.detailArea == nil {
		return
	}

	targetLabel := "No Target"
	switch GetTargetType(option.BehaviorKey) {
	case TargetFriendlySquad:
		targetLabel = "Friendly Squad"
	case TargetEnemySquad:
		targetLabel = "Enemy Squad"
	}

	detail := fmt.Sprintf("=== %s ===\nTarget: %s\n\n%s",
		option.Name, targetLabel, option.Description)

	if !option.Available {
		detail += "\n\n[Charge spent this battle]"
	}

	ap.detailArea.SetText(detail)

	if ap.activateButton != nil {
		ap.activateButton.GetWidget().Disabled = !option.Available
	}
}

// Refresh populates the list from the handler and clears selection.
func (ap *ArtifactPanelController) Refresh() {
	allArtifacts := ap.deps.Handler.GetAvailableArtifacts()

	if ap.artifactList != nil {
		entries := make([]interface{}, len(allArtifacts))
		for i := range allArtifacts {
			opt := allArtifacts[i] // copy to avoid closure issue
			entries[i] = &opt
		}
		ap.artifactList.GetList().SetEntries(entries)
		ap.artifactList.MarkDirty()
	}

	ap.selectedArtifact = nil
	if ap.detailArea != nil {
		ap.detailArea.SetText("Select an artifact to view details")
	}
	if ap.activateButton != nil {
		ap.activateButton.GetWidget().Disabled = true
	}
}

// Show validates preconditions, refreshes data, and shows the panel.
func (ap *ArtifactPanelController) Show() {
	allArtifacts := ap.deps.Handler.GetAvailableArtifacts()
	if len(allArtifacts) == 0 {
		ap.deps.AddCombatLog("No activatable artifacts equipped")
		return
	}

	ap.deps.BattleState.InArtifactMode = true
	ap.Refresh()
	ap.deps.ShowSubmenu()
}

// Hide hides the panel and clears selection.
func (ap *ArtifactPanelController) Hide() {
	ap.selectedArtifact = nil
	ap.deps.CloseSubmenu()
}

// OnActivateClicked selects the artifact for targeting (or activates immediately) and hides the panel.
func (ap *ArtifactPanelController) OnActivateClicked() {
	if ap.selectedArtifact == nil {
		return
	}
	ap.deps.Handler.SelectArtifact(ap.selectedArtifact.BehaviorKey)
	ap.Hide()
}

// OnCancelClicked cancels artifact mode and hides the panel.
func (ap *ArtifactPanelController) OnCancelClicked() {
	ap.deps.Handler.CancelArtifactMode()
	ap.Hide()
}

// Toggle consolidates the artifact button click logic: cancel+hide if active, show if not.
func (ap *ArtifactPanelController) Toggle() {
	if ap.deps.Handler.IsInArtifactMode() {
		ap.deps.Handler.CancelArtifactMode()
		ap.Hide()
		return
	}
	ap.Show()
}
