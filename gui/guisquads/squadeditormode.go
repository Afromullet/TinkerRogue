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
	squadNav *SquadSelector

	// Interactive widget references (stored here for refresh/access)
	// These are populated from panel registry after BuildPanels()
	squadSelector *widget.List
	gridCells     [3][3]*widget.Button
	unitList      *widget.List
	rosterList    *widget.List

	// Commander selector
	commanderSelector *CommanderSelector

	// Sub-menu controller for right-side panels (units, roster)
	subMenus *framework.SubMenuController

	// Unit and roster panel containers (for widget replacement during refresh)
	unitContent   *widget.Container
	rosterContent *widget.Container

	// Attack pattern toggle
	attackGridCells     [3][3]*widget.Button
	attackGridContainer *widget.Container
	attackLabel         *widget.Text
	showAttackPattern   bool

	// Support pattern toggle
	supportGridCells     [3][3]*widget.Button
	supportGridContainer *widget.Container
	supportLabel         *widget.Text
	showSupportPattern   bool

	// Input action map
	actionMap *framework.ActionMap

	// State
	selectedGridCell *GridCell    // Currently selected grid cell
	selectedUnitID   ecs.EntityID // Currently selected unit in squad
}

// GridCell represents a selected cell in the 3x3 grid
type GridCell struct {
	Row int
	Col int
}

func NewSquadEditorMode(modeManager *framework.UIModeManager) *SquadEditorMode {
	mode := &SquadEditorMode{}
	mode.squadNav = NewSquadSelector(nil, nil, nil)
	mode.SetModeName("squad_editor")
	mode.ModeManager = modeManager
	mode.SetSelf(mode) // Required for panel registry building
	return mode
}

// GetActionMap implements framework.ActionMapProvider.
func (sem *SquadEditorMode) GetActionMap() *framework.ActionMap {
	return sem.actionMap
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
		ModeName:    "squad_editor",
		ReturnMode:  returnMode,
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

	// Initialize action map for semantic keybindings
	sem.actionMap = framework.DefaultSquadEditorBindings()

	// Initialize sub-menu controller before building panels (panels register with it).
	// Pass RootContainer so hidden panels are fully removed from the widget tree,
	// preventing their ScrollContainers from blocking input on overlapping panels.
	sem.subMenus = framework.NewSubMenuController(sem.RootContainer)

	// Build panels from registry
	if err := sem.BuildPanels(
		SquadEditorPanelSquadSelector,
		SquadEditorPanelGridEditor,
		SquadEditorPanelUnitList,
		SquadEditorPanelRoster,
	); err != nil {
		return err
	}

	// Remove initially-hidden sub-menu panels from widget tree.
	// BuildPanels adds all panels to RootContainer; CloseAll removes the
	// sub-menu panels so they don't block input until explicitly shown.
	sem.subMenus.CloseAll()

	// Initialize widget references from registry
	sem.initializeWidgetReferences()

	// Add action button clusters
	sem.RootContainer.AddChild(sem.buildContextActions())
	sem.RootContainer.AddChild(sem.buildNavigationActions())

	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (sem *SquadEditorMode) initializeWidgetReferences() {
	// Commander selector (now in squad list panel header)
	sem.commanderSelector = NewCommanderSelector(
		framework.GetPanelWidget[*widget.Text](sem.Panels, SquadEditorPanelSquadSelector, "commanderLabel"),
		framework.GetPanelWidget[*widget.Button](sem.Panels, SquadEditorPanelSquadSelector, "commanderPrevBtn"),
		framework.GetPanelWidget[*widget.Button](sem.Panels, SquadEditorPanelSquadSelector, "commanderNextBtn"),
	)

	// List widgets
	sem.squadSelector = framework.GetPanelWidget[*widget.List](sem.Panels, SquadEditorPanelSquadSelector, "squadList")
	sem.unitList = framework.GetPanelWidget[*widget.List](sem.Panels, SquadEditorPanelUnitList, "unitList")
	sem.rosterList = framework.GetPanelWidget[*widget.List](sem.Panels, SquadEditorPanelRoster, "rosterList")

	// Panel containers (for widget replacement during refresh)
	sem.unitContent = sem.GetPanelContainer(SquadEditorPanelUnitList)
	sem.rosterContent = sem.GetPanelContainer(SquadEditorPanelRoster)

	// Grid cells
	sem.gridCells = framework.GetPanelWidget[[3][3]*widget.Button](sem.Panels, SquadEditorPanelGridEditor, "gridCells")

	// Attack pattern grid
	sem.attackGridCells = framework.GetPanelWidget[[3][3]*widget.Button](sem.Panels, SquadEditorPanelGridEditor, "attackGridCells")
	sem.attackGridContainer = framework.GetPanelWidget[*widget.Container](sem.Panels, SquadEditorPanelGridEditor, "attackGridContainer")
	sem.attackLabel = framework.GetPanelWidget[*widget.Text](sem.Panels, SquadEditorPanelGridEditor, "attackLabel")

	// Support pattern grid
	sem.supportGridCells = framework.GetPanelWidget[[3][3]*widget.Button](sem.Panels, SquadEditorPanelGridEditor, "supportGridCells")
	sem.supportGridContainer = framework.GetPanelWidget[*widget.Container](sem.Panels, SquadEditorPanelGridEditor, "supportGridContainer")
	sem.supportLabel = framework.GetPanelWidget[*widget.Text](sem.Panels, SquadEditorPanelGridEditor, "supportLabel")
}

// buildContextActions creates bottom-left action buttons for current squad context
func (sem *SquadEditorMode) buildContextActions() *widget.Container {
	return builders.CreateLeftActionBar(sem.Layout, []builders.ButtonSpec{
		{Text: "Units (U)", OnClick: sem.subMenus.Toggle("units")},
		{Text: "Roster (R)", OnClick: sem.subMenus.Toggle("roster")},
		{Text: "Atk Pattern (V)", OnClick: func() { sem.toggleAttackPattern() }},
		{Text: "Support Pattern (B)", OnClick: func() { sem.toggleSupportPattern() }},
	})
}

// buildNavigationActions creates bottom-right action buttons for mode navigation
func (sem *SquadEditorMode) buildNavigationActions() *widget.Container {
	closeText := "Exploration (ESC)"
	if sem.GetReturnMode() != "" {
		closeText = "Overworld (ESC)"
	}

	return builders.CreateRightActionBar(sem.Layout, []builders.ButtonSpec{
		{Text: "Buy Units (P)", OnClick: func() {
			if mode, exists := sem.ModeManager.GetMode("unit_purchase"); exists {
				sem.ModeManager.RequestTransition(mode, "Buy Units clicked")
			}
		}},
		{Text: "Artifacts", OnClick: func() {
			if mode, exists := sem.ModeManager.GetMode("artifact_manager"); exists {
				sem.ModeManager.RequestTransition(mode, "Artifacts clicked")
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
	if !sem.squadNav.HasSquads() {
		sem.SetStatus("No squads available")
	}

	return nil
}

func (sem *SquadEditorMode) Exit(toMode framework.UIMode) error {
	sem.selectedGridCell = nil
	sem.selectedUnitID = 0
	sem.showAttackPattern = false
	sem.attackLabel.GetWidget().Visibility = widget.Visibility_Hide
	sem.attackGridContainer.GetWidget().Visibility = widget.Visibility_Hide
	sem.showSupportPattern = false
	sem.supportLabel.GetWidget().Visibility = widget.Visibility_Hide
	sem.supportGridContainer.GetWidget().Visibility = widget.Visibility_Hide
	sem.subMenus.CloseAll()
	return nil
}

func (sem *SquadEditorMode) Update(deltaTime float64) error {
	return nil
}

func (sem *SquadEditorMode) Render(screen *ebiten.Image) {
	// No custom rendering needed
}

func (sem *SquadEditorMode) HandleInput(inputState *framework.InputState) bool {
	// ESC cascade: close right panel first, then exit mode
	if inputState.ActionActive(framework.ActionCancel) {
		if sem.subMenus.AnyActive() {
			sem.subMenus.CloseAll()
			return true
		}
		// Fall through to HandleCommonInput for mode exit
	}

	// Handle common input (hotkeys + ESC for mode exit)
	if sem.HandleCommonInput(inputState) {
		return true
	}

	// U key toggles units panel
	if inputState.ActionActive(framework.ActionToggleUnits) {
		sem.subMenus.Toggle("units")()
		return true
	}

	// R key toggles roster panel
	if inputState.ActionActive(framework.ActionToggleRoster) {
		sem.subMenus.Toggle("roster")()
		return true
	}

	// N key creates new squad
	if inputState.ActionActive(framework.ActionNewSquad) {
		sem.onNewSquad()
		return true
	}

	// V key toggles attack pattern view
	if inputState.ActionActive(framework.ActionToggleAttackPattern) {
		sem.toggleAttackPattern()
		return true
	}

	// B key toggles support pattern view
	if inputState.ActionActive(framework.ActionToggleSupportPattern) {
		sem.toggleSupportPattern()
		return true
	}

	// Tab key cycles to next commander
	if inputState.ActionActive(framework.ActionCycleCommanderEditor) {
		sem.showNextCommander()
		return true
	}

	return false
}

// === Attack Pattern Toggle ===

func (sem *SquadEditorMode) toggleAttackPattern() {
	sem.showAttackPattern = !sem.showAttackPattern
	vis := widget.Visibility_Hide
	if sem.showAttackPattern {
		vis = widget.Visibility_Show
		sem.refreshAttackPattern()
	}
	sem.attackLabel.GetWidget().Visibility = vis
	sem.attackGridContainer.GetWidget().Visibility = vis
}

func (sem *SquadEditorMode) toggleSupportPattern() {
	sem.showSupportPattern = !sem.showSupportPattern
	vis := widget.Visibility_Hide
	if sem.showSupportPattern {
		vis = widget.Visibility_Show
		sem.refreshSupportPattern()
	}
	sem.supportLabel.GetWidget().Visibility = vis
	sem.supportGridContainer.GetWidget().Visibility = vis
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
	if sem.squadNav.SelectByID(squadID) {
		sem.refreshCurrentSquad()
	}
}
