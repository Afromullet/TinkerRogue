package guisquads

import (
	"fmt"
	"game_main/gui/builders"
	"game_main/tactical/squadcommands"
	"game_main/tactical/squads"
)

// Roster management logic for SquadEditorMode

// defaultDialogPosition returns consistent dialog positioning values
func (sem *SquadEditorMode) defaultDialogPosition() (width, height, centerX, centerY int) {
	return 400, 200, sem.Layout.ScreenWidth / 2, sem.Layout.ScreenHeight / 3
}

// onNewSquad creates a new empty squad and adds it to the active commander's roster
func (sem *SquadEditorMode) onNewSquad() {
	rosterOwnerID := sem.Context.GetSquadRosterOwnerID()
	squadRoster := squads.GetPlayerSquadRoster(rosterOwnerID, sem.Queries.ECSManager)
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
			squadID := squads.CreateEmptySquad(sem.Context.ECSManager, name)
			if err := squadRoster.AddSquad(squadID); err != nil {
				sem.SetStatus(fmt.Sprintf("Failed to add squad: %v", err))
				return
			}

			// Refresh UI
			sem.syncSquadOrderFromRoster()
			sem.refreshSquadSelector()

			// Auto-select the new squad (it's the last one added)
			for i, id := range sem.allSquadIDs {
				if id == squadID {
					sem.currentSquadIndex = i
					break
				}
			}

			sem.refreshCurrentSquad()
			sem.updateNavigationButtons()
			sem.refreshRosterList()
			sem.SetStatus(fmt.Sprintf("Created squad: %s", name))
		},
		OnCancel: func() {
			sem.SetStatus("Squad creation cancelled")
		},
	})

	sem.GetEbitenUI().AddWindow(dialog)
}

// onAddUnitFromRoster adds a unit from the roster to the squad
func (sem *SquadEditorMode) onAddUnitFromRoster() {
	if len(sem.allSquadIDs) == 0 {
		sem.SetStatus("No squad selected")
		return
	}

	selectedEntry := sem.rosterList.SelectedEntry()
	if selectedEntry == nil {
		sem.SetStatus("No unit selected from roster")
		return
	}

	if sem.selectedGridCell == nil {
		sem.SetStatus("No grid cell selected. Click an empty cell first")
		return
	}

	// Parse template name from entry (format: "TemplateName (xN)")
	entryStr, ok := selectedEntry.(string)
	if !ok {
		sem.SetStatus("Invalid roster selection")
		return
	}
	if entryStr == "No units available" {
		return
	}

	// Extract template name (everything before " (x")
	templateName := entryStr
	for i, c := range entryStr {
		if c == ' ' && i+1 < len(entryStr) && entryStr[i+1] == '(' {
			templateName = entryStr[:i]
			break
		}
	}

	currentSquadID := sem.currentSquadID()

	// Create and execute add unit command
	cmd := squadcommands.NewAddUnitCommand(
		sem.Queries.ECSManager,
		sem.Context.PlayerData.PlayerEntityID,
		currentSquadID,
		templateName,
		sem.selectedGridCell.Row,
		sem.selectedGridCell.Col,
	)

	sem.CommandHistory.Execute(cmd)
	sem.selectedGridCell = nil
}

// getSelectedUnitForAction validates that a squad and unit are selected, returning the UnitIdentity.
// Returns false if any guard fails (status message is set automatically).
func (sem *SquadEditorMode) getSelectedUnitForAction() (squads.UnitIdentity, bool) {
	if len(sem.allSquadIDs) == 0 {
		sem.SetStatus("No squad selected")
		return squads.UnitIdentity{}, false
	}

	selectedEntry := sem.unitList.SelectedEntry()
	if selectedEntry == nil {
		sem.SetStatus("No unit selected")
		return squads.UnitIdentity{}, false
	}

	unitIdentity, ok := selectedEntry.(squads.UnitIdentity)
	if !ok {
		sem.SetStatus("Invalid unit selection")
		return squads.UnitIdentity{}, false
	}
	return unitIdentity, true
}

// onRemoveUnit removes the selected unit from the squad
func (sem *SquadEditorMode) onRemoveUnit() {
	unitIdentity, ok := sem.getSelectedUnitForAction()
	if !ok {
		return
	}
	unitID := unitIdentity.ID

	// Check if this is the leader
	isLeader := sem.Queries.ECSManager.HasComponent(unitID, squads.LeaderComponent)
	if isLeader {
		sem.SetStatus("Cannot remove leader. Make another unit leader first")
		return
	}

	currentSquadID := sem.currentSquadID()
	dialogWidth, dialogHeight, centerX, centerY := sem.defaultDialogPosition()

	// Show confirmation dialog
	dialog := builders.CreateConfirmationDialog(builders.DialogConfig{
		Title:     "Confirm Remove Unit",
		Message:   fmt.Sprintf("Remove '%s' from squad?\n\nUnit will return to roster.\n\nYou can undo with Ctrl+Z.", unitIdentity.Name),
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
	isLeader := sem.Queries.ECSManager.HasComponent(unitID, squads.LeaderComponent)
	if isLeader {
		sem.SetStatus("Unit is already the leader")
		return
	}

	currentSquadID := sem.currentSquadID()
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

// onRenameSquad prompts for a new name and executes RenameSquadCommand
func (sem *SquadEditorMode) onRenameSquad() {
	if len(sem.allSquadIDs) == 0 {
		sem.SetStatus("No squad selected")
		return
	}

	currentSquadID := sem.currentSquadID()
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
	roster := squads.GetPlayerRoster(sem.Context.PlayerData.PlayerEntityID, sem.Queries.ECSManager)
	if roster == nil {
		return
	}

	// Get squads from the active commander's roster (not all squads globally)
	rosterOwnerID := sem.Context.GetSquadRosterOwnerID()
	squadRoster := squads.GetPlayerSquadRoster(rosterOwnerID, sem.Queries.ECSManager)
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
				err := squads.RegisterSquadUnitInRoster(roster, unitID, squadID, sem.Queries.ECSManager)
				if err != nil {
					fmt.Printf("Warning: Failed to register unit %d in roster: %v\n", unitID, err)
				}
			}
		}
	}
}
