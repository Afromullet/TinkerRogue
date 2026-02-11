package guioverworld

import (
	"image/color"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/overworld/core"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// OverworldRenderer handles rendering overworld nodes and influence zones
type OverworldRenderer struct {
	manager  *common.EntityManager
	state    *framework.OverworldState
	gameMap  *worldmap.GameMap
	tileSize int
	context  *framework.UIContext // For accessing PlayerData
}

// NewOverworldRenderer creates a new overworld renderer
func NewOverworldRenderer(manager *common.EntityManager, state *framework.OverworldState, gameMap *worldmap.GameMap, tileSize int, context *framework.UIContext) *OverworldRenderer {
	return &OverworldRenderer{
		manager:  manager,
		state:    state,
		gameMap:  gameMap,
		tileSize: tileSize,
		context:  context,
	}
}

// Render draws all overworld elements (map tiles, nodes, influence, etc.)
func (r *OverworldRenderer) Render(screen *ebiten.Image) {
	// Render map tiles first (background)
	r.renderOverworldMap(screen)

	// Render influence zones (middle layer)
	if r.state.ShowInfluence {
		r.renderInfluenceZones(screen)
	}

	// Render all nodes (threats, settlements, neutral POIs)
	r.renderNodes(screen)

	// Render player avatar (above nodes)
	r.renderPlayerAvatar(screen)

	// Render selection highlight on top (last)
	if r.state.HasSelection() {
		r.renderSelectionHighlight(screen)
	}
}

// renderOverworldMap draws the game map tiles
func (r *OverworldRenderer) renderOverworldMap(screen *ebiten.Image) {
	if r.gameMap == nil {
		return
	}

	// Render full map with all tiles revealed (strategic view)
	rendering.DrawMap(screen, r.gameMap, true)
}

// renderNodes draws all overworld nodes using the unified OverworldNodeComponent.
// Threats render as circles (size scales with intensity).
// Settlements/fortresses render as squares with owner-colored borders.
func (r *OverworldRenderer) renderNodes(screen *ebiten.Image) {
	for _, result := range core.OverworldNodeView.Get() {
		entity := result.Entity
		pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		data := common.GetComponentType[*core.OverworldNodeData](entity, core.OverworldNodeComponent)

		if pos == nil || data == nil {
			continue
		}

		screenX := (pos.X - r.state.CameraX) * r.tileSize
		screenY := (pos.Y - r.state.CameraY) * r.tileSize
		centerX := float32(screenX) + float32(r.tileSize)/2
		centerY := float32(screenY) + float32(r.tileSize)/2

		switch data.Category {
		case core.NodeCategoryThreat:
			// Threats: circles, size scales with intensity
			radius := float32(8 + (data.Intensity * 2))
			nodeDef := core.GetNodeRegistry().GetNodeByID(data.NodeTypeID)
			threatColor := color.RGBA{R: 200, G: 50, B: 50, A: 255}
			if nodeDef != nil {
				threatColor = nodeDef.Color
			}
			vector.DrawFilledCircle(screen, centerX, centerY, radius, threatColor, true)

		case core.NodeCategorySettlement, core.NodeCategoryFortress:
			// Settlements/fortresses: squares with owner-colored border
			nodeDef := core.GetNodeRegistry().GetNodeByID(data.NodeTypeID)
			nodeColor := color.RGBA{R: 100, G: 200, B: 100, A: 255}
			if nodeDef != nil {
				nodeColor = nodeDef.Color
			}

			halfSize := float32(r.tileSize) / 2
			vector.DrawFilledRect(screen, centerX-halfSize, centerY-halfSize, halfSize*2, halfSize*2, nodeColor, true)

			// Border color based on owner
			borderColor := r.getOwnerBorderColor(data.OwnerID)
			vector.StrokeRect(screen, centerX-halfSize, centerY-halfSize, halfSize*2, halfSize*2, 2, borderColor, true)
		}
	}
}

// renderInfluenceZones draws influence radius for all nodes using a single unified query.
func (r *OverworldRenderer) renderInfluenceZones(screen *ebiten.Image) {
	for _, result := range core.OverworldNodeView.Get() {
		entity := result.Entity
		pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		influenceData := common.GetComponentType[*core.InfluenceData](entity, core.InfluenceComponent)
		nodeData := common.GetComponentType[*core.OverworldNodeData](entity, core.OverworldNodeComponent)

		if pos == nil || influenceData == nil || nodeData == nil {
			continue
		}

		screenX := (pos.X - r.state.CameraX) * r.tileSize
		screenY := (pos.Y - r.state.CameraY) * r.tileSize
		centerX := float32(screenX) + float32(r.tileSize)/2
		centerY := float32(screenY) + float32(r.tileSize)/2
		influenceRadius := float32(influenceData.Radius * r.tileSize)

		// Color based on owner type
		var influenceColor color.RGBA
		if core.IsHostileOwner(nodeData.OwnerID) {
			influenceColor = color.RGBA{255, 200, 100, 50} // warm for hostile
		} else if nodeData.OwnerID == core.OwnerNeutral {
			influenceColor = color.RGBA{220, 200, 100, 40} // muted yellow for neutral
		} else {
			influenceColor = color.RGBA{100, 200, 255, 50} // cool for player
		}

		vector.DrawFilledCircle(screen, centerX, centerY, influenceRadius, influenceColor, true)
	}
}

// renderPlayerAvatar draws the player sprite at their current position
func (r *OverworldRenderer) renderPlayerAvatar(screen *ebiten.Image) {
	if r.context == nil || r.context.PlayerData == nil {
		return
	}

	playerEntity := r.manager.FindEntityByID(r.context.PlayerData.PlayerEntityID)
	if playerEntity == nil {
		return
	}

	pos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)
	if pos == nil {
		return
	}

	renderable := common.GetComponentType[*rendering.Renderable](playerEntity, rendering.RenderableComponent)
	if renderable == nil || renderable.Image == nil || !renderable.Visible {
		return
	}

	screenX := float64((pos.X - r.state.CameraX) * r.tileSize)
	screenY := float64((pos.Y - r.state.CameraY) * r.tileSize)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY)

	screen.DrawImage(renderable.Image, op)
}

// renderSelectionHighlight draws a highlight around the selected node.
// Uses unified OverworldNodeData.
func (r *OverworldRenderer) renderSelectionHighlight(screen *ebiten.Image) {
	selected := r.manager.FindEntityByID(r.state.SelectedNodeID)
	if selected == nil {
		return
	}

	pos := common.GetComponentType[*coords.LogicalPosition](selected, common.PositionComponent)
	data := common.GetComponentType[*core.OverworldNodeData](selected, core.OverworldNodeComponent)

	if pos == nil || data == nil {
		return
	}

	screenX := (pos.X - r.state.CameraX) * r.tileSize
	screenY := (pos.Y - r.state.CameraY) * r.tileSize
	centerX := float32(screenX) + float32(r.tileSize)/2
	centerY := float32(screenY) + float32(r.tileSize)/2

	selectionColor := color.RGBA{255, 255, 255, 200}

	if data.Category == core.NodeCategoryThreat {
		radius := float32(8 + (data.Intensity * 2) + 4)
		vector.StrokeCircle(screen, centerX, centerY, radius, 2, selectionColor, true)
	} else {
		halfSize := float32(r.tileSize)/2 + 2
		vector.StrokeRect(screen, centerX-halfSize, centerY-halfSize, halfSize*2, halfSize*2, 2, selectionColor, true)
	}
}

// getOwnerBorderColor returns the border color based on node owner.
func (r *OverworldRenderer) getOwnerBorderColor(ownerID string) color.RGBA {
	switch {
	case ownerID == core.OwnerPlayer:
		return color.RGBA{R: 255, G: 255, B: 255, A: 180} // white for player
	case ownerID == core.OwnerNeutral:
		return color.RGBA{R: 218, G: 165, B: 32, A: 200} // gold for neutral
	default:
		return color.RGBA{R: 255, G: 50, B: 50, A: 180} // red for hostile
	}
}

// ScreenToLogical converts screen coordinates to logical tile position (accounting for camera).
func (r *OverworldRenderer) ScreenToLogical(screenX, screenY int) coords.LogicalPosition {
	logicalX := (screenX / r.tileSize) + r.state.CameraX
	logicalY := (screenY / r.tileSize) + r.state.CameraY
	return coords.LogicalPosition{X: logicalX, Y: logicalY}
}

// GetThreatAtPosition returns threat entity at screen coordinates (for mouse clicks)
func (r *OverworldRenderer) GetThreatAtPosition(screenX, screenY int) ecs.EntityID {
	logicalPos := r.ScreenToLogical(screenX, screenY)
	return core.GetThreatNodeAt(r.manager, logicalPos)
}

// GetNodeAtPosition returns any overworld node at screen coordinates (threats, settlements, etc.)
func (r *OverworldRenderer) GetNodeAtPosition(screenX, screenY int) ecs.EntityID {
	logicalPos := r.ScreenToLogical(screenX, screenY)
	return core.GetNodeAtPosition(r.manager, logicalPos)
}

