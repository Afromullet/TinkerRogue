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
