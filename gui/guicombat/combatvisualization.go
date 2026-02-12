package guicombat

import (
	"game_main/common"
	"game_main/gui/framework"
	"game_main/mind/behavior"
	"game_main/tactical/combatservices"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatVisualizationManager manages all combat visualization systems including
// threat visualization (both danger and layer modes) and threat evaluation.
type CombatVisualizationManager struct {
	// Rendering systems
	movementRenderer  *rendering.MovementTileRenderer
	highlightRenderer *rendering.SquadHighlightRenderer
	healthBarRenderer *rendering.HealthBarRenderer
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
		movementRenderer:  rendering.NewMovementTileRenderer(),
		highlightRenderer: rendering.NewSquadHighlightRenderer(queries),
		healthBarRenderer: rendering.NewHealthBarRenderer(queries),
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
