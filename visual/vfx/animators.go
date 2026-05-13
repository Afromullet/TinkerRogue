package vfx

import (
	"game_main/core/common"
	"math"
)

// RandomAnimator consolidates random-range animation behavior.
// It replaces FlickerAnimator, BrightnessFlickerAnimator, and ShimmerAnimator.
// Zero-value ranges are skipped (Scale/Opacity default to 1.0).
type RandomAnimator struct {
	ScaleRange      [2]float64 // min, max scale (zero = default 1.0)
	OpacityRange    [2]float64 // min, max opacity (zero = default 1.0)
	BrightnessRange [2]float64 // min, max brightness (zero = not used)
	ColorShiftRange [2]float64 // min, max color shift (zero = not used)
	JitterPos       bool       // whether to jitter position
	JitterAmount    float64    // defaults to 0.5 if JitterPos && zero
	JitterInterval  int        // defaults to 1 (every frame) if zero
	flickerTimer    int
}

func (a *RandomAnimator) Update(effect *BaseEffect, elapsed float64) AnimationState {
	a.flickerTimer++

	state := AnimationState{
		Scale:   1.0,
		Opacity: 1.0,
	}

	if a.ScaleRange[0] != 0 || a.ScaleRange[1] != 0 {
		state.Scale = a.ScaleRange[0] + (a.ScaleRange[1]-a.ScaleRange[0])*common.RandomFloat()
	}

	if a.OpacityRange[0] != 0 || a.OpacityRange[1] != 0 {
		state.Opacity = a.OpacityRange[0] + (a.OpacityRange[1]-a.OpacityRange[0])*common.RandomFloat()
	}

	if a.BrightnessRange[0] != 0 || a.BrightnessRange[1] != 0 {
		state.Brightness = a.BrightnessRange[0] + (a.BrightnessRange[1]-a.BrightnessRange[0])*common.RandomFloat()
	}

	if a.ColorShiftRange[0] != 0 || a.ColorShiftRange[1] != 0 {
		state.ColorShift = a.ColorShiftRange[0] + (a.ColorShiftRange[1]-a.ColorShiftRange[0])*common.RandomFloat()
	}

	if a.JitterPos {
		jitterAmount := a.JitterAmount
		if jitterAmount == 0 {
			jitterAmount = 0.5
		}
		jitterInterval := a.JitterInterval
		if jitterInterval == 0 {
			jitterInterval = 1
		}
		if a.flickerTimer%jitterInterval == 0 {
			state.OffsetX = -jitterAmount + 2*jitterAmount*common.RandomFloat()
			state.OffsetY = -jitterAmount + 2*jitterAmount*common.RandomFloat()
		}
	}

	return state
}

func (a *RandomAnimator) Reset() {
	a.flickerTimer = 0
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
