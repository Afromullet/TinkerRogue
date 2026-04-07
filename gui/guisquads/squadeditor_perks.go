package guisquads

import (
	"fmt"
	"strings"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/widgets"
	"game_main/tactical/powers/perks"

	"github.com/ebitenui/ebitenui/widget"
)

// perkPanelController manages the perk equip/unequip UI within the squad editor.
type perkPanelController struct {
	equippedList *widget.List
	availableList *widget.List
	detailArea   *widgets.CachedTextAreaWrapper
	equipBtn     *widget.Button
	unequipBtn   *widget.Button
	slotLabel    *widget.Text

	// Panel container (for SubMenuController registration)
	container *widget.Container

	// Currently selected perk in each list
	selectedEquipped  *perks.PerkDefinition
	selectedAvailable *perks.PerkDefinition

	// Back-reference to mode for squad context
	mode *SquadEditorMode
}

// initPerkPanel creates and wires the perk panel controller from the panel registry.
func (sem *SquadEditorMode) initPerkPanel() {
	sem.perkPanel = &perkPanelController{
		mode: sem,
	}

	sem.perkPanel.equippedList = framework.GetPanelWidget[*widget.List](
		sem.Panels, SquadEditorPanelPerks, "equippedList")
	sem.perkPanel.availableList = framework.GetPanelWidget[*widget.List](
		sem.Panels, SquadEditorPanelPerks, "availableList")
	sem.perkPanel.detailArea = framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](
		sem.Panels, SquadEditorPanelPerks, "detailArea")
	sem.perkPanel.equipBtn = framework.GetPanelWidget[*widget.Button](
		sem.Panels, SquadEditorPanelPerks, "equipBtn")
	sem.perkPanel.unequipBtn = framework.GetPanelWidget[*widget.Button](
		sem.Panels, SquadEditorPanelPerks, "unequipBtn")
	sem.perkPanel.slotLabel = framework.GetPanelWidget[*widget.Text](
		sem.Panels, SquadEditorPanelPerks, "slotLabel")
	sem.perkPanel.container = sem.GetPanelContainer(SquadEditorPanelPerks)
}

// refreshPerkPanel populates both perk lists for the currently selected squad.
func (pc *perkPanelController) refreshPerkPanel() {
	if !pc.mode.squadNav.HasSquads() {
		return
	}

	squadID := pc.mode.squadNav.CurrentID()
	manager := pc.mode.Context.ECSManager

	// Get equipped perk IDs
	equippedIDs := perks.GetEquippedPerkIDs(squadID, manager)

	// Build equipped entries
	equippedEntries := make([]interface{}, 0, len(equippedIDs))
	equippedSet := make(map[string]bool, len(equippedIDs))
	for _, id := range equippedIDs {
		def := perks.GetPerkDefinition(id)
		if def != nil {
			equippedEntries = append(equippedEntries, def)
			equippedSet[id] = true
		}
	}

	// Build available entries (all perks not currently equipped)
	allIDs := perks.GetAllPerkIDs()
	availableEntries := make([]interface{}, 0, len(allIDs)-len(equippedIDs))
	for _, id := range allIDs {
		if !equippedSet[id] {
			def := perks.GetPerkDefinition(id)
			if def != nil {
				availableEntries = append(availableEntries, def)
			}
		}
	}

	// Update lists by rebuilding (same pattern as other squad editor panels)
	pc.equippedList = pc.mode.replaceListInContainer(pc.container, pc.equippedList, func() *widget.List {
		return builders.CreateListWithConfig(builders.ListConfig{
			Entries:   equippedEntries,
			MinWidth:  250,
			MinHeight: 120,
			EntryLabelFunc: func(e interface{}) string {
				def := e.(*perks.PerkDefinition)
				return def.Name
			},
			OnEntrySelected: func(e interface{}) {
				pc.onEquippedSelected(e.(*perks.PerkDefinition))
			},
		})
	})

	pc.availableList = pc.mode.replaceListInContainer(pc.container, pc.availableList, func() *widget.List {
		return builders.CreateListWithConfig(builders.ListConfig{
			Entries:   availableEntries,
			MinWidth:  250,
			MinHeight: 150,
			EntryLabelFunc: func(e interface{}) string {
				def := e.(*perks.PerkDefinition)
				return fmt.Sprintf("%s [%s]", def.Name, def.Category)
			},
			OnEntrySelected: func(e interface{}) {
				pc.onAvailableSelected(e.(*perks.PerkDefinition))
			},
		})
	})

	// Update slot count label
	pc.slotLabel.Label = fmt.Sprintf("Perks (%d/%d)", len(equippedIDs), perks.MaxPerkSlots)

	// Clear selections
	pc.selectedEquipped = nil
	pc.selectedAvailable = nil
	pc.detailArea.SetText("Select a perk to view details.")
	pc.equipBtn.GetWidget().Disabled = true
	pc.unequipBtn.GetWidget().Disabled = true
}

// onEquippedSelected handles selection in the equipped perks list.
func (pc *perkPanelController) onEquippedSelected(def *perks.PerkDefinition) {
	pc.selectedEquipped = def
	pc.selectedAvailable = nil
	pc.unequipBtn.GetWidget().Disabled = false
	pc.equipBtn.GetWidget().Disabled = true
	pc.detailArea.SetText(formatPerkDetail(def))
}

// onAvailableSelected handles selection in the available perks list.
func (pc *perkPanelController) onAvailableSelected(def *perks.PerkDefinition) {
	pc.selectedAvailable = def
	pc.selectedEquipped = nil
	pc.equipBtn.GetWidget().Disabled = false
	pc.unequipBtn.GetWidget().Disabled = true
	pc.detailArea.SetText(formatPerkDetail(def))
}

// onEquipClicked equips the selected available perk on the current squad.
func (pc *perkPanelController) onEquipClicked() {
	if pc.selectedAvailable == nil || !pc.mode.squadNav.HasSquads() {
		return
	}

	squadID := pc.mode.squadNav.CurrentID()
	err := perks.EquipPerk(squadID, pc.selectedAvailable.ID, perks.MaxPerkSlots, pc.mode.Context.ECSManager)
	if err != nil {
		pc.mode.SetStatus(fmt.Sprintf("Cannot equip: %v", err))
		return
	}

	pc.mode.SetStatus(fmt.Sprintf("Equipped %s", pc.selectedAvailable.Name))
	pc.refreshPerkPanel()
}

// onUnequipClicked removes the selected equipped perk from the current squad.
func (pc *perkPanelController) onUnequipClicked() {
	if pc.selectedEquipped == nil || !pc.mode.squadNav.HasSquads() {
		return
	}

	squadID := pc.mode.squadNav.CurrentID()
	err := perks.UnequipPerk(squadID, pc.selectedEquipped.ID, pc.mode.Context.ECSManager)
	if err != nil {
		pc.mode.SetStatus(fmt.Sprintf("Cannot unequip: %v", err))
		return
	}

	pc.mode.SetStatus(fmt.Sprintf("Unequipped %s", pc.selectedEquipped.Name))
	pc.refreshPerkPanel()
}

// formatPerkDetail builds the detail text for a perk definition.
func formatPerkDetail(def *perks.PerkDefinition) string {
	var b strings.Builder

	b.WriteString(def.Name)
	b.WriteString("\n\n")

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

	if len(def.ExclusiveWith) > 0 {
		b.WriteString("\n\nExclusive with: ")
		names := make([]string, 0, len(def.ExclusiveWith))
		for _, exID := range def.ExclusiveWith {
			exDef := perks.GetPerkDefinition(exID)
			if exDef != nil {
				names = append(names, exDef.Name)
			} else {
				names = append(names, exID)
			}
		}
		b.WriteString(strings.Join(names, ", "))
	}

	return b.String()
}
