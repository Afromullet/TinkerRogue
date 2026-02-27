package main

import (
	"fmt"
	"sort"
)

// MatchupKey identifies a unique attacker→defender matchup.
type MatchupKey struct {
	AttackerName    string
	DefenderName    string
	IsCounterattack bool
}

// MatchupStats accumulates combat statistics for a matchup.
type MatchupStats struct {
	TotalAttacks int
	Hits         int
	Misses       int
	Dodges       int
	Criticals    int
	TotalDamage  int
	TotalKills   int
	BattlesSeen  map[string]bool
}

// HealKey identifies a unique healer→target matchup.
type HealKey struct {
	HealerName string
	TargetName string
}

// HealStats accumulates healing statistics for a healer→target pair.
type HealStats struct {
	TotalHeals  int
	TotalAmount int
	BattlesSeen map[string]bool
}

// AggregateResult holds the final aggregated data.
type AggregateResult struct {
	Matchups      map[MatchupKey]*MatchupStats
	HealMatchups  map[HealKey]*HealStats
	TotalBattles  int
}

// Aggregate processes all battle records and builds matchup statistics.
func Aggregate(records []*BattleRecord) *AggregateResult {
	result := &AggregateResult{
		Matchups:     make(map[MatchupKey]*MatchupStats),
		HealMatchups: make(map[HealKey]*HealStats),
		TotalBattles: len(records),
	}

	for _, record := range records {
		processRecord(record, result)
	}

	return result
}

// processRecord handles a single battle record.
func processRecord(record *BattleRecord, result *AggregateResult) {
	for _, eng := range record.Engagements {
		if eng.CombatLog == nil {
			continue
		}
		processEngagement(record.BattleID, eng.CombatLog, result)
	}
}

// processEngagement handles a single engagement within a battle.
func processEngagement(battleID string, log *CombatLog, result *AggregateResult) {
	// Build ID→Name lookup from both sides
	nameMap := make(map[int64]string)
	for _, u := range log.AttackingUnits {
		nameMap[u.UnitID] = u.UnitName
	}
	for _, u := range log.DefendingUnits {
		nameMap[u.UnitID] = u.UnitName
	}

	for _, event := range log.AttackEvents {
		attackerName, ok := nameMap[event.AttackerID]
		if !ok {
			fmt.Printf("Warning: unknown attacker ID %d in battle %s, skipping event\n", event.AttackerID, battleID)
			continue
		}
		defenderName, ok := nameMap[event.DefenderID]
		if !ok {
			fmt.Printf("Warning: unknown defender ID %d in battle %s, skipping event\n", event.DefenderID, battleID)
			continue
		}

		key := MatchupKey{
			AttackerName:    attackerName,
			DefenderName:    defenderName,
			IsCounterattack: event.IsCounterattack,
		}

		stats, exists := result.Matchups[key]
		if !exists {
			stats = &MatchupStats{
				BattlesSeen: make(map[string]bool),
			}
			result.Matchups[key] = stats
		}

		stats.TotalAttacks++
		stats.TotalDamage += event.FinalDamage
		stats.BattlesSeen[battleID] = true

		if event.WasKilled {
			stats.TotalKills++
		}

		switch event.HitResult.Type {
		case HitTypeMiss:
			stats.Misses++
		case HitTypeDodge:
			stats.Dodges++
		case HitTypeNormal:
			stats.Hits++
		case HitTypeCritical:
			stats.Criticals++
		case HitTypeCounterattack:
			stats.Hits++ // Counterattack hit counts as a hit
		case HitTypeHeal:
			// Heals go through HealEvents, not AttackEvents — skip
		}
	}

	// Process heal events
	for _, heal := range log.HealEvents {
		healerName, ok := nameMap[heal.HealerID]
		if !ok {
			fmt.Printf("Warning: unknown healer ID %d in battle %s, skipping heal event\n", heal.HealerID, battleID)
			continue
		}
		targetName, ok := nameMap[heal.TargetID]
		if !ok {
			fmt.Printf("Warning: unknown heal target ID %d in battle %s, skipping heal event\n", heal.TargetID, battleID)
			continue
		}

		key := HealKey{
			HealerName: healerName,
			TargetName: targetName,
		}

		hstats, exists := result.HealMatchups[key]
		if !exists {
			hstats = &HealStats{
				BattlesSeen: make(map[string]bool),
			}
			result.HealMatchups[key] = hstats
		}

		hstats.TotalHeals++
		hstats.TotalAmount += heal.HealAmount
		hstats.BattlesSeen[battleID] = true
	}
}

// SortedHealKeys returns heal matchup keys sorted by (Healer, Target).
func SortedHealKeys(matchups map[HealKey]*HealStats) []HealKey {
	keys := make([]HealKey, 0, len(matchups))
	for k := range matchups {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].HealerName != keys[j].HealerName {
			return keys[i].HealerName < keys[j].HealerName
		}
		return keys[i].TargetName < keys[j].TargetName
	})

	return keys
}

// SortedKeys returns matchup keys sorted by (Attacker, Defender, AttackType).
func SortedKeys(matchups map[MatchupKey]*MatchupStats) []MatchupKey {
	keys := make([]MatchupKey, 0, len(matchups))
	for k := range matchups {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].AttackerName != keys[j].AttackerName {
			return keys[i].AttackerName < keys[j].AttackerName
		}
		if keys[i].DefenderName != keys[j].DefenderName {
			return keys[i].DefenderName < keys[j].DefenderName
		}
		// Regular before Counterattack
		return !keys[i].IsCounterattack && keys[j].IsCounterattack
	})

	return keys
}
