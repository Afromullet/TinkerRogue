package main

// InputRow represents a single parsed row from the combat_balance_report CSV.
type InputRow struct {
	Attacker       string
	Defender       string
	AttackType     string // "Regular" or "Counterattack"
	TotalAttacks   int
	Hits           int
	Misses         int
	Dodges         int
	Criticals      int
	HitRate        float64
	DodgeRate      float64
	CritRate       float64
	TotalDamage    int
	AvgDmgPerAttack float64
	AvgDmgPerHit   float64
	TotalKills     int
	BattlesSampled int
}

// UnitStats holds aggregated offense/defense stats for a single unit.
// Only counts Regular attacks (counterattacks excluded to avoid double-counting).
type UnitStats struct {
	Unit            string
	AttacksMade     int
	AttacksReceived int
	OffHits         int // hits + crits when attacking
	DefDodges       int // dodges when defending
	OffCrits        int // crits when attacking
	DmgDealt        int
	DmgTaken        int
	Kills           int
	Deaths          int
}

// CompressedMatchup merges Regular + Counterattack rows for a single attacker-defender pair.
type CompressedMatchup struct {
	Attacker       string
	Defender       string
	TotalAttacks   int
	Hits           int // hits + crits combined
	Dodges         int
	Crits          int
	TotalDamage    int
	Kills          int
	BattlesSampled int
}

// Alert represents a balance outlier flagged by the detection rules.
type Alert struct {
	AlertType string
	Subject   string
	Value     string
	Threshold string
	Details   string
}
