package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

// LoadCSV reads the combat_balance_report CSV and returns parsed rows.
func LoadCSV(path string) ([]InputRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV has no data rows")
	}

	// Skip header row
	var rows []InputRow
	for i, rec := range records[1:] {
		if len(rec) != 16 {
			return nil, fmt.Errorf("row %d: expected 16 columns, got %d", i+2, len(rec))
		}

		row, err := parseRow(rec, i+2)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func parseRow(rec []string, lineNum int) (InputRow, error) {
	var row InputRow
	var err error

	row.Attacker = rec[0]
	row.Defender = rec[1]
	row.AttackType = rec[2]

	row.TotalAttacks, err = strconv.Atoi(rec[3])
	if err != nil {
		return row, fmt.Errorf("row %d: bad TotalAttacks %q: %w", lineNum, rec[3], err)
	}
	row.Hits, err = strconv.Atoi(rec[4])
	if err != nil {
		return row, fmt.Errorf("row %d: bad Hits %q: %w", lineNum, rec[4], err)
	}
	row.Misses, err = strconv.Atoi(rec[5])
	if err != nil {
		return row, fmt.Errorf("row %d: bad Misses %q: %w", lineNum, rec[5], err)
	}
	row.Dodges, err = strconv.Atoi(rec[6])
	if err != nil {
		return row, fmt.Errorf("row %d: bad Dodges %q: %w", lineNum, rec[6], err)
	}
	row.Criticals, err = strconv.Atoi(rec[7])
	if err != nil {
		return row, fmt.Errorf("row %d: bad Criticals %q: %w", lineNum, rec[7], err)
	}
	row.HitRate, err = strconv.ParseFloat(rec[8], 64)
	if err != nil {
		return row, fmt.Errorf("row %d: bad HitRate %q: %w", lineNum, rec[8], err)
	}
	row.DodgeRate, err = strconv.ParseFloat(rec[9], 64)
	if err != nil {
		return row, fmt.Errorf("row %d: bad DodgeRate %q: %w", lineNum, rec[9], err)
	}
	row.CritRate, err = strconv.ParseFloat(rec[10], 64)
	if err != nil {
		return row, fmt.Errorf("row %d: bad CritRate %q: %w", lineNum, rec[10], err)
	}
	row.TotalDamage, err = strconv.Atoi(rec[11])
	if err != nil {
		return row, fmt.Errorf("row %d: bad TotalDamage %q: %w", lineNum, rec[11], err)
	}
	row.AvgDmgPerAttack, err = strconv.ParseFloat(rec[12], 64)
	if err != nil {
		return row, fmt.Errorf("row %d: bad AvgDmgPerAttack %q: %w", lineNum, rec[12], err)
	}
	row.AvgDmgPerHit, err = strconv.ParseFloat(rec[13], 64)
	if err != nil {
		return row, fmt.Errorf("row %d: bad AvgDmgPerHit %q: %w", lineNum, rec[13], err)
	}
	row.TotalKills, err = strconv.Atoi(rec[14])
	if err != nil {
		return row, fmt.Errorf("row %d: bad TotalKills %q: %w", lineNum, rec[14], err)
	}
	row.BattlesSampled, err = strconv.Atoi(rec[15])
	if err != nil {
		return row, fmt.Errorf("row %d: bad BattlesSampled %q: %w", lineNum, rec[15], err)
	}

	return row, nil
}
