package squads

import (
	"github.com/bytearena/ecs"
)

// AttackEvent captures a single unit-to-unit attack with full damage breakdown
type AttackEvent struct {
	// Combatants
	AttackerID ecs.EntityID
	DefenderID ecs.EntityID

	// Attack metadata
	AttackIndex int        // Sequential number in attack sequence
	TargetInfo  TargetInfo // Row/cell targeting info

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
	default:
		return "UNKNOWN"
	}
}

// CoverBreakdown tracks which units provided cover and how much
type CoverBreakdown struct {
	TotalReduction float64         // Final percentage (0.0-1.0)
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

	// All individual attacks
	AttackEvents []AttackEvent

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
}
