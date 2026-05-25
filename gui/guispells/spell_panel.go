package guispells

import (
	"fmt"
	"game_main/gui/framework"
	"game_main/gui/widgets"
	"game_main/templates"

	"github.com/ebitenui/ebitenui/widget"
)

// SpellPanelDeps holds injected dependencies for the spell panel controller.
type SpellPanelDeps struct {
	Handler      *SpellCastingHandler
	BattleState  *framework.TacticalState
	ShowSubmenu  func() // injected from CombatMode's subMenus.Show("spell")
	CloseSubmenu func() // injected from CombatMode's subMenus.CloseAll()
}

// SpellPanelController manages spell selection panel state and interactions.
// Common widget refs and selection plumbing live on the embedded
// SelectionPanelCore; spell-specific behavior (mana check, EnterSpellMode
// gating) lives here.
type SpellPanelController struct {
	widgets.SelectionPanelCore
	deps *SpellPanelDeps
}

// NewSpellPanelController creates a new spell panel controller.
func NewSpellPanelController(deps *SpellPanelDeps) *SpellPanelController {
	return &SpellPanelController{
		deps: deps,
		SelectionPanelCore: widgets.SelectionPanelCore{
			ShowSubmenu:  deps.ShowSubmenu,
			CloseSubmenu: deps.CloseSubmenu,
		},
	}
}

// SetWidgets stores widget references after panel construction.
func (sp *SpellPanelController) SetWidgets(list *widgets.CachedListWrapper, detail *widgets.CachedTextAreaWrapper, mana *widget.Text, cast *widget.Button) {
	sp.List = list
	sp.Detail = detail
	sp.StatusLabel = mana
	sp.ActionButton = cast
}

// Handler returns the underlying SpellCastingHandler.
func (sp *SpellPanelController) Handler() *SpellCastingHandler {
	return sp.deps.Handler
}

// selectedSpell returns the currently selected spell, or nil if none.
func (sp *SpellPanelController) selectedSpell() *templates.SpellDefinition {
	if sp.Selected == nil {
		return nil
	}
	return sp.Selected.(*templates.SpellDefinition)
}

// OnSpellSelected is the list click callback — updates detail area and cast button.
func (sp *SpellPanelController) OnSpellSelected(spell *templates.SpellDefinition) {
	sp.Selected = spell
	sp.UpdateDetailPanel()
}

// UpdateDetailPanel shows spell details and checks mana affordability.
func (sp *SpellPanelController) UpdateDetailPanel() {
	spell := sp.selectedSpell()
	if spell == nil {
		return
	}

	currentMana, _ := sp.deps.Handler.GetSquadMana()
	canAfford := currentMana >= spell.ManaCost

	targetType := "Single Target"
	if spell.IsAoE() {
		targetType = "AoE"
	}

	detail := fmt.Sprintf("=== %s ===\nCost: %d MP\nDamage: %d\nTarget: %s\n\n%s",
		spell.Name, spell.ManaCost, spell.Damage, targetType, spell.Description)

	if !canAfford {
		detail += "\n\n[color=ff4444]Not enough mana![/color]"
	}

	sp.SetDetail(detail, canAfford)
}

// Refresh populates the list from the spell handler, updates mana label, clears selection.
func (sp *SpellPanelController) Refresh() {
	allSpells := sp.deps.Handler.GetAllSpells()
	currentMana, maxMana := sp.deps.Handler.GetSquadMana()

	sp.SetStatusText(fmt.Sprintf("Mana: %d/%d", currentMana, maxMana))

	entries := make([]interface{}, len(allSpells))
	for i, spell := range allSpells {
		entries[i] = spell
	}
	sp.SetEntries(entries)

	sp.ClearSelection("Select a spell to view details")
}

// Show validates preconditions, refreshes data, and shows the panel.
func (sp *SpellPanelController) Show() {
	if !sp.deps.Handler.CanSelectedSquadCast() {
		if reason := sp.deps.Handler.CastBlockReason(); reason != "" {
			fmt.Printf("[Spells] Cannot cast: %s\n", reason)
		}
		return
	}

	allSpells := sp.deps.Handler.GetAllSpells()
	if len(allSpells) == 0 {
		fmt.Println("[Spells] Cannot cast: no spells available")
		return
	}

	sp.deps.Handler.EnterSpellMode()
	sp.Refresh()
	sp.ShowSubmenu()
}

// OnCastClicked selects the spell for targeting and hides the panel.
func (sp *SpellPanelController) OnCastClicked() {
	spell := sp.selectedSpell()
	if spell == nil {
		return
	}
	sp.deps.Handler.SelectSpell(spell.ID)
	sp.Hide()
}

// OnCancelClicked cancels spell mode and hides the panel.
func (sp *SpellPanelController) OnCancelClicked() {
	sp.deps.Handler.CancelSpellMode()
	sp.Hide()
}

// Toggle consolidates the spell button click logic: cancel+hide if active, show if not.
func (sp *SpellPanelController) Toggle() {
	if sp.deps.Handler.IsInSpellMode() {
		sp.deps.Handler.CancelSpellMode()
		sp.Hide()
		return
	}
	sp.Show()
}
