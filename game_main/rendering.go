package main

import (
	"game_main/common"
	"game_main/graphics"
	"game_main/worldmap"

	"github.com/hajimehoshi/ebiten/v2"
)

func ProcessRenderables(g *Game, gameMap worldmap.GameMap, screen *ebiten.Image) {
	for _, result := range g.World.Query(g.WorldTags["renderables"]) {
		pos := result.Components[common.PositionComponent].(*common.Position)
		img := result.Components[RenderableComponent].(*Renderable).Image

		if gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
			index := graphics.IndexFromXY(pos.X, pos.Y)
			tile := gameMap.Tiles[index]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
			screen.DrawImage(img, op)
		}
	}
}
