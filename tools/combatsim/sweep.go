package combatsim

import (
	"fmt"
)

// =============================================================================
// SWEEP EXECUTION
// =============================================================================

// RunSweep executes a complete parameter sweep
func RunSweep(sim *Simulator, baseScenario CombatScenario, config SweepConfig) (*SweepResult, error) {
	result := &SweepResult{
		Config:       config,
		TestedValues: make([]int, 0),
		WinRates:     make([]float64, 0),
		AvgDamage:    make([]float64, 0),
		AvgSurvival:  make([]float64, 0),
		BreakPoints:  make([]BreakPoint, 0),
	}

	// Validate config
	if config.StepSize <= 0 {
		config.StepSize = 1
	}
	if config.IterationsPerStep <= 0 {
		config.IterationsPerStep = sim.config.Iterations
	}

	// Run sweep
	for value := config.MinValue; value <= config.MaxValue; value += config.StepSize {
		// Create mutated scenario
		mutated := MutateScenario(baseScenario, config, value)

		// Create simulator for this step
		stepSim := NewSimulator(SimulationConfig{
			Iterations: config.IterationsPerStep,
			Verbose:    false,
		})

		// Run simulation
		simResult, err := stepSim.Run(mutated)
		if err != nil {
			return nil, fmt.Errorf("sweep at value %d failed: %w", value, err)
		}

		// Record results
		result.TestedValues = append(result.TestedValues, value)

		winRate := float64(simResult.AttackerWins) / float64(simResult.Iterations)
		result.WinRates = append(result.WinRates, winRate)

		// Calculate average damage dealt
		totalDamage := 0
		for _, dmg := range simResult.TotalDamageDealt {
			totalDamage += dmg
		}
		avgDamage := float64(totalDamage) / float64(simResult.Iterations)
		result.AvgDamage = append(result.AvgDamage, avgDamage)

		// Calculate survival (inverse of units killed)
		totalKilled := 0
		for _, kills := range simResult.TotalUnitsKilled {
			totalKilled += kills
		}
		survivalRate := 1.0 - (float64(totalKilled) / float64(simResult.Iterations))
		result.AvgSurvival = append(result.AvgSurvival, survivalRate)
	}

	// Analyze results
	result.BreakPoints = DetectBreakpoints(result)
	result.OptimalValue = FindBalancePoint(result)
	AnalyzeScalingCurve(result)

	return result, nil
}

// MutateScenario creates a modified scenario with attribute change
func MutateScenario(base CombatScenario, config SweepConfig, newValue int) CombatScenario {
	mutated := base.Clone()
	mutated.Name = fmt.Sprintf("%s (%s=%d)", base.Name, config.Attribute, newValue)

	var targetSetup *SquadSetup
	if config.TargetSquad == "Attacker" {
		targetSetup = &mutated.AttackerSetup
	} else {
		targetSetup = &mutated.DefenderSetup
	}

	// Apply attribute override to matching units
	for i := range targetSetup.Units {
		if config.TargetUnit == "*" || targetSetup.Units[i].TemplateName == config.TargetUnit {
			if targetSetup.Units[i].AttributeOverrides == nil {
				targetSetup.Units[i].AttributeOverrides = make(map[string]int)
			}
			targetSetup.Units[i].AttributeOverrides[config.Attribute] = newValue
		}
	}

	return mutated
}

// =============================================================================
// BREAKPOINT DETECTION
// =============================================================================

// DetectBreakpoints finds where metrics cross significant thresholds
func DetectBreakpoints(result *SweepResult) []BreakPoint {
	breakpoints := make([]BreakPoint, 0)

	// Win rate thresholds to detect
	thresholds := []float64{0.40, 0.45, 0.50, 0.55, 0.60}

	for _, threshold := range thresholds {
		bp := detectThresholdCrossing(result.TestedValues, result.WinRates, threshold, "WinRate")
		if bp != nil {
			breakpoints = append(breakpoints, *bp)
		}
	}

	return breakpoints
}

// detectThresholdCrossing finds where a metric crosses a threshold
func detectThresholdCrossing(values []int, metrics []float64, threshold float64, metricName string) *BreakPoint {
	if len(values) < 2 || len(metrics) < 2 {
		return nil
	}

	for i := 1; i < len(metrics); i++ {
		prev := metrics[i-1]
		curr := metrics[i]

		// Check for crossing
		if (prev < threshold && curr >= threshold) || (prev >= threshold && curr < threshold) {
			direction := "above"
			if curr < threshold {
				direction = "below"
			}

			return &BreakPoint{
				AttributeValue: values[i],
				MetricName:     metricName,
				CrossedValue:   threshold,
				Direction:      direction,
			}
		}
	}

	return nil
}

// FindBalancePoint finds the attribute value closest to 50% win rate
func FindBalancePoint(result *SweepResult) int {
	if len(result.WinRates) == 0 {
		return result.Config.BaseValue
	}

	closestIdx := 0
	closestDiff := absFloat(result.WinRates[0] - 0.5)

	for i, wr := range result.WinRates {
		diff := absFloat(wr - 0.5)
		if diff < closestDiff {
			closestDiff = diff
			closestIdx = i
		}
	}

	return result.TestedValues[closestIdx]
}

// =============================================================================
// SCALING CURVE ANALYSIS
// =============================================================================

// AnalyzeScalingCurve analyzes diminishing returns
func AnalyzeScalingCurve(result *SweepResult) {
	if len(result.WinRates) < 3 {
		return
	}

	// Calculate marginal win rate changes
	marginals := make([]float64, len(result.WinRates)-1)
	for i := 1; i < len(result.WinRates); i++ {
		marginals[i-1] = (result.WinRates[i] - result.WinRates[i-1]) / float64(result.Config.StepSize)
	}

	// Find where marginal returns start to decrease significantly
	if len(marginals) > 0 {
		result.MarginalAtBase = marginals[0]
		result.MarginalAtCap = marginals[len(marginals)-1]
	}

	// Linear region: where marginal changes are consistent (within 20%)
	linearEnd := 0
	if len(marginals) > 1 {
		baseMarginal := absFloat(marginals[0])
		for i, m := range marginals {
			if absFloat(absFloat(m)-baseMarginal) > baseMarginal*0.2 {
				break
			}
			linearEnd = i
		}
		result.LinearRegionEnd = result.TestedValues[linearEnd]
	}

	// Diminishing returns: where marginal drops below 50% of initial
	for i, m := range marginals {
		if absFloat(m) < absFloat(result.MarginalAtBase)*0.5 {
			result.DiminishingStart = result.TestedValues[i]
			break
		}
	}
}

// =============================================================================
// SWEEP REPORT FORMATTING
// =============================================================================

// FormatSweepReport creates a human-readable sweep report
func FormatSweepReport(result *SweepResult) string {
	report := "═══════════════════════════════════════════════════════════\n"
	report += " PARAMETER SWEEP REPORT\n"
	report += "═══════════════════════════════════════════════════════════\n\n"

	report += fmt.Sprintf("Attribute: %s\n", result.Config.Attribute)
	report += fmt.Sprintf("Target: %s / %s\n", result.Config.TargetSquad, result.Config.TargetUnit)
	report += fmt.Sprintf("Range: %d to %d (step %d)\n\n", result.Config.MinValue, result.Config.MaxValue, result.Config.StepSize)

	// Win rate curve
	report += "WIN RATE BY VALUE:\n"
	report += "───────────────────────────────────────────────────────────\n"

	for i, value := range result.TestedValues {
		winRate := result.WinRates[i]
		barLen := int(winRate * 40)
		bar := ""
		for j := 0; j < barLen; j++ {
			bar += "█"
		}
		for j := barLen; j < 40; j++ {
			bar += "░"
		}

		marker := " "
		if value == result.OptimalValue {
			marker = "*"
		}

		report += fmt.Sprintf("  %3d %s %s %.1f%%\n", value, marker, bar, winRate*100)
	}

	report += "\n"

	// Balance point
	report += fmt.Sprintf("OPTIMAL VALUE (closest to 50%%): %d\n", result.OptimalValue)

	// Breakpoints
	if len(result.BreakPoints) > 0 {
		report += "\nBREAKPOINTS:\n"
		for _, bp := range result.BreakPoints {
			report += fmt.Sprintf("  - %s crosses %.0f%% at value %d (%s)\n",
				bp.MetricName, bp.CrossedValue*100, bp.AttributeValue, bp.Direction)
		}
	}

	// Scaling analysis
	report += "\nSCALING ANALYSIS:\n"
	report += fmt.Sprintf("  Marginal at base: %.2f%% per point\n", result.MarginalAtBase*100)
	report += fmt.Sprintf("  Marginal at cap:  %.2f%% per point\n", result.MarginalAtCap*100)
	if result.LinearRegionEnd > 0 {
		report += fmt.Sprintf("  Linear region ends at: %d\n", result.LinearRegionEnd)
	}
	if result.DiminishingStart > 0 {
		report += fmt.Sprintf("  Diminishing returns start at: %d\n", result.DiminishingStart)
	}

	report += "\n═══════════════════════════════════════════════════════════\n"

	return report
}
