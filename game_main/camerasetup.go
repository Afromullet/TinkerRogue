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

	//g.camera.ZoomFactor = 10
	//zoom := math.Pow(1.01, float64(g.camera.ZoomFactor))
	zoom := 1.5
	g.camera.ZoomLevel = zoom

	// Adjust ViewPort based on zoom
	g.camera.ViewPort = f64.Vec2{
		float64(widthX) / zoom, // Divide both width and height by zoom
		float64(widthY) / zoom,
	}
	graphics.MainCamera = &g.camera
	// Get the player's position in pixel coordinates

	//UpdateCameraPosition(g)

}

func UpdateCameraPosition(g *Game) {

	//gd := graphics.NewScreenData()

	//Center the camera on the player

	centerX, centerY := common.PixelsFromPosition(g.playerData.Pos, graphics.ScreenInfo.TileWidth, graphics.ScreenInfo.TileWidth)
	centeredX := float64(centerX) - (g.camera.ViewPort[0] / 2)
	centeredY := float64(centerY) - (g.camera.ViewPort[1] / 2)
	g.camera.Position = f64.Vec2{centeredX, centeredY}

}

// clampCameraPosition ensures that the camera's position does not go beyond the map's boundaries.
func clampCameraPosition(camX, camY float64, viewPort f64.Vec2, mapWidth, mapHeight float64) f64.Vec2 {
	clampedX := math.Max(0, math.Min(camX, mapWidth-viewPort[0]))
	clampedY := math.Max(0, math.Min(camY, mapHeight-viewPort[1]))
	return f64.Vec2{clampedX, clampedY}
}
