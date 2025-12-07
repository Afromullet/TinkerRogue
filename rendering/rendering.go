// Package rendering handles the display and drawing of game entities to the screen.
// It processes renderable components, manages sprite drawing, and coordinates
// with the graphics package to render entities at their correct screen positions.
package rendering

import (
	"game_main/common"
	"game_main/coords"
	"game_main/graphics"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

var (
	RenderableComponent *ecs.Component //Putting this here for now rather than in graphics
	RenderablesTag      ecs.Tag        // Tag for querying renderable entities
	MessengersTag       ecs.Tag        // Tag for querying messenger (UI message) entities

	EnableFieldOfView = false
)

type Renderable struct {
	Image   *ebiten.Image
	Visible bool
}

// Draw everything with a renderable component that's visible
func ProcessRenderables(ecsmanager *common.EntityManager, gameMap worldmap.GameMap, screen *ebiten.Image, debugMode bool) {
	for _, result := range ecsmanager.World.Query(RenderablesTag) {
		pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
		renderable := common.GetComponentType[*Renderable](result.Entity, RenderableComponent)
		img := renderable.Image

		if !renderable.Visible {
			continue
		}

		if debugMode || !EnableFieldOfView {

			logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)
			tile := gameMap.Tiles[index]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
			screen.DrawImage(img, op)

		} else if gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
			logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)
			tile := gameMap.Tiles[index]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
			screen.DrawImage(img, op)
		}

	}
}

func ProcessRenderablesInSquare(ecsmanager *common.EntityManager, gameMap worldmap.GameMap, screen *ebiten.Image, playerPos *coords.LogicalPosition, squareSize int, debugMode bool) {
	// Calculate the starting and ending coordinates of the square
	sq := coords.NewDrawableSection(playerPos.X, playerPos.Y, squareSize)

	for _, result := range ecsmanager.World.Query(RenderablesTag) {
		pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
		renderable := common.GetComponentType[*Renderable](result.Entity, RenderableComponent)
		img := renderable.Image

		if !renderable.Visible {
			continue
		}

		// Check if the entity's position is within the square bounds
		if pos.X >= sq.StartX && pos.X <= sq.EndX && pos.Y >= sq.StartY && pos.Y <= sq.EndY {
			logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}

			op := &ebiten.DrawImageOptions{}

			// Apply scaling first (still needed for sprite scaling)
			op.GeoM.Scale(float64(graphics.ScreenInfo.ScaleFactor), float64(graphics.ScreenInfo.ScaleFactor))

			// Use unified coordinate transformation - handles scrolling mode and viewport centering automatically
			screenX, screenY := coords.CoordManager.LogicalToScreen(logicalPos, playerPos)
			op.GeoM.Translate(screenX, screenY)

			if debugMode || !EnableFieldOfView {
				// In debug mode, we can draw the image directly without visibility checks
				screen.DrawImage(img, op)
			} else if gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
				// Only draw if the tile is visible to the player
				screen.DrawImage(img, op)
			}
		}
	}
}
