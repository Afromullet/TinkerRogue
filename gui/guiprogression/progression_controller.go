package guiprogression

import (
	"fmt"
	"sort"
	"strings"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/gui/widgets"
	"game_main/tactical/powers/perks"
	"game_main/tactical/powers/progression"
	"game_main/templates"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// libraryItem is a uniform view of a perk or spell entry for the locked/unlocked lists.
type libraryItem struct {
	id         string // primary key, also the string form for progression lookups
	name       string
	unlockCost int
	detail     string // pre-rendered detail body (minus cost line)
}

// librarySource abstracts over perk vs. spell data for the library panel controller.
// Having one interface shape lets a single controller instance drive either panel.
type librarySource struct {
	label         string // panel title, e.g. "Perks (Skill Points)"
	currencyName  string // e.g. "Skill" or "Arcana"
	unlockBtnText string // e.g. "Unlock Perk"
	detailPrompt  string // e.g. "Select a perk to view details."

	allItems      func() []libraryItem
	isUnlocked    func(playerID ecs.EntityID, itemID string, manager *common.EntityManager) bool
	currentPoints func(data *progression.ProgressionData) int
	unlock        func(playerID ecs.EntityID, itemID string, manager *common.EntityManager) error
}

// libraryEntry wraps a libraryItem for placement in the widget.List. Pointers
// are used so identity checks work when the user re-selects an entry.
type libraryEntry struct {
	item  libraryItem
	label string
}

// libraryPanelController drives one locked/unlocked library panel.
type libraryPanelController struct {
	source librarySource
	mode   *ProgressionMode

	unlockedList *widgets.CachedListWrapper
	lockedList   *widgets.CachedListWrapper
	detail       *widgets.CachedTextAreaWrapper
	unlockBtn    *widget.Button

	selected *libraryEntry // only set when a locked entry is selected
}

func (pc *libraryPanelController) onUnlockedSelected(entry *libraryEntry) {
	pc.selected = nil
	pc.unlockBtn.GetWidget().Disabled = true
	pc.detail.SetText(formatDetail(entry.item, pc.source.currencyName, false))
}

func (pc *libraryPanelController) onLockedSelected(entry *libraryEntry) {
	pc.selected = entry
	data := progression.GetProgression(pc.mode.activePlayerID(), pc.mode.Context.ECSManager)
	canAfford := data != nil && pc.source.currentPoints(data) >= entry.item.unlockCost
	pc.unlockBtn.GetWidget().Disabled = !canAfford
	pc.detail.SetText(formatDetail(entry.item, pc.source.currencyName, true))
}

func (pc *libraryPanelController) onUnlockClicked() {
	if pc.selected == nil {
		return
	}
	if err := pc.source.unlock(pc.mode.activePlayerID(), pc.selected.item.id, pc.mode.Context.ECSManager); err != nil {
		pc.mode.SetStatus(fmt.Sprintf("Unlock failed: %v", err))
		return
	}
	pc.mode.SetStatus(fmt.Sprintf("Unlocked %s: %s", pc.source.currencyName, pc.selected.item.name))
	pc.mode.controller.refresh()
}

// progressionController owns the header labels and both library sub-panels.
type progressionController struct {
	mode *ProgressionMode

	arcanaLabel *widget.Text
	skillLabel  *widget.Text

	perkPanel  *libraryPanelController
	spellPanel *libraryPanelController
}

func newProgressionController(mode *ProgressionMode) *progressionController {
	return &progressionController{
		mode: mode,
		perkPanel: &libraryPanelController{
			mode:   mode,
			source: perkLibrarySource,
		},
		spellPanel: &libraryPanelController{
			mode:   mode,
			source: spellLibrarySource,
		},
	}
}

func (pc *progressionController) initWidgets() {
	pc.arcanaLabel = framework.GetPanelWidget[*widget.Text](pc.mode.Panels, ProgressionPanelHeader, "arcanaLabel")
	pc.skillLabel = framework.GetPanelWidget[*widget.Text](pc.mode.Panels, ProgressionPanelHeader, "skillLabel")

	pc.perkPanel.unlockedList = framework.GetPanelWidget[*widgets.CachedListWrapper](pc.mode.Panels, ProgressionPanelPerks, "unlockedList")
	pc.perkPanel.lockedList = framework.GetPanelWidget[*widgets.CachedListWrapper](pc.mode.Panels, ProgressionPanelPerks, "lockedList")
	pc.perkPanel.detail = framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](pc.mode.Panels, ProgressionPanelPerks, "detail")
	pc.perkPanel.unlockBtn = framework.GetPanelWidget[*widget.Button](pc.mode.Panels, ProgressionPanelPerks, "unlockBtn")

	pc.spellPanel.unlockedList = framework.GetPanelWidget[*widgets.CachedListWrapper](pc.mode.Panels, ProgressionPanelSpells, "unlockedList")
	pc.spellPanel.lockedList = framework.GetPanelWidget[*widgets.CachedListWrapper](pc.mode.Panels, ProgressionPanelSpells, "lockedList")
	pc.spellPanel.detail = framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](pc.mode.Panels, ProgressionPanelSpells, "detail")
	pc.spellPanel.unlockBtn = framework.GetPanelWidget[*widget.Button](pc.mode.Panels, ProgressionPanelSpells, "unlockBtn")
}

// === Package-level library sources (built once at init). ===
//
// Hoisted out of factory funcs so panel descriptors and the runtime controller
// share the same closure values instead of allocating duplicates per call.
var (
	perkLibrarySource = librarySource{
		label:         "Perks (Skill Points)",
		currencyName:  "Skill",
		unlockBtnText: "Unlock Perk",
		detailPrompt:  "Select a perk to view details.",
		allItems:      allPerkItems,
		isUnlocked: func(playerID ecs.EntityID, itemID string, manager *common.EntityManager) bool {
			return progression.IsPerkUnlocked(playerID, perks.PerkID(itemID), manager)
		},
		currentPoints: func(d *progression.ProgressionData) int { return d.SkillPoints },
		unlock: func(playerID ecs.EntityID, itemID string, manager *common.EntityManager) error {
			return progression.UnlockPerk(playerID, perks.PerkID(itemID), manager)
		},
	}

	spellLibrarySource = librarySource{
		label:         "Spells (Arcana Points)",
		currencyName:  "Arcana",
		unlockBtnText: "Unlock Spell",
		detailPrompt:  "Select a spell to view details.",
		allItems:      allSpellItems,
		isUnlocked:    progression.IsSpellUnlocked,
		currentPoints: func(d *progression.ProgressionData) int { return d.ArcanaPoints },
		unlock:        progression.UnlockSpell,
	}
)

func allPerkItems() []libraryItem {
	ids := perks.GetAllPerkIDs()
	sort.Slice(ids, func(i, j int) bool { return string(ids[i]) < string(ids[j]) })
	items := make([]libraryItem, 0, len(ids))
	for _, id := range ids {
		def := perks.GetPerkDefinition(id)
		if def == nil {
			continue
		}
		items = append(items, libraryItem{
			id:         string(id),
			name:       def.Name,
			unlockCost: def.UnlockCost,
			detail:     formatPerkBody(def),
		})
	}
	return items
}

func allSpellItems() []libraryItem {
	ids := templates.GetAllSpellIDs()
	sort.Strings(ids)
	items := make([]libraryItem, 0, len(ids))
	for _, id := range ids {
		def := templates.GetSpellDefinition(id)
		if def == nil {
			continue
		}
		items = append(items, libraryItem{
			id:         id,
			name:       def.Name,
			unlockCost: def.UnlockCost,
			detail:     formatSpellBody(def),
		})
	}
	return items
}

// === Detail formatters ===

// formatDetail renders a libraryItem's detail body, optionally appending a cost line.
func formatDetail(item libraryItem, currency string, showCost bool) string {
	var b strings.Builder
	b.WriteString(item.name)
	b.WriteString("\n\n")
	b.WriteString(item.detail)
	if showCost {
		b.WriteString(fmt.Sprintf("\n\nCost: %d %s", item.unlockCost, currency))
	}
	return b.String()
}

func formatPerkBody(def *perks.PerkDefinition) string {
	var b strings.Builder
	b.WriteString("Tier: ")
	b.WriteString(def.Tier.String())
	b.WriteString("\nCategory: ")
	b.WriteString(def.Category.String())
	if len(def.Roles) > 0 {
		b.WriteString("\nRoles: ")
		b.WriteString(strings.Join(def.Roles, ", "))
	}
	b.WriteString("\n\n")
	b.WriteString(def.Description)
	return b.String()
}

func formatSpellBody(def *templates.SpellDefinition) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Mana: %d", def.ManaCost))
	if def.Damage > 0 {
		b.WriteString(fmt.Sprintf("\nDamage: %d", def.Damage))
	}
	b.WriteString("\nTarget: ")
	b.WriteString(string(def.TargetType))
	b.WriteString("\nEffect: ")
	b.WriteString(string(def.EffectType))
	b.WriteString("\n\n")
	b.WriteString(def.Description)
	return b.String()
}
