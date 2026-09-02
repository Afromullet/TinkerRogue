package behavior

import (
	"testing"

	"game_main/tactical/combat/combatstate"
)

// TestFactionThreatLevelManager_Reset verifies Reset drops all tracked faction
// state. This is the leak fix for tech-debt item 7: entity IDs are never
// recycled, so without resetting at combat teardown the map would accumulate one
// dead entry per disposed faction for the whole session.
func TestFactionThreatLevelManager_Reset(t *testing.T) {
	manager := createTestCombatManager()
	cache := combatstate.NewCombatQueryCache(manager)
	mgr := NewFactionThreatLevelManager(manager, cache)
	fm := combatstate.NewCombatFactionManager(manager, cache)

	mgr.AddFaction(fm.CreateCombatFaction("Player", true))
	mgr.AddFaction(fm.CreateCombatFaction("Enemy", false))

	if len(mgr.factions) != 2 {
		t.Fatalf("expected 2 tracked factions before reset, got %d", len(mgr.factions))
	}

	mgr.Reset()

	if len(mgr.factions) != 0 {
		t.Errorf("expected no tracked factions after Reset, got %d", len(mgr.factions))
	}

	// A faction added after reset is tracked normally (map is still usable).
	mgr.AddFaction(fm.CreateCombatFaction("Enemy2", false))
	if len(mgr.factions) != 1 {
		t.Errorf("expected 1 tracked faction after post-reset add, got %d", len(mgr.factions))
	}
}
