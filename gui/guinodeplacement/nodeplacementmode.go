package guinodeplacement

import (
	"fmt"

	"game_main/gui/framework"
	"game_main/gui/guioverworld"
	"game_main/overworld/core"
	"game_main/overworld/playernode"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// NodePlacementMode handles placing player nodes on the overworld map.
type NodePlacementMode struct {
	framework.BaseMode

	state    *framework.OverworldState
	renderer *guioverworld.OverworldRenderer

	// Placement state
	selectedNodeType core.NodeTypeID
	nodeTypes        []*core.NodeDefinition
	cursorPos        *coords.LogicalPosition
	lastValidation   *playernode.PlacementResult

	// Widget references
	nodeListText      *widget.TextArea
	placementInfoText *widget.TextArea
}

func NewNodePlacementMode(modeManager *framework.UIModeManager) *NodePlacementMode {
	npm := &NodePlacementMode{}
	npm.SetModeName("node_placement")
	npm.SetReturnMode("overworld")
	npm.ModeManager = modeManager
	npm.SetSelf(npm)
	return npm
}

func (npm *NodePlacementMode) Initialize(ctx *framework.UIContext) error {
	npm.state = ctx.ModeCoordinator.GetOverworldState()

	err := framework.NewModeBuilder(&npm.BaseMode, framework.ModeConfig{
		ModeName:   "node_placement",
		ReturnMode: "overworld",
	}).Build(ctx)
	if err != nil {
		return err
	}

	// Build panels
	if err := npm.BuildPanels(
		NodePlacementPanelNodeList,
		NodePlacementPanelInfo,
		NodePlacementPanelControls,
	); err != nil {
		return err
	}

	// Initialize widget references
	npm.nodeListText = GetNodeListText(npm.Panels)
	npm.placementInfoText = GetPlacementInfoText(npm.Panels)

	// Create renderer (reuse overworld renderer for map + threat drawing)
	gameMap, ok := ctx.GameMap.(*worldmap.GameMap)
	if !ok {
		return fmt.Errorf("GameMap is not *worldmap.GameMap")
	}
	npm.renderer = guioverworld.NewOverworldRenderer(ctx.ECSManager, npm.state, gameMap, ctx.TileSize, ctx)

	return nil
}

func (npm *NodePlacementMode) Enter(fromMode framework.UIMode) error {
	fmt.Println("Entering Node Placement Mode")

	// Load placeable node types
	npm.nodeTypes = core.GetNodeRegistry().GetPlaceableNodeTypes()

	// Default select first type if available
	if len(npm.nodeTypes) > 0 && npm.selectedNodeType == "" {
		npm.selectedNodeType = core.NodeTypeID(npm.nodeTypes[0].ID)
	}

	npm.refreshNodeList()
	npm.refreshPlacementInfo()

	return nil
}

func (npm *NodePlacementMode) Exit(toMode framework.UIMode) error {
	fmt.Println("Exiting Node Placement Mode")
	npm.cursorPos = nil
	npm.lastValidation = nil
	return nil
}

func (npm *NodePlacementMode) Update(deltaTime float64) error {
	return nil
}

func (npm *NodePlacementMode) Render(screen *ebiten.Image) {
	// Render the overworld map underneath (includes threats, player nodes, avatar)
	if npm.renderer != nil {
		npm.renderer.Render(screen)
	}

	// Render placement preview overlay
	npm.renderPlacementPreview(screen)

	// Render HUD showing selected node type
	npm.renderSelectionHUD(screen)
}

func (npm *NodePlacementMode) HandleInput(inputState *framework.InputState) bool {
	// ESC -> return to overworld
	if npm.HandleCommonInput(inputState) {
		return true
	}

	// Tab to cycle through node types
	if inputState.KeysJustPressed[ebiten.KeyTab] && len(npm.nodeTypes) > 0 {
		npm.cycleNodeType()
		return true
	}

	// Number keys 1-4 to select node type directly
	numberKeys := []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4}
	for i, key := range numberKeys {
		if inputState.KeysJustPressed[key] && i < len(npm.nodeTypes) {
			npm.selectedNodeType = core.NodeTypeID(npm.nodeTypes[i].ID)
			npm.refreshNodeList()
			npm.refreshPlacementInfo()
			return true
		}
	}

	// Track mouse position for preview
	logicalPos := npm.renderer.ScreenToLogical(inputState.MouseX, inputState.MouseY)
	npm.cursorPos = &logicalPos

	// Validate on hover for preview feedback
	result := playernode.ValidatePlacement(npm.Context.ECSManager, logicalPos, npm.Context.PlayerData)
	npm.lastValidation = &result

	// Left click to place node
	if inputState.MousePressed && inputState.MouseButton == ebiten.MouseButtonLeft {
		npm.handlePlaceNode(logicalPos)
		return true
	}

	return false
}

func (npm *NodePlacementMode) handlePlaceNode(pos coords.LogicalPosition) {
	if npm.selectedNodeType == "" {
		npm.setInfo("No node type selected")
		return
	}

	result := playernode.ValidatePlacement(npm.Context.ECSManager, pos, npm.Context.PlayerData)
	if !result.Valid {
		npm.setInfo(fmt.Sprintf("Cannot place: %s", result.Reason))
		return
	}

	currentTick := core.GetCurrentTick(npm.Context.ECSManager)
	_, err := playernode.CreatePlayerNode(npm.Context.ECSManager, pos, npm.selectedNodeType, currentTick)
	if err != nil {
		npm.setInfo(fmt.Sprintf("Failed to place node: %v", err))
		return
	}

	nodeDef := core.GetNodeRegistry().GetNodeByID(string(npm.selectedNodeType))
	displayName := string(npm.selectedNodeType)
	if nodeDef != nil {
		displayName = nodeDef.DisplayName
	}

	npm.setInfo(fmt.Sprintf("Placed %s at (%d, %d)", displayName, pos.X, pos.Y))
	npm.refreshNodeList()
}

func (npm *NodePlacementMode) cycleNodeType() {
	currentIdx := -1
	for i, node := range npm.nodeTypes {
		if node.ID == string(npm.selectedNodeType) {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + 1) % len(npm.nodeTypes)
	npm.selectedNodeType = core.NodeTypeID(npm.nodeTypes[nextIdx].ID)
	npm.refreshNodeList()
	npm.refreshPlacementInfo()
}

func (npm *NodePlacementMode) refreshNodeList() {
	if npm.nodeListText == nil {
		return
	}

	count := playernode.CountPlayerNodes(npm.Context.ECSManager)
	maxNodes := core.GetMaxPlayerNodes()

	text := fmt.Sprintf("=== Node Types ===\nPlaced: %d / %d\n\n", count, maxNodes)

	for i, node := range npm.nodeTypes {
		marker := "  "
		if node.ID == string(npm.selectedNodeType) {
			marker = "> "
		}
		text += fmt.Sprintf("%s[%d] %s (%s)\n", marker, i+1, node.DisplayName, node.Category)
	}

	if len(npm.nodeTypes) == 0 {
		text += "(No placeable node types defined)"
	}

	npm.nodeListText.SetText(text)
}

func (npm *NodePlacementMode) refreshPlacementInfo() {
	if npm.placementInfoText == nil {
		return
	}

	text := "=== Placement Info ===\n"

	if npm.selectedNodeType != "" {
		nodeDef := core.GetNodeRegistry().GetNodeByID(string(npm.selectedNodeType))
		if nodeDef != nil {
			text += fmt.Sprintf("Selected: %s\n", nodeDef.DisplayName)
			text += fmt.Sprintf("Category: %s\n", nodeDef.Category)
			text += fmt.Sprintf("Radius: %d\n", nodeDef.BaseRadius)
			if len(nodeDef.Services) > 0 {
				text += fmt.Sprintf("Services: %v\n", nodeDef.Services)
			}
		}
	} else {
		text += "No type selected\n"
	}

	text += fmt.Sprintf("\nMax Range: %d tiles\n", core.GetMaxPlacementRange())
	text += "\nTab: cycle type, 1-4: select type\nClick map to place, ESC to cancel"

	npm.placementInfoText.SetText(text)
}

func (npm *NodePlacementMode) setInfo(msg string) {
	if npm.placementInfoText != nil {
		npm.placementInfoText.SetText(msg)
	}
	fmt.Println(msg)
}

