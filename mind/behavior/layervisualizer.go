package behavior

import (
	"game_main/common"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
)

// LayerVisualizer is a backward-compatible wrapper around ThreatVisualizer.
// Use ThreatVisualizer directly for new code.
type LayerVisualizer struct {
	*ThreatVisualizer
}

// NewLayerVisualizer creates a new layer visualizer (wrapper around ThreatVisualizer)
func NewLayerVisualizer(
	manager *common.EntityManager,
	gameMap *worldmap.GameMap,
	threatEvaluator *CompositeThreatEvaluator,
) *LayerVisualizer {
	// Get threat manager from evaluator (it's needed for the unified visualizer)
	// Note: The unified visualizer doesn't need threatManager for layer mode,
	// so we pass nil here
	return &LayerVisualizer{
		ThreatVisualizer: NewThreatVisualizer(manager, gameMap, nil, threatEvaluator),
	}
}

// CycleMode advances to next layer mode
// Kept for backward compatibility
func (lv *LayerVisualizer) CycleMode() {
	lv.ThreatVisualizer.CycleLayerMode()
}

// GetCurrentMode returns the active layer mode
// Kept for backward compatibility
func (lv *LayerVisualizer) GetCurrentMode() LayerMode {
	return lv.ThreatVisualizer.GetLayerMode()
}

// GetCurrentModeInfo returns display metadata for current mode
// Kept for backward compatibility
func (lv *LayerVisualizer) GetCurrentModeInfo() LayerModeInfo {
	return lv.ThreatVisualizer.GetLayerModeInfo()
}

// Update delegates to the underlying ThreatVisualizer
func (lv *LayerVisualizer) Update(
	currentFactionID ecs.EntityID,
	currentRound int,
	playerPos coords.LogicalPosition,
	viewportSize int,
) {
	lv.ThreatVisualizer.mode = VisualizerModeLayer
	lv.ThreatVisualizer.Update(currentFactionID, currentRound, playerPos, viewportSize)
}
