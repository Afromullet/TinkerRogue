package graphics

import "fmt"

// This is not an elegant way to update the shape. At all. Every TileBasedShape implements an UpdateShape(u ShapeUpdater) method
// ShapeUpdater contains all the parameters of the shapes, and the implemented will use only the parameters it needs to udpate it
type ShapeUpdater struct {
	PixelX    int
	PixelY    int
	Size      int
	Length    int
	Width     int
	Radius    int
	height    int
	Direction ShapeDirection
}

func SquareUpdate(pixelX, pixelY, size int) ShapeUpdater {

	updater := ShapeUpdater{
		PixelX: pixelX,
		PixelY: pixelY,
	}

	updater.Size = size

	return updater

}

func LineUpdate(pixelX, pixelY, length int, direction ShapeDirection) ShapeUpdater {

	updater := ShapeUpdater{
		PixelX: pixelX,
		PixelY: pixelY,
	}

	updater.Length = length
	updater.Direction = direction
	return updater

}

func ConeUpdate(pixelX, pixelY, length int, direction ShapeDirection) ShapeUpdater {

	updater := ShapeUpdater{
		PixelX: pixelX,
		PixelY: pixelY,
	}

	updater.Length = length
	updater.Direction = direction

	return updater

}

func CircleUpdate(pixelX, pixelY, radius int) ShapeUpdater {

	updater := ShapeUpdater{
		PixelX: pixelX,
		PixelY: pixelY,
	}

	updater.Radius = radius

	return updater

}

func RectangleUpdate(pixelX, pixelY, length, width int) ShapeUpdater {

	updater := ShapeUpdater{
		PixelX: pixelX,
		PixelY: pixelY,
	}

	updater.Length = length
	updater.Width = width

	return updater

}

// Gets a ShapeUpdater containing the current shapes parameters
func ExtractShapeParams(shape TileBasedShape) ShapeUpdater {

	updater := ShapeUpdater{}
	// Type switch to determine the underlying type
	switch s := shape.(type) {
	case *TileSquare:
		updater.PixelX = s.PixelX
		updater.PixelY = s.PixelY
		updater.Size = s.Size

	case *TileCircle:
		updater.PixelX = s.pixelX
		updater.PixelY = s.pixelY
		updater.Radius = s.radius

	case *TileLine:
		updater.PixelX = s.pixelX
		updater.PixelY = s.pixelY
		updater.Length = s.length
		updater.Direction = s.direction
	case *TileCone:
		updater.PixelX = s.pixelX
		updater.PixelY = s.pixelY
		updater.Length = s.length
		updater.Direction = s.direction

	case *TileRectangle:
		updater.PixelX = s.pixelX
		updater.PixelY = s.pixelY
		updater.Width = s.width
		updater.height = s.height

	default:
		// Handle any other cases (if applicable)
		fmt.Println("Unknown shape type")
	}

	return updater

}

// Used for changing the direction for the drawing portion of the game.
func GetDirection(shape TileBasedShape) ShapeDirection {

	switch s := shape.(type) {
	case *TileLine:
		return s.direction

	case *TileCone:
		return s.direction
	default:
		return NoDirection
	}

}
