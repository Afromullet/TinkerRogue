package combatsim

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// =============================================================================
// SQUAD STATE QUERIES
// =============================================================================

// getSquadUnits returns all living unit entities for a squad
func getSquadUnits(squadID ecs.EntityID, manager *common.EntityManager) []*ecs.Entity {
	var units []*ecs.Entity

	for _, result := range manager.World.Query(squads.SquadMemberTag) {
		unitEntity := result.Entity
		memberData := common.GetComponentType[*squads.SquadMemberData](unitEntity, squads.SquadMemberComponent)

		if memberData != nil && memberData.SquadID == squadID {
			// Check if unit is alive
			attrs := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
			if attrs != nil && attrs.CurrentHealth > 0 {
				units = append(units, unitEntity)
			}
		}
	}

	return units
}

// getSquadHP returns (currentHP, maxHP, livingUnits) for a squad
func getSquadHP(squadID ecs.EntityID, manager *common.EntityManager) (int, int, int) {
	currentHP := 0
	maxHP := 0
	livingUnits := 0

	for _, result := range manager.World.Query(squads.SquadMemberTag) {
		unitEntity := result.Entity
		memberData := common.GetComponentType[*squads.SquadMemberData](unitEntity, squads.SquadMemberComponent)

		if memberData != nil && memberData.SquadID == squadID {
			attrs := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
			if attrs != nil {
				maxHP += attrs.MaxHealth
				if attrs.CurrentHealth > 0 {
					currentHP += attrs.CurrentHealth
					livingUnits++
				}
			}
		}
	}

	return currentHP, maxHP, livingUnits
}

// isSquadDefeated returns true if all units in squad have 0 HP
func isSquadDefeated(squadID ecs.EntityID, manager *common.EntityManager) bool {
	for _, result := range manager.World.Query(squads.SquadMemberTag) {
		unitEntity := result.Entity
		memberData := common.GetComponentType[*squads.SquadMemberData](unitEntity, squads.SquadMemberComponent)

		if memberData != nil && memberData.SquadID == squadID {
			attrs := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
			if attrs != nil && attrs.CurrentHealth > 0 {
				return false // At least one unit alive
			}
		}
	}

	return true
}

// =============================================================================
// COMBAT COMPLETION
// =============================================================================

// IsCombatComplete checks if combat should end
// Returns (isComplete, winner) where winner is "Attacker", "Defender", or "Draw"
func IsCombatComplete(attackerSquadID, defenderSquadID ecs.EntityID, manager *common.EntityManager) (bool, string) {
	attackerDefeated := isSquadDefeated(attackerSquadID, manager)
	defenderDefeated := isSquadDefeated(defenderSquadID, manager)

	if attackerDefeated && defenderDefeated {
		return true, "Draw"
	}
	if defenderDefeated {
		return true, "Attacker"
	}
	if attackerDefeated {
		return true, "Defender"
	}

	return false, ""
}

// =============================================================================
// MOMENTUM CALCULATION
// =============================================================================

// CalculateMomentum computes current combat momentum
// Positive = attacker advantage, Negative = defender advantage
// Range: -1.0 to +1.0
func CalculateMomentum(attackerSquadID, defenderSquadID ecs.EntityID, manager *common.EntityManager) float64 {
	attackerHP, attackerMaxHP, attackerUnits := getSquadHP(attackerSquadID, manager)
	defenderHP, defenderMaxHP, defenderUnits := getSquadHP(defenderSquadID, manager)

	// Avoid division by zero
	if attackerMaxHP == 0 || defenderMaxHP == 0 {
		return 0
	}

	// HP percentage difference
	attackerHPPercent := float64(attackerHP) / float64(attackerMaxHP)
	defenderHPPercent := float64(defenderHP) / float64(defenderMaxHP)

	// Weight by unit count (more units = more advantage)
	totalUnits := attackerUnits + defenderUnits
	if totalUnits == 0 {
		return 0
	}

	unitWeight := float64(attackerUnits-defenderUnits) / float64(totalUnits) * 0.3 // 30% weight for units

	// Combined momentum: HP difference (70%) + Unit count (30%)
	momentum := (attackerHPPercent - defenderHPPercent) * 0.7 + unitWeight

	// Clamp to [-1, 1]
	if momentum > 1.0 {
		momentum = 1.0
	}
	if momentum < -1.0 {
		momentum = -1.0
	}

	return momentum
}

// =============================================================================
// ROUND SNAPSHOT
// =============================================================================

// CaptureRoundSnapshot creates a snapshot of combat state at current round
func CaptureRoundSnapshot(round int, attackerSquadID, defenderSquadID ecs.EntityID, manager *common.EntityManager) RoundSnapshot {
	attackerHP, attackerMaxHP, attackerUnits := getSquadHP(attackerSquadID, manager)
	defenderHP, defenderMaxHP, defenderUnits := getSquadHP(defenderSquadID, manager)

	return RoundSnapshot{
		RoundNumber:        round,
		AttackerUnitsAlive: attackerUnits,
		AttackerTotalHP:    attackerHP,
		AttackerMaxHP:      attackerMaxHP,
		DefenderUnitsAlive: defenderUnits,
		DefenderTotalHP:    defenderHP,
		DefenderMaxHP:      defenderMaxHP,
		Momentum:           CalculateMomentum(attackerSquadID, defenderSquadID, manager),
	}
}

// UpdateSnapshotFromCombat updates a snapshot with combat results
func UpdateSnapshotFromCombat(snapshot *RoundSnapshot, result *squads.CombatResult) {
	snapshot.DamageDealtThisRound = result.TotalDamage
	snapshot.UnitsKilledThisRound = len(result.UnitsKilled)

	// Count crits and dodges from combat log
	if result.CombatLog != nil {
		for _, event := range result.CombatLog.AttackEvents {
			switch event.HitResult.Type {
			case squads.HitTypeCritical:
				snapshot.CritsThisRound++
			case squads.HitTypeDodge:
				snapshot.DodgesThisRound++
			}
		}
	}
}

// =============================================================================
// TIMELINE ANALYSIS
// =============================================================================

// DetectTurningPoint finds the round where momentum permanently shifted
// Returns 0 if no clear turning point
func DetectTurningPoint(timeline TimelineData) int {
	if len(timeline.Rounds) < 2 {
		return 0
	}

	// Find the round where momentum sign changed and stayed changed
	var lastSign int
	for i, round := range timeline.Rounds {
		currentSign := 0
		if round.Momentum > 0.1 {
			currentSign = 1
		} else if round.Momentum < -0.1 {
			currentSign = -1
		}

		if i > 0 && currentSign != 0 && lastSign != 0 && currentSign != lastSign {
			// Check if this sign persists for remaining rounds
			persists := true
			for j := i + 1; j < len(timeline.Rounds); j++ {
				futureSign := 0
				if timeline.Rounds[j].Momentum > 0.1 {
					futureSign = 1
				} else if timeline.Rounds[j].Momentum < -0.1 {
					futureSign = -1
				}
				if futureSign != 0 && futureSign != currentSign {
					persists = false
					break
				}
			}
			if persists {
				return round.RoundNumber
			}
		}

		if currentSign != 0 {
			lastSign = currentSign
		}
	}

	return 0
}

// CalculateSnowballFactor measures how much early leads compound
// Returns a value where 1.0 = no snowball, >1.0 = momentum compounds, <1.0 = comebacks possible
func CalculateSnowballFactor(timeline TimelineData) float64 {
	if len(timeline.Rounds) < 3 {
		return 1.0
	}

	// Compare momentum change rate in early vs late combat
	midpoint := len(timeline.Rounds) / 2

	// Early momentum changes
	earlyChanges := 0.0
	for i := 1; i <= midpoint && i < len(timeline.Rounds); i++ {
		earlyChanges += absFloat(timeline.Rounds[i].Momentum - timeline.Rounds[i-1].Momentum)
	}
	if midpoint > 0 {
		earlyChanges /= float64(midpoint)
	}

	// Late momentum changes
	lateChanges := 0.0
	lateCount := 0
	for i := midpoint + 1; i < len(timeline.Rounds); i++ {
		lateChanges += absFloat(timeline.Rounds[i].Momentum - timeline.Rounds[i-1].Momentum)
		lateCount++
	}
	if lateCount > 0 {
		lateChanges /= float64(lateCount)
	}

	// Snowball factor: if momentum stabilizes late (less changes), it indicates snowball
	if lateChanges > 0 {
		return earlyChanges / lateChanges
	}

	return 1.0
}

// =============================================================================
// TIMELINE AGGREGATION
// =============================================================================

// AggregateTimelines combines timelines across simulations
func AggregateTimelines(timelines []TimelineData) *TimelineAggregated {
	if len(timelines) == 0 {
		return &TimelineAggregated{}
	}

	agg := &TimelineAggregated{
		RoundStats:         make([]RoundStatistics, 0),
		FirstBloodValues:   make([]int, 0),
		TurningPointValues: make([]int, 0),
		DurationValues:     make([]int, 0),
	}

	// Find max rounds across all timelines
	maxRounds := 0
	for _, tl := range timelines {
		if len(tl.Rounds) > maxRounds {
			maxRounds = len(tl.Rounds)
		}
	}

	// Initialize round stats
	for r := 0; r < maxRounds; r++ {
		agg.RoundStats = append(agg.RoundStats, RoundStatistics{
			RoundNumber:      r + 1,
			AttackerHPValues: make([]int, 0),
			DefenderHPValues: make([]int, 0),
			DamageValues:     make([]int, 0),
		})
	}

	// Aggregate each timeline
	for _, tl := range timelines {
		// First blood
		if tl.FirstBloodRound > 0 {
			agg.FirstBloodValues = append(agg.FirstBloodValues, tl.FirstBloodRound)
		}

		// Turning point
		if tl.TurningPoint > 0 {
			agg.TurningPointValues = append(agg.TurningPointValues, tl.TurningPoint)
		}

		// Duration
		agg.DurationValues = append(agg.DurationValues, tl.CombatDuration)

		// Per-round stats
		for i, round := range tl.Rounds {
			if i < len(agg.RoundStats) {
				agg.RoundStats[i].SampleCount++
				agg.RoundStats[i].AttackerHPValues = append(agg.RoundStats[i].AttackerHPValues, round.AttackerTotalHP)
				agg.RoundStats[i].DefenderHPValues = append(agg.RoundStats[i].DefenderHPValues, round.DefenderTotalHP)
				agg.RoundStats[i].DamageValues = append(agg.RoundStats[i].DamageValues, round.DamageDealtThisRound)
			}
		}
	}

	// Calculate averages
	agg.AvgFirstBlood = CalculateMeanInt(agg.FirstBloodValues)
	agg.AvgTurningPoint = CalculateMeanInt(agg.TurningPointValues)
	agg.AvgDuration = CalculateMeanInt(agg.DurationValues)

	// Calculate per-round averages
	for i := range agg.RoundStats {
		if agg.RoundStats[i].SampleCount > 0 {
			agg.RoundStats[i].AvgAttackerHP = CalculateMeanInt(agg.RoundStats[i].AttackerHPValues)
			agg.RoundStats[i].AvgDefenderHP = CalculateMeanInt(agg.RoundStats[i].DefenderHPValues)
			agg.RoundStats[i].AvgDamageDealt = CalculateMeanInt(agg.RoundStats[i].DamageValues)
		}
	}

	return agg
}

// GenerateDamageCurve creates round-by-round damage progression
func GenerateDamageCurve(agg *TimelineAggregated) []float64 {
	curve := make([]float64, len(agg.RoundStats))
	for i, stats := range agg.RoundStats {
		curve[i] = stats.AvgDamageDealt
	}
	return curve
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// NewTimelineData creates an empty timeline
func NewTimelineData() *TimelineData {
	return &TimelineData{
		Rounds: make([]RoundSnapshot, 0),
	}
}
