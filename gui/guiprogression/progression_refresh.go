package guiprogression

import (
	"fmt"
	"game_main/tactical/powers/progression"
)

// UI refresh logic for ProgressionMode.
//
// Lists are wrapped in CachedListWrapper; callers must invoke MarkDirty()
// after SetEntries so the next frame re-renders the cached image.

// refresh updates header totals and both library panels.
func (pc *progressionController) refresh() {
	data := progression.GetProgression(pc.mode.activePlayerID(), pc.mode.Context.ECSManager)
	arcana, skill := 0, 0
	if data != nil {
		arcana = data.ArcanaPoints
		skill = data.SkillPoints
	}
	pc.arcanaLabel.Label = fmt.Sprintf("Arcana: %d", arcana)
	pc.skillLabel.Label = fmt.Sprintf("Skill: %d", skill)
	pc.perkPanel.refresh()
	pc.spellPanel.refresh()
}

// refresh repopulates a single library panel's lists and resets selection state.
func (pc *libraryPanelController) refresh() {
	playerID := pc.mode.activePlayerID()
	manager := pc.mode.Context.ECSManager
	unlocked := []interface{}{}
	locked := []interface{}{}

	for _, item := range pc.source.allItems() {
		entry := &libraryEntry{item: item, label: item.name}
		if pc.source.isUnlocked(playerID, item.id, manager) {
			unlocked = append(unlocked, entry)
			continue
		}
		entry.label = fmt.Sprintf("%s (%d)", item.name, item.unlockCost)
		locked = append(locked, entry)
	}

	pc.unlockedList.GetList().SetEntries(unlocked)
	pc.unlockedList.MarkDirty()
	pc.lockedList.GetList().SetEntries(locked)
	pc.lockedList.MarkDirty()

	pc.selected = nil
	pc.unlockBtn.GetWidget().Disabled = true
	pc.detail.SetText(pc.source.detailPrompt)
}
