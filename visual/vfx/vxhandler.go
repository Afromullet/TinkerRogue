package vfx

import (
	"game_main/core/coords"

	"github.com/hajimehoshi/ebiten/v2"
)

var VXHandler VisualEffectHandler

// VisualEffectArea applies one effect to every tile in a set of indices.
// All copies share completion state — when the first is done, the whole area
// is treated as complete.
type VisualEffectArea struct {
	visEffects []VisualEffect
}

// NewVisualEffectArea stamps a copy of vx at each tile index, positioned at
// the appropriate screen coordinate relative to the given logical center.
// Callers compute the index list from their own shape/targeting logic before
// invoking this — vfx does not know about shapes, only positions.
func NewVisualEffectArea(centerX, centerY int, indices []int, vx VisualEffect) VisualEffectArea {
	visEffects := make([]VisualEffect, 0, len(indices))

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
	VXHandler.AddVisualEffectArea(a)
}

func (vis *VisualEffectHandler) AddVisualEffect(a VisualEffect) {
	vis.vx = append(vis.vx, a)
}

func (vis *VisualEffectHandler) AddVisualEffectArea(a VisualEffectArea) {
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
