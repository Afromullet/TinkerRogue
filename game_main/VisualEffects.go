package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var vxHandler VisualEffectHandler

// The VisualEffectHandler is called during the Update and Draw functions of the game loop
// To Draw whatever effect is in the list.
// ClearVisualEffect
type VisualEffectHandler struct {
	vx []VisualEffect
}

func (vis *VisualEffectHandler) AddVisualEffect(a VisualEffect) {

	vis.vx = append(vis.vx, a)

}

func (vis *VisualEffectHandler) clearVisualEffects() {

	remainingEffects := make([]VisualEffect, 0)

	for _, v := range vis.vx {

		if !v.IsCompleted() {
			remainingEffects = append(remainingEffects, v)
		}

	}

	vis.vx = remainingEffects

}

func (vis *VisualEffectHandler) UpdateVisualEffects() {

	vis.clearVisualEffects()
	remainingEffects := make([]VisualEffect, 0)

	for _, v := range vis.vx {

		if !v.IsCompleted() {
			remainingEffects = append(remainingEffects, v)
		}

	}

	vis.vx = remainingEffects

	for _, v := range vis.vx {
		v.UpdateVisualEffect()
	}

}

func (vis VisualEffectHandler) DrawVisualEffects(screen *ebiten.Image) {

	for _, v := range vis.vx {
		v.DrawVisualEffect(screen)
	}

}

type VisualEffectArea struct {
	indices []int
	VisualEffect
}

type VisualEffect interface {
	UpdateVisualEffect()
	DrawVisualEffect(screen *ebiten.Image)
	IsCompleted() bool
}

type VXCommon struct {
	completed      bool
	startX, startY float64
	img            *ebiten.Image // Your arrow image
}

func NewVXCommon(imgPath string, startX, startY int) VXCommon {

	vxImg, _, _ := ebitenutil.NewImageFromFile(imgPath)
	return VXCommon{
		img:       vxImg,
		completed: false,
		startX:    float64(startX),
		startY:    float64(startY),
	}

}

func (vc *VXCommon) isComplete() bool {
	return vc.completed

}

type Projectile struct {
	VXCommon

	endX, endY         float64
	currentX, currentY float64
	speed              float64
}

func NewProjectile(startX, startY, endX, endY int) *Projectile {

	vxCom := NewVXCommon("assets/effects/arrow3.png", startX, startY)
	pro := &Projectile{
		VXCommon: vxCom,

		endX:     float64(endX),
		endY:     float64(endY),
		currentX: float64(startX),
		currentY: float64(startY),
		speed:    5.0,
	}

	return pro
}

// Moves the projectile allong the path every time UpdateVisualEffect is called
// This happens during the Update function in the game loop
func (a *Projectile) UpdateVisualEffect() {
	if a.completed {
		return
	}

	dirX := float64(a.endX - a.startX)
	dirY := float64(a.endY - a.startY)

	length := math.Sqrt(dirX*dirX + dirY*dirY)
	dirX /= length
	dirY /= length

	a.currentX += dirX * a.speed
	a.currentY += dirY * a.speed

	// Check if we've arrived at the target
	if math.Abs(a.currentX-float64(a.endX)) < a.speed && math.Abs(a.currentY-float64(a.endY)) < a.speed {
		a.completed = true
	}
}

func (a *Projectile) DrawVisualEffect(screen *ebiten.Image) {
	if a.completed {
		return
	}

	// Calculate the rotation angle based on the direction
	angle := math.Atan2(float64(a.endY-a.startY), float64(a.endX-a.startX))

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(-float64(a.img.Bounds().Dx())/2, -float64(a.img.Bounds().Dy())/2) // Center the image
	opts.GeoM.Rotate(angle)
	opts.GeoM.Translate(a.currentX, a.currentY)

	// Draw the arrow on the screen
	screen.DrawImage(a.img, opts)
}

func (a *Projectile) IsCompleted() bool {

	return a.isComplete()

}

type FireEffect struct {
	VXCommon
	startTime              time.Time
	flickerTimer, duration int     // Timer for flickering
	scale                  float64 // Scale of the fire (for flickering size)
	opacity                float64 // Opacity of the fire (for flickering brightness)

}

func NewFireEffect(startX, startY, flickerTimer, duration int, scale, opacity float64) *FireEffect {

	vxImg, _, _ := ebitenutil.NewImageFromFile("assets/effects/cloud_fire2.png")

	pro := &FireEffect{
		VXCommon: VXCommon{
			img:       vxImg,
			completed: false,
			startX:    float64(startX),
			startY:    float64(startY),
		},
		flickerTimer: flickerTimer,
		startTime:    time.Now(),
		duration:     duration,
		scale:        scale,
		opacity:      opacity,
	}

	return pro
}

func (f *FireEffect) UpdateVisualEffect() {

	elapsed := time.Since(f.startTime).Seconds()

	// Increment flicker timer
	f.flickerTimer++
	fmt.Println("Flicker timer ", f.flickerTimer)

	// Randomly change the scale slightly to simulate flickering
	f.scale = 0.95 + 0.1*rand.Float64()

	// Randomly adjust opacity to simulate flickering brightness
	f.opacity = 0.7 + 0.3*rand.Float64()

	// Optional: You can also vary position slightly to simulate movement
	if f.flickerTimer%5 == 0 { // Adjust every few frames
		f.startX += -0.5 + rand.Float64()
		f.startY += -0.5 + rand.Float64()
	}

	// Check if the effect has burned for the specified duration
	if int(elapsed) >= f.duration {
		f.completed = true
	}

}

func (f *FireEffect) DrawVisualEffect(screen *ebiten.Image) {

	opts := &ebiten.DrawImageOptions{}

	// Set the scale for flickering size
	opts.GeoM.Scale(f.scale, f.scale)

	// Set the position
	opts.GeoM.Translate(f.startX, f.startY)

	// Set the opacity (color modulation)
	opts.ColorM.Scale(1, 1, 1, f.opacity)

	// Draw the fire image on the screen with flickering effect
	screen.DrawImage(f.img, opts)

}

func (f *FireEffect) IsCompleted() bool {

	return f.isComplete()

}
