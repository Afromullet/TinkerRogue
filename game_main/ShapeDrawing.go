package main

// Currently a duplicate of the one found in GameMap. Don't want to pass the GameMap parameter to the shapes here
func InBounds(x, y int) bool {
	gd := NewScreenData()
	if x < 0 || x > gd.ScreenWidth || y < 0 || y > levelHeight {
		return false
	}
	return true
}

// Interfaces shapes which are drawn over tiles
// Get Indices returns the indices of the tiles we want to draw ove r
type TileBasedShape interface {
	GetIndices() []int
	UpdatePosition(pixelX int, pixelY int)
}

// Draws a tile Map based square of specified size at the pixel position.
type TileSquare struct {
	pixelX int
	pixelY int
	size   int
}

func (s TileSquare) GetIndices() []int {
	gd := NewScreenData()
	halfSize := s.size / 2
	indices := make([]int, 0)

	s.pixelX = s.pixelX / gd.TileWidth
	s.pixelY = s.pixelY / gd.TileHeight

	for y := s.pixelY - halfSize; y <= s.pixelY+halfSize; y++ {
		for x := s.pixelX - halfSize; x <= s.pixelX+halfSize; x++ {
			if InBounds(x, y) {
				index := GetIndexFromXY(x, y)
				indices = append(indices, index)
			}
		}
	}

	return indices

}

func (s *TileSquare) UpdatePosition(pixelX int, pixelY int) {
	s.pixelX = pixelX
	s.pixelY = pixelY

}

func NewTileSquare(pixelX, pixelY, size int) TileSquare {

	return TileSquare{
		pixelX: pixelX,
		pixelY: pixelY,
		size:   size,
	}

}

type ShapeDirection int

const (
	LineUp = iota
	LineDown
	LineRight
	LineLeft
)

type TileLine struct {
	pixelX    int
	pixelY    int
	length    int
	direction ShapeDirection
}

func (l TileLine) GetIndices() []int {
	indices := make([]int, 0)
	gd := NewScreenData()

	// Convert pixel coordinates to grid coordinates
	gridX := l.pixelX / gd.TileWidth
	gridY := l.pixelY / gd.TileHeight

	// Add indices based on the direction of the line
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

		// Ensure the calculated coordinates are within bounds
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

	// Convert pixel coordinates to grid coordinates
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
