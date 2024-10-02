package graphics

import (
	"game_main/randgen"
	"image/color"
	"math"

	"math/rand/v2"
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

		var x, y = 0, 0
		if MAP_SCROLLING_ENABLED {
			x, y = LogicalCoordsFromIndex(ind)
		} else {
			x, y = PixelsFromIndex(ind)
		}

		// Transform logical coordinates to screen coordinates
		screenX, screenY := OffsetFromCenter(centerX, centerY, x*ScreenInfo.TileSize, y*ScreenInfo.TileSize, ScreenInfo)

		if vx != nil {
			if MAP_SCROLLING_ENABLED {
				vx.SetVXCommon(int(screenX), int(screenY), vx.VXImg())
			} else {
				vx.SetVXCommon(x, y, vx.VXImg())
			}
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

type Projectile struct {
	VXCommon

	endX, endY         float64
	currentX, currentY float64
	speed              float64
}

// Todo, every projectile currently uses the same VX
func NewProjectile(startX, startY, endX, endY int) *Projectile {

	vxCom := NewVXCommon("../assets/effects/arrow3.png", startX, startY)
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

func (a *Projectile) SetVXCommon(x, y int, img *ebiten.Image) {
	a.startX = float64(x)
	a.startY = float64(y)
	a.img = img
}

func (a *Projectile) IsCompleted() bool {

	return a.isComplete()

}

func (a *Projectile) VXImg() *ebiten.Image {
	return a.img
}

// Projectile does not have a duration but we still need the function to implement the interface
func (a *Projectile) ResetVX() {

}

func (s *Projectile) Copy() VisualEffect {

	return &Projectile{
		VXCommon: s.VXCommon,
		endX:     s.endX,
		endY:     s.endY,
		currentX: s.currentX,
		currentY: s.currentY,
	}

}

type FireEffect struct {
	VXCommon
	startTime                                time.Time
	flickerTimer, duration, originalDuration int     // Timer for flickering
	scale                                    float64 // Scale of the fire (for flickering size)
	opacity                                  float64 // Opacity of the fire (for flickering brightness)

}

func NewFireEffect(startX, startY, flickerTimer, duration int, scale, opacity float64) *FireEffect {

	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/cloud_fire2.png")

	pro := &FireEffect{
		VXCommon: VXCommon{
			img:       vxImg,
			completed: false,
			startX:    float64(startX),
			startY:    float64(startY),
		},
		flickerTimer:     flickerTimer,
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		scale:            scale,
		opacity:          opacity,
	}

	return pro
}

func (f *FireEffect) UpdateVisualEffect() {

	elapsed := time.Since(f.startTime).Seconds()

	// Increment flicker timer
	f.flickerTimer++

	// Randomly change the scale slightly to simulate flickering
	f.scale = 0.95 + 0.1*rand.Float64()

	// Randomly adjust opacity to simulate flickering brightness
	f.opacity = 0.7 + 0.3*rand.Float64()

	// Vary position slightly to simulate movement
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

func (f *FireEffect) SetVXCommon(x, y int, img *ebiten.Image) {
	f.startX = float64(x)
	f.startY = float64(y)
	f.img = img
}

func (f *FireEffect) IsCompleted() bool {

	return f.isComplete()

}

func (f *FireEffect) VXImg() *ebiten.Image {
	return f.img
}

// Projectile does not have a duration but we still need the function to implement the interface
func (f *FireEffect) ResetVX() {

	f.startTime = time.Now()
	f.completed = false
	f.duration = f.originalDuration

}

func (f *FireEffect) Copy() VisualEffect {

	return &FireEffect{
		VXCommon:         f.VXCommon,
		startTime:        f.startTime,
		flickerTimer:     f.flickerTimer,
		duration:         f.duration,
		originalDuration: f.originalDuration,
		scale:            f.scale,
		opacity:          f.opacity,
	}

}

type IceEffect struct {
	img                        *ebiten.Image // Your ice image
	startX, startY             float64
	scale                      float64
	opacity                    float64
	startTime                  time.Time
	duration, originalDuration int
	completed                  bool
	colorShift                 float64
}

func NewIceEffect(startX, startY int, duration int) *IceEffect {

	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/frost0.png")
	return &IceEffect{
		img:              vxImg,
		startX:           float64(startX),
		startY:           float64(startY),
		scale:            1.0, // Initial scale
		opacity:          1.0, // Initial opacity
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		completed:        false,
	}
}

func (ice *IceEffect) UpdateVisualEffect() {
	elapsed := time.Since(ice.startTime).Seconds()

	// Randomly change the scale slightly to simulate shimmering
	ice.scale = 0.98 + 0.04*rand.Float64()

	// Randomly adjust opacity to simulate light reflections
	ice.opacity = 0.85 + 0.15*rand.Float64()

	// Subtle color shifting (optional)
	ice.colorShift = 0.9 + 0.1*rand.Float64()

	// Check if the effect has lasted for the specified duration
	if int(elapsed) >= ice.duration {
		ice.completed = true
	}
}

func (ice *IceEffect) DrawVisualEffect(screen *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}

	// Set the scale for the shimmering effect
	opts.GeoM.Scale(ice.scale, ice.scale)

	// Set the position of the ice effect
	opts.GeoM.Translate(ice.startX, ice.startY)

	// Apply the opacity and color shifting to simulate shimmer
	opts.ColorM.Scale(ice.colorShift, ice.colorShift, 1.0, ice.opacity)

	// Draw the ice image with the shimmering effect
	screen.DrawImage(ice.img, opts)
}

func (s *IceEffect) SetVXCommon(x, y int, img *ebiten.Image) {
	s.startX = float64(x)
	s.startY = float64(y)
	s.img = img
}

func (i *IceEffect) IsCompleted() bool {
	return i.completed
}

func (i *IceEffect) VXImg() *ebiten.Image {
	return i.img
}

// Projectile does not have a duration but we still need the function to implement the interface
func (i *IceEffect) ResetVX() {

	i.startTime = time.Now()
	i.completed = false
	i.duration = i.originalDuration

}

func (ice *IceEffect) Copy() VisualEffect {
	return &IceEffect{
		img:              ice.img,
		startX:           ice.startX,
		startY:           ice.startY,
		scale:            ice.scale,
		opacity:          ice.opacity,
		startTime:        ice.startTime,
		duration:         ice.duration,
		originalDuration: ice.duration,
		completed:        ice.completed,
		colorShift:       ice.colorShift,
	}
}

type IceEffect2 struct {
	img              *ebiten.Image
	startX           float64
	startY           float64
	startTime        time.Time
	duration         int
	originalDuration int
	completed        bool
	shimmerPhase     float64
	scale            float64
}

func NewIceEffect2(x, y int, duration int) *IceEffect2 {

	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/frost0.png")

	return &IceEffect2{
		img:              vxImg,
		startX:           float64(x),
		startY:           float64(y),
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		scale:            1.0,
	}
}

func (i *IceEffect2) UpdateVisualEffect() {
	elapsed := time.Since(i.startTime).Seconds()

	// Update shimmer phase
	i.shimmerPhase += 0.1 // Adjust this value to change shimmer speed

	// Slightly vary scale to simulate ice "growing" or "shrinking"
	i.scale = 1.0 + 0.05*math.Sin(i.shimmerPhase)

	// Check if the effect has lasted for the specified duration
	if int(elapsed) >= i.duration {
		i.completed = true
	}
}

func (i *IceEffect2) DrawVisualEffect(screen *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}

	// Set the scale
	opts.GeoM.Scale(i.scale, i.scale)

	// Set the position
	opts.GeoM.Translate(i.startX, i.startY)

	// Create shimmering effect by manipulating color
	shimmerIntensity := 0.2 + 0.1*math.Sin(i.shimmerPhase)
	opts.ColorM.Scale(1+shimmerIntensity, 1+shimmerIntensity, 1+shimmerIntensity, 1)

	// Add a slight blue tint
	opts.ColorM.Translate(0, 0, 0.1, 0)

	// Draw the ice image on the screen with shimmering effect
	screen.DrawImage(i.img, opts)
}

func (s *IceEffect2) SetVXCommon(x, y int, img *ebiten.Image) {
	s.startX = float64(x)
	s.startY = float64(y)
	s.img = img
}

func (i *IceEffect2) IsCompleted() bool {
	return i.completed
}

func (s *IceEffect2) VXImg() *ebiten.Image {
	return s.img
}

// Projectile does not have a duration but we still need the function to implement the interface
func (i *IceEffect2) ResetVX() {

	i.startTime = time.Now()
	i.completed = false
	i.duration = i.originalDuration

}

func (s *IceEffect2) Copy() VisualEffect {
	return &IceEffect2{
		img:              s.img,
		startX:           float64(s.startX),
		startY:           float64(s.startY),
		startTime:        s.startTime,
		duration:         s.duration,
		originalDuration: s.duration,
		completed:        s.completed,
		shimmerPhase:     s.shimmerPhase,
		scale:            s.scale,
	}
}

type CloudEffect struct {
	img              *ebiten.Image
	startX, startY   float64
	scale            float64
	opacity          float64
	startTime        time.Time
	duration         int
	originalDuration int
	completed        bool
	puffinessPhase   float64
}

func NewCloudEffect(startX, startY int, duration int) *CloudEffect {
	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/cloud_poison0.png")
	return &CloudEffect{
		img:              vxImg,
		startX:           float64(startX),
		startY:           float64(startY),
		scale:            1.0,
		opacity:          0.8,
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		completed:        false,
		puffinessPhase:   0,
	}
}

func (c *CloudEffect) UpdateVisualEffect() {
	elapsed := time.Since(c.startTime).Seconds()

	// Update puffiness phase
	c.puffinessPhase += 0.05 // Adjust this value to change the speed of the effect

	// Simulate cloud puffiness by slightly changing scale
	c.scale = 1.0 + 0.05*math.Sin(c.puffinessPhase)

	// Simulate cloud density changes by adjusting opacity
	c.opacity = 0.7 + 0.2*math.Sin(c.puffinessPhase*0.7)

	// Check if the effect has lasted for the specified duration
	if int(elapsed) >= c.duration {
		c.completed = true
	}
}

func (c *CloudEffect) DrawVisualEffect(screen *ebiten.Image) {
	bounds := c.img.Bounds()
	imgWidth := float64(bounds.Dx())
	imgHeight := float64(bounds.Dy())

	// Draw the main cloud image
	opts := &ebiten.DrawImageOptions{}

	// Set the scale for the puffiness effect
	opts.GeoM.Scale(c.scale, c.scale)

	// Adjust position to keep the center of the cloud in place
	adjustedX := c.startX - (imgWidth * (c.scale - 1) / 2)
	adjustedY := c.startY - (imgHeight * (c.scale - 1) / 2)

	// Set the adjusted position of the cloud effect
	opts.GeoM.Translate(adjustedX, adjustedY)

	// Apply the opacity to simulate density changes
	opts.ColorM.Scale(1, 1, 1, c.opacity)

	screen.DrawImage(c.img, opts)

	// Create a subtle "fluffiness" effect by drawing multiple layers
	for i := 0; i < 2; i++ { // Reduced from 3 to 2 layers for less expansion
		layerOpts := &ebiten.DrawImageOptions{}
		layerScale := c.scale * (1.0 - float64(i)*0.1) // Slightly smaller for each layer
		layerOpts.GeoM.Scale(layerScale, layerScale)

		// Adjust position for each layer
		layerAdjustedX := c.startX - (imgWidth * (layerScale - 1) / 2)
		layerAdjustedY := c.startY - (imgHeight * (layerScale - 1) / 2)
		layerOpts.GeoM.Translate(layerAdjustedX, layerAdjustedY)

		layerOpts.ColorM.Scale(1, 1, 1, 0.3*c.opacity) // Adjust opacity for each layer
		screen.DrawImage(c.img, layerOpts)
	}
}

func (c *CloudEffect) IsCompleted() bool {
	return c.completed
}

func (c *CloudEffect) SetVXCommon(x, y int, img *ebiten.Image) {
	c.startX = float64(x)
	c.startY = float64(y)
	c.img = img
}

func (c *CloudEffect) VXImg() *ebiten.Image {
	return c.img
}

// Projectile does not have a duration but we still need the function to implement the interface
func (c *CloudEffect) ResetVX() {

	c.startTime = time.Now()
	c.completed = false
	c.duration = c.originalDuration

}

func (c *CloudEffect) Copy() VisualEffect {
	return &CloudEffect{
		img:              c.img,
		startX:           c.startX,
		startY:           c.startY,
		scale:            c.scale,
		opacity:          c.opacity,
		startTime:        c.startTime,
		duration:         c.duration,
		originalDuration: c.duration,
		completed:        c.completed,
		puffinessPhase:   c.puffinessPhase,
	}
}

type ElectricityEffectNoImage struct {
	startX, startY   float64
	segments         []lineSegment
	color            color.RGBA
	startTime        time.Time
	duration         int
	originalDuration int
	completed        bool
}

type lineSegment struct {
	x1, y1, x2, y2 float64
}

func NewElectricityEffectNoImage(startX, startY int, duration int, numSegments int) *ElectricityEffectNoImage {
	segments := make([]lineSegment, numSegments)
	currentX, currentY := float64(startX), float64(startY)

	for i := 0; i < numSegments; i++ {
		nextX := currentX + -10 + 20*rand.Float64() // Random horizontal displacement
		nextY := currentY + -10 + 20*rand.Float64() // Random vertical displacement

		segments[i] = lineSegment{
			x1: currentX,
			y1: currentY,
			x2: nextX,
			y2: nextY,
		}

		currentX, currentY = nextX, nextY
	}

	// Random bright color for the electricity
	color := color.RGBA{
		R: uint8(200 + randgen.GetDiceRoll(55)),
		G: uint8(200 + randgen.GetDiceRoll(55)),
		B: 255,
		A: 255,
	}

	return &ElectricityEffectNoImage{
		startX:           float64(startX),
		startY:           float64(startY),
		segments:         segments,
		color:            color,
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		completed:        false,
	}
}
func (elec *ElectricityEffectNoImage) UpdateVisualEffect() {
	elapsed := time.Since(elec.startTime).Seconds()

	// Regenerate line segments to simulate flickering
	for i := range elec.segments {
		elec.segments[i].x2 += -5 + 10*rand.Float64() // Jitter x
		elec.segments[i].y2 += -5 + 10*rand.Float64() // Jitter y
	}

	// Slightly adjust the color to simulate electrical surges
	elec.color.R = uint8(200 + randgen.GetDiceRoll(55))
	elec.color.G = uint8(200 + randgen.GetDiceRoll(55))
	elec.color.B = 255

	// Check if the effect has lasted for the specified duration
	if int(elapsed) >= elec.duration {
		elec.completed = true
	}
}

func (elec *ElectricityEffectNoImage) DrawVisualEffect(screen *ebiten.Image) {
	for _, segment := range elec.segments {
		ebitenutil.DrawLine(screen, segment.x1, segment.y1, segment.x2, segment.y2, elec.color)
	}
}

func (elec *ElectricityEffectNoImage) IsCompleted() bool {
	return elec.completed
}

func (elec *ElectricityEffectNoImage) SetVXCommon(x, y int, img *ebiten.Image) {
	elec.startX = float64(x)
	elec.startY = float64(y)
	//elec.img = img
}

func (elec *ElectricityEffectNoImage) VXImg() *ebiten.Image {
	return nil
}

// Projectile does not have a duration but we still need the function to implement the interface
func (elec *ElectricityEffectNoImage) ResetVX() {

	elec.startTime = time.Now()
	elec.completed = false
	elec.duration = elec.originalDuration

}

func (elec *ElectricityEffectNoImage) Copy() VisualEffect {
	copiedSegments := make([]lineSegment, len(elec.segments))
	copy(copiedSegments, elec.segments)

	return &ElectricityEffectNoImage{
		startX:           elec.startX,
		startY:           elec.startY,
		segments:         copiedSegments,
		color:            elec.color,
		startTime:        elec.startTime,
		duration:         elec.duration,
		originalDuration: elec.originalDuration,
		completed:        elec.completed,
	}
}

type ElectricityEffect struct {
	img              *ebiten.Image // Your electricity image
	startX, startY   float64
	scale            float64
	brightness       float64
	startTime        time.Time
	duration         int
	originalDuration int
	completed        bool
}

func NewElectricityEffect(startX, startY int, duration int) *ElectricityEffect {
	vxImg, _, _ := ebitenutil.NewImageFromFile("../assets/effects/zap0.png") // Load your electricity image here
	return &ElectricityEffect{
		img:              vxImg,
		startX:           float64(startX),
		startY:           float64(startY),
		scale:            1.0, // Initial scale
		brightness:       1.0, // Initial brightness
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		completed:        false,
	}
}

func (elec *ElectricityEffect) UpdateVisualEffect() {
	elapsed := time.Since(elec.startTime).Seconds()

	// Randomly change the scale to simulate rapid flickering arcs
	elec.scale = 0.9 + 0.2*rand.Float64()

	// Randomly adjust brightness to simulate electrical surges
	elec.brightness = 1.0 + 0.5*rand.Float64()

	// Jitter the position slightly to simulate erratic movement
	elec.startX += -1.0 + 2.0*rand.Float64()
	elec.startY += -1.0 + 2.0*rand.Float64()

	// Check if the effect has lasted for the specified duration
	if int(elapsed) >= elec.duration {
		elec.completed = true
	}
}

func (elec *ElectricityEffect) DrawVisualEffect(screen *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}

	// Set the scale for the flickering effect
	opts.GeoM.Scale(elec.scale, elec.scale)

	// Set the position of the electricity effect
	opts.GeoM.Translate(elec.startX, elec.startY)

	// Apply the brightness (color modulation)
	opts.ColorM.Scale(elec.brightness, elec.brightness, elec.brightness, 1.0)

	// Draw the electricity image with the flickering effect
	screen.DrawImage(elec.img, opts)
}

func (elec *ElectricityEffect) IsCompleted() bool {
	return elec.completed
}

func (elec *ElectricityEffect) SetVXCommon(x, y int, img *ebiten.Image) {
	elec.startX = float64(x)
	elec.startY = float64(y)
	elec.img = img
}

func (elec *ElectricityEffect) VXImg() *ebiten.Image {
	return elec.img
}

// Projectile does not have a duration but we still need the function to implement the interface
func (elec *ElectricityEffect) ResetVX() {

	elec.startTime = time.Now()
	elec.completed = false
	elec.duration = elec.originalDuration

}

func (elec *ElectricityEffect) Copy() VisualEffect {
	return &ElectricityEffect{
		img:              elec.img,
		startX:           elec.startX,
		startY:           elec.startY,
		scale:            elec.scale,
		brightness:       elec.brightness,
		startTime:        elec.startTime,
		duration:         elec.duration,
		originalDuration: elec.duration,
		completed:        elec.completed,
	}
}

// Todo does not work fix later
type ElectricArc struct {
	startX, startY   float64
	endX, endY       float64
	segments         [][]float64
	color            color.RGBA
	thickness        float32
	startTime        time.Time
	duration         int
	originalDuration int
	completed        bool
}

func NewElectricArc(startX, startY, endX, endY int, duration int) *ElectricArc {
	return &ElectricArc{
		startX:           float64(startX),
		startY:           float64(startY),
		endX:             float64(endX),
		endY:             float64(endY),
		segments:         make([][]float64, 0),
		color:            color.RGBA{0, 191, 255, 255}, // Light blue color
		thickness:        2,
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		completed:        false,
	}
}

func (e *ElectricArc) UpdateVisualEffect() {
	elapsed := time.Since(e.startTime).Seconds()

	// Generate new segments for the electricity
	e.generateSegments()

	// Randomly adjust color and thickness for visual variety

	e.color.R = uint8(randgen.GetDiceRoll(50))
	e.color.G = uint8(200 + randgen.GetDiceRoll(55))
	e.color.B = uint8(200 + randgen.GetDiceRoll(55))
	e.thickness = float32(1.5 + rand.Float32())

	// Check if the effect has lasted for the specified duration
	if int(elapsed) >= e.duration {
		e.completed = true
	}
}

func (e *ElectricArc) DrawVisualEffect(screen *ebiten.Image) {
	for i := 0; i < len(e.segments)-1; i++ {
		vector.StrokeLine(screen, float32(e.segments[i][0]), float32(e.segments[i][1]),
			float32(e.segments[i+1][0]), float32(e.segments[i+1][1]),
			e.thickness, e.color, false)
	}
}

func (e *ElectricArc) generateSegments() {
	e.segments = make([][]float64, 0)
	e.segments = append(e.segments, []float64{e.startX, e.startY})

	currentX, currentY := e.startX, e.startY
	for i := 0; i < 10; i++ { // Adjust the number of segments as needed
		nextX := currentX + (e.endX-currentX)/float64(10-i) + (rand.Float64()-0.5)*20
		nextY := currentY + (e.endY-currentY)/float64(10-i) + (rand.Float64()-0.5)*20
		e.segments = append(e.segments, []float64{nextX, nextY})
		currentX, currentY = nextX, nextY
	}

	e.segments = append(e.segments, []float64{e.endX, e.endY})
}

func (e *ElectricArc) IsCompleted() bool {
	return e.completed
}

func (e *ElectricArc) SetVXCommon(x, y int, img *ebiten.Image) {
	// This effect doesn't use an image, so we'll just update the start position
	e.startX = float64(x)
	e.startY = float64(y)
}

func (e *ElectricArc) VXImg() *ebiten.Image {
	// This effect doesn't use an image, so we return nil
	return nil
}

// Projectile does not have a duration but we still need the function to implement the interface
func (e *ElectricArc) ResetVX() {

	e.startTime = time.Now()
	e.completed = false
	e.duration = e.originalDuration

}

func (e *ElectricArc) Copy() VisualEffect {
	return &ElectricArc{
		startX:           e.startX,
		startY:           e.startY,
		endX:             e.endX,
		endY:             e.endY,
		segments:         make([][]float64, len(e.segments)),
		color:            e.color,
		thickness:        e.thickness,
		startTime:        e.startTime,
		duration:         e.duration,
		originalDuration: e.duration,
		completed:        e.completed,
	}
}

type StickyGroundEffect struct {
	startX, startY   float64
	scale            float64
	opacity          float64
	startTime        time.Time
	duration         int
	originalDuration int
	completed        bool
	waveOffset       float64     // For slow-moving "sticky" animation
	color            color.Color // The base color for the sticky effect
}

// Constructor for StickyGroundEffect
func NewStickyGroundEffect(startX, startY int, duration int) *StickyGroundEffect {
	return &StickyGroundEffect{
		startX:           float64(startX),
		startY:           float64(startY),
		scale:            1.0, // Initial scale
		opacity:          1.0, // Initial opacity
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		completed:        false,
		waveOffset:       0.0,
		color:            color.RGBA{0x90, 0xEE, 0x90, 0xFF}, // Example: dark green for a sticky effect
	}
}

// Update logic for StickyGroundEffect
func (s *StickyGroundEffect) UpdateVisualEffect() {
	elapsed := time.Since(s.startTime).Seconds()

	// Slowly move the "sticky" shapes to simulate gooey movement
	s.waveOffset += 0.01

	// Slight modulation of opacity to simulate depth or thickness
	s.opacity = 0.8 + 0.2*math.Sin(s.waveOffset)

	// Check if the effect has lasted for the specified duration
	if int(elapsed) >= s.duration {
		s.completed = true
	}
}

// Draw logic for StickyGroundEffect
func (s *StickyGroundEffect) DrawVisualEffect(screen *ebiten.Image) {
	// Create a color-based sticky ground effect
	for i := 0; i < 5; i++ {
		opts := &ebiten.DrawImageOptions{}

		// Create some basic shapes to represent the sticky ground
		radius := 10 + 5*math.Sin(s.waveOffset+float64(i)) // Vary the radius slightly
		x := s.startX + 20*math.Cos(float64(i)+s.waveOffset)
		y := s.startY + 20*math.Sin(float64(i)+s.waveOffset)

		// Generate an offscreen image to represent the shape (circle here)
		circleImage := ebiten.NewImage(int(2*radius), int(2*radius))
		circleImage.Fill(s.color)

		// Apply scaling and position transformations
		opts.GeoM.Translate(-radius, -radius) // Center the circle
		opts.GeoM.Translate(x, y)
		opts.GeoM.Scale(s.scale, s.scale)

		// Apply opacity modulation
		opts.ColorM.Scale(1, 1, 1, s.opacity)

		// Draw the shape on the screen
		screen.DrawImage(circleImage, opts)
	}
}

// Copy method for StickyGroundEffect
func (s *StickyGroundEffect) Copy() VisualEffect {
	return &StickyGroundEffect{
		startX:           s.startX,
		startY:           s.startY,
		scale:            s.scale,
		opacity:          s.opacity,
		startTime:        s.startTime,
		duration:         s.duration,
		originalDuration: s.originalDuration,
		completed:        s.completed,
		waveOffset:       s.waveOffset,
		color:            s.color,
	}
}

// Other required interface methods
func (s *StickyGroundEffect) IsCompleted() bool {
	return s.completed
}

func (s *StickyGroundEffect) SetVXCommon(x, y int, img *ebiten.Image) {
	s.startX = float64(x)
	s.startY = float64(y)
	// img is unused since we're using color-based shapes
}

// Projectile does not have a duration but we still need the function to implement the interface
func (s *StickyGroundEffect) ResetVX() {

	s.startTime = time.Now()
	s.completed = false
	s.duration = s.originalDuration

}

func (s *StickyGroundEffect) VXImg() *ebiten.Image {
	return nil // No image since we're drawing directly with colors
}
