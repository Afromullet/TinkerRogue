package behavior

import (
	"game_main/common"
	"game_main/visual/graphics"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
)

// LayerMode represents which threat layer to visualize
type LayerMode int

const (
	// LayerMelee shows melee threat zones (orange gradient)
	LayerMelee LayerMode = iota
	// LayerRanged shows ranged fire zones (cyan gradient)
	LayerRanged
	// LayerSupport shows support value zones (green gradient)
	LayerSupport
	// LayerPositionalFlanking shows flanking exposure (yellow gradient)
	LayerPositionalFlanking
	// LayerPositionalIsolation shows isolation risk (purple gradient)
	LayerPositionalIsolation
	// LayerPositionalEngagement shows engagement pressure (red-orange gradient)
	LayerPositionalEngagement
	// LayerPositionalRetreat shows retreat quality (green gradient, high=good)
	LayerPositionalRetreat
	// LayerModeCount is the total number of layer modes
	LayerModeCount
)

// LayerModeInfo provides display metadata for each layer mode
type LayerModeInfo struct {
	Name        string
	Description string
	ColorKey    string
}

// LayerModeMetadata maps each mode to its display information
var LayerModeMetadata = map[LayerMode]LayerModeInfo{
	LayerMelee: {
		Name:        "Melee Threat",
		Description: "Enemy melee engagement zones",
		ColorKey:    "Orange (low -> high)",
	},
	LayerRanged: {
		Name:        "Ranged Fire",
		Description: "Enemy ranged attack zones",
		ColorKey:    "Cyan (low -> high)",
	},
	LayerSupport: {
		Name:        "Support Value",
		Description: "Healing/buff priority zones",
		ColorKey:    "Green (low -> high)",
	},
	LayerPositionalFlanking: {
		Name:        "Flanking Risk",
		Description: "Multi-directional threat exposure",
		ColorKey:    "Yellow (safe -> flanked)",
	},
	LayerPositionalIsolation: {
		Name:        "Isolation Risk",
		Description: "Distance from ally support",
		ColorKey:    "Purple (safe -> isolated)",
	},
	LayerPositionalEngagement: {
		Name:        "Engagement Pressure",
		Description: "Combined damage exposure",
		ColorKey:    "Red-Orange (low -> high)",
	},
	LayerPositionalRetreat: {
		Name:        "Retreat Quality",
		Description: "Escape route availability",
		ColorKey:    "Red-Green (trapped -> safe)",
	},
}

// LayerVisualizer handles visualization of individual threat layers
type LayerVisualizer struct {
	manager         *common.EntityManager
	gameMap         *worldmap.GameMap
	threatEvaluator *CompositeThreatEvaluator

	// State
	isActive         bool
	currentMode      LayerMode
	lastUpdateRound  int
	currentFactionID ecs.EntityID
}

// NewLayerVisualizer creates a new layer visualizer
func NewLayerVisualizer(
	manager *common.EntityManager,
	gameMap *worldmap.GameMap,
	threatEvaluator *CompositeThreatEvaluator,
) *LayerVisualizer {
	return &LayerVisualizer{
		manager:         manager,
		gameMap:         gameMap,
		threatEvaluator: threatEvaluator,
		isActive:        false,
		currentMode:     LayerMelee, // Start with melee layer
		lastUpdateRound: -1,
	}
}

// Toggle enables/disables the layer visualization
func (lv *LayerVisualizer) Toggle() {
	lv.isActive = !lv.isActive
	if !lv.isActive {
		lv.ClearVisualization()
	}
}

// IsActive returns whether visualization is currently enabled
func (lv *LayerVisualizer) IsActive() bool {
	return lv.isActive
}

// CycleMode advances to next layer mode
func (lv *LayerVisualizer) CycleMode() {
	lv.currentMode = (lv.currentMode + 1) % LayerModeCount
	lv.lastUpdateRound = -1 // Force recalculation
	if lv.isActive {
		lv.ClearVisualization()
	}
}

// GetCurrentMode returns the active layer mode
func (lv *LayerVisualizer) GetCurrentMode() LayerMode {
	return lv.currentMode
}

// GetCurrentModeInfo returns display metadata for current mode
func (lv *LayerVisualizer) GetCurrentModeInfo() LayerModeInfo {
	return LayerModeMetadata[lv.currentMode]
}

// Update recalculates and applies layer visualization
// Parameters:
//   - currentFactionID: The faction whose turn it is
//   - currentRound: Combat round number (for caching)
//   - playerPos: Center of viewport for visible tile calculation
//   - viewportSize: Size of visible area
func (lv *LayerVisualizer) Update(
	currentFactionID ecs.EntityID,
	currentRound int,
	playerPos coords.LogicalPosition,
	viewportSize int,
) {
	if !lv.isActive {
		return
	}

	// Only recalculate if round changed
	if lv.lastUpdateRound == currentRound {
		return
	}

	lv.lastUpdateRound = currentRound
	lv.currentFactionID = currentFactionID

	// Ensure threat evaluator is up-to-date
	lv.threatEvaluator.Update(currentRound)

	// Calculate visible tile bounds (30x30 viewport optimization)
	minX := playerPos.X - viewportSize/2
	maxX := playerPos.X + viewportSize/2
	minY := playerPos.Y - viewportSize/2
	maxY := playerPos.Y + viewportSize/2

	// Visualize based on current mode
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			tilePos := coords.LogicalPosition{X: x, Y: y}

			if !lv.gameMap.InBounds(x, y) {
				continue
			}

			// Get threat value for current mode
			value := lv.getThreatValueAt(tilePos)

			// Apply color matrix
			colorMatrix := lv.valueToColorMatrix(value)
			tileIdx := coords.CoordManager.LogicalToIndex(tilePos)
			lv.gameMap.ApplyColorMatrixToIndex(tileIdx, colorMatrix)
		}
	}
}

// getThreatValueAt returns threat value for current mode at position
func (lv *LayerVisualizer) getThreatValueAt(pos coords.LogicalPosition) float64 {
	switch lv.currentMode {
	case LayerMelee:
		return lv.threatEvaluator.GetMeleeLayer().GetMeleeThreatAt(pos)
	case LayerRanged:
		return lv.threatEvaluator.GetRangedLayer().GetRangedPressureAt(pos)
	case LayerSupport:
		return lv.threatEvaluator.GetSupportLayer().GetSupportValueAt(pos)
	case LayerPositionalFlanking:
		return lv.threatEvaluator.GetPositionalLayer().GetFlankingRiskAt(pos)
	case LayerPositionalIsolation:
		return lv.threatEvaluator.GetPositionalLayer().GetIsolationRiskAt(pos)
	case LayerPositionalEngagement:
		return lv.threatEvaluator.GetPositionalLayer().GetEngagementPressureAt(pos)
	case LayerPositionalRetreat:
		return lv.threatEvaluator.GetPositionalLayer().GetRetreatQuality(pos)
	default:
		return 0.0
	}
}

// valueToColorMatrix converts threat value to color gradient
func (lv *LayerVisualizer) valueToColorMatrix(value float64) graphics.ColorMatrix {
	if value == 0 {
		return graphics.NewEmptyMatrix()
	}

	// Get gradient function for current mode
	gradientFunc := lv.getGradientFunction()

	// Apply intensity thresholds (normalized 0-1 range)
	var opacity float32
	if value <= 0.25 {
		opacity = 0.2
	} else if value <= 0.5 {
		opacity = 0.5
	} else if value <= 0.75 {
		opacity = 0.7
	} else {
		opacity = 0.9
	}

	return gradientFunc(opacity)
}

// getGradientFunction returns color gradient for current mode
func (lv *LayerVisualizer) getGradientFunction() func(float32) graphics.ColorMatrix {
	switch lv.currentMode {
	case LayerMelee:
		return graphics.CreateOrangeGradient
	case LayerRanged:
		return graphics.CreateCyanGradient
	case LayerSupport:
		return graphics.CreateGreenGradient
	case LayerPositionalFlanking:
		return graphics.CreateYellowGradient
	case LayerPositionalIsolation:
		return graphics.CreatePurpleGradient
	case LayerPositionalEngagement:
		return graphics.CreateRedOrangeGradient
	case LayerPositionalRetreat:
		// Green for good retreat paths (high value = good)
		return graphics.CreateGreenGradient
	default:
		return graphics.CreateRedGradient
	}
}

// ClearVisualization removes all colors from the map
func (lv *LayerVisualizer) ClearVisualization() {
	for i := 0; i < lv.gameMap.NumTiles; i++ {
		lv.gameMap.ApplyColorMatrixToIndex(i, graphics.NewEmptyMatrix())
	}
	lv.lastUpdateRound = -1
}
