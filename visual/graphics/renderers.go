package graphics

import (
	"game_main/common"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ImageRenderer draws image-based effects
type ImageRenderer struct{}

func (r *ImageRenderer) Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState) {
	if effect.img == nil {
		return
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(state.Scale*float64(ScreenInfo.ScaleFactor), state.Scale*float64(ScreenInfo.ScaleFactor))
	opts.GeoM.Translate(effect.startX+state.OffsetX, effect.startY+state.OffsetY)

	// Apply brightness if set
	if state.Brightness > 0 {
		opts.ColorM.Scale(state.Brightness, state.Brightness, state.Brightness, state.Opacity)
	} else if state.ColorShift > 0 {
		// Apply color shift if set (for ice effects)
		opts.ColorM.Scale(state.ColorShift, state.ColorShift, 1.0, state.Opacity)
	} else {
		// Default: just apply opacity
		opts.ColorM.Scale(1, 1, 1, state.Opacity)
	}

	screen.DrawImage(effect.img, opts)
}

// ProjectileRenderer draws projectile effects with rotation
type ProjectileRenderer struct {
	endX, endY float64
}

func (r *ProjectileRenderer) Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState) {
	if effect.img == nil {
		return
	}

	// Calculate the rotation angle based on the direction
	angle := math.Atan2(r.endY-effect.startY, r.endX-effect.startX)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(float64(ScreenInfo.ScaleFactor), float64(ScreenInfo.ScaleFactor))
	opts.GeoM.Translate(-float64(effect.img.Bounds().Dx())/2, -float64(effect.img.Bounds().Dy())/2) // Center the image
	opts.GeoM.Rotate(angle)
	opts.GeoM.Translate(effect.startX+state.OffsetX, effect.startY+state.OffsetY)

	screen.DrawImage(effect.img, opts)
}

// CloudRenderer draws cloud effects with multiple layers for fluffiness
type CloudRenderer struct{}

func (r *CloudRenderer) Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState) {
	if effect.img == nil {
		return
	}

	bounds := effect.img.Bounds()
	imgWidth := float64(bounds.Dx())
	imgHeight := float64(bounds.Dy())

	// Draw the main cloud image
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(state.Scale*float64(ScreenInfo.ScaleFactor), state.Scale*float64(ScreenInfo.ScaleFactor))

	// Adjust position to keep the center of the cloud in place
	adjustedX := effect.startX - (imgWidth * (state.Scale - 1) / 2)
	adjustedY := effect.startY - (imgHeight * (state.Scale - 1) / 2)
	opts.GeoM.Translate(adjustedX, adjustedY)
	opts.ColorM.Scale(1, 1, 1, state.Opacity)

	screen.DrawImage(effect.img, opts)

	// Create a subtle "fluffiness" effect by drawing multiple layers
	for i := 0; i < 2; i++ {
		layerOpts := &ebiten.DrawImageOptions{}
		layerScale := state.Scale * (1.0 - float64(i)*0.1)
		layerOpts.GeoM.Scale(layerScale, layerScale)

		layerAdjustedX := effect.startX - (imgWidth * (layerScale - 1) / 2)
		layerAdjustedY := effect.startY - (imgHeight * (layerScale - 1) / 2)
		layerOpts.GeoM.Translate(layerAdjustedX, layerAdjustedY)

		layerOpts.ColorM.Scale(1, 1, 1, 0.3*state.Opacity)
		screen.DrawImage(effect.img, layerOpts)
	}
}

// lineSegment type used by line-based effects
type lineSegment struct {
	x1, y1, x2, y2 float64
}

// LineSegmentRenderer draws line-based electrical effects
type LineSegmentRenderer struct {
	segments     []lineSegment
	color        color.RGBA
	numSegments  int
	jitterAmount float64
}

func NewLineSegmentRenderer(startX, startY int, numSegments int) *LineSegmentRenderer {
	segments := make([]lineSegment, numSegments)
	currentX, currentY := float64(startX), float64(startY)

	for i := 0; i < numSegments; i++ {
		nextX := currentX + -10 + 20*common.RandomFloat()
		nextY := currentY + -10 + 20*common.RandomFloat()

		segments[i] = lineSegment{
			x1: currentX,
			y1: currentY,
			x2: nextX,
			y2: nextY,
		}

		currentX, currentY = nextX, nextY
	}

	return &LineSegmentRenderer{
		segments:     segments,
		color:        color.RGBA{200 + uint8(common.GetDiceRoll(55)), 200 + uint8(common.GetDiceRoll(55)), 255, 255},
		numSegments:  numSegments,
		jitterAmount: 5.0,
	}
}

func (r *LineSegmentRenderer) Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState) {
	// Regenerate line segments to simulate flickering
	for i := range r.segments {
		r.segments[i].x2 += -r.jitterAmount + 2*r.jitterAmount*common.RandomFloat()
		r.segments[i].y2 += -r.jitterAmount + 2*r.jitterAmount*common.RandomFloat()
	}

	// Adjust color for electrical surges
	r.color.R = 200 + uint8(common.GetDiceRoll(55))
	r.color.G = 200 + uint8(common.GetDiceRoll(55))
	r.color.B = 255

	// Draw all segments
	for _, segment := range r.segments {
		ebitenutil.DrawLine(screen, segment.x1, segment.y1, segment.x2, segment.y2, r.color)
	}
}

// ElectricArcRenderer draws electric arc effects with multiple segments
type ElectricArcRenderer struct {
	endX, endY float64
	segments   [][]float64
	color      color.RGBA
	thickness  float32
}

func NewElectricArcRenderer(startX, startY, endX, endY int) *ElectricArcRenderer {
	return &ElectricArcRenderer{
		endX:      float64(endX),
		endY:      float64(endY),
		segments:  make([][]float64, 0),
		color:     color.RGBA{0, 191, 255, 255},
		thickness: 2,
	}
}

func (r *ElectricArcRenderer) Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState) {
	// Generate new segments for the electricity
	r.generateSegments(effect.startX, effect.startY)

	// Randomly adjust color and thickness
	r.color.R = uint8(common.GetDiceRoll(50))
	r.color.G = 200 + uint8(common.GetDiceRoll(55))
	r.color.B = 200 + uint8(common.GetDiceRoll(55))
	r.thickness = float32(1.5 + float32(common.RandomFloat()))

	// Draw segments
	for i := 0; i < len(r.segments)-1; i++ {
		vector.StrokeLine(screen, float32(r.segments[i][0]), float32(r.segments[i][1]),
			float32(r.segments[i+1][0]), float32(r.segments[i+1][1]),
			r.thickness, r.color, false)
	}
}

func (r *ElectricArcRenderer) generateSegments(startX, startY float64) {
	r.segments = make([][]float64, 0)
	r.segments = append(r.segments, []float64{startX, startY})

	currentX, currentY := startX, startY
	for i := 0; i < 10; i++ {
		nextX := currentX + (r.endX-currentX)/float64(10-i) + (common.RandomFloat()-0.5)*20
		nextY := currentY + (r.endY-currentY)/float64(10-i) + (common.RandomFloat()-0.5)*20
		r.segments = append(r.segments, []float64{nextX, nextY})
		currentX, currentY = nextX, nextY
	}

	r.segments = append(r.segments, []float64{r.endX, r.endY})
}

// ProceduralRenderer draws procedurally generated shapes for sticky ground effects
type ProceduralRenderer struct {
	baseColor color.Color
}

func NewProceduralRenderer(baseColor color.Color) *ProceduralRenderer {
	return &ProceduralRenderer{
		baseColor: baseColor,
	}
}

func (r *ProceduralRenderer) Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState) {
	waveOffset := time.Since(effect.startTime).Seconds() * 0.1

	// Create a color-based sticky ground effect with multiple circles
	for i := 0; i < 5; i++ {
		radius := 10 + 5*math.Sin(waveOffset+float64(i))
		x := effect.startX + 20*math.Cos(float64(i)+waveOffset)
		y := effect.startY + 20*math.Sin(float64(i)+waveOffset)

		// Generate an offscreen image to represent the shape (circle here)
		circleImage := ebiten.NewImage(int(2*radius), int(2*radius))
		circleImage.Fill(r.baseColor)

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-radius, -radius) // Center the circle
		opts.GeoM.Translate(x, y)
		opts.GeoM.Scale(state.Scale*float64(ScreenInfo.ScaleFactor), state.Scale*float64(ScreenInfo.ScaleFactor))
		opts.ColorM.Scale(1, 1, 1, state.Opacity)

		screen.DrawImage(circleImage, opts)
	}
}
