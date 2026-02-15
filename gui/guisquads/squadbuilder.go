package guisquads

import (
	"fmt"

	"game_main/gui/framework"
	"game_main/gui/widgets"
	"game_main/tactical/commander"
	"game_main/tactical/squads"
	"game_main/tactical/squadservices"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadBuilderMode provides an interface for creating new squads
type SquadBuilderMode struct {
	framework.BaseMode // Embed common mode infrastructure

	// Managers and services
	gridManager     *GridEditorManager
	squadBuilderSvc *squadservices.SquadBuilderService

	// Interactive widget references (stored here for refresh/access)
	// These are populated from panel registry after BuildPanels()
	squadNameInput  *widget.TextInput
	unitPalette     *widgets.CachedListWrapper
	capacityDisplay *widget.TextArea
	unitDetailsArea *widget.TextArea

	// Grid cells for UI updates (wrapped with metadata)
	gridCells [3][3]*GridCellButton

	// Commander selector
	allCommanderIDs     []ecs.EntityID
	currentCommanderIdx int
	commanderLabel      *widget.Text
	commanderPrevBtn    *widget.Button
	commanderNextBtn    *widget.Button

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

func NewSquadBuilderMode(modeManager *framework.UIModeManager) *SquadBuilderMode {
	mode := &SquadBuilderMode{
		selectedRosterEntry: nil,
	}
	mode.SetModeName("squad_builder")
	mode.SetReturnMode("squad_management") // ESC returns to squad management
	mode.ModeManager = modeManager
	mode.SetSelf(mode) // Required for panel registry building
	return mode
}

func (sbm *SquadBuilderMode) Initialize(ctx *framework.UIContext) error {
	// Create services and managers first (needed by UI builders)
	sbm.squadBuilderSvc = squadservices.NewSquadBuilderService(ctx.ECSManager)
	sbm.gridManager = NewGridEditorManager(ctx.ECSManager)

	// Build base UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&sbm.BaseMode, framework.ModeConfig{
		ModeName:   "squad_builder",
		ReturnMode: "squad_management",
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Build panels from registry
	if err := sbm.BuildPanels(
		SquadBuilderPanelCommanderSelector,
		SquadBuilderPanelNameInput,
		SquadBuilderPanelGrid,
		SquadBuilderPanelRosterPalette,
		SquadBuilderPanelCapacity,
		SquadBuilderPanelDetails,
		SquadBuilderPanelActionButtons,
	); err != nil {
		return err
	}

	// Initialize widget references from registry
	sbm.initializeWidgetReferences()

	// Set grid cells in manager
	sbm.gridManager.SetGridCells(sbm.gridCells)

	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (sbm *SquadBuilderMode) initializeWidgetReferences() {
	// Commander selector widgets
	sbm.commanderPrevBtn = framework.GetPanelWidget[*widget.Button](sbm.Panels, SquadBuilderPanelCommanderSelector, "commanderPrevBtn")
	sbm.commanderNextBtn = framework.GetPanelWidget[*widget.Button](sbm.Panels, SquadBuilderPanelCommanderSelector, "commanderNextBtn")
	sbm.commanderLabel = framework.GetPanelWidget[*widget.Text](sbm.Panels, SquadBuilderPanelCommanderSelector, "commanderLabel")

	sbm.squadNameInput = framework.GetPanelWidget[*widget.TextInput](sbm.Panels, SquadBuilderPanelNameInput, "squadNameInput")
	sbm.unitPalette = framework.GetPanelWidget[*widgets.CachedListWrapper](sbm.Panels, SquadBuilderPanelRosterPalette, "unitPalette")
	sbm.capacityDisplay = framework.GetPanelWidget[*widget.TextArea](sbm.Panels, SquadBuilderPanelCapacity, "capacityDisplay")
	sbm.unitDetailsArea = framework.GetPanelWidget[*widget.TextArea](sbm.Panels, SquadBuilderPanelDetails, "unitDetailsArea")

	// Initialize grid cells with button references
	buttons := framework.GetPanelWidget[[3][3]*widget.Button](sbm.Panels, SquadBuilderPanelGrid, "gridButtons")
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
	unitTemplate := squads.GetTemplateByName(rosterEntry.TemplateName)
	if unitTemplate == nil {
		fmt.Printf("Failed to find template for unit: %s\n", rosterEntry.TemplateName)
		return
	}

	// Get an available unit entity from the roster
	roster := squads.GetPlayerRoster(sbm.Context.PlayerData.PlayerEntityID, sbm.Queries.ECSManager)
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
	// Read name from input field instead of potentially stale currentSquadName
	squadName := sbm.squadNameInput.GetText()
	if squadName == "" {
		squadName = "New Squad"
	}
	sbm.currentSquadName = squadName

	// Create squad directly using base package function
	squadID := squads.CreateEmptySquad(sbm.Context.ECSManager, squadName)
	sbm.currentSquadID = squadID
	fmt.Printf("Created temporary squad: %s (ID: %d)\n", squadName, squadID)
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
	unitTemplate := squads.GetTemplateByName(sbm.selectedRosterEntry.TemplateName)
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

	// Add squad to the active commander's squad roster
	rosterOwnerID := sbm.Context.GetSquadRosterOwnerID()
	squadRoster := squads.GetPlayerSquadRoster(rosterOwnerID, sbm.Queries.ECSManager)
	if squadRoster != nil {
		if err := squadRoster.AddSquad(sbm.currentSquadID); err != nil {
			fmt.Printf("Warning: Failed to add squad to roster: %v\n", err)
		}
	}

	fmt.Printf("Squad created: %s with %d units\n", sbm.currentSquadName, unitCount)

	// Reset builder state for next squad WITHOUT destroying the created squad's units.
	// onClearGrid() would dispose the units we just placed, so we only reset UI state here.
	sbm.currentSquadID = 0
	sbm.currentSquadName = ""
	sbm.squadNameInput.SetText("")
	sbm.selectedRosterEntry = nil
	sbm.gridManager.ClearGrid()
	sbm.refreshUnitPalette()
	sbm.updateCapacityDisplay()
	sbm.updateUnitDetails()
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
		sbm.unitPalette.GetList().SetEntries([]interface{}{"No units available - visit Buy Units (P)"})
		sbm.unitPalette.MarkDirty() // Trigger re-render with updated entries
		return
	}

	// Convert to interface slice for the list
	entries := make([]interface{}, len(availableUnits))
	for i := range availableUnits {
		entries[i] = availableUnits[i]
	}

	sbm.unitPalette.GetList().SetEntries(entries)
	sbm.unitPalette.MarkDirty() // Trigger re-render with updated entries
}

func (sbm *SquadBuilderMode) Enter(fromMode framework.UIMode) error {
	fmt.Println("Entering Squad Builder Mode")

	// Load commander list and sync current selection
	sbm.loadCommanders()

	// Reset state on entry
	sbm.onClearGrid()

	// Refresh unit palette with available roster units
	sbm.refreshUnitPalette()

	return nil
}

func (sbm *SquadBuilderMode) Exit(toMode framework.UIMode) error {
	fmt.Println("Exiting Squad Builder Mode")
	return nil
}

func (sbm *SquadBuilderMode) Update(deltaTime float64) error {
	return nil
}

func (sbm *SquadBuilderMode) Render(screen *ebiten.Image) {
	// No custom rendering needed - ebitenui handles everything
}

func (sbm *SquadBuilderMode) HandleInput(inputState *framework.InputState) bool {
	// Handle common input (ESC key)
	if sbm.HandleCommonInput(inputState) {
		return true
	}

	// L key to toggle leader
	if inputState.KeysJustPressed[ebiten.KeyL] {
		sbm.onToggleLeader()
		return true
	}

	// Tab key cycles to next commander
	if inputState.KeysJustPressed[ebiten.KeyTab] {
		sbm.showNextCommander()
		return true
	}

	return false
}

// === Commander Selector Functions ===

// loadCommanders enumerates all commanders and finds the current one
func (sbm *SquadBuilderMode) loadCommanders() {
	sbm.allCommanderIDs = commander.GetAllCommanders(sbm.Context.PlayerData.PlayerEntityID, sbm.Context.ECSManager)

	// Find index of currently selected commander
	owState := sbm.Context.ModeCoordinator.GetOverworldState()
	selectedID := owState.SelectedCommanderID
	sbm.currentCommanderIdx = 0
	for i, id := range sbm.allCommanderIDs {
		if id == selectedID {
			sbm.currentCommanderIdx = i
			break
		}
	}

	sbm.updateCommanderLabel()
	sbm.updateCommanderButtons()
}

// showPreviousCommander cycles to the previous commander
func (sbm *SquadBuilderMode) showPreviousCommander() {
	if len(sbm.allCommanderIDs) <= 1 {
		return
	}
	sbm.currentCommanderIdx--
	if sbm.currentCommanderIdx < 0 {
		sbm.currentCommanderIdx = len(sbm.allCommanderIDs) - 1
	}
	sbm.applyCommanderSwitch()
}

// showNextCommander cycles to the next commander
func (sbm *SquadBuilderMode) showNextCommander() {
	if len(sbm.allCommanderIDs) <= 1 {
		return
	}
	sbm.currentCommanderIdx++
	if sbm.currentCommanderIdx >= len(sbm.allCommanderIDs) {
		sbm.currentCommanderIdx = 0
	}
	sbm.applyCommanderSwitch()
}

// applyCommanderSwitch updates OverworldState and refreshes builder data
func (sbm *SquadBuilderMode) applyCommanderSwitch() {
	newCommanderID := sbm.allCommanderIDs[sbm.currentCommanderIdx]

	// Update overworld state so GetSquadRosterOwnerID() returns the new commander
	owState := sbm.Context.ModeCoordinator.GetOverworldState()
	owState.SelectedCommanderID = newCommanderID

	// Clear any in-progress squad before switching
	sbm.onClearGrid()

	// Refresh unit palette for the new commander's roster
	sbm.refreshUnitPalette()

	sbm.updateCommanderLabel()
	sbm.updateCommanderButtons()

	cmdrData := commander.GetCommanderData(newCommanderID, sbm.Context.ECSManager)
	if cmdrData != nil {
		fmt.Printf("Switched to commander: %s\n", cmdrData.Name)
	}
}

// updateCommanderLabel updates the commander name display
func (sbm *SquadBuilderMode) updateCommanderLabel() {
	if sbm.commanderLabel == nil {
		return
	}
	if len(sbm.allCommanderIDs) == 0 {
		sbm.commanderLabel.Label = "No Commanders"
		return
	}
	cmdrID := sbm.allCommanderIDs[sbm.currentCommanderIdx]
	cmdrData := commander.GetCommanderData(cmdrID, sbm.Context.ECSManager)
	if cmdrData != nil {
		sbm.commanderLabel.Label = fmt.Sprintf("Commander: %s", cmdrData.Name)
	} else {
		sbm.commanderLabel.Label = "Commander: ???"
	}
}

// updateCommanderButtons disables prev/next when only one commander exists
func (sbm *SquadBuilderMode) updateCommanderButtons() {
	hasMultiple := len(sbm.allCommanderIDs) > 1
	if sbm.commanderPrevBtn != nil {
		sbm.commanderPrevBtn.GetWidget().Disabled = !hasMultiple
	}
	if sbm.commanderNextBtn != nil {
		sbm.commanderNextBtn.GetWidget().Disabled = !hasMultiple
	}
}
