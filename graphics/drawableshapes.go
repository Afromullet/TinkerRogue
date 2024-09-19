package graphics

//...There has to be a way to make this less repetitive.

/*
Lots of steps in adding a new shape. It's clunky, but there shouldn't be too many more shapes.

To add a new shape:

1) Implement the TileBasedShape interface. GetIndices returns the indices of the TileMap.
UpdatePosition updates the shape with the new X and Y
UpdateShape updates the parametes with the params a shape needs.

2) ShapeUpdater also requires an update...

Add the type to ExtractShapeParams to get a ShapeUpdater.
If the shape uses a direction, add an additional case to HasDirection




*/

// This type helps us identify which direction to draw some shapes in
// lines and cones have a direction. Cones look weird since we're basic the coordinates of X,Y tiles
// Not worrying about that for now
type ShapeDirection int

const (
	LineUp = iota
	LineDown
	LineRight
	LineLeft
	LineDiagonalUpRight
	LineDiagonalDownRight
	LineDiagonalUpLeft
	LinedDiagonalDownLeft
	NoDirection
)

// Used for rotating directions. Does not contain NoDirection since that's not used for rotating
// The other matters here since we'll be incrementing through this slice to change direction
var AllDirections = []ShapeDirection{
	LineUp,
	LineDiagonalUpRight,
	LineRight,
	LineDiagonalDownRight,
	LineDown,
	LinedDiagonalDownLeft,
	LineLeft,
	LineDiagonalUpLeft,
}

func RotateRight(dir ShapeDirection) ShapeDirection {

	newDir := 0
	for i, direction := range AllDirections {

		if direction == dir {
			newDir = i + 1
			//Wrap around
			if newDir >= len(AllDirections) {
				newDir = 0
			}
			break

		}
	}

	return AllDirections[newDir]
}

func RotateLeft(dir ShapeDirection) ShapeDirection {

	newDir := 0
	for i, direction := range AllDirections {

		if direction == dir {
			newDir = i - 1
			//Wrap around
			if newDir < 0 {
				newDir = len(AllDirections) - 1
			}
			break

		}
	}

	return AllDirections[newDir]
}

// Currently a duplicate of the one found in GameMap. Don't want to pass the GameMap parameter to the shapes here
func InBounds(x, y int) bool {
	gd := NewScreenData()
	if x < 0 || x > gd.ScreenWidth || y < 0 || y > LevelHeight {
		return false
	}
	return true
}

// Interfaces for a Type that draws a shape on the map
// GetIndices returns the indices the shape covers.
// UpdatePositions updates the starting position of the shape.
// GetStartPosition returns the Position of the coordinates the drawing starts at
// Each shape will have to determine how to draw the shape using indices
// This does not handle the ColorMatrix..DrawLevel currently applies teh ColorMatrix
type TileBasedShape interface {
	GetIndices() []int
	UpdatePosition(pixelX int, pixelY int)
	UpdateShape(updater ShapeUpdater)
	StartPosition() (int, int)
}

// The square is drawn around (PixelX,PixelY)
type TileSquare struct {
	PixelX int
	PixelY int
	Size   int
}

func (s *TileSquare) GetIndices() []int {

	halfSize := s.Size / 2
	indices := make([]int, 0)

	gridX, gridY := XYFromPixels(s.PixelX, s.PixelY)

	for y := gridY - halfSize; y <= gridY+halfSize; y++ {
		for x := gridX - halfSize; x <= gridX+halfSize; x++ {
			if InBounds(x, y) {
				index := IndexFromXY(x, y)
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

func (s *TileSquare) UpdateShape(updater ShapeUpdater) {

	s.PixelX = updater.PixelX
	s.PixelY = updater.PixelY
	s.Size = updater.Size

}

func (s *TileSquare) StartPosition() (int, int) {

	return s.PixelX, s.PixelY

}

func NewTileSquare(pixelX, pixelY, size int) *TileSquare {

	return &TileSquare{
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

func (l *TileLine) GetIndices() []int {
	indices := make([]int, 0)

	gridX, gridY := XYFromPixels(l.pixelX, l.pixelY)

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
		case LineDiagonalUpRight:
			x, y = gridX+i, gridY-i
		case LineDiagonalDownRight:
			x, y = gridX+i, gridY+i
		case LineDiagonalUpLeft:
			x, y = gridX-i, gridY-i
		case LinedDiagonalDownLeft:
			x, y = gridX-i, gridY+i

		}

		if InBounds(x, y) {
			index := IndexFromXY(x, y)
			indices = append(indices, index)
		}
	}

	return indices
}

func (l *TileLine) UpdatePosition(pixelX, pixelY int) {
	l.pixelX = pixelX
	l.pixelY = pixelY

}

func (l *TileLine) UpdateShape(updater ShapeUpdater) {

	l.pixelX = updater.PixelX
	l.pixelY = updater.PixelY
	l.direction = updater.Direction

}

func (l *TileLine) StartPosition() (int, int) {

	return l.pixelX, l.pixelY

}

func NewTileLine(pixelX, pixelY, length int, direction ShapeDirection) *TileLine {

	return &TileLine{
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

func (c *TileCone) GetIndices() []int {
	indices := make([]int, 0)

	gridX, gridY := XYFromPixels(c.pixelX, c.pixelY)

	// Loop through each step of the cone's length
	for i := 0; i < c.length; i++ {
		switch c.direction {
		case LineUp:
			for j := -i; j <= i; j++ { // Widening cone
				x, y := gridX+j, gridY-i
				if InBounds(x, y) {
					index := IndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		case LineDown:
			for j := -i; j <= i; j++ {
				x, y := gridX+j, gridY+i
				if InBounds(x, y) {
					index := IndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		case LineRight:
			for j := -i; j <= i; j++ {
				x, y := gridX+i, gridY+j
				if InBounds(x, y) {
					index := IndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		case LineLeft:
			for j := -i; j <= i; j++ {
				x, y := gridX-i, gridY+j
				if InBounds(x, y) {
					index := IndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		case LineDiagonalUpRight:
			for j := -i; j <= i; j++ {
				x, y := gridX+i, gridY-i+j // Move diagonally up-right
				if InBounds(x, y) {
					index := IndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		case LineDiagonalDownRight:
			for j := -i; j <= i; j++ {
				x, y := gridX+i, gridY+i-j // Move diagonally down-right
				if InBounds(x, y) {
					index := IndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		case LineDiagonalUpLeft:
			for j := -i; j <= i; j++ {
				x, y := gridX-i, gridY-i+j // Move diagonally up-left
				if InBounds(x, y) {
					index := IndexFromXY(x, y)
					indices = append(indices, index)
				}
			}
		case LinedDiagonalDownLeft:
			for j := -i; j <= i; j++ {
				x, y := gridX-i, gridY+i-j // Move diagonally down-left
				if InBounds(x, y) {
					index := IndexFromXY(x, y)
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

func (c *TileCone) UpdateShape(updater ShapeUpdater) {

	c.pixelX = updater.PixelX
	c.pixelY = updater.PixelY
	c.length = updater.Length
	c.direction = updater.Direction

}

func (c *TileCone) StartPosition() (int, int) {

	return c.pixelX, c.pixelY

}

func NewTileCone(pixelX, pixelY, length int, direction ShapeDirection) *TileCone {

	return &TileCone{
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

func (c *TileCircle) GetIndices() []int {
	indices := make([]int, 0)

	centerX, centerY := XYFromPixels(c.pixelX, c.pixelY)

	x := 0
	y := c.radius
	d := 1 - c.radius

	// Helper function to add points in all octants of the circle and fill horizontally between them
	addAndFillCirclePoints := func(x, y int) {
		// Add points in all octants
		points := []struct{ X, Y int }{
			{centerX + x, centerY + y},
			{centerX - x, centerY + y},
			{centerX + x, centerY - y},
			{centerX - x, centerY - y},
			{centerX + y, centerY + x},
			{centerX - y, centerY + x},
			{centerX + y, centerY - x},
			{centerX - y, centerY - x},
		}
		for _, p := range points {
			if InBounds(p.X, p.Y) {
				index := IndexFromXY(p.X, p.Y)
				indices = append(indices, index)
			}
		}

		// Fill horizontal lines between the points
		// Fill between (centerX - x, centerY + y) and (centerX + x, centerY + y)
		for fillX := centerX - x; fillX <= centerX+x; fillX++ {
			if InBounds(fillX, centerY+y) {
				index := IndexFromXY(fillX, centerY+y)
				indices = append(indices, index)
			}
			if InBounds(fillX, centerY-y) {
				index := IndexFromXY(fillX, centerY-y)
				indices = append(indices, index)
			}
		}
		// Fill between (centerX - y, centerY + x) and (centerX + y, centerY + x)
		for fillX := centerX - y; fillX <= centerX+y; fillX++ {
			if InBounds(fillX, centerY+x) {
				index := IndexFromXY(fillX, centerY+x)
				indices = append(indices, index)
			}
			if InBounds(fillX, centerY-x) {
				index := IndexFromXY(fillX, centerY-x)
				indices = append(indices, index)
			}
		}
	}

	// Midpoint circle algorithm to generate the points and fill
	for x <= y {
		addAndFillCirclePoints(x, y)
		if d < 0 {
			d += 2*x + 3
		} else {
			d += 2*(x-y) + 5
			y--
		}
		x++
	}

	return indices
}

func (c *TileCircle) UpdatePosition(pixelX, pixelY int) {
	c.pixelX = pixelX
	c.pixelY = pixelY

}

func (c *TileCircle) UpdateShape(updater ShapeUpdater) {

	c.pixelX = updater.PixelX
	c.pixelY = updater.PixelY
	c.radius = updater.Radius
}

func (c *TileCircle) StartPosition() (int, int) {

	return c.pixelX, c.pixelY
}

func NewTileCircle(pixelX, pixelY, radius int) *TileCircle {

	return &TileCircle{
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

func (r *TileRectangle) GetIndices() []int {
	indices := make([]int, 0)

	// Convert pixel coordinates to grid coordinates (if necessary)

	gridX, gridY := XYFromPixels(r.pixelX, r.pixelY)

	// Iterate through the width and height of the rectangle
	for y := gridY; y < gridY+r.height; y++ {
		for x := gridX; x < gridX+r.width; x++ {
			if InBounds(x, y) {
				index := IndexFromXY(x, y)
				indices = append(indices, index)
			}
		}
	}

	return indices
}

func (r *TileRectangle) UpdatePosition(pixelX, pixelY int) {
	r.pixelX = pixelX
	r.pixelY = pixelY

}

func (r *TileRectangle) UpdateShape(updater ShapeUpdater) {

	r.pixelX = updater.PixelX
	r.pixelY = updater.PixelY
	r.width = updater.Width
	r.height = updater.height

}

func (r *TileRectangle) StartPosition() (int, int) {

	return r.pixelX, r.pixelY

}

func NewTileRectangle(pixelX, pixelY, width, height int) *TileRectangle {

	return &TileRectangle{
		pixelX: pixelX,
		pixelY: pixelY,
		width:  width,
		height: height,
	}

}

type TileCircleOutline struct {
	pixelX int
	pixelY int
	radius int
}

func (c *TileCircleOutline) GetIndices() []int {
	indices := make([]int, 0)

	centerX, centerY := XYFromPixels(c.pixelX, c.pixelY)

	x := 0
	y := c.radius
	d := 1 - c.radius

	// Helper function to add points in all octants of the circle
	addCirclePoints := func(x, y int) {
		points := []struct{ X, Y int }{
			{centerX + x, centerY + y},
			{centerX - x, centerY + y},
			{centerX + x, centerY - y},
			{centerX - x, centerY - y},
			{centerX + y, centerY + x},
			{centerX - y, centerY + x},
			{centerX + y, centerY - x},
			{centerX - y, centerY - x},
		}
		for _, p := range points {
			if InBounds(p.X, p.Y) {
				index := IndexFromXY(p.X, p.Y)
				indices = append(indices, index)
			}
		}
	}

	// Midpoint circle algorithm to generate the points
	for x <= y {
		addCirclePoints(x, y)
		if d < 0 {
			d += 2*x + 3
		} else {
			d += 2*(x-y) + 5
			y--
		}
		x++
	}
	return indices
}

func (c *TileCircleOutline) UpdatePosition(pixelX, pixelY int) {
	c.pixelX = pixelX
	c.pixelY = pixelY

}

func (c *TileCircleOutline) UpdateShape(updater ShapeUpdater) {

	c.pixelX = updater.PixelX
	c.pixelY = updater.PixelY
	c.radius = updater.Radius

}

func NewTileCircleOutline(pixelX, pixelY, radius int) TileCircleOutline {

	return TileCircleOutline{
		pixelX: pixelX,
		pixelY: pixelY,
		radius: radius,
	}

}

// This shape has only the outer edges of a square.
type TileSquareOutline struct {
	PixelX int
	PixelY int
	Size   int
}

func (s *TileSquareOutline) GetIndices() []int {

	halfSize := s.Size / 2
	indices := make([]int, 0)

	gridX, gridY := XYFromPixels(s.PixelX, s.PixelY)

	// Top and bottom edges
	for x := gridX - halfSize; x <= gridX+halfSize; x++ {
		if InBounds(x, gridY-halfSize) {
			index := IndexFromXY(x, gridY-halfSize)
			indices = append(indices, index)
		}
		if InBounds(x, gridY+halfSize) {
			index := IndexFromXY(x, gridY+halfSize)
			indices = append(indices, index)
		}
	}

	// Left and right edges (excluding corners already handled by top/bottom)
	for y := gridY - halfSize + 1; y <= gridY+halfSize-1; y++ {
		if InBounds(gridX-halfSize, y) {
			index := IndexFromXY(gridX-halfSize, y)
			indices = append(indices, index)
		}
		if InBounds(gridX+halfSize, y) {
			index := IndexFromXY(gridX+halfSize, y)
			indices = append(indices, index)
		}
	}

	return indices

}

func (s *TileSquareOutline) UpdatePosition(pixelX int, pixelY int) {
	s.PixelX = pixelX
	s.PixelY = pixelY

}

func (s *TileSquareOutline) UpdateShape(updater ShapeUpdater) {

	s.PixelX = updater.PixelX
	s.PixelY = updater.PixelY
	s.Size = updater.Size

}

func NewTileSquareOutline(pixelX, pixelY, size int) *TileSquareOutline {

	return &TileSquareOutline{
		PixelX: pixelX,
		PixelY: pixelY,
		Size:   size,
	}

}
