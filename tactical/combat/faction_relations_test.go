package combat

import (
	"testing"

	"github.com/bytearena/ecs"
)

func TestFreeForAllRelations_SelfIsAllied(t *testing.T) {
	r := &FreeForAllRelations{}
	if r.GetRelation(1, 1) != RelationAllied {
		t.Error("expected self-relation to be Allied")
	}
}

func TestFreeForAllRelations_OtherIsHostile(t *testing.T) {
	r := &FreeForAllRelations{}
	if r.GetRelation(1, 2) != RelationHostile {
		t.Error("expected other-faction relation to be Hostile")
	}
}

func TestFreeForAllRelations_GetHostileFactions(t *testing.T) {
	r := &FreeForAllRelations{}
	allFactions := []ecs.EntityID{10, 20, 30}

	hostile := r.GetHostileFactions(20, allFactions)
	if len(hostile) != 2 {
		t.Fatalf("expected 2 hostile factions, got %d", len(hostile))
	}
	if hostile[0] != 10 || hostile[1] != 30 {
		t.Errorf("expected [10, 30], got %v", hostile)
	}
}

func TestFreeForAllRelations_EmptyFactions(t *testing.T) {
	r := &FreeForAllRelations{}
	hostile := r.GetHostileFactions(1, []ecs.EntityID{})
	if len(hostile) != 0 {
		t.Errorf("expected 0 hostile factions for empty list, got %d", len(hostile))
	}
}

func TestFreeForAllRelations_SingleFaction(t *testing.T) {
	r := &FreeForAllRelations{}
	hostile := r.GetHostileFactions(5, []ecs.EntityID{5})
	if len(hostile) != 0 {
		t.Errorf("expected 0 hostile factions when alone, got %d", len(hostile))
	}
}

func TestAreFactionsHostile(t *testing.T) {
	manager := CreateTestCombatManager()
	cache := NewCombatQueryCache(manager)

	if !AreFactionsHostile(1, 2, cache) {
		t.Error("expected different factions to be hostile")
	}
	if AreFactionsHostile(1, 1, cache) {
		t.Error("expected same faction to not be hostile")
	}
}
