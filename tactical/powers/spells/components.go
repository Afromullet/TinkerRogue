package spells

import (
	"github.com/bytearena/ecs"
)

// Component and tag variables
var (
	ManaComponent      *ecs.Component
	SpellBookComponent *ecs.Component

	ManaTag      ecs.Tag
	SpellBookTag ecs.Tag
)

// ManaData tracks a squad's mana pool.
// Mana persists across battles, making mana management a strategic overworld decision.
// The squad leader uses this pool to cast spells.
type ManaData struct {
	CurrentMana int
	MaxMana     int
}

// SpellBookData holds references to spells a squad can cast via its leader.
// SpellIDs are keys into the global SpellRegistry.
type SpellBookData struct {
	SpellIDs []string
}

// SpellCastResult contains the outcome of casting a spell.
type SpellCastResult struct {
	Success          bool
	ErrorReason      string
	TotalDamageDealt int
	AffectedSquadIDs []ecs.EntityID
	SquadsDestroyed  []ecs.EntityID
}
