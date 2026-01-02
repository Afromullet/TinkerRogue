package combatsim

import (
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// SimulationResult contains aggregated statistics from N combat simulations
type SimulationResult struct {
	Scenario   CombatScenario
	Iterations int

	// Win rates
	AttackerWins int
	DefenderWins int
	Draws        int

	// Combat duration
	AvgTurnsUntilEnd float64
	MinTurns         int
	MaxTurns         int
	TurnHistogram    map[int]int // Turn count -> frequency

	// Damage statistics (by squad name)
	TotalDamageDealt map[string]int
	TotalDamageTaken map[string]int

	// Casualty analysis
	TotalUnitsKilled map[string]int

	// Combat mechanics totals
	TotalHits     int
	TotalMisses   int
	TotalDodges   int
	TotalCrits    int
	CoverApplied  int
	TotalCoverReduction float64

	// Raw combat logs (if verbose mode)
	CombatLogs []*squads.CombatLog
}

// CombatMetrics contains detailed metrics from a single combat simulation
type CombatMetrics struct {
	Winner       string // "Attacker", "Defender", or "Draw"
	TurnsElapsed int

	// Damage by squad
	DamageDealt map[string]int
	DamageTaken map[string]int

	// Units killed by squad
	UnitsKilled map[string]int

	// Combat mechanics
	HitCount          int
	MissCount         int
	DodgeCount        int
	CritCount         int
	CoverApplications int
	TotalCoverReduction float64

	// Combat log
	Log *squads.CombatLog
}

// NewSimulationResult creates a new result aggregator
func NewSimulationResult(scenario CombatScenario, iterations int) *SimulationResult {
	return &SimulationResult{
		Scenario:         scenario,
		Iterations:       iterations,
		TurnHistogram:    make(map[int]int),
		TotalDamageDealt: make(map[string]int),
		TotalDamageTaken: make(map[string]int),
		TotalUnitsKilled: make(map[string]int),
		MinTurns:         9999,
		CombatLogs:       []*squads.CombatLog{},
	}
}

// AddMetrics aggregates metrics from a single combat run
func (r *SimulationResult) AddMetrics(metrics *CombatMetrics) {
	// Win tracking
	switch metrics.Winner {
	case "Attacker":
		r.AttackerWins++
	case "Defender":
		r.DefenderWins++
	case "Draw":
		r.Draws++
	}

	// Turn tracking
	if metrics.TurnsElapsed < r.MinTurns {
		r.MinTurns = metrics.TurnsElapsed
	}
	if metrics.TurnsElapsed > r.MaxTurns {
		r.MaxTurns = metrics.TurnsElapsed
	}
	r.TurnHistogram[metrics.TurnsElapsed]++

	// Damage tracking
	for squad, damage := range metrics.DamageDealt {
		r.TotalDamageDealt[squad] += damage
	}
	for squad, damage := range metrics.DamageTaken {
		r.TotalDamageTaken[squad] += damage
	}

	// Casualty tracking
	for squad, kills := range metrics.UnitsKilled {
		r.TotalUnitsKilled[squad] += kills
	}

	// Mechanics tracking
	r.TotalHits += metrics.HitCount
	r.TotalMisses += metrics.MissCount
	r.TotalDodges += metrics.DodgeCount
	r.TotalCrits += metrics.CritCount
	r.CoverApplied += metrics.CoverApplications
	r.TotalCoverReduction += metrics.TotalCoverReduction
}

// Finalize computes averages and derived statistics
func (r *SimulationResult) Finalize() {
	if r.Iterations == 0 {
		return
	}

	// Calculate average turns
	totalTurns := 0
	for turns, frequency := range r.TurnHistogram {
		totalTurns += turns * frequency
	}
	r.AvgTurnsUntilEnd = float64(totalTurns) / float64(r.Iterations)
}

// NewCombatMetrics creates metrics for a single combat
func NewCombatMetrics() *CombatMetrics {
	return &CombatMetrics{
		DamageDealt: make(map[string]int),
		DamageTaken: make(map[string]int),
		UnitsKilled: make(map[string]int),
	}
}

// ExtractMetrics parses a CombatResult to extract detailed metrics
func ExtractMetrics(result *squads.CombatResult, attackerName, defenderName string, attackerSquadID, defenderSquadID ecs.EntityID) *CombatMetrics {
	metrics := NewCombatMetrics()

	// Determine winner by checking if any units were killed
	if len(result.UnitsKilled) > 0 {
		metrics.Winner = "Attacker"
	} else if result.TotalDamage == 0 {
		metrics.Winner = "Defender"
	} else {
		metrics.Winner = "Draw"
	}

	// Track damage
	metrics.DamageDealt[attackerName] = result.TotalDamage
	metrics.DamageTaken[defenderName] = result.TotalDamage

	// Track kills
	metrics.UnitsKilled[defenderName] = len(result.UnitsKilled)

	// Parse combat log for mechanics
	if result.CombatLog != nil {
		for _, event := range result.CombatLog.AttackEvents {
			// Count hit types using HitResult.Type
			switch event.HitResult.Type {
			case squads.HitTypeMiss:
				metrics.MissCount++
			case squads.HitTypeDodge:
				metrics.DodgeCount++
			case squads.HitTypeCritical:
				metrics.CritCount++
			case squads.HitTypeNormal:
				metrics.HitCount++
			}

			// Track cover using CoverBreakdown
			if event.CoverReduction.TotalReduction > 0 {
				metrics.CoverApplications++
				metrics.TotalCoverReduction += event.CoverReduction.TotalReduction
			}
		}
		metrics.TurnsElapsed = 1 // Simplified - assuming 1 turn per ExecuteSquadAttack
	}

	metrics.Log = result.CombatLog
	return metrics
}
