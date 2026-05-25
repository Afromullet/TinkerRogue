package guiartifacts

import (
	"fmt"
	"game_main/gui/framework"
	"game_main/gui/widgets"
	"game_main/tactical/powers/artifacts"

	"github.com/ebitenui/ebitenui/widget"
)

// ArtifactPanelDeps holds injected dependencies for the artifact panel controller.
type ArtifactPanelDeps struct {
	Handler      *ArtifactActivationHandler
	BattleState  *framework.TacticalState
	ShowSubmenu  func() // injected from CombatMode's subMenus.Show("artifact")
	CloseSubmenu func() // injected from CombatMode's subMenus.CloseAll()
}

// ArtifactPanelController manages artifact selection panel state and
// interactions. Common widget refs and selection plumbing live on the
// embedded SelectionPanelCore; artifact-specific behavior
// (charge-availability detail formatting, InArtifactMode mode flag) lives
// here.
type ArtifactPanelController struct {
	widgets.SelectionPanelCore
	deps *ArtifactPanelDeps
}

// NewArtifactPanelController creates a new artifact panel controller.
func NewArtifactPanelController(deps *ArtifactPanelDeps) *ArtifactPanelController {
	return &ArtifactPanelController{
		deps: deps,
		SelectionPanelCore: widgets.SelectionPanelCore{
			ShowSubmenu:  deps.ShowSubmenu,
			CloseSubmenu: deps.CloseSubmenu,
		},
	}
}

// SetWidgets stores widget references after panel construction.
func (ap *ArtifactPanelController) SetWidgets(list *widgets.CachedListWrapper, detail *widgets.CachedTextAreaWrapper, activate *widget.Button) {
	ap.List = list
	ap.Detail = detail
	ap.ActionButton = activate
}

// Handler returns the underlying ArtifactActivationHandler.
func (ap *ArtifactPanelController) Handler() *ArtifactActivationHandler {
	return ap.deps.Handler
}

// selectedArtifact returns the currently selected artifact option, or nil.
func (ap *ArtifactPanelController) selectedArtifact() *ArtifactOption {
	if ap.Selected == nil {
		return nil
	}
	return ap.Selected.(*ArtifactOption)
}

// OnArtifactSelected is the list click callback - updates detail area and activate button.
func (ap *ArtifactPanelController) OnArtifactSelected(option *ArtifactOption) {
	ap.Selected = option
	ap.UpdateDetailPanel()
}

// UpdateDetailPanel shows artifact details and checks charge availability.
func (ap *ArtifactPanelController) UpdateDetailPanel() {
	option := ap.selectedArtifact()
	if option == nil {
		return
	}

	targetLabel := GetTargetType(option.BehaviorKey).String()

	detail := fmt.Sprintf("=== %s ===\nTarget: %s\n\n%s",
		option.Name, targetLabel, option.Description)

	if !option.Available {
		b := artifacts.GetBehavior(option.BehaviorKey)
		if b != nil && b.ChargeType() == artifacts.ChargeOncePerRound {
			detail += "\n\n[Charge refreshes each round]"
		} else {
			detail += "\n\n[Charge spent this battle]"
		}
	}

	ap.SetDetail(detail, option.Available)
}

// Refresh populates the list from the handler and clears selection.
func (ap *ArtifactPanelController) Refresh() {
	allArtifacts := ap.deps.Handler.GetAvailableArtifacts()

	entries := make([]interface{}, len(allArtifacts))
	for i := range allArtifacts {
		opt := allArtifacts[i] // copy to avoid closure issue
		entries[i] = &opt
	}
	ap.SetEntries(entries)

	ap.ClearSelection("Select an artifact to view details")
}

// Show validates preconditions, refreshes data, and shows the panel.
func (ap *ArtifactPanelController) Show() {
	allArtifacts := ap.deps.Handler.GetAvailableArtifacts()
	if len(allArtifacts) == 0 {
		return
	}

	ap.deps.BattleState.InArtifactMode = true
	ap.Refresh()
	ap.ShowSubmenu()
}

// OnActivateClicked selects the artifact for targeting (or activates immediately) and hides the panel.
func (ap *ArtifactPanelController) OnActivateClicked() {
	option := ap.selectedArtifact()
	if option == nil {
		return
	}
	ap.deps.Handler.SelectArtifact(option.BehaviorKey)
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
