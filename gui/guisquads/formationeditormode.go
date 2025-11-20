package guisquads

import (
	"game_main/gui"

	"fmt"
	"game_main/gui/core"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// FormationEditorMode provides 3x3 grid editing for squad formations
type FormationEditorMode struct {
	gui.BaseMode // Embed common mode infrastructure

	gridContainer *widget.Container
	unitPalette   *widget.List
	actionButtons *widget.Container

	gridCells [3][3]*widget.Button // 3x3 grid of cells
}

func NewFormationEditorMode(modeManager *core.UIModeManager) *FormationEditorMode {
	mode := &FormationEditorMode{}
	mode.SetModeName("formation_editor")
	mode.ModeManager = modeManager
	return mode
}

func (fem *FormationEditorMode) Initialize(ctx *core.UIContext) error {
	// Initialize common mode infrastructure
	fem.InitializeBase(ctx)

	// Build formation editor UI
	// Build 3x3 grid editor (center)
	fem.gridContainer, fem.gridCells = fem.PanelBuilders.BuildGridEditor(widgets.GridEditorConfig{
		OnCellClick: func(row, col int) {
			fem.onCellClicked(row, col)
		},
	})
	fem.RootContainer.AddChild(fem.gridContainer)

	fem.buildUnitPalette()
	fem.buildActionButtons()

	return nil
}

func (fem *FormationEditorMode) buildUnitPalette() {
	// Left side unit palette
	listWidth := int(float64(fem.Layout.ScreenWidth) * widgets.PanelWidthStandard)
	listHeight := int(float64(fem.Layout.ScreenHeight) * widgets.PanelHeightExtraTall)

	// Unit types
	entries := []interface{}{
		"Tank",
		"DPS",
		"Support",
		"Remove Unit",
	}

	fem.unitPalette = widgets.CreateListWithConfig(widgets.ListConfig{
		Entries:   entries,
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		},
	})

	fem.RootContainer.AddChild(fem.unitPalette)
}

func (fem *FormationEditorMode) buildActionButtons() {
	// Create action buttons
	saveBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Save Formation",
		OnClick: func() {
			fmt.Println("Formation saved!")
		},
	})

	loadBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
		Text: "Load Formation",
		OnClick: func() {
			fmt.Println("Formation loaded!")
		},
	})

	// Build action buttons container using helper
	fem.actionButtons = gui.CreateBottomCenterButtonContainer(fem.PanelBuilders)

	fem.actionButtons.AddChild(saveBtn)
	fem.actionButtons.AddChild(loadBtn)

	// Create close button using helper
	closeBtn := gui.CreateCloseButton(fem.ModeManager, "exploration", "Close (ESC)")
	fem.actionButtons.AddChild(closeBtn)
	fem.RootContainer.AddChild(fem.actionButtons)
}

func (fem *FormationEditorMode) onCellClicked(row, col int) {
	// Get selected unit type from palette
	selectedEntry := fem.unitPalette.SelectedEntry()
	if selectedEntry == nil {
		return
	}

	unitType := selectedEntry.(string)
	fmt.Printf("Placed %s at [%d,%d]\n", unitType, row, col)

	// Update button text to show placed unit
	fem.gridCells[row][col].Text().Label = unitType
}

func (fem *FormationEditorMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Formation Editor Mode")
	return nil
}

func (fem *FormationEditorMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Formation Editor Mode")
	return nil
}

func (fem *FormationEditorMode) Update(deltaTime float64) error {
	return nil
}

func (fem *FormationEditorMode) Render(screen *ebiten.Image) {
	// No custom rendering needed
}

func (fem *FormationEditorMode) HandleInput(inputState *core.InputState) bool {
	// Handle common input (ESC key)
	if fem.HandleCommonInput(inputState) {
		return true
	}

	return false
}
