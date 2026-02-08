package guinodeplacement

import (
	"image/color"

	"game_main/overworld/core"
	"game_main/overworld/playernode"

	"github.com/hajimehoshi/ebiten/v2"
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
}
