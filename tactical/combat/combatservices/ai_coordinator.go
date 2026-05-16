package combatservices

import (
	"github.com/bytearena/ecs"
)

// AICoordinator owns the per-combat AI controller, the faction threat provider,
// and the lazily-built per-faction threat evaluators. It exists so CombatService
// stops holding three orthogonal AI/threat fields and three separate setters
// (SetAIController, SetThreatProvider, SetThreatEvaluatorFactory) — they
// collapse into one Install entry point and one coherent type.
type AICoordinator struct {
	controller      AITurnController
	threatProvider  ThreatProvider
	layerEvaluators map[ecs.EntityID]ThreatLayerEvaluator
	evalFactory     func(factionID ecs.EntityID) ThreatLayerEvaluator
}

// NewAICoordinator constructs an empty coordinator. The AI controller, threat
// provider, and evaluator factory must be injected via Install before the
// coordinator can answer queries — concrete types live in mind/ai and
// mind/behavior, which import this package.
func NewAICoordinator() *AICoordinator {
	return &AICoordinator{
		layerEvaluators: make(map[ecs.EntityID]ThreatLayerEvaluator),
	}
}

// Install wires up all three AI dependencies in a single call. Replaces the
// three setters that previously lived on CombatService.
func (a *AICoordinator) Install(
	controller AITurnController,
	threat ThreatProvider,
	evalFactory func(factionID ecs.EntityID) ThreatLayerEvaluator,
) {
	a.controller = controller
	a.threatProvider = threat
	a.evalFactory = evalFactory
}

// SetController updates only the AI controller. Kept for callers that want to
// swap controllers without re-installing the threat side.
func (a *AICoordinator) SetController(ctrl AITurnController) {
	a.controller = ctrl
}

// SetThreatProvider updates only the threat provider.
func (a *AICoordinator) SetThreatProvider(tp ThreatProvider) {
	a.threatProvider = tp
}

// SetThreatEvaluatorFactory updates only the evaluator factory. Existing cached
// evaluators stay — call ResetEvaluators if you need them rebuilt.
func (a *AICoordinator) SetThreatEvaluatorFactory(fn func(factionID ecs.EntityID) ThreatLayerEvaluator) {
	a.evalFactory = fn
}

// Controller returns the installed AI controller (may be nil if Install was
// not called).
func (a *AICoordinator) Controller() AITurnController { return a.controller }

// ThreatProvider returns the installed threat provider.
func (a *AICoordinator) ThreatProvider() ThreatProvider { return a.threatProvider }

// GetThreatEvaluator returns the composite evaluator for a faction, lazily
// building one via evalFactory on first access. Returns nil if no factory was
// installed.
func (a *AICoordinator) GetThreatEvaluator(factionID ecs.EntityID) ThreatLayerEvaluator {
	if eval, exists := a.layerEvaluators[factionID]; exists {
		return eval
	}
	if a.evalFactory == nil {
		return nil
	}
	eval := a.evalFactory(factionID)
	a.layerEvaluators[factionID] = eval
	return eval
}

// UpdateThreatLayers refreshes base threat data and every cached composite
// evaluator. Called at the start of an AI turn.
func (a *AICoordinator) UpdateThreatLayers() {
	if a.threatProvider != nil {
		a.threatProvider.UpdateAllFactions()
	}
	for _, evaluator := range a.layerEvaluators {
		evaluator.Update()
	}
}
