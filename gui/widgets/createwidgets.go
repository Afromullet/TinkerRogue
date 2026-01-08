package widgets

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
)

// SetContainerLocation sets the absolute position of a container
func SetContainerLocation(w *widget.Container, x, y int) {
	r := image.Rect(0, 0, 0, 0)
	r = r.Add(image.Point{x, y})
	w.SetLocation(r)
}

// StringDisplay is an interface for types that can display themselves as strings
type StringDisplay interface {
	DisplayString()
}
