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
