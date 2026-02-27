package main

import (
	"encoding/csv"
	"fmt"
	"os"
)

// WriteCSV writes the aggregated matchup data to a CSV file.
func WriteCSV(path string, result *AggregateResult) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	header := []string{
		"Attacker", "Defender", "AttackType",
		"TotalAttacks", "Hits", "Misses", "Dodges", "Criticals",
		"HitRate", "DodgeRate", "CritRate",
		"TotalDamage", "AvgDmgPerAttack", "AvgDmgPerHit",
		"TotalKills", "BattlesSampled",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	keys := SortedKeys(result.Matchups)

	for _, key := range keys {
		stats := result.Matchups[key]

		attackType := "Regular"
		if key.IsCounterattack {
			attackType = "Counterattack"
		}

		totalAttacks := stats.TotalAttacks
		successfulHits := stats.Hits + stats.Criticals

		hitRate := safeRate(successfulHits, totalAttacks)
		dodgeRate := safeRate(stats.Dodges, totalAttacks)
		critRate := safeRate(stats.Criticals, totalAttacks)
		avgDmgPerAttack := safeAvg(stats.TotalDamage, totalAttacks)
		avgDmgPerHit := safeAvg(stats.TotalDamage, successfulHits)

		row := []string{
			key.AttackerName,
			key.DefenderName,
			attackType,
			fmt.Sprintf("%d", totalAttacks),
			fmt.Sprintf("%d", stats.Hits),
			fmt.Sprintf("%d", stats.Misses),
			fmt.Sprintf("%d", stats.Dodges),
			fmt.Sprintf("%d", stats.Criticals),
			fmt.Sprintf("%.3f", hitRate),
			fmt.Sprintf("%.3f", dodgeRate),
			fmt.Sprintf("%.3f", critRate),
			fmt.Sprintf("%d", stats.TotalDamage),
			fmt.Sprintf("%.2f", avgDmgPerAttack),
			fmt.Sprintf("%.2f", avgDmgPerHit),
			fmt.Sprintf("%d", stats.TotalKills),
			fmt.Sprintf("%d", len(stats.BattlesSeen)),
		}

		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	// Write heal section if any heal data exists
	if len(result.HealMatchups) > 0 {
		// Blank separator row
		if err := w.Write([]string{""}); err != nil {
			return fmt.Errorf("failed to write separator: %w", err)
		}

		// Heal section header
		healHeader := []string{
			"# Heal",
			"Healer", "Target", "TotalHeals", "TotalHealing",
			"AvgHealPerAction", "BattlesSampled",
		}
		if err := w.Write(healHeader); err != nil {
			return fmt.Errorf("failed to write heal header: %w", err)
		}

		healKeys := SortedHealKeys(result.HealMatchups)
		for _, key := range healKeys {
			hstats := result.HealMatchups[key]
			avgHeal := safeAvg(hstats.TotalAmount, hstats.TotalHeals)

			row := []string{
				"",
				key.HealerName,
				key.TargetName,
				fmt.Sprintf("%d", hstats.TotalHeals),
				fmt.Sprintf("%d", hstats.TotalAmount),
				fmt.Sprintf("%.2f", avgHeal),
				fmt.Sprintf("%d", len(hstats.BattlesSeen)),
			}
			if err := w.Write(row); err != nil {
				return fmt.Errorf("failed to write heal row: %w", err)
			}
		}
	}

	return nil
}

// safeRate computes numerator/denominator, returning 0.0 on division by zero.
func safeRate(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0.0
	}
	return float64(numerator) / float64(denominator)
}

// safeAvg computes numerator/denominator as float64, returning 0.0 on division by zero.
func safeAvg(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0.0
	}
	return float64(numerator) / float64(denominator)
}
