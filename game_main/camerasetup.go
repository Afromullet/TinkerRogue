package main

import (
	"game_main/common"
	"game_main/graphics"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/math/f64"
)

var world = ebiten.NewImage(4200, 2560)

func SetupCamera(g *Game) {

	widthX := graphics.LevelWidth
	widthY := graphics.LevelHeight

	// Apply zoom factor
	g.camera.ZoomFactor = 1
	zoom := math.Pow(1.01, float64(g.camera.ZoomFactor))

	// Adjust ViewPort based on zoom
	g.camera.ViewPort = f64.Vec2{
		float64(widthX) / zoom, // Divide both width and height by zoom
		float64(widthY) / zoom,
	}

	// Get the player's position in pixel coordinates
	//centerX, centerY := common.PixelsFromPosition(g.playerData.Pos, gd.TileWidth, gd.TileWidth)
	//	gd := graphics.NewScreenData()
	// Center the camera on the player
	//centeredX := float64(centerX) - (g.camera.ViewPort[0] * zoom / 2)
	//	centeredY := float64(centerY) - (g.camera.ViewPort[1] * zoom / 2)

	// Update camera position to center on the player
	//g.camera.Position = f64.Vec2{centeredX, centeredY}
	g.camera.Position = f64.Vec2{0, 0}
}

func UpdateCameraPosition(g *Game) {

	gd := graphics.NewScreenData()
	centerX, centerY := common.PixelsFromPosition(g.playerData.Pos, gd.TileWidth, gd.TileWidth)
	zoom := math.Pow(1.01, float64(g.camera.ZoomFactor))

	// Center the camera on the player
	centeredX := float64(centerX) - (g.camera.ViewPort[0] * zoom / 2)
	centeredY := float64(centerY) - (g.camera.ViewPort[1] * zoom / 2)

	// Update camera position to center on the player
	g.camera.Position = f64.Vec2{centeredX, centeredY}
	g.camera.Position = clampCameraPosition(centeredX, centeredY, g.camera.ViewPort, float64(gd.DungeonWidth*gd.TileWidth), float64(gd.DungeonHeight*gd.TileHeight))

}

// clampCameraPosition ensures that the camera's position does not go beyond the map's boundaries.
func clampCameraPosition(camX, camY float64, viewPort f64.Vec2, mapWidth, mapHeight float64) f64.Vec2 {
	clampedX := math.Max(0, math.Min(camX, mapWidth-viewPort[0]))
	clampedY := math.Max(0, math.Min(camY, mapHeight-viewPort[1]))
	return f64.Vec2{clampedX, clampedY}
}
