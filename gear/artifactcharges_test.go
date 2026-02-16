package gear

import (
	"testing"

	"github.com/bytearena/ecs"
)

func TestNewChargeTracker(t *testing.T) {
	ct := NewArtifactChargeTracker()
	if ct == nil {
		t.Fatal("Expected non-nil charge tracker")
	}
	// Verify all charges are available on a fresh tracker
	if !ct.IsAvailable(BehaviorDoubleTime) {
		t.Error("Expected double_time to be available on fresh tracker")
	}
	if !ct.IsAvailable(BehaviorEchoDrums) {
		t.Error("Expected echo_drums to be available on fresh tracker")
	}
	if ct.HasPendingEffects() {
		t.Error("Expected no pending effects on fresh tracker")
	}
}

func TestUseAndCheckCharge(t *testing.T) {
	ct := NewArtifactChargeTracker()

	// Initially available
	if !ct.IsAvailable(BehaviorDoubleTime) {
		t.Error("Expected double_time to be available initially")
	}

	// Use battle charge
	ct.UseCharge(BehaviorDoubleTime, ChargeOncePerBattle)
	if ct.IsAvailable(BehaviorDoubleTime) {
		t.Error("Expected double_time to be unavailable after battle charge")
	}

	// Use round charge for a different behavior
	ct.UseCharge(BehaviorEchoDrums, ChargeOncePerRound)
	if ct.IsAvailable(BehaviorEchoDrums) {
		t.Error("Expected echo_drums to be unavailable after round charge")
	}

	// Unrelated behavior still available
	if !ct.IsAvailable(BehaviorMomentumStandard) {
		t.Error("Expected momentum_standard to still be available")
	}
}

func TestRefreshRoundCharges(t *testing.T) {
	ct := NewArtifactChargeTracker()

	ct.UseCharge(BehaviorDoubleTime, ChargeOncePerBattle)
	ct.UseCharge(BehaviorEchoDrums, ChargeOncePerRound)

	ct.RefreshRoundCharges()

	// Battle charge persists
	if ct.IsAvailable(BehaviorDoubleTime) {
		t.Error("Battle charge should persist after refresh")
	}

	// Round charge cleared
	if !ct.IsAvailable(BehaviorEchoDrums) {
		t.Error("Round charge should be available after refresh")
	}
}

func TestPendingEffects(t *testing.T) {
	ct := NewArtifactChargeTracker()

	ct.AddPendingEffect(BehaviorSaboteurWsHourglass, ecs.EntityID(10))
	ct.AddPendingEffect(BehaviorSaboteurWsHourglass, ecs.EntityID(20))
	ct.AddPendingEffect("other_effect", ecs.EntityID(30))

	// Consume matching effects
	matched := ct.ConsumePendingEffects(BehaviorSaboteurWsHourglass)
	if len(matched) != 2 {
		t.Errorf("Expected 2 matched effects, got %d", len(matched))
	}
	if matched[0].TargetSquadID != ecs.EntityID(10) {
		t.Errorf("Expected target squad 10, got %d", matched[0].TargetSquadID)
	}
	if matched[1].TargetSquadID != ecs.EntityID(20) {
		t.Errorf("Expected target squad 20, got %d", matched[1].TargetSquadID)
	}

	// Consume again returns empty
	matched2 := ct.ConsumePendingEffects(BehaviorSaboteurWsHourglass)
	if len(matched2) != 0 {
		t.Errorf("Expected 0 matched effects on second consume, got %d", len(matched2))
	}

	// Other effect still present
	other := ct.ConsumePendingEffects("other_effect")
	if len(other) != 1 {
		t.Errorf("Expected 1 other effect, got %d", len(other))
	}
}

func TestReset(t *testing.T) {
	ct := NewArtifactChargeTracker()

	ct.UseCharge(BehaviorDoubleTime, ChargeOncePerBattle)
	ct.UseCharge(BehaviorEchoDrums, ChargeOncePerRound)
	ct.AddPendingEffect(BehaviorSaboteurWsHourglass, ecs.EntityID(10))

	ct.Reset()

	if !ct.IsAvailable(BehaviorDoubleTime) {
		t.Error("Expected double_time available after reset")
	}
	if !ct.IsAvailable(BehaviorEchoDrums) {
		t.Error("Expected echo_drums available after reset")
	}
	if ct.PendingEffectCount() != 0 {
		t.Errorf("Expected 0 pending effects after reset, got %d", ct.PendingEffectCount())
	}
}
