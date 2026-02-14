package graphics

import (
	"game_main/world/coords"

	"github.com/hajimehoshi/ebiten/v2"
)

var VXHandler VisualEffectHandler

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
// to draw whatever effect is in the list.
type VisualEffectHandler struct {
	vx     []VisualEffect
	vxArea []VisualEffectArea
}

// Modifying the global VXHandler
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
