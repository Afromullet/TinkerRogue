package behavior

import (
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"
	"testing"
)

// TestCombatThreatLayer_Compute tests basic combat threat computation
func TestCombatThreatLayer_Compute(t *testing.T) {
	// Setup test environment
	manager := combat.CreateTestCombatManager()
	cache := combat.NewCombatQueryCache(manager)
	baseThreatMgr := NewFactionThreatLevelManager(manager, cache)
	fm := combat.NewCombatFactionManager(manager, cache)

	// Create two factions
	faction1 := fm.CreateCombatFaction("Player", true)
	faction2 := fm.CreateCombatFaction("Enemy", false)

	// Add factions to threat manager
	baseThreatMgr.AddFaction(faction1)
	baseThreatMgr.AddFaction(faction2)

	// Create unified combat threat layer for faction1 (viewing threats FROM faction2)
	combatLayer := NewCombatThreatLayer(faction1, manager, cache, baseThreatMgr)

	// Verify initial state
	if combatLayer.factionID != faction1 {
		t.Errorf("Expected factionID %d, got %d", faction1, combatLayer.factionID)
	}

	// Compute threats
	combatLayer.Compute()

	// Verify layer marked as clean
	currentRound := 0
	if !combatLayer.IsValid(currentRound) {
		t.Error("Layer should be valid after Compute()")
	}

	// Verify melee data structures initialized
	if combatLayer.meleeThreatByPos == nil {
		t.Error("meleeThreatByPos should be initialized")
	}
	if combatLayer.meleeThreatBySquad == nil {
		t.Error("meleeThreatBySquad should be initialized")
	}

	// Verify ranged data structures initialized
	if combatLayer.rangedPressureByPos == nil {
		t.Error("rangedPressureByPos should be initialized")
	}
	if combatLayer.lineOfFireZones == nil {
		t.Error("lineOfFireZones should be initialized")
	}
}

// TestCompositeThreatEvaluator_Update tests layer update and caching
func TestCompositeThreatEvaluator_Update(t *testing.T) {
	// Setup test environment
	manager := combat.CreateTestCombatManager()
	cache := combat.NewCombatQueryCache(manager)
	baseThreatMgr := NewFactionThreatLevelManager(manager, cache)
	fm := combat.NewCombatFactionManager(manager, cache)

	faction1 := fm.CreateCombatFaction("Player", true)
	baseThreatMgr.AddFaction(faction1)

	// Create composite evaluator
	evaluator := NewCompositeThreatEvaluator(faction1, manager, cache, baseThreatMgr)

	// Initial update
	currentRound := 0
	evaluator.Update(currentRound)

	// Verify layers were computed
	if evaluator.lastUpdateRound != currentRound {
		t.Errorf("Expected lastUpdateRound %d, got %d", currentRound, evaluator.lastUpdateRound)
	}
	if evaluator.isDirty {
		t.Error("Evaluator should not be dirty after Update()")
	}

	// Update again with same round (should skip computation)
	evaluator.Update(currentRound)
	if evaluator.isDirty {
		t.Error("Evaluator should still be clean")
	}

	// Mark dirty and verify recomputation needed
	evaluator.MarkDirty()
	if !evaluator.isDirty {
		t.Error("Evaluator should be dirty after MarkDirty()")
	}
}

// TestCompositeThreatEvaluator_RoleWeights tests role-specific threat weighting
func TestCompositeThreatEvaluator_RoleWeights(t *testing.T) {
	// Verify role weights from config
	tankWeights := GetRoleBehaviorWeights(squads.RoleTank)
	dpsWeights := GetRoleBehaviorWeights(squads.RoleDPS)
	supportWeights := GetRoleBehaviorWeights(squads.RoleSupport)

	// Tank should seek melee danger (negative weight)
	if tankWeights.MeleeWeight >= 0 {
		t.Error("Tank melee weight should be negative (attraction)")
	}

	// DPS should avoid melee
	if dpsWeights.MeleeWeight <= 0 {
		t.Error("DPS melee weight should be positive (avoidance)")
	}

	// Support should strongly avoid melee
	if supportWeights.MeleeWeight <= dpsWeights.MeleeWeight {
		t.Error("Support should avoid melee more than DPS")
	}

	// Support should strongly avoid ranged
	if supportWeights.RangedWeight <= 0 {
		t.Error("Support ranged weight should be positive (avoidance)")
	}
}

// TestGetOptimalPositionForRole tests position selection
func TestGetOptimalPositionForRole(t *testing.T) {
	// Setup test environment
	manager := combat.CreateTestCombatManager()
	cache := combat.NewCombatQueryCache(manager)
	baseThreatMgr := NewFactionThreatLevelManager(manager, cache)
	fm := combat.NewCombatFactionManager(manager, cache)

	faction1 := fm.CreateCombatFaction("Player", true)
	baseThreatMgr.AddFaction(faction1)

	evaluator := NewCompositeThreatEvaluator(faction1, manager, cache, baseThreatMgr)
	evaluator.Update(0)

	// Test with empty candidates
	emptyPos := evaluator.GetOptimalPositionForRole(1, []coords.LogicalPosition{})
	if emptyPos.X != 0 || emptyPos.Y != 0 {
		t.Error("Should return zero position for empty candidates")
	}

	// Test with single candidate
	positions := []coords.LogicalPosition{{X: 5, Y: 5}}
	result := evaluator.GetOptimalPositionForRole(1, positions)
	if result.X != 5 || result.Y != 5 {
		t.Error("Should return the only candidate position")
	}
}

// TestThreatLayerBase_Caching tests cache invalidation logic
func TestThreatLayerBase_Caching(t *testing.T) {
	manager := combat.CreateTestCombatManager()
	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)
	faction1 := fm.CreateCombatFaction("Player", true)

	base := NewThreatLayerBase(faction1, manager, cache)

	// Initially dirty
	if !base.IsDirty() {
		t.Error("New layer should be dirty")
	}

	// Mark clean
	currentRound := 0
	base.markClean(currentRound)

	if base.IsDirty() {
		t.Error("Layer should not be dirty after markClean()")
	}
	if !base.IsValid(currentRound) {
		t.Error("Layer should be valid for current round")
	}

	// Different round should invalidate
	if base.IsValid(currentRound + 1) {
		t.Error("Layer should not be valid for different round")
	}

	// Mark dirty should invalidate
	base.MarkDirty()
	if base.IsValid(currentRound) {
		t.Error("Layer should not be valid after MarkDirty()")
	}
}

// TestGetSquadPrimaryRole tests role detection from unit composition
func TestGetSquadPrimaryRole(t *testing.T) {
	manager := combat.CreateTestCombatManager()

	// Test returns default when squad not found
	role := squads.GetSquadPrimaryRole(999, manager)
	if role != squads.RoleDPS {
		t.Error("Should return default DPS role for non-existent squad")
	}
}

// TestThreatLayerBase_GetEnemyFactions tests enemy faction detection
func TestThreatLayerBase_GetEnemyFactions(t *testing.T) {
	manager := combat.CreateTestCombatManager()
	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)

	// Create multiple factions
	faction1 := fm.CreateCombatFaction("Player", true)
	faction2 := fm.CreateCombatFaction("Enemy1", false)
	faction3 := fm.CreateCombatFaction("Enemy2", false)

	base := NewThreatLayerBase(faction1, manager, cache)
	enemies := base.getEnemyFactions()

	// Should return all factions except faction1
	expectedCount := 2
	if len(enemies) != expectedCount {
		t.Errorf("Expected %d enemy factions, got %d", expectedCount, len(enemies))
	}

	// Verify faction1 not in enemies
	for _, enemyID := range enemies {
		if enemyID == faction1 {
			t.Error("Own faction should not be in enemy list")
		}
	}

	// Verify faction2 and faction3 are in enemies
	foundFaction2 := false
	foundFaction3 := false
	for _, enemyID := range enemies {
		if enemyID == faction2 {
			foundFaction2 = true
		}
		if enemyID == faction3 {
			foundFaction3 = true
		}
	}

	if !foundFaction2 || !foundFaction3 {
		t.Error("All other factions should be in enemy list")
	}
}
