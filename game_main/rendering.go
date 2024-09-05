package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func ProcessRenderables(g *Game, gameMap GameMap, screen *ebiten.Image) {
	for _, result := range g.World.Query(g.WorldTags["renderables"]) {
		pos := result.Components[PositionComponent].(*Position)
		img := result.Components[RenderableComponent].(*Renderable).Image

		if gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
			index := IndexFromXY(pos.X, pos.Y)
			tile := gameMap.Tiles[index]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
			screen.DrawImage(img, op)
		}
	}
}
