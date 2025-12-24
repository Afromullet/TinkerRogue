package behavior

import (
	"game_main/combat"
	"game_main/common"
	"game_main/coords"
	"game_main/graphics"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

// Exists for debugging
// ThreatViewMode represents which faction's threats to visualize
type ThreatViewMode int

const (
	// ViewEnemyThreats shows danger from enemy squads to the current faction
	ViewEnemyThreats ThreatViewMode = iota
	// ViewPlayerThreats shows danger projection from player's own squads
	ViewPlayerThreats
)

// ThreatMetric represents which metric to visualize
type ThreatMetric int

const (
	// MetricDanger shows heuristic threat (role/composition bonuses)
	MetricDanger ThreatMetric = iota
	// MetricExpectedDamage shows actual damage output (uses combat formulas)
	MetricExpectedDamage
)

// DangerVisualizer handles visualization of squad threat levels on the map
type DangerVisualizer struct {
	manager         *common.EntityManager
	gameMap         *worldmap.GameMap
	threatManager   *FactionThreatLevelManager
	isActive        bool
	viewMode        ThreatViewMode
	metricMode      ThreatMetric // Which metric to visualize (danger or expected damage)
	lastUpdateRound int          // Avoid recalculating every frame
}

// squadThreatInfo caches squad position and threat data to avoid repeated ECS queries
type squadThreatInfo struct {
	position              coords.LogicalPosition
	dangerByRange         map[int]float64
	expectedDamageByRange map[int]float64
}

// NewDangerVisualizer creates a new danger visualizer
func NewDangerVisualizer(manager *common.EntityManager, gameMap *worldmap.GameMap, threatManager *FactionThreatLevelManager) *DangerVisualizer {
	return &DangerVisualizer{
		manager:         manager,
		gameMap:         gameMap,
		threatManager:   threatManager,
		isActive:        false,
		viewMode:        ViewEnemyThreats, // Default to showing enemy threats
		metricMode:      MetricDanger,     // Default to danger metric
		lastUpdateRound: -1,
	}
}

// Toggle enables/disables the danger visualization
func (dv *DangerVisualizer) Toggle() {
	dv.isActive = !dv.isActive
	if !dv.isActive {
		dv.ClearVisualization()
	}
}

// IsActive returns whether visualization is currently enabled
func (dv *DangerVisualizer) IsActive() bool {
	return dv.isActive
}

// SwitchView toggles between enemy threat view and player threat view
func (dv *DangerVisualizer) SwitchView() {
	if dv.viewMode == ViewEnemyThreats {
		dv.viewMode = ViewPlayerThreats
	} else {
		dv.viewMode = ViewEnemyThreats
	}
	// Force recalculation on next update
	dv.lastUpdateRound = -1
	// If active, immediately update with new view
	if dv.isActive {
		dv.ClearVisualization()
	}
}

// GetViewMode returns the current threat view mode
func (dv *DangerVisualizer) GetViewMode() ThreatViewMode {
	return dv.viewMode
}

// CycleMetric toggles between danger and expected damage visualization
func (dv *DangerVisualizer) CycleMetric() {
	if dv.metricMode == MetricDanger {
		dv.metricMode = MetricExpectedDamage
	} else {
		dv.metricMode = MetricDanger
	}
	// Force recalculation on next update
	dv.lastUpdateRound = -1
	// If active, immediately update with new metric
	if dv.isActive {
		dv.ClearVisualization()
	}
}

// GetMetricMode returns the current metric mode
func (dv *DangerVisualizer) GetMetricMode() ThreatMetric {
	return dv.metricMode
}

// Update recalculates and applies danger visualization
// Parameters:
//   - currentFactionID: The faction whose turn it is
//   - currentRound: Combat round number (for caching)
//   - playerPos: Center of viewport for visible tile calculation
//   - viewportSize: Size of visible area
func (dv *DangerVisualizer) Update(currentFactionID ecs.EntityID, currentRound int, playerPos coords.LogicalPosition, viewportSize int) {
	if !dv.isActive {
		return
	}

	// Only recalculate if round changed (squad positions/stats may have changed)
	if dv.lastUpdateRound == currentRound {
		return
	}

	dv.lastUpdateRound = currentRound

	// Get squads to visualize based on current view mode
	var relevantSquads []ecs.EntityID
	if dv.viewMode == ViewEnemyThreats {
		relevantSquads = dv.getEnemySquads(currentFactionID)
	} else {
		relevantSquads = dv.getPlayerSquads(currentFactionID)
	}

	if len(relevantSquads) == 0 {
		dv.ClearVisualization()
		return
	}

	// Cache squad positions and threat data (avoid repeated ECS queries)
	squadThreats := make([]squadThreatInfo, 0, len(relevantSquads))
	for _, squadID := range relevantSquads {
		pos, err := combat.GetSquadMapPosition(squadID, dv.manager)
		if err != nil {
			continue
		}

		factionID := combat.GetSquadFaction(squadID, dv.manager)
		if factionID == 0 {
			continue
		}

		factionThreat := dv.threatManager.factions[factionID]
		if factionThreat == nil {
			continue
		}

		squadThreat := factionThreat.squadDangerLevel[squadID]
		if squadThreat == nil {
			continue
		}

		squadThreats = append(squadThreats, squadThreatInfo{
			position:              pos,
			dangerByRange:         squadThreat.DangerByRange,
			expectedDamageByRange: squadThreat.ExpectedDamageByRange,
		})
	}

	// Calculate visible tile bounds (only process tiles player can see)
	minX := playerPos.X - viewportSize/2
	maxX := playerPos.X + viewportSize/2
	minY := playerPos.Y - viewportSize/2
	maxY := playerPos.Y + viewportSize/2

	// Calculate danger for each visible tile
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			tilePos := coords.LogicalPosition{X: x, Y: y}

			// Check map bounds
			if !dv.gameMap.InBounds(x, y) {
				continue
			}

			// Calculate metric value at this tile (danger or expected damage)
			totalValue := dv.calculateTileValue(tilePos, squadThreats)

			// Apply color matrix based on value
			colorMatrix := dv.valueToColorMatrix(totalValue)
			tileIdx := coords.CoordManager.LogicalToIndex(tilePos)
			dv.gameMap.ApplyColorMatrixToIndex(tileIdx, colorMatrix)
		}
	}
}

// calculateTileValue sums the selected metric from all squads at a tile position
func (dv *DangerVisualizer) calculateTileValue(tilePos coords.LogicalPosition, squadThreats []squadThreatInfo) float64 {
	totalValue := 0.0

	for _, squadThreat := range squadThreats {
		distance := tilePos.ManhattanDistance(&squadThreat.position)

		// Select metric based on current mode
		var valueMap map[int]float64
		if dv.metricMode == MetricDanger {
			valueMap = squadThreat.dangerByRange
		} else {
			valueMap = squadThreat.expectedDamageByRange
		}

		// Check if this distance has a value
		if value, exists := valueMap[distance]; exists {
			totalValue += value
		}
	}

	return totalValue
}

// valueToColorMatrix converts a metric value to a color gradient ColorMatrix
// Uses red for danger, blue for expected damage
func (dv *DangerVisualizer) valueToColorMatrix(value float64) graphics.ColorMatrix {
	if value == 0 {
		return graphics.NewEmptyMatrix()
	}

	// Select color gradient based on metric
	var createGradient func(float32) graphics.ColorMatrix
	if dv.metricMode == MetricDanger {
		createGradient = graphics.CreateRedGradient
	} else {
		createGradient = graphics.CreateBlueGradient
	}

	// Apply same thresholds for both metrics
	if value <= 50 {
		return createGradient(0.2) // Low
	} else if value <= 100 {
		return createGradient(0.5) // Medium
	} else if value <= 150 {
		return createGradient(0.7) // High
	} else {
		return createGradient(0.9) // Very high
	}
}

// ClearVisualization removes all danger colors from the map
func (dv *DangerVisualizer) ClearVisualization() {
	// Clear all tiles (could optimize to only clear previously colored tiles)
	for i := 0; i < dv.gameMap.NumTiles; i++ {
		dv.gameMap.ApplyColorMatrixToIndex(i, graphics.NewEmptyMatrix())
	}
	dv.lastUpdateRound = -1 // Force recalculation on next activation
}

// getEnemySquads returns all squads from enemy factions (not current faction)
func (dv *DangerVisualizer) getEnemySquads(currentFactionID ecs.EntityID) []ecs.EntityID {
	var enemySquads []ecs.EntityID

	// Get all factions
	allFactions := combat.GetAllFactions(dv.manager)

	for _, factionID := range allFactions {
		// Skip current faction (we want enemy threats)
		if factionID == currentFactionID {
			continue
		}

		// Get squads for this enemy faction
		squads := combat.GetSquadsForFaction(factionID, dv.manager)
		enemySquads = append(enemySquads, squads...)
	}

	return enemySquads
}

// getPlayerSquads returns all squads from the current faction
func (dv *DangerVisualizer) getPlayerSquads(currentFactionID ecs.EntityID) []ecs.EntityID {
	// Get squads for current faction only
	return combat.GetSquadsForFaction(currentFactionID, dv.manager)
}
