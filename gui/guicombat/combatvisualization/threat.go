package combatvisualization

import (
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/tactical/combat/combatservices"
	"game_main/tactical/combat/combatstate"
	"game_main/visual/graphics"
	"game_main/world/worldmapcore"

	"github.com/bytearena/ecs"
)

// VisualizerMode represents the primary visualization mode
type VisualizerMode int

const (
	VisualizerModeThreat VisualizerMode = iota
	VisualizerModeLayer
)

// LayerMode represents which threat layer to visualize (for layer mode)
type LayerMode int

const (
	LayerMelee LayerMode = iota
	LayerRanged
	LayerSupport
	LayerPositionalFlanking
	LayerPositionalIsolation
	LayerPositionalEngagement
	LayerPositionalRetreat
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
type ThreatVisualizer struct {
	manager        *common.EntityManager
	gameMap        *worldmapcore.GameMap
	threatProvider combatservices.ThreatProvider

	evaluators map[ecs.EntityID]combatservices.ThreatLayerEvaluator

	factionIDs       []ecs.EntityID
	viewFactionIndex int

	lastRound        int
	dirty            bool
	isActive         bool
	mode             VisualizerMode
	layerMode        LayerMode
	currentFactionID ecs.EntityID
}

// NewThreatVisualizer creates a unified threat visualizer
func NewThreatVisualizer(
	manager *common.EntityManager,
	gameMap *worldmapcore.GameMap,
	threatProvider combatservices.ThreatProvider,
) *ThreatVisualizer {
	return &ThreatVisualizer{
		manager:          manager,
		gameMap:          gameMap,
		threatProvider:   threatProvider,
		evaluators:       make(map[ecs.EntityID]combatservices.ThreatLayerEvaluator),
		lastRound:        -1,
		dirty:            true,
		isActive:         false,
		mode:             VisualizerModeThreat,
		viewFactionIndex: 0,
		layerMode:        LayerMelee,
	}
}

func (tv *ThreatVisualizer) Toggle() {
	tv.isActive = !tv.isActive
	if tv.isActive {
		tv.dirty = true
	} else {
		tv.ClearVisualization()
	}
}

func (tv *ThreatVisualizer) IsActive() bool {
	return tv.isActive
}

func (tv *ThreatVisualizer) SetFactions(factionIDs []ecs.EntityID) {
	tv.factionIDs = factionIDs
	if tv.viewFactionIndex >= len(tv.factionIDs) {
		tv.viewFactionIndex = 0
	}
}

func (tv *ThreatVisualizer) SetEvaluators(evaluators map[ecs.EntityID]combatservices.ThreatLayerEvaluator) {
	tv.evaluators = evaluators
}

func (tv *ThreatVisualizer) CycleFaction() {
	if len(tv.factionIDs) == 0 {
		return
	}
	tv.viewFactionIndex = (tv.viewFactionIndex + 1) % len(tv.factionIDs)
	tv.dirty = true
	if tv.isActive {
		tv.ClearVisualization()
	}
}

func (tv *ThreatVisualizer) GetViewFactionID() ecs.EntityID {
	if len(tv.factionIDs) == 0 {
		return 0
	}
	return tv.factionIDs[tv.viewFactionIndex]
}

func (tv *ThreatVisualizer) SetMode(mode VisualizerMode) {
	if tv.mode != mode {
		tv.mode = mode
		tv.dirty = true
		tv.lastRound = -1
		if tv.isActive {
			tv.ClearVisualization()
		}
	}
}

func (tv *ThreatVisualizer) GetMode() VisualizerMode {
	return tv.mode
}

func (tv *ThreatVisualizer) ClearVisualization() {
	for i := 0; i < tv.gameMap.TileCount(); i++ {
		tv.gameMap.ApplyColorMatrixToIndex(i, graphics.NewEmptyMatrix())
	}
	tv.dirty = true
}

func (tv *ThreatVisualizer) Update(
	currentFactionID ecs.EntityID,
	currentRound int,
	playerPos coords.LogicalPosition,
	viewportSize int,
) {
	if !tv.isActive {
		return
	}

	if !tv.dirty && tv.lastRound == currentRound {
		return
	}

	tv.currentFactionID = currentFactionID

	viewFactionID := tv.GetViewFactionID()

	if tv.mode == VisualizerModeLayer {
		if eval, ok := tv.evaluators[viewFactionID]; ok && eval != nil {
			eval.Update()
		}
	}

	var relevantSquads []ecs.EntityID
	if tv.mode == VisualizerModeThreat {
		relevantSquads = combatstate.GetSquadsForFaction(viewFactionID, tv.manager)
	}

	iterateViewport(playerPos, viewportSize, func(pos coords.LogicalPosition) {
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

	tv.lastRound = currentRound
	tv.dirty = false
}

func (tv *ThreatVisualizer) calculateThreatValueForSquads(pos coords.LogicalPosition, relevantSquads []ecs.EntityID) float64 {
	if tv.threatProvider == nil {
		return 0.0
	}

	totalValue := 0.0
	for _, squadID := range relevantSquads {
		squadPos, err := combatstate.GetSquadMapPosition(squadID, tv.manager)
		if err != nil {
			continue
		}

		factionID := combatstate.GetSquadFaction(squadID, tv.manager)
		if factionID == 0 {
			continue
		}

		distance := pos.ManhattanDistance(&squadPos)
		if value, exists := tv.threatProvider.GetSquadThreatAtRange(factionID, squadID, distance); exists {
			totalValue += value
		}
	}

	return totalValue
}

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

func (tv *ThreatVisualizer) CycleLayerMode() {
	tv.layerMode = (tv.layerMode + 1) % LayerModeCount
	tv.dirty = true
	if tv.isActive {
		tv.ClearVisualization()
	}
}

func (tv *ThreatVisualizer) GetLayerMode() LayerMode {
	return tv.layerMode
}

func (tv *ThreatVisualizer) GetLayerModeInfo() LayerModeInfo {
	return LayerModeMetadata[tv.layerMode]
}

func (tv *ThreatVisualizer) getLayerValueAt(pos coords.LogicalPosition) float64 {
	eval, ok := tv.evaluators[tv.GetViewFactionID()]
	if !ok || eval == nil {
		return 0.0
	}

	snapshot := eval.EvaluateAt(pos)

	const maxThreatValue = 200.0

	switch tv.layerMode {
	case LayerMelee:
		return min(snapshot.MeleeThreat/maxThreatValue, 1.0)
	case LayerRanged:
		return min(snapshot.RangedPressure/maxThreatValue, 1.0)
	case LayerSupport:
		return snapshot.SupportValue
	case LayerPositionalFlanking:
		return snapshot.FlankingRisk
	case LayerPositionalIsolation:
		return snapshot.IsolationRisk
	case LayerPositionalEngagement:
		return snapshot.EngagementPressure
	case LayerPositionalRetreat:
		return snapshot.RetreatQuality
	default:
		return 0.0
	}
}

func (tv *ThreatVisualizer) layerValueToColorMatrix(value float64) graphics.ColorMatrix {
	if value == 0 {
		return graphics.NewEmptyMatrix()
	}

	gradientFunc := tv.getLayerGradientFunction()

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

func iterateViewport(center coords.LogicalPosition, viewportSize int, callback func(pos coords.LogicalPosition)) {
	minX := center.X - viewportSize/2
	maxX := center.X + viewportSize/2
	minY := center.Y - viewportSize/2
	maxY := center.Y + viewportSize/2

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			callback(coords.LogicalPosition{X: x, Y: y})
		}
	}
}
