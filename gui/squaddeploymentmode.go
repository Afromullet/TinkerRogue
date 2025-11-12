package gui

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/graphics"
	"game_main/squads"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadDeploymentMode allows placing squads on the map before combat
type SquadDeploymentMode struct {
	BaseMode // Embed common mode infrastructure

	squadListPanel    *widget.Container
	squadListComponent *SquadListComponent
	selectedSquadID   ecs.EntityID
	instructionText   *widget.Text
	confirmButton     *widget.Button
	clearAllButton    *widget.Button
	closeButton       *widget.Button

	isPlacingSquad   bool
	pendingMouseX    int
	pendingMouseY    int
	pendingPlacement bool

	// Rendering systems
	highlightRenderer *SquadHighlightRenderer
}

func NewSquadDeploymentMode(modeManager *UIModeManager) *SquadDeploymentMode {
	return &SquadDeploymentMode{
		BaseMode: BaseMode{
			modeManager: modeManager,
			modeName:    "squad_deployment",
			returnMode:  "exploration",
		},
	}
}

func (sdm *SquadDeploymentMode) Initialize(ctx *UIContext) error {
	// Initialize common mode infrastructure
	sdm.InitializeBase(ctx)

	// Build UI components
	sdm.buildSquadListPanel()

	// Build instruction text (top-center) using BuildPanel
	sdm.instructionText = CreateTextWithConfig(TextConfig{
		Text:     "Select a squad from the list, then click on the map to place it",
		FontFace: SmallFace,
		Color:    color.White,
		LayoutData: widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
			Padding: widget.Insets{
				Top: int(float64(sdm.layout.ScreenHeight) * 0.02),
			},
		},
	})
	sdm.rootContainer.AddChild(sdm.instructionText)

	sdm.buildActionButtons()

	// Initialize rendering system
	sdm.highlightRenderer = NewSquadHighlightRenderer(sdm.queries)

	return nil
}

func (sdm *SquadDeploymentMode) buildSquadListPanel() {
	// Build squad list panel using BuildPanel
	sdm.squadListPanel = sdm.panelBuilders.BuildPanel(
		LeftCenter(),
		Size(0.2, 0.8),
		Padding(0.01),
		RowLayout(),
	)

	// Add label
	listLabel := CreateTextWithConfig(TextConfig{
		Text:     "Squads:",
		FontFace: SmallFace,
		Color:    color.White,
	})
	sdm.squadListPanel.AddChild(listLabel)

	// Create squad list component - show all alive squads for placement
	// Uses centralized AliveSquadsOnly filter to eliminate code duplication
	sdm.squadListComponent = NewSquadListComponent(
		sdm.squadListPanel,
		sdm.queries,
		AliveSquadsOnly,
		func(squadID ecs.EntityID) {
			sdm.selectedSquadID = squadID
			sdm.isPlacingSquad = true
			sdm.updateInstructionText()
		},
	)

	sdm.rootContainer.AddChild(sdm.squadListPanel)
}

func (sdm *SquadDeploymentMode) buildActionButtons() {
	// Create action buttons
	sdm.clearAllButton = CreateButtonWithConfig(ButtonConfig{
		Text: "Clear All",
		OnClick: func() {
			sdm.clearAllSquadPositions()
		},
	})

	sdm.confirmButton = CreateButtonWithConfig(ButtonConfig{
		Text: "Start Combat",
		OnClick: func() {
			if combatMode, exists := sdm.modeManager.GetMode("combat"); exists {
				sdm.modeManager.RequestTransition(combatMode, "Squads deployed, starting combat")
			}
		},
	})

	// Create close button using helper
	sdm.closeButton = CreateCloseButton(sdm.modeManager, "exploration", "Close (ESC)")

	// Build action buttons container using BuildPanel
	buttonContainer := sdm.panelBuilders.BuildPanel(
		BottomCenter(),
		HorizontalRowLayout(),
		CustomPadding(widget.Insets{
			Bottom: int(float64(sdm.layout.ScreenHeight) * 0.08),
		}),
	)

	buttonContainer.AddChild(sdm.clearAllButton)
	buttonContainer.AddChild(sdm.confirmButton)
	buttonContainer.AddChild(sdm.closeButton)
	sdm.rootContainer.AddChild(buttonContainer)
}

func (sdm *SquadDeploymentMode) updateInstructionText() {
	if sdm.selectedSquadID == 0 {
		sdm.instructionText.Label = "Select a squad from the list, then click on the map to place it"
		return
	}

	squadName := sdm.queries.GetSquadName(sdm.selectedSquadID)
	sdm.instructionText.Label = fmt.Sprintf("Placing %s - Click on the map to position it", squadName)
}


func (sdm *SquadDeploymentMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Squad Deployment Mode")

	// Refresh the squad list using the component
	sdm.squadListComponent.Refresh()

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
	// Render squad highlights showing all squad positions
	playerPos := *sdm.context.PlayerData.Pos

	// Show all squads with highlights (no faction distinction in deployment mode)
	// Using faction ID 0 to show all squads uniformly
	sdm.highlightRenderer.Render(screen, playerPos, 0, sdm.selectedSquadID)
}

func (sdm *SquadDeploymentMode) HandleInput(inputState *InputState) bool {
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

				squadName := sdm.queries.GetSquadName(squadID)
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

