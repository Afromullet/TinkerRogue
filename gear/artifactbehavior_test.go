package gear

import (
	"testing"
)

func TestBehaviorRegistry_AllRegistered(t *testing.T) {
	all := AllBehaviors()

	expectedKeys := []string{
		BehaviorVanguardMovement,
		BehaviorEngagementChains,
		BehaviorMomentumStandard,
		BehaviorEchoDrums,
		BehaviorSaboteurWsHourglass,
		BehaviorDoubleTime,
		BehaviorStandDown,
		BehaviorDeadlockShackles,
		BehaviorAnthemPerseverance,
		BehaviorRallyingHorn,
		BehaviorChainOfCommand,
	}

	if len(all) != len(expectedKeys) {
		t.Errorf("Expected %d behaviors, got %d", len(expectedKeys), len(all))
	}

	for _, key := range expectedKeys {
		if GetBehavior(key) == nil {
			t.Errorf("Expected behavior %q to be registered", key)
		}
	}
}

func TestBaseBehavior_NoOps(t *testing.T) {
	var b BaseBehavior

	// These should not panic
	b.OnPostReset(nil, 0, nil)
	b.OnAttackComplete(nil, 0, 0, nil)
	b.OnTurnEnd(nil, 0)

	if b.IsPlayerActivated() {
		t.Error("BaseBehavior should not be player-activated")
	}

	err := b.Activate(nil, 0)
	if err == nil {
		t.Error("BaseBehavior.Activate should return error")
	}
}
