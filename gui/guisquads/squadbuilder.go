package guisquads

import (
	"game_main/gui"

	"fmt"
	"game_main/gui/core"
	"game_main/squads"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadBuilderMode provides an interface for creating new squads
type SquadBuilderMode struct {
	gui.BaseMode // Embed common mode infrastructure

	// Managers and services
	gridManager       *GridEditorManager
	squadBuilderSvc   *squads.SquadBuilderService
	uiFactory         *SquadBuilderUIFactory

	// UI widgets
	gridContainer   *widget.Container
	unitPalette     *widget.List
	capacityDisplay *widget.TextArea
	squadNameInput  *widget.TextInput
	actionButtons   *widget.Container
	unitDetailsArea *widget.TextArea

	// Grid cells for UI updates
	gridCells [3][3]*GridCellButton

	// State
	currentSquadID      ecs.EntityID
	currentSquadName    string
	selectedRosterEntry *squads.UnitRosterEntry // Currently selected roster unit
}

// GridCellButton wraps a button with squad grid metadata
type GridCellButton struct {
	button        *widget.Button
	row           int
	col           int
	unitID        ecs.EntityID // 0 if empty
	rosterEntryID ecs.EntityID // Entity ID of the roster unit (0 if empty)
}

func NewSquadBuilderMode(modeManager *core.UIModeManager) *SquadBuilderMode {
	mode := &SquadBuilderMode{
		selectedRosterEntry: nil,
	}
	mode.SetModeName("squad_builder")
	mode.ModeManager = modeManager
	return mode
}

func (sbm *SquadBuilderMode) Initialize(ctx *core.UIContext) error {
	// Initialize common mode infrastructure
	sbm.InitializeBase(ctx)

	// Create squad builder service
	sbm.squadBuilderSvc = squads.NewSquadBuilderService(ctx.ECSManager)

	// Create managers
	sbm.gridManager = NewGridEditorManager(ctx.ECSManager)
	sbm.uiFactory = NewSquadBuilderUIFactory(sbm.Layout, sbm.PanelBuilders)

	// Build UI components
	sbm.buildUI()

	return nil
}

func (sbm *SquadBuilderMode) buildUI() {
	// Build grid editor and wrap buttons in GridCellButton structs
	var buttons [3][3]*widget.Button
	sbm.gridContainer, buttons = sbm.uiFactory.CreateGridPanel(func(row, col int) {
		sbm.onCellClicked(row, col)
	})

	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			sbm.gridCells[row][col] = &GridCellButton{
				button:        buttons[row][col],
				row:           row,
				col:           col,
				unitID:        0,
				rosterEntryID: 0,
			}
		}
	}
	sbm.RootContainer.AddChild(sbm.gridContainer)

	// Set grid cells in manager
	sbm.gridManager.SetGridCells(sbm.gridCells)

	// Build unit palette - will be populated in Enter()
	sbm.unitPalette = sbm.uiFactory.CreateRosterPalettePanel(func(entry interface{}) {
		if rosterEntry, ok := entry.(*squads.UnitRosterEntry); ok {
			sbm.selectedRosterEntry = rosterEntry
			sbm.updateUnitDetails()
		} else {
			// Deselect if it's a message string
			sbm.selectedRosterEntry = nil
			sbm.updateUnitDetails()
		}
	}, func() *squads.UnitRoster {
		return squads.GetPlayerRoster(sbm.Context.PlayerData.PlayerEntityID, sbm.Context.ECSManager)
	})
	sbm.RootContainer.AddChild(sbm.unitPalette)

	// Build capacity display
	sbm.capacityDisplay = sbm.uiFactory.CreateCapacityDisplay()
	sbm.RootContainer.AddChild(sbm.capacityDisplay)

	// Build unit details area
	sbm.unitDetailsArea = sbm.uiFactory.CreateDetailsPanel()
	sbm.RootContainer.AddChild(sbm.unitDetailsArea)

	// Build squad name input
	var nameInputContainer *widget.Container
	nameInputContainer, sbm.squadNameInput = sbm.uiFactory.CreateSquadNameInput(func(text string) {
		sbm.currentSquadName = text
	})
	sbm.RootContainer.AddChild(nameInputContainer)

	// Build action buttons
	sbm.actionButtons = sbm.uiFactory.CreateActionButtons(
		sbm.onCreateSquad,
		sbm.onClearGrid,
		sbm.onToggleLeader,
		sbm.handleClose,
	)
	sbm.RootContainer.AddChild(sbm.actionButtons)
}

func (sbm *SquadBuilderMode) handleClose() {
	if squadMgmtMode, exists := sbm.ModeManager.GetMode("squad_management"); exists {
		sbm.ModeManager.RequestTransition(squadMgmtMode, "Close Squad Builder")
	}
}

func (sbm *SquadBuilderMode) onCellClicked(row, col int) {
	cell := sbm.gridCells[row][col]

	// If cell is occupied, remove unit (right-click behavior or explicit remove)
	if cell.unitID != 0 {
		sbm.removeUnitFromCell(row, col)
		return
	}

	// If no unit selected from palette, do nothing
	if sbm.selectedRosterEntry == nil {
		fmt.Println("No unit selected from roster")
		return
	}

	// Place selected roster unit
	sbm.placeRosterUnitInCell(row, col, sbm.selectedRosterEntry)
}

func (sbm *SquadBuilderMode) placeRosterUnitInCell(row, col int, rosterEntry *squads.UnitRosterEntry) {
	if sbm.currentSquadID == 0 {
		// Create temporary squad if none exists
		sbm.createTemporarySquad()
	}

	// Get the unit template by name
	var unitTemplate *squads.UnitTemplate
	for i := range squads.Units {
		if squads.Units[i].Name == rosterEntry.TemplateName {
			unitTemplate = &squads.Units[i]
			break
		}
	}

	if unitTemplate == nil {
		fmt.Printf("Failed to find template for unit: %s\n", rosterEntry.TemplateName)
		return
	}

	// Get an available unit entity from the roster
	roster := squads.GetPlayerRoster(sbm.Context.PlayerData.PlayerEntityID, sbm.Context.ECSManager)
	if roster == nil {
		fmt.Printf("Failed to get roster\n")
		return
	}

	unitEntityID := roster.GetUnitEntityForTemplate(rosterEntry.TemplateName)
	if unitEntityID == 0 {
		fmt.Printf("No available units of type %s\n", rosterEntry.TemplateName)
		return
	}

	// Use service to assign roster unit to squad - handles both placement AND roster marking atomically
	result := sbm.squadBuilderSvc.AssignRosterUnitToSquad(
		sbm.Context.PlayerData.PlayerEntityID,
		sbm.currentSquadID,
		unitEntityID,
		*unitTemplate,
		row, col,
	)

	if !result.Success {
		fmt.Printf("Failed to place unit: %s\n", result.Error)
		return
	}

	// Update grid display after successful placement
	if err := sbm.gridManager.UpdateDisplayForPlacedUnit(result.PlacedUnitID, unitTemplate, row, col, result.RosterUnitID); err != nil {
		fmt.Printf("Warning: Failed to update grid display: %v\n", err)
	}

	// Refresh the unit palette to update counts
	sbm.refreshUnitPalette()

	// Update capacity display
	sbm.updateCapacityDisplay()
}

func (sbm *SquadBuilderMode) removeUnitFromCell(row, col int) {
	cell := sbm.gridCells[row][col]
	if cell.unitID == 0 {
		return
	}

	unitID := cell.unitID
	rosterEntryID := cell.rosterEntryID

	// Use service to unassign roster unit - handles both removal AND roster return atomically
	result := sbm.squadBuilderSvc.UnassignRosterUnitFromSquad(
		sbm.Context.PlayerData.PlayerEntityID,
		sbm.currentSquadID,
		rosterEntryID,
		row, col,
	)

	if !result.Success {
		fmt.Printf("Error removing unit: %s\n", result.Error)
		return
	}

	// Update grid display after successful removal
	sbm.gridManager.UpdateDisplayForRemovedUnit(unitID)

	// Refresh unit palette to show unit is available again
	sbm.refreshUnitPalette()

	// Update capacity display
	sbm.updateCapacityDisplay()
}

func (sbm *SquadBuilderMode) createTemporarySquad() {
	squadName := sbm.currentSquadName
	if squadName == "" {
		squadName = "New Squad"
	}

	// Use service to create squad
	result := sbm.squadBuilderSvc.CreateSquad(squadName)
	if result.Success {
		sbm.currentSquadID = result.SquadID
		fmt.Printf("Created temporary squad: %s (ID: %d)\n", result.SquadName, result.SquadID)
	} else {
		fmt.Printf("Failed to create squad: %s\n", result.Error)
	}
}

func (sbm *SquadBuilderMode) updateCapacityDisplay() {
	if sbm.currentSquadID == 0 {
		sbm.capacityDisplay.SetText("Capacity: 0.0 / 6.0\n(No squad created)")
		return
	}

	// Use service to get capacity info instead of direct ECS queries
	capacityInfo := sbm.squadBuilderSvc.GetCapacityInfo(sbm.currentSquadID)

	leaderStatus := "No leader"
	if capacityInfo.HasLeader {
		leaderStatus = "Leader assigned ★"
	}

	capacityText := fmt.Sprintf("Capacity: %.1f / %d\nRemaining: %.1f\n%s", capacityInfo.UsedCapacity, capacityInfo.TotalCapacity, capacityInfo.RemainingCapacity, leaderStatus)

	if capacityInfo.RemainingCapacity < 0 {
		capacityText += "\nWARNING: Over capacity!"
	}

	sbm.capacityDisplay.SetText(capacityText)
}

func (sbm *SquadBuilderMode) updateUnitDetails() {
	if sbm.selectedRosterEntry == nil {
		sbm.unitDetailsArea.SetText("Select a unit from your roster")
		return
	}

	// Get the unit template by name
	var unitTemplate *squads.UnitTemplate
	for i := range squads.Units {
		if squads.Units[i].Name == sbm.selectedRosterEntry.TemplateName {
			unitTemplate = &squads.Units[i]
			break
		}
	}

	if unitTemplate == nil {
		sbm.unitDetailsArea.SetText("Unit template not found")
		return
	}

	attr := unitTemplate.Attributes

	details := fmt.Sprintf(
		"Unit: %s\nRole: %s\n\nAttributes:\nSTR: %d\nDEX: %d\nMAG: %d\nLDR: %d\nARM: %d\nWPN: %d\n\nHP: %d\nCapacity Cost: %.1f\n\nSize: %dx%d",
		unitTemplate.Name,
		unitTemplate.Role.String(),
		attr.Strength,
		attr.Dexterity,
		attr.Magic,
		attr.Leadership,
		attr.Armor,
		attr.Weapon,
		attr.GetMaxHealth(),
		attr.GetCapacityCost(),
		unitTemplate.GridWidth,
		unitTemplate.GridHeight,
	)

	sbm.unitDetailsArea.SetText(details)
}

func (sbm *SquadBuilderMode) onCreateSquad() {
	if sbm.currentSquadID == 0 {
		fmt.Println("No squad to create - grid is empty")
		return
	}

	// Use service to get unit count instead of direct query
	unitCount := sbm.squadBuilderSvc.GetSquadUnitCount(sbm.currentSquadID)
	if unitCount == 0 {
		fmt.Println("Cannot create empty squad")
		return
	}

	// Check if a leader was designated
	leaderID := sbm.gridManager.GetLeader()
	if leaderID == 0 {
		fmt.Println("Warning: No leader designated. Please designate a leader before creating the squad.")
		return
	}

	// Update squad name if it was changed
	if sbm.currentSquadName != "" {
		sbm.squadBuilderSvc.UpdateSquadName(sbm.currentSquadID, sbm.currentSquadName)
	}

	// Designate leader using service
	leaderResult := sbm.squadBuilderSvc.DesignateLeader(leaderID)
	if leaderResult.Success {
		fmt.Printf("Unit %d designated as squad leader\n", leaderID)
	} else {
		fmt.Printf("Warning: %s\n", leaderResult.Error)
	}

	fmt.Printf("Squad created: %s with %d units\n", sbm.currentSquadName, unitCount)

	// Use service to get visualization instead of direct query
	visualization := sbm.squadBuilderSvc.GetSquadVisualization(sbm.currentSquadID)
	fmt.Println(visualization)

	// Clear the builder for next squad
	sbm.onClearGrid()
}

func (sbm *SquadBuilderMode) onToggleLeader() {
	// Find a unit to set as leader (look for first placed unit)
	var foundUnitID ecs.EntityID
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if sbm.gridManager.GetCellUnitID(row, col) != 0 {
				foundUnitID = sbm.gridManager.GetCellUnitID(row, col)
				break
			}
		}
		if foundUnitID != 0 {
			break
		}
	}

	if foundUnitID == 0 {
		fmt.Println("No units placed in squad to designate as leader")
		return
	}

	// Toggle leader status
	if sbm.gridManager.GetLeader() == foundUnitID {
		// Remove leader status
		sbm.gridManager.SetLeader(0)
		fmt.Println("Leader status removed")
	} else {
		// Set new leader
		sbm.gridManager.SetLeader(foundUnitID)
		fmt.Printf("Unit %d designated as leader\n", foundUnitID)
	}

	// Refresh grid display to show leader marker
	sbm.gridManager.RefreshGridDisplay()
	sbm.updateCapacityDisplay()
}

func (sbm *SquadBuilderMode) setUnitAsLeader(row, col int) {
	unitID := sbm.gridManager.GetCellUnitID(row, col)

	if unitID == 0 {
		fmt.Println("No unit at this position to set as leader")
		return
	}

	if sbm.gridManager.GetLeader() == unitID {
		// Unset leader
		sbm.gridManager.SetLeader(0)
		fmt.Println("Leader status removed")
	} else {
		// Set new leader
		sbm.gridManager.SetLeader(unitID)
		fmt.Printf("Unit at [%d,%d] designated as leader\n", row, col)
	}

	// Refresh grid display to show leader marker
	sbm.gridManager.RefreshGridDisplay()
	sbm.updateCapacityDisplay()
}

func (sbm *SquadBuilderMode) onClearGrid() {
	if sbm.currentSquadID == 0 {
		return
	}

	// Build map of placed units to roster units for service
	rosterUnitsMap := make(map[ecs.EntityID]ecs.EntityID)
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cell := sbm.gridCells[row][col]
			if cell.unitID != 0 && cell.rosterEntryID != 0 {
				rosterUnitsMap[cell.unitID] = cell.rosterEntryID
			}
		}
	}

	// Use service to clear squad and return all units to roster atomically
	result := sbm.squadBuilderSvc.ClearSquadAndReturnAllUnits(
		sbm.Context.PlayerData.PlayerEntityID,
		sbm.currentSquadID,
		rosterUnitsMap,
	)

	if !result.Success {
		fmt.Printf("Failed to clear grid: %s\n", result.Error)
		return
	}

	fmt.Printf("Grid cleared (%d units returned to roster)\n", result.UnitsCleared)

	// Reset state
	sbm.currentSquadID = 0
	sbm.currentSquadName = ""
	sbm.squadNameInput.SetText("")
	sbm.selectedRosterEntry = nil

	// Clear grid display
	sbm.gridManager.ClearGrid()

	// Refresh unit palette
	sbm.refreshUnitPalette()

	// Update UI
	sbm.updateCapacityDisplay()
	sbm.updateUnitDetails()
}

func (sbm *SquadBuilderMode) refreshUnitPalette() {
	// Use service to get available roster units
	availableUnits := sbm.squadBuilderSvc.GetAvailableRosterUnits(sbm.Context.PlayerData.PlayerEntityID)

	if len(availableUnits) == 0 {
		// Show message when no units available
		sbm.unitPalette.SetEntries([]interface{}{"No units available - visit Buy Units (P)"})
		return
	}

	// Convert to interface slice for the list
	entries := make([]interface{}, len(availableUnits))
	for i := range availableUnits {
		entries[i] = availableUnits[i]
	}

	sbm.unitPalette.SetEntries(entries)
}

func (sbm *SquadBuilderMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Squad Builder Mode")

	// Reset state on entry
	sbm.onClearGrid()

	// Refresh unit palette with available roster units
	sbm.refreshUnitPalette()

	return nil
}

func (sbm *SquadBuilderMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Squad Builder Mode")
	return nil
}

func (sbm *SquadBuilderMode) Update(deltaTime float64) error {
	return nil
}

func (sbm *SquadBuilderMode) Render(screen *ebiten.Image) {
	// No custom rendering needed - ebitenui handles everything
}

func (sbm *SquadBuilderMode) HandleInput(inputState *core.InputState) bool {
	// Handle common input (ESC key)
	if sbm.HandleCommonInput(inputState) {
		return true
	}

	// L key to toggle leader
	if inputState.KeysJustPressed[ebiten.KeyL] {
		sbm.onToggleLeader()
		return true
	}

	return false
}
