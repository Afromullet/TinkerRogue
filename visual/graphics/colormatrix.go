package graphics

// ColorMatrix is the RGBA transformation applied to a tile during rendering.
// A matrix with all four channels at zero is treated as a no-op (no tint applied).
// To force a true transparent-black tint, set A explicitly (any non-zero channel
// disables the no-op path).
type ColorMatrix struct {
	R float32
	G float32
	B float32
	A float32
}

func NewEmptyMatrix() ColorMatrix {
	return ColorMatrix{}
}

// Common predefined color matrices.
var (
	GreenColorMatrix = ColorMatrix{G: 1, A: 1}
	RedColorMatrix   = ColorMatrix{R: 1, A: 1}
)

// IsTransparent reports whether the matrix has all four channels at zero.
// The tile renderer treats this as "no color transformation needed" and skips
// the color application path entirely.
func (c ColorMatrix) IsTransparent() bool {
	return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0
}

// CreateRedGradient creates a red ColorMatrix with specified opacity
// Used for danger visualization where intensity indicates threat level
func CreateRedGradient(opacity float32) ColorMatrix {
	return ColorMatrix{R: 1, A: opacity}
}

// CreateBlueGradient creates a blue ColorMatrix with specified opacity
// Used for expected damage visualization
func CreateBlueGradient(opacity float32) ColorMatrix {
	return ColorMatrix{B: 1, A: opacity}
}

// CreateGreenGradient creates a green ColorMatrix with specified opacity
// Used for allied unit visualization
func CreateGreenGradient(opacity float32) ColorMatrix {
	return ColorMatrix{G: 1, A: opacity}
}

// CreateMagentaGradient creates a magenta/purple ColorMatrix with specified opacity
// Used for tiles with both ally and enemy units
func CreateMagentaGradient(opacity float32) ColorMatrix {
	return ColorMatrix{R: 1, B: 1, A: opacity}
}

// CreateYellowGradient creates a yellow ColorMatrix with specified opacity
// Used for highlighting selected squad's position
func CreateYellowGradient(opacity float32) ColorMatrix {
	return ColorMatrix{R: 1, G: 1, A: opacity}
}

// CreateOrangeGradient creates an orange ColorMatrix with specified opacity
// Used for melee threat zone visualization
func CreateOrangeGradient(opacity float32) ColorMatrix {
	return ColorMatrix{R: 1, G: 0.5, A: opacity}
}

// CreateCyanGradient creates a cyan ColorMatrix with specified opacity
// Used for ranged fire zone visualization
func CreateCyanGradient(opacity float32) ColorMatrix {
	return ColorMatrix{G: 1, B: 1, A: opacity}
}

// CreatePurpleGradient creates a purple ColorMatrix with specified opacity
// Used for isolation risk visualization
func CreatePurpleGradient(opacity float32) ColorMatrix {
	return ColorMatrix{R: 0.5, B: 0.5, A: opacity}
}

// CreateRedOrangeGradient creates a red-orange ColorMatrix with specified opacity
// Used for engagement pressure visualization
func CreateRedOrangeGradient(opacity float32) ColorMatrix {
	return ColorMatrix{R: 1, G: 0.3, A: opacity}
}
