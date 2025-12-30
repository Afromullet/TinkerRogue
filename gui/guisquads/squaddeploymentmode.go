package guisquads

import (
	"fmt"
	"image/color"

	"game_main/common"
	"game_main/world/coords"
	"game_main/gui"
	"game_main/gui/builders"
	"game_main/gui/core"
	"game_main/gui/guimodes"
	"game_main/gui/specs"
	"game_main/gui/widgets"
	"game_main/tactical/squads"
	"game_main/tactical/squadservices"
	"game_main/visual/graphics"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadDeploymentMode allows placing squads on the map before combat
type SquadDeploymentMode struct {
	gui.BaseMode // Embed common mode infrastructure

	deploymentService *squadservices.SquadDeploymentService
	squadList         *widgets.CachedListWrapper
	detailPanel       *widget.Container
	detailTextArea    *widgets.CachedTextAreaWrapper // Cached for performance
	selectedSquadID   ecs.EntityID
	instructionText   *widget.Text

	isPlacingSquad   bool
	pendingMouseX    int
	pendingMouseY    int
	pendingPlacement bool

	// Rendering systems
	highlightRenderer *guimodes.SquadHighlightRenderer
}

func NewSquadDeploymentMode(modeManager *core.UIModeManager) *SquadDeploymentMode {
	mode := &SquadDeploymentMode{}
	mode.SetModeName("squad_deployment")
	mode.SetReturnMode("exploration") // ESC returns to exploration
	mode.ModeManager = modeManager
	return mode
}

func (sdm *SquadDeploymentMode) Initialize(ctx *core.UIContext) error {
	// Create deployment service first (needed by UI builders)
	sdm.deploymentService = squadservices.NewSquadDeploymentService(ctx.ECSManager)

	// Build the mode UI first
	err := gui.NewModeBuilder(&sdm.BaseMode, gui.ModeConfig{
		ModeName:   "squad_deployment",
		ReturnMode: "exploration",

		Panels: []gui.PanelSpec{
			{CustomBuild: sdm.buildInstructionText},
			{CustomBuild: sdm.buildSquadList},
			{CustomBuild: sdm.buildDetailPanel},
		},
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Initialize rendering system AFTER BaseMode is initialized (so Queries is available)
	sdm.highlightRenderer = guimodes.NewSquadHighlightRenderer(sdm.Queries)

	// Add action buttons after ModeBuilder completes
	actionButtons := sdm.buildActionButtons()
	sdm.RootContainer.AddChild(actionButtons)

	return nil
}

func (sdm *SquadDeploymentMode) buildInstructionText() *widget.Container {
	// Build instruction text (top-center)
	instructionText := builders.CreateSmallLabel("Select a squad from the list, then click on the map to place it")
	topPad := int(float64(sdm.Layout.ScreenHeight) * specs.PaddingStandard)

	sdm.instructionText = instructionText

	// Wrap in container with LayoutData
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(gui.AnchorCenterStart(topPad))),
	)
	container.AddChild(instructionText)
	return container
}

func (sdm *SquadDeploymentMode) buildSquadList() *widget.Container {
	// Left side squad list (same pattern as unit purchase mode)
	listWidth := int(float64(sdm.Layout.ScreenWidth) * specs.SquadDeployListWidth)
	listHeight := int(float64(sdm.Layout.ScreenHeight) * specs.SquadDeployListHeight)

	baseList := builders.CreateListWithConfig(builders.ListConfig{
		Entries:   []interface{}{}, // Will be populated in Enter
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			if squadID, ok := e.(ecs.EntityID); ok {
				squadName := sdm.Queries.SquadCache.GetSquadName(squadID)
				unitCount := len(sdm.Queries.SquadCache.GetUnitIDsInSquad(squadID))

				// Check if squad has been placed
				allPositions := sdm.deploymentService.GetAllSquadPositions()
				if pos, hasPosition := allPositions[squadID]; hasPosition {
					return fmt.Sprintf("%s (%d units) - Placed at (%d, %d)", squadName, unitCount, pos.X, pos.Y)
				}
				return fmt.Sprintf("%s (%d units)", squadName, unitCount)
			}
			return fmt.Sprintf("%v", e)
		},
		OnEntrySelected: func(selectedEntry interface{}) {
			if squadID, ok := selectedEntry.(ecs.EntityID); ok {
				sdm.selectedSquadID = squadID
				sdm.isPlacingSquad = true
				sdm.updateInstructionText()
				sdm.updateDetailPanel()
			}
		},
	})

	// Wrap with caching for performance (~90% render reduction for static lists)
	sdm.squadList = widgets.NewCachedListWrapper(baseList)

	// Position below instruction text using Start-Start anchor (left-top)
	leftPad := int(float64(sdm.Layout.ScreenWidth) * specs.PaddingStandard)
	topOffset := int(float64(sdm.Layout.ScreenHeight) * (specs.PaddingStandard * 3))

	// Wrap in container with LayoutData
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(gui.AnchorStartStart(leftPad, topOffset))),
	)
	// Add the underlying list to maintain interaction functionality
	container.AddChild(baseList)
	return container
}

func (sdm *SquadDeploymentMode) buildDetailPanel() *widget.Container {
	// Right side detail panel (35% width, 60% height - same as unit purchase mode)
	panelWidth := int(float64(sdm.Layout.ScreenWidth) * 0.35)
	panelHeight := int(float64(sdm.Layout.ScreenHeight) * 0.6)

	detailPanel := builders.CreateStaticPanel(builders.PanelConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(gui.NewResponsiveRowPadding(sdm.Layout, specs.PaddingTight)),
		),
	})

	rightPad := int(float64(sdm.Layout.ScreenWidth) * specs.PaddingStandard)
	detailPanel.GetWidget().LayoutData = gui.AnchorEndCenter(rightPad)

	// Detail text area - cached for performance
	detailTextArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
		MinWidth:  panelWidth - 30,
		MinHeight: panelHeight - 30,
		FontColor: color.White,
	})
	detailTextArea.SetText("Select a squad to view details") // SetText calls MarkDirty() internally
	detailPanel.AddChild(detailTextArea)

	sdm.detailPanel = detailPanel
	sdm.detailTextArea = detailTextArea

	return detailPanel
}

func (sdm *SquadDeploymentMode) buildActionButtons() *widget.Container {
	// Create UI factory
	uiFactory := gui.NewUIComponentFactory(sdm.Queries, sdm.PanelBuilders, sdm.Layout)

	// Create button callbacks (no panel wrapper - like combat mode)
	buttonContainer := uiFactory.CreateSquadDeploymentActionButtons(
		// Clear All
		func() {
			sdm.clearAllSquadPositions()
		},
		// Start Combat
		func() {
			if combatMode, exists := sdm.ModeManager.GetMode("combat"); exists {
				sdm.ModeManager.RequestTransition(combatMode, "Squads deployed, starting combat")
			}
		},
		// Close
		func() {
			if mode, exists := sdm.ModeManager.GetMode("exploration"); exists {
				sdm.ModeManager.RequestTransition(mode, "Close button pressed")
			}
		},
	)

	return buttonContainer
}

func (sdm *SquadDeploymentMode) updateInstructionText() {
	if sdm.selectedSquadID == 0 {
		sdm.instructionText.Label = "Select a squad from the list, then click on the map to place it"
		return
	}

	squadName := sdm.Queries.SquadCache.GetSquadName(sdm.selectedSquadID)
	sdm.instructionText.Label = fmt.Sprintf("Placing %s - Click on the map to position it", squadName)
}

func (sdm *SquadDeploymentMode) updateDetailPanel() {
	if sdm.selectedSquadID == 0 {
		sdm.detailTextArea.SetText("Select a squad to view details")
		return
	}

	squadName := sdm.Queries.SquadCache.GetSquadName(sdm.selectedSquadID)
	unitIDs := sdm.Queries.SquadCache.GetUnitIDsInSquad(sdm.selectedSquadID)

	// Get current deployment position (if any)
	allPositions := sdm.deploymentService.GetAllSquadPositions()
	currentPos, hasPosition := allPositions[sdm.selectedSquadID]

	info := fmt.Sprintf("Squad: %s\nUnits: %d\n\n", squadName, len(unitIDs))

	if hasPosition {
		info += fmt.Sprintf("Current Position: (%d, %d)\n\n", currentPos.X, currentPos.Y)
	} else {
		info += "Not yet placed\n\n"
	}

	info += "Click on the map to place this squad"

	sdm.detailTextArea.SetText(info)
}

func (sdm *SquadDeploymentMode) refreshSquadList() {
	// Repopulate squad list to update placement status in labels
	allSquads := sdm.Queries.SquadCache.FindAllSquads()
	aliveSquads := sdm.Queries.ApplyFilterToSquads(allSquads, sdm.Queries.FilterSquadsAlive())

	entries := make([]interface{}, 0, len(aliveSquads))
	for _, squadID := range aliveSquads {
		entries = append(entries, squadID)
	}
	sdm.squadList.GetList().SetEntries(entries)
	sdm.squadList.MarkDirty() // Trigger re-render with updated entries
}

func (sdm *SquadDeploymentMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Squad Deployment Mode")

	// Populate squad list with all alive squads
	allSquads := sdm.Queries.SquadCache.FindAllSquads()
	aliveSquads := sdm.Queries.ApplyFilterToSquads(allSquads, sdm.Queries.FilterSquadsAlive())

	entries := make([]interface{}, 0, len(aliveSquads))
	for _, squadID := range aliveSquads {
		entries = append(entries, squadID)
	}
	sdm.squadList.GetList().SetEntries(entries)
	sdm.squadList.MarkDirty() // Trigger re-render with updated entries

	sdm.selectedSquadID = 0
	sdm.isPlacingSquad = false
	sdm.updateInstructionText()
	sdm.updateDetailPanel()

	return nil
}

func (sdm *SquadDeploymentMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Squad Deployment Mode")
	return nil
}

func (sdm *SquadDeploymentMode) Update(deltaTime float64) error {
	// Process pending placement (after UI has been updated)
	if sdm.pendingPlacement && sdm.isPlacingSquad && sdm.selectedSquadID != 0 {
		fmt.Printf("DEBUG: Processing pending placement in Update()\n")

		// Get player position (for viewport centering)
		playerPos := *sdm.Context.PlayerData.Pos

		// Convert mouse position to logical position (handles both scrolling modes)
		clickedPos := graphics.MouseToLogicalPosition(sdm.pendingMouseX, sdm.pendingMouseY, playerPos)
		fmt.Printf("DEBUG: Converted click to logical position: (%d, %d)\n", clickedPos.X, clickedPos.Y)

		// Place the squad at the clicked position
		sdm.placeSquadAt(sdm.selectedSquadID, clickedPos)

		sdm.pendingPlacement = false
	}

	return nil
}

func (sdm *SquadDeploymentMode) Render(screen *ebiten.Image) {
	// Render squad highlights showing all squad positions
	playerPos := *sdm.Context.PlayerData.Pos

	// Show all squads with highlights (no faction distinction in deployment mode)
	// Using faction ID 0 to show all squads uniformly
	sdm.highlightRenderer.Render(screen, playerPos, 0, sdm.selectedSquadID)
}

func (sdm *SquadDeploymentMode) HandleInput(inputState *core.InputState) bool {
	// Handle common input (ESC key)
	if sdm.HandleCommonInput(inputState) {
		return true
	}

	// Capture mouse clicks for processing after UI update
	if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
		fmt.Printf("DEBUG: Mouse click captured at (%d, %d), placing=%v, squadID=%d\n",
			inputState.MouseX, inputState.MouseY, sdm.isPlacingSquad, sdm.selectedSquadID)

		// Check if click is inside the squad list (UI area) - don't process as map click
		listBounds := sdm.squadList.GetWidget().Rect
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

	// Find the squad entity
	squadEntity := squads.GetSquadEntity(squadID, sdm.Context.ECSManager)
	if squadEntity == nil {
		fmt.Printf("DEBUG: ERROR - Squad %d not found\n", squadID)
		return
	}

	// Get squad data for name
	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	squadName := "Unknown Squad"
	if squadData != nil {
		squadName = squadData.Name
	}

	// Get current position
	posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
	if posPtr == nil {
		fmt.Printf("DEBUG: ERROR - Squad has no position component\n")
		return
	}

	// Move entity atomically (updates both component and GlobalPositionSystem)
	oldPos := *posPtr
	err := sdm.Context.ECSManager.MoveEntity(squadID, squadEntity, oldPos, pos)
	if err != nil {
		fmt.Printf("DEBUG: ERROR - Failed to move squad: %v\n", err)
		return
	}

	fmt.Printf("✓ Placed %s at (%d, %d) [was at (%d, %d)]\n", squadName, pos.X, pos.Y, oldPos.X, oldPos.Y)

	// Refresh list to show updated placement status
	sdm.refreshSquadList()

	// Reset placement mode
	sdm.isPlacingSquad = false
	sdm.selectedSquadID = 0
	sdm.updateInstructionText()
	sdm.updateDetailPanel()
}

func (sdm *SquadDeploymentMode) clearAllSquadPositions() {
	// Use service to clear all squad positions
	result := sdm.deploymentService.ClearAllSquadPositions()

	if !result.Success {
		fmt.Printf("DEBUG: ERROR - Failed to clear positions: %s\n", result.Error)
		return
	}

	// Refresh list to show updated placement status
	sdm.refreshSquadList()

	sdm.selectedSquadID = 0
	sdm.isPlacingSquad = false
	sdm.updateInstructionText()
	sdm.updateDetailPanel()

	fmt.Printf("All squads cleared (%d squads reset)\n", result.SquadsCleared)
}
