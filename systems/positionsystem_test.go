package systems

import (
	"game_main/coords"
	"testing"

	"github.com/bytearena/ecs"
)

// BenchmarkPositionSystem_GetEntityIDAt benchmarks the O(1) PositionSystem lookup
func BenchmarkPositionSystem_GetEntityIDAt(b *testing.B) {
	manager := ecs.NewManager()
	ps := NewPositionSystem(manager)

	// Setup: Create 50 entities at different positions (simulating 50 monsters on map)
	for i := 0; i < 50; i++ {
		pos := coords.LogicalPosition{X: i % 10, Y: i / 10}
		ps.AddEntity(ecs.EntityID(i+1), pos)
	}

	// Benchmark: Look up entity at position (0, 0)
	searchPos := coords.LogicalPosition{X: 0, Y: 0}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ps.GetEntityIDAt(searchPos)
	}
}

// BenchmarkPositionSystem_GetEntityIDAt_LargeGrid benchmarks with 100 entities
func BenchmarkPositionSystem_GetEntityIDAt_LargeGrid(b *testing.B) {
	manager := ecs.NewManager()
	ps := NewPositionSystem(manager)

	// Setup: Create 100 entities (simulating heavy combat scenario)
	for i := 0; i < 100; i++ {
		pos := coords.LogicalPosition{X: i % 20, Y: i / 20}
		ps.AddEntity(ecs.EntityID(i+1), pos)
	}

	// Benchmark: Look up entity at middle position
	searchPos := coords.LogicalPosition{X: 10, Y: 2}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ps.GetEntityIDAt(searchPos)
	}
}

// BenchmarkPositionSystem_MoveEntity benchmarks entity movement
func BenchmarkPositionSystem_MoveEntity(b *testing.B) {
	manager := ecs.NewManager()
	ps := NewPositionSystem(manager)

	// Setup: Create 50 entities
	for i := 0; i < 50; i++ {
		pos := coords.LogicalPosition{X: i % 10, Y: i / 10}
		ps.AddEntity(ecs.EntityID(i+1), pos)
	}

	// Benchmark: Move entity from (0,0) to (1,1) repeatedly
	oldPos := coords.LogicalPosition{X: 0, Y: 0}
	newPos := coords.LogicalPosition{X: 1, Y: 1}
	entityID := ecs.EntityID(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.MoveEntity(entityID, oldPos, newPos)
		ps.MoveEntity(entityID, newPos, oldPos) // Move back
	}
}

// TestPositionSystem_BasicOperations tests all basic CRUD operations
func TestPositionSystem_BasicOperations(t *testing.T) {
	manager := ecs.NewManager()
	ps := NewPositionSystem(manager)

	// Test Add
	pos1 := coords.LogicalPosition{X: 5, Y: 10}
	entityID1 := ecs.EntityID(42)
	err := ps.AddEntity(entityID1, pos1)
	if err != nil {
		t.Fatalf("AddEntity failed: %v", err)
	}

	// Test GetEntityIDAt
	foundID := ps.GetEntityIDAt(pos1)
	if foundID != entityID1 {
		t.Errorf("Expected entity ID %d, got %d", entityID1, foundID)
	}

	// Test GetEntityIDAt for non-existent position
	emptyPos := coords.LogicalPosition{X: 99, Y: 99}
	foundID = ps.GetEntityIDAt(emptyPos)
	if foundID != 0 {
		t.Errorf("Expected 0 for empty position, got %d", foundID)
	}

	// Test Move
	pos2 := coords.LogicalPosition{X: 6, Y: 11}
	err = ps.MoveEntity(entityID1, pos1, pos2)
	if err != nil {
		t.Fatalf("MoveEntity failed: %v", err)
	}

	// Verify entity moved
	foundID = ps.GetEntityIDAt(pos2)
	if foundID != entityID1 {
		t.Errorf("Entity not found at new position. Expected %d, got %d", entityID1, foundID)
	}

	// Verify old position is empty
	foundID = ps.GetEntityIDAt(pos1)
	if foundID != 0 {
		t.Errorf("Old position should be empty, got %d", foundID)
	}

	// Test Remove
	err = ps.RemoveEntity(entityID1, pos2)
	if err != nil {
		t.Fatalf("RemoveEntity failed: %v", err)
	}

	// Verify entity removed
	foundID = ps.GetEntityIDAt(pos2)
	if foundID != 0 {
		t.Errorf("Entity should be removed, got %d", foundID)
	}
}

// TestPositionSystem_MultipleEntitiesAtPosition tests stacking entities
func TestPositionSystem_MultipleEntitiesAtPosition(t *testing.T) {
	manager := ecs.NewManager()
	ps := NewPositionSystem(manager)

	pos := coords.LogicalPosition{X: 5, Y: 5}
	entity1 := ecs.EntityID(1)
	entity2 := ecs.EntityID(2)
	entity3 := ecs.EntityID(3)

	// Add multiple entities at same position (e.g., items stacked)
	ps.AddEntity(entity1, pos)
	ps.AddEntity(entity2, pos)
	ps.AddEntity(entity3, pos)

	// GetEntityIDAt should return first entity
	foundID := ps.GetEntityIDAt(pos)
	if foundID != entity1 && foundID != entity2 && foundID != entity3 {
		t.Errorf("Expected one of the entities, got %d", foundID)
	}

	// GetAllEntityIDsAt should return all entities
	allIDs := ps.GetAllEntityIDsAt(pos)
	if len(allIDs) != 3 {
		t.Errorf("Expected 3 entities at position, got %d", len(allIDs))
	}
}

// TestPositionSystem_GetEntitiesInRadius tests radius queries
func TestPositionSystem_GetEntitiesInRadius(t *testing.T) {
	manager := ecs.NewManager()
	ps := NewPositionSystem(manager)

	// Place entities in a 3x3 grid
	for x := 0; x < 3; x++ {
		for y := 0; y < 3; y++ {
			pos := coords.LogicalPosition{X: x, Y: y}
			entityID := ecs.EntityID(x*3 + y + 1)
			ps.AddEntity(entityID, pos)
		}
	}

	// Query center position with radius 1 (should get all 9 entities in 3x3)
	center := coords.LogicalPosition{X: 1, Y: 1}
	entities := ps.GetEntitiesInRadius(center, 1)

	if len(entities) != 9 {
		t.Errorf("Expected 9 entities in radius 1, got %d", len(entities))
	}

	// Query corner with radius 1 (should get 4 entities)
	corner := coords.LogicalPosition{X: 0, Y: 0}
	entities = ps.GetEntitiesInRadius(corner, 1)

	if len(entities) != 4 {
		t.Errorf("Expected 4 entities at corner radius 1, got %d", len(entities))
	}
}

// TestPositionSystem_Clear tests clearing all entities
func TestPositionSystem_Clear(t *testing.T) {
	manager := ecs.NewManager()
	ps := NewPositionSystem(manager)

	// Add several entities
	for i := 0; i < 10; i++ {
		pos := coords.LogicalPosition{X: i, Y: 0}
		ps.AddEntity(ecs.EntityID(i+1), pos)
	}

	// Verify entities exist
	if ps.GetEntityCount() != 10 {
		t.Errorf("Expected 10 entities, got %d", ps.GetEntityCount())
	}

	// Clear
	ps.Clear()

	// Verify all entities removed
	if ps.GetEntityCount() != 0 {
		t.Errorf("Expected 0 entities after clear, got %d", ps.GetEntityCount())
	}

	// Verify position is empty
	pos := coords.LogicalPosition{X: 5, Y: 0}
	if ps.GetEntityIDAt(pos) != 0 {
		t.Errorf("Position should be empty after clear")
	}
}
