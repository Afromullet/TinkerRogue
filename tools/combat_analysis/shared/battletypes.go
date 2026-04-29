package shared

import "time"

type BattleRecord struct {
	BattleID    string             `json:"battle_id"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     time.Time          `json:"end_time"`
	FinalRound  int                `json:"final_round"`
	VictorName  string             `json:"victor_name,omitempty"`
	Engagements []EngagementRecord `json:"engagements"`
}

type EngagementRecord struct {
	Index     int        `json:"index"`
	Round     int        `json:"round"`
	CombatLog *CombatLog `json:"combat_log"`
}

// CombatLog uses the full version with AttackEvents.
// Packages that don't need AttackEvents simply ignore the field.
type CombatLog struct {
	AttackerSquadName string         `json:"AttackerSquadName"`
	DefenderSquadName string         `json:"DefenderSquadName"`
	SquadDistance     int            `json:"SquadDistance"`
	AttackingUnits    []UnitSnapshot `json:"AttackingUnits"`
	DefendingUnits    []UnitSnapshot `json:"DefendingUnits"`
	AttackEvents      []AttackEvent  `json:"AttackEvents"`
	HealEvents        []HealEvent    `json:"HealEvents"`
	TotalHealing      int            `json:"TotalHealing"`
}

type UnitSnapshot struct {
	UnitID   int64  `json:"UnitID"`
	UnitName string `json:"UnitName"`
	GridRow  int    `json:"GridRow"`
	GridCol  int    `json:"GridCol"`
	RoleName string `json:"RoleName"`
}

type HealEvent struct {
	HealerID       int64 `json:"HealerID"`
	TargetID       int64 `json:"TargetID"`
	HealAmount     int   `json:"HealAmount"`
	TargetHPBefore int   `json:"TargetHPBefore"`
	TargetHPAfter  int   `json:"TargetHPAfter"`
	AttackIndex    int   `json:"AttackIndex"`
}

type AttackEvent struct {
	AttackerID      int64      `json:"AttackerID"`
	DefenderID      int64      `json:"DefenderID"`
	AttackIndex     int        `json:"AttackIndex"`
	TargetInfo      TargetInfo `json:"TargetInfo"`
	IsCounterattack bool       `json:"IsCounterattack"`
	HitResult       HitResult  `json:"HitResult"`
	BaseDamage      int        `json:"BaseDamage"`
	CritMultiplier  float64    `json:"CritMultiplier"`
	FinalDamage     int        `json:"FinalDamage"`
	WasKilled       bool       `json:"WasKilled"`
}

type TargetInfo struct {
	TargetMode string `json:"TargetMode"`
	TargetRow  int    `json:"TargetRow"`
	TargetCol  int    `json:"TargetCol"`
}

type HitResult struct {
	Type           int `json:"Type"`
	HitRoll        int `json:"HitRoll"`
	HitThreshold   int `json:"HitThreshold"`
	DodgeRoll      int `json:"DodgeRoll"`
	DodgeThreshold int `json:"DodgeThreshold"`
	CritRoll       int `json:"CritRoll"`
	CritThreshold  int `json:"CritThreshold"`
}
