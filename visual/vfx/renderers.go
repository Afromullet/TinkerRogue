package vfx

import (
	"game_main/core/coords"

	"github.com/hajimehoshi/ebiten/v2"
)

// ImageRenderer draws image-based effects
type ImageRenderer struct{}

func (r *ImageRenderer) Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState) {
	if effect.img == nil {
		return
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(state.Scale*float64(coords.ScreenInfo.ScaleFactor), state.Scale*float64(coords.ScreenInfo.ScaleFactor))
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
	opts.GeoM.Scale(state.Scale*float64(coords.ScreenInfo.ScaleFactor), state.Scale*float64(coords.ScreenInfo.ScaleFactor))

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
