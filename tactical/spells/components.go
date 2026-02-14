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

// ManaData tracks a commander's mana pool.
// Mana persists across battles, making mana management a strategic overworld decision.
type ManaData struct {
	CurrentMana int
	MaxMana     int
}

// SpellBookData holds references to spells a commander can cast.
// SpellIDs are keys into the global SpellRegistry.
type SpellBookData struct {
	SpellIDs []string
}

// SpellCastResult contains the outcome of casting a spell.
type SpellCastResult struct {
	Success          bool
	ErrorReason      string
	SpellID          string
	SpellName        string
	DamageByUnit     map[ecs.EntityID]int // damage dealt to each unit
	SquadsDestroyed  []ecs.EntityID
	TotalDamageDealt int
	AffectedSquadIDs []ecs.EntityID
}
