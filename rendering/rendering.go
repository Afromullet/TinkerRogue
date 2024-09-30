package rendering

import (
	"game_main/common"
	"game_main/graphics"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

var RenderableComponent *ecs.Component //Putting this here for now rather than in graphics

type Renderable struct {
	Image   *ebiten.Image
	Visible bool
}

// Draw everything with a renderable component that's visible
func ProcessRenderables(ecsmanager *common.EntityManager, gameMap worldmap.GameMap, screen *ebiten.Image, debugMode bool) {
	for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["renderables"]) {
		pos := result.Components[common.PositionComponent].(*common.Position)
		img := result.Components[RenderableComponent].(*Renderable).Image

		if !result.Components[RenderableComponent].(*Renderable).Visible {
			continue
		}

		if debugMode {

			index := graphics.IndexFromXY(pos.X, pos.Y)
			tile := gameMap.Tiles[index]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
			screen.DrawImage(img, op)

		} else if gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
			index := graphics.IndexFromXY(pos.X, pos.Y)
			tile := gameMap.Tiles[index]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
			screen.DrawImage(img, op)
		}

	}
}

func ProcessRenderablesInSquare(ecsmanager *common.EntityManager, gameMap worldmap.GameMap, screen *ebiten.Image, playerPos *common.Position, squareSize int, debugMode bool) {
	// Calculate the starting and ending coordinates of the square
	startX, startY := graphics.SquareStartXY(playerPos.X, playerPos.Y, squareSize)
	endX, endY := graphics.SquareEndXY(playerPos.X, playerPos.Y, squareSize)

	// Calculate center offset to adjust rendering
	centerOffsetX, centerOffsetY := graphics.CenterOffset(playerPos.X, playerPos.Y)

	for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["renderables"]) {
		pos := result.Components[common.PositionComponent].(*common.Position)
		img := result.Components[RenderableComponent].(*Renderable).Image

		if !result.Components[RenderableComponent].(*Renderable).Visible {
			continue
		}

		// Check if the entity's position is within the square bounds
		if pos.X >= startX && pos.X <= endX && pos.Y >= startY && pos.Y <= endY {
			// Calculate the tile's pixel position, adjusted by the center offset
			tilePixelX := gameMap.Tiles[graphics.IndexFromXY(pos.X, pos.Y)].PixelX
			tilePixelY := gameMap.Tiles[graphics.IndexFromXY(pos.X, pos.Y)].PixelY

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(tilePixelX+centerOffsetX), float64(tilePixelY+centerOffsetY))

			if debugMode {
				// In debug mode, we can draw the image directly without visibility checks
				screen.DrawImage(img, op)
			} else if gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
				// Only draw if the tile is visible to the player
				screen.DrawImage(img, op)
			}
		}
	}
}
