package guicombat

import (
	"game_main/core/common"
	"game_main/gui/framework"
	"game_main/tactical/combat/combatservices"
	"game_main/visual/combatrender"

	"game_main/core/coords"
	"game_main/world/worldmapcore"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatVisualizationManager manages all combat visualization systems including
// threat visualization (both danger and layer modes) and threat evaluation.
type CombatVisualizationManager struct {
	// Rendering systems
	movementRenderer  *combatrender.MovementTileRenderer
	highlightRenderer *combatrender.SquadHighlightRenderer
	healthBarRenderer *combatrender.HealthBarRenderer
	threatVisualizer  *ThreatVisualizer

	// Threat providers (backed by CombatService's threat systems, accessed via interfaces)
	threatProvider  combatservices.ThreatProvider
	layerEvaluators map[ecs.EntityID]combatservices.ThreatLayerEvaluator

	// References needed for late initialization
	ecsManager    *common.EntityManager
	gameMap       *worldmapcore.GameMap
	queries       *framework.GUIQueries
	combatService *combatservices.CombatService
}

// NewCombatVisualizationManager creates and initializes all visualization systems.
// Uses CombatService's existing threat systems instead of creating its own.
func NewCombatVisualizationManager(
	ctx *framework.UIContext,
	queries *framework.GUIQueries,
	gameMap *worldmapcore.GameMap,
	combatService *combatservices.CombatService,
) *CombatVisualizationManager {
	cvm := &CombatVisualizationManager{
		movementRenderer:  combatrender.NewMovementTileRenderer(),
		highlightRenderer: combatrender.NewSquadHighlightRenderer(queries),
		healthBarRenderer: combatrender.NewHealthBarRenderer(queries),
		ecsManager:        ctx.ECSManager,
		gameMap:           gameMap,
		queries:           queries,
		combatService:     combatService,
	}

	// Use CombatService's threat provider via interface
	cvm.threatProvider = combatService.GetThreatProvider()
	allFactions := queries.GetAllFactions()
	if cvm.threatProvider != nil {
		for _, factionID := range allFactions {
			cvm.threatProvider.AddFaction(factionID)
		}
	}

	// Get evaluators from CombatService via interface
	cvm.layerEvaluators = make(map[ecs.EntityID]combatservices.ThreatLayerEvaluator)
	for _, factionID := range allFactions {
		eval := combatService.GetThreatEvaluator(factionID)
		if eval != nil {
			cvm.layerEvaluators[factionID] = eval
		}
	}

	// Initialize unified threat visualizer (supports both danger and layer modes)
	cvm.threatVisualizer = NewThreatVisualizer(
		ctx.ECSManager,
		gameMap,
		cvm.threatProvider,
	)
	cvm.threatVisualizer.SetFactions(allFactions)
	cvm.threatVisualizer.SetEvaluators(cvm.layerEvaluators)

	return cvm
}

// RenderAll renders all combat visualization layers: squad highlights, movement tiles, and health bars
func (cvm *CombatVisualizationManager) RenderAll(
	screen *ebiten.Image,
	playerPos coords.LogicalPosition,
	currentFactionID ecs.EntityID,
	battleState *framework.TacticalState,
	combatService *combatservices.CombatService,
) {
	cvm.highlightRenderer.Render(screen, playerPos, currentFactionID, battleState.SelectedSquadID)

	if battleState.InMoveMode {
		validTiles := cvm.GetValidMoveTiles(combatService, battleState.SelectedSquadID, battleState.InMoveMode)
		if len(validTiles) > 0 {
			cvm.movementRenderer.Render(screen, playerPos, validTiles)
		}
	}

	if battleState.ShowHealthBars {
		cvm.healthBarRenderer.Render(screen, playerPos)
	}
}

// GetValidMoveTiles computes valid movement tiles on-demand
func (cvm *CombatVisualizationManager) GetValidMoveTiles(
	combatService *combatservices.CombatService,
	selectedSquadID ecs.EntityID,
	inMoveMode bool,
) []coords.LogicalPosition {
	if selectedSquadID == 0 || !inMoveMode {
		return []coords.LogicalPosition{}
	}

	tiles := combatService.MovementSystem.GetValidMovementTiles(selectedSquadID)
	if tiles == nil {
		return []coords.LogicalPosition{}
	}
	return tiles
}

// GetThreatVisualizer returns the unified threat visualizer
func (cvm *CombatVisualizationManager) GetThreatVisualizer() *ThreatVisualizer {
	return cvm.threatVisualizer
}

// RefreshFactions adds any new factions to the threat manager.
// Should be called when combat starts and factions are created.
func (cvm *CombatVisualizationManager) RefreshFactions(queries *framework.GUIQueries) {
	if cvm.threatProvider == nil {
		return
	}

	// Add all factions to threat manager via interface
	allFactions := queries.GetAllFactions()
	for _, factionID := range allFactions {
		cvm.threatProvider.AddFaction(factionID)
	}

	// Create evaluators for any new factions via CombatService
	if cvm.layerEvaluators == nil {
		cvm.layerEvaluators = make(map[ecs.EntityID]combatservices.ThreatLayerEvaluator)
	}
	for _, factionID := range allFactions {
		if _, exists := cvm.layerEvaluators[factionID]; !exists {
			eval := cvm.combatService.GetThreatEvaluator(factionID)
			if eval != nil {
				cvm.layerEvaluators[factionID] = eval
			}
		}
	}

	// If threat visualizer was nil, create it now
	if cvm.threatVisualizer == nil {
		cvm.threatVisualizer = NewThreatVisualizer(
			cvm.ecsManager,
			cvm.gameMap,
			cvm.threatProvider,
		)
	}

	// Always refresh factions and evaluators on the visualizer
	cvm.threatVisualizer.SetFactions(allFactions)
	cvm.threatVisualizer.SetEvaluators(cvm.layerEvaluators)
}

// UpdateThreatManagers updates all threat-related systems
func (cvm *CombatVisualizationManager) UpdateThreatManagers() {
	if cvm.threatProvider != nil {
		cvm.threatProvider.UpdateAllFactions()
	}
}

// UpdateThreatEvaluator updates all per-faction threat evaluators for a given round
func (cvm *CombatVisualizationManager) UpdateThreatEvaluator(round int) {
	for _, eval := range cvm.layerEvaluators {
		if eval != nil {
			eval.Update(round)
		}
	}
}

// UpdateThreatVisualization updates threat visualization if active
func (cvm *CombatVisualizationManager) UpdateThreatVisualization(
	currentFactionID ecs.EntityID,
	currentRound int,
	playerPos coords.LogicalPosition,
	viewportSize int,
) {
	if cvm.threatVisualizer != nil && cvm.threatVisualizer.IsActive() {
		cvm.threatVisualizer.Update(currentFactionID, currentRound, playerPos, viewportSize)
	}
}

// ResetHighlightColors resets faction highlight colors for a new combat.
func (cvm *CombatVisualizationManager) ResetHighlightColors() {
	if cvm.highlightRenderer != nil {
		cvm.highlightRenderer.ResetFactionColors()
	}
}

// ClearAllVisualizations clears all active visualizations
func (cvm *CombatVisualizationManager) ClearAllVisualizations() {
	if cvm.threatVisualizer != nil {
		cvm.threatVisualizer.ClearVisualization()
	}
}
