package main

import (
	"fmt"
	"math"
	"sort"
)

// matchupKey is used for grouping rows by attacker-defender pair.
type matchupKey struct {
	Attacker string
	Defender string
}

// BuildUnitOverview aggregates per-unit offense/defense stats from Regular attacks only.
func BuildUnitOverview(rows []InputRow) []UnitStats {
	units := make(map[string]*UnitStats)

	getUnit := func(name string) *UnitStats {
		u, ok := units[name]
		if !ok {
			u = &UnitStats{Unit: name}
			units[name] = u
		}
		return u
	}

	for _, r := range rows {
		if r.AttackType != "Regular" {
			continue
		}

		// Attacker offense stats
		atk := getUnit(r.Attacker)
		atk.AttacksMade += r.TotalAttacks
		atk.OffHits += r.Hits + r.Criticals
		atk.OffCrits += r.Criticals
		atk.DmgDealt += r.TotalDamage
		atk.Kills += r.TotalKills

		// Defender defense stats
		def := getUnit(r.Defender)
		def.AttacksReceived += r.TotalAttacks
		def.DefDodges += r.Dodges
		def.DmgTaken += r.TotalDamage
		def.Deaths += r.TotalKills
	}

	// Sort by unit name
	names := make([]string, 0, len(units))
	for name := range units {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]UnitStats, 0, len(names))
	for _, name := range names {
		result = append(result, *units[name])
	}
	return result
}

// BuildCompressedMatchups merges Regular + Counterattack rows per attacker-defender pair.
func BuildCompressedMatchups(rows []InputRow) []CompressedMatchup {
	matchups := make(map[matchupKey]*CompressedMatchup)

	for _, r := range rows {
		key := matchupKey{Attacker: r.Attacker, Defender: r.Defender}

		m, ok := matchups[key]
		if !ok {
			m = &CompressedMatchup{
				Attacker: r.Attacker,
				Defender: r.Defender,
			}
			matchups[key] = m
		}

		m.TotalAttacks += r.TotalAttacks
		m.Hits += r.Hits + r.Criticals
		m.Dodges += r.Dodges
		m.Crits += r.Criticals
		m.TotalDamage += r.TotalDamage
		m.Kills += r.TotalKills

		if r.BattlesSampled > m.BattlesSampled {
			m.BattlesSampled = r.BattlesSampled
		}
	}

	// Sort by (Attacker, Defender)
	keys := make([]matchupKey, 0, len(matchups))
	for k := range matchups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Attacker != keys[j].Attacker {
			return keys[i].Attacker < keys[j].Attacker
		}
		return keys[i].Defender < keys[j].Defender
	})

	result := make([]CompressedMatchup, 0, len(keys))
	for _, k := range keys {
		result = append(result, *matchups[k])
	}
	return result
}

// DetectAlerts checks units and matchups for balance outliers.
func DetectAlerts(units []UnitStats, matchups []CompressedMatchup) []Alert {
	var alerts []Alert

	// Compute top 5% damage threshold across matchups with ≥5 attacks
	var dmgValues []float64
	for _, m := range matchups {
		if m.TotalAttacks >= 5 {
			dmgValues = append(dmgValues, float64(m.TotalDamage)/float64(m.TotalAttacks))
		}
	}
	sort.Float64s(dmgValues)

	highDmgThreshold := math.MaxFloat64
	if len(dmgValues) > 0 {
		idx := int(float64(len(dmgValues)) * 0.95)
		if idx >= len(dmgValues) {
			idx = len(dmgValues) - 1
		}
		highDmgThreshold = dmgValues[idx]
	}

	// Check matchups
	for _, m := range matchups {
		if m.TotalAttacks < 5 {
			continue
		}

		hitRate := float64(m.Hits) / float64(m.TotalAttacks)
		critRate := float64(m.Crits) / float64(m.TotalAttacks)
		avgDmg := float64(m.TotalDamage) / float64(m.TotalAttacks)
		subject := fmt.Sprintf("%s vs %s", m.Attacker, m.Defender)

		if hitRate >= 1.0 {
			alerts = append(alerts, Alert{
				AlertType: "PerfectHitRate",
				Subject:   subject,
				Value:     fmt.Sprintf("%.1f%%", hitRate*100),
				Threshold: "100%",
				Details:   fmt.Sprintf("%s never misses %s (%d attacks)", m.Attacker, m.Defender, m.TotalAttacks),
			})
		}

		if hitRate <= 0.0 {
			alerts = append(alerts, Alert{
				AlertType: "ZeroHitRate",
				Subject:   subject,
				Value:     fmt.Sprintf("%.1f%%", hitRate*100),
				Threshold: "0%",
				Details:   fmt.Sprintf("%s never hits %s (%d attacks)", m.Attacker, m.Defender, m.TotalAttacks),
			})
		}

		if critRate > 0.5 {
			alerts = append(alerts, Alert{
				AlertType: "HighCrit",
				Subject:   subject,
				Value:     fmt.Sprintf("%.1f%%", critRate*100),
				Threshold: ">50%",
				Details:   fmt.Sprintf("%s crits %s %.0f%% of the time (%d attacks)", m.Attacker, m.Defender, critRate*100, m.TotalAttacks),
			})
		}

		if avgDmg >= highDmgThreshold {
			alerts = append(alerts, Alert{
				AlertType: "HighDamage",
				Subject:   subject,
				Value:     fmt.Sprintf("%.2f", avgDmg),
				Threshold: fmt.Sprintf(">=%.2f (top 5%%)", highDmgThreshold),
				Details:   fmt.Sprintf("%s deals %.2f avg damage to %s per attack", m.Attacker, avgDmg, m.Defender),
			})
		}
	}

	// Check unit K/D ratios
	for _, u := range units {
		kd := float64(u.Kills)
		if u.Deaths > 0 {
			kd = float64(u.Kills) / float64(u.Deaths)
		}

		if u.Kills >= 5 && u.Deaths == 0 {
			alerts = append(alerts, Alert{
				AlertType: "SkewedKD",
				Subject:   u.Unit,
				Value:     fmt.Sprintf("%d/%d", u.Kills, u.Deaths),
				Threshold: ">=5 kills, 0 deaths",
				Details:   fmt.Sprintf("%s scored %d kills with 0 deaths", u.Unit, u.Kills),
			})
		} else if kd > 5.0 && u.Deaths > 0 {
			alerts = append(alerts, Alert{
				AlertType: "SkewedKD",
				Subject:   u.Unit,
				Value:     fmt.Sprintf("%.2f", kd),
				Threshold: ">5.0",
				Details:   fmt.Sprintf("%s has a K/D of %.2f (%d kills, %d deaths)", u.Unit, kd, u.Kills, u.Deaths),
			})
		}
	}

	// Sort alerts by type then subject
	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].AlertType != alerts[j].AlertType {
			return alerts[i].AlertType < alerts[j].AlertType
		}
		return alerts[i].Subject < alerts[j].Subject
	})

	return alerts
}
