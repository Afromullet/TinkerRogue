package guinodeplacement

import (
	"fmt"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/gui/guioverworld"
	"game_main/overworld/core"
	"game_main/overworld/node"
	"game_main/templates"
	"game_main/world/coords"

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
	lastValidation   *node.PlacementResult

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
	npm.renderer = guioverworld.NewOverworldRenderer(ctx.ECSManager, npm.state, ctx.GameMap, ctx.TileSize, ctx)

	return nil
}

func (npm *NodePlacementMode) Enter(fromMode framework.UIMode) error {
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

	// Validate on hover for preview feedback (includes resource check)
	commanderPos := npm.getSelectedCommanderPos()
	result := node.ValidatePlayerPlacementWithCost(npm.Context.ECSManager, logicalPos, commanderPos, npm.Context.PlayerData.PlayerEntityID, string(npm.selectedNodeType))
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

	playerEntityID := npm.Context.PlayerData.PlayerEntityID
	commanderPos := npm.getSelectedCommanderPos()
	result := node.ValidatePlayerPlacementWithCost(npm.Context.ECSManager, pos, commanderPos, playerEntityID, string(npm.selectedNodeType))
	if !result.Valid {
		npm.setInfo(fmt.Sprintf("Cannot place: %s", result.Reason))
		return
	}

	currentTick := core.GetCurrentTick(npm.Context.ECSManager)
	_, err := node.CreatePlayerNode(npm.Context.ECSManager, pos, npm.selectedNodeType, currentTick, playerEntityID)
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
	npm.refreshPlacementInfo()
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

	count := node.CountPlayerNodes(npm.Context.ECSManager)
	maxNodes := templates.OverworldConfigTemplate.PlayerNodes.MaxNodes

	text := fmt.Sprintf("=== Node Types ===\nPlaced: %d / %d\n\n", count, maxNodes)

	// Get player stockpile for affordability display
	var stockpile *common.ResourceStockpile
	if npm.Context.PlayerData != nil {
		stockpile = common.GetResourceStockpile(npm.Context.PlayerData.PlayerEntityID, npm.Context.ECSManager)
	}

	for i, nodeDef := range npm.nodeTypes {
		marker := "  "
		if nodeDef.ID == string(npm.selectedNodeType) {
			marker = "> "
		}
		costStr := fmt.Sprintf("I:%d W:%d S:%d", nodeDef.Cost.Iron, nodeDef.Cost.Wood, nodeDef.Cost.Stone)
		affordable := ""
		if stockpile != nil && !core.CanAfford(stockpile, nodeDef.Cost) {
			affordable = " [!]"
		}
		text += fmt.Sprintf("%s[%d] %s - %s%s\n", marker, i+1, nodeDef.DisplayName, costStr, affordable)
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

	// Show current resources
	if npm.Context.PlayerData != nil {
		stockpile := common.GetResourceStockpile(npm.Context.PlayerData.PlayerEntityID, npm.Context.ECSManager)
		if stockpile != nil {
			text += fmt.Sprintf("Resources: Iron %d | Wood %d | Stone %d\n\n", stockpile.Iron, stockpile.Wood, stockpile.Stone)
		}
	}

	if npm.selectedNodeType != "" {
		nodeDef := core.GetNodeRegistry().GetNodeByID(string(npm.selectedNodeType))
		if nodeDef != nil {
			text += fmt.Sprintf("Selected: %s\n", nodeDef.DisplayName)
			text += fmt.Sprintf("Category: %s\n", nodeDef.Category)
			text += fmt.Sprintf("Radius: %d\n", nodeDef.BaseRadius)
			text += fmt.Sprintf("Cost: Iron %d | Wood %d | Stone %d\n", nodeDef.Cost.Iron, nodeDef.Cost.Wood, nodeDef.Cost.Stone)
			if len(nodeDef.Services) > 0 {
				text += fmt.Sprintf("Services: %v\n", nodeDef.Services)
			}
		}
	} else {
		text += "No type selected\n"
	}

	text += fmt.Sprintf("\nMax Range: %d tiles\n", templates.OverworldConfigTemplate.PlayerNodes.MaxPlacementRange)
	text += "\nTab: cycle type, 1-4: select type\nClick map to place, ESC to cancel"

	npm.placementInfoText.SetText(text)
}

// getSelectedCommanderPos returns the selected commander's position for placement range checks.
func (npm *NodePlacementMode) getSelectedCommanderPos() *coords.LogicalPosition {
	if npm.state != nil && npm.state.SelectedCommanderID != 0 {
		entity := npm.Context.ECSManager.FindEntityByID(npm.state.SelectedCommanderID)
		if entity != nil {
			return common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		}
	}
	return nil
}

func (npm *NodePlacementMode) setInfo(msg string) {
	if npm.placementInfoText != nil {
		npm.placementInfoText.SetText(msg)
	}
	fmt.Println(msg)
}
