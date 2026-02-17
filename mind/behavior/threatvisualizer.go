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
	// VisualizerModeThreat shows danger projection from squads
	VisualizerModeThreat VisualizerMode = iota
	// VisualizerModeLayer shows individual threat layers
	VisualizerModeLayer
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
	manager       *common.EntityManager
	gameMap       *worldmap.GameMap
	threatManager *FactionThreatLevelManager

	// Per-faction threat evaluators (for layer mode)
	evaluators map[ecs.EntityID]*CompositeThreatEvaluator

	// Faction cycling
	factionIDs       []ecs.EntityID // All factions in combat
	viewFactionIndex int            // Index into factionIDs for viewed faction

	// State
	*evaluation.DirtyCache
	isActive         bool
	mode             VisualizerMode
	layerMode        LayerMode // For layer mode: which layer
	currentFactionID ecs.EntityID
}

// NewThreatVisualizer creates a unified threat visualizer
func NewThreatVisualizer(
	manager *common.EntityManager,
	gameMap *worldmap.GameMap,
	threatManager *FactionThreatLevelManager,
) *ThreatVisualizer {
	return &ThreatVisualizer{
		manager:          manager,
		gameMap:           gameMap,
		threatManager:    threatManager,
		evaluators:       make(map[ecs.EntityID]*CompositeThreatEvaluator),
		DirtyCache:       evaluation.NewDirtyCache(),
		isActive:         false,
		mode:             VisualizerModeThreat,
		viewFactionIndex: 0,
		layerMode:        LayerMelee,
	}
}

// =========================================
// Core API
// =========================================

// Toggle enables/disables the visualization
func (tv *ThreatVisualizer) Toggle() {
	tv.isActive = !tv.isActive
	if tv.isActive {
		tv.MarkDirty() // Force redraw when activating
	} else {
		tv.ClearVisualization()
	}
}

// IsActive returns whether visualization is currently enabled
func (tv *ThreatVisualizer) IsActive() bool {
	return tv.isActive
}

// SetFactions sets the list of factions available for cycling.
// Resets viewFactionIndex to 0 if it would be out of bounds.
func (tv *ThreatVisualizer) SetFactions(factionIDs []ecs.EntityID) {
	tv.factionIDs = factionIDs
	if tv.viewFactionIndex >= len(tv.factionIDs) {
		tv.viewFactionIndex = 0
	}
}

// SetEvaluators sets the per-faction threat evaluators (for layer mode).
func (tv *ThreatVisualizer) SetEvaluators(evaluators map[ecs.EntityID]*CompositeThreatEvaluator) {
	tv.evaluators = evaluators
}

// CycleFaction advances viewFactionIndex to the next faction.
func (tv *ThreatVisualizer) CycleFaction() {
	if len(tv.factionIDs) == 0 {
		return
	}
	tv.viewFactionIndex = (tv.viewFactionIndex + 1) % len(tv.factionIDs)
	tv.MarkDirty()
	if tv.isActive {
		tv.ClearVisualization()
	}
}

// GetViewFactionID returns the currently viewed faction ID.
// Returns 0 if no factions are set.
func (tv *ThreatVisualizer) GetViewFactionID() ecs.EntityID {
	if len(tv.factionIDs) == 0 {
		return 0
	}
	return tv.factionIDs[tv.viewFactionIndex]
}

// SetMode sets the primary visualization mode
func (tv *ThreatVisualizer) SetMode(mode VisualizerMode) {
	if tv.mode != mode {
		tv.mode = mode
		// Force re-render by marking dirty AND resetting the round
		// This is necessary because we now have ONE cache shared between modes
		tv.MarkDirty()
		tv.DirtyCache = evaluation.NewDirtyCache() // Reset cache completely
		// Also mark evaluator dirty when switching to layer mode
		if mode == VisualizerModeLayer {
			factionID := tv.GetViewFactionID()
			if eval, ok := tv.evaluators[factionID]; ok && eval != nil {
				eval.MarkDirty()
			}
		}
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

	// Determine which faction we're viewing
	viewFactionID := tv.GetViewFactionID()

	// Ensure threat evaluator is up-to-date (for layer mode)
	if tv.mode == VisualizerModeLayer {
		if eval, ok := tv.evaluators[viewFactionID]; ok && eval != nil {
			eval.Update(currentRound)
		}
	}

	// Pre-compute squad lists once per update (invariant across all tiles)
	var relevantSquads []ecs.EntityID
	if tv.mode == VisualizerModeThreat {
		relevantSquads = combat.GetSquadsForFaction(viewFactionID, tv.manager)
	}

	// Visualize based on current mode
	IterateViewport(playerPos, viewportSize, func(pos coords.LogicalPosition) {
		if !tv.gameMap.InBounds(pos.X, pos.Y) {
			return
		}

		var value float64
		var colorMatrix graphics.ColorMatrix

		switch tv.mode {
		case VisualizerModeThreat:
			value = tv.calculateThreatValueForSquads(pos, relevantSquads)
			colorMatrix = tv.threatValueToColorMatrix(value)
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
// Threat Mode API (formerly DangerVisualizer)
// =========================================

// calculateThreatValueForSquads calculates danger at a position for a pre-computed squad list.
// Used by Update() to avoid re-querying squad lists on every tile.
func (tv *ThreatVisualizer) calculateThreatValueForSquads(pos coords.LogicalPosition, relevantSquads []ecs.EntityID) float64 {
	if tv.threatManager == nil {
		return 0.0
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

		squadThreat := factionThreat.squadThreatLevels[squadID]
		if squadThreat == nil {
			continue
		}

		distance := pos.ManhattanDistance(&squadPos)
		if value, exists := squadThreat.ThreatByRange[distance]; exists {
			totalValue += value
		}
	}

	return totalValue
}

// threatValueToColorMatrix converts danger value to red gradient
func (tv *ThreatVisualizer) threatValueToColorMatrix(value float64) graphics.ColorMatrix {
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

// getLayerValueAt returns threat layer value at position (normalized to 0-1 range)
func (tv *ThreatVisualizer) getLayerValueAt(pos coords.LogicalPosition) float64 {
	eval, ok := tv.evaluators[tv.GetViewFactionID()]
	if !ok || eval == nil {
		return 0.0
	}

	// Max values for normalization (melee/ranged/support use raw power values)
	const maxThreatValue = 200.0
	const maxSupportValue = 1.0 // Support is already normalized by heal priority (0-1)

	switch tv.layerMode {
	case LayerMelee:
		raw := eval.GetCombatLayer().GetMeleeThreatAt(pos)
		return min(raw/maxThreatValue, 1.0)
	case LayerRanged:
		raw := eval.GetCombatLayer().GetRangedPressureAt(pos)
		return min(raw/maxThreatValue, 1.0)
	case LayerSupport:
		// Support value is already in 0-1 range from heal priority
		return eval.GetSupportLayer().GetSupportValueAt(pos)
	case LayerPositionalFlanking:
		return eval.GetPositionalLayer().GetFlankingRiskAt(pos)
	case LayerPositionalIsolation:
		return eval.GetPositionalLayer().GetIsolationRiskAt(pos)
	case LayerPositionalEngagement:
		return eval.GetPositionalLayer().GetEngagementPressureAt(pos)
	case LayerPositionalRetreat:
		return eval.GetPositionalLayer().GetRetreatQuality(pos)
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
