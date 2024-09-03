package main

// This type helps us identify which direction to draw some shapes in
// I.E, lines and cones go into a direction
type ShapeDirection int

const (
	LineUp = iota
	LineDown
	LineRight
	LineLeft
)

// Currently a duplicate of the one found in GameMap. Don't want to pass the GameMap parameter to the shapes here
func InBounds(x, y int) bool {
	gd := NewScreenData()
	if x < 0 || x > gd.ScreenWidth || y < 0 || y > levelHeight {
		return false
	}
	return true
}

// Interfaces for a Type that draws a shape on the map
// GetIndices returns the indices the shape covers.
// UpdatePositions updates the starting position of the shape.
// Each shape will have to determine how to draw the shape using indices
// This does not handle the ColorMatrix..DrawLevel currently applies teh ColorMatrix
type TileBasedShape interface {
	GetIndices() []int
	UpdatePosition(pixelX int, pixelY int)
}

// A TileBasedShape returns indices that correspond to the elements in the gamemaps Tiles slice
// This returns the X,Y positions since we handle player and creature location through Position
func GetTilePositions(ts TileBasedShape) []Position {

	indices := ts.GetIndices()

	pos := make([]Position, len(indices))

	for i, inds := range indices {

		pos[i] = GetXYFromIndex(inds)

	}

	return pos

}

// The square is drawn around (PixelX,PixelY)
type TileSquare struct {
	PixelX int
	PixelY int
	Size   int
}

func (s TileSquare) GetIndices() []int {
	gd := NewScreenData()
	halfSize := s.Size / 2
	indices := make([]int, 0)

	s.PixelX = s.PixelX / gd.TileWidth
	s.PixelY = s.PixelY / gd.TileHeight

	for y := s.PixelY - halfSize; y <= s.PixelY+halfSize; y++ {
		for x := s.PixelX - halfSize; x <= s.PixelX+halfSize; x++ {
			if InBounds(x, y) {
				index := GetIndexFromXY(x, y)
				indices = append(indices, index)
			}
		}
	}

	return indices

}

func (s *TileSquare) UpdatePosition(pixelX int, pixelY int) {
	s.PixelX = pixelX
	s.PixelY = pixelY

}

func NewTileSquare(pixelX, pixelY, size int) TileSquare {

	return TileSquare{
		PixelX: pixelX,
		PixelY: pixelY,
		Size:   size,
	}

}

type TileLine struct {
	pixelX    int
	pixelY    int
	length    int
	direction ShapeDirection
}

func (l TileLine) GetIndices() []int {
	indices := make([]int, 0)
	gd := NewScreenData()

	gridX := l.pixelX / gd.TileWidth
	gridY := l.pixelY / gd.TileHeight

	for i := 0; i < l.length; i++ {
		var x, y int

		switch l.direction {
		case LineUp:
			x, y = gridX, gridY-i
		case LineDown:
			x, y = gridX, gridY+i
		case LineRight:
			x, y = gridX+i, gridY
		case LineLeft:
			x, y = gridX-i, gridY
		}

		if InBounds(x, y) {
			index := GetIndexFromXY(x, y)
			indices = append(indices, index)
		}
	}

	return indices
}

func (l *TileLine) UpdatePosition(pixelX, pixelY int) {
	l.pixelX = pixelX
	l.pixelY = pixelY

}

func NewTileLine(pixelX, pixelY, length int, direction ShapeDirection) TileLine {

	return TileLine{
		pixelX:    pixelX,
		pixelY:    pixelY,
		length:    length,
		direction: direction,
	}

}

type TileCone struct {
	pixelX    int
	pixelY    int
	length    int
	direction ShapeDirection
}

func (c TileCone) GetIndices() []int {
	indices := make([]int, 0)
	gd := NewScreenData()

	gridX := c.pixelX / gd.TileWidth
	gridY := c.pixelY / gd.TileHeight

	// Loop through each step of the cone's length
	for i := 0; i < c.length; i++ {
		switch c.direction {
		case LineUp:
			for j := -i; j <= i; j++ { // Widening cone
				x, y := gridX+j, gridY-i
				if InBounds(x, y) {
					index := GetIndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		case LineDown:
			for j := -i; j <= i; j++ {
				x, y := gridX+j, gridY+i
				if InBounds(x, y) {
					index := GetIndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		case LineRight:
			for j := -i; j <= i; j++ {
				x, y := gridX+i, gridY+j
				if InBounds(x, y) {
					index := GetIndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		case LineLeft:
			for j := -i; j <= i; j++ {
				x, y := gridX-i, gridY+j
				if InBounds(x, y) {
					index := GetIndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		}
	}

	return indices
}

func (c *TileCone) UpdatePosition(pixelX, pixelY int) {
	c.pixelX = pixelX
	c.pixelY = pixelY

}

func NewTileCone(pixelX, pixelY, length int, direction ShapeDirection) TileCone {

	return TileCone{
		pixelX:    pixelX,
		pixelY:    pixelY,
		length:    length,
		direction: direction,
	}

}

type TileCircle struct {
	pixelX int
	pixelY int
	radius int
}

func (c TileCircle) GetIndices() []int {
	indices := make([]int, 0)
	gd := NewScreenData()

	centerX := c.pixelX / gd.TileWidth
	centerY := c.pixelY / gd.TileHeight

	for y := centerY - c.radius; y <= centerY+c.radius; y++ {
		for x := centerX - c.radius; x <= centerX+c.radius; x++ {
			// Check if the point (x, y) is within the circle
			if (x-centerX)*(x-centerX)+(y-centerY)*(y-centerY) <= c.radius*c.radius {
				if InBounds(x, y) {
					index := GetIndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		}
	}

	return indices
}

func (c *TileCircle) UpdatePosition(pixelX, pixelY int) {
	c.pixelX = pixelX
	c.pixelY = pixelY

}

func NewTileCircle(pixelX, pixelY, radius int) TileCircle {

	return TileCircle{
		pixelX: pixelX,
		pixelY: pixelY,
		radius: radius,
	}

}

type TileRectangle struct {
	pixelX int
	pixelY int
	width  int
	height int
}

func (r TileRectangle) GetIndices() []int {
	indices := make([]int, 0)

	// Convert pixel coordinates to grid coordinates (if necessary)
	gd := NewScreenData()
	pixelX := r.pixelX / gd.TileWidth
	pixelY := r.pixelY / gd.TileHeight

	// Iterate through the width and height of the rectangle
	for y := pixelY; y < pixelY+r.height; y++ {
		for x := pixelX; x < pixelX+r.width; x++ {
			if InBounds(x, y) {
				index := GetIndexFromXY(x, y)
				indices = append(indices, index)
			}
		}
	}

	return indices
}

func (c *TileRectangle) UpdatePosition(pixelX, pixelY int) {
	c.pixelX = pixelX
	c.pixelY = pixelY

}

func NewTileRectangle(pixelX, pixelY, width, height int) TileRectangle {

	return TileRectangle{
		pixelX: pixelX,
		pixelY: pixelY,
		width:  width,
		height: height,
	}

}
