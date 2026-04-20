package guisquads

import (
	"fmt"
	"game_main/gui/builders"
	"game_main/gui/guiunitview"
	"game_main/tactical/combat/combattypes"
	rstr "game_main/tactical/squads/roster"
	"game_main/tactical/squads/squadcommands"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// Roster management logic for SquadEditorMode

// defaultDialogPosition returns consistent dialog positioning values
func (sem *SquadEditorMode) defaultDialogPosition() (width, height, centerX, centerY int) {
	return 400, 200, sem.Layout.ScreenWidth / 2, sem.Layout.ScreenHeight / 3
}

// onNewSquad creates a new empty squad and adds it to the active commander's roster
func (sem *SquadEditorMode) onNewSquad() {
	rosterOwnerID := sem.Context.GetSquadRosterOwnerID()
	squadRoster := rstr.GetPlayerSquadRoster(rosterOwnerID, sem.Queries.ECSManager)
	if squadRoster == nil {
		sem.SetStatus("No squad roster found")
		return
	}

	if !squadRoster.CanAddSquad() {
		current, max := squadRoster.GetSquadCount()
		sem.SetStatus(fmt.Sprintf("Squad roster full (%d/%d)", current, max))
		return
	}

	// Show text input dialog for squad name
	dialog := builders.CreateTextInputDialog(builders.TextInputDialogConfig{
		Title:       "New Squad",
		Message:     "Enter squad name:",
		Placeholder: "Squad name",
		InitialText: "",
		OnConfirm: func(name string) {
			if name == "" {
				sem.SetStatus("Squad creation cancelled")
				return
			}

			// Create empty squad and add to roster
			squadID := squadcore.CreateEmptySquad(sem.Context.ECSManager, name)
			if err := squadRoster.AddSquad(squadID); err != nil {
				sem.SetStatus(fmt.Sprintf("Failed to add squad: %v", err))
				return
			}

			// Refresh UI
			sem.squadNav.Load(sem.Context.GetSquadRosterOwnerID(), sem.Context.ECSManager)
			sem.refreshSquadSelector()

			// Auto-select the new squad
			sem.squadNav.SelectByID(squadID)

			sem.refreshCurrentSquad()
			sem.refreshRosterList()
			sem.SetStatus(fmt.Sprintf("Created squad: %s", name))
		},
		OnCancel: func() {
			sem.SetStatus("Squad creation cancelled")
		},
	})

	sem.GetEbitenUI().AddWindow(dialog)
}

// tryAddSelectedRosterUnitToCell executes an AddUnitCommand using the currently
// selected roster entry and the given cell coordinates. Returns true if the
// command was dispatched. Does not alter sem.selectedGridCell.
func (sem *SquadEditorMode) tryAddSelectedRosterUnitToCell(row, col int) bool {
	if !sem.squadNav.HasSquads() {
		sem.SetStatus("No squad selected")
		return false
	}
	selectedEntry := sem.rosterList.SelectedEntry()
	if selectedEntry == nil {
		return false
	}
	entry, ok := selectedEntry.(rstr.RosterUnitEntry)
	if !ok || entry.TemplateName == "" {
		return false
	}

	cmd := squadcommands.NewAddUnitCommand(
		sem.Queries.ECSManager,
		sem.Context.PlayerData.PlayerEntityID,
		sem.squadNav.CurrentID(),
		entry.TemplateName,
		row, col,
	)
	sem.CommandHistory.Execute(cmd)
	return true
}

// onAddUnitFromRoster adds a unit from the roster to the squad
func (sem *SquadEditorMode) onAddUnitFromRoster() {
	if sem.selectedGridCell == nil {
		sem.SetStatus("No grid cell selected. Click an empty cell first")
		return
	}
	if sem.rosterList.SelectedEntry() == nil {
		sem.SetStatus("No unit selected from roster")
		return
	}
	if !sem.tryAddSelectedRosterUnitToCell(sem.selectedGridCell.Row, sem.selectedGridCell.Col) {
		sem.SetStatus("Invalid roster selection")
		return
	}
	sem.selectedGridCell = nil
}

// getSelectedUnitForAction validates that a squad and unit are selected, returning the UnitIdentity.
// Returns false if any guard fails (status message is set automatically).
func (sem *SquadEditorMode) getSelectedUnitForAction() (combattypes.UnitIdentity, bool) {
	if !sem.squadNav.HasSquads() {
		sem.SetStatus("No squad selected")
		return combattypes.UnitIdentity{}, false
	}

	selectedEntry := sem.unitList.SelectedEntry()
	if selectedEntry == nil {
		sem.SetStatus("No unit selected")
		return combattypes.UnitIdentity{}, false
	}

	unitIdentity, ok := selectedEntry.(combattypes.UnitIdentity)
	if !ok {
		sem.SetStatus("Invalid unit selection")
		return combattypes.UnitIdentity{}, false
	}
	return unitIdentity, true
}

// onRemoveUnit removes the selected unit from the squad (via unit list selection)
func (sem *SquadEditorMode) onRemoveUnit() {
	unitIdentity, ok := sem.getSelectedUnitForAction()
	if !ok {
		return
	}
	sem.removeUnitByID(unitIdentity.ID, unitIdentity.Name)
}

// removeUnitByID removes the specified unit from the current squad with a confirmation dialog.
// This is the shared logic used by both the "Remove Selected Unit" button and right-click on grid.
func (sem *SquadEditorMode) removeUnitByID(unitID ecs.EntityID, unitName string) {
	if !sem.squadNav.HasSquads() {
		sem.SetStatus("No squad selected")
		return
	}

	// Check if this is the leader
	isLeader := sem.Queries.ECSManager.HasComponent(unitID, squadcore.LeaderComponent)
	if isLeader {
		sem.SetStatus("Cannot remove leader. Make another unit leader first")
		return
	}

	currentSquadID := sem.squadNav.CurrentID()
	dialogWidth, dialogHeight, centerX, centerY := sem.defaultDialogPosition()

	// Show confirmation dialog
	dialog := builders.CreateConfirmationDialog(builders.DialogConfig{
		Title:     "Confirm Remove Unit",
		Message:   fmt.Sprintf("Remove '%s' from squad?\n\nUnit will return to roster.\n\nYou can undo with Ctrl+Z.", unitName),
		MinWidth:  dialogWidth,
		MinHeight: dialogHeight,
		CenterX:   centerX,
		CenterY:   centerY,
		OnConfirm: func() {
			cmd := squadcommands.NewRemoveUnitCommand(
				sem.Queries.ECSManager,
				sem.Context.PlayerData.PlayerEntityID,
				currentSquadID,
				unitID,
			)

			sem.CommandHistory.Execute(cmd)
		},
		OnCancel: func() {
			sem.SetStatus("Remove cancelled")
		},
	})

	sem.GetEbitenUI().AddWindow(dialog)
}

// onMakeLeader changes the squad leader to the selected unit
func (sem *SquadEditorMode) onMakeLeader() {
	unitIdentity, ok := sem.getSelectedUnitForAction()
	if !ok {
		return
	}
	unitID := unitIdentity.ID

	// Check if already leader
	isLeader := sem.Queries.ECSManager.HasComponent(unitID, squadcore.LeaderComponent)
	if isLeader {
		sem.SetStatus("Unit is already the leader")
		return
	}

	currentSquadID := sem.squadNav.CurrentID()
	dialogWidth, dialogHeight, centerX, centerY := sem.defaultDialogPosition()

	// Show confirmation dialog
	dialog := builders.CreateConfirmationDialog(builders.DialogConfig{
		Title:     "Confirm Change Leader",
		Message:   fmt.Sprintf("Make '%s' the new squad leader?\n\nYou can undo with Ctrl+Z.", unitIdentity.Name),
		MinWidth:  dialogWidth,
		MinHeight: dialogHeight,
		CenterX:   centerX,
		CenterY:   centerY,
		OnConfirm: func() {
			cmd := squadcommands.NewChangeLeaderCommand(
				sem.Queries.ECSManager,
				currentSquadID,
				unitID,
			)

			sem.CommandHistory.Execute(cmd)
		},
		OnCancel: func() {
			sem.SetStatus("Change leader cancelled")
		},
	})

	sem.GetEbitenUI().AddWindow(dialog)
}

// onViewUnit transitions to the unit view mode for the selected unit (via unit list selection)
func (sem *SquadEditorMode) onViewUnit() {
	unitIdentity, ok := sem.getSelectedUnitForAction()
	if !ok {
		return
	}
	sem.viewUnitByID(unitIdentity.ID)
}

// viewUnitByID transitions to the unit view mode for the specified unit.
// This is the shared logic used by both the "View Unit" button and shift+click on grid.
func (sem *SquadEditorMode) viewUnitByID(unitID ecs.EntityID) {
	mode, exists := sem.ModeManager.GetMode("unit_view")
	if !exists {
		sem.SetStatus("Unit view mode not available")
		return
	}

	viewMode := mode.(*guiunitview.UnitViewMode)
	viewMode.SetUnitID(unitID)
	sem.ModeManager.RequestTransition(mode, "View Unit clicked")
}

// onRenameSquad prompts for a new name and executes RenameSquadCommand
func (sem *SquadEditorMode) onRenameSquad() {
	if !sem.squadNav.HasSquads() {
		sem.SetStatus("No squad selected")
		return
	}

	currentSquadID := sem.squadNav.CurrentID()
	currentName := sem.Queries.SquadCache.GetSquadName(currentSquadID)

	// Show text input dialog
	dialog := builders.CreateTextInputDialog(builders.TextInputDialogConfig{
		Title:       "Rename Squad",
		Message:     "Enter new squad name:",
		Placeholder: "Squad name",
		InitialText: currentName,
		OnConfirm: func(newName string) {
			if newName == "" || newName == currentName {
				sem.SetStatus("Rename cancelled")
				return
			}

			// Create and execute rename command
			cmd := squadcommands.NewRenameSquadCommand(
				sem.Queries.ECSManager,
				currentSquadID,
				newName,
			)

			sem.CommandHistory.Execute(cmd)
		},
		OnCancel: func() {
			sem.SetStatus("Rename cancelled")
		},
	})

	sem.GetEbitenUI().AddWindow(dialog)
}

// backfillRosterWithSquadUnits registers all existing squad units in the roster
// This is called when entering the mode to handle units created before roster tracking
func (sem *SquadEditorMode) backfillRosterWithSquadUnits() {
	roster := rstr.GetPlayerRoster(sem.Context.PlayerData.PlayerEntityID, sem.Queries.ECSManager)
	if roster == nil {
		return
	}

	// Get squads from the active commander's roster (not all squads globally)
	rosterOwnerID := sem.Context.GetSquadRosterOwnerID()
	squadRoster := rstr.GetPlayerSquadRoster(rosterOwnerID, sem.Queries.ECSManager)
	if squadRoster == nil {
		return
	}
	allSquads := squadRoster.OwnedSquads

	for _, squadID := range allSquads {
		// Get all units in this squad
		unitIDs := sem.Queries.SquadCache.GetUnitIDsInSquad(squadID)

		for _, unitID := range unitIDs {
			// Check if unit is already in roster
			alreadyRegistered := false
			for _, entry := range roster.Units {
				for _, existingID := range entry.UnitEntities {
					if existingID == unitID {
						alreadyRegistered = true
						break
					}
				}
				if alreadyRegistered {
					break
				}
			}

			// Register if not already in roster
			if !alreadyRegistered {
				err := rstr.RegisterSquadUnitInRoster(roster, unitID, squadID, sem.Queries.ECSManager)
				if err != nil {
					fmt.Printf("Warning: Failed to register unit %d in roster: %v\n", unitID, err)
				}
			}
		}
	}
}
