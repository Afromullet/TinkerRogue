package guisquads

import (
	"fmt"
	"sort"
	"strings"

	"game_main/gear"
	"game_main/gui/builders"
	"game_main/tactical/squads"
	"game_main/templates"

	"github.com/ebitenui/ebitenui/widget"
)

// UI refresh logic for ArtifactMode

// replaceListInContainer removes an old list widget from a container, creates a new one,
// and re-inserts it after the title label while preserving any trailing children (buttons, etc).
func (am *ArtifactMode) replaceListInContainer(
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

// refreshAllUI refreshes squad navigation and the active tab
func (am *ArtifactMode) refreshAllUI() {
	am.updateSquadCounter()
	am.refreshActiveTab()
}

// refreshActiveTab refreshes whichever tab is currently visible
func (am *ArtifactMode) refreshActiveTab() {
	switch am.activeTab {
	case "inventory":
		am.refreshInventory()
	case "equipment":
		am.refreshEquipment()
	}
}

// refreshInventory rebuilds the artifact inventory list.
// Shows all owned artifacts with status. Equip button works on available artifacts only.
func (am *ArtifactMode) refreshInventory() {
	if am.inventoryContent == nil {
		return
	}

	playerID := am.Context.PlayerData.PlayerEntityID
	inv := gear.GetPlayerArtifactInventory(playerID, am.Queries.ECSManager)
	if inv == nil {
		am.inventoryTitle.Label = "Artifacts (0/0)"
		return
	}

	current, max := inv.GetArtifactCount()
	am.inventoryTitle.Label = fmt.Sprintf("Artifacts (%d/%d)", current, max)

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
			squadName := squads.GetSquadName(info.EquippedOn, am.Queries.ECSManager)
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

	am.selectedInventoryArtifact = ""
	if am.inventoryButton != nil {
		am.inventoryButton.Text().Label = "Equip on Squad"
		am.inventoryButton.GetWidget().Disabled = true
	}

	am.inventoryList = am.replaceListInContainer(am.inventoryContent, am.inventoryList, func() *widget.List {
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
							am.selectedInventoryArtifact = artID
							if am.inventoryButton != nil {
								am.inventoryButton.GetWidget().Disabled = false
							}
						} else {
							am.selectedInventoryArtifact = ""
							if am.inventoryButton != nil {
								am.inventoryButton.GetWidget().Disabled = true
							}
						}
						// Show details for any selected artifact (even equipped ones)
						if i < len(allInstances) {
							am.refreshInventoryDetail(allInstances[i].DefinitionID)
						}
						return
					}
				}
			},
		})
	})
}

// onInventoryEquipAction equips the selected available artifact on the current squad.
func (am *ArtifactMode) onInventoryEquipAction() {
	if len(am.allSquadIDs) == 0 {
		am.SetStatus("No squad selected")
		return
	}
	if am.selectedInventoryArtifact == "" {
		am.SetStatus("Select an available artifact first")
		return
	}

	squadID := am.currentSquadID()
	playerID := am.Context.PlayerData.PlayerEntityID
	manager := am.Queries.ECSManager

	err := gear.EquipArtifact(playerID, squadID, am.selectedInventoryArtifact, manager)
	if err != nil {
		am.SetStatus(fmt.Sprintf("Equip failed: %v", err))
		return
	}

	def := templates.GetArtifactDefinition(am.selectedInventoryArtifact)
	name := am.selectedInventoryArtifact
	if def != nil {
		name = def.Name
	}
	am.SetStatus(fmt.Sprintf("Equipped %s", name))
	am.selectedInventoryArtifact = ""
	am.refreshInventory()
}

// refreshInventoryDetail displays full details for a selected artifact.
func (am *ArtifactMode) refreshInventoryDetail(artifactID string) {
	if am.inventoryDetail == nil {
		return
	}

	def := templates.GetArtifactDefinition(artifactID)
	if def == nil {
		am.inventoryDetail.SetText(fmt.Sprintf("Unknown artifact: %s", artifactID))
		return
	}

	playerID := am.Context.PlayerData.PlayerEntityID
	inv := gear.GetPlayerArtifactInventory(playerID, am.Queries.ECSManager)

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
				squadName := squads.GetSquadName(info.EquippedOn, am.Queries.ECSManager)
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

	am.inventoryDetail.SetText(b.String())
}

// refreshEquipment rebuilds the equipment list for the current squad.
// Shows only equipped artifacts with an Unequip button.
func (am *ArtifactMode) refreshEquipment() {
	if am.equipmentContent == nil || len(am.allSquadIDs) == 0 {
		return
	}

	squadID := am.currentSquadID()
	squadName := squads.GetSquadName(squadID, am.Queries.ECSManager)

	equipData := gear.GetEquipmentData(squadID, am.Queries.ECSManager)
	var equipped []string
	if equipData != nil {
		equipped = equipData.EquippedArtifacts
	}

	am.equipmentTitle.Label = fmt.Sprintf("%s - Equipment (%d/%d)", squadName, len(equipped), gear.MaxArtifactSlots)

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

	am.selectedEquippedArtifact = ""
	if am.equipmentButton != nil {
		am.equipmentButton.GetWidget().Disabled = true
	}

	am.equipmentList = am.replaceListInContainer(am.equipmentContent, am.equipmentList, func() *widget.List {
		return builders.CreateSimpleStringList(builders.SimpleStringListConfig{
			Entries:       entries,
			ScreenWidth:   400,
			ScreenHeight:  200,
			WidthPercent:  1.0,
			HeightPercent: 0.5,
			OnSelect: func(selected string) {
				for i, e := range entries {
					if e == selected && i < len(entryIDs) && entryIDs[i] != "" {
						am.selectedEquippedArtifact = entryIDs[i]
						am.refreshEquipmentDetail(entryIDs[i])
						if am.equipmentButton != nil {
							am.equipmentButton.GetWidget().Disabled = false
						}
						return
					}
				}
			},
		})
	})
}

// refreshEquipmentDetail shows artifact details in the equipment detail text area.
func (am *ArtifactMode) refreshEquipmentDetail(artifactID string) {
	if am.equipmentDetail == nil {
		return
	}

	def := templates.GetArtifactDefinition(artifactID)
	if def == nil {
		am.equipmentDetail.SetText(fmt.Sprintf("Unknown artifact: %s", artifactID))
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

	am.equipmentDetail.SetText(b.String())
}

// onEquipmentAction handles the Unequip button click on the Equipment tab.
func (am *ArtifactMode) onEquipmentAction() {
	if len(am.allSquadIDs) == 0 || am.selectedEquippedArtifact == "" {
		am.SetStatus("Select an equipped artifact first")
		return
	}

	squadID := am.currentSquadID()
	playerID := am.Context.PlayerData.PlayerEntityID
	manager := am.Queries.ECSManager

	err := gear.UnequipArtifact(playerID, squadID, am.selectedEquippedArtifact, manager)
	if err != nil {
		am.SetStatus(fmt.Sprintf("Unequip failed: %v", err))
		return
	}

	def := templates.GetArtifactDefinition(am.selectedEquippedArtifact)
	name := am.selectedEquippedArtifact
	if def != nil {
		name = def.Name
	}
	am.SetStatus(fmt.Sprintf("Unequipped %s", name))
	am.selectedEquippedArtifact = ""
	am.refreshEquipment()
}
