package overworld

import (
	"game_main/common"
	testfx "game_main/testing"
	"game_main/world/coords"
	"testing"
)

// TestFactionCreation verifies faction entities are created correctly
func TestFactionCreation(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create a faction
	homePos := coords.LogicalPosition{X: 50, Y: 40}
	factionID := CreateFaction(manager, FactionBandits, homePos, 5)

	if factionID == 0 {
		t.Fatal("CreateFaction returned zero ID")
	}

	// Verify faction data
	factionData := GetFactionByID(manager, factionID)
	if factionData == nil {
		t.Fatal("GetFactionByID returned nil")
	}

	if factionData.FactionType != FactionBandits {
		t.Errorf("Expected FactionBandits, got %v", factionData.FactionType)
	}

	if factionData.Strength != 5 {
		t.Errorf("Expected strength 5, got %d", factionData.Strength)
	}

	if factionData.TerritorySize != 1 {
		t.Errorf("Expected territory size 1, got %d", factionData.TerritorySize)
	}
}

// TestFactionExpansion verifies territory expansion
func TestFactionExpansion(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create a faction
	homePos := coords.LogicalPosition{X: 50, Y: 40}
	factionID := CreateFaction(manager, FactionOrcs, homePos, 10) // High strength to enable expansion

	factionEntity := manager.FindEntityByID(factionID)
	if factionEntity == nil {
		t.Fatal("Faction entity not found")
	}

	factionData := GetFactionByID(manager, factionID)
	initialSize := factionData.TerritorySize

	// Execute expansion
	ExpandTerritory(manager, factionEntity, factionData)

	// Verify territory grew
	if factionData.TerritorySize <= initialSize {
		t.Errorf("Territory did not expand: %d <= %d", factionData.TerritorySize, initialSize)
	}

	if factionData.TerritorySize != 2 {
		t.Errorf("Expected territory size 2, got %d", factionData.TerritorySize)
	}
}

// TestFactionIntentEvaluation verifies AI decision-making
func TestFactionIntentEvaluation(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create a weak faction
	homePos := coords.LogicalPosition{X: 50, Y: 40}
	weakFactionID := CreateFaction(manager, FactionBeasts, homePos, 1) // Weak

	weakEntity := manager.FindEntityByID(weakFactionID)
	weakData := GetFactionByID(manager, weakFactionID)
	weakIntent := common.GetComponentType[*StrategicIntentData](weakEntity, StrategicIntentComponent)

	// Evaluate intent
	EvaluateFactionIntent(manager, weakEntity, weakData, weakIntent)

	// Weak factions should fortify or idle
	if weakIntent.Intent != IntentFortify && weakIntent.Intent != IntentIdle {
		t.Errorf("Weak faction should fortify or idle, got %v", weakIntent.Intent)
	}

	// Create a strong faction
	strongFactionID := CreateFaction(manager, FactionBandits, coords.LogicalPosition{X: 60, Y: 40}, 15)

	strongEntity := manager.FindEntityByID(strongFactionID)
	strongData := GetFactionByID(manager, strongFactionID)
	strongIntent := common.GetComponentType[*StrategicIntentData](strongEntity, StrategicIntentComponent)

	// Evaluate intent
	EvaluateFactionIntent(manager, strongEntity, strongData, strongIntent)

	// Strong bandits should raid or expand
	if strongIntent.Intent != IntentRaid && strongIntent.Intent != IntentExpand {
		t.Errorf("Strong bandit faction should raid or expand, got %v", strongIntent.Intent)
	}
}

// TestFactionQueries verifies query functions
func TestFactionQueries(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create multiple factions
	CreateFaction(manager, FactionBandits, coords.LogicalPosition{X: 10, Y: 10}, 5)
	CreateFaction(manager, FactionOrcs, coords.LogicalPosition{X: 20, Y: 20}, 10)
	CreateFaction(manager, FactionBeasts, coords.LogicalPosition{X: 30, Y: 30}, 3)

	// Test count
	count := CountFactions(manager)
	if count != 3 {
		t.Errorf("Expected 3 factions, got %d", count)
	}

	// Test strongest
	_, strongestData := GetStrongestFaction(manager)
	if strongestData == nil {
		t.Fatal("GetStrongestFaction returned nil")
	}

	if strongestData.Strength != 10 {
		t.Errorf("Expected strongest faction to have strength 10, got %d", strongestData.Strength)
	}

	// Test weakest
	_, weakestData := GetWeakestFaction(manager)
	if weakestData == nil {
		t.Fatal("GetWeakestFaction returned nil")
	}

	if weakestData.Strength != 3 {
		t.Errorf("Expected weakest faction to have strength 3, got %d", weakestData.Strength)
	}
}

// TestInfluenceCache verifies caching optimization
func TestInfluenceCache(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create ECS-managed influence cache
	CreateInfluenceCacheEntity(manager, 100, 80)
	cache := GetInfluenceCache(manager)
	if cache == nil {
		t.Fatal("Failed to create influence cache")
	}

	// Create threat with influence
	threatPos := coords.LogicalPosition{X: 50, Y: 40}
	CreateThreatNode(manager, threatPos, ThreatNecromancer, 3, 0)

	// Initially dirty
	if !cache.IsDirty() {
		t.Error("Cache should start dirty")
	}

	// Get influence (should rebuild)
	influence := cache.GetInfluenceAt(manager, threatPos)
	if influence <= 0.0 {
		t.Errorf("Expected positive influence at threat position, got %f", influence)
	}

	// Cache should be clean after rebuild
	if cache.IsDirty() {
		t.Error("Cache should be clean after rebuild")
	}

	// Get influence again (should use cache)
	influence2 := cache.GetInfluenceAt(manager, threatPos)
	if influence != influence2 {
		t.Errorf("Cached influence mismatch: %f != %f", influence, influence2)
	}

	// Create new threat (should mark dirty)
	CreateThreatNode(manager, coords.LogicalPosition{X: 60, Y: 40}, ThreatBanditCamp, 2, 0)
	cache.Update(manager)

	if !cache.IsDirty() {
		t.Error("Cache should be dirty after new threat")
	}
}

// TestFactionUpdate verifies tick-based updates
func TestFactionUpdate(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create faction
	factionID := CreateFaction(manager, FactionOrcs, coords.LogicalPosition{X: 50, Y: 40}, 8)

	factionData := GetFactionByID(manager, factionID)
	initialStrength := factionData.Strength
	initialTerritory := factionData.TerritorySize

	// Update factions for 20 ticks
	for i := 0; i < 20; i++ {
		err := UpdateFactions(manager, int64(i))
		if err != nil {
			t.Fatalf("UpdateFactions failed: %v", err)
		}
	}

	// Verify faction changed (either expanded or fortified)
	if factionData.Strength == initialStrength && factionData.TerritorySize == initialTerritory {
		t.Error("Faction should have changed after 20 ticks")
	}
}
