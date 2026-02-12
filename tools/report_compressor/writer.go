package main

import (
	"encoding/csv"
	"fmt"
	"os"
)

// WriteCompressedReport writes the 3-section compressed report to a CSV file.
func WriteCompressedReport(path string, units []UnitStats, matchups []CompressedMatchup, alerts []Alert) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := writeUnitOverview(w, units); err != nil {
		return err
	}
	if err := writeBlankRow(w); err != nil {
		return err
	}
	if err := writeCompressedMatchups(w, matchups); err != nil {
		return err
	}
	if err := writeBlankRow(w); err != nil {
		return err
	}
	if err := writeAlerts(w, alerts); err != nil {
		return err
	}

	return nil
}

func writeUnitOverview(w *csv.Writer, units []UnitStats) error {
	// Section header
	if err := w.Write([]string{"# Unit Overview"}); err != nil {
		return fmt.Errorf("failed to write section header: %w", err)
	}

	// Column headers
	header := []string{
		"Unit", "AttacksMade", "AttacksReceived",
		"OffHitRate", "DefDodgeRate", "OffCritRate",
		"AvgDmgDealt", "AvgDmgTaken",
		"Kills", "Deaths", "KDRatio",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write unit header: %w", err)
	}

	for _, u := range units {
		offHitRate := safeDiv(float64(u.OffHits), float64(u.AttacksMade))
		defDodgeRate := safeDiv(float64(u.DefDodges), float64(u.AttacksReceived))
		offCritRate := safeDiv(float64(u.OffCrits), float64(u.AttacksMade))
		avgDmgDealt := safeDiv(float64(u.DmgDealt), float64(u.AttacksMade))
		avgDmgTaken := safeDiv(float64(u.DmgTaken), float64(u.AttacksReceived))

		kdRatio := 0.0
		if u.Deaths > 0 {
			kdRatio = float64(u.Kills) / float64(u.Deaths)
		} else if u.Kills > 0 {
			kdRatio = float64(u.Kills) // Show kills as ratio when 0 deaths
		}

		row := []string{
			u.Unit,
			fmt.Sprintf("%d", u.AttacksMade),
			fmt.Sprintf("%d", u.AttacksReceived),
			fmt.Sprintf("%.3f", offHitRate),
			fmt.Sprintf("%.3f", defDodgeRate),
			fmt.Sprintf("%.3f", offCritRate),
			fmt.Sprintf("%.2f", avgDmgDealt),
			fmt.Sprintf("%.2f", avgDmgTaken),
			fmt.Sprintf("%d", u.Kills),
			fmt.Sprintf("%d", u.Deaths),
			fmt.Sprintf("%.2f", kdRatio),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write unit row: %w", err)
		}
	}
	return nil
}

func writeCompressedMatchups(w *csv.Writer, matchups []CompressedMatchup) error {
	if err := w.Write([]string{"# Compressed Matchups"}); err != nil {
		return fmt.Errorf("failed to write section header: %w", err)
	}

	header := []string{
		"Attacker", "Defender", "TotalAttacks",
		"HitRate", "CritRate", "DodgeRate",
		"TotalDamage", "AvgDmgPerAttack",
		"Kills", "BattlesSampled",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write matchup header: %w", err)
	}

	for _, m := range matchups {
		hitRate := safeDiv(float64(m.Hits), float64(m.TotalAttacks))
		critRate := safeDiv(float64(m.Crits), float64(m.TotalAttacks))
		dodgeRate := safeDiv(float64(m.Dodges), float64(m.TotalAttacks))
		avgDmg := safeDiv(float64(m.TotalDamage), float64(m.TotalAttacks))

		row := []string{
			m.Attacker,
			m.Defender,
			fmt.Sprintf("%d", m.TotalAttacks),
			fmt.Sprintf("%.3f", hitRate),
			fmt.Sprintf("%.3f", critRate),
			fmt.Sprintf("%.3f", dodgeRate),
			fmt.Sprintf("%d", m.TotalDamage),
			fmt.Sprintf("%.2f", avgDmg),
			fmt.Sprintf("%d", m.Kills),
			fmt.Sprintf("%d", m.BattlesSampled),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write matchup row: %w", err)
		}
	}
	return nil
}

func writeAlerts(w *csv.Writer, alerts []Alert) error {
	if err := w.Write([]string{"# Balance Alerts"}); err != nil {
		return fmt.Errorf("failed to write section header: %w", err)
	}

	header := []string{"AlertType", "Subject", "Value", "Threshold", "Details"}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write alert header: %w", err)
	}

	for _, a := range alerts {
		row := []string{a.AlertType, a.Subject, a.Value, a.Threshold, a.Details}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write alert row: %w", err)
		}
	}
	return nil
}

func writeBlankRow(w *csv.Writer) error {
	if err := w.Write([]string{""}); err != nil {
		return fmt.Errorf("failed to write blank row: %w", err)
	}
	return nil
}

func safeDiv(num, denom float64) float64 {
	if denom == 0 {
		return 0.0
	}
	return num / denom
}
