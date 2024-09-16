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

// revealEverything
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
