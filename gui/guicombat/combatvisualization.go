package guicombat

import (
	"game_main/gui/framework"

	"game_main/gui/guimodes"
	"game_main/tactical/behavior"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
)

// CombatVisualizationManager manages all combat visualization systems including
// danger visualization, layer visualization, and threat evaluation.
type CombatVisualizationManager struct {
	// Rendering systems
	movementRenderer  *guimodes.MovementTileRenderer
	highlightRenderer *guimodes.SquadHighlightRenderer
	dangerVisualizer  *behavior.DangerVisualizer
	layerVisualizer   *behavior.LayerVisualizer

	// Threat management
	threatManager   *behavior.FactionThreatLevelManager
	threatEvaluator *behavior.CompositeThreatEvaluator
}

// NewCombatVisualizationManager creates and initializes all visualization systems
func NewCombatVisualizationManager(
	ctx *framework.UIContext,
	queries *framework.GUIQueries,
	gameMap *worldmap.GameMap,
) *CombatVisualizationManager {
	cvm := &CombatVisualizationManager{
		movementRenderer:  guimodes.NewMovementTileRenderer(),
		highlightRenderer: guimodes.NewSquadHighlightRenderer(queries),
	}

	// Create the initial Faction Threat Level Manager and add all factions
	cvm.threatManager = behavior.NewFactionThreatLevelManager(ctx.ECSManager, queries.CombatCache)
	for _, factionID := range queries.GetAllFactions() {
		cvm.threatManager.AddFaction(factionID)
	}

	// Initialize danger visualizer
	cvm.dangerVisualizer = behavior.NewDangerVisualizer(ctx.ECSManager, gameMap, cvm.threatManager)

	// Create threat evaluators for layer visualization
	allFactions := queries.GetAllFactions()
	if len(allFactions) > 0 {
		// Use player faction (first faction) for threat evaluation
		playerFactionID := allFactions[0]
		cvm.threatEvaluator = behavior.NewCompositeThreatEvaluator(
			playerFactionID,
			ctx.ECSManager,
			queries.CombatCache,
			cvm.threatManager,
		)
		cvm.layerVisualizer = behavior.NewLayerVisualizer(
			ctx.ECSManager,
			gameMap,
			cvm.threatEvaluator,
		)
	}

	return cvm
}

// GetMovementRenderer returns the movement tile renderer
func (cvm *CombatVisualizationManager) GetMovementRenderer() *guimodes.MovementTileRenderer {
	return cvm.movementRenderer
}

// GetHighlightRenderer returns the squad highlight renderer
func (cvm *CombatVisualizationManager) GetHighlightRenderer() *guimodes.SquadHighlightRenderer {
	return cvm.highlightRenderer
}

// GetDangerVisualizer returns the danger visualizer
func (cvm *CombatVisualizationManager) GetDangerVisualizer() *behavior.DangerVisualizer {
	return cvm.dangerVisualizer
}

// GetLayerVisualizer returns the layer visualizer
func (cvm *CombatVisualizationManager) GetLayerVisualizer() *behavior.LayerVisualizer {
	return cvm.layerVisualizer
}

// GetThreatManager returns the faction threat level manager
func (cvm *CombatVisualizationManager) GetThreatManager() *behavior.FactionThreatLevelManager {
	return cvm.threatManager
}

// GetThreatEvaluator returns the composite threat evaluator
func (cvm *CombatVisualizationManager) GetThreatEvaluator() *behavior.CompositeThreatEvaluator {
	return cvm.threatEvaluator
}

// UpdateThreatManagers updates all threat-related systems
func (cvm *CombatVisualizationManager) UpdateThreatManagers() {
	if cvm.threatManager != nil {
		cvm.threatManager.UpdateAllFactions()
	}
}

// UpdateThreatEvaluator updates the threat evaluator for a given round
func (cvm *CombatVisualizationManager) UpdateThreatEvaluator(round int) {
	if cvm.threatEvaluator != nil {
		cvm.threatEvaluator.Update(round)
	}
}

// UpdateDangerVisualization updates danger visualization if active
func (cvm *CombatVisualizationManager) UpdateDangerVisualization(
	currentFactionID ecs.EntityID,
	currentRound int,
	playerPos coords.LogicalPosition,
	viewportSize int,
) {
	if cvm.dangerVisualizer != nil && cvm.dangerVisualizer.IsActive() {
		cvm.dangerVisualizer.Update(currentFactionID, currentRound, playerPos, viewportSize)
	}
}

// UpdateLayerVisualization updates layer visualization if active
func (cvm *CombatVisualizationManager) UpdateLayerVisualization(
	currentFactionID ecs.EntityID,
	currentRound int,
	playerPos coords.LogicalPosition,
	viewportSize int,
) {
	if cvm.layerVisualizer != nil && cvm.layerVisualizer.IsActive() {
		cvm.layerVisualizer.Update(currentFactionID, currentRound, playerPos, viewportSize)
	}
}

// ClearAllVisualizations clears all active visualizations
func (cvm *CombatVisualizationManager) ClearAllVisualizations() {
	if cvm.dangerVisualizer != nil {
		cvm.dangerVisualizer.ClearVisualization()
	}
	if cvm.layerVisualizer != nil {
		cvm.layerVisualizer.ClearVisualization()
	}
}
