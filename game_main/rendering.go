package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func ProcessRenderables(g *Game, gameMap GameMap, screen *ebiten.Image) {
	for _, result := range g.World.Query(g.WorldTags["renderables"]) {
		pos := result.Components[position].(*Position)
		img := result.Components[renderable].(*Renderable).Image

		visible := result.Components[renderable].(*Renderable).visible

		index := GetIndexFromXY(pos.X, pos.Y)
		tile := gameMap.Tiles[index]
		op := &ebiten.DrawImageOptions{}

		op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))

		if visible {
			screen.DrawImage(img, op)
		}
	}
}
