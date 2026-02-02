package combatsim

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// =============================================================================
// BALANCE REPORT GENERATION
// =============================================================================

// GenerateBalanceReport creates a comprehensive balance report
func GenerateBalanceReport(
	result *SimulationResult,
	timelines []TimelineData,
	unitPerf [][]UnitPerformanceData,
	analysisMode string,
) *BalanceReport {
	report := &BalanceReport{
		ScenarioName: result.Scenario.Name,
		Iterations:   result.Iterations,
		AnalysisMode: analysisMode,
	}

	// Always include win rate and duration (quick mode)
	report.WinRate = AnalyzeWinRate(result)
	report.Duration = AnalyzeDuration(result, timelines)

	// Standard mode: add unit performance and mechanics
	if analysisMode == AnalysisModeStandard || analysisMode == AnalysisModeComprehensive {
		unitAggs := AggregateUnitPerformance(unitPerf)
		report.UnitPerformance = CreateUnitPerformanceSection(unitAggs, result)
		report.Mechanics = CreateMechanicsSection(result)
	}

	// Comprehensive mode: add timeline and confidence
	if analysisMode == AnalysisModeComprehensive {
		report.Timeline = CreateTimelineSection(timelines)
		report.Confidence = CreateConfidenceSection(result)
	}

	// Generate insights based on analysis mode
	report.Insights = GenerateInsights(report)

	return report
}

// =============================================================================
// REPORT FORMATTING
// =============================================================================

// FormatBalanceReport creates human-readable balance report
func FormatBalanceReport(report *BalanceReport) string {
	var sb strings.Builder

	// Header
	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	sb.WriteString(" COMBAT BALANCE ANALYSIS REPORT\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n\n")

	sb.WriteString(fmt.Sprintf("Scenario: %s\n", report.ScenarioName))
	sb.WriteString(fmt.Sprintf("Iterations: %d", report.Iterations))
	if report.Confidence != nil {
		sb.WriteString(fmt.Sprintf(" | MOE: ±%.1f%%", report.Confidence.MarginOfError*100))
	}
	sb.WriteString(fmt.Sprintf(" | Mode: %s\n\n", report.AnalysisMode))

	// Win Rate Section
	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(" WIN RATE ANALYSIS\n")
	sb.WriteString("───────────────────────────────────────────────────────────\n\n")

	sb.WriteString(fmt.Sprintf("  Attacker: %.1f%%", report.WinRate.AttackerWinRate*100))
	if report.WinRate.AttackerCI95[0] > 0 || report.WinRate.AttackerCI95[1] > 0 {
		sb.WriteString(fmt.Sprintf(" (95%% CI: %.1f%% - %.1f%%)",
			report.WinRate.AttackerCI95[0]*100, report.WinRate.AttackerCI95[1]*100))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  Defender: %.1f%%", report.WinRate.DefenderWinRate*100))
	if report.WinRate.DefenderCI95[0] > 0 || report.WinRate.DefenderCI95[1] > 0 {
		sb.WriteString(fmt.Sprintf(" (95%% CI: %.1f%% - %.1f%%)",
			report.WinRate.DefenderCI95[0]*100, report.WinRate.DefenderCI95[1]*100))
	}
	sb.WriteString("\n")

	if report.WinRate.DrawRate > 0 {
		sb.WriteString(fmt.Sprintf("  Draws: %.1f%%\n", report.WinRate.DrawRate*100))
	}

	sb.WriteString("\n  Verdict: ")
	if report.WinRate.IsBalanced {
		sb.WriteString("BALANCED\n")
	} else {
		sb.WriteString(fmt.Sprintf("IMBALANCED (%s favored by %.1f%%)\n",
			report.WinRate.FavoredSide, report.WinRate.ImbalanceAmount*200))
	}
	sb.WriteString("\n")

	// Duration Section
	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(" COMBAT DURATION\n")
	sb.WriteString("───────────────────────────────────────────────────────────\n\n")

	sb.WriteString(fmt.Sprintf("  Average: %.1f rounds\n", report.Duration.Average))
	sb.WriteString(fmt.Sprintf("  Median: %.1f rounds\n", report.Duration.Median))
	sb.WriteString(fmt.Sprintf("  Range: %d - %d rounds\n", report.Duration.Min, report.Duration.Max))

	if len(report.Duration.DistributionBuckets) > 0 {
		sb.WriteString("\n  Distribution:\n")
		buckets := []string{"1-2", "3-4", "5-6", "7-8", "9+"}
		for _, bucket := range buckets {
			if count, ok := report.Duration.DistributionBuckets[bucket]; ok && count > 0 {
				pct := float64(count) / float64(report.Iterations) * 100
				barLen := int(pct / 2.5)
				bar := strings.Repeat("█", barLen)
				sb.WriteString(fmt.Sprintf("    %s: %s %.0f%%\n", bucket, bar, pct))
			}
		}
	}
	sb.WriteString("\n")

	// Unit Performance Section (standard+)
	if report.UnitPerformance != nil {
		sb.WriteString("───────────────────────────────────────────────────────────\n")
		sb.WriteString(" PER-UNIT PERFORMANCE\n")
		sb.WriteString("───────────────────────────────────────────────────────────\n\n")

		for name, data := range report.UnitPerformance.ByUnit {
			sb.WriteString(fmt.Sprintf("  %s (%s):\n", name, data.Role))
			sb.WriteString(fmt.Sprintf("    Dmg Dealt: %.1f | Dmg Taken: %.1f\n",
				data.AvgDamageDealt, data.AvgDamageTaken))
			sb.WriteString(fmt.Sprintf("    Survival: %.0f%% | Efficiency: %.2f\n",
				data.SurvivalRate*100, data.Efficiency))
			if data.CoverProvided > 0 {
				sb.WriteString(fmt.Sprintf("    Cover Provided: %.1f\n", data.CoverProvided))
			}
			sb.WriteString("\n")
		}

		// Top/Under performers
		if len(report.UnitPerformance.TopPerformers) > 0 {
			sb.WriteString("  Top Performers: " + strings.Join(report.UnitPerformance.TopPerformers, ", ") + "\n")
		}
		if len(report.UnitPerformance.UnderPerformers) > 0 {
			sb.WriteString("  Under-Performers: " + strings.Join(report.UnitPerformance.UnderPerformers, ", ") + "\n")
		}
		sb.WriteString("\n")
	}

	// Timeline Section (comprehensive)
	if report.Timeline != nil {
		sb.WriteString("───────────────────────────────────────────────────────────\n")
		sb.WriteString(" TIMELINE ANALYSIS\n")
		sb.WriteString("───────────────────────────────────────────────────────────\n\n")

		sb.WriteString(fmt.Sprintf("  First Blood: Round %.1f\n", report.Timeline.AvgFirstBlood))
		if report.Timeline.AvgTurningPoint > 0 {
			sb.WriteString(fmt.Sprintf("  Turning Point: Round %.1f\n", report.Timeline.AvgTurningPoint))
		}
		sb.WriteString(fmt.Sprintf("  Snowball Factor: %.2f", report.Timeline.SnowballFactor))
		if report.Timeline.SnowballFactor < 1.0 {
			sb.WriteString(" (comebacks possible)")
		} else if report.Timeline.SnowballFactor > 1.5 {
			sb.WriteString(" (strong snowball)")
		} else {
			sb.WriteString(" (moderate)")
		}
		sb.WriteString("\n")

		// Damage curve
		if len(report.Timeline.DamageCurve) > 0 {
			sb.WriteString("\n  Damage Curve:\n")
			maxDmg := 0.0
			for _, d := range report.Timeline.DamageCurve {
				if d > maxDmg {
					maxDmg = d
				}
			}
			for i, d := range report.Timeline.DamageCurve {
				if i >= 10 {
					break // Limit display to 10 rounds
				}
				barLen := 0
				if maxDmg > 0 {
					barLen = int(d / maxDmg * 20)
				}
				bar := strings.Repeat("█", barLen)
				sb.WriteString(fmt.Sprintf("    R%d: %s %.0f\n", i+1, bar, d))
			}
		}
		sb.WriteString("\n")
	}

	// Mechanics Section (standard+)
	if report.Mechanics != nil {
		sb.WriteString("───────────────────────────────────────────────────────────\n")
		sb.WriteString(" MECHANICS IMPACT\n")
		sb.WriteString("───────────────────────────────────────────────────────────\n\n")

		sb.WriteString(fmt.Sprintf("  Cover: %.0f%% activation, %.0f%% avg reduction\n",
			report.Mechanics.CoverActivation*100, report.Mechanics.CoverEffectiveness*100))
		sb.WriteString(fmt.Sprintf("  Crit: %.0f%% rate, +%.0f%% damage\n",
			report.Mechanics.CritRate*100, report.Mechanics.CritImpact*100))
		sb.WriteString(fmt.Sprintf("  Dodge: %.0f%% rate\n",
			report.Mechanics.DodgeRate*100))
		sb.WriteString("\n")
	}

	// Insights Section
	if len(report.Insights) > 0 {
		sb.WriteString("───────────────────────────────────────────────────────────\n")
		sb.WriteString(" BALANCE INSIGHTS\n")
		sb.WriteString("───────────────────────────────────────────────────────────\n\n")

		for _, insight := range report.Insights {
			icon := "ℹ"
			switch insight.Severity {
			case "Critical":
				icon = "⚠"
			case "Warning":
				icon = "⚡"
			}
			sb.WriteString(fmt.Sprintf("  [%s %s] %s\n", icon, insight.Severity, insight.Issue))
			sb.WriteString(fmt.Sprintf("      Evidence: %s\n", insight.Evidence))
			sb.WriteString(fmt.Sprintf("      Suggestion: %s\n\n", insight.Suggestion))
		}
	}

	// Confidence Section (comprehensive)
	if report.Confidence != nil {
		sb.WriteString("───────────────────────────────────────────────────────────\n")
		sb.WriteString(" STATISTICAL CONFIDENCE\n")
		sb.WriteString("───────────────────────────────────────────────────────────\n\n")

		sb.WriteString(fmt.Sprintf("  Sample Size: %d\n", report.Confidence.SampleSize))
		sb.WriteString(fmt.Sprintf("  Margin of Error: ±%.1f%%\n", report.Confidence.MarginOfError*100))

		if report.Confidence.IsStatisticallySound {
			sb.WriteString("  Status: Statistically Sound\n")
		} else {
			sb.WriteString(fmt.Sprintf("  Status: Needs More Samples (recommend %d)\n",
				report.Confidence.RecommendedSamples))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("═══════════════════════════════════════════════════════════\n")

	return sb.String()
}

// =============================================================================
// EXPORT FUNCTIONS
// =============================================================================

// ExportJSON exports report data in JSON format
func ExportJSON(report *BalanceReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ExportCSV exports key metrics in CSV format
func ExportCSV(report *BalanceReport, path string) error {
	var sb strings.Builder

	// Header
	sb.WriteString("Metric,Value\n")

	// Win rates
	sb.WriteString(fmt.Sprintf("AttackerWinRate,%.4f\n", report.WinRate.AttackerWinRate))
	sb.WriteString(fmt.Sprintf("DefenderWinRate,%.4f\n", report.WinRate.DefenderWinRate))
	sb.WriteString(fmt.Sprintf("DrawRate,%.4f\n", report.WinRate.DrawRate))
	sb.WriteString(fmt.Sprintf("IsBalanced,%v\n", report.WinRate.IsBalanced))

	// Duration
	sb.WriteString(fmt.Sprintf("AvgDuration,%.2f\n", report.Duration.Average))
	sb.WriteString(fmt.Sprintf("MinDuration,%d\n", report.Duration.Min))
	sb.WriteString(fmt.Sprintf("MaxDuration,%d\n", report.Duration.Max))

	// Mechanics
	if report.Mechanics != nil {
		sb.WriteString(fmt.Sprintf("CoverActivation,%.4f\n", report.Mechanics.CoverActivation))
		sb.WriteString(fmt.Sprintf("CoverEffectiveness,%.4f\n", report.Mechanics.CoverEffectiveness))
		sb.WriteString(fmt.Sprintf("CritRate,%.4f\n", report.Mechanics.CritRate))
		sb.WriteString(fmt.Sprintf("DodgeRate,%.4f\n", report.Mechanics.DodgeRate))
	}

	// Confidence
	if report.Confidence != nil {
		sb.WriteString(fmt.Sprintf("SampleSize,%d\n", report.Confidence.SampleSize))
		sb.WriteString(fmt.Sprintf("MarginOfError,%.4f\n", report.Confidence.MarginOfError))
	}

	err := os.WriteFile(path, []byte(sb.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ExportUnitCSV exports per-unit metrics in CSV format
func ExportUnitCSV(report *BalanceReport, path string) error {
	if report.UnitPerformance == nil {
		return fmt.Errorf("no unit performance data available")
	}

	var sb strings.Builder

	// Header
	sb.WriteString("Unit,Role,AvgDamageDealt,AvgDamageTaken,SurvivalRate,Efficiency,CoverProvided\n")

	for name, data := range report.UnitPerformance.ByUnit {
		sb.WriteString(fmt.Sprintf("%s,%s,%.2f,%.2f,%.4f,%.4f,%.2f\n",
			name, data.Role, data.AvgDamageDealt, data.AvgDamageTaken,
			data.SurvivalRate, data.Efficiency, data.CoverProvided))
	}

	err := os.WriteFile(path, []byte(sb.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// =============================================================================
// COMPARISON REPORTS
// =============================================================================

// =============================================================================
// QUICK REPORT
// =============================================================================

// FormatQuickReport creates a minimal report for quick analysis
func FormatQuickReport(result *SimulationResult) string {
	var sb strings.Builder

	attackerWR := float64(result.AttackerWins) / float64(result.Iterations) * 100
	defenderWR := float64(result.DefenderWins) / float64(result.Iterations) * 100

	sb.WriteString(fmt.Sprintf("%s: ", result.Scenario.Name))
	sb.WriteString(fmt.Sprintf("Attacker %.0f%% | Defender %.0f%% | ", attackerWR, defenderWR))
	sb.WriteString(fmt.Sprintf("Avg %.1f rounds | ", result.AvgTurnsUntilEnd))

	// Quick balance verdict
	if attackerWR >= 45 && attackerWR <= 55 {
		sb.WriteString("BALANCED")
	} else if attackerWR > 55 {
		sb.WriteString(fmt.Sprintf("Attacker +%.0f%%", attackerWR-50))
	} else {
		sb.WriteString(fmt.Sprintf("Defender +%.0f%%", defenderWR-50))
	}

	return sb.String()
}
