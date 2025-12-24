package graphics

import (
	"game_main/common"
	"game_main/coords"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var VXHandler VisualEffectHandler

// There is VisualEffectArea and VisualEffectArea
// VisualEffectArea contains a VisualEffect and a TileBasedShape. The effects are drawn in the shape
// The VisualEffectHandler handler contains a slice of VisualEffects and VisualEffect areas and calls the Update and Draw function
// Copy returns a shallow copy

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

// BaseEffect implements VisualEffect interface
func (e *BaseEffect) UpdateVisualEffect() {
	if e.completed {
		return
	}

	elapsed := time.Since(e.startTime).Seconds()
	if int(elapsed) >= e.duration {
		e.completed = true
		return
	}

	// Animator will be called in DrawVisualEffect
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
	// Get wave offset from animator state (stored in OffsetX for convenience)
	// WaveAnimator should set waveOffset somehow - let's use a simple time-based approach
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

// FlickerAnimator implements flickering behavior (used by fire, electricity, etc.)
type FlickerAnimator struct {
	flickerTimer int
	scaleRange   [2]float64 // min, max scale
	opacityRange [2]float64 // min, max opacity
	jitterPos    bool       // whether to jitter position
}

func (a *FlickerAnimator) Update(effect *BaseEffect, elapsed float64) AnimationState {
	a.flickerTimer++
	state := AnimationState{
		Scale:   a.scaleRange[0] + (a.scaleRange[1]-a.scaleRange[0])*common.RandomFloat(),
		Opacity: a.opacityRange[0] + (a.opacityRange[1]-a.opacityRange[0])*common.RandomFloat(),
	}

	// Position jitter every 5 frames
	if a.jitterPos && a.flickerTimer%5 == 0 {
		state.OffsetX = -0.5 + common.RandomFloat()
		state.OffsetY = -0.5 + common.RandomFloat()
	}

	return state
}

func (a *FlickerAnimator) Reset() {
	a.flickerTimer = 0
}

// BrightnessFlickerAnimator for effects that flicker with brightness changes (like electricity with image)
type BrightnessFlickerAnimator struct {
	scaleRange      [2]float64 // min, max scale
	brightnessRange [2]float64 // min, max brightness
	jitterPos       bool       // whether to jitter position
}

func (a *BrightnessFlickerAnimator) Update(effect *BaseEffect, elapsed float64) AnimationState {
	state := AnimationState{
		Scale:      a.scaleRange[0] + (a.scaleRange[1]-a.scaleRange[0])*common.RandomFloat(),
		Brightness: a.brightnessRange[0] + (a.brightnessRange[1]-a.brightnessRange[0])*common.RandomFloat(),
		Opacity:    1.0,
	}

	// Position jitter for erratic movement
	if a.jitterPos {
		state.OffsetX = -1.0 + 2.0*common.RandomFloat()
		state.OffsetY = -1.0 + 2.0*common.RandomFloat()
	}

	return state
}

func (a *BrightnessFlickerAnimator) Reset() {
	// No state to reset
}

// ShimmerAnimator implements random shimmering behavior (used by ice effects)
type ShimmerAnimator struct {
	scaleRange   [2]float64 // min, max scale
	opacityRange [2]float64 // min, max opacity
	colorRange   [2]float64 // min, max color shift
}

func (a *ShimmerAnimator) Update(effect *BaseEffect, elapsed float64) AnimationState {
	return AnimationState{
		Scale:      a.scaleRange[0] + (a.scaleRange[1]-a.scaleRange[0])*common.RandomFloat(),
		Opacity:    a.opacityRange[0] + (a.opacityRange[1]-a.opacityRange[0])*common.RandomFloat(),
		ColorShift: a.colorRange[0] + (a.colorRange[1]-a.colorRange[0])*common.RandomFloat(),
	}
}

func (a *ShimmerAnimator) Reset() {
	// No state to reset
}

// SineShimmerAnimator implements sine-wave based shimmering (used by IceEffect2)
type SineShimmerAnimator struct {
	shimmerPhase float64
	scaleBase    float64
	scaleAmp     float64
	shimmerSpeed float64
}

func (a *SineShimmerAnimator) Update(effect *BaseEffect, elapsed float64) AnimationState {
	a.shimmerPhase += a.shimmerSpeed

	shimmerIntensity := 0.2 + 0.1*math.Sin(a.shimmerPhase)
	scale := a.scaleBase + a.scaleAmp*math.Sin(a.shimmerPhase)

	state := AnimationState{
		Scale:      scale,
		Opacity:    1.0,
		ColorShift: 1.0 + shimmerIntensity, // For brightness effect
	}

	return state
}

func (a *SineShimmerAnimator) Reset() {
	a.shimmerPhase = 0
}

// PulseAnimator implements smooth pulsing behavior (used by cloud effects)
type PulseAnimator struct {
	puffinessPhase float64
	scaleBase      float64
	scaleAmp       float64
	opacityBase    float64
	opacityAmp     float64
	phaseSpeed     float64
}

func (a *PulseAnimator) Update(effect *BaseEffect, elapsed float64) AnimationState {
	a.puffinessPhase += a.phaseSpeed

	return AnimationState{
		Scale:   a.scaleBase + a.scaleAmp*math.Sin(a.puffinessPhase),
		Opacity: a.opacityBase + a.opacityAmp*math.Sin(a.puffinessPhase*0.7),
	}
}

func (a *PulseAnimator) Reset() {
	a.puffinessPhase = 0
}

// MotionAnimator implements linear motion from start to end (used by projectiles)
type MotionAnimator struct {
	endX, endY         float64
	currentX, currentY float64
	speed              float64
	completed          bool
}

func NewMotionAnimator(startX, startY, endX, endY int, speed float64) *MotionAnimator {
	return &MotionAnimator{
		endX:      float64(endX),
		endY:      float64(endY),
		currentX:  float64(startX),
		currentY:  float64(startY),
		speed:     speed,
		completed: false,
	}
}

func (a *MotionAnimator) Update(effect *BaseEffect, elapsed float64) AnimationState {
	if a.completed {
		return AnimationState{Scale: 1.0, Opacity: 1.0}
	}

	dirX := a.endX - effect.startX
	dirY := a.endY - effect.startY

	length := math.Sqrt(dirX*dirX + dirY*dirY)
	dirX /= length
	dirY /= length

	a.currentX += dirX * a.speed
	a.currentY += dirY * a.speed

	// Check if we've arrived at the target
	if math.Abs(a.currentX-a.endX) < a.speed && math.Abs(a.currentY-a.endY) < a.speed {
		a.completed = true
		effect.completed = true
	}

	return AnimationState{
		Scale:   1.0,
		Opacity: 1.0,
		OffsetX: a.currentX - effect.startX,
		OffsetY: a.currentY - effect.startY,
	}
}

func (a *MotionAnimator) Reset() {
	a.currentX = 0
	a.currentY = 0
	a.completed = false
}

// WaveAnimator implements slow wave-based movement (used by sticky ground effects)
type WaveAnimator struct {
	waveOffset float64
	waveSpeed  float64
}

func (a *WaveAnimator) Update(effect *BaseEffect, elapsed float64) AnimationState {
	a.waveOffset += a.waveSpeed

	return AnimationState{
		Scale:   1.0,
		Opacity: 0.8 + 0.2*math.Sin(a.waveOffset),
	}
}

func (a *WaveAnimator) Reset() {
	a.waveOffset = 0
}

// completed tells us whether the VX is done displaying
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

// Applies the Visual Effect to the positions at the indices
// The same effect is drawn at all positions, which means when one is completed, all are completed
type VisualEffectArea struct {
	shape      TileBasedShape
	visEffects []VisualEffect
}

func NewVisualEffectArea(centerX, centerY int, shape TileBasedShape, vx VisualEffect) VisualEffectArea {
	indices := shape.GetIndices()
	visEffects := make([]VisualEffect, 0)

	for _, ind := range indices {
		// Use unified coordinate transformation - handles scrolling mode automatically
		centerPos := coords.LogicalPosition{X: centerX, Y: centerY}
		sx, sy := coords.CoordManager.IndexToScreen(ind, &centerPos)
		screenX, screenY := int(sx), int(sy)

		if vx != nil {
			vx.SetVXCommon(screenX, screenY, vx.VXImg())
			visEffects = append(visEffects, vx.Copy())
		}
	}

	return VisualEffectArea{
		shape:      shape,
		visEffects: visEffects,
	}
}

func (visArea *VisualEffectArea) DrawVisualEffect(screen *ebiten.Image) {

	for _, vx := range visArea.visEffects {

		vx.DrawVisualEffect(screen)

	}

}

func (visArea *VisualEffectArea) UpdateVisualEffect() {

	for _, vx := range visArea.visEffects {

		vx.UpdateVisualEffect()

	}

}

// If the first entry is completed, treat them all as complete
func (visArea *VisualEffectArea) IsCompleted() bool {

	if len(visArea.visEffects) > 0 {
		return visArea.visEffects[0].IsCompleted()

	}
	return false
}

// The VisualEffectHandler is called during the Update and Draw functions of the game loop
// To Draw whatever effect is in the list.
// ClearVisualEffect
type VisualEffectHandler struct {
	vx     []VisualEffect
	vxArea []VisualEffectArea
}

// Modifying the global vxHandler
func AddVX(a VisualEffect) {
	VXHandler.AddVisualEffect(a)
}

func AddVXArea(a VisualEffectArea) {
	VXHandler.AddVisualEffecArea(a)
}

func (vis *VisualEffectHandler) AddVisualEffect(a VisualEffect) {

	vis.vx = append(vis.vx, a)

}

func (vis *VisualEffectHandler) AddVisualEffecArea(a VisualEffectArea) {

	vis.vxArea = append(vis.vxArea, a)

}

func (vis *VisualEffectHandler) clearVisualEffects() {

	remainingEffects := make([]VisualEffect, 0)
	remainingAreaEffects := make([]VisualEffectArea, 0)

	for _, v := range vis.vx {

		if !v.IsCompleted() {
			remainingEffects = append(remainingEffects, v)
		}

	}

	for _, v := range vis.vxArea {
		if !v.IsCompleted() {

			remainingAreaEffects = append(remainingAreaEffects, v)

		}
	}

	vis.vx = remainingEffects
	vis.vxArea = remainingAreaEffects

}

func (vis *VisualEffectHandler) UpdateVisualEffects() {

	vis.clearVisualEffects()

	for _, v := range vis.vx {
		v.UpdateVisualEffect()
	}

	for _, v := range vis.vxArea {
		v.UpdateVisualEffect()
	}

}

func (vis VisualEffectHandler) DrawVisualEffects(screen *ebiten.Image) {

	for _, v := range vis.vx {
		v.DrawVisualEffect(screen)
	}

	for _, v := range vis.vxArea {
		v.DrawVisualEffect(screen)
	}

}

// NewProjectile creates a projectile effect using BaseEffect + MotionAnimator + ProjectileRenderer
func NewProjectile(startX, startY, endX, endY int) VisualEffect {
	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/arrow3.png")

	return &BaseEffect{
		startX:           float64(startX),
		startY:           float64(startY),
		startTime:        time.Now(),
		duration:         999999, // Projectile completes when it reaches target, not after duration
		originalDuration: 999999,
		img:              vxImg,
		animator:         NewMotionAnimator(startX, startY, endX, endY, 5.0),
		renderer:         &ProjectileRenderer{endX: float64(endX), endY: float64(endY)},
	}
}

// NewFireEffect creates a fire effect using BaseEffect + FlickerAnimator
// Parameters: startX, startY, flickerTimer (ignored - kept for backward compatibility),
// duration, scale (ignored - handled by animator), opacity (ignored - handled by animator)
func NewFireEffect(startX, startY, flickerTimer, duration int, scale, opacity float64) VisualEffect {
	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/cloud_fire2.png")

	return &BaseEffect{
		startX:           float64(startX),
		startY:           float64(startY),
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		img:              vxImg,
		animator: &FlickerAnimator{
			scaleRange:   [2]float64{0.95, 1.05},
			opacityRange: [2]float64{0.7, 1.0},
			jitterPos:    true,
		},
		renderer: &ImageRenderer{},
	}
}

// NewIceEffect creates an ice effect using BaseEffect + ShimmerAnimator
func NewIceEffect(startX, startY int, duration int) VisualEffect {
	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/frost0.png")

	return &BaseEffect{
		startX:           float64(startX),
		startY:           float64(startY),
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		img:              vxImg,
		animator: &ShimmerAnimator{
			scaleRange:   [2]float64{0.98, 1.02},
			opacityRange: [2]float64{0.85, 1.0},
			colorRange:   [2]float64{0.9, 1.0},
		},
		renderer: &ImageRenderer{},
	}
}

// NewIceEffect2 creates an ice effect with sine-wave shimmering using BaseEffect + SineShimmerAnimator
func NewIceEffect2(x, y int, duration int) VisualEffect {
	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/frost0.png")

	return &BaseEffect{
		startX:           float64(x),
		startY:           float64(y),
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		img:              vxImg,
		animator: &SineShimmerAnimator{
			scaleBase:    1.0,
			scaleAmp:     0.05,
			shimmerSpeed: 0.1,
		},
		renderer: &ImageRenderer{},
	}
}

// NewCloudEffect creates a cloud effect with pulsing animation using BaseEffect + PulseAnimator + CloudRenderer
func NewCloudEffect(startX, startY int, duration int) VisualEffect {
	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/cloud_poison0.png")

	return &BaseEffect{
		startX:           float64(startX),
		startY:           float64(startY),
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		img:              vxImg,
		animator: &PulseAnimator{
			scaleBase:   1.0,
			scaleAmp:    0.05,
			opacityBase: 0.7,
			opacityAmp:  0.2,
			phaseSpeed:  0.05,
		},
		renderer: &CloudRenderer{},
	}
}

// lineSegment type used by line-based effects
type lineSegment struct {
	x1, y1, x2, y2 float64
}

// NewElectricityEffectNoImage creates a line-based electricity effect using BaseEffect + LineSegmentRenderer
func NewElectricityEffectNoImage(startX, startY int, duration int, numSegments int) VisualEffect {
	return &BaseEffect{
		startX:           float64(startX),
		startY:           float64(startY),
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		img:              nil,
		animator:         nil, // No animator - rendering is stateless
		renderer:         NewLineSegmentRenderer(startX, startY, numSegments),
	}
}

// NewElectricityEffect creates an electricity effect using BaseEffect + BrightnessFlickerAnimator
func NewElectricityEffect(startX, startY int, duration int) VisualEffect {
	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/zap0.png")

	return &BaseEffect{
		startX:           float64(startX),
		startY:           float64(startY),
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		img:              vxImg,
		animator: &BrightnessFlickerAnimator{
			scaleRange:      [2]float64{0.9, 1.1},
			brightnessRange: [2]float64{1.0, 1.5},
			jitterPos:       true,
		},
		renderer: &ImageRenderer{},
	}
}

// NewElectricArc creates an electric arc effect using BaseEffect + ElectricArcRenderer
func NewElectricArc(startX, startY, endX, endY int, duration int) VisualEffect {
	return &BaseEffect{
		startX:           float64(startX),
		startY:           float64(startY),
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		img:              nil,
		animator:         nil, // No animator - rendering is stateless
		renderer:         NewElectricArcRenderer(startX, startY, endX, endY),
	}
}

// NewStickyGroundEffect creates a sticky ground effect using BaseEffect + WaveAnimator + ProceduralRenderer
func NewStickyGroundEffect(startX, startY int, duration int) VisualEffect {
	return &BaseEffect{
		startX:           float64(startX),
		startY:           float64(startY),
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		img:              nil,
		animator: &WaveAnimator{
			waveOffset: 0.0,
			waveSpeed:  0.01,
		},
		renderer: NewProceduralRenderer(color.RGBA{0x90, 0xEE, 0x90, 0xFF}),
	}
}
