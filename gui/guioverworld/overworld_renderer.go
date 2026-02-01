package guioverworld

import (
	"fmt"
	"image/color"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"game_main/world/overworld"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// OverworldRenderer handles rendering threat nodes and influence zones
type OverworldRenderer struct {
	manager  *common.EntityManager
	state    *OverworldState
	gameMap  *worldmap.GameMap
	tileSize int
	context  *framework.UIContext // For accessing PlayerData
}

// NewOverworldRenderer creates a new overworld renderer
func NewOverworldRenderer(manager *common.EntityManager, state *OverworldState, gameMap *worldmap.GameMap, tileSize int, context *framework.UIContext) *OverworldRenderer {
	return &OverworldRenderer{
		manager:  manager,
		state:    state,
		gameMap:  gameMap,
		tileSize: tileSize,
		context:  context,
	}
}

// Render draws all overworld elements (map tiles, threat nodes, influence, etc.)
func (r *OverworldRenderer) Render(screen *ebiten.Image) {
	// Render map tiles first (background)
	r.renderOverworldMap(screen)

	// Render influence zones (middle layer)
	if r.state.ShowInfluence {
		r.renderInfluenceZones(screen)
	}

	// Render threat nodes
	r.renderThreatNodes(screen)

	// Render player avatar (above threats)
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

// renderThreatNodes draws all threat nodes as colored circles
func (r *OverworldRenderer) renderThreatNodes(screen *ebiten.Image) {
	// Query threats directly following ECS best practices
	for _, result := range r.manager.World.Query(overworld.ThreatNodeTag) {
		threat := result.Entity
		pos := common.GetComponentType[*coords.LogicalPosition](threat, common.PositionComponent)
		data := common.GetComponentType[*overworld.ThreatNodeData](threat, overworld.ThreatNodeComponent)

		if pos == nil || data == nil {
			continue
		}

		// Calculate screen position (accounting for camera)
		screenX := (pos.X - r.state.CameraX) * r.tileSize
		screenY := (pos.Y - r.state.CameraY) * r.tileSize

		// Size scales with intensity (minimum 8px, maximum 32px)
		radius := float32(8 + (data.Intensity * 2))

		// Color based on threat type
		threatColor := r.getThreatColor(data.ThreatType)

		// Draw circle for threat node
		centerX := float32(screenX) + float32(r.tileSize)/2
		centerY := float32(screenY) + float32(r.tileSize)/2

		vector.DrawFilledCircle(screen, centerX, centerY, radius, threatColor, true)

		// Draw intensity number in center
		// Note: Text rendering requires a different approach - would need ebitenutil.DebugPrintAt
		// For Phase 1, just draw the circle
	}
}

// renderInfluenceZones draws influence radius for threats
func (r *OverworldRenderer) renderInfluenceZones(screen *ebiten.Image) {
	// Query threats directly following ECS best practices
	for _, result := range r.manager.World.Query(overworld.ThreatNodeTag) {
		threat := result.Entity
		pos := common.GetComponentType[*coords.LogicalPosition](threat, common.PositionComponent)
		influenceData := common.GetComponentType[*overworld.InfluenceData](threat, overworld.InfluenceComponent)

		if pos == nil || influenceData == nil {
			continue
		}

		// Calculate screen position
		screenX := (pos.X - r.state.CameraX) * r.tileSize
		screenY := (pos.Y - r.state.CameraY) * r.tileSize

		// Draw semi-transparent circle for influence radius
		centerX := float32(screenX) + float32(r.tileSize)/2
		centerY := float32(screenY) + float32(r.tileSize)/2
		influenceRadius := float32(influenceData.Radius * r.tileSize)

		// Semi-transparent red/yellow tint
		influenceColor := color.RGBA{255, 200, 100, 50}

		vector.DrawFilledCircle(screen, centerX, centerY, influenceRadius, influenceColor, true)
	}
}

// renderPlayerAvatar draws the player sprite at their current position
func (r *OverworldRenderer) renderPlayerAvatar(screen *ebiten.Image) {
	// 1. Get player entity from manager
	if r.context == nil || r.context.PlayerData == nil {
		return
	}

	playerEntity := r.manager.FindEntityByID(r.context.PlayerData.PlayerEntityID)
	if playerEntity == nil {
		return // Player not initialized yet
	}

	// 2. Get player position component
	pos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)
	if pos == nil {
		return
	}

	// 3. Get player renderable component (has sprite image)
	renderable := common.GetComponentType[*rendering.Renderable](playerEntity, rendering.RenderableComponent)
	if renderable == nil || renderable.Image == nil || !renderable.Visible {
		return
	}

	// 4. Calculate screen position (using camera offset)
	screenX := float64((pos.X - r.state.CameraX) * r.tileSize)
	screenY := float64((pos.Y - r.state.CameraY) * r.tileSize)

	// 5. Draw player sprite
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY)

	screen.DrawImage(renderable.Image, op)
}

// renderSelectionHighlight draws a highlight around the selected threat
func (r *OverworldRenderer) renderSelectionHighlight(screen *ebiten.Image) {
	threat := r.manager.FindEntityByID(r.state.SelectedThreatID)
	if threat == nil {
		return
	}

	pos := common.GetComponentType[*coords.LogicalPosition](threat, common.PositionComponent)
	data := common.GetComponentType[*overworld.ThreatNodeData](threat, overworld.ThreatNodeComponent)

	if pos == nil || data == nil {
		return
	}

	// Calculate screen position
	screenX := (pos.X - r.state.CameraX) * r.tileSize
	screenY := (pos.Y - r.state.CameraY) * r.tileSize

	// Draw selection ring
	centerX := float32(screenX) + float32(r.tileSize)/2
	centerY := float32(screenY) + float32(r.tileSize)/2
	radius := float32(8 + (data.Intensity * 2) + 4)

	selectionColor := color.RGBA{255, 255, 255, 200}

	// Draw ring (circle outline)
	vector.StrokeCircle(screen, centerX, centerY, radius, 2, selectionColor, true)
}

// getThreatColor returns color for each threat type
func (r *OverworldRenderer) getThreatColor(threatType overworld.ThreatType) color.RGBA {
	switch threatType {
	case overworld.ThreatNecromancer:
		return color.RGBA{150, 50, 150, 255} // Purple
	case overworld.ThreatBanditCamp:
		return color.RGBA{200, 100, 50, 255} // Brown
	case overworld.ThreatCorruption:
		return color.RGBA{100, 200, 50, 255} // Sickly green
	case overworld.ThreatBeastNest:
		return color.RGBA{200, 150, 50, 255} // Orange
	case overworld.ThreatOrcWarband:
		return color.RGBA{200, 50, 50, 255} // Red
	default:
		return color.RGBA{128, 128, 128, 255} // Gray
	}
}

// GetThreatAtPosition returns threat entity at screen coordinates (for mouse clicks)
func (r *OverworldRenderer) GetThreatAtPosition(screenX, screenY int) ecs.EntityID {
	// Convert screen to logical position
	logicalX := (screenX / r.tileSize) + r.state.CameraX
	logicalY := (screenY / r.tileSize) + r.state.CameraY

	logicalPos := coords.LogicalPosition{X: logicalX, Y: logicalY}

	// Check if threat exists at this position
	return overworld.GetThreatNodeAt(r.manager, logicalPos)
}

// FormatThreatInfo returns formatted string for threat details
func FormatThreatInfo(threat *ecs.Entity, manager *common.EntityManager) string {
	if threat == nil {
		return "Select a threat to view details"
	}

	data := common.GetComponentType[*overworld.ThreatNodeData](threat, overworld.ThreatNodeComponent)
	pos := common.GetComponentType[*coords.LogicalPosition](threat, common.PositionComponent)

	if data == nil {
		return "Invalid threat"
	}

	threatTypeName := data.ThreatType.String()
	containedStatus := ""
	if data.IsContained {
		containedStatus = " (CONTAINED)"
	}

	return fmt.Sprintf(
		"=== Threat Details ===\n"+
			"Type: %s%s\n"+
			"Position: (%d, %d)\n"+
			"Intensity: %d / %d\n"+
			"Growth: %.1f%%\n"+
			"Age: %d ticks",
		threatTypeName,
		containedStatus,
		pos.X, pos.Y,
		data.Intensity,
		overworld.GetThreatTypeParams(data.ThreatType).MaxIntensity,
		data.GrowthProgress*100,
		data.SpawnedTick,
	)
}

