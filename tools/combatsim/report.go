package combatsim

import (
	"fmt"
	"strings"
)

// ReportFormatter formats simulation results for output
type ReportFormatter struct {
	showDetails bool
}

// NewReportFormatter creates a new report formatter
func NewReportFormatter(showDetails bool) *ReportFormatter {
	return &ReportFormatter{
		showDetails: showDetails,
	}
}

// FormatSimulationResult produces a human-readable report
func (r *ReportFormatter) FormatSimulationResult(result *SimulationResult) string {
	var sb strings.Builder

	// Header
	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	sb.WriteString(" COMBAT SIMULATION REPORT\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n\n")

	sb.WriteString(fmt.Sprintf("Scenario: %s\n", result.Scenario.Name))
	sb.WriteString(fmt.Sprintf("Iterations: %d\n\n", result.Iterations))

	// Win Rate Analysis
	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(" WIN RATE ANALYSIS\n")
	sb.WriteString("───────────────────────────────────────────────────────────\n\n")

	attackerWinRate := float64(result.AttackerWins) / float64(result.Iterations) * 100
	defenderWinRate := float64(result.DefenderWins) / float64(result.Iterations) * 100
	drawRate := float64(result.Draws) / float64(result.Iterations) * 100

	sb.WriteString(fmt.Sprintf("Attacker (%s): %d wins (%.1f%%)\n",
		result.Scenario.AttackerSetup.Name, result.AttackerWins, attackerWinRate))
	sb.WriteString(fmt.Sprintf("Defender (%s): %d wins (%.1f%%)\n",
		result.Scenario.DefenderSetup.Name, result.DefenderWins, defenderWinRate))
	sb.WriteString(fmt.Sprintf("Draws: %d (%.1f%%)\n\n", result.Draws, drawRate))

	// Balance verdict
	if attackerWinRate > 55 || attackerWinRate < 45 {
		sb.WriteString("Verdict: ⚠ IMBALANCED\n\n")
	} else {
		sb.WriteString("Verdict: ✓ BALANCED\n\n")
	}

	// Win rate chart
	sb.WriteString("Win Rate Chart:\n")
	sb.WriteString(r.createProgressBar("Attacker", attackerWinRate))
	sb.WriteString(r.createProgressBar("Defender", defenderWinRate))
	sb.WriteString("\n")

	// Combat Duration
	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(" COMBAT DURATION\n")
	sb.WriteString("───────────────────────────────────────────────────────────\n\n")

	sb.WriteString(fmt.Sprintf("Average Turns: %.1f turns\n", result.AvgTurnsUntilEnd))
	sb.WriteString(fmt.Sprintf("Min Turns:     %d turns\n", result.MinTurns))
	sb.WriteString(fmt.Sprintf("Max Turns:     %d turns\n\n", result.MaxTurns))

	// Damage Analysis
	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(" DAMAGE ANALYSIS\n")
	sb.WriteString("───────────────────────────────────────────────────────────\n\n")

	attackerName := result.Scenario.AttackerSetup.Name
	defenderName := result.Scenario.DefenderSetup.Name

	avgAttackerDamage := float64(result.TotalDamageDealt[attackerName]) / float64(result.Iterations)
	avgDefenderDamage := float64(result.TotalDamageTaken[defenderName]) / float64(result.Iterations)

	sb.WriteString(fmt.Sprintf("Attacker (%s):\n", attackerName))
	sb.WriteString(fmt.Sprintf("  Avg Damage Dealt: %.1f damage\n", avgAttackerDamage))
	sb.WriteString(fmt.Sprintf("  Avg Damage Taken: %.1f damage\n\n", avgDefenderDamage))

	avgAttackerKills := float64(result.TotalUnitsKilled[defenderName]) / float64(result.Iterations)
	sb.WriteString(fmt.Sprintf("Avg Units Killed: %.1f units\n\n", avgAttackerKills))

	// Combat Mechanics
	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(" COMBAT MECHANICS\n")
	sb.WriteString("───────────────────────────────────────────────────────────\n\n")

	totalAttacks := result.TotalHits + result.TotalMisses + result.TotalDodges + result.TotalCrits
	if totalAttacks > 0 {
		hitRate := float64(result.TotalHits+result.TotalCrits) / float64(totalAttacks) * 100
		dodgeRate := float64(result.TotalDodges) / float64(totalAttacks) * 100
		critRate := float64(result.TotalCrits) / float64(totalAttacks) * 100
		missRate := float64(result.TotalMisses) / float64(totalAttacks) * 100

		sb.WriteString(fmt.Sprintf("Hit Rate:   %.1f%%\n", hitRate))
		sb.WriteString(fmt.Sprintf("Dodge Rate: %.1f%%\n", dodgeRate))
		sb.WriteString(fmt.Sprintf("Crit Rate:  %.1f%%\n", critRate))
		sb.WriteString(fmt.Sprintf("Miss Rate:  %.1f%%\n\n", missRate))
	}

	if result.CoverApplied > 0 {
		avgCoverReduction := result.TotalCoverReduction / float64(result.CoverApplied)
		sb.WriteString(fmt.Sprintf("Cover Applications: %d times\n", result.CoverApplied))
		sb.WriteString(fmt.Sprintf("Avg Cover Reduction: %.1f%% damage reduction\n\n", avgCoverReduction*100))
	}

	// Insights
	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(" INSIGHTS & RECOMMENDATIONS\n")
	sb.WriteString("───────────────────────────────────────────────────────────\n\n")

	insights := r.generateInsights(result)
	for _, insight := range insights {
		sb.WriteString(fmt.Sprintf("%s\n", insight))
	}

	sb.WriteString("\n═══════════════════════════════════════════════════════════\n")

	return sb.String()
}

// createProgressBar creates an ASCII progress bar
func (r *ReportFormatter) createProgressBar(label string, percentage float64) string {
	barLength := 40
	filled := int(percentage / 100.0 * float64(barLength))
	empty := barLength - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("%-10s: %s %.1f%%\n", label, bar, percentage)
}

// generateInsights analyzes results and provides recommendations
func (r *ReportFormatter) generateInsights(result *SimulationResult) []string {
	insights := []string{}

	attackerWinRate := float64(result.AttackerWins) / float64(result.Iterations) * 100

	// Win rate balance check
	if attackerWinRate > 55 {
		insights = append(insights, fmt.Sprintf("⚠ HIGH: %s wins %.1f%% (expected ~50%%)",
			result.Scenario.AttackerSetup.Name, attackerWinRate))
		insights = append(insights, "  → Recommendation: Reduce attacker power OR increase defender power")
	} else if attackerWinRate < 45 {
		insights = append(insights, fmt.Sprintf("⚠ HIGH: %s wins only %.1f%% (expected ~50%%)",
			result.Scenario.AttackerSetup.Name, attackerWinRate))
		insights = append(insights, "  → Recommendation: Increase attacker power OR reduce defender power")
	} else {
		insights = append(insights, "✓ OK: Win rates are balanced (~50%)")
	}

	// Combat duration check
	if result.AvgTurnsUntilEnd < 3 {
		insights = append(insights, "\n⚠ INFO: Combats are very short (avg < 3 turns)")
		insights = append(insights, "  → Consider increasing unit HP for longer battles")
	} else if result.AvgTurnsUntilEnd > 15 {
		insights = append(insights, "\n⚠ INFO: Combats are very long (avg > 15 turns)")
		insights = append(insights, "  → Consider increasing damage for faster battles")
	} else {
		insights = append(insights, "\n✓ OK: Combat duration within expected range")
	}

	// Cover system check
	if result.CoverApplied > 0 {
		avgCoverReduction := result.TotalCoverReduction / float64(result.CoverApplied)
		if avgCoverReduction > 0.5 {
			insights = append(insights, "\n⚠ INFO: Cover reducing >50% damage on average")
			insights = append(insights, "  → Cover may be too strong")
		} else {
			insights = append(insights, "\n✓ OK: Cover system working as intended")
		}
	}

	return insights
}
