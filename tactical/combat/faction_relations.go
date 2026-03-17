package combat

import "github.com/bytearena/ecs"

// FactionRelation describes the relationship between two factions.
type FactionRelation int

const (
	RelationHostile FactionRelation = iota
	RelationNeutral
	RelationAllied
)

// FactionRelationResolver determines relationships between factions.
// Default implementation is FreeForAllRelations (self=allied, everyone else=hostile).
// Swap in a custom resolver for alliance mechanics.
type FactionRelationResolver interface {
	GetRelation(factionA, factionB ecs.EntityID) FactionRelation
	GetHostileFactions(factionID ecs.EntityID, allFactions []ecs.EntityID) []ecs.EntityID
}

// FreeForAllRelations implements free-for-all: self is allied, everyone else is hostile.
// This produces identical behavior to the pre-existing inline "skip self" checks.
type FreeForAllRelations struct{}

func (f *FreeForAllRelations) GetRelation(factionA, factionB ecs.EntityID) FactionRelation {
	if factionA == factionB {
		return RelationAllied
	}
	return RelationHostile
}

func (f *FreeForAllRelations) GetHostileFactions(factionID ecs.EntityID, allFactions []ecs.EntityID) []ecs.EntityID {
	hostile := make([]ecs.EntityID, 0, len(allFactions))
	for _, id := range allFactions {
		if id != factionID {
			hostile = append(hostile, id)
		}
	}
	return hostile
}
