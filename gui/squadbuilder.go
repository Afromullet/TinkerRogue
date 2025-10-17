package gui

import (
	"fmt"
	"game_main/common"
	"game_main/squads"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadBuilderMode provides an interface for creating new squads
type SquadBuilderMode struct {
	ui          *ebitenui.UI
	context     *UIContext
	layout      *LayoutConfig
	modeManager *UIModeManager

	rootContainer    *widget.Container
	gridContainer    *widget.Container
	unitPalette      *widget.List
	capacityDisplay  *widget.TextArea
	squadNameInput   *widget.TextInput
	actionButtons    *widget.Container
	unitDetailsArea  *widget.TextArea

	gridCells        [3][3]*GridCellButton
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

func NewSquadBuilderMode(modeManager *UIModeManager) *SquadBuilderMode {
	return &SquadBuilderMode{
		modeManager:     modeManager,
		selectedUnitIdx: -1,
	}
}

func (sbm *SquadBuilderMode) Initialize(ctx *UIContext) error {
	sbm.context = ctx
	sbm.layout = NewLayoutConfig(ctx)

	sbm.ui = &ebitenui.UI{}
	sbm.rootContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	sbm.ui.Container = sbm.rootContainer

	// Build UI components
	sbm.buildGridEditor()
	sbm.buildUnitPalette()
	sbm.buildCapacityDisplay()
	sbm.buildSquadNameInput()
	sbm.buildUnitDetailsArea()
	sbm.buildActionButtons()

	return nil
}

func (sbm *SquadBuilderMode) buildGridEditor() {
	// Center 3x3 grid for squad formation
	sbm.gridContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(PanelRes.image),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true, true, true}),
			widget.GridLayoutOpts.Spacing(5, 5),
			widget.GridLayoutOpts.Padding(widget.Insets{
				Left: 15, Right: 15, Top: 15, Bottom: 15,
			}),
		)),
	)

	// Create 3x3 grid buttons
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cellRow, cellCol := row, col // Capture for closure

			cellBtn := widget.NewButton(
				widget.ButtonOpts.Image(buttonImage),
				widget.ButtonOpts.Text(fmt.Sprintf("Empty\n[%d,%d]", row, col), SmallFace, &widget.ButtonTextColor{
					Idle: color.NRGBA{0xdf, 0xf4, 0xff, 0xff},
				}),
				widget.ButtonOpts.TextPadding(widget.Insets{
					Left: 8, Right: 8, Top: 8, Bottom: 8,
				}),
				widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
					sbm.onCellClicked(cellRow, cellCol)
				}),
			)

			sbm.gridCells[row][col] = &GridCellButton{
				button:    cellBtn,
				row:       row,
				col:       col,
				unitID:    0,
				unitIndex: -1,
			}

			sbm.gridContainer.AddChild(cellBtn)
		}
	}

	// Center positioning
	sbm.gridContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}

	sbm.rootContainer.AddChild(sbm.gridContainer)
}

func (sbm *SquadBuilderMode) buildUnitPalette() {
	// Left side unit palette showing available unit templates
	listWidth := int(float64(sbm.layout.ScreenWidth) * 0.2)
	listHeight := int(float64(sbm.layout.ScreenHeight) * 0.5)

	// Build entries from squads.Units
	entries := make([]interface{}, len(squads.Units)+1)
	entries[0] = "[Remove Unit]" // Special entry for removing units
	for i, unit := range squads.Units {
		entries[i+1] = fmt.Sprintf("%s (%s)", unit.Name, unit.Role.String())
	}

	sbm.unitPalette = widget.NewList(
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

	// Add selection handler for unit palette
	sbm.unitPalette.EntrySelectedEvent.AddHandler(func(args interface{}) {
		a := args.(*widget.ListEntrySelectedEventArgs)
		// Cast entry back to determine which entry was selected
		if entryStr, ok := a.Entry.(string); ok {
			// Find the index based on the string
			for i, unit := range squads.Units {
				expectedStr := fmt.Sprintf("%s (%s)", unit.Name, unit.Role.String())
				if entryStr == expectedStr {
					sbm.selectedUnitIdx = i
					sbm.updateUnitDetails()
					return
				}
			}
			// If we get here, it's the "Remove Unit" entry
			sbm.selectedUnitIdx = -1
			sbm.updateUnitDetails()
		}
	})

	// Position list widget
	sbm.unitPalette.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		Padding: widget.Insets{
			Left: 20,
			Top:  20,
		},
	}

	sbm.rootContainer.AddChild(sbm.unitPalette)
}

func (sbm *SquadBuilderMode) buildCapacityDisplay() {
	// Right side capacity display
	displayWidth := int(float64(sbm.layout.ScreenWidth) * 0.18)
	displayHeight := int(float64(sbm.layout.ScreenHeight) * 0.15)

	config := TextAreaConfig{
		MinWidth:  displayWidth,
		MinHeight: displayHeight,
		FontColor: color.White,
	}

	sbm.capacityDisplay = CreateTextAreaWithConfig(config)
	sbm.capacityDisplay.SetText("Capacity: 0.0 / 6.0\n(No leader)")

	sbm.capacityDisplay.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding: widget.Insets{
			Right: 20,
			Top:   80,
		},
	}

	sbm.rootContainer.AddChild(sbm.capacityDisplay)
}

func (sbm *SquadBuilderMode) buildUnitDetailsArea() {
	// Right side unit details display
	displayWidth := int(float64(sbm.layout.ScreenWidth) * 0.18)
	displayHeight := int(float64(sbm.layout.ScreenHeight) * 0.3)

	config := TextAreaConfig{
		MinWidth:  displayWidth,
		MinHeight: displayHeight,
		FontColor: color.White,
	}

	sbm.unitDetailsArea = CreateTextAreaWithConfig(config)
	sbm.unitDetailsArea.SetText("Select a unit to view details")

	sbm.unitDetailsArea.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		Padding: widget.Insets{
			Right: 20,
		},
	}

	sbm.rootContainer.AddChild(sbm.unitDetailsArea)
}

func (sbm *SquadBuilderMode) buildSquadNameInput() {
	// Top center squad name input
	inputContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
		)),
	)

	// Label
	nameLabel := widget.NewText(
		widget.TextOpts.Text("Squad Name:", LargeFace, color.White),
	)
	inputContainer.AddChild(nameLabel)

	// Text input - use simple color background instead of complex image
	sbm.squadNameInput = widget.NewTextInput(
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(300, 50),
		),
		widget.TextInputOpts.Image(&widget.TextInputImage{
			Idle:     defaultWidgetColor,
			Disabled: defaultWidgetColor,
		}),
		widget.TextInputOpts.Face(SmallFace),
		widget.TextInputOpts.Color(&widget.TextInputColor{
			Idle:     color.White,
			Disabled: hexToColor(textDisabledColor),
			Caret:    color.White,
		}),
		widget.TextInputOpts.Padding(widget.Insets{
			Left: 15, Right: 15, Top: 10, Bottom: 10,
		}),
		widget.TextInputOpts.Placeholder("Enter squad name..."),
		widget.TextInputOpts.CaretOpts(
			widget.CaretOpts.Size(SmallFace, 2),
		),
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			sbm.currentSquadName = args.InputText
		}),
	)
	inputContainer.AddChild(sbm.squadNameInput)

	// Position at top center
	inputContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding: widget.Insets{
			Top: 20,
		},
	}

	sbm.rootContainer.AddChild(inputContainer)
}

func (sbm *SquadBuilderMode) buildActionButtons() {
	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(15),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10}),
		)),
	)

	// Create Squad button
	createBtn := CreateButton("Create Squad")
	createBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			sbm.onCreateSquad()
		}),
	)
	buttonContainer.AddChild(createBtn)

	// Clear Grid button
	clearBtn := CreateButton("Clear Grid")
	clearBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			sbm.onClearGrid()
		}),
	)
	buttonContainer.AddChild(clearBtn)

	// Close button
	closeBtn := CreateButton("Close (ESC)")
	closeBtn.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if exploreMode, exists := sbm.modeManager.GetMode("exploration"); exists {
				sbm.modeManager.RequestTransition(exploreMode, "Close Squad Builder")
			}
		}),
	)
	buttonContainer.AddChild(closeBtn)

	// Position at bottom center
	buttonContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionEnd,
		Padding: widget.Insets{
			Bottom: int(float64(sbm.layout.ScreenHeight) * 0.08),
		},
	}

	sbm.rootContainer.AddChild(buttonContainer)
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

	unit := squads.Units[unitIndex]

	// Attempt to add unit to squad (this checks capacity constraints)
	err := squads.AddUnitToSquad(sbm.currentSquadID, sbm.context.ECSManager, unit, row, col)
	if err != nil {
		fmt.Printf("Failed to place unit: %v\n", err)
		return
	}

	// Find the newly created unit entity
	unitIDs := squads.GetUnitIDsAtGridPosition(sbm.currentSquadID, row, col, sbm.context.ECSManager)
	if len(unitIDs) == 0 {
		fmt.Println("Error: Unit was not placed correctly")
		return
	}

	// Update grid cell display
	cell := sbm.gridCells[row][col]
	cell.unitID = unitIDs[0]
	cell.unitIndex = unitIndex

	// Update button text to show unit name and role
	cellText := fmt.Sprintf("%s\n%s\n[%d,%d]", unit.Name, unit.Role.String(), row, col)
	cell.button.Text().Label = cellText

	// Update capacity display
	sbm.updateCapacityDisplay()

	fmt.Printf("Placed %s at [%d,%d]\n", unit.Name, row, col)
}

func (sbm *SquadBuilderMode) removeUnitFromCell(row, col int) {
	cell := sbm.gridCells[row][col]

	if cell.unitID == 0 {
		return
	}

	// Remove unit from squad
	err := squads.RemoveUnitFromSquad(cell.unitID, sbm.context.ECSManager)
	if err != nil {
		fmt.Printf("Failed to remove unit: %v\n", err)
		return
	}

	// Clear cell state
	cell.unitID = 0
	cell.unitIndex = -1
	cell.button.Text().Label = fmt.Sprintf("Empty\n[%d,%d]", row, col)

	// Update capacity display
	sbm.updateCapacityDisplay()

	fmt.Printf("Removed unit from [%d,%d]\n", row, col)
}

func (sbm *SquadBuilderMode) createTemporarySquad() {
	squadName := sbm.currentSquadName
	if squadName == "" {
		squadName = "New Squad"
	}

	squads.CreateEmptySquad(sbm.context.ECSManager, squadName)

	// Find the newly created squad
	// Squads are created with SquadComponent, so we can query for it
	allSquads := sbm.findAllSquads()
	if len(allSquads) > 0 {
		sbm.currentSquadID = allSquads[len(allSquads)-1] // Get most recent
		fmt.Printf("Created temporary squad: %s (ID: %d)\n", squadName, sbm.currentSquadID)
	}
}

func (sbm *SquadBuilderMode) findAllSquads() []ecs.EntityID {
	allSquads := make([]ecs.EntityID, 0)
	entityIDs := sbm.context.ECSManager.GetAllEntities()

	for _, entityID := range entityIDs {
		if sbm.context.ECSManager.HasComponent(entityID, squads.SquadComponent) {
			allSquads = append(allSquads, entityID)
		}
	}

	return allSquads
}

func (sbm *SquadBuilderMode) updateCapacityDisplay() {
	if sbm.currentSquadID == 0 {
		sbm.capacityDisplay.SetText("Capacity: 0.0 / 6.0\n(No squad created)")
		return
	}

	used := squads.GetSquadUsedCapacity(sbm.currentSquadID, sbm.context.ECSManager)
	total := squads.GetSquadTotalCapacity(sbm.currentSquadID, sbm.context.ECSManager)
	remaining := float64(total) - used

	capacityText := fmt.Sprintf("Capacity: %.1f / %d\nRemaining: %.1f", used, total, remaining)

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
	unitIDs := squads.GetUnitIDsInSquad(sbm.currentSquadID, sbm.context.ECSManager)
	if len(unitIDs) == 0 {
		fmt.Println("Cannot create empty squad")
		return
	}

	// Update squad name if it was changed
	squadEntity := squads.GetSquadEntity(sbm.currentSquadID, sbm.context.ECSManager)
	if squadEntity != nil {
		squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
		squadData.Name = sbm.currentSquadName
	}

	fmt.Printf("Squad created: %s with %d units\n", sbm.currentSquadName, len(unitIDs))

	// Visualize the squad
	visualization := squads.VisualizeSquad(sbm.currentSquadID, sbm.context.ECSManager)
	fmt.Println(visualization)

	// Clear the builder for next squad
	sbm.onClearGrid()
}

func (sbm *SquadBuilderMode) onClearGrid() {
	// If there's a current squad, we could either delete it or just reset the UI
	// For now, just reset the UI and create a new squad on next placement
	sbm.currentSquadID = 0
	sbm.currentSquadName = ""
	sbm.squadNameInput.SetText("")
	sbm.selectedUnitIdx = -1

	// Clear all grid cells
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cell := sbm.gridCells[row][col]
			cell.unitID = 0
			cell.unitIndex = -1
			cell.button.Text().Label = fmt.Sprintf("Empty\n[%d,%d]", row, col)
		}
	}

	sbm.updateCapacityDisplay()
	sbm.updateUnitDetails()

	fmt.Println("Grid cleared")
}

func (sbm *SquadBuilderMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Squad Builder Mode")

	// Reset state on entry
	sbm.onClearGrid()

	return nil
}

func (sbm *SquadBuilderMode) Exit(toMode UIMode) error {
	fmt.Println("Exiting Squad Builder Mode")
	return nil
}

func (sbm *SquadBuilderMode) Update(deltaTime float64) error {
	return nil
}

func (sbm *SquadBuilderMode) Render(screen *ebiten.Image) {
	// No custom rendering needed - ebitenui handles everything
}

func (sbm *SquadBuilderMode) HandleInput(inputState *InputState) bool {
	// ESC to close
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		if exploreMode, exists := sbm.modeManager.GetMode("exploration"); exists {
			sbm.modeManager.RequestTransition(exploreMode, "ESC pressed")
			return true
		}
	}

	return false
}

func (sbm *SquadBuilderMode) GetEbitenUI() *ebitenui.UI {
	return sbm.ui
}

func (sbm *SquadBuilderMode) GetModeName() string {
	return "squad_builder"
}
