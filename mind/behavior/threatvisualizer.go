package behavior

import (
	"game_main/common"
	"game_main/mind/evaluation"
	"game_main/tactical/combat"
	"game_main/visual/graphics"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
)

// VisualizerMode represents the primary visualization mode
type VisualizerMode int

const (
	// VisualizerModeDanger shows danger projection from squads
	VisualizerModeDanger VisualizerMode = iota
	// VisualizerModeLayer shows individual threat layers
	VisualizerModeLayer
)

// ThreatViewMode represents which faction's threats to visualize (for danger mode)
type ThreatViewMode int

const (
	// ViewEnemyThreats shows danger from enemy squads to the current faction
	ViewEnemyThreats ThreatViewMode = iota
	// ViewPlayerThreats shows danger projection from player's own squads
	ViewPlayerThreats
)

// LayerMode represents which threat layer to visualize (for layer mode)
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

// ThreatVisualizer provides unified visualization for both danger projection and threat layers.
// Combines the functionality of the former DangerVisualizer and LayerVisualizer.
type ThreatVisualizer struct {
	// Dependencies
	manager         *common.EntityManager
	gameMap         *worldmap.GameMap
	threatManager   *FactionThreatLevelManager
	threatEvaluator *CompositeThreatEvaluator

	// State
	*evaluation.DirtyCache
	isActive        bool
	mode            VisualizerMode
	threatViewMode  ThreatViewMode // For danger mode: enemy vs player
	layerMode       LayerMode      // For layer mode: which layer
	currentFactionID ecs.EntityID
}

// NewThreatVisualizer creates a unified threat visualizer
func NewThreatVisualizer(
	manager *common.EntityManager,
	gameMap *worldmap.GameMap,
	threatManager *FactionThreatLevelManager,
	threatEvaluator *CompositeThreatEvaluator,
) *ThreatVisualizer {
	return &ThreatVisualizer{
		manager:         manager,
		gameMap:         gameMap,
		threatManager:   threatManager,
		threatEvaluator: threatEvaluator,
		DirtyCache:      evaluation.NewDirtyCache(),
		isActive:        false,
		mode:            VisualizerModeDanger,
		threatViewMode:  ViewEnemyThreats,
		layerMode:       LayerMelee,
	}
}

// =========================================
// Core API
// =========================================

// Toggle enables/disables the visualization
func (tv *ThreatVisualizer) Toggle() {
	tv.isActive = !tv.isActive
	if !tv.isActive {
		tv.ClearVisualization()
	}
}

// IsActive returns whether visualization is currently enabled
func (tv *ThreatVisualizer) IsActive() bool {
	return tv.isActive
}

// SetMode sets the primary visualization mode
func (tv *ThreatVisualizer) SetMode(mode VisualizerMode) {
	if tv.mode != mode {
		tv.mode = mode
		tv.MarkDirty()
		if tv.isActive {
			tv.ClearVisualization()
		}
	}
}

// GetMode returns the current visualization mode
func (tv *ThreatVisualizer) GetMode() VisualizerMode {
	return tv.mode
}

// ClearVisualization removes all colors from the map
func (tv *ThreatVisualizer) ClearVisualization() {
	for i := 0; i < tv.gameMap.NumTiles; i++ {
		tv.gameMap.ApplyColorMatrixToIndex(i, graphics.NewEmptyMatrix())
	}
	tv.MarkDirty()
}

// Update recalculates and applies visualization
func (tv *ThreatVisualizer) Update(
	currentFactionID ecs.EntityID,
	currentRound int,
	playerPos coords.LogicalPosition,
	viewportSize int,
) {
	if !tv.isActive {
		return
	}

	// Only recalculate if round changed
	if tv.IsValid(currentRound) {
		return
	}

	tv.currentFactionID = currentFactionID

	// Ensure threat evaluator is up-to-date (for layer mode)
	if tv.mode == VisualizerModeLayer && tv.threatEvaluator != nil {
		tv.threatEvaluator.Update(currentRound)
	}

	// Visualize based on current mode
	IterateViewport(playerPos, viewportSize, func(pos coords.LogicalPosition) {
		if !tv.gameMap.InBounds(pos.X, pos.Y) {
			return
		}

		var value float64
		var colorMatrix graphics.ColorMatrix

		switch tv.mode {
		case VisualizerModeDanger:
			value = tv.calculateDangerValue(pos, currentFactionID)
			colorMatrix = tv.dangerValueToColorMatrix(value)
		case VisualizerModeLayer:
			value = tv.getLayerValueAt(pos)
			colorMatrix = tv.layerValueToColorMatrix(value)
		}

		tileIdx := coords.CoordManager.LogicalToIndex(pos)
		tv.gameMap.ApplyColorMatrixToIndex(tileIdx, colorMatrix)
	})

	tv.MarkClean(currentRound)
}

// =========================================
// Danger Mode API (formerly DangerVisualizer)
// =========================================

// SwitchThreatView toggles between enemy threat view and player threat view
func (tv *ThreatVisualizer) SwitchThreatView() {
	if tv.threatViewMode == ViewEnemyThreats {
		tv.threatViewMode = ViewPlayerThreats
	} else {
		tv.threatViewMode = ViewEnemyThreats
	}
	tv.MarkDirty()
	if tv.isActive {
		tv.ClearVisualization()
	}
}

// GetThreatViewMode returns the current threat view mode
func (tv *ThreatVisualizer) GetThreatViewMode() ThreatViewMode {
	return tv.threatViewMode
}

// calculateDangerValue calculates danger at a position using DangerByRange
func (tv *ThreatVisualizer) calculateDangerValue(pos coords.LogicalPosition, currentFactionID ecs.EntityID) float64 {
	// Get relevant squads based on view mode
	var relevantSquads []ecs.EntityID
	if tv.threatViewMode == ViewEnemyThreats {
		relevantSquads = tv.getEnemySquads(currentFactionID)
	} else {
		relevantSquads = tv.getPlayerSquads(currentFactionID)
	}

	totalValue := 0.0
	for _, squadID := range relevantSquads {
		squadPos, err := combat.GetSquadMapPosition(squadID, tv.manager)
		if err != nil {
			continue
		}

		factionID := combat.GetSquadFaction(squadID, tv.manager)
		if factionID == 0 {
			continue
		}

		factionThreat := tv.threatManager.factions[factionID]
		if factionThreat == nil {
			continue
		}

		squadThreat := factionThreat.squadDangerLevel[squadID]
		if squadThreat == nil {
			continue
		}

		distance := pos.ManhattanDistance(&squadPos)
		if value, exists := squadThreat.DangerByRange[distance]; exists {
			totalValue += value
		}
	}

	return totalValue
}

// dangerValueToColorMatrix converts danger value to red gradient
func (tv *ThreatVisualizer) dangerValueToColorMatrix(value float64) graphics.ColorMatrix {
	if value == 0 {
		return graphics.NewEmptyMatrix()
	}

	if value <= 50 {
		return graphics.CreateRedGradient(0.2)
	} else if value <= 100 {
		return graphics.CreateRedGradient(0.5)
	} else if value <= 150 {
		return graphics.CreateRedGradient(0.7)
	} else {
		return graphics.CreateRedGradient(0.9)
	}
}

// getEnemySquads returns all squads from enemy factions
func (tv *ThreatVisualizer) getEnemySquads(currentFactionID ecs.EntityID) []ecs.EntityID {
	var enemySquads []ecs.EntityID
	allFactions := combat.GetAllFactions(tv.manager)

	for _, factionID := range allFactions {
		if factionID == currentFactionID {
			continue
		}
		squads := combat.GetSquadsForFaction(factionID, tv.manager)
		enemySquads = append(enemySquads, squads...)
	}

	return enemySquads
}

// getPlayerSquads returns all squads from the current faction
func (tv *ThreatVisualizer) getPlayerSquads(currentFactionID ecs.EntityID) []ecs.EntityID {
	return combat.GetSquadsForFaction(currentFactionID, tv.manager)
}

// =========================================
// Layer Mode API (formerly LayerVisualizer)
// =========================================

// CycleLayerMode advances to next layer mode
func (tv *ThreatVisualizer) CycleLayerMode() {
	tv.layerMode = (tv.layerMode + 1) % LayerModeCount
	tv.MarkDirty()
	if tv.isActive {
		tv.ClearVisualization()
	}
}

// GetLayerMode returns the current layer mode
func (tv *ThreatVisualizer) GetLayerMode() LayerMode {
	return tv.layerMode
}

// GetLayerModeInfo returns display metadata for current layer mode
func (tv *ThreatVisualizer) GetLayerModeInfo() LayerModeInfo {
	return LayerModeMetadata[tv.layerMode]
}

// getLayerValueAt returns threat layer value at position
func (tv *ThreatVisualizer) getLayerValueAt(pos coords.LogicalPosition) float64 {
	if tv.threatEvaluator == nil {
		return 0.0
	}

	switch tv.layerMode {
	case LayerMelee:
		return tv.threatEvaluator.GetCombatLayer().GetMeleeThreatAt(pos)
	case LayerRanged:
		return tv.threatEvaluator.GetCombatLayer().GetRangedPressureAt(pos)
	case LayerSupport:
		return tv.threatEvaluator.GetSupportLayer().GetSupportValueAt(pos)
	case LayerPositionalFlanking:
		return tv.threatEvaluator.GetPositionalLayer().GetFlankingRiskAt(pos)
	case LayerPositionalIsolation:
		return tv.threatEvaluator.GetPositionalLayer().GetIsolationRiskAt(pos)
	case LayerPositionalEngagement:
		return tv.threatEvaluator.GetPositionalLayer().GetEngagementPressureAt(pos)
	case LayerPositionalRetreat:
		return tv.threatEvaluator.GetPositionalLayer().GetRetreatQuality(pos)
	default:
		return 0.0
	}
}

// layerValueToColorMatrix converts layer value to colored gradient
func (tv *ThreatVisualizer) layerValueToColorMatrix(value float64) graphics.ColorMatrix {
	if value == 0 {
		return graphics.NewEmptyMatrix()
	}

	gradientFunc := tv.getLayerGradientFunction()

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

// getLayerGradientFunction returns color gradient for current layer mode
func (tv *ThreatVisualizer) getLayerGradientFunction() func(float32) graphics.ColorMatrix {
	switch tv.layerMode {
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
		return graphics.CreateGreenGradient
	default:
		return graphics.CreateRedGradient
	}
}
