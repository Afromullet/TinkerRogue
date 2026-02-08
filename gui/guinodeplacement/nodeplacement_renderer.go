package guinodeplacement

import (
	"fmt"
	"image/color"

	"game_main/gui/guiresources"
	"game_main/overworld/core"
	"game_main/overworld/playernode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// renderPlacementPreview draws a ghost node at the cursor position with validity feedback.
func (npm *NodePlacementMode) renderPlacementPreview(screen *ebiten.Image) {
	if npm.cursorPos == nil || npm.renderer == nil || npm.selectedNodeType == "" {
		return
	}

	// Calculate screen position from cursor logical position
	tileSize := npm.Context.TileSize
	screenX := (npm.cursorPos.X - npm.state.CameraX) * tileSize
	screenY := (npm.cursorPos.Y - npm.state.CameraY) * tileSize

	halfSize := float32(tileSize) / 2
	centerX := float32(screenX) + halfSize
	centerY := float32(screenY) + halfSize

	// Get node color
	previewColor := color.RGBA{R: 100, G: 200, B: 100, A: 120} // semi-transparent green
	nodeDef := core.GetNodeRegistry().GetNodeByID(string(npm.selectedNodeType))
	if nodeDef != nil {
		previewColor = color.RGBA{R: nodeDef.Color.R, G: nodeDef.Color.G, B: nodeDef.Color.B, A: 120}
	}

	// Determine validity tint
	valid := false
	if npm.lastValidation != nil {
		valid = npm.lastValidation.Valid
	} else {
		// Re-validate
		result := playernode.ValidatePlacement(npm.Context.ECSManager, *npm.cursorPos, npm.Context.PlayerData)
		valid = result.Valid
	}

	if !valid {
		// Red tint for invalid placement
		previewColor = color.RGBA{R: 255, G: 50, B: 50, A: 120}
	}

	// Draw ghost square
	vector.DrawFilledRect(screen, centerX-halfSize, centerY-halfSize, halfSize*2, halfSize*2, previewColor, true)

	// Draw border (white for valid, red for invalid)
	borderColor := color.RGBA{R: 255, G: 255, B: 255, A: 200}
	if !valid {
		borderColor = color.RGBA{R: 255, G: 50, B: 50, A: 200}
	}
	vector.StrokeRect(screen, centerX-halfSize, centerY-halfSize, halfSize*2, halfSize*2, 2, borderColor, true)

	// Draw node type label above the cursor preview
	displayName := string(npm.selectedNodeType)
	if nodeDef != nil {
		displayName = nodeDef.DisplayName
	}

	face := guiresources.SmallFace
	bounds := text.BoundString(face, displayName)
	labelX := int(centerX) - bounds.Dx()/2
	labelY := screenY - 8
	text.Draw(screen, displayName, face, labelX, labelY, color.White)
}

// renderSelectionHUD draws a persistent header showing the currently selected node type.
func (npm *NodePlacementMode) renderSelectionHUD(screen *ebiten.Image) {
	if npm.selectedNodeType == "" {
		return
	}

	nodeDef := core.GetNodeRegistry().GetNodeByID(string(npm.selectedNodeType))
	displayName := string(npm.selectedNodeType)
	category := ""
	if nodeDef != nil {
		displayName = nodeDef.DisplayName
		category = string(nodeDef.Category)
	}

	// Find current selection index for display
	selIndex := 0
	for i, node := range npm.nodeTypes {
		if node.ID == string(npm.selectedNodeType) {
			selIndex = i + 1
			break
		}
	}

	maxKey := len(npm.nodeTypes)
	if maxKey > 4 {
		maxKey = 4
	}

	hudText := fmt.Sprintf("Node Placement  |  Selected: %s (%s)  [%d/%d]  |  Tab=cycle  1-%d=select  ESC=cancel",
		displayName, category, selIndex, len(npm.nodeTypes), maxKey)

	face := guiresources.SmallFace
	metrics := face.Metrics()
	barHeight := float32(metrics.Height.Round() + 12)

	// Draw background bar
	vector.DrawFilledRect(screen, 0, 0, float32(screen.Bounds().Dx()), barHeight, color.RGBA{R: 0, G: 0, B: 0, A: 200}, false)

	// Draw node color swatch
	swatchSize := float32(metrics.Height.Round() - 4)
	swatchX := float32(8)
	swatchY := float32(6)
	if nodeDef != nil {
		vector.DrawFilledRect(screen, swatchX, swatchY, swatchSize, swatchSize, nodeDef.Color, false)
		vector.StrokeRect(screen, swatchX, swatchY, swatchSize, swatchSize, 1, color.RGBA{R: 255, G: 255, B: 255, A: 200}, false)
	}

	// Draw text baseline-aligned within the bar
	textX := int(swatchX+swatchSize) + 10
	textY := int(barHeight) - 8
	text.Draw(screen, hudText, face, textX, textY, color.White)
}
