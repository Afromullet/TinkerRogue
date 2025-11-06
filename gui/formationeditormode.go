package gui

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// FormationEditorMode provides 3x3 grid editing for squad formations
type FormationEditorMode struct {
	ui          *ebitenui.UI
	context     *UIContext
	layout      *LayoutConfig
	modeManager *UIModeManager

	rootContainer *widget.Container
	gridContainer *widget.Container
	unitPalette   *widget.List
	actionButtons *widget.Container

	gridCells [3][3]*widget.Button // 3x3 grid of cells

	// Panel builders for UI composition
	panelBuilders *PanelBuilders
}

func NewFormationEditorMode(modeManager *UIModeManager) *FormationEditorMode {
	return &FormationEditorMode{
		modeManager: modeManager,
	}
}

func (fem *FormationEditorMode) Initialize(ctx *UIContext) error {
	fem.context = ctx
	fem.layout = NewLayoutConfig(ctx)
	fem.panelBuilders = NewPanelBuilders(fem.layout, fem.modeManager)

	fem.ui = &ebitenui.UI{}
	fem.rootContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	fem.ui.Container = fem.rootContainer

	// Build formation editor UI
	fem.buildGridEditor()
	fem.buildUnitPalette()
	fem.buildActionButtons()

	return nil
}

func (fem *FormationEditorMode) buildGridEditor() {
	// Use panel builder for 3x3 grid editor
	fem.gridContainer, fem.gridCells = fem.panelBuilders.BuildGridEditor(GridEditorConfig{
		OnCellClick: func(row, col int) {
			fem.onCellClicked(row, col)
		},
	})

	fem.rootContainer.AddChild(fem.gridContainer)
}

func (fem *FormationEditorMode) buildUnitPalette() {
	// Left side unit palette
	listWidth := int(float64(fem.layout.ScreenWidth) * 0.2)
	listHeight := int(float64(fem.layout.ScreenHeight) * 0.6)

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

	closeBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Close (ESC)",
		OnClick: func() {
			if exploreMode, exists := fem.modeManager.GetMode("exploration"); exists {
				fem.modeManager.RequestTransition(exploreMode, "Close Formation Editor")
			}
		},
	})

	// Use panel builder for action buttons
	buttons := []*widget.Button{saveBtn, loadBtn, closeBtn}
	fem.actionButtons = fem.panelBuilders.BuildActionButtons(buttons)

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
	// ESC to close
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		if exploreMode, exists := fem.modeManager.GetMode("exploration"); exists {
			fem.modeManager.RequestTransition(exploreMode, "ESC pressed")
			return true
		}
	}

	return false
}

func (fem *FormationEditorMode) GetEbitenUI() *ebitenui.UI {
	return fem.ui
}

func (fem *FormationEditorMode) GetModeName() string {
	return "formation_editor"
}
