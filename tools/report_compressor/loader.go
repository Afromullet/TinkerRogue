package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

// LoadCSV reads the combat_balance_report CSV and returns parsed damage rows and heal rows.
func LoadCSV(path string) ([]InputRow, []HealRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1 // Allow variable-length records (heal section has different column count)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, nil, fmt.Errorf("CSV has no data rows")
	}

	// Find where the heal section starts (blank row separator followed by "# Heal" header)
	healStart := -1
	for i, rec := range records {
		if len(rec) > 0 && rec[0] == "# Heal" {
			healStart = i
			break
		}
	}

	// Parse damage rows (skip header, stop at heal section or end)
	damageEnd := len(records)
	if healStart > 0 {
		// The blank separator row is one before the heal header
		damageEnd = healStart - 1
	}

	var rows []InputRow
	for i := 1; i < damageEnd; i++ {
		rec := records[i]
		// Skip blank separator rows
		if len(rec) == 0 || (len(rec) == 1 && rec[0] == "") {
			continue
		}
		if len(rec) != 16 {
			return nil, nil, fmt.Errorf("row %d: expected 16 columns, got %d", i+1, len(rec))
		}

		row, err := parseRow(rec, i+1)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, row)
	}

	// Parse heal rows if heal section exists
	var healRows []HealRow
	if healStart >= 0 {
		// Heal data rows start after the header row (healStart is "# Heal" header)
		for i := healStart + 1; i < len(records); i++ {
			rec := records[i]
			// Skip blank rows
			if len(rec) == 0 || (len(rec) == 1 && rec[0] == "") {
				continue
			}
			// Heal rows have an empty first column, then 6 data columns (total 7)
			if len(rec) < 7 {
				continue
			}

			healRow, err := parseHealRow(rec, i+1)
			if err != nil {
				return nil, nil, err
			}
			healRows = append(healRows, healRow)
		}
	}

	return rows, healRows, nil
}

func parseHealRow(rec []string, lineNum int) (HealRow, error) {
	var row HealRow
	var err error

	// First column is empty (section marker), data starts at index 1
	row.Healer = rec[1]
	row.Target = rec[2]

	row.TotalHeals, err = strconv.Atoi(rec[3])
	if err != nil {
		return row, fmt.Errorf("row %d: bad TotalHeals %q: %w", lineNum, rec[3], err)
	}
	row.TotalHealing, err = strconv.Atoi(rec[4])
	if err != nil {
		return row, fmt.Errorf("row %d: bad TotalHealing %q: %w", lineNum, rec[4], err)
	}
	row.AvgHealPerAction, err = strconv.ParseFloat(rec[5], 64)
	if err != nil {
		return row, fmt.Errorf("row %d: bad AvgHealPerAction %q: %w", lineNum, rec[5], err)
	}
	row.BattlesSampled, err = strconv.Atoi(rec[6])
	if err != nil {
		return row, fmt.Errorf("row %d: bad BattlesSampled %q: %w", lineNum, rec[6], err)
	}

	return row, nil
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
