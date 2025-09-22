package graphics

import (
	"game_main/common"
	"math/rand"
)

// ============================================================================
// SHAPE DIRECTION SYSTEM (preserved from original)
// ============================================================================

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
	for i, direction := range AllDirections {
		if direction == dir {
			newDir := i + 1
			if newDir >= len(AllDirections) {
				newDir = 0
			}
			return AllDirections[newDir]
		}
	}
	return dir
}

func RotateLeft(dir ShapeDirection) ShapeDirection {
	for i, direction := range AllDirections {
		if direction == dir {
			newDir := i - 1
			if newDir < 0 {
				newDir = len(AllDirections) - 1
			}
			return AllDirections[newDir]
		}
	}
	return dir
}

// ============================================================================
// NEW SIMPLIFIED SHAPE SYSTEM
// ============================================================================

type BasicShapeType int

const (
	Circular BasicShapeType = iota    // radius-based (Circle types)
	Rectangular                       // width/height-based (Square/Rectangle types)
	Linear                           // length/direction-based (Line/Cone types)
)

type BaseShape struct {
	Position   PixelPosition
	Type       BasicShapeType
	Size       int              // Primary dimension (radius, length, or width)
	Width      int              // For rectangles only
	Height     int              // For rectangles only
	Direction  *ShapeDirection  // nil for non-directional shapes
	Quality    common.QualityType
}

// TileBasedShape interface - maintains compatibility with existing code
type TileBasedShape interface {
	GetIndices() []int
	UpdatePosition(pixelX, pixelY int)
	StartPositionPixels() (int, int)
	GetDirection() ShapeDirection
	CanRotate() bool
}

// ============================================================================
// INTERFACE IMPLEMENTATION
// ============================================================================

func (s *BaseShape) GetIndices() []int {
	logical := CoordManager.PixelToLogical(s.Position)

	switch s.Type {
	case Circular:
		return s.calculateCircle(logical.X, logical.Y)
	case Rectangular:
		return s.calculateRectangle(logical.X, logical.Y)
	case Linear:
		return s.calculateLine(logical.X, logical.Y)
	}
	return nil
}

func (s *BaseShape) UpdatePosition(pixelX, pixelY int) {
	s.Position = PixelPosition{X: pixelX, Y: pixelY}
}

func (s *BaseShape) StartPositionPixels() (int, int) {
	return s.Position.X, s.Position.Y
}

func (s *BaseShape) GetDirection() ShapeDirection {
	if s.Direction != nil {
		return *s.Direction
	}
	return NoDirection
}

func (s *BaseShape) CanRotate() bool {
	return s.Direction != nil
}

// ============================================================================
// FACTORY FUNCTIONS WITH INTEGRATED QUALITY
// ============================================================================

func NewCircle(pixelX, pixelY int, quality common.QualityType) *BaseShape {
	var radius int
	switch quality {
	case common.LowQuality:
		radius = rand.Intn(3)      // 0-2 (matches current system)
	case common.NormalQuality:
		radius = rand.Intn(4)      // 0-3
	case common.HighQuality:
		radius = rand.Intn(9)      // 0-8
	}

	return &BaseShape{
		Position: PixelPosition{X: pixelX, Y: pixelY},
		Type:     Circular,
		Size:     radius,
		Quality:  quality,
	}
}

func NewSquare(pixelX, pixelY int, quality common.QualityType) *BaseShape {
	var size int
	switch quality {
	case common.LowQuality:
		size = rand.Intn(2) + 1    // 1-2 (matches current system)
	case common.NormalQuality:
		size = rand.Intn(3) + 1    // 1-3
	case common.HighQuality:
		size = rand.Intn(4) + 1    // 1-4
	}

	return &BaseShape{
		Position: PixelPosition{X: pixelX, Y: pixelY},
		Type:     Rectangular,
		Size:     size,
		Width:    size,    // Square: width = height = size
		Height:   size,
		Quality:  quality,
	}
}

func NewRectangle(pixelX, pixelY int, quality common.QualityType) *BaseShape {
	var width, height int
	switch quality {
	case common.LowQuality:
		width = rand.Intn(5)       // 0-4 (matches current system)
		height = rand.Intn(3)      // 0-2
	case common.NormalQuality:
		width = rand.Intn(7)       // 0-6
		height = rand.Intn(5)      // 0-4
	case common.HighQuality:
		width = rand.Intn(9)       // 0-8
		height = rand.Intn(7)      // 0-6
	}

	return &BaseShape{
		Position: PixelPosition{X: pixelX, Y: pixelY},
		Type:     Rectangular,
		Size:     width,   // Primary dimension
		Width:    width,
		Height:   height,
		Quality:  quality,
	}
}

func NewLine(pixelX, pixelY int, direction ShapeDirection, quality common.QualityType) *BaseShape {
	var length int
	switch quality {
	case common.LowQuality:
		length = rand.Intn(3) + 1  // 1-3 (matches current system)
	case common.NormalQuality:
		length = rand.Intn(5) + 1  // 1-5
	case common.HighQuality:
		length = rand.Intn(7) + 1  // 1-7
	}

	return &BaseShape{
		Position:  PixelPosition{X: pixelX, Y: pixelY},
		Type:      Linear,
		Size:      length,
		Direction: &direction,
		Quality:   quality,
	}
}

func NewCone(pixelX, pixelY int, direction ShapeDirection, quality common.QualityType) *BaseShape {
	var length int
	switch quality {
	case common.LowQuality:
		length = rand.Intn(3) + 1  // 1-3
	case common.NormalQuality:
		length = rand.Intn(5) + 1  // 1-5
	case common.HighQuality:
		length = rand.Intn(7) + 1  // 1-7
	}

	return &BaseShape{
		Position:  PixelPosition{X: pixelX, Y: pixelY},
		Type:      Linear,
		Size:      length,
		Direction: &direction,
		Quality:   quality,
	}
}

// ============================================================================
// UPDATE METHODS (Replace ShapeUpdater pattern)
// ============================================================================

func (s *BaseShape) UpdateSize(newSize int) {
	s.Size = newSize
	if s.Type == Rectangular && s.Width == s.Height {
		// Square case - update both dimensions
		s.Width = newSize
		s.Height = newSize
	}
}

func (s *BaseShape) UpdateDimensions(width, height int) {
	if s.Type == Rectangular {
		s.Width = width
		s.Height = height
		s.Size = width // Primary dimension
	}
}

func (s *BaseShape) Rotate() {
	if s.Direction != nil {
		*s.Direction = RotateRight(*s.Direction)
	}
}

func (s *BaseShape) SetDirection(direction ShapeDirection) {
	if s.Direction != nil {
		*s.Direction = direction
	}
}

// ============================================================================
// SHAPE ALGORITHMS
// ============================================================================

func (s *BaseShape) calculateCircle(centerX, centerY int) []int {
	var indices []int
	radius := s.Size
	for x := -radius; x <= radius; x++ {
		for y := -radius; y <= radius; y++ {
			if x*x + y*y <= radius*radius {
				indices = append(indices, CoordManager.LogicalToIndex(LogicalPosition{X: centerX+x, Y: centerY+y}))
			}
		}
	}
	return indices
}

func (s *BaseShape) calculateRectangle(centerX, centerY int) []int {
	var indices []int
	halfWidth := s.Width / 2
	halfHeight := s.Height / 2
	for x := -halfWidth; x <= halfWidth; x++ {
		for y := -halfHeight; y <= halfHeight; y++ {
			indices = append(indices, CoordManager.LogicalToIndex(LogicalPosition{X: centerX+x, Y: centerY+y}))
		}
	}
	return indices
}

func (s *BaseShape) calculateLine(centerX, centerY int) []int {
	var indices []int
	length := s.Size

	if s.Direction == nil {
		// Fallback: horizontal line
		for i := 0; i < length; i++ {
			indices = append(indices, CoordManager.LogicalToIndex(LogicalPosition{X: centerX+i, Y: centerY}))
		}
		return indices
	}

	// Calculate line based on direction
	deltaX, deltaY := DirectionToCoords(*s.Direction)
	for i := 0; i < length; i++ {
		x := centerX + i*deltaX
		y := centerY + i*deltaY
		indices = append(indices, CoordManager.LogicalToIndex(LogicalPosition{X: x, Y: y}))
	}

	return indices
}

// Helper function for direction to coordinates
func DirectionToCoords(direction ShapeDirection) (int, int) {
	switch direction {
	case LineUp:
		return 0, -1
	case LineDown:
		return 0, 1
	case LineRight:
		return 1, 0
	case LineLeft:
		return -1, 0
	case LineDiagonalUpRight:
		return 1, -1
	case LineDiagonalUpLeft:
		return -1, -1
	case LineDiagonalDownRight:
		return 1, 1
	case LinedDiagonalDownLeft:
		return -1, 1
	default:
		return 1, 0 // Default to right
	}
}

// GetLineTo creates a line from start position to end position
func GetLineTo(startPos common.Position, endPos common.Position) []int {
	startPixelPos := CoordManager.LogicalToPixel(LogicalPosition{X: startPos.X, Y: startPos.Y})
	endPixelPos := CoordManager.LogicalToPixel(LogicalPosition{X: endPos.X, Y: endPos.Y})

	// Calculate direction and length
	deltaX := endPixelPos.X - startPixelPos.X
	deltaY := endPixelPos.Y - startPixelPos.Y

	// Simple line drawing using step-based approach
	var indices []int
	steps := max(abs(deltaX), abs(deltaY))

	if steps == 0 {
		return []int{CoordManager.LogicalToIndex(LogicalPosition{X: startPos.X, Y: startPos.Y})}
	}

	for i := 0; i <= steps; i++ {
		x := startPixelPos.X + (deltaX*i)/steps
		y := startPixelPos.Y + (deltaY*i)/steps
		logical := CoordManager.PixelToLogical(PixelPosition{X: x, Y: y})
		indices = append(indices, CoordManager.LogicalToIndex(logical))
	}

	return indices
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}