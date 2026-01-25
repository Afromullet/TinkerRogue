package overworld

import (
	"game_main/common"
	testfx "game_main/testing"
	"game_main/world/coords"
	"testing"
)

func TestCheckVictoryCondition_AllThreatsEliminated(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create victory state
	victoryEntity := manager.World.NewEntity()
	victoryEntity.AddComponent(VictoryStateComponent, &VictoryStateData{
		Condition:       VictoryNone,
		VictoryAchieved: false,
	})

	// No threats exist
	condition := CheckVictoryCondition(manager)

	if condition != VictoryPlayerWins {
		t.Errorf("Expected VictoryPlayerWins when all threats eliminated, got %v", condition)
	}

	// Verify victory state updated
	victoryState := GetVictoryState(manager)
	if !victoryState.VictoryAchieved {
		t.Error("Expected VictoryAchieved=true")
	}

	if victoryState.Condition != VictoryPlayerWins {
		t.Errorf("Expected Condition=VictoryPlayerWins, got %v", victoryState.Condition)
	}
}

func TestCheckVictoryCondition_SurvivalVictory(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create tick state
	tickEntity := manager.World.NewEntity()
	tickEntity.AddComponent(TickStateComponent, &TickStateData{
		CurrentTick: 100,
		IsGameOver:  false,
	})

	// Create victory state with survival condition
	victoryEntity := manager.World.NewEntity()
	victoryEntity.AddComponent(VictoryStateComponent, &VictoryStateData{
		Condition:       VictoryTimeLimit,
		TicksToSurvive:  50, // Player only needs to survive 50 ticks
		VictoryAchieved: false,
	})

	// Create a threat (player hasn't eliminated all threats)
	CreateThreatNode(manager, coords.LogicalPosition{X: 10, Y: 10}, ThreatNecromancer, 1, 0)

	condition := CheckVictoryCondition(manager)

	// Should achieve survival victory even with threats remaining
	if condition != VictoryTimeLimit {
		t.Errorf("Expected VictoryTimeLimit after surviving required ticks, got %v", condition)
	}

	victoryState := GetVictoryState(manager)
	if !victoryState.VictoryAchieved {
		t.Error("Expected VictoryAchieved=true for survival victory")
	}
}

func TestCheckVictoryCondition_FactionDefeat(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create victory state targeting Bandits (not FactionNecromancers which is 0)
	victoryEntity := manager.World.NewEntity()
	victoryEntity.AddComponent(VictoryStateComponent, &VictoryStateData{
		Condition:         VictoryNone,
		TargetFactionType: FactionBandits, // Use non-zero faction type
		TicksToSurvive:    0,               // No survival victory
		VictoryAchieved:   false,
	})

	// Create threats but no Bandit faction
	// This prevents "all threats eliminated" victory from triggering
	// But Bandit faction doesn't exist, so faction defeat condition is met
	CreateThreatNode(manager, coords.LogicalPosition{X: 10, Y: 10}, ThreatNecromancer, 1, 0)

	condition := CheckVictoryCondition(manager)

	// Should achieve faction defeat victory (target faction doesn't exist, but other threats do)
	if condition != VictoryFactionDefeat {
		t.Errorf("Expected VictoryFactionDefeat when target faction eliminated, got %v", condition)
	}

	victoryState := GetVictoryState(manager)
	if !victoryState.VictoryAchieved {
		t.Error("Expected VictoryAchieved=true")
	}
}

func TestCheckVictoryCondition_FactionDefeat_TargetStillExists(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create victory state targeting Necromancers
	victoryEntity := manager.World.NewEntity()
	victoryEntity.AddComponent(VictoryStateComponent, &VictoryStateData{
		Condition:         VictoryFactionDefeat,
		TargetFactionType: FactionNecromancers,
		VictoryAchieved:   false,
	})

	// Create target faction (Necromancers still exist)
	CreateFaction(manager, FactionNecromancers, coords.LogicalPosition{X: 10, Y: 10}, 5)

	condition := CheckVictoryCondition(manager)

	// Should NOT win while target faction exists
	if condition != VictoryNone {
		t.Errorf("Expected VictoryNone while target faction exists, got %v", condition)
	}
}

func TestIsPlayerDefeated_HighInfluence(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create many high-intensity threats for high total influence
	for i := 0; i < 15; i++ {
		CreateThreatNode(manager, coords.LogicalPosition{X: i * 5, Y: i * 5}, ThreatNecromancer, 8, 0)
	}

	defeated := IsPlayerDefeated(manager)

	if !defeated {
		t.Error("Expected player defeated with high total influence")
	}
}

func TestIsPlayerDefeated_ManyPowerfulThreats(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create 6 tier-8+ threats (exceeds threshold of 5)
	for i := 0; i < 6; i++ {
		CreateThreatNode(manager, coords.LogicalPosition{X: i * 10, Y: i * 10}, ThreatNecromancer, 9, 0)
	}

	defeated := IsPlayerDefeated(manager)

	if !defeated {
		t.Error("Expected player defeated with many powerful threats")
	}
}

func TestIsPlayerDefeated_SafeState(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create a few low-intensity threats
	CreateThreatNode(manager, coords.LogicalPosition{X: 10, Y: 10}, ThreatBanditCamp, 2, 0)
	CreateThreatNode(manager, coords.LogicalPosition{X: 20, Y: 20}, ThreatBeastNest, 1, 0)

	defeated := IsPlayerDefeated(manager)

	if defeated {
		t.Error("Expected player NOT defeated with low threat level")
	}
}

func TestHasPlayerEliminatedAllThreats(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// No threats
	if !HasPlayerEliminatedAllThreats(manager) {
		t.Error("Expected true when no threats exist")
	}

	// Add a threat
	CreateThreatNode(manager, coords.LogicalPosition{X: 10, Y: 10}, ThreatNecromancer, 1, 0)

	if HasPlayerEliminatedAllThreats(manager) {
		t.Error("Expected false when threats exist")
	}
}

func TestHasPlayerDefeatedFactionType(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// No Necromancer factions exist
	if !HasPlayerDefeatedFactionType(manager, FactionNecromancers) {
		t.Error("Expected true when faction type doesn't exist")
	}

	// Add Necromancer faction
	CreateFaction(manager, FactionNecromancers, coords.LogicalPosition{X: 10, Y: 10}, 5)

	if HasPlayerDefeatedFactionType(manager, FactionNecromancers) {
		t.Error("Expected false when faction type exists")
	}

	// Check different faction type (Bandits don't exist)
	if !HasPlayerDefeatedFactionType(manager, FactionBandits) {
		t.Error("Expected true for faction type that doesn't exist")
	}
}

func TestGetDefeatReason_HighInfluence(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create many threats for high influence
	for i := 0; i < 15; i++ {
		CreateThreatNode(manager, coords.LogicalPosition{X: i * 5, Y: i * 5}, ThreatNecromancer, 7, 0)
	}

	reason := GetDefeatReason(manager)

	// Should mention influence
	if reason == "" {
		t.Error("Expected non-empty defeat reason")
	}
}

func TestGetDefeatReason_ManyPowerfulThreats(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create many tier-8+ threats
	for i := 0; i < 10; i++ {
		CreateThreatNode(manager, coords.LogicalPosition{X: i * 10, Y: i * 10}, ThreatNecromancer, 9, 0)
	}

	reason := GetDefeatReason(manager)

	if reason == "" {
		t.Error("Expected non-empty defeat reason")
	}
}

func TestGetDefeatReason_NoDefeat(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// No threats, player not defeated
	reason := GetDefeatReason(manager)

	if reason != "Defeat! Unknown reason" {
		t.Errorf("Expected 'Defeat! Unknown reason' when player not defeated, got %q", reason)
	}
}

func TestGetVictoryProgress(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// No victory state
	progress := GetVictoryProgress(manager)
	if progress != "No victory condition set" {
		t.Errorf("Expected 'No victory condition set', got %q", progress)
	}

	// Create victory state
	victoryEntity := manager.World.NewEntity()
	victoryEntity.AddComponent(VictoryStateComponent, &VictoryStateData{
		Condition:       VictoryNone,
		VictoryAchieved: false,
	})

	// Add some threats
	CreateThreatNode(manager, coords.LogicalPosition{X: 10, Y: 10}, ThreatNecromancer, 1, 0)
	CreateThreatNode(manager, coords.LogicalPosition{X: 20, Y: 20}, ThreatBanditCamp, 1, 0)

	progress = GetVictoryProgress(manager)

	// Should show threat count
	if progress == "" {
		t.Error("Expected non-empty progress string")
	}
}

func TestCreateVictoryStateEntity(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	entityID := CreateVictoryStateEntity(manager, 100, FactionNecromancers)

	if entityID == 0 {
		t.Fatal("Failed to create victory state entity")
	}

	victoryState := GetVictoryState(manager)
	if victoryState == nil {
		t.Fatal("Failed to retrieve victory state")
	}

	if victoryState.Condition != VictoryNone {
		t.Errorf("Expected Condition=VictoryNone initially, got %v", victoryState.Condition)
	}

	if victoryState.TicksToSurvive != 100 {
		t.Errorf("Expected TicksToSurvive=100, got %d", victoryState.TicksToSurvive)
	}

	if victoryState.TargetFactionType != FactionNecromancers {
		t.Errorf("Expected TargetFactionType=FactionNecromancers, got %v", victoryState.TargetFactionType)
	}

	if victoryState.VictoryAchieved {
		t.Error("Expected VictoryAchieved=false initially")
	}
}

func TestMultipleVictoryConditions(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create tick state (100 ticks elapsed)
	tickEntity := manager.World.NewEntity()
	tickEntity.AddComponent(TickStateComponent, &TickStateData{
		CurrentTick: 100,
		IsGameOver:  false,
	})

	// Create victory state: survival (50 ticks) AND all threats eliminated
	victoryEntity := manager.World.NewEntity()
	victoryEntity.AddComponent(VictoryStateComponent, &VictoryStateData{
		Condition:       VictoryTimeLimit,
		TicksToSurvive:  50,
		VictoryAchieved: false,
	})

	// No threats (both conditions met)
	condition := CheckVictoryCondition(manager)

	// Survival victory takes priority
	if condition != VictoryTimeLimit {
		t.Errorf("Expected VictoryTimeLimit (survival priority), got %v", condition)
	}
}
