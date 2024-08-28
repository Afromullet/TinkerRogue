package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var ShapesToDraw []RenderableShape

type RenderableShape interface {
	DrawShape(screen *ebiten.Image)
}

func AddShapeToDraw(s RenderableShape) {
	ShapesToDraw = append(ShapesToDraw, s)

}

type RenderableLine struct {
	x0          float32
	y0          float32
	x1          float32
	y1          float32
	strokeWidth float32
	lineColor   color.Color
}

func NewRenderableLine(x0, y0, x1, y1, strokeWidth float32, lineColor color.Color) RenderableLine {
	return RenderableLine{
		x0:          x0,
		y0:          y0,
		x1:          x1,
		y1:          y1,
		strokeWidth: strokeWidth,
		lineColor:   lineColor,
	}
}

func (l RenderableLine) DrawShape(screen *ebiten.Image) {

	vector.StrokeLine(screen, l.x0, l.y0, l.x1, l.y1, l.strokeWidth, l.lineColor, false)

}

type RenderableRect struct {
	x           float32
	y           float32
	width       float32
	height      float32
	strokeWidth float32
	lineColor   color.Color
}

func MewRenderableRect(x, y, width, height, strokeWidth float32, lineColor color.Color) RenderableRect {
	return RenderableRect{
		x:           x,
		y:           y,
		width:       width,
		height:      height,
		strokeWidth: strokeWidth,
		lineColor:   lineColor,
	}
}

func (r RenderableRect) DrawShape(screen *ebiten.Image) {

	vector.StrokeRect(screen, r.x, r.y, r.width, r.height, r.strokeWidth, r.lineColor, false)

}

// Currently a duplicate of the one found in GameMap. Don't want to pass the GameMap parameter to the shapes here
func InBounds(x, y int) bool {
	gd := NewScreenData()
	if x < 0 || x > gd.ScreenWidth || y < 0 || y > levelHeight {
		return false
	}
	return true
}

var TileShapesToDraw []TileBasedShape

func AddTileShapeToDraw(s TileBasedShape) {
	TileShapesToDraw = append(TileShapesToDraw, s)

}

// For now it's just revealing the tile. Makes it easier to test
func DrawTileShapes(gameMap *GameMap, screen *ebiten.Image) {

	for _, s := range TileShapesToDraw {

		indices := s.GetIndices()

		for i := 0; i < len(indices); i++ {

			//gameMap.Tiles[indices[i]].IsRevealed = true

		}

	}

}

// Backup of the original
func DrawTileShapes2(gameMap *GameMap, screen *ebiten.Image) {

	for _, s := range TileShapesToDraw {

		indices := s.GetIndices()

		for i := 0; i < len(indices); i++ {

			pixelX := gameMap.Tiles[indices[i]].PixelX
			pixelY := gameMap.Tiles[indices[i]].PixelY

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(pixelX), float64(pixelY))
			op.ColorM.Translate(100, 100, 100, 0.35)
			//op.ColorScale(100, 100, 100, 0.35)
			//screen.DrawImage(gameMap.Tiles[indices[i]].Image, op)

			gameMap.Tiles[indices[i]].IsRevealed = true

		}

	}

}

// Interfaces shapes which are drawn over tiles
// Get Indices returns the indices of the tiles we want to draw ove r
type TileBasedShape interface {
	GetIndices() []int
}

// Draws a tile Map based square of specified size at the pixel position.
type SquareAtPixel struct {
	pixelX int
	pixelY int
	size   int
}

func (s SquareAtPixel) GetIndices() []int {
	gd := NewScreenData()
	halfSize := s.size / 2
	squareIndices := make([]int, 0)

	s.pixelX = s.pixelX / gd.TileWidth
	s.pixelY = s.pixelY / gd.TileHeight

	for y := s.pixelY - halfSize; y <= s.pixelY+halfSize; y++ {
		for x := s.pixelX - halfSize; x <= s.pixelX+halfSize; x++ {
			if InBounds(x, y) {
				index := GetIndexFromXY(x, y)
				squareIndices = append(squareIndices, index)
			}
		}
	}

	return squareIndices

}

func NewSquareAtPixel(pixelX, pixelY, size int) SquareAtPixel {

	return SquareAtPixel{
		pixelX: pixelX,
		pixelY: pixelY,
		size:   size,
	}

}

// Draws a tile Map based square of specified size at the pixel position.
// Requires the game map since we're using A-Star to draw the line
type LineToPixel struct {
	startX  int
	startY  int
	endX    int
	endY    int
	gameMap *GameMap
}

func NewLineToPixel(startX, startY, endX, endY int, gameMap *GameMap) LineToPixel {

	return LineToPixel{
		startX:  startX,
		startY:  startY,
		endX:    endX,
		endY:    endY,
		gameMap: gameMap,
	}

}

func (l LineToPixel) GetIndices() []int {

	startPos := GetPositionFromPixels(l.startX, l.startY)
	endPos := GetPositionFromPixels(l.endX, l.endY)

	indices := make([]int, 0)
	astar := AStar{}.GetPath(*l.gameMap, &startPos, &endPos, true)

	for _, p := range astar {
		indices = append(indices, GetIndexFromXY(p.X, p.Y))
	}

	return indices

}
