package guisquads

import (
	"fmt"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"

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

	// State
	selectedGridCell *GridCell    // Currently selected grid cell
	selectedUnitID   ecs.EntityID // Currently selected unit in squad
	swapState        *SwapState   // Click-to-swap state for squad reordering
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
	mode.SetReturnMode("squad_management")
	mode.ModeManager = modeManager
	mode.SetSelf(mode) // Required for panel registry building
	return mode
}

func (sem *SquadEditorMode) Initialize(ctx *framework.UIContext) error {
	// Build base UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&sem.BaseMode, framework.ModeConfig{
		ModeName:    "squad_editor",
		ReturnMode:  "squad_management",
		StatusLabel: true,
		Commands:    true,
		OnRefresh:   sem.refreshAfterUndoRedo,
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Build panels from registry
	if err := sem.buildPanelsFromRegistry(); err != nil {
		return err
	}

	// Initialize widget references from registry
	sem.initializeWidgetReferences()

	// Add action buttons (needs callbacks, so done separately)
	actionButtons := sem.buildActionButtons()
	sem.RootContainer.AddChild(actionButtons)

	return nil
}

// buildPanelsFromRegistry builds all squad editor panels from the global registry
func (sem *SquadEditorMode) buildPanelsFromRegistry() error {
	return sem.BuildPanels(
		SquadEditorPanelNavigation,
		SquadEditorPanelSquadSelector,
		SquadEditorPanelGridEditor,
		SquadEditorPanelUnitList,
		SquadEditorPanelRosterList,
	)
}

// initializeWidgetReferences populates mode fields from panel registry
func (sem *SquadEditorMode) initializeWidgetReferences() {
	// Navigation widgets
	sem.prevButton = framework.GetPanelWidget[*widget.Button](sem.Panels, SquadEditorPanelNavigation, "prevButton")
	sem.nextButton = framework.GetPanelWidget[*widget.Button](sem.Panels, SquadEditorPanelNavigation, "nextButton")
	sem.squadCounterLabel = framework.GetPanelWidget[*widget.Text](sem.Panels, SquadEditorPanelNavigation, "counterLabel")

	// List widgets
	sem.squadSelector = framework.GetPanelWidget[*widget.List](sem.Panels, SquadEditorPanelSquadSelector, "squadList")
	sem.unitList = framework.GetPanelWidget[*widget.List](sem.Panels, SquadEditorPanelUnitList, "unitList")
	sem.rosterList = framework.GetPanelWidget[*widget.List](sem.Panels, SquadEditorPanelRosterList, "rosterList")

	// Grid cells
	sem.gridCells = framework.GetPanelWidget[[3][3]*widget.Button](sem.Panels, SquadEditorPanelGridEditor, "gridCells")
}

// buildActionButtons creates bottom action buttons (needs callbacks, so done separately)
func (sem *SquadEditorMode) buildActionButtons() *widget.Container {
	// Calculate responsive spacing
	spacing := int(float64(sem.Layout.ScreenWidth) * specs.PaddingTight)

	// Create button group using builders.CreateButtonGroup with LayoutData
	bottomPad := int(float64(sem.Layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Rename Squad", OnClick: func() { sem.onRenameSquad() }},
			{Text: "Undo (Ctrl+Z)", OnClick: func() { sem.CommandHistory.Undo() }},
			{Text: "Redo (Ctrl+Y)", OnClick: func() { sem.CommandHistory.Redo() }},
			{Text: "Close (ESC)", OnClick: func() {
				if mode, exists := sem.ModeManager.GetMode("squad_management"); exists {
					sem.ModeManager.RequestTransition(mode, "Close button pressed")
				}
			}},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(sem.Layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	return buttonContainer
}

func (sem *SquadEditorMode) Enter(fromMode framework.UIMode) error {
	fmt.Println("Entering Squad Editor Mode")

	// Backfill roster with any existing squad units
	// This handles units created before roster tracking was implemented
	sem.backfillRosterWithSquadUnits()

	// Sync from roster (source of truth)
	sem.syncSquadOrderFromRoster()

	// Refresh squad selector with current squads
	sem.refreshSquadSelector()

	// Reset to first squad if we have any
	if len(sem.allSquadIDs) > 0 {
		sem.currentSquadIndex = 0
		sem.refreshCurrentSquad()
	} else {
		sem.SetStatus("No squads available")
	}

	sem.updateNavigationButtons()
	sem.refreshRosterList()

	return nil
}

func (sem *SquadEditorMode) Exit(toMode framework.UIMode) error {
	fmt.Println("Exiting Squad Editor Mode")
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

	// Handle undo/redo input (Ctrl+Z, Ctrl+Y)
	if sem.CommandHistory.HandleInput(inputState) {
		return true
	}

	return false
}

// === Navigation Functions ===

// showPreviousSquad cycles to previous squad
func (sem *SquadEditorMode) showPreviousSquad() {
	if len(sem.allSquadIDs) == 0 {
		return
	}

	sem.currentSquadIndex--
	if sem.currentSquadIndex < 0 {
		sem.currentSquadIndex = len(sem.allSquadIDs) - 1
	}

	sem.refreshCurrentSquad()
	sem.updateNavigationButtons()
}

// showNextSquad cycles to next squad
func (sem *SquadEditorMode) showNextSquad() {
	if len(sem.allSquadIDs) == 0 {
		return
	}

	sem.currentSquadIndex++
	if sem.currentSquadIndex >= len(sem.allSquadIDs) {
		sem.currentSquadIndex = 0
	}

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
