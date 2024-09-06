package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var VisualEffects = make([]*Arrow, 0)

func AddVisualEffect(a *Arrow) {

	VisualEffects = append(VisualEffects, a)

}

func ClearVisualEffects() {
	VisualEffects = VisualEffects[:0]
}

func UpdateVisualEffects() {
	for _, v := range VisualEffects {
		v.Update()
	}
}

func DrawVisualEffects(screen *ebiten.Image) {
	for _, v := range VisualEffects {
		v.Draw(screen)
	}
}

type Arrow struct {
	img                *ebiten.Image // Your arrow image
	startX, startY     int
	endX, endY         int
	currentX, currentY float64
	speed              float64
	completed          bool
}

func NewArrow(startX, startY, endX, endY int) *Arrow {

	arrowImg, _, _ := ebitenutil.NewImageFromFile("assets/effects/arrow3.png")

	arrow := &Arrow{
		img:       arrowImg, // The arrow image you loaded
		startX:    startX,   // Starting position of the arrow
		startY:    startY,
		endX:      endX, // Target position
		endY:      endY,
		currentX:  float64(startX), // Current position, starts at the start point
		currentY:  float64(startY),
		speed:     5.0, // Adjust the speed as needed
		completed: false,
	}

	return arrow
}

func (a *Arrow) Update() {
	if a.completed {
		return
	}

	// Calculate the direction vector
	dirX := float64(a.endX - a.startX)
	dirY := float64(a.endY - a.startY)

	// Normalize the direction vector
	length := math.Sqrt(dirX*dirX + dirY*dirY)
	dirX /= length
	dirY /= length

	// Move the arrow along the direction
	a.currentX += dirX * a.speed
	a.currentY += dirY * a.speed

	// Check if the arrow has reached the target (or close enough)
	if math.Abs(a.currentX-float64(a.endX)) < a.speed && math.Abs(a.currentY-float64(a.endY)) < a.speed {
		a.completed = true
	}
}

func (a *Arrow) Draw(screen *ebiten.Image) {
	if a.completed {
		return
	}

	// Calculate the rotation angle based on the direction
	angle := math.Atan2(float64(a.endY-a.startY), float64(a.endX-a.startX))

	// Create draw options and set the rotation
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(-float64(a.img.Bounds().Dx())/2, -float64(a.img.Bounds().Dy())/2) // Center the image
	opts.GeoM.Rotate(angle)
	opts.GeoM.Translate(a.currentX, a.currentY)

	// Draw the arrow on the screen
	screen.DrawImage(a.img, opts)
}
