package combatservices

import (
	"testing"

	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// fakeThreatProvider records whether Reset was called.
type fakeThreatProvider struct{ resetCalls int }

func (f *fakeThreatProvider) AddFaction(ecs.EntityID) {}
func (f *fakeThreatProvider) UpdateAllFactions()      {}
func (f *fakeThreatProvider) Reset()                  { f.resetCalls++ }
func (f *fakeThreatProvider) GetSquadThreatAtRange(ecs.EntityID, ecs.EntityID, int) (float64, bool) {
	return 0, false
}

// fakeController records whether Reset was called.
type fakeController struct{ resetCalls int }

func (f *fakeController) DecideFactionTurn(ecs.EntityID) bool { return false }
func (f *fakeController) HasQueuedAttacks() bool              { return false }
func (f *fakeController) GetQueuedAttacks() []QueuedAttack    { return nil }
func (f *fakeController) ClearAttackQueue()                   {}
func (f *fakeController) Reset()                              { f.resetCalls++ }

type fakeEvaluator struct{}

func (fakeEvaluator) Update()                                          {}
func (fakeEvaluator) EvaluateAt(coords.LogicalPosition) ThreatSnapshot { return ThreatSnapshot{} }

// TestAICoordinatorReset verifies Reset clears the evaluator cache and propagates
// to the installed threat provider and controller. This is the leak fix for
// tech-debt item 7: without it, each disposed faction leaves a dead entry behind.
func TestAICoordinatorReset(t *testing.T) {
	provider := &fakeThreatProvider{}
	controller := &fakeController{}

	coord := NewAICoordinator()
	coord.Install(controller, provider, func(ecs.EntityID) ThreatLayerEvaluator {
		return fakeEvaluator{}
	})

	// Populate the coordinator's evaluator cache for a couple of factions.
	coord.GetThreatEvaluator(ecs.EntityID(1))
	coord.GetThreatEvaluator(ecs.EntityID(2))
	if len(coord.layerEvaluators) != 2 {
		t.Fatalf("expected 2 cached evaluators before reset, got %d", len(coord.layerEvaluators))
	}

	coord.Reset()

	if len(coord.layerEvaluators) != 0 {
		t.Errorf("expected empty evaluator cache after reset, got %d", len(coord.layerEvaluators))
	}
	if provider.resetCalls != 1 {
		t.Errorf("expected threat provider Reset called once, got %d", provider.resetCalls)
	}
	if controller.resetCalls != 1 {
		t.Errorf("expected controller Reset called once, got %d", controller.resetCalls)
	}
}

// TestAICoordinatorResetNilDependencies verifies Reset is safe before Install.
func TestAICoordinatorResetNilDependencies(t *testing.T) {
	coord := NewAICoordinator()
	coord.Reset() // must not panic with nil provider/controller
	if len(coord.layerEvaluators) != 0 {
		t.Errorf("expected empty evaluator cache, got %d", len(coord.layerEvaluators))
	}
}
