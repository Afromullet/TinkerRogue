package guisquads

import (
	"fmt"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/tactical/commander"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadEditorMode provides comprehensive squad editing capabilities:
// - Select squads
// - Add units from roster
// - Remove units from squad
// - Change squad leader
// - Change unit positions in grid
//
// Code organization:
// - squadeditormode.go: Lifecycle, panel building, navigation
// - squadeditor_panels_registry.go: Panel registrations via init()
// - squadeditor_grid.go: Grid cell interaction
// - squadeditor_roster.go: Roster/unit management actions
// - squadeditor_refresh.go: UI refresh logic
type SquadEditorMode struct {
	framework.BaseMode // Embed common mode infrastructure

	// Squad navigation
	currentSquadIndex int
	allSquadIDs       []ecs.EntityID

	// Interactive widget references (stored here for refresh/access)
	// These are populated from panel registry after BuildPanels()
	squadSelector     *widget.List
	gridCells         [3][3]*widget.Button
	unitList          *widget.List
	rosterList        *widget.List
	squadCounterLabel *widget.Text
	prevButton        *widget.Button
	nextButton        *widget.Button

	// Commander selector
	commanderSelector *CommanderSelector

	// Tabbed unit/roster panel
	activeTab     string             // "units" or "roster"
	unitContent   *widget.Container
	rosterContent *widget.Container

	// State
	selectedGridCell *GridCell    // Currently selected grid cell
	selectedUnitID   ecs.EntityID // Currently selected unit in squad
	swapState        *SwapState   // Click-to-swap state for squad reordering
}

// currentSquadID returns the entity ID of the currently selected squad.
func (sem *SquadEditorMode) currentSquadID() ecs.EntityID {
	return sem.allSquadIDs[sem.currentSquadIndex]
}

// GridCell represents a selected cell in the 3x3 grid
type GridCell struct {
	Row int
	Col int
}

func NewSquadEditorMode(modeManager *framework.UIModeManager) *SquadEditorMode {
	mode := &SquadEditorMode{
		currentSquadIndex: 0,
		allSquadIDs:       make([]ecs.EntityID, 0),
		swapState:         NewSwapState(),
	}
	mode.SetModeName("squad_editor")
	mode.ModeManager = modeManager
	mode.SetSelf(mode) // Required for panel registry building
	return mode
}

func (sem *SquadEditorMode) Initialize(ctx *framework.UIContext) error {
	// Determine return mode based on context:
	// In overworld context, ESC returns to overworld mode
	// In tactical context, ESC is handled by the close button (context switch)
	returnMode := ""
	if _, exists := sem.ModeManager.GetMode("overworld"); exists {
		returnMode = "overworld"
	}

	// Build base UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&sem.BaseMode, framework.ModeConfig{
		ModeName:   "squad_editor",
		ReturnMode: returnMode,
		StatusLabel: true,
		Commands:    true,
		OnRefresh:   sem.refreshAfterCommand,

		Hotkeys: []framework.HotkeySpec{
			{Key: ebiten.KeyP, TargetMode: "unit_purchase"},
		},
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Build panels from registry
	if err := sem.BuildPanels(
		SquadEditorPanelCommanderSelector,
		SquadEditorPanelSquadSelector,
		SquadEditorPanelGridEditor,
		SquadEditorPanelUnitRoster,
	); err != nil {
		return err
	}

	// Initialize widget references from registry
	sem.initializeWidgetReferences()

	// Add action buttons (needs callbacks, so done separately)
	actionButtons := sem.buildActionButtons()
	sem.RootContainer.AddChild(actionButtons)

	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (sem *SquadEditorMode) initializeWidgetReferences() {
	// Commander selector
	sem.commanderSelector = NewCommanderSelector(
		framework.GetPanelWidget[*widget.Text](sem.Panels, SquadEditorPanelCommanderSelector, "commanderLabel"),
		framework.GetPanelWidget[*widget.Button](sem.Panels, SquadEditorPanelCommanderSelector, "commanderPrevBtn"),
		framework.GetPanelWidget[*widget.Button](sem.Panels, SquadEditorPanelCommanderSelector, "commanderNextBtn"),
	)

	// Navigation widgets (merged into squad selector panel)
	sem.prevButton = framework.GetPanelWidget[*widget.Button](sem.Panels, SquadEditorPanelSquadSelector, "prevButton")
	sem.nextButton = framework.GetPanelWidget[*widget.Button](sem.Panels, SquadEditorPanelSquadSelector, "nextButton")
	sem.squadCounterLabel = framework.GetPanelWidget[*widget.Text](sem.Panels, SquadEditorPanelSquadSelector, "counterLabel")

	// List widgets
	sem.squadSelector = framework.GetPanelWidget[*widget.List](sem.Panels, SquadEditorPanelSquadSelector, "squadList")
	sem.unitList = framework.GetPanelWidget[*widget.List](sem.Panels, SquadEditorPanelUnitRoster, "unitList")
	sem.rosterList = framework.GetPanelWidget[*widget.List](sem.Panels, SquadEditorPanelUnitRoster, "rosterList")

	// Tabbed panel containers
	sem.unitContent = framework.GetPanelWidget[*widget.Container](sem.Panels, SquadEditorPanelUnitRoster, "unitContent")
	sem.rosterContent = framework.GetPanelWidget[*widget.Container](sem.Panels, SquadEditorPanelUnitRoster, "rosterContent")
	sem.activeTab = "units"

	// Grid cells
	sem.gridCells = framework.GetPanelWidget[[3][3]*widget.Button](sem.Panels, SquadEditorPanelGridEditor, "gridCells")
}

// buildActionButtons creates bottom action buttons (needs callbacks, so done separately)
func (sem *SquadEditorMode) buildActionButtons() *widget.Container {
	closeText := "Exploration (ESC)"
	if sem.GetReturnMode() != "" {
		closeText = "Overworld (ESC)"
	}

	return builders.CreateBottomActionBar(sem.Layout, []builders.ButtonSpec{
		{Text: "New Squad (N)", OnClick: func() { sem.onNewSquad() }},
		{Text: "Rename Squad", OnClick: func() { sem.onRenameSquad() }},
		{Text: "Buy Units (P)", OnClick: func() {
			if mode, exists := sem.ModeManager.GetMode("unit_purchase"); exists {
				sem.ModeManager.RequestTransition(mode, "Buy Units clicked")
			}
		}},
		{Text: closeText, OnClick: func() {
			if returnMode, exists := sem.ModeManager.GetMode(sem.GetReturnMode()); exists {
				sem.ModeManager.RequestTransition(returnMode, "Close button pressed")
				return
			}
			if sem.Context.ModeCoordinator != nil {
				if err := sem.Context.ModeCoordinator.EnterTactical("exploration"); err != nil {
					fmt.Printf("ERROR: Failed to enter tactical context: %v\n", err)
				}
			}
		}},
	})
}

func (sem *SquadEditorMode) Enter(fromMode framework.UIMode) error {
	// Load commander list and sync current selection
	sem.loadCommanders()

	// Backfill roster with any existing squad units
	// This handles units created before roster tracking was implemented
	sem.backfillRosterWithSquadUnits()

	// Refresh all UI with index reset (entering mode starts at first squad)
	sem.refreshAllUI(true)
	if len(sem.allSquadIDs) == 0 {
		sem.SetStatus("No squads available")
	}

	return nil
}

func (sem *SquadEditorMode) Exit(toMode framework.UIMode) error {
	sem.selectedGridCell = nil
	sem.selectedUnitID = 0
	sem.swapState.Reset()
	return nil
}

func (sem *SquadEditorMode) Update(deltaTime float64) error {
	return nil
}

func (sem *SquadEditorMode) Render(screen *ebiten.Image) {
	// No custom rendering needed
}

func (sem *SquadEditorMode) HandleInput(inputState *framework.InputState) bool {
	// Handle swap FIRST (before other input)
	if sem.handleSwapInput(inputState) {
		return true
	}

	// Handle common input (ESC key)
	if sem.HandleCommonInput(inputState) {
		return true
	}

	// N key creates new squad
	if inputState.KeysJustPressed[ebiten.KeyN] {
		sem.onNewSquad()
		return true
	}

	// Tab key cycles to next commander
	if inputState.KeysJustPressed[ebiten.KeyTab] {
		sem.showNextCommander()
		return true
	}

	return false
}

// === Navigation Functions ===

// cycleSquad advances the squad index by delta (use -1 for previous, +1 for next)
func (sem *SquadEditorMode) cycleSquad(delta int) {
	if len(sem.allSquadIDs) == 0 {
		return
	}
	sem.currentSquadIndex = (sem.currentSquadIndex + delta + len(sem.allSquadIDs)) % len(sem.allSquadIDs)
	sem.refreshCurrentSquad()
	sem.updateNavigationButtons()
}

// updateNavigationButtons enables/disables navigation based on squad count
func (sem *SquadEditorMode) updateNavigationButtons() {
	hasMultipleSquads := len(sem.allSquadIDs) > 1

	if sem.prevButton != nil {
		sem.prevButton.GetWidget().Disabled = !hasMultipleSquads
	}

	if sem.nextButton != nil {
		sem.nextButton.GetWidget().Disabled = !hasMultipleSquads
	}
}

// === Tab Switching Functions ===

// switchTab switches the combined panel to show either "units" or "roster"
func (sem *SquadEditorMode) switchTab(tabName string) {
	if sem.activeTab == tabName {
		return
	}
	sem.activeTab = tabName
	showUnits := tabName == "units"
	if showUnits {
		sem.unitContent.GetWidget().Visibility = widget.Visibility_Show
		sem.rosterContent.GetWidget().Visibility = widget.Visibility_Hide
	} else {
		sem.unitContent.GetWidget().Visibility = widget.Visibility_Hide
		sem.rosterContent.GetWidget().Visibility = widget.Visibility_Show
	}
}

// === Commander Selector Functions ===

// loadCommanders enumerates all commanders and finds the current one
func (sem *SquadEditorMode) loadCommanders() {
	owState := sem.Context.ModeCoordinator.GetOverworldState()
	sem.commanderSelector.Load(
		sem.Context.PlayerData.PlayerEntityID,
		owState.SelectedCommanderID,
		sem.Context.ECSManager,
	)
}

// showPreviousCommander cycles to the previous commander
func (sem *SquadEditorMode) showPreviousCommander() {
	sem.commanderSelector.ShowPrevious(sem.Context.ECSManager, sem.onCommanderSwitched)
}

// showNextCommander cycles to the next commander
func (sem *SquadEditorMode) showNextCommander() {
	sem.commanderSelector.ShowNext(sem.Context.ECSManager, sem.onCommanderSwitched)
}

// onCommanderSwitched updates OverworldState and refreshes all mode data
func (sem *SquadEditorMode) onCommanderSwitched(newCommanderID ecs.EntityID) {
	// Update overworld state so GetSquadRosterOwnerID() returns the new commander
	owState := sem.Context.ModeCoordinator.GetOverworldState()
	owState.SelectedCommanderID = newCommanderID

	// Refresh all mode data for the new commander
	sem.refreshAllUI(true)

	cmdrData := commander.GetCommanderData(newCommanderID, sem.Context.ECSManager)
	if cmdrData != nil {
		sem.SetStatus(fmt.Sprintf("Switched to commander: %s", cmdrData.Name))
	}
}

// onSquadSelected handles squad selection from the list
func (sem *SquadEditorMode) onSquadSelected(squadID ecs.EntityID) {
	// Find index of selected squad
	for i, id := range sem.allSquadIDs {
		if id == squadID {
			sem.currentSquadIndex = i
			sem.refreshCurrentSquad()
			return
		}
	}
}
