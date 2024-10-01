package graphics

import (
	"github.com/hajimehoshi/ebiten/v2"
)

var LevelHeight int = 0
var LevelWidth int = 0
var StatsUIOffset int = 1000 //Offset to where the UI starts

var ScreenInfo = NewScreenData()

// Contains the data we need to render the map. Also used for coordinate conversions
type ScreenData struct {
	ScreenWidth  int
	ScreenHeight int

	TileWidth  int
	TileHeight int

	DungeonWidth  int
	DungeonHeight int
	ScaleX        float64
	ScaleY        float64
}

// Todo, need to do less work inside of here because this is called every time we do coordinate conversions
// All of this needs to only be calculated once. Currently, everything that needs Screen Information calls
// NewScreenData, which calculates things again. Those values don't change, so it only needs to be calcualted once
func NewScreenData() ScreenData {
	g := ScreenData{
		DungeonWidth:  100,
		DungeonHeight: 80,
	}
	tileWidthPixels := 32
	tileHeightPixels := 32

	// Calculate the scale based on the dungeon dimensions

	g.ScaleX = float64(g.DungeonWidth) * float64(tileWidthPixels) / float64(g.DungeonWidth)
	g.ScaleY = float64(g.DungeonHeight) * float64(tileHeightPixels) / float64(g.DungeonHeight)

	g.TileWidth = int(g.ScaleX)
	g.TileHeight = int(g.ScaleY)

	LevelHeight = int(float64(g.DungeonHeight) * g.ScaleY)
	LevelWidth = int(float64(g.DungeonWidth) * g.ScaleX)

	return g
}

func (s ScreenData) GetCanvasWidth() int {

	return int((float64(s.TileWidth * s.DungeonWidth)))

}

func (s ScreenData) GetCanvasHeight() int {

	return int((float64(s.TileHeight * s.DungeonHeight)))

}

// IndexFromXY gets the index of the map array from a given X,Y TILE coordinate.
// This coordinate is logical tiles, not pixels.
func IndexFromLogicalXY(x int, y int) int {

	return (y * ScreenInfo.DungeonWidth) + x
}

// Gets XY coordinates from the map tile index
func XYFromIndex(i int) (int, int) {

	return i % ScreenInfo.DungeonWidth, i / ScreenInfo.DungeonWidth
}

func PixelsFromLogicalXY(x, y, tileWidth, tileHeight int) (int, int) {
	return x * tileWidth, y * tileHeight

}

// Gets the pixels from the index
func PixelsFromIndex(i int) (int, int) {

	x, y := i%ScreenInfo.DungeonWidth, i/ScreenInfo.DungeonWidth
	return x * ScreenInfo.TileWidth, y * ScreenInfo.TileHeight
}

// Return the Grid X,Y coordinates from pixel positions
func LogicalXYFromPixels(x, y int) (int, int) {

	return x / ScreenInfo.TileWidth, y / ScreenInfo.TileHeight

}

// Helper function to convert screen coordinates to world coordinates
func ScreenToWorldCoordinates(screenX, screenY, posX, posY int, scaleFactor float64) (int, int) {
	screenWidth, screenHeight := ebiten.WindowSize()
	scaledTileSize := float64(ScreenInfo.TileWidth) * scaleFactor

	worldX := int(float64(screenX-screenWidth/2)/scaledTileSize) + posX
	worldY := int(float64(screenY-screenHeight/2)/scaledTileSize) + posY

	return worldX, worldY
}

// Helper function to convert screen coordinates to world coordinates
func ScreenToWorldCoordinates2(screenX, screenY, posX, posY int) (int, int) {
	screenWidth, screenHeight := ebiten.WindowSize()

	// Calculate the scaled tile size
	scaledTileSize := float64(ScreenInfo.TileWidth) * float64(ScaleFactor)

	// Calculate the position to center the scaled map
	scaledCenterOffsetX := float64(screenWidth)/2 - float64(posX)*scaledTileSize
	scaledCenterOffsetY := float64(screenHeight)/2 - float64(posY)*scaledTileSize

	// Calculate world coordinates
	worldX := int((float64(screenX) - scaledCenterOffsetX) / (float64(ScaleFactor) * float64(ScreenInfo.TileWidth)))
	worldY := int((float64(screenY) - scaledCenterOffsetY) / (float64(ScaleFactor) * float64(ScreenInfo.TileHeight)))

	return worldX, worldY
}
