package vfx

import (
	"game_main/core/config"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// EffectConfig defines the configuration for creating a visual effect.
type EffectConfig struct {
	ImagePath string   // "" = no image (procedural/line effects)
	Animator  Animator // nil = no animation
	Renderer  Renderer // required
}

// NewEffect creates a VisualEffect from a configuration.
func NewEffect(startX, startY, duration int, cfg EffectConfig) VisualEffect {
	var img *ebiten.Image
	if cfg.ImagePath != "" {
		img, _, _ = ebitenutil.NewImageFromFile(cfg.ImagePath)
	}
	return &BaseEffect{
		startX:           float64(startX),
		startY:           float64(startY),
		startTime:        time.Now(),
		duration:         duration,
		originalDuration: duration,
		img:              img,
		animator:         cfg.Animator,
		renderer:         cfg.Renderer,
	}
}

// NewFireEffect creates a fire effect with flickering animation.
func NewFireEffect(startX, startY, duration int) VisualEffect {
	return NewEffect(startX, startY, duration, EffectConfig{
		ImagePath: config.AssetPath("effects/cloud_fire2.png"),
		Animator: &RandomAnimator{
			ScaleRange:     [2]float64{0.95, 1.05},
			OpacityRange:   [2]float64{0.7, 1.0},
			JitterPos:      true,
			JitterAmount:   0.5,
			JitterInterval: 5,
		},
		Renderer: &ImageRenderer{},
	})
}

// NewIceEffect creates an ice effect with random shimmering.
func NewIceEffect(startX, startY int, duration int) VisualEffect {
	return NewEffect(startX, startY, duration, EffectConfig{
		ImagePath: config.AssetPath("effects/frost0.png"),
		Animator: &RandomAnimator{
			ScaleRange:      [2]float64{0.98, 1.02},
			OpacityRange:    [2]float64{0.85, 1.0},
			ColorShiftRange: [2]float64{0.9, 1.0},
		},
		Renderer: &ImageRenderer{},
	})
}

// NewCloudEffect creates a cloud effect with pulsing animation.
func NewCloudEffect(startX, startY int, duration int) VisualEffect {
	return NewEffect(startX, startY, duration, EffectConfig{
		ImagePath: config.AssetPath("effects/cloud_poison0.png"),
		Animator: &PulseAnimator{
			scaleBase:   1.0,
			scaleAmp:    0.05,
			opacityBase: 0.7,
			opacityAmp:  0.2,
			phaseSpeed:  0.05,
		},
		Renderer: &CloudRenderer{},
	})
}

// NewElectricityEffect creates an image-based electricity effect with brightness flickering.
func NewElectricityEffect(startX, startY int, duration int) VisualEffect {
	return NewEffect(startX, startY, duration, EffectConfig{
		ImagePath: config.AssetPath("effects/zap0.png"),
		Animator: &RandomAnimator{
			ScaleRange:      [2]float64{0.9, 1.1},
			BrightnessRange: [2]float64{1.0, 1.5},
			JitterPos:       true,
			JitterAmount:    1.0,
		},
		Renderer: &ImageRenderer{},
	})
}

// CreateVisualEffectByType creates a visual effect by type name string.
// Decouples callers from needing to know the specific effect constructors.
func CreateVisualEffectByType(vxType string, x, y, duration int) VisualEffect {
	switch vxType {
	case "fire":
		return NewFireEffect(x, y, duration)
	case "ice":
		return NewIceEffect(x, y, duration)
	case "electricity":
		return NewElectricityEffect(x, y, duration)
	case "cloud":
		return NewCloudEffect(x, y, duration)
	default:
		return NewFireEffect(x, y, duration)
	}
}
