package guisquads

import (
	"fmt"
	"game_main/coords"
	"game_main/graphics"
	"game_main/gui"
	"game_main/gui/core"
	"game_main/gui/guicomponents"
	"game_main/gui/guimodes"
	"game_main/gui/widgets"
	"game_main/squads"
	"game_main/squads/squadservices"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadDeploymentMode allows placing squads on the map before combat
type SquadDeploymentMode struct {
	gui.BaseMode // Embed common mode infrastructure

	deploymentService  *squadservices.SquadDeploymentService
	squadListPanel     *widget.Container
	squadListComponent *guicomponents.SquadListComponent
	selectedSquadID    ecs.EntityID
	instructionText    *widget.Text

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
	// Initialize common mode infrastructure
	sdm.InitializeBase(ctx)

	// Create deployment service
	sdm.deploymentService = squadservices.NewSquadDeploymentService(ctx.ECSManager)

	// Build UI components
	sdm.buildSquadListPanel()

	// Build instruction text (top-center) with explicit size
	sdm.instructionText = widgets.CreateSmallLabel("Select a squad from the list, then click on the map to place it")
	topPad := int(float64(sdm.Layout.ScreenHeight) * widgets.PaddingStandard)
	sdm.instructionText.GetWidget().LayoutData = gui.AnchorCenterStart(topPad)
	sdm.RootContainer.AddChild(sdm.instructionText)

	sdm.buildActionButtons()

	// Initialize rendering system
	sdm.highlightRenderer = guimodes.NewSquadHighlightRenderer(sdm.Queries)

	return nil
}

func (sdm *SquadDeploymentMode) buildSquadListPanel() {
	// Build squad list panel using BuildPanel with deployment-specific constants
	sdm.squadListPanel = sdm.PanelBuilders.BuildPanel(
		widgets.LeftCenter(),
		widgets.Size(widgets.SquadDeployListWidth, widgets.SquadDeployListHeight),
		widgets.Padding(widgets.PaddingTight),
		widgets.RowLayout(),
	)

	// Add label
	listLabel := widgets.CreateSmallLabel("Squads:")
	sdm.squadListPanel.AddChild(listLabel)

	// Create squad list component - show all alive squads for placement
	// Uses centralized filter from guicomponents.GUIQueries for consistency
	sdm.squadListComponent = guicomponents.NewSquadListComponent(
		sdm.squadListPanel,
		sdm.Queries,
		sdm.Queries.FilterSquadsAlive(),
		func(squadID ecs.EntityID) {
			sdm.selectedSquadID = squadID
			sdm.isPlacingSquad = true
			sdm.updateInstructionText()
		},
	)

	sdm.RootContainer.AddChild(sdm.squadListPanel)
}

func (sdm *SquadDeploymentMode) buildActionButtons() {
	// Build action buttons using helper (consolidates positioning + button creation)
	buttonSpecs := []widgets.ButtonSpec{
		{
			Text: "Clear All",
			OnClick: func() {
				sdm.clearAllSquadPositions()
			},
		},
		{
			Text: "Start Combat",
			OnClick: func() {
				if combatMode, exists := sdm.ModeManager.GetMode("combat"); exists {
					sdm.ModeManager.RequestTransition(combatMode, "Squads deployed, starting combat")
				}
			},
		},
		{
			Text: "Close (ESC)",
			OnClick: func() {
				if mode, exists := sdm.ModeManager.GetMode("exploration"); exists {
					sdm.ModeManager.RequestTransition(mode, "Close button pressed")
				}
			},
		},
	}

	buttonContainer := gui.CreateActionButtonGroup(sdm.PanelBuilders, widgets.BottomCenter(), buttonSpecs)
	sdm.RootContainer.AddChild(buttonContainer)
}

func (sdm *SquadDeploymentMode) updateInstructionText() {
	if sdm.selectedSquadID == 0 {
		sdm.instructionText.Label = "Select a squad from the list, then click on the map to place it"
		return
	}

	squadName := squads.GetSquadName(sdm.selectedSquadID, sdm.Queries.ECSManager)
	sdm.instructionText.Label = fmt.Sprintf("Placing %s - Click on the map to position it", squadName)
}

func (sdm *SquadDeploymentMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Squad Deployment Mode")

	// Refresh the squad list using the component
	sdm.squadListComponent.Refresh()

	sdm.selectedSquadID = 0
	sdm.isPlacingSquad = false
	sdm.updateInstructionText()

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

	// Use service to place squad
	result := sdm.deploymentService.PlaceSquadAtPosition(squadID, pos)

	if !result.Success {
		fmt.Printf("DEBUG: ERROR - Failed to place squad: %s\n", result.Error)
		return
	}

	oldPos := sdm.deploymentService.GetAllSquadPositions()[squadID]
	fmt.Printf("✓ Placed %s at (%d, %d) [was at (%d, %d)]\n", result.SquadName, pos.X, pos.Y, oldPos.X, oldPos.Y)

	// Reset placement mode
	sdm.isPlacingSquad = false
	sdm.selectedSquadID = 0
	sdm.updateInstructionText()
}

func (sdm *SquadDeploymentMode) clearAllSquadPositions() {
	// Use service to clear all squad positions
	result := sdm.deploymentService.ClearAllSquadPositions()

	if !result.Success {
		fmt.Printf("DEBUG: ERROR - Failed to clear positions: %s\n", result.Error)
		return
	}

	sdm.selectedSquadID = 0
	sdm.isPlacingSquad = false
	sdm.updateInstructionText()

	fmt.Printf("All squads cleared (%d squads reset)\n", result.SquadsCleared)
}
