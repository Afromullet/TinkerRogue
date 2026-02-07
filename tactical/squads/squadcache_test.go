package squads

import (
	"game_main/common"
	testfx "game_main/testing"
	"testing"

	"github.com/bytearena/ecs"
)

// setupTestSquads creates a test environment with multiple squads and members
func setupTestSquads(t *testing.B, numSquads, unitsPerSquad int) (*common.EntityManager, []ecs.EntityID) {
	// Use the same setup pattern as existing tests
	manager := testfx.NewTestEntityManager()
	if err := InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}

	squadIDs := make([]ecs.EntityID, numSquads)

	// Create squads
	for i := 0; i < numSquads; i++ {
		squadEntity := manager.World.NewEntity()
		squadID := squadEntity.GetID()
		squadIDs[i] = squadID

		// Add squad component
		squadData := &SquadData{
			SquadID: squadID,
			Name:    "Test Squad",
		}
		squadEntity.AddComponent(SquadComponent, squadData)

		// Create members for this squad
		for j := 0; j < unitsPerSquad; j++ {
			memberEntity := manager.World.NewEntity()
			memberData := &SquadMemberData{
				SquadID: squadID,
			}
			memberEntity.AddComponent(SquadMemberComponent, memberData)

			// Add attributes
			attrs := common.NewAttributes(
				10, // Strength
				10, // Dexterity
				10, // Magic
				10, // Leadership
				0,  // Armor
				0,  // Weapon
			)
			memberEntity.AddComponent(common.AttributeComponent, &attrs)

			// Make first member the leader
			if j == 0 {
				memberEntity.AddComponent(LeaderComponent, &LeaderData{
					Leadership: 10,
					Experience: 0,
				})
			}
		}
	}

	return manager, squadIDs
}

// ========================================
// BENCHMARKS: GetSquadEntity
// ========================================

// BenchmarkGetSquadEntity_NoCache tests the original World.Query() approach
func BenchmarkGetSquadEntity_NoCache(b *testing.B) {
	manager, squadIDs := setupTestSquads(b, 20, 9)
	targetSquadID := squadIDs[10] // Middle squad

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entity := GetSquadEntity(targetSquadID, manager)
		if entity == nil {
			b.Fatal("squad not found")
		}
	}
}

// BenchmarkGetSquadEntity_WithCache tests the new view-based approach
func BenchmarkGetSquadEntity_WithCache(b *testing.B) {
	manager, squadIDs := setupTestSquads(b, 20, 9)
	cache := NewSquadQueryCache(manager)
	targetSquadID := squadIDs[10] // Middle squad

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entity := cache.GetSquadEntity(targetSquadID)
		if entity == nil {
			b.Fatal("squad not found")
		}
	}
}

// ========================================
// BENCHMARKS: GetUnitIDsInSquad
// ========================================

// BenchmarkGetUnitIDsInSquad_NoCache tests the original World.Query() approach
func BenchmarkGetUnitIDsInSquad_NoCache(b *testing.B) {
	manager, squadIDs := setupTestSquads(b, 20, 9)
	targetSquadID := squadIDs[10]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		unitIDs := GetUnitIDsInSquad(targetSquadID, manager)
		if len(unitIDs) != 9 {
			b.Fatalf("expected 9 units, got %d", len(unitIDs))
		}
	}
}

// BenchmarkGetUnitIDsInSquad_WithCache tests the new view-based approach
func BenchmarkGetUnitIDsInSquad_WithCache(b *testing.B) {
	manager, squadIDs := setupTestSquads(b, 20, 9)
	cache := NewSquadQueryCache(manager)
	targetSquadID := squadIDs[10]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		unitIDs := cache.GetUnitIDsInSquad(targetSquadID)
		if len(unitIDs) != 9 {
			b.Fatalf("expected 9 units, got %d", len(unitIDs))
		}
	}
}

// ========================================
// BENCHMARKS: GetLeaderID
// ========================================

// BenchmarkGetLeaderID_NoCache tests the original World.Query() approach
func BenchmarkGetLeaderID_NoCache(b *testing.B) {
	manager, squadIDs := setupTestSquads(b, 20, 9)
	targetSquadID := squadIDs[10]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leaderID := GetLeaderID(targetSquadID, manager)
		if leaderID == 0 {
			b.Fatal("leader not found")
		}
	}
}

// BenchmarkGetLeaderID_WithCache tests the new view-based approach
func BenchmarkGetLeaderID_WithCache(b *testing.B) {
	manager, squadIDs := setupTestSquads(b, 20, 9)
	cache := NewSquadQueryCache(manager)
	targetSquadID := squadIDs[10]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leaderID := cache.GetLeaderID(targetSquadID)
		if leaderID == 0 {
			b.Fatal("leader not found")
		}
	}
}

// ========================================
// BENCHMARKS: FindAllSquads
// ========================================

// BenchmarkFindAllSquads_WithCache tests the new view-based approach
func BenchmarkFindAllSquads_WithCache(b *testing.B) {
	manager, _ := setupTestSquads(b, 20, 9)
	cache := NewSquadQueryCache(manager)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		squadIDs := cache.FindAllSquads()
		if len(squadIDs) != 20 {
			b.Fatalf("expected 20 squads, got %d", len(squadIDs))
		}
	}
}

// ========================================
// BENCHMARKS: Multiple Queries Per Frame (Realistic Scenario)
// ========================================

// BenchmarkMultipleQueries_NoCache simulates a frame with multiple squad queries (no cache)
func BenchmarkMultipleQueries_NoCache(b *testing.B) {
	manager, squadIDs := setupTestSquads(b, 20, 9)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate 15 GetSquadEntity calls per frame
		for j := 0; j < 15; j++ {
			squadIdx := j % len(squadIDs)
			_ = GetSquadEntity(squadIDs[squadIdx], manager)
		}

		// Simulate 20 GetUnitIDsInSquad calls per frame
		for j := 0; j < 20; j++ {
			squadIdx := j % len(squadIDs)
			_ = GetUnitIDsInSquad(squadIDs[squadIdx], manager)
		}

		// Simulate 10 GetLeaderID calls per frame
		for j := 0; j < 10; j++ {
			squadIdx := j % len(squadIDs)
			_ = GetLeaderID(squadIDs[squadIdx], manager)
		}
	}
}

// BenchmarkMultipleQueries_WithCache simulates a frame with multiple squad queries (with cache)
func BenchmarkMultipleQueries_WithCache(b *testing.B) {
	manager, squadIDs := setupTestSquads(b, 20, 9)
	cache := NewSquadQueryCache(manager)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate 15 GetSquadEntity calls per frame
		for j := 0; j < 15; j++ {
			squadIdx := j % len(squadIDs)
			_ = cache.GetSquadEntity(squadIDs[squadIdx])
		}

		// Simulate 20 GetUnitIDsInSquad calls per frame
		for j := 0; j < 20; j++ {
			squadIdx := j % len(squadIDs)
			_ = cache.GetUnitIDsInSquad(squadIDs[squadIdx])
		}

		// Simulate 10 GetLeaderID calls per frame
		for j := 0; j < 10; j++ {
			squadIdx := j % len(squadIDs)
			_ = cache.GetLeaderID(squadIDs[squadIdx])
		}
	}
}
