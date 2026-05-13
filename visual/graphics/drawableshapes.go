package graphics

import (
	"game_main/core/common"
	"game_main/core/coords"
)

// ShapeSize determines the scale of shape dimensions.
// Only SmallShape is currently used by any call site; the type is preserved
// for future expansion (Medium/Large quality tiers were removed pending a
// concrete need).
type ShapeSize int

const (
	SmallShape ShapeSize = iota
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
	LineDiagonalDownLeft
	NoDirection // Returned by BaseShape.GetDirection when Direction is nil. Callers should prefer CanRotate() to detect direction-less shapes.
)

// allDirections is an iteration order for rotation helpers. Internal to this
// file; external callers should use RotateRight / RotateLeft.
var allDirections = []ShapeDirection{
	LineUp,
	LineDiagonalUpRight,
	LineRight,
	LineDiagonalDownRight,
	LineDown,
	LineDiagonalDownLeft,
	LineLeft,
	LineDiagonalUpLeft,
}

func RotateRight(dir ShapeDirection) ShapeDirection {
	for i, direction := range allDirections {
		if direction == dir {
			newDir := i + 1
			if newDir >= len(allDirections) {
				newDir = 0
			}
			return allDirections[newDir]
		}
	}
	return dir
}

func RotateLeft(dir ShapeDirection) ShapeDirection {
	for i, direction := range allDirections {
		if direction == dir {
			newDir := i - 1
			if newDir < 0 {
				newDir = len(allDirections) - 1
			}
			return allDirections[newDir]
		}
	}
	return dir
}

// ShapeConfig describes a tile-based shape to create from JSON data.
type ShapeConfig struct {
	Type   string
	Size   int
	Length int
	Width  int
	Height int
	Radius int
}

// CreateShapeFromConfig creates a BaseShape from a ShapeConfig.
// Returns a default 1x1 square if config is nil.
func CreateShapeFromConfig(cfg *ShapeConfig) *BaseShape {
	if cfg == nil {
		return NewSquare(0, 0, SmallShape)
	}

	var s *BaseShape
	switch cfg.Type {
	case "Circle":
		s = NewCircle(0, 0, SmallShape)
		if cfg.Size > 0 {
			s.UpdateSize(cfg.Size)
		}
	case "Square":
		s = NewSquare(0, 0, SmallShape)
		if cfg.Size > 0 {
			s.UpdateSize(cfg.Size)
		}
	case "Rectangle":
		s = NewRectangle(0, 0, SmallShape)
		if cfg.Width > 0 && cfg.Height > 0 {
			s.UpdateDimensions(cfg.Width, cfg.Height)
		}
	case "Line":
		s = NewLine(0, 0, LineDown, SmallShape)
		if cfg.Length > 0 {
			s.UpdateSize(cfg.Length)
		}
	case "Cone":
		s = NewCone(0, 0, LineDown, SmallShape)
		if cfg.Length > 0 {
			s.UpdateSize(cfg.Length)
		}
	default:
		return NewSquare(0, 0, SmallShape)
	}
	return s
}

// ============================================================================
// NEW SIMPLIFIED SHAPE SYSTEM
// ============================================================================

type BasicShapeType int

const (
	Circular    BasicShapeType = iota // radius-based (Circle types)
	Rectangular                       // width/height-based (Square/Rectangle types)
	Linear                            // length/direction-based (Line/Cone types)
)

type BaseShape struct {
	Position     coords.PixelPosition
	Type         BasicShapeType
	Size         int             // Primary dimension (radius, length, or width)
	Width        int             // For rectangles only
	Height       int             // For rectangles only
	Direction    *ShapeDirection // nil for non-directional shapes
	SizeCategory ShapeSize
}

// ============================================================================
// SHAPE METHODS
// ============================================================================

func (s *BaseShape) GetIndices() []int {
	logical := coords.CoordManager.PixelToLogical(s.Position)

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
	s.Position = coords.PixelPosition{X: pixelX, Y: pixelY}
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

func NewCircle(pixelX, pixelY int, size ShapeSize) *BaseShape {
	radius := common.RandomInt(3) // 0-2

	return &BaseShape{
		Position:     coords.PixelPosition{X: pixelX, Y: pixelY},
		Type:         Circular,
		Size:         radius,
		SizeCategory: size,
	}
}

func NewSquare(pixelX, pixelY int, shapeSize ShapeSize) *BaseShape {
	size := common.RandomInt(2) + 1 // 1-2

	return &BaseShape{
		Position:     coords.PixelPosition{X: pixelX, Y: pixelY},
		Type:         Rectangular,
		Size:         size,
		Width:        size, // Square: width = height = size
		Height:       size,
		SizeCategory: shapeSize,
	}
}

func NewRectangle(pixelX, pixelY int, size ShapeSize) *BaseShape {
	width := common.RandomInt(5)  // 0-4
	height := common.RandomInt(3) // 0-2

	return &BaseShape{
		Position:     coords.PixelPosition{X: pixelX, Y: pixelY},
		Type:         Rectangular,
		Size:         width, // Primary dimension
		Width:        width,
		Height:       height,
		SizeCategory: size,
	}
}

func NewLine(pixelX, pixelY int, direction ShapeDirection, size ShapeSize) *BaseShape {
	length := common.RandomInt(3) + 1 // 1-3

	return &BaseShape{
		Position:     coords.PixelPosition{X: pixelX, Y: pixelY},
		Type:         Linear,
		Size:         length,
		Direction:    &direction,
		SizeCategory: size,
	}
}

func NewCone(pixelX, pixelY int, direction ShapeDirection, size ShapeSize) *BaseShape {
	length := common.RandomInt(3) + 1 // 1-3

	return &BaseShape{
		Position:     coords.PixelPosition{X: pixelX, Y: pixelY},
		Type:         Linear,
		Size:         length,
		Direction:    &direction,
		SizeCategory: size,
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
			if x*x+y*y <= radius*radius {
				indices = append(indices, coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: centerX + x, Y: centerY + y}))
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
			indices = append(indices, coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: centerX + x, Y: centerY + y}))
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
			indices = append(indices, coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: centerX + i, Y: centerY}))
		}
		return indices
	}

	// Calculate line based on direction
	deltaX, deltaY := DirectionToCoords(*s.Direction)
	for i := 0; i < length; i++ {
		x := centerX + i*deltaX
		y := centerY + i*deltaY
		indices = append(indices, coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: x, Y: y}))
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
	case LineDiagonalDownLeft:
		return -1, 1
	default:
		return 1, 0 // Default to right
	}
}
