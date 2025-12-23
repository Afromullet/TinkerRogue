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

// DangerVisualizer handles visualization of squad threat levels on the map
type DangerVisualizer struct {
	manager         *common.EntityManager
	gameMap         *worldmap.GameMap
	threatManager   *FactionThreatLevelManager
	isActive        bool
	viewMode        ThreatViewMode
	lastUpdateRound int // Avoid recalculating every frame
}

// squadThreatInfo caches squad position and threat data to avoid repeated ECS queries
type squadThreatInfo struct {
	position      coords.LogicalPosition
	dangerByRange map[int]float64
}

// NewDangerVisualizer creates a new danger visualizer
func NewDangerVisualizer(manager *common.EntityManager, gameMap *worldmap.GameMap) *DangerVisualizer {
	return &DangerVisualizer{
		manager:         manager,
		gameMap:         gameMap,
		threatManager:   ThreatLevelManager, // Use global instance
		isActive:        false,
		viewMode:        ViewEnemyThreats, // Default to showing enemy threats
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
		if squadThreat == nil || squadThreat.DangerByRange == nil {
			continue
		}

		squadThreats = append(squadThreats, squadThreatInfo{
			position:      pos,
			dangerByRange: squadThreat.DangerByRange,
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

			// Calculate total danger at this tile
			totalDanger := dv.calculateTileDanger(tilePos, squadThreats)

			// Apply color matrix based on danger level
			colorMatrix := dv.dangerToColorMatrix(totalDanger)
			tileIdx := coords.CoordManager.LogicalToIndex(tilePos)
			dv.gameMap.ApplyColorMatrixToIndex(tileIdx, colorMatrix)
		}
	}
}

// calculateTileDanger sums danger from all squads at a tile position
func (dv *DangerVisualizer) calculateTileDanger(tilePos coords.LogicalPosition, squadThreats []squadThreatInfo) float64 {
	totalDanger := 0.0

	for _, squadThreat := range squadThreats {
		distance := tilePos.ManhattanDistance(&squadThreat.position)

		// Check if this distance has a danger value
		if danger, exists := squadThreat.dangerByRange[distance]; exists {
			totalDanger += danger
		}
	}

	return totalDanger
}

// dangerToColorMatrix converts a danger value to a red gradient ColorMatrix
func (dv *DangerVisualizer) dangerToColorMatrix(danger float64) graphics.ColorMatrix {
	if danger == 0 {
		// No danger - no color
		return graphics.NewEmptyMatrix()
	} else if danger <= 50 {
		// Low danger - light red
		return graphics.CreateRedGradient(0.2)
	} else if danger <= 100 {
		// Medium danger - medium red
		return graphics.CreateRedGradient(0.5)
	} else if danger <= 150 {
		// High danger - strong red
		return graphics.CreateRedGradient(0.7)
	} else {
		// Very high danger - max red
		return graphics.CreateRedGradient(0.9)
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
