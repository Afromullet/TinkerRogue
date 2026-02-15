package guisquads

import (
	"fmt"
	"game_main/gui/builders"
	"game_main/tactical/squads"

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
	sem.updateNavigationButtons()
}

// refreshAfterCommand is called after successful command execution to update the UI
func (sem *SquadEditorMode) refreshAfterCommand() {
	sem.refreshAllUI(false)
}
