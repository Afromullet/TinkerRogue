package guisquads

import (
	"game_main/gui"

	"fmt"
	"game_main/common"
	"game_main/gui/core"
	"game_main/squads"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadBuilderMode provides an interface for creating new squads
type SquadBuilderMode struct {
	gui.BaseMode // Embed common mode infrastructure

	// Managers
	gridManager *GridEditorManager
	uiFactory   *SquadBuilderUIFactory

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
	currentSquadID   ecs.EntityID
	currentSquadName string
	selectedUnitIdx  int // Index in Units array
}

// GridCellButton wraps a button with squad grid metadata
type GridCellButton struct {
	button    *widget.Button
	row       int
	col       int
	unitID    ecs.EntityID // 0 if empty
	unitIndex int          // -1 if empty, otherwise index in squads.Units
}

func NewSquadBuilderMode(modeManager *core.UIModeManager) *SquadBuilderMode {
	mode := &SquadBuilderMode{
		selectedUnitIdx: -1,
	}
	mode.SetModeName("squad_builder")
	mode.ModeManager = modeManager
	return mode
}

func (sbm *SquadBuilderMode) Initialize(ctx *core.UIContext) error {
	// Initialize common mode infrastructure
	sbm.InitializeBase(ctx)

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
				button:    buttons[row][col],
				row:       row,
				col:       col,
				unitID:    0,
				unitIndex: -1,
			}
		}
	}
	sbm.RootContainer.AddChild(sbm.gridContainer)

	// Set grid cells in manager
	sbm.gridManager.SetGridCells(sbm.gridCells)

	// Build unit palette
	sbm.unitPalette = sbm.uiFactory.CreatePalettePanel(func(entry interface{}) {
		if entryStr, ok := entry.(string); ok {
			for i, unit := range squads.Units {
				expectedStr := fmt.Sprintf("%s (%s)", unit.Name, unit.Role.String())
				if entryStr == expectedStr {
					sbm.selectedUnitIdx = i
					sbm.updateUnitDetails()
					return
				}
			}
			sbm.selectedUnitIdx = -1
			sbm.updateUnitDetails()
		}
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
	if exploreMode, exists := sbm.ModeManager.GetMode("exploration"); exists {
		sbm.ModeManager.RequestTransition(exploreMode, "Close Squad Builder")
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
	if sbm.selectedUnitIdx < 0 || sbm.selectedUnitIdx >= len(squads.Units) {
		fmt.Println("No unit selected from palette")
		return
	}

	// Place selected unit
	sbm.placeUnitInCell(row, col, sbm.selectedUnitIdx)
}

func (sbm *SquadBuilderMode) placeUnitInCell(row, col, unitIndex int) {
	if sbm.currentSquadID == 0 {
		// Create temporary squad if none exists
		sbm.createTemporarySquad()
	}

	// Delegate to grid manager
	err := sbm.gridManager.PlaceUnitInCell(row, col, unitIndex, sbm.currentSquadID)
	if err != nil {
		fmt.Printf("Failed to place unit: %v\n", err)
		return
	}

	// Update capacity display
	sbm.updateCapacityDisplay()
}

func (sbm *SquadBuilderMode) removeUnitFromCell(row, col int) {
	// Delegate to grid manager
	err := sbm.gridManager.RemoveUnitFromCell(row, col)
	if err != nil {
		fmt.Printf("Error removing unit: %v\n", err)
		return
	}

	// Update capacity display
	sbm.updateCapacityDisplay()
}

func (sbm *SquadBuilderMode) createTemporarySquad() {
	squadName := sbm.currentSquadName
	if squadName == "" {
		squadName = "New Squad"
	}

	squads.CreateEmptySquad(sbm.Context.ECSManager, squadName)

	// Find the newly created squad
	// Squads are created with SquadComponent, so we can query for it
	allSquads := sbm.Queries.FindAllSquads()
	if len(allSquads) > 0 {
		sbm.currentSquadID = allSquads[len(allSquads)-1] // Get most recent
		fmt.Printf("Created temporary squad: %s (ID: %d)\n", squadName, sbm.currentSquadID)
	}
}

func (sbm *SquadBuilderMode) updateCapacityDisplay() {
	if sbm.currentSquadID == 0 {
		sbm.capacityDisplay.SetText("Capacity: 0.0 / 6.0\n(No squad created)")
		return
	}

	used := squads.GetSquadUsedCapacity(sbm.currentSquadID, sbm.Context.ECSManager)
	total := squads.GetSquadTotalCapacity(sbm.currentSquadID, sbm.Context.ECSManager)
	remaining := float64(total) - used

	leaderStatus := "No leader"
	if sbm.gridManager.GetLeader() != 0 {
		leaderStatus = "Leader assigned ★"
	}

	capacityText := fmt.Sprintf("Capacity: %.1f / %d\nRemaining: %.1f\n%s", used, total, remaining, leaderStatus)

	if remaining < 0 {
		capacityText += "\nWARNING: Over capacity!"
	}

	sbm.capacityDisplay.SetText(capacityText)
}

func (sbm *SquadBuilderMode) updateUnitDetails() {
	if sbm.selectedUnitIdx < 0 || sbm.selectedUnitIdx >= len(squads.Units) {
		sbm.unitDetailsArea.SetText("Select a unit to view details")
		return
	}

	unit := squads.Units[sbm.selectedUnitIdx]
	attr := unit.Attributes

	details := fmt.Sprintf(
		"Unit: %s\nRole: %s\n\nAttributes:\nSTR: %d\nDEX: %d\nMAG: %d\nLDR: %d\nARM: %d\nWPN: %d\n\nHP: %d\nCapacity Cost: %.1f\n\nSize: %dx%d",
		unit.Name,
		unit.Role.String(),
		attr.Strength,
		attr.Dexterity,
		attr.Magic,
		attr.Leadership,
		attr.Armor,
		attr.Weapon,
		attr.GetMaxHealth(),
		attr.GetCapacityCost(),
		unit.GridWidth,
		unit.GridHeight,
	)

	sbm.unitDetailsArea.SetText(details)
}

func (sbm *SquadBuilderMode) onCreateSquad() {
	if sbm.currentSquadID == 0 {
		fmt.Println("No squad to create - grid is empty")
		return
	}

	// Check if squad has at least one unit
	unitIDs := squads.GetUnitIDsInSquad(sbm.currentSquadID, sbm.Context.ECSManager)
	if len(unitIDs) == 0 {
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
	squadEntity := squads.GetSquadEntity(sbm.currentSquadID, sbm.Context.ECSManager)
	if squadEntity != nil {
		squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
		squadData.Name = sbm.currentSquadName
	}

	// Assign leader component to the designated unit
	// Check if leader exists first
	if sbm.Context.ECSManager.HasComponentByIDWithTag(leaderID, squads.SquadMemberTag, squads.SquadMemberComponent) {
		// Need to get the entity to add component (AddComponent is not available via ID)
		leaderEntity := common.FindEntityByIDWithTag(sbm.Context.ECSManager, leaderID, squads.SquadMemberTag)
		if leaderEntity != nil {
			// Add LeaderComponent to designate this unit as leader
			leaderEntity.AddComponent(squads.LeaderComponent, &squads.LeaderData{})
			fmt.Printf("Unit %d designated as squad leader\n", leaderID)
		}
	} else {
		fmt.Println("Warning: Could not find designated leader unit")
	}

	fmt.Printf("Squad created: %s with %d units\n", sbm.currentSquadName, len(unitIDs))

	// Visualize the squad
	visualization := squads.VisualizeSquad(sbm.currentSquadID, sbm.Context.ECSManager)
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
	// Reset state
	sbm.currentSquadID = 0
	sbm.currentSquadName = ""
	sbm.squadNameInput.SetText("")
	sbm.selectedUnitIdx = -1

	// Clear grid using manager
	sbm.gridManager.ClearGrid()

	// Update UI
	sbm.updateCapacityDisplay()
	sbm.updateUnitDetails()

	fmt.Println("Grid cleared")
}

func (sbm *SquadBuilderMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Squad Builder Mode")

	// Reset state on entry
	sbm.onClearGrid()

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
