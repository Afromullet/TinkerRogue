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

			index := graphics.CoordTransformer.IndexFromLogicalXY(pos.X, pos.Y)
			tile := gameMap.Tiles[index]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
			screen.DrawImage(img, op)

		} else if gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
			index := graphics.CoordTransformer.IndexFromLogicalXY(pos.X, pos.Y)
			tile := gameMap.Tiles[index]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
			screen.DrawImage(img, op)
		}

	}
}

func ProcessRenderablesInSquare(ecsmanager *common.EntityManager, gameMap worldmap.GameMap, screen *ebiten.Image, playerPos *common.Position, squareSize int, debugMode bool) {
	// Calculate the starting and ending coordinates of the square

	sq := graphics.NewDrawableSection(playerPos.X, playerPos.Y, squareSize)

	// Get the dimensions of the screen
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Calculate the scaled tile size
	scaledTileSize := graphics.ScreenInfo.TileSize * graphics.ScreenInfo.ScaleFactor

	// Calculate the position to center the scaled map
	scaledCenterOffsetX := float64(screenWidth)/2 - float64(playerPos.X*scaledTileSize)
	scaledCenterOffsetY := float64(screenHeight)/2 - float64(playerPos.Y*scaledTileSize)

	for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["renderables"]) {
		pos := result.Components[common.PositionComponent].(*common.Position)
		img := result.Components[RenderableComponent].(*Renderable).Image

		if !result.Components[RenderableComponent].(*Renderable).Visible {
			continue
		}

		// Check if the entity's position is within the square bounds
		if pos.X >= sq.StartX && pos.X <= sq.EndX && pos.Y >= sq.StartY && pos.Y <= sq.EndY {
			// Calculate the tile's pixel position

			ind := graphics.CoordTransformer.IndexFromLogicalXY(pos.X, pos.Y)
			tilePixelX := gameMap.Tiles[ind].PixelX
			tilePixelY := gameMap.Tiles[ind].PixelY

			op := &ebiten.DrawImageOptions{}

			// Apply scaling first

			op.GeoM.Scale(float64(graphics.ScreenInfo.ScaleFactor), float64(graphics.ScreenInfo.ScaleFactor))

			// Translate the scaled position
			op.GeoM.Translate(
				float64(tilePixelX)*float64(graphics.ScreenInfo.ScaleFactor)+scaledCenterOffsetX,
				float64(tilePixelY)*float64(graphics.ScreenInfo.ScaleFactor)+scaledCenterOffsetY,
			)

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
