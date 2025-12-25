package graphics

// The ColorMatrix lets us track what RGBA transformations we want to apply to a tile
// It either scales a color or draws a color
type ColorMatrix struct {
	R           float32
	G           float32
	B           float32
	A           float32
	ApplyMatrix bool
}

func NewEmptyMatrix() ColorMatrix {
	return ColorMatrix{
		R:           0,
		G:           0,
		B:           0,
		A:           0,
		ApplyMatrix: true,
	}
}

func (c ColorMatrix) IsEmpty() bool {
	if c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0 {
		return true
	}

	return false
}

// CreateRedGradient creates a red ColorMatrix with specified opacity
// Used for danger visualization where intensity indicates threat level
func CreateRedGradient(opacity float32) ColorMatrix {
	return ColorMatrix{
		R:           1,
		G:           0,
		B:           0,
		A:           opacity,
		ApplyMatrix: true,
	}
}

// CreateBlueGradient creates a blue ColorMatrix with specified opacity
// Used for expected damage visualization
func CreateBlueGradient(opacity float32) ColorMatrix {
	return ColorMatrix{
		R:           0,
		G:           0,
		B:           1,
		A:           opacity,
		ApplyMatrix: true,
	}
}

// CreateGreenGradient creates a green ColorMatrix with specified opacity
// Used for allied unit visualization
func CreateGreenGradient(opacity float32) ColorMatrix {
	return ColorMatrix{
		R:           0,
		G:           1,
		B:           0,
		A:           opacity,
		ApplyMatrix: true,
	}
}

// CreateMagentaGradient creates a magenta/purple ColorMatrix with specified opacity
// Used for tiles with both ally and enemy units
func CreateMagentaGradient(opacity float32) ColorMatrix {
	return ColorMatrix{
		R:           1,
		G:           0,
		B:           1,
		A:           opacity,
		ApplyMatrix: true,
	}
}

// CreateYellowGradient creates a yellow ColorMatrix with specified opacity
// Used for highlighting selected squad's position
func CreateYellowGradient(opacity float32) ColorMatrix {
	return ColorMatrix{
		R:           1,
		G:           1,
		B:           0,
		A:           opacity,
		ApplyMatrix: true,
	}
}
