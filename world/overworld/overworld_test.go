package overworld

import (
	"game_main/common"
	testfx "game_main/testing"
	"game_main/world/coords"
	"testing"
)

func TestTickStateCreation(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	tickID := CreateTickStateEntity(manager)
	if tickID == 0 {
		t.Fatal("Failed to create tick state entity")
	}

	tickState := GetTickState(manager)
	if tickState == nil {
		t.Fatal("Failed to retrieve tick state")
	}

	if tickState.CurrentTick != 0 {
		t.Errorf("Expected CurrentTick = 0, got %d", tickState.CurrentTick)
	}

	if tickState.IsGameOver {
		t.Error("Expected IsGameOver = false")
	}
}

func TestThreatNodeCreation(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	pos := coords.LogicalPosition{X: 10, Y: 10}
	threatID := CreateThreatNode(manager, pos, ThreatNecromancer, 1, 0)

	if threatID == 0 {
		t.Fatal("Failed to create threat node")
	}

	// Verify threat can be retrieved
	threat := GetThreatNodeByID(manager, threatID)
	if threat == nil {
		t.Fatal("Failed to retrieve created threat node")
	}

	data := common.GetComponentType[*ThreatNodeData](threat, ThreatNodeComponent)
	if data == nil {
		t.Fatal("Threat node missing ThreatNodeData component")
	}

	if data.ThreatType != ThreatNecromancer {
		t.Errorf("Expected ThreatNecromancer, got %v", data.ThreatType)
	}

	if data.Intensity != 1 {
		t.Errorf("Expected Intensity = 1, got %d", data.Intensity)
	}

	// Verify position system registration
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(pos)
	found := false
	for _, id := range entityIDs {
		if id == threatID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Threat node not registered in position system")
	}
}

func TestThreatEvolution(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	pos := coords.LogicalPosition{X: 10, Y: 10}
	threatID := CreateThreatNode(manager, pos, ThreatNecromancer, 1, 0)

	threat := GetThreatNodeByID(manager, threatID)
	data := common.GetComponentType[*ThreatNodeData](threat, ThreatNodeComponent)

	initialIntensity := data.Intensity

	// Manually set growth progress to trigger evolution
	data.GrowthProgress = 1.0

	// Advance one tick
	err := UpdateThreatNodes(manager, 1)
	if err != nil {
		t.Fatalf("UpdateThreatNodes failed: %v", err)
	}

	// Verify intensity increased
	if data.Intensity != initialIntensity+1 {
		t.Errorf("Expected Intensity = %d, got %d", initialIntensity+1, data.Intensity)
	}

	// Verify growth progress reset
	if data.GrowthProgress != 0.0 {
		t.Errorf("Expected GrowthProgress = 0.0, got %f", data.GrowthProgress)
	}
}

func TestTickAdvancement(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	tickState := GetTickState(manager)
	initialTick := tickState.CurrentTick

	// Advance tick
	err := AdvanceTick(manager)
	if err != nil {
		t.Fatalf("AdvanceTick failed: %v", err)
	}

	if tickState.CurrentTick != initialTick+1 {
		t.Errorf("Expected CurrentTick = %d, got %d", initialTick+1, tickState.CurrentTick)
	}
}

func TestThreatQueries(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create multiple threats
	pos1 := coords.LogicalPosition{X: 10, Y: 10}
	pos2 := coords.LogicalPosition{X: 15, Y: 15}
	pos3 := coords.LogicalPosition{X: 20, Y: 20}

	CreateThreatNode(manager, pos1, ThreatNecromancer, 1, 0)
	CreateThreatNode(manager, pos2, ThreatBanditCamp, 2, 0)
	CreateThreatNode(manager, pos3, ThreatNecromancer, 3, 0)

	// Test GetAllThreatNodes
	allThreats := GetAllThreatNodes(manager)
	if len(allThreats) != 3 {
		t.Errorf("Expected 3 threats, got %d", len(allThreats))
	}

	// Test CountThreatNodes
	count := CountThreatNodes(manager)
	if count != 3 {
		t.Errorf("Expected count = 3, got %d", count)
	}

	// Test GetThreatsByType
	necromancers := GetThreatsByType(manager, ThreatNecromancer)
	if len(necromancers) != 2 {
		t.Errorf("Expected 2 necromancers, got %d", len(necromancers))
	}

	bandits := GetThreatsByType(manager, ThreatBanditCamp)
	if len(bandits) != 1 {
		t.Errorf("Expected 1 bandit camp, got %d", len(bandits))
	}

	// Test GetThreatNodeAt
	threat := GetThreatNodeAt(manager, pos1)
	if threat == nil {
		t.Error("Failed to get threat at position")
	}

	// Test GetThreatsInRadius
	centerPos := coords.LogicalPosition{X: 10, Y: 10}
	// Threats at (10,10), (15,15), (20,20)
	// Chebyshev distances: 0, 5, 10
	// Radius 10 should include all 3
	nearby := GetThreatsInRadius(manager, centerPos, 10)
	if len(nearby) != 3 {
		t.Errorf("Expected 3 threats in radius 10, got %d", len(nearby))
	}

	// Test with smaller radius - should only get (10,10) and (15,15)
	nearby = GetThreatsInRadius(manager, centerPos, 9)
	if len(nearby) != 2 {
		t.Errorf("Expected 2 threats in radius 9, got %d", len(nearby))
	}
}

func TestGameOverPreventsTickAdvancement(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	tickState := GetTickState(manager)

	// Set game over state
	tickState.IsGameOver = true

	// Try to advance tick while game is over
	initialTick := tickState.CurrentTick
	err := AdvanceTick(manager)
	if err != nil {
		t.Fatalf("AdvanceTick failed: %v", err)
	}

	// Tick should not advance when game is over
	if tickState.CurrentTick != initialTick {
		t.Errorf("Tick advanced while game over: %d -> %d", initialTick, tickState.CurrentTick)
	}

	// Clear game over state
	tickState.IsGameOver = false

	// Tick should advance now
	err = AdvanceTick(manager)
	if err != nil {
		t.Fatalf("AdvanceTick failed: %v", err)
	}

	if tickState.CurrentTick != initialTick+1 {
		t.Errorf("Expected tick to advance after clearing game over, got %d", tickState.CurrentTick)
	}
}

func TestChildNodeSpawning(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	pos := coords.LogicalPosition{X: 50, Y: 50}
	threatID := CreateThreatNode(manager, pos, ThreatNecromancer, 2, 0)

	threat := GetThreatNodeByID(manager, threatID)
	data := common.GetComponentType[*ThreatNodeData](threat, ThreatNodeComponent)

	// Set intensity to 3 to trigger child spawning
	data.Intensity = 2
	data.GrowthProgress = 1.0

	initialCount := CountThreatNodes(manager)

	// Trigger evolution (should spawn child at intensity 3)
	err := UpdateThreatNodes(manager, 1)
	if err != nil {
		t.Fatalf("UpdateThreatNodes failed: %v", err)
	}

	// Verify child was spawned
	newCount := CountThreatNodes(manager)
	if newCount != initialCount+1 {
		t.Errorf("Expected %d threats after child spawn, got %d", initialCount+1, newCount)
	}

	// Verify parent intensity increased
	if data.Intensity != 3 {
		t.Errorf("Expected parent intensity = 3, got %d", data.Intensity)
	}
}
