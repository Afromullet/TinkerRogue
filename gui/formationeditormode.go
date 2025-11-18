package gui

import (
	"fmt"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// FormationEditorMode provides 3x3 grid editing for squad formations
type FormationEditorMode struct {
	BaseMode // Embed common mode infrastructure

	gridContainer *widget.Container
	unitPalette   *widget.List
	actionButtons *widget.Container

	gridCells [3][3]*widget.Button // 3x3 grid of cells
}

func NewFormationEditorMode(modeManager *UIModeManager) *FormationEditorMode {
	return &FormationEditorMode{
		BaseMode: BaseMode{
			modeManager: modeManager,
			modeName:    "formation_editor",
			returnMode:  "exploration",
		},
	}
}

func (fem *FormationEditorMode) Initialize(ctx *UIContext) error {
	// Initialize common mode infrastructure
	fem.InitializeBase(ctx)

	// Build formation editor UI
	// Build 3x3 grid editor (center)
	fem.gridContainer, fem.gridCells = fem.panelBuilders.BuildGridEditor(GridEditorConfig{
		OnCellClick: func(row, col int) {
			fem.onCellClicked(row, col)
		},
	})
	fem.rootContainer.AddChild(fem.gridContainer)

	fem.buildUnitPalette()
	fem.buildActionButtons()

	return nil
}

func (fem *FormationEditorMode) buildUnitPalette() {
	// Left side unit palette
	listWidth := int(float64(fem.layout.ScreenWidth) * PanelWidthStandard)
	listHeight := int(float64(fem.layout.ScreenHeight) * PanelHeightExtraTall)

	// Unit types
	entries := []interface{}{
		"Tank",
		"DPS",
		"Support",
		"Remove Unit",
	}

	fem.unitPalette = CreateListWithConfig(ListConfig{
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

	fem.rootContainer.AddChild(fem.unitPalette)
}

func (fem *FormationEditorMode) buildActionButtons() {
	// Create action buttons
	saveBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Save Formation",
		OnClick: func() {
			fmt.Println("Formation saved!")
		},
	})

	loadBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Load Formation",
		OnClick: func() {
			fmt.Println("Formation loaded!")
		},
	})

	// Build action buttons container using helper
	fem.actionButtons = CreateBottomCenterButtonContainer(fem.panelBuilders)

	fem.actionButtons.AddChild(saveBtn)
	fem.actionButtons.AddChild(loadBtn)

	// Create close button using helper
	closeBtn := CreateCloseButton(fem.modeManager, "exploration", "Close (ESC)")
	fem.actionButtons.AddChild(closeBtn)
	fem.rootContainer.AddChild(fem.actionButtons)
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

func (fem *FormationEditorMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Formation Editor Mode")
	return nil
}

func (fem *FormationEditorMode) Exit(toMode UIMode) error {
	fmt.Println("Exiting Formation Editor Mode")
	return nil
}

func (fem *FormationEditorMode) Update(deltaTime float64) error {
	return nil
}

func (fem *FormationEditorMode) Render(screen *ebiten.Image) {
	// No custom rendering needed
}

func (fem *FormationEditorMode) HandleInput(inputState *InputState) bool {
	// Handle common input (ESC key)
	if fem.HandleCommonInput(inputState) {
		return true
	}

	return false
}

