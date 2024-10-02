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
