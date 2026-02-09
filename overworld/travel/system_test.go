package travel

import (
	"game_main/common"
	"game_main/config"
	"game_main/overworld/core"
	"game_main/world/coords"
	"math"
	"testing"

	"github.com/bytearena/ecs"
)

// setupTestEnvironment creates a minimal ECS environment for testing
func setupTestEnvironment() (*common.EntityManager, *common.PlayerData) {
	// Create entity manager
	manager := common.NewEntityManager()

	// Initialize only required components for testing
	common.PositionComponent = manager.World.NewComponent()
	common.AttributeComponent = manager.World.NewComponent()
	core.TravelStateComponent = manager.World.NewComponent()
	core.TravelStateTag = ecs.BuildTag(core.TravelStateComponent)

	// Initialize position system
	common.GlobalPositionSystem = common.NewPositionSystem(manager.World)

	// Create travel state
	CreateTravelStateEntity(manager)

	// Create test player entity
	playerEntity := manager.World.NewEntity()
	playerEntityID := playerEntity.GetID()

	// Add position component
	playerPos := coords.LogicalPosition{X: 0, Y: 0}
	playerEntity.AddComponent(common.PositionComponent, &playerPos)

	// Add attributes component with default MovementSpeed
	attr := common.Attributes{
		Strength:      10,
		Dexterity:     10,
		Magic:         10,
		Leadership:    10,
		Armor:         5,
		Weapon:        5,
		MovementSpeed: config.DefaultMovementSpeed, // Default: 3
		AttackRange:   1,
		CurrentHealth: 30,
		MaxHealth:     30,
		CanAct:        true,
	}
	playerEntity.AddComponent(common.AttributeComponent, &attr)

	// Register with position system
	common.GlobalPositionSystem.AddEntity(playerEntityID, playerPos)

	// Create player data
	playerData := &common.PlayerData{
		PlayerEntityID: playerEntityID,
	}

	return manager, playerData
}

func TestManhattanDistance(t *testing.T) {
	tests := []struct {
		name     string
		from     coords.LogicalPosition
		to       coords.LogicalPosition
		expected int
	}{
		{
			name:     "Origin to (3,4) should be 7",
			from:     coords.LogicalPosition{X: 0, Y: 0},
			to:       coords.LogicalPosition{X: 3, Y: 4},
			expected: 7,
		},
		{
			name:     "Same position should be 0",
			from:     coords.LogicalPosition{X: 5, Y: 5},
			to:       coords.LogicalPosition{X: 5, Y: 5},
			expected: 0,
		},
		{
			name:     "Horizontal distance (10,0) to (15,0) should be 5",
			from:     coords.LogicalPosition{X: 10, Y: 0},
			to:       coords.LogicalPosition{X: 15, Y: 0},
			expected: 5,
		},
		{
			name:     "Vertical distance (0,10) to (0,20) should be 10",
			from:     coords.LogicalPosition{X: 0, Y: 10},
			to:       coords.LogicalPosition{X: 0, Y: 20},
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manhattanDistance(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("manhattanDistance() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestStartTravel(t *testing.T) {
	manager, playerData := setupTestEnvironment()

	// Create a dummy threat entity
	threatEntity := manager.World.NewEntity()
	threatID := threatEntity.GetID()

	// Create a dummy encounter entity
	encounterEntity := manager.World.NewEntity()
	encounterID := encounterEntity.GetID()

	// Test starting travel to (3,4) — Manhattan distance = 7, speed 3 → ceil(7/3) = 3 ticks
	destination := coords.LogicalPosition{X: 3, Y: 4}
	err := StartTravel(manager, playerData, destination, threatID, encounterID)

	if err != nil {
		t.Fatalf("StartTravel() failed: %v", err)
	}

	// Verify travel state
	travelState := GetTravelState(manager)
	if travelState == nil {
		t.Fatal("Travel state not found")
	}

	if !travelState.IsTraveling {
		t.Error("Expected IsTraveling to be true")
	}

	// Manhattan distance = 7, speed = 3, ceil(7/3) = 3 ticks
	expectedTicks := int(math.Ceil(7.0 / 3.0))
	if travelState.TicksRemaining != expectedTicks {
		t.Errorf("TicksRemaining = %d, want %d", travelState.TicksRemaining, expectedTicks)
	}

	if travelState.TargetThreatID != threatID {
		t.Errorf("TargetThreatID = %d, want %d", travelState.TargetThreatID, threatID)
	}

	if travelState.TargetEncounterID != encounterID {
		t.Errorf("TargetEncounterID = %d, want %d", travelState.TargetEncounterID, encounterID)
	}
}

func TestTravelProgression(t *testing.T) {
	manager, playerData := setupTestEnvironment()

	// Create dummy entities
	threatEntity := manager.World.NewEntity()
	threatID := threatEntity.GetID()
	encounterEntity := manager.World.NewEntity()
	encounterID := encounterEntity.GetID()

	// Start travel: Manhattan distance from (0,0) to (12,9) = 21, speed 3 → 7 ticks
	destination := coords.LogicalPosition{X: 12, Y: 9}
	err := StartTravel(manager, playerData, destination, threatID, encounterID)
	if err != nil {
		t.Fatalf("StartTravel() failed: %v", err)
	}

	travelState := GetTravelState(manager)
	expectedTicks := travelState.TicksRemaining

	// Advance ticks
	for tick := 1; tick <= expectedTicks; tick++ {
		completed, err := AdvanceTravelTick(manager, playerData)
		if err != nil {
			t.Fatalf("AdvanceTravelTick() failed on tick %d: %v", tick, err)
		}

		if tick < expectedTicks {
			// Should not be completed yet
			if completed {
				t.Errorf("Travel completed early at tick %d (expected %d)", tick, expectedTicks)
			}
			if !travelState.IsTraveling {
				t.Errorf("IsTraveling became false at tick %d (expected %d)", tick, expectedTicks)
			}
		} else {
			// Final tick - should be completed
			if !completed {
				t.Errorf("Travel not completed at tick %d", tick)
			}
			if travelState.IsTraveling {
				t.Error("IsTraveling still true after completion")
			}
		}
	}

	// Verify player arrived at exact destination
	playerEntity := manager.FindEntityByID(playerData.PlayerEntityID)
	playerPos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)
	if playerPos.X != destination.X || playerPos.Y != destination.Y {
		t.Errorf("Player position = (%d,%d), want (%d,%d)",
			playerPos.X, playerPos.Y, destination.X, destination.Y)
	}
}

func TestTravelCancellation(t *testing.T) {
	manager, playerData := setupTestEnvironment()

	// Create dummy entities
	threatEntity := manager.World.NewEntity()
	threatID := threatEntity.GetID()
	encounterEntity := manager.World.NewEntity()
	encounterID := encounterEntity.GetID()

	// Record origin position
	playerEntity := manager.FindEntityByID(playerData.PlayerEntityID)
	originPos := *common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)

	// Start travel
	destination := coords.LogicalPosition{X: 20, Y: 20}
	err := StartTravel(manager, playerData, destination, threatID, encounterID)
	if err != nil {
		t.Fatalf("StartTravel() failed: %v", err)
	}

	// Advance 2 ticks (player doesn't move until arrival in tick-based system)
	for i := 0; i < 2; i++ {
		_, err := AdvanceTravelTick(manager, playerData)
		if err != nil {
			t.Fatalf("AdvanceTravelTick() failed: %v", err)
		}
	}

	// Cancel travel
	err = CancelTravel(manager, playerData)
	if err != nil {
		t.Fatalf("CancelTravel() failed: %v", err)
	}

	// Verify travel state cleared
	travelState := GetTravelState(manager)
	if travelState.IsTraveling {
		t.Error("IsTraveling still true after cancellation")
	}

	// Verify player returned to origin
	finalPos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)
	if finalPos.X != originPos.X || finalPos.Y != originPos.Y {
		t.Errorf("Player position = (%d,%d), want origin (%d,%d)",
			finalPos.X, finalPos.Y, originPos.X, originPos.Y)
	}

	// Verify encounter entity was disposed
	if manager.FindEntityByID(encounterID) != nil {
		t.Error("Encounter entity was not disposed")
	}
}

func TestZeroDistance(t *testing.T) {
	manager, playerData := setupTestEnvironment()

	// Create dummy entities
	threatEntity := manager.World.NewEntity()
	threatID := threatEntity.GetID()
	encounterEntity := manager.World.NewEntity()
	encounterID := encounterEntity.GetID()

	// Start travel to same position (distance = 0)
	destination := coords.LogicalPosition{X: 0, Y: 0}
	err := StartTravel(manager, playerData, destination, threatID, encounterID)
	if err != nil {
		t.Fatalf("StartTravel() failed: %v", err)
	}

	// Should complete immediately on first tick
	completed, err := AdvanceTravelTick(manager, playerData)
	if err != nil {
		t.Fatalf("AdvanceTravelTick() failed: %v", err)
	}

	if !completed {
		t.Error("Zero-distance travel should complete immediately")
	}

	travelState := GetTravelState(manager)
	if travelState.IsTraveling {
		t.Error("IsTraveling still true after zero-distance travel")
	}
}

func TestIsTraveling(t *testing.T) {
	manager, playerData := setupTestEnvironment()

	// Initially not traveling
	if IsTraveling(manager) {
		t.Error("IsTraveling() = true before starting travel")
	}

	// Create dummy entities and start travel
	threatEntity := manager.World.NewEntity()
	threatID := threatEntity.GetID()
	encounterEntity := manager.World.NewEntity()
	encounterID := encounterEntity.GetID()

	destination := coords.LogicalPosition{X: 10, Y: 10}
	StartTravel(manager, playerData, destination, threatID, encounterID)

	// Should be traveling now
	if !IsTraveling(manager) {
		t.Error("IsTraveling() = false after starting travel")
	}

	// Cancel travel
	CancelTravel(manager, playerData)

	// Should not be traveling anymore
	if IsTraveling(manager) {
		t.Error("IsTraveling() = true after canceling travel")
	}
}
