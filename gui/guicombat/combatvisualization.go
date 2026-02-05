package guicombat

import (
	"game_main/common"
	"game_main/gui/framework"
	"game_main/mind/behavior"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
)

// CombatVisualizationManager manages all combat visualization systems including
// threat visualization (both danger and layer modes) and threat evaluation.
type CombatVisualizationManager struct {
	// Rendering systems
	movementRenderer  *framework.MovementTileRenderer
	highlightRenderer *framework.SquadHighlightRenderer
	healthBarRenderer *framework.HealthBarRenderer
	threatVisualizer  *behavior.ThreatVisualizer

	// Threat management
	threatManager   *behavior.FactionThreatLevelManager
	threatEvaluator *behavior.CompositeThreatEvaluator

	// References needed for late initialization
	ecsManager *common.EntityManager
	gameMap    *worldmap.GameMap
	queries    *framework.GUIQueries
}

// NewCombatVisualizationManager creates and initializes all visualization systems
func NewCombatVisualizationManager(
	ctx *framework.UIContext,
	queries *framework.GUIQueries,
	gameMap *worldmap.GameMap,
) *CombatVisualizationManager {
	cvm := &CombatVisualizationManager{
		movementRenderer:  framework.NewMovementTileRenderer(),
		highlightRenderer: framework.NewSquadHighlightRenderer(queries),
		healthBarRenderer: framework.NewHealthBarRenderer(queries),
		ecsManager:        ctx.ECSManager,
		gameMap:           gameMap,
		queries:           queries,
	}

	// Create the initial Faction Threat Level Manager and add all factions
	cvm.threatManager = behavior.NewFactionThreatLevelManager(ctx.ECSManager, queries.CombatCache)
	for _, factionID := range queries.GetAllFactions() {
		cvm.threatManager.AddFaction(factionID)
	}

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
	}

	// Initialize unified threat visualizer (supports both danger and layer modes)
	cvm.threatVisualizer = behavior.NewThreatVisualizer(
		ctx.ECSManager,
		gameMap,
		cvm.threatManager,
		cvm.threatEvaluator,
	)

	return cvm
}

// GetMovementRenderer returns the movement tile renderer
func (cvm *CombatVisualizationManager) GetMovementRenderer() *framework.MovementTileRenderer {
	return cvm.movementRenderer
}

// GetHighlightRenderer returns the squad highlight renderer
func (cvm *CombatVisualizationManager) GetHighlightRenderer() *framework.SquadHighlightRenderer {
	return cvm.highlightRenderer
}

// GetHealthBarRenderer returns the health bar renderer
func (cvm *CombatVisualizationManager) GetHealthBarRenderer() *framework.HealthBarRenderer {
	return cvm.healthBarRenderer
}

// GetThreatVisualizer returns the unified threat visualizer
func (cvm *CombatVisualizationManager) GetThreatVisualizer() *behavior.ThreatVisualizer {
	return cvm.threatVisualizer
}

// GetThreatManager returns the faction threat level manager
func (cvm *CombatVisualizationManager) GetThreatManager() *behavior.FactionThreatLevelManager {
	return cvm.threatManager
}

// GetThreatEvaluator returns the composite threat evaluator
func (cvm *CombatVisualizationManager) GetThreatEvaluator() *behavior.CompositeThreatEvaluator {
	return cvm.threatEvaluator
}

// RefreshFactions adds any new factions to the threat manager
// Should be called when combat starts and factions are created
func (cvm *CombatVisualizationManager) RefreshFactions(queries *framework.GUIQueries) {
	if cvm.threatManager == nil {
		return
	}

	// Add all factions to threat manager
	allFactions := queries.GetAllFactions()
	for _, factionID := range allFactions {
		cvm.threatManager.AddFaction(factionID)
	}

	// If threat evaluator was nil during initialization (no factions existed yet), create it now
	if len(allFactions) > 0 && cvm.threatEvaluator == nil {
		// Use player faction (first faction) for threat evaluation
		playerFactionID := allFactions[0]
		cvm.threatEvaluator = behavior.NewCompositeThreatEvaluator(
			playerFactionID,
			cvm.ecsManager,
			queries.CombatCache,
			cvm.threatManager,
		)

		// Update the existing visualizer with the new evaluator
		if cvm.threatVisualizer != nil {
			cvm.threatVisualizer.SetThreatEvaluator(cvm.threatEvaluator)
		}
	}

	// If threat visualizer was nil, create it now
	if cvm.threatVisualizer == nil {
		cvm.threatVisualizer = behavior.NewThreatVisualizer(
			cvm.ecsManager,
			cvm.gameMap,
			cvm.threatManager,
			cvm.threatEvaluator,
		)
	}
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

// ClearAllVisualizations clears all active visualizations
func (cvm *CombatVisualizationManager) ClearAllVisualizations() {
	if cvm.threatVisualizer != nil {
		cvm.threatVisualizer.ClearVisualization()
	}
}
