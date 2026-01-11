package guisquads

import (
	"fmt"
	"game_main/gui/builders"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// UI refresh logic for SquadEditorMode

// refreshCurrentSquad loads the current squad's data into the UI
func (sem *SquadEditorMode) refreshCurrentSquad() {
	if len(sem.allSquadIDs) == 0 {
		return
	}

	currentSquadID := sem.allSquadIDs[sem.currentSquadIndex]

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
	// Get container from panel registry
	container := sem.GetPanelContainer(SquadEditorPanelSquadSelector)
	if container == nil {
		return
	}

	// Remove old list widget
	container.RemoveChild(sem.squadSelector)

	// Create new squad list
	sem.squadSelector = builders.CreateSquadList(builders.SquadListConfig{
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

	// Insert at position 1 (after title label)
	children := container.Children()
	container.RemoveChildren()
	container.AddChild(children[0]) // Title label
	container.AddChild(sem.squadSelector)
}

// rebuildUnitListWidget updates the unit list for the current squad (rebuilds widget)
func (sem *SquadEditorMode) rebuildUnitListWidget(squadID ecs.EntityID) {
	// Get container from panel registry
	container := sem.GetPanelContainer(SquadEditorPanelUnitList)
	if container == nil {
		return
	}

	unitIDs := sem.Queries.SquadCache.GetUnitIDsInSquad(squadID)

	// Remove old list widget
	container.RemoveChild(sem.unitList)

	// Create new unit list
	sem.unitList = builders.CreateUnitList(builders.UnitListConfig{
		UnitIDs:       unitIDs,
		Manager:       sem.Queries.ECSManager,
		ScreenWidth:   400,
		ScreenHeight:  300,
		WidthPercent:  1.0,
		HeightPercent: 1.0,
	})

	// Insert at position 1 (after title label)
	children := container.Children()
	container.RemoveChildren()
	container.AddChild(children[0]) // Title label
	container.AddChild(sem.unitList)
	for i := 1; i < len(children); i++ {
		container.AddChild(children[i])
	}
}

// refreshRosterList updates the available units from player's roster (rebuilds widget)
func (sem *SquadEditorMode) refreshRosterList() {
	// Get container from panel registry
	container := sem.GetPanelContainer(SquadEditorPanelRosterList)
	if container == nil {
		return
	}

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

	// Remove old list widget
	container.RemoveChild(sem.rosterList)

	// Create new roster list
	sem.rosterList = builders.CreateSimpleStringList(builders.SimpleStringListConfig{
		Entries:       entries,
		ScreenWidth:   400,
		ScreenHeight:  200,
		WidthPercent:  1.0,
		HeightPercent: 1.0,
	})

	// Insert at position 1 (after title label)
	children := container.Children()
	container.RemoveChildren()
	container.AddChild(children[0]) // Title label
	container.AddChild(sem.rosterList)
	for i := 1; i < len(children); i++ {
		container.AddChild(children[i])
	}
}

// refreshAfterUndoRedo is called after successful undo/redo operations
func (sem *SquadEditorMode) refreshAfterUndoRedo() {
	// Refresh squad list (squads might have been created/destroyed or renamed)
	sem.allSquadIDs = sem.Queries.SquadCache.FindAllSquads()

	// Adjust index if needed
	if sem.currentSquadIndex >= len(sem.allSquadIDs) && len(sem.allSquadIDs) > 0 {
		sem.currentSquadIndex = 0
	}

	// Refresh all UI elements (squad list, formation grid, status)
	sem.refreshSquadSelector()
	sem.refreshCurrentSquad()
	sem.refreshRosterList()
	sem.updateNavigationButtons()
}
