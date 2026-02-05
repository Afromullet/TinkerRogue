package behavior

import (
	"game_main/common"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
)

// ThreatMetric represents which metric to visualize
// Currently only DangerByRange is available (ExpectedDamageByRange was removed)
type ThreatMetric int

const (
	// MetricDanger shows heuristic threat (role/composition bonuses)
	MetricDanger ThreatMetric = iota
)

// DangerVisualizer is a backward-compatible wrapper around ThreatVisualizer.
// Use ThreatVisualizer directly for new code.
type DangerVisualizer struct {
	*ThreatVisualizer
}

// NewDangerVisualizer creates a new danger visualizer (wrapper around ThreatVisualizer)
func NewDangerVisualizer(
	manager *common.EntityManager,
	gameMap *worldmap.GameMap,
	threatManager *FactionThreatLevelManager,
) *DangerVisualizer {
	return &DangerVisualizer{
		ThreatVisualizer: NewThreatVisualizer(manager, gameMap, threatManager, nil),
	}
}

// SwitchView toggles between enemy threat view and player threat view
// Kept for backward compatibility
func (dv *DangerVisualizer) SwitchView() {
	dv.ThreatVisualizer.SwitchThreatView()
}

// GetViewMode returns the current threat view mode
// Kept for backward compatibility
func (dv *DangerVisualizer) GetViewMode() ThreatViewMode {
	return dv.ThreatVisualizer.GetThreatViewMode()
}

// CycleMetric cycles through available metrics (currently only danger)
// Kept for API compatibility; no-op until additional metrics are added
func (dv *DangerVisualizer) CycleMetric() {
	// Only MetricDanger is available
	// This is a no-op but kept for API compatibility
}

// GetMetricMode returns the current metric mode
func (dv *DangerVisualizer) GetMetricMode() ThreatMetric {
	return MetricDanger
}

// Update delegates to the underlying ThreatVisualizer
func (dv *DangerVisualizer) Update(
	currentFactionID ecs.EntityID,
	currentRound int,
	playerPos coords.LogicalPosition,
	viewportSize int,
) {
	dv.ThreatVisualizer.mode = VisualizerModeDanger
	dv.ThreatVisualizer.Update(currentFactionID, currentRound, playerPos, viewportSize)
}
