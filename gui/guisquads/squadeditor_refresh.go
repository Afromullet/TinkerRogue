package guisquads

import (
	"fmt"
	"sort"
	"strings"

	"game_main/gear"
	"game_main/gui/builders"
	"game_main/tactical/squads"
	"game_main/templates"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// UI refresh logic for SquadEditorMode

// replaceListInContainer removes an old list widget from a container, creates a new one,
// and re-inserts it after the title label while preserving any trailing children (buttons, etc).
func (sem *SquadEditorMode) replaceListInContainer(
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

// refreshCurrentSquad loads the current squad's data into the UI
func (sem *SquadEditorMode) refreshCurrentSquad() {
	if len(sem.allSquadIDs) == 0 {
		return
	}

	currentSquadID := sem.currentSquadID()

	// Update squad counter
	counterText := fmt.Sprintf("Squad %d of %d", sem.currentSquadIndex+1, len(sem.allSquadIDs))
	sem.squadCounterLabel.Label = counterText

	// Load squad formation into grid
	sem.loadSquadFormation(currentSquadID)

	// Refresh unit list
	sem.rebuildUnitListWidget(currentSquadID)

	// Update status
	squadName := sem.Queries.SquadCache.GetSquadName(currentSquadID)
	sem.SetStatus(fmt.Sprintf("Editing squad: %s", squadName))
}

// refreshSquadSelector updates the squad selector list (rebuilds widget)
func (sem *SquadEditorMode) refreshSquadSelector() {
	container := sem.GetPanelContainer(SquadEditorPanelSquadSelector)
	sem.squadSelector = sem.replaceListInContainer(container, sem.squadSelector, func() *widget.List {
		return builders.CreateSquadList(builders.SquadListConfig{
			SquadIDs:      sem.allSquadIDs,
			Manager:       sem.Context.ECSManager,
			ScreenWidth:   sem.Layout.ScreenWidth,
			ScreenHeight:  sem.Layout.ScreenHeight,
			WidthPercent:  0.2,
			HeightPercent: 0.4,
			OnSelect: func(squadID ecs.EntityID) {
				sem.onSquadSelected(squadID)
			},
		})
	})
}

// rebuildUnitListWidget updates the unit list for the current squad (rebuilds widget)
func (sem *SquadEditorMode) rebuildUnitListWidget(squadID ecs.EntityID) {
	unitIDs := sem.Queries.SquadCache.GetUnitIDsInSquad(squadID)

	sem.unitList = sem.replaceListInContainer(sem.unitContent, sem.unitList, func() *widget.List {
		return builders.CreateUnitList(builders.UnitListConfig{
			UnitIDs:       unitIDs,
			Manager:       sem.Queries.ECSManager,
			ScreenWidth:   400,
			ScreenHeight:  300,
			WidthPercent:  1.0,
			HeightPercent: 1.0,
		})
	})
}

// refreshRosterList updates the available units from player's roster (rebuilds widget)
func (sem *SquadEditorMode) refreshRosterList() {
	roster := squads.GetPlayerRoster(sem.Context.PlayerData.PlayerEntityID, sem.Queries.ECSManager)
	if roster == nil {
		return
	}

	entries := make([]string, 0)
	for templateName := range roster.Units {
		availableCount := roster.GetAvailableCount(templateName)
		if availableCount > 0 {
			entries = append(entries, fmt.Sprintf("%s (x%d)", templateName, availableCount))
		}
	}

	if len(entries) == 0 {
		entries = append(entries, "No units available")
	}

	sem.rosterList = sem.replaceListInContainer(sem.rosterContent, sem.rosterList, func() *widget.List {
		return builders.CreateSimpleStringList(builders.SimpleStringListConfig{
			Entries:       entries,
			ScreenWidth:   400,
			ScreenHeight:  200,
			WidthPercent:  1.0,
			HeightPercent: 1.0,
		})
	})
}

// refreshInventory rebuilds the artifact inventory list.
// Shows all owned artifacts with status. Equip button works on available artifacts only.
func (sem *SquadEditorMode) refreshInventory() {
	if sem.inventoryContent == nil {
		return
	}

	playerID := sem.Context.PlayerData.PlayerEntityID
	inv := gear.GetPlayerArtifactInventory(playerID, sem.Queries.ECSManager)
	if inv == nil {
		sem.inventoryTitle.Label = "Artifacts (0/0)"
		return
	}

	current, max := inv.GetArtifactCount()
	sem.inventoryTitle.Label = fmt.Sprintf("Artifacts (%d/%d)", current, max)

	// Get flat list of all instances and sort by definition ID + instance index
	allInstances := inv.GetAllInstances()
	sort.Slice(allInstances, func(i, j int) bool {
		if allInstances[i].DefinitionID != allInstances[j].DefinitionID {
			return allInstances[i].DefinitionID < allInstances[j].DefinitionID
		}
		return allInstances[i].InstanceIndex < allInstances[j].InstanceIndex
	})

	// Build display entries; track artifact IDs for selection
	entries := make([]string, 0, len(allInstances))
	entryIDs := make([]string, 0, len(allInstances))

	for _, info := range allInstances {
		def := templates.GetArtifactDefinition(info.DefinitionID)

		name := info.DefinitionID
		tier := ""
		if def != nil {
			name = def.Name
			tier = strings.Title(def.Tier)
		}

		copyCount := inv.GetInstanceCount(info.DefinitionID)
		if copyCount > 1 {
			name = fmt.Sprintf("%s (#%d)", name, info.InstanceIndex)
		}

		status := "Available"
		if info.EquippedOn != 0 {
			squadName := squads.GetSquadName(info.EquippedOn, sem.Queries.ECSManager)
			status = fmt.Sprintf("Equipped (%s)", squadName)
		}

		entry := fmt.Sprintf("%s [%s] - %s", name, tier, status)
		entries = append(entries, entry)
		// Only store ID for available instances (equippable)
		if info.EquippedOn == 0 {
			entryIDs = append(entryIDs, info.DefinitionID)
		} else {
			entryIDs = append(entryIDs, "")
		}
	}

	if len(entries) == 0 {
		entries = append(entries, "No artifacts owned")
		entryIDs = append(entryIDs, "")
	}

	sem.selectedInventoryArtifact = ""
	if sem.inventoryButton != nil {
		sem.inventoryButton.Text().Label = "Equip on Squad"
		sem.inventoryButton.GetWidget().Disabled = true
	}

	sem.inventoryList = sem.replaceListInContainer(sem.inventoryContent, sem.inventoryList, func() *widget.List {
		return builders.CreateSimpleStringList(builders.SimpleStringListConfig{
			Entries:       entries,
			ScreenWidth:   400,
			ScreenHeight:  200,
			WidthPercent:  1.0,
			HeightPercent: 0.5,
			OnSelect: func(selected string) {
				for i, e := range entries {
					if e == selected && i < len(entryIDs) {
						artID := entryIDs[i]
						if artID != "" {
							sem.selectedInventoryArtifact = artID
							if sem.inventoryButton != nil {
								sem.inventoryButton.GetWidget().Disabled = false
							}
						} else {
							sem.selectedInventoryArtifact = ""
							if sem.inventoryButton != nil {
								sem.inventoryButton.GetWidget().Disabled = true
							}
						}
						// Show details for any selected artifact (even equipped ones)
						if i < len(allInstances) {
							sem.refreshInventoryDetail(allInstances[i].DefinitionID)
						}
						return
					}
				}
			},
		})
	})
}

// onInventoryEquipAction equips the selected available artifact on the current squad.
func (sem *SquadEditorMode) onInventoryEquipAction() {
	if len(sem.allSquadIDs) == 0 {
		sem.SetStatus("No squad selected")
		return
	}
	if sem.selectedInventoryArtifact == "" {
		sem.SetStatus("Select an available artifact first")
		return
	}

	squadID := sem.currentSquadID()
	playerID := sem.Context.PlayerData.PlayerEntityID
	manager := sem.Queries.ECSManager

	err := gear.EquipArtifact(playerID, squadID, sem.selectedInventoryArtifact, manager)
	if err != nil {
		sem.SetStatus(fmt.Sprintf("Equip failed: %v", err))
		return
	}

	def := templates.GetArtifactDefinition(sem.selectedInventoryArtifact)
	name := sem.selectedInventoryArtifact
	if def != nil {
		name = def.Name
	}
	sem.SetStatus(fmt.Sprintf("Equipped %s", name))
	sem.selectedInventoryArtifact = ""
	sem.refreshInventory()
}

// refreshInventoryDetail displays full details for a selected artifact.
func (sem *SquadEditorMode) refreshInventoryDetail(artifactID string) {
	if sem.inventoryDetail == nil {
		return
	}

	def := templates.GetArtifactDefinition(artifactID)
	if def == nil {
		sem.inventoryDetail.SetText(fmt.Sprintf("Unknown artifact: %s", artifactID))
		return
	}

	playerID := sem.Context.PlayerData.PlayerEntityID
	inv := gear.GetPlayerArtifactInventory(playerID, sem.Queries.ECSManager)

	var b strings.Builder
	b.WriteString(def.Name)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Tier: %s", strings.Title(def.Tier)))
	b.WriteString("\n")

	// Multi-instance summary
	if inv != nil {
		totalCopies := inv.GetInstanceCount(artifactID)
		availableCount := 0
		var equippedSquads []string
		for _, info := range inv.GetAllInstances() {
			if info.DefinitionID != artifactID {
				continue
			}
			if info.EquippedOn == 0 {
				availableCount++
			} else {
				squadName := squads.GetSquadName(info.EquippedOn, sem.Queries.ECSManager)
				equippedSquads = append(equippedSquads, squadName)
			}
		}
		b.WriteString(fmt.Sprintf("Owned: %d copies, Available: %d", totalCopies, availableCount))
		if len(equippedSquads) > 0 {
			b.WriteString(fmt.Sprintf("\nEquipped on: %s", strings.Join(equippedSquads, ", ")))
		}
	}
	b.WriteString("\n\n")

	// Description
	if def.Description != "" {
		b.WriteString(def.Description)
		b.WriteString("\n\n")
	}

	// Stat modifiers
	if len(def.StatModifiers) > 0 {
		b.WriteString("Stat Modifiers:\n")
		for _, mod := range def.StatModifiers {
			sign := "+"
			if mod.Modifier < 0 {
				sign = ""
			}
			b.WriteString(fmt.Sprintf("  %s%d %s\n", sign, mod.Modifier, strings.Title(mod.Stat)))
		}
	}

	sem.inventoryDetail.SetText(b.String())
}

// refreshEquipment rebuilds the equipment list for the current squad.
// Shows only equipped artifacts with an Unequip button.
func (sem *SquadEditorMode) refreshEquipment() {
	if sem.equipmentContent == nil || len(sem.allSquadIDs) == 0 {
		return
	}

	squadID := sem.currentSquadID()
	squadName := squads.GetSquadName(squadID, sem.Queries.ECSManager)

	equipData := gear.GetEquipmentData(squadID, sem.Queries.ECSManager)
	var equipped []string
	if equipData != nil {
		equipped = equipData.EquippedArtifacts
	}

	sem.equipmentTitle.Label = fmt.Sprintf("%s - Equipment (%d/%d)", squadName, len(equipped), gear.MaxArtifactSlots)

	entries := make([]string, 0, len(equipped))
	entryIDs := make([]string, 0, len(equipped))

	for _, id := range equipped {
		def := templates.GetArtifactDefinition(id)
		name := id
		tier := ""
		if def != nil {
			name = def.Name
			tier = strings.Title(def.Tier)
		}
		entries = append(entries, fmt.Sprintf("%s [%s]", name, tier))
		entryIDs = append(entryIDs, id)
	}

	if len(entries) == 0 {
		entries = append(entries, "No artifacts equipped")
		entryIDs = append(entryIDs, "")
	}

	sem.selectedEquippedArtifact = ""
	if sem.equipmentButton != nil {
		sem.equipmentButton.GetWidget().Disabled = true
	}

	sem.equipmentList = sem.replaceListInContainer(sem.equipmentContent, sem.equipmentList, func() *widget.List {
		return builders.CreateSimpleStringList(builders.SimpleStringListConfig{
			Entries:       entries,
			ScreenWidth:   400,
			ScreenHeight:  200,
			WidthPercent:  1.0,
			HeightPercent: 0.5,
			OnSelect: func(selected string) {
				for i, e := range entries {
					if e == selected && i < len(entryIDs) && entryIDs[i] != "" {
						sem.selectedEquippedArtifact = entryIDs[i]
						sem.refreshEquipmentDetail(entryIDs[i])
						if sem.equipmentButton != nil {
							sem.equipmentButton.GetWidget().Disabled = false
						}
						return
					}
				}
			},
		})
	})
}

// refreshEquipmentDetail shows artifact details in the equipment detail text area.
func (sem *SquadEditorMode) refreshEquipmentDetail(artifactID string) {
	if sem.equipmentDetail == nil {
		return
	}

	def := templates.GetArtifactDefinition(artifactID)
	if def == nil {
		sem.equipmentDetail.SetText(fmt.Sprintf("Unknown artifact: %s", artifactID))
		return
	}

	var b strings.Builder
	b.WriteString(def.Name)
	b.WriteString(fmt.Sprintf("\nTier: %s", strings.Title(def.Tier)))

	if def.Description != "" {
		b.WriteString(fmt.Sprintf("\n\n%s", def.Description))
	}

	if len(def.StatModifiers) > 0 {
		b.WriteString("\n\nStat Modifiers:")
		for _, mod := range def.StatModifiers {
			sign := "+"
			if mod.Modifier < 0 {
				sign = ""
			}
			b.WriteString(fmt.Sprintf("\n  %s%d %s", sign, mod.Modifier, strings.Title(mod.Stat)))
		}
	}

	if def.Behavior != "" {
		b.WriteString(fmt.Sprintf("\n\nBehavior: %s", def.Behavior))
	}

	sem.equipmentDetail.SetText(b.String())
}

// onEquipmentAction handles the Unequip button click on the Equipment tab.
func (sem *SquadEditorMode) onEquipmentAction() {
	if len(sem.allSquadIDs) == 0 || sem.selectedEquippedArtifact == "" {
		sem.SetStatus("Select an equipped artifact first")
		return
	}

	squadID := sem.currentSquadID()
	playerID := sem.Context.PlayerData.PlayerEntityID
	manager := sem.Queries.ECSManager

	err := gear.UnequipArtifact(playerID, squadID, sem.selectedEquippedArtifact, manager)
	if err != nil {
		sem.SetStatus(fmt.Sprintf("Unequip failed: %v", err))
		return
	}

	def := templates.GetArtifactDefinition(sem.selectedEquippedArtifact)
	name := sem.selectedEquippedArtifact
	if def != nil {
		name = def.Name
	}
	sem.SetStatus(fmt.Sprintf("Unequipped %s", name))
	sem.selectedEquippedArtifact = ""
	sem.refreshEquipment()
}

// refreshAllUI syncs squad data and refreshes all UI elements.
// If resetIndex is true, the squad index is reset to 0.
// Otherwise, the index is clamped to valid range.
func (sem *SquadEditorMode) refreshAllUI(resetIndex bool) {
	sem.syncSquadOrderFromRoster()

	if resetIndex {
		sem.currentSquadIndex = 0
	} else if sem.currentSquadIndex >= len(sem.allSquadIDs) && len(sem.allSquadIDs) > 0 {
		sem.currentSquadIndex = 0
	}

	sem.refreshSquadSelector()
	if len(sem.allSquadIDs) > 0 {
		sem.refreshCurrentSquad()
	}
	sem.refreshRosterList()
	sem.refreshInventory()
	if sem.activeTab == "equipment" {
		sem.refreshEquipment()
	}
	sem.updateNavigationButtons()
}

// refreshAfterCommand is called after successful command execution to update the UI
func (sem *SquadEditorMode) refreshAfterCommand() {
	sem.refreshAllUI(false)
}
