package graphics

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// VisualEffect is the core interface for all visual effects.
type VisualEffect interface {
	UpdateVisualEffect()
	DrawVisualEffect(screen *ebiten.Image)
	SetVXCommon(x, y int, img *ebiten.Image)
	IsCompleted() bool
	VXImg() *ebiten.Image
	ResetVX()
	Copy() VisualEffect
}

// AnimationState holds animated properties for rendering
type AnimationState struct {
	Scale      float64
	Opacity    float64
	Brightness float64
	ColorShift float64
	OffsetX    float64
	OffsetY    float64
}

// Animator interface defines how effect properties change over time
type Animator interface {
	Update(effect *BaseEffect, elapsed float64) AnimationState
	Reset()
}

// Renderer interface defines how to draw the effect
type Renderer interface {
	Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState)
}

// BaseEffect handles lifecycle, timing, and position (used by all effects)
type BaseEffect struct {
	startX, startY   float64
	startTime        time.Time
	duration         int
	originalDuration int
	completed        bool
	img              *ebiten.Image
	animator         Animator
	renderer         Renderer
}

func (e *BaseEffect) UpdateVisualEffect() {
	if e.completed {
		return
	}

	elapsed := time.Since(e.startTime).Seconds()
	if int(elapsed) >= e.duration {
		e.completed = true
		return
	}
}

func (e *BaseEffect) DrawVisualEffect(screen *ebiten.Image) {
	if e.completed {
		return
	}

	elapsed := time.Since(e.startTime).Seconds()
	state := AnimationState{Scale: 1.0, Opacity: 1.0} // defaults
	if e.animator != nil {
		state = e.animator.Update(e, elapsed)
	}

	if e.renderer != nil {
		e.renderer.Draw(screen, e, state)
	}
}

func (e *BaseEffect) IsCompleted() bool {
	return e.completed
}

func (e *BaseEffect) SetVXCommon(x, y int, img *ebiten.Image) {
	e.startX = float64(x)
	e.startY = float64(y)
	e.img = img
}

func (e *BaseEffect) VXImg() *ebiten.Image {
	return e.img
}

func (e *BaseEffect) ResetVX() {
	e.startTime = time.Now()
	e.completed = false
	e.duration = e.originalDuration
	if e.animator != nil {
		e.animator.Reset()
	}
}

func (e *BaseEffect) Copy() VisualEffect {
	// Shallow copy - animators and renderers are stateless except for frame counters
	copy := *e
	return &copy
}
