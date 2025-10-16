package gui

import (
	"fmt"
	"image/color"

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
}

func NewFormationEditorMode(modeManager *UIModeManager) *FormationEditorMode {
	return &FormationEditorMode{
		modeManager: modeManager,
	}
}

func (fem *FormationEditorMode) Initialize(ctx *UIContext) error {
	fem.context = ctx
	fem.layout = NewLayoutConfig(ctx)

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
	// Center 3x3 grid editor
	fem.gridContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(PanelRes.image),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true, true, true}),
			widget.GridLayoutOpts.Spacing(5, 5),
			widget.GridLayoutOpts.Padding(widget.Insets{
				Left: 10, Right: 10, Top: 10, Bottom: 10,
			}),
		)),
	)

	// Create 3x3 grid buttons
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cellRow, cellCol := row, col // Capture for closure
			cellBtn := widget.NewButton(
				widget.ButtonOpts.Image(buttonImage),
				widget.ButtonOpts.Text(fmt.Sprintf("[%d,%d]", row, col), SmallFace, &widget.ButtonTextColor{
					Idle: color.NRGBA{0xdf, 0xf4, 0xff, 0xff},
				}),
				widget.ButtonOpts.TextPadding(widget.Insets{
					Left: 10, Right: 10, Top: 10, Bottom: 10,
				}),
				widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
					fem.onCellClicked(cellRow, cellCol)
				}),
			)

			fem.gridCells[row][col] = cellBtn
			fem.gridContainer.AddChild(cellBtn)
		}
	}

	// Add LayoutData for center positioning
	fem.gridContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}

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

	fem.unitPalette = widget.NewList(
		widget.ListOpts.Entries(entries),
		widget.ListOpts.EntryLabelFunc(func(e interface{}) string {
			return e.(string)
		}),
		widget.ListOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(ListRes.image),
		),
		widget.ListOpts.SliderOpts(
			widget.SliderOpts.Images(ListRes.track, ListRes.handle),
		),
		widget.ListOpts.EntryColor(ListRes.entry),
		widget.ListOpts.EntryFontFace(ListRes.face),
		widget.ListOpts.ContainerOpts(
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(listWidth, listHeight),
			),
		),
	)

	// Position list widget using LayoutData
	fem.unitPalette.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}

	fem.rootContainer.AddChild(fem.unitPalette)
}

func (fem *FormationEditorMode) buildActionButtons() {
	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10}),
		)),
	)

	// Save button
	saveBtn := CreateButton("Save Formation")
	saveBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			fmt.Println("Formation saved!")
		}),
	)
	buttonContainer.AddChild(saveBtn)

	// Load button
	loadBtn := CreateButton("Load Formation")
	loadBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			fmt.Println("Formation loaded!")
		}),
	)
	buttonContainer.AddChild(loadBtn)

	// Close button
	closeBtn := CreateButton("Close (ESC)")
	closeBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if exploreMode, exists := fem.modeManager.GetMode("exploration"); exists {
				fem.modeManager.RequestTransition(exploreMode, "Close Formation Editor")
			}
		}),
	)
	buttonContainer.AddChild(closeBtn)

	// Add LayoutData for bottom-center positioning
	buttonContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionEnd,
		Padding: widget.Insets{
			Bottom: int(float64(fem.layout.ScreenHeight) * 0.08),
		},
	}

	fem.rootContainer.AddChild(buttonContainer)
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
