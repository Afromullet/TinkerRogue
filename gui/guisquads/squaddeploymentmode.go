package guisquads

import (
	"fmt"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/gui/widgets"
	"game_main/tactical/squads"
	"game_main/tactical/squadservices"
	"game_main/visual/graphics"
	"game_main/visual/rendering"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadDeploymentMode allows placing squads on the map before combat
type SquadDeploymentMode struct {
	framework.BaseMode // Embed common mode infrastructure

	deploymentService *squadservices.SquadDeploymentService

	// Interactive widget references (stored here for refresh/access)
	// These are populated from panel registry after BuildPanels()
	squadList       *widgets.CachedListWrapper
	detailTextArea  *widgets.CachedTextAreaWrapper
	instructionText *widget.Text

	selectedSquadID ecs.EntityID

	isPlacingSquad   bool
	pendingMouseX    int
	pendingMouseY    int
	pendingPlacement bool

	// Rendering systems
	highlightRenderer *rendering.SquadHighlightRenderer
}

func NewSquadDeploymentMode(modeManager *framework.UIModeManager) *SquadDeploymentMode {
	mode := &SquadDeploymentMode{}
	mode.SetModeName("squad_deployment")
	mode.SetReturnMode("exploration") // ESC returns to exploration
	mode.ModeManager = modeManager
	mode.SetSelf(mode) // Required for panel registry building
	return mode
}

func (sdm *SquadDeploymentMode) Initialize(ctx *framework.UIContext) error {
	// Create deployment service first (needed by UI builders)
	sdm.deploymentService = squadservices.NewSquadDeploymentService(ctx.ECSManager)

	// Build base UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&sdm.BaseMode, framework.ModeConfig{
		ModeName:   "squad_deployment",
		ReturnMode: "exploration",
	}).Build(ctx)

	if err != nil {
		return err
	}

	// Build panels from registry
	if err := sdm.BuildPanels(
		SquadDeploymentPanelInstruction,
		SquadDeploymentPanelSquadList,
		SquadDeploymentPanelDetailPanel,
		SquadDeploymentPanelActionButtons,
	); err != nil {
		return err
	}

	// Initialize widget references from registry
	sdm.initializeWidgetReferences()

	// Initialize rendering system AFTER BaseMode is initialized (so Queries is available)
	sdm.highlightRenderer = rendering.NewSquadHighlightRenderer(sdm.Queries)

	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (sdm *SquadDeploymentMode) initializeWidgetReferences() {
	sdm.instructionText = framework.GetPanelWidget[*widget.Text](sdm.Panels, SquadDeploymentPanelInstruction, "instructionText")
	sdm.squadList = framework.GetPanelWidget[*widgets.CachedListWrapper](sdm.Panels, SquadDeploymentPanelSquadList, "squadList")
	sdm.detailTextArea = framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](sdm.Panels, SquadDeploymentPanelDetailPanel, "detailTextArea")
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

func (sdm *SquadDeploymentMode) Enter(fromMode framework.UIMode) error {
	fmt.Println("Entering Squad Deployment Mode")

	sdm.refreshSquadList()

	sdm.selectedSquadID = 0
	sdm.isPlacingSquad = false
	sdm.updateInstructionText()
	sdm.updateDetailPanel()

	return nil
}

func (sdm *SquadDeploymentMode) Exit(toMode framework.UIMode) error {
	fmt.Println("Exiting Squad Deployment Mode")
	return nil
}

func (sdm *SquadDeploymentMode) Update(deltaTime float64) error {
	// Process pending placement (after UI has been updated)
	if sdm.pendingPlacement && sdm.isPlacingSquad && sdm.selectedSquadID != 0 {
		// Get player position (for viewport centering)
		playerPos := *sdm.Context.PlayerData.Pos

		// Convert mouse position to logical position (handles both scrolling modes)
		clickedPos := graphics.MouseToLogicalPosition(sdm.pendingMouseX, sdm.pendingMouseY, playerPos)

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

func (sdm *SquadDeploymentMode) HandleInput(inputState *framework.InputState) bool {
	// Handle common input (ESC key)
	if sdm.HandleCommonInput(inputState) {
		return true
	}

	// Capture mouse clicks for processing after UI update
	if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
		// Check if click is inside the squad list (UI area) - don't process as map click
		listBounds := sdm.squadList.GetWidget().Rect
		isInsideList := inputState.MouseX >= listBounds.Min.X && inputState.MouseX <= listBounds.Max.X &&
			inputState.MouseY >= listBounds.Min.Y && inputState.MouseY <= listBounds.Max.Y

		if isInsideList {
			return false
		}

		// Store the click for processing in Update() after ebitenui has processed widget events
		sdm.pendingMouseX = inputState.MouseX
		sdm.pendingMouseY = inputState.MouseY
		sdm.pendingPlacement = true
	}

	return false
}

func (sdm *SquadDeploymentMode) placeSquadAt(squadID ecs.EntityID, pos coords.LogicalPosition) {
	// Find the squad entity
	squadEntity := squads.GetSquadEntity(squadID, sdm.Context.ECSManager)
	if squadEntity == nil {
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
		return
	}

	// Move entity atomically (updates both component and GlobalPositionSystem)
	oldPos := *posPtr
	err := sdm.Context.ECSManager.MoveEntity(squadID, squadEntity, oldPos, pos)
	if err != nil {
		return
	}

	fmt.Printf("Placed %s at (%d, %d) [was at (%d, %d)]\n", squadName, pos.X, pos.Y, oldPos.X, oldPos.Y)

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
