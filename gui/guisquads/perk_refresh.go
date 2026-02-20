package guisquads

import (
	"fmt"
	"sort"
	"strings"

	"game_main/common"
	"game_main/gui/builders"
	"game_main/tactical/perks"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// UI refresh logic for PerkMode

// replaceListInPerkContainer removes an old list widget from a container, creates a new one,
// and re-inserts it after the title label while preserving any trailing children.
func (pm *PerkMode) replaceListInPerkContainer(
	container *widget.Container,
	oldWidget *widget.List,
	createNew func() *widget.List,
) *widget.List {
	if container == nil {
		return oldWidget
	}
	container.RemoveChild(oldWidget)
	newWidget := createNew()
	children := container.Children()
	container.RemoveChildren()
	container.AddChild(children[0]) // Title label
	container.AddChild(newWidget)
	for i := 1; i < len(children); i++ {
		container.AddChild(children[i])
	}
	return newWidget
}

// refreshAllUI refreshes navigation and the active tab
func (pm *PerkMode) refreshAllUI() {
	pm.updateSquadCounter()
	pm.rebuildSlotButtons()
	pm.refreshActiveTab()
}

// refreshActiveTab refreshes whichever tab is currently visible
func (pm *PerkMode) refreshActiveTab() {
	switch pm.activeTab {
	case "available":
		pm.refreshAvailable()
	case "equipped":
		pm.refreshEquipped()
	}
}

// ensurePerkComponent attaches the perk component if the entity doesn't have it yet.
// Handles legacy entities and squads from CreateSquadFromTemplate.
func (pm *PerkMode) ensurePerkComponent() {
	entityID := pm.getCurrentEntityID()
	if entityID == 0 {
		return
	}
	manager := pm.Context.ECSManager

	switch pm.activeLevel {
	case "squad":
		if !manager.HasComponent(entityID, perks.SquadPerkComponent) {
			entity := manager.FindEntityByID(entityID)
			if entity != nil {
				perks.AttachSquadPerkComponent(entity)
			}
		}
	case "unit":
		if !manager.HasComponent(entityID, perks.UnitPerkComponent) {
			entity := manager.FindEntityByID(entityID)
			if entity != nil {
				perks.AttachUnitPerkComponent(entity)
			}
		}
	case "commander":
		if !manager.HasComponent(entityID, perks.CommanderPerkComponent) {
			entity := manager.FindEntityByID(entityID)
			if entity != nil {
				perks.AttachCommanderPerkComponent(entity)
			}
		}
	}
}

// refreshAvailable rebuilds the available perks list filtered by current level.
func (pm *PerkMode) refreshAvailable() {
	if pm.availableContent == nil {
		return
	}

	pm.ensurePerkComponent()

	entityID := pm.getCurrentEntityID()
	perkLevel := pm.getCurrentPerkLevel()
	manager := pm.Context.ECSManager

	// Get all perks matching the current level
	allIDs := perks.GetAllPerkIDs()
	var matchingIDs []string
	for _, id := range allIDs {
		def := perks.GetPerkDefinition(id)
		if def != nil && def.Level == perkLevel {
			matchingIDs = append(matchingIDs, id)
		}
	}

	// Sort by name for stable display
	sort.Slice(matchingIDs, func(i, j int) bool {
		defI := perks.GetPerkDefinition(matchingIDs[i])
		defJ := perks.GetPerkDefinition(matchingIDs[j])
		return defI.Name < defJ.Name
	})

	// Build display entries
	entries := make([]string, 0, len(matchingIDs))
	pm.availablePerkIDs = make([]string, 0, len(matchingIDs))

	levelName := pm.activeLevel
	pm.availableTitle.Label = fmt.Sprintf("Available %s Perks (%d compatible)", strings.Title(levelName), len(matchingIDs))

	for _, id := range matchingIDs {
		def := perks.GetPerkDefinition(id)
		entry := fmt.Sprintf("%s - %s", def.Name, def.Description)

		// Check if equippable and annotate issues
		if entityID != 0 {
			reason := perks.CanEquipPerk(entityID, id, 0, manager)
			if reason != "" {
				if strings.Contains(reason, "requires role") {
					entry = fmt.Sprintf("(!) %s - %s [%s]", def.Name, def.Description, reason)
				} else if strings.Contains(reason, "exclusive") {
					entry = fmt.Sprintf("(X) %s - %s [%s]", def.Name, def.Description, reason)
				}
			}
		}

		// Truncate long entries
		if len(entry) > 80 {
			entry = entry[:77] + "..."
		}

		entries = append(entries, entry)
		pm.availablePerkIDs = append(pm.availablePerkIDs, id)
	}

	if len(entries) == 0 {
		entries = append(entries, "No perks available for this level")
		pm.availablePerkIDs = append(pm.availablePerkIDs, "")
	}

	pm.selectedAvailablePerkID = ""
	pm.disableSlotButtons()

	pm.availableList = pm.replaceListInPerkContainer(pm.availableContent, pm.availableList, func() *widget.List {
		return builders.CreateSimpleStringList(builders.SimpleStringListConfig{
			Entries:       entries,
			ScreenWidth:   400,
			ScreenHeight:  200,
			WidthPercent:  1.0,
			HeightPercent: 0.5,
			OnSelect: func(selected string) {
				for i, e := range entries {
					if e == selected && i < len(pm.availablePerkIDs) {
						perkID := pm.availablePerkIDs[i]
						if perkID != "" {
							pm.selectedAvailablePerkID = perkID
							pm.refreshDetail(perkID)
							pm.enableSlotButtons()
						}
						return
					}
				}
			},
		})
	})
}

// refreshEquipped rebuilds the equipped slot list for the current entity.
func (pm *PerkMode) refreshEquipped() {
	if pm.equippedContent == nil {
		return
	}

	pm.ensurePerkComponent()

	entityID := pm.getCurrentEntityID()
	if entityID == 0 {
		pm.equippedTitle.Label = "Equipped (no entity)"
		return
	}

	manager := pm.Context.ECSManager
	slotCount := pm.getSlotCount()

	// Get equipped perk IDs per slot
	slotPerks := pm.getEquippedSlots(entityID, manager)

	equippedCount := 0
	entries := make([]string, 0, slotCount)
	entrySlotIndices := make([]int, 0, slotCount)

	for i := 0; i < slotCount; i++ {
		perkID := ""
		if i < len(slotPerks) {
			perkID = slotPerks[i]
		}

		if perkID != "" {
			def := perks.GetPerkDefinition(perkID)
			name := perkID
			if def != nil {
				name = def.Name
			}
			entries = append(entries, fmt.Sprintf("Slot %d: %s", i+1, name))
			equippedCount++
		} else {
			entries = append(entries, fmt.Sprintf("Slot %d: (empty)", i+1))
		}
		entrySlotIndices = append(entrySlotIndices, i)
	}

	pm.equippedTitle.Label = fmt.Sprintf("Equipped (%d/%d)", equippedCount, slotCount)

	pm.selectedEquipSlotIndex = -1
	if pm.unequipButton != nil {
		pm.unequipButton.GetWidget().Disabled = true
	}

	pm.equippedList = pm.replaceListInPerkContainer(pm.equippedContent, pm.equippedList, func() *widget.List {
		return builders.CreateSimpleStringList(builders.SimpleStringListConfig{
			Entries:       entries,
			ScreenWidth:   400,
			ScreenHeight:  200,
			WidthPercent:  1.0,
			HeightPercent: 0.5,
			OnSelect: func(selected string) {
				for i, e := range entries {
					if e == selected && i < len(entrySlotIndices) {
						slotIdx := entrySlotIndices[i]
						pm.selectedEquipSlotIndex = slotIdx

						// Show detail if slot has a perk
						perkID := ""
						if slotIdx < len(slotPerks) {
							perkID = slotPerks[slotIdx]
						}
						if perkID != "" {
							pm.refreshEquippedDetail(perkID)
							if pm.unequipButton != nil {
								pm.unequipButton.GetWidget().Disabled = false
							}
						} else {
							if pm.equippedDetail != nil {
								pm.equippedDetail.SetText("Empty slot.")
							}
							if pm.unequipButton != nil {
								pm.unequipButton.GetWidget().Disabled = true
							}
						}
						return
					}
				}
			},
		})
	})
}

// getEquippedSlots returns the array of equipped perk IDs for the current entity/level.
func (pm *PerkMode) getEquippedSlots(entityID ecs.EntityID, manager *common.EntityManager) []string {
	switch pm.activeLevel {
	case "squad":
		data := common.GetComponentTypeByID[*perks.SquadPerkData](manager, entityID, perks.SquadPerkComponent)
		if data != nil {
			return data.EquippedPerks[:]
		}
	case "unit":
		data := common.GetComponentTypeByID[*perks.UnitPerkData](manager, entityID, perks.UnitPerkComponent)
		if data != nil {
			return data.EquippedPerks[:]
		}
	case "commander":
		data := common.GetComponentTypeByID[*perks.CommanderPerkData](manager, entityID, perks.CommanderPerkComponent)
		if data != nil {
			return data.EquippedPerks[:]
		}
	}
	return nil
}

// refreshDetail formats and displays perk info in the available detail TextArea.
func (pm *PerkMode) refreshDetail(perkID string) {
	if pm.availableDetail == nil {
		return
	}

	def := perks.GetPerkDefinition(perkID)
	if def == nil {
		pm.availableDetail.SetText(fmt.Sprintf("Unknown perk: %s", perkID))
		return
	}

	var b strings.Builder
	b.WriteString(def.Name)
	b.WriteString(fmt.Sprintf("\nLevel: %s | Category: %s", perkLevelName(def.Level), perkCategoryName(def.Category)))
	b.WriteString(fmt.Sprintf("\n%s", def.Description))

	if len(def.StatModifiers) > 0 {
		b.WriteString("\n\nStat Modifiers:")
		for _, mod := range def.StatModifiers {
			if mod.Percent != 0 {
				b.WriteString(fmt.Sprintf("\n  +%.0f%% %s", mod.Percent*100, strings.Title(mod.Stat)))
			} else {
				sign := "+"
				if mod.Modifier < 0 {
					sign = ""
				}
				b.WriteString(fmt.Sprintf("\n  %s%d %s", sign, mod.Modifier, strings.Title(mod.Stat)))
			}
		}
	}

	if def.RoleGate != "" {
		b.WriteString(fmt.Sprintf("\nRole: %s", def.RoleGate))
	} else {
		b.WriteString("\nRole: Any")
	}

	if len(def.ExclusiveWith) > 0 {
		b.WriteString(fmt.Sprintf("\nExclusive With: %s", strings.Join(def.ExclusiveWith, ", ")))
	}

	pm.availableDetail.SetText(b.String())
}

// refreshEquippedDetail formats and displays perk info in the equipped detail TextArea.
func (pm *PerkMode) refreshEquippedDetail(perkID string) {
	if pm.equippedDetail == nil {
		return
	}

	def := perks.GetPerkDefinition(perkID)
	if def == nil {
		pm.equippedDetail.SetText(fmt.Sprintf("Unknown perk: %s", perkID))
		return
	}

	var b strings.Builder
	b.WriteString(def.Name)
	b.WriteString(fmt.Sprintf("\n%s", def.Description))

	if len(def.StatModifiers) > 0 {
		b.WriteString("\n\nStat Modifiers:")
		for _, mod := range def.StatModifiers {
			if mod.Percent != 0 {
				b.WriteString(fmt.Sprintf("\n  +%.0f%% %s", mod.Percent*100, strings.Title(mod.Stat)))
			} else {
				sign := "+"
				if mod.Modifier < 0 {
					sign = ""
				}
				b.WriteString(fmt.Sprintf("\n  %s%d %s", sign, mod.Modifier, strings.Title(mod.Stat)))
			}
		}
	}

	pm.equippedDetail.SetText(b.String())
}

// === Equip/Unequip Actions ===

// onEquipAction equips the selected perk to the given slot index.
func (pm *PerkMode) onEquipAction(slotIndex int) {
	if pm.selectedAvailablePerkID == "" {
		pm.SetStatus("Select a perk first")
		return
	}

	entityID := pm.getCurrentEntityID()
	if entityID == 0 {
		pm.SetStatus("No entity selected")
		return
	}

	manager := pm.Context.ECSManager
	err := perks.EquipPerk(entityID, pm.selectedAvailablePerkID, slotIndex, manager)
	if err != nil {
		pm.SetStatus(fmt.Sprintf("Equip failed: %v", err))
		return
	}

	def := perks.GetPerkDefinition(pm.selectedAvailablePerkID)
	name := pm.selectedAvailablePerkID
	if def != nil {
		name = def.Name
	}
	pm.SetStatus(fmt.Sprintf("Equipped %s in slot %d", name, slotIndex+1))
	pm.selectedAvailablePerkID = ""
	pm.refreshActiveTab()
}

// onUnequipAction unequips the perk from the selected slot.
func (pm *PerkMode) onUnequipAction() {
	if pm.selectedEquipSlotIndex < 0 {
		pm.SetStatus("Select a slot first")
		return
	}

	entityID := pm.getCurrentEntityID()
	if entityID == 0 {
		pm.SetStatus("No entity selected")
		return
	}

	manager := pm.Context.ECSManager

	// Check slot is not empty
	slotPerks := pm.getEquippedSlots(entityID, manager)
	if pm.selectedEquipSlotIndex >= len(slotPerks) || slotPerks[pm.selectedEquipSlotIndex] == "" {
		pm.SetStatus("Slot is already empty")
		return
	}

	perkLevel := pm.getCurrentPerkLevel()
	removedID := slotPerks[pm.selectedEquipSlotIndex]

	err := perks.UnequipPerk(entityID, perkLevel, pm.selectedEquipSlotIndex, manager)
	if err != nil {
		pm.SetStatus(fmt.Sprintf("Unequip failed: %v", err))
		return
	}

	def := perks.GetPerkDefinition(removedID)
	name := removedID
	if def != nil {
		name = def.Name
	}
	pm.SetStatus(fmt.Sprintf("Unequipped %s", name))
	pm.selectedEquipSlotIndex = -1
	pm.refreshActiveTab()
}

// === Slot Button Management ===

// rebuildSlotButtons rebuilds the equip slot buttons based on current slot count.
func (pm *PerkMode) rebuildSlotButtons() {
	if pm.slotButtonRow == nil {
		return
	}

	pm.slotButtonRow.RemoveChildren()
	slotCount := pm.getSlotCount()
	for i := 0; i < slotCount; i++ {
		slotIdx := i
		btn := builders.CreateButtonWithConfig(builders.ButtonConfig{
			Text:    fmt.Sprintf("Equip Slot %d", slotIdx+1),
			OnClick: func() { pm.onEquipAction(slotIdx) },
		})
		btn.GetWidget().Disabled = true
		pm.slotButtonRow.AddChild(btn)
	}
}

// disableSlotButtons disables all slot buttons.
func (pm *PerkMode) disableSlotButtons() {
	if pm.slotButtonRow == nil {
		return
	}
	for _, child := range pm.slotButtonRow.Children() {
		child.GetWidget().Disabled = true
	}
}

// enableSlotButtons enables all slot buttons.
func (pm *PerkMode) enableSlotButtons() {
	if pm.slotButtonRow == nil {
		return
	}
	for _, child := range pm.slotButtonRow.Children() {
		child.GetWidget().Disabled = false
	}
}

// === Helper functions ===

func perkLevelName(level perks.PerkLevel) string {
	switch level {
	case perks.PerkLevelSquad:
		return "Squad"
	case perks.PerkLevelUnit:
		return "Unit"
	case perks.PerkLevelCommander:
		return "Commander"
	default:
		return "Unknown"
	}
}

func perkCategoryName(cat perks.PerkCategory) string {
	switch cat {
	case perks.CategorySpecialization:
		return "Specialization"
	case perks.CategoryGeneralization:
		return "Generalization"
	case perks.CategoryAttackPattern:
		return "Attack Pattern"
	case perks.CategoryAttribute:
		return "Attribute"
	case perks.CategoryAttackCounter:
		return "Attack Counter"
	case perks.CategoryDepth:
		return "Depth"
	case perks.CategoryCommander:
		return "Commander"
	default:
		return "Unknown"
	}
}
