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
// It owns the widget references and selected spell, delegating spell logic to SpellCastingHandler.
type SpellPanelController struct {
	deps          *SpellPanelDeps
	spellList     *widgets.CachedListWrapper
	detailArea    *widgets.CachedTextAreaWrapper
	manaLabel     *widget.Text
	castButton    *widget.Button
	selectedSpell *templates.SpellDefinition
}

// NewSpellPanelController creates a new spell panel controller.
func NewSpellPanelController(deps *SpellPanelDeps) *SpellPanelController {
	return &SpellPanelController{
		deps: deps,
	}
}

// SetWidgets sets widget references after panel construction.
func (sp *SpellPanelController) SetWidgets(list *widgets.CachedListWrapper, detail *widgets.CachedTextAreaWrapper, mana *widget.Text, cast *widget.Button) {
	sp.spellList = list
	sp.detailArea = detail
	sp.manaLabel = mana
	sp.castButton = cast
}

// Handler returns the underlying SpellCastingHandler.
func (sp *SpellPanelController) Handler() *SpellCastingHandler {
	return sp.deps.Handler
}

// OnSpellSelected is the list click callback — updates detail area and cast button.
func (sp *SpellPanelController) OnSpellSelected(spell *templates.SpellDefinition) {
	sp.selectedSpell = spell
	sp.UpdateDetailPanel()
}

// UpdateDetailPanel shows spell details and checks mana affordability.
func (sp *SpellPanelController) UpdateDetailPanel() {
	spell := sp.selectedSpell
	if spell == nil || sp.detailArea == nil {
		return
	}

	currentMana, _ := sp.deps.Handler.GetCommanderMana()
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

	sp.detailArea.SetText(detail)

	if sp.castButton != nil {
		sp.castButton.GetWidget().Disabled = !canAfford
	}
}

// Refresh populates the list from the spell handler, updates mana label, clears selection.
func (sp *SpellPanelController) Refresh() {
	allSpells := sp.deps.Handler.GetAllSpells()
	currentMana, maxMana := sp.deps.Handler.GetCommanderMana()

	if sp.manaLabel != nil {
		sp.manaLabel.Label = fmt.Sprintf("Mana: %d/%d", currentMana, maxMana)
	}

	if sp.spellList != nil {
		entries := make([]interface{}, len(allSpells))
		for i, spell := range allSpells {
			entries[i] = spell
		}
		sp.spellList.GetList().SetEntries(entries)
		sp.spellList.MarkDirty()
	}

	sp.selectedSpell = nil
	if sp.detailArea != nil {
		sp.detailArea.SetText("Select a spell to view details")
	}
	if sp.castButton != nil {
		sp.castButton.GetWidget().Disabled = true
	}
}

// Show validates preconditions, refreshes data, and shows the panel.
func (sp *SpellPanelController) Show() {
	if sp.deps.BattleState.HasCastSpell {
		return
	}

	allSpells := sp.deps.Handler.GetAllSpells()
	if len(allSpells) == 0 {
		return
	}

	sp.deps.Handler.EnterSpellMode()
	sp.Refresh()
	sp.deps.ShowSubmenu()
}

// Hide hides the panel and clears selection.
func (sp *SpellPanelController) Hide() {
	sp.selectedSpell = nil
	sp.deps.CloseSubmenu()
}

// OnCastClicked selects the spell for targeting and hides the panel.
func (sp *SpellPanelController) OnCastClicked() {
	if sp.selectedSpell == nil {
		return
	}
	sp.deps.Handler.SelectSpell(sp.selectedSpell.ID)
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
