package combattypes

import (
	"github.com/bytearena/ecs"
)

// AttackEvent captures a single unit-to-unit attack with full damage breakdown
type AttackEvent struct {
	// Combatants
	AttackerID ecs.EntityID
	DefenderID ecs.EntityID

	// Attack metadata
	AttackIndex     int        // Sequential number in attack sequence
	TargetInfo      TargetInfo // Row/cell targeting info
	IsCounterattack bool       // True if this is a counterattack

	// Combat rolls
	HitResult HitResult // Hit, miss, dodge, crit

	// Damage pipeline
	BaseDamage       int
	CritMultiplier   float64        // 1.0 (normal) or 1.5 (crit)
	ResistanceAmount int            // Subtracted amount
	CoverReduction   CoverBreakdown // Detailed cover info
	FinalDamage      int

	// Result state
	DefenderHPBefore int
	DefenderHPAfter  int
	WasKilled        bool
}

// TargetInfo describes how/where the target was selected
type TargetInfo struct {
	TargetMode string // "MeleeRow", "MeleeColumn", "Ranged", or "Magic"
	TargetRow  int    // Which row (0-2)
	TargetCol  int    // Which col (0-2)
}

// HitResult describes the attack roll outcome
type HitResult struct {
	Type           HitType
	HitRoll        int // 1-100
	HitThreshold   int // Hit if roll <= threshold
	DodgeRoll      int // 1-100
	DodgeThreshold int
	CritRoll       int // 1-100
	CritThreshold  int
}

type HitType int

const (
	HitTypeMiss HitType = iota
	HitTypeDodge
	HitTypeNormal
	HitTypeCritical
	HitTypeCounterattack
	HitTypeHeal
)

func (h HitType) String() string {
	switch h {
	case HitTypeMiss:
		return "MISS"
	case HitTypeDodge:
		return "DODGE"
	case HitTypeNormal:
		return "HIT"
	case HitTypeCritical:
		return "CRITICAL"
	case HitTypeCounterattack:
		return "COUNTERATTACK"
	case HitTypeHeal:
		return "HEAL"
	default:
		return "UNKNOWN"
	}
}

// CoverBreakdown tracks which units provided cover and how much
type CoverBreakdown struct {
	TotalReduction float64 // Final percentage (0.0-1.0)
	Providers      []CoverProvider
}

// CoverProvider identifies a single unit providing cover
type CoverProvider struct {
	UnitID     ecs.EntityID
	UnitName   string  // Cached for display
	CoverValue float64 // This unit's contribution (0.0-1.0)
	GridRow    int     // Provider's position
	GridCol    int
}

// HealEvent captures a single unit-to-unit heal with full breakdown
type HealEvent struct {
	HealerID       ecs.EntityID
	TargetID       ecs.EntityID
	HealAmount     int
	TargetHPBefore int
	TargetHPAfter  int
	AttackIndex    int
}

// CombatLog aggregates all attacks in a squad-vs-squad engagement
type CombatLog struct {
	// Squad-level info
	AttackerSquadID   ecs.EntityID
	DefenderSquadID   ecs.EntityID
	AttackerSquadName string
	DefenderSquadName string
	SquadDistance     int

	// Participating units
	AttackingUnits []UnitSnapshot // Units in range
	DefendingUnits []UnitSnapshot // All defending units

	// All individual attacks
	AttackEvents []AttackEvent

	// Healing events
	HealEvents   []HealEvent
	TotalHealing int

	// Summary stats
	TotalDamage    int
	UnitsKilled    int
	DefenderStatus SquadStatus
}

// UnitSnapshot captures unit state for display
type UnitSnapshot struct {
	UnitID      ecs.EntityID
	UnitName    string
	GridRow     int
	GridCol     int
	AttackRange int
	RoleName    string // "Tank", "DPS", "Support"
}

// SquadStatus summarizes squad health after combat
type SquadStatus struct {
	AliveUnits int
	TotalUnits int
	AverageHP  int // Percentage
}

// UnitIdentity provides display-ready unit information
type UnitIdentity struct {
	ID        ecs.EntityID
	Name      string
	GridRow   int
	GridCol   int
	CurrentHP int
	MaxHP     int
	IsLeader  bool
}

// DamageModifiers holds modifiers for damage calculation
type DamageModifiers struct {
	HitModifier      int     // Subtracted from attacker's hit threshold. Positive = penalty; negative = accuracy bonus.
	DamageMultiplier float64
	IsCounterattack  bool

	// Perk-related modifiers
	CritBonus   int     // Added to crit threshold (Executioner's Instinct)
	CoverBonus  float64 // Added to cover calculation (Brace, Fortify)
	SkipCounter bool    // If true, no counterattack phase
	SkipCrit    bool    // If true, crits become normal hits (Vigilance)
}

// NewAttackModifiers returns the default modifiers for a normal attack:
// no hit penalty, full damage, not a counterattack.
func NewAttackModifiers() DamageModifiers {
	return DamageModifiers{HitModifier: 0, DamageMultiplier: 1.0, IsCounterattack: false}
}

// NewCounterattackModifiers returns modifiers for a counterattack with the given
// hit penalty and damage multiplier (IsCounterattack is always true).
func NewCounterattackModifiers(hitModifier int, damageMultiplier float64) DamageModifiers {
	return DamageModifiers{HitModifier: hitModifier, DamageMultiplier: damageMultiplier, IsCounterattack: true}
}

// CombatResult is the unified result of a combat action. It groups three
// separately-owned concerns so callers can reach into the part they care about
// without scanning a flat 9-field struct:
//
//   - Status: orchestration outcome (Success, ErrorReason, destruction flags)
//     set by CombatActionSystem.
//   - Damage: execution data (damage and healing totals, kill list) populated
//     by the damage pipeline in combatprocessing/combatcalculation.
//   - Log: display data (squad names, per-attack events) populated by battlelog.
type CombatResult struct {
	Status CombatStatus
	Damage *DamageRecord
	Log    *CombatLog
}

// CombatStatus carries the orchestration outcome of an attack action.
type CombatStatus struct {
	Success           bool
	ErrorReason       string
	TargetDestroyed   bool
	AttackerDestroyed bool
}

// DamageRecord accumulates the execution data of an attack: total damage,
// per-unit damage and healing maps, and the kill list. The maps must be
// initialized before use; helpers in combatmath rely on map[id]int semantics.
type DamageRecord struct {
	TotalDamage   int
	UnitsKilled   []ecs.EntityID
	DamageByUnit  map[ecs.EntityID]int
	HealingByUnit map[ecs.EntityID]int
}

// NewDamageRecord returns a DamageRecord with all maps and slices initialized
// to non-nil empty values, ready for the damage pipeline to populate.
func NewDamageRecord() *DamageRecord {
	return &DamageRecord{
		DamageByUnit:  make(map[ecs.EntityID]int),
		HealingByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:   []ecs.EntityID{},
	}
}
