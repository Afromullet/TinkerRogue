package gui

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/graphics"
	"game_main/squads"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadDeploymentMode allows placing squads on the map before combat
type SquadDeploymentMode struct {
	ui          *ebitenui.UI
	context     *UIContext
	layout      *LayoutConfig
	modeManager *UIModeManager

	rootContainer     *widget.Container
	squadListPanel    *widget.Container
	squadList         *widget.List
	selectedSquadID   ecs.EntityID
	allSquads         []ecs.EntityID
	squadNames        []string
	instructionText   *widget.Text
	confirmButton     *widget.Button
	clearAllButton    *widget.Button

	isPlacingSquad   bool
	pendingMouseX    int
	pendingMouseY    int
	pendingPlacement bool
}

func NewSquadDeploymentMode(modeManager *UIModeManager) *SquadDeploymentMode {
	return &SquadDeploymentMode{
		modeManager: modeManager,
		allSquads:   make([]ecs.EntityID, 0),
		squadNames:  make([]string, 0),
	}
}

func (sdm *SquadDeploymentMode) Initialize(ctx *UIContext) error {
	sdm.context = ctx
	sdm.layout = NewLayoutConfig(ctx)

	sdm.ui = &ebitenui.UI{}
	sdm.rootContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	sdm.ui.Container = sdm.rootContainer

	// Build UI components
	sdm.buildSquadListPanel()
	sdm.buildInstructionText()
	sdm.buildActionButtons()

	return nil
}

func (sdm *SquadDeploymentMode) buildSquadListPanel() {
	// Left side squad list
	width := int(float64(sdm.layout.ScreenWidth) * 0.2)
	height := int(float64(sdm.layout.ScreenHeight) * 0.8)

	sdm.squadListPanel = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(PanelRes.image),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.Insets{
				Left: 10, Right: 10, Top: 10, Bottom: 10,
			}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(width, height),
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				Padding: widget.Insets{
					Left: int(float64(sdm.layout.ScreenWidth) * 0.01),
				},
			}),
		),
	)

	// Create list widget with squad names
	sdm.squadList = widget.NewList(
		widget.ListOpts.ContainerOpts(
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(width-20, height-20),
			),
		),
		widget.ListOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(ListRes.image),
		),
		widget.ListOpts.SliderOpts(
			widget.SliderOpts.Images(ListRes.track, ListRes.handle),
			widget.SliderOpts.MinHandleSize(ListRes.handleSize),
			widget.SliderOpts.TrackPadding(ListRes.trackPadding),
		),
		widget.ListOpts.EntryColor(ListRes.entry),
		widget.ListOpts.EntryFontFace(ListRes.face),
		widget.ListOpts.EntryLabelFunc(func(e interface{}) string {
			if str, ok := e.(string); ok {
				return str
			}
			return ""
		}),
	)

	// Add selection handler for squad selection
	sdm.squadList.EntrySelectedEvent.AddHandler(func(args interface{}) {
		fmt.Printf("DEBUG: Squad selection event fired!\n")
		a := args.(*widget.ListEntrySelectedEventArgs)
		selectedEntry := a.Entry
		fmt.Printf("DEBUG: Selected entry: %v\n", selectedEntry)

		if squadName, ok := selectedEntry.(string); ok {
			fmt.Printf("DEBUG: Squad name selected: %s\n", squadName)
			// Find the squad ID matching this name
			for i, name := range sdm.squadNames {
				if name == squadName && i < len(sdm.allSquads) {
					sdm.selectedSquadID = sdm.allSquads[i]
					sdm.isPlacingSquad = true
					fmt.Printf("DEBUG: Set selectedSquadID=%d, isPlacingSquad=true\n", sdm.selectedSquadID)
					sdm.updateInstructionText()
					break
				}
			}
		} else {
			fmt.Printf("DEBUG: Failed to cast entry to string\n")
		}
	})

	sdm.squadListPanel.AddChild(sdm.squadList)
	sdm.rootContainer.AddChild(sdm.squadListPanel)
}

func (sdm *SquadDeploymentMode) buildInstructionText() {
	sdm.instructionText = widget.NewText(
		widget.TextOpts.Text("Select a squad from the list, then click on the map to place it", SmallFace, color.White),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				Padding: widget.Insets{
					Top: int(float64(sdm.layout.ScreenHeight) * 0.02),
				},
			}),
		),
	)

	sdm.rootContainer.AddChild(sdm.instructionText)
}

func (sdm *SquadDeploymentMode) buildActionButtons() {
	buttonContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				Padding: widget.Insets{
					Bottom: int(float64(sdm.layout.ScreenHeight) * 0.08),
				},
			}),
		),
	)

	// Clear All button
	sdm.clearAllButton = CreateButton("Clear All")
	sdm.clearAllButton.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			sdm.clearAllSquadPositions()
		}),
	)
	buttonContainer.AddChild(sdm.clearAllButton)

	// Confirm/Start Combat button
	sdm.confirmButton = CreateButton("Start Combat")
	sdm.confirmButton.Configure(
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if combatMode, exists := sdm.modeManager.GetMode("combat"); exists {
				sdm.modeManager.RequestTransition(combatMode, "Squads deployed, starting combat")
			}
		}),
	)
	buttonContainer.AddChild(sdm.confirmButton)

	sdm.rootContainer.AddChild(buttonContainer)
}

func (sdm *SquadDeploymentMode) updateInstructionText() {
	if sdm.selectedSquadID == 0 {
		sdm.instructionText.Label = "Select a squad from the list, then click on the map to place it"
		return
	}

	squadName := sdm.getSquadName(sdm.selectedSquadID)
	sdm.instructionText.Label = fmt.Sprintf("Placing %s - Click on the map to position it", squadName)
}

func (sdm *SquadDeploymentMode) getSquadName(squadID ecs.EntityID) string {
	for _, result := range sdm.context.ECSManager.World.Query(sdm.context.ECSManager.Tags["squad"]) {
		squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
		if squadData.SquadID == squadID {
			return squadData.Name
		}
	}
	return "Unknown Squad"
}

func (sdm *SquadDeploymentMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Squad Deployment Mode")

	// Collect all squads
	sdm.allSquads = sdm.allSquads[:0]
	sdm.squadNames = sdm.squadNames[:0]

	// Query all entities with SquadComponent
	for _, result := range sdm.context.ECSManager.World.Query(sdm.context.ECSManager.Tags["squad"]) {
		squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
		sdm.allSquads = append(sdm.allSquads, squadData.SquadID)
		sdm.squadNames = append(sdm.squadNames, squadData.Name)
	}

	// Convert squad names to interface{} for list widget
	entries := make([]interface{}, len(sdm.squadNames))
	for i, name := range sdm.squadNames {
		entries[i] = name
	}

	// Update list with squad names
	sdm.squadList.SetEntries(entries)
	sdm.selectedSquadID = 0
	sdm.isPlacingSquad = false
	sdm.updateInstructionText()

	return nil
}

func (sdm *SquadDeploymentMode) Exit(toMode UIMode) error {
	fmt.Println("Exiting Squad Deployment Mode")
	return nil
}

func (sdm *SquadDeploymentMode) Update(deltaTime float64) error {
	// Process pending placement (after UI has been updated)
	if sdm.pendingPlacement && sdm.isPlacingSquad && sdm.selectedSquadID != 0 {
		fmt.Printf("DEBUG: Processing pending placement in Update()\n")

		// Get player position (for viewport centering)
		playerPos := *sdm.context.PlayerData.Pos

		// Create viewport to convert screen to logical coordinates
		manager := coords.NewCoordinateManager(graphics.ScreenInfo)
		viewport := coords.NewViewport(manager, playerPos)

		// Convert mouse position to logical position
		clickedPos := viewport.ScreenToLogical(sdm.pendingMouseX, sdm.pendingMouseY)
		fmt.Printf("DEBUG: Converted click to logical position: (%d, %d)\n", clickedPos.X, clickedPos.Y)

		// Place the squad at the clicked position
		sdm.placeSquadAt(sdm.selectedSquadID, clickedPos)

		sdm.pendingPlacement = false
	}

	return nil
}

func (sdm *SquadDeploymentMode) Render(screen *ebiten.Image) {
	// Could add visualization of valid placement zones, squad formations, etc.
}

func (sdm *SquadDeploymentMode) HandleInput(inputState *InputState) bool {
	// Capture mouse clicks for processing after UI update
	if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
		fmt.Printf("DEBUG: Mouse click captured at (%d, %d), placing=%v, squadID=%d\n",
			inputState.MouseX, inputState.MouseY, sdm.isPlacingSquad, sdm.selectedSquadID)

		// Check if click is inside the squad list panel (UI area) - don't process as map click
		listBounds := sdm.squadListPanel.GetWidget().Rect
		isInsideList := inputState.MouseX >= listBounds.Min.X && inputState.MouseX <= listBounds.Max.X &&
			inputState.MouseY >= listBounds.Min.Y && inputState.MouseY <= listBounds.Max.Y

		if isInsideList {
			fmt.Printf("DEBUG: Click is inside UI panel, not processing as map placement\n")
			return false
		}

		// Store the click for processing in Update() after ebitenui has processed widget events
		sdm.pendingMouseX = inputState.MouseX
		sdm.pendingMouseY = inputState.MouseY
		sdm.pendingPlacement = true

		fmt.Printf("DEBUG: Click stored as pending for map placement\n")
	}

	return false
}

func (sdm *SquadDeploymentMode) placeSquadAt(squadID ecs.EntityID, pos coords.LogicalPosition) {
	fmt.Printf("DEBUG: placeSquadAt called with squadID=%d, pos=(%d,%d)\n", squadID, pos.X, pos.Y)

	// Find the squad entity and update its position
	found := false
	for _, result := range sdm.context.ECSManager.World.Query(sdm.context.ECSManager.Tags["squad"]) {
		squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
		fmt.Printf("DEBUG: Checking squad %d against target %d\n", squadData.SquadID, squadID)

		if squadData.SquadID == squadID {
			found = true
			fmt.Printf("DEBUG: Found matching squad entity\n")

			// Check if entity has PositionComponent
			if !result.Entity.HasComponent(common.PositionComponent) {
				fmt.Printf("DEBUG: ERROR - Squad entity has no PositionComponent!\n")
				return
			}

			// Update position component
			posPtr := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
			if posPtr != nil {
				oldX, oldY := posPtr.X, posPtr.Y
				posPtr.X = pos.X
				posPtr.Y = pos.Y

				squadName := sdm.getSquadName(squadID)
				fmt.Printf("✓ Placed %s at (%d, %d) [was at (%d, %d)]\n", squadName, pos.X, pos.Y, oldX, oldY)

				// Reset placement mode
				sdm.isPlacingSquad = false
				sdm.selectedSquadID = 0
				sdm.updateInstructionText()
			} else {
				fmt.Printf("DEBUG: ERROR - posPtr is nil!\n")
			}
			return
		}
	}

	if !found {
		fmt.Printf("DEBUG: ERROR - Could not find squad with ID %d\n", squadID)
	}
}

func (sdm *SquadDeploymentMode) clearAllSquadPositions() {
	// Reset all squads to position (0, 0)
	for _, result := range sdm.context.ECSManager.World.Query(sdm.context.ECSManager.Tags["squad"]) {
		if result.Entity.HasComponent(common.PositionComponent) {
			posPtr := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
			if posPtr != nil {
				posPtr.X = 0
				posPtr.Y = 0
			}
		}
	}

	sdm.selectedSquadID = 0
	sdm.isPlacingSquad = false
	sdm.updateInstructionText()

	fmt.Println("All squads cleared")
}

func (sdm *SquadDeploymentMode) GetEbitenUI() *ebitenui.UI {
	return sdm.ui
}

func (sdm *SquadDeploymentMode) GetModeName() string {
	return "squad_deployment"
}
