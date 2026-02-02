package combatsim

// =============================================================================
// UNIT PERFORMANCE AGGREGATION
// =============================================================================

// AggregateUnitPerformance combines unit performance across simulations
// Returns a map keyed by TemplateName
func AggregateUnitPerformance(allPerf [][]UnitPerformanceData) map[string]*UnitPerformanceAggregated {
	aggs := make(map[string]*UnitPerformanceAggregated)

	for _, simPerf := range allPerf {
		for _, perf := range simPerf {
			if perf.TemplateName == "" {
				continue
			}

			agg, exists := aggs[perf.TemplateName]
			if !exists {
				agg = &UnitPerformanceAggregated{
					TemplateName:         perf.TemplateName,
					Role:                 perf.Role,
					DamageDealtValues:    make([]int, 0),
					DamageReceivedValues: make([]int, 0),
					DeathTurns:           make([]int, 0),
				}
				aggs[perf.TemplateName] = agg
			}

			agg.SampleCount++

			// Damage aggregation
			agg.TotalDamageDealt += int64(perf.DamageDealt)
			agg.TotalDamageReceived += int64(perf.DamageReceived)
			agg.TotalDamageBlocked += int64(perf.DamageBlocked)

			// Combat actions
			agg.TotalAttacksAttempted += perf.AttacksAttempted
			agg.TotalAttacksHit += perf.AttacksHit
			agg.TotalAttacksCrit += perf.AttacksCrit
			agg.TotalAttacksMissed += perf.AttacksMissed
			agg.TotalAttacksDodged += perf.AttacksDodged

			// Survivability
			if perf.TurnOfDeath < 0 {
				agg.SurvivalCount++
			} else {
				agg.DeathCount++
				agg.DeathTurns = append(agg.DeathTurns, perf.TurnOfDeath)
			}

			// For variance calculations
			agg.DamageDealtValues = append(agg.DamageDealtValues, perf.DamageDealt)
			agg.DamageReceivedValues = append(agg.DamageReceivedValues, perf.DamageReceived)

			// Cover
			agg.TotalCoverInstances += perf.CoverInstancesProvided
			agg.TotalCoverReduction += perf.DamageReductionProvided
		}
	}

	return aggs
}

// CalculateUnitEfficiency computes damage dealt per damage received
func CalculateUnitEfficiency(agg *UnitPerformanceAggregated) float64 {
	if agg.TotalDamageReceived == 0 {
		if agg.TotalDamageDealt > 0 {
			return float64(agg.TotalDamageDealt) // Perfect efficiency if no damage taken
		}
		return 0
	}
	return float64(agg.TotalDamageDealt) / float64(agg.TotalDamageReceived)
}

// CalculateSurvivalRate computes survival rate
func CalculateSurvivalRate(agg *UnitPerformanceAggregated) float64 {
	if agg.SampleCount == 0 {
		return 0
	}
	return float64(agg.SurvivalCount) / float64(agg.SampleCount)
}

// CalculateCritRate computes critical hit rate
func CalculateCritRate(agg *UnitPerformanceAggregated) float64 {
	if agg.TotalAttacksAttempted == 0 {
		return 0
	}
	return float64(agg.TotalAttacksCrit) / float64(agg.TotalAttacksAttempted)
}

// CalculateAvgDamageDealt computes average damage per simulation
func CalculateAvgDamageDealt(agg *UnitPerformanceAggregated) float64 {
	if agg.SampleCount == 0 {
		return 0
	}
	return float64(agg.TotalDamageDealt) / float64(agg.SampleCount)
}

// CalculateAvgDamageReceived computes average damage received per simulation
func CalculateAvgDamageReceived(agg *UnitPerformanceAggregated) float64 {
	if agg.SampleCount == 0 {
		return 0
	}
	return float64(agg.TotalDamageReceived) / float64(agg.SampleCount)
}

// CalculateAvgDeathTurn computes average turn of death
func CalculateAvgDeathTurn(agg *UnitPerformanceAggregated) float64 {
	if len(agg.DeathTurns) == 0 {
		return 0
	}
	return CalculateMeanInt(agg.DeathTurns)
}

// =============================================================================
// ROLE EFFECTIVENESS ANALYSIS
// =============================================================================

// AnalyzeRoleEffectiveness computes role contribution metrics
func AnalyzeRoleEffectiveness(unitAggs map[string]*UnitPerformanceAggregated) map[string]*RoleEffectivenessData {
	roles := make(map[string]*RoleEffectivenessData)

	// Initialize role aggregators
	roleAggs := make(map[string]struct {
		totalDamage   int64
		totalReceived int64
		survivalCount int
		totalCount    int
		deathTurns    []int
		coverProvided float64
		unitCount     int
	})

	// Aggregate by role
	for _, agg := range unitAggs {
		role := agg.Role
		if role == "" {
			role = "Unknown"
		}

		ra := roleAggs[role]
		ra.totalDamage += agg.TotalDamageDealt
		ra.totalReceived += agg.TotalDamageReceived
		ra.survivalCount += agg.SurvivalCount
		ra.totalCount += agg.SampleCount
		ra.deathTurns = append(ra.deathTurns, agg.DeathTurns...)
		ra.coverProvided += agg.TotalCoverReduction
		ra.unitCount++
		roleAggs[role] = ra
	}

	// Calculate total damage for share calculation
	var totalDamageAll int64
	for _, ra := range roleAggs {
		totalDamageAll += ra.totalDamage
	}

	// Create role effectiveness data
	for role, ra := range roleAggs {
		red := &RoleEffectivenessData{
			Role:      role,
			UnitCount: ra.unitCount,
		}

		if ra.totalCount > 0 {
			red.AvgDamageDealt = float64(ra.totalDamage) / float64(ra.totalCount)
			red.AvgSurvivalRate = float64(ra.survivalCount) / float64(ra.totalCount)
		}

		if totalDamageAll > 0 {
			red.DamageShare = float64(ra.totalDamage) / float64(totalDamageAll)
		}

		if len(ra.deathTurns) > 0 {
			red.AvgDeathTurn = CalculateMeanInt(ra.deathTurns)
		}

		red.CoverProvided = ra.coverProvided

		roles[role] = red
	}

	return roles
}

// =============================================================================
// MECHANICS IMPACT ANALYSIS
// =============================================================================

// AnalyzeCoverImpact quantifies cover system effectiveness
func AnalyzeCoverImpact(result *SimulationResult) *MechanicImpactData {
	impact := &MechanicImpactData{
		MechanicName: "Cover",
	}

	totalAttacks := result.TotalHits + result.TotalMisses + result.TotalDodges + result.TotalCrits

	if totalAttacks > 0 {
		impact.TotalOpportunities = totalAttacks
		impact.ActivationCount = result.CoverApplied
		impact.ActivationRate = float64(result.CoverApplied) / float64(totalAttacks)
	}

	if result.CoverApplied > 0 {
		impact.AvgDamageReduction = result.TotalCoverReduction / float64(result.CoverApplied)
	}

	return impact
}

// AnalyzeCritImpact quantifies critical hit system impact
func AnalyzeCritImpact(result *SimulationResult) *MechanicImpactData {
	impact := &MechanicImpactData{
		MechanicName: "Crit",
	}

	totalHits := result.TotalHits + result.TotalCrits
	if totalHits > 0 {
		impact.TotalOpportunities = totalHits
		impact.ActivationCount = result.TotalCrits
		impact.ActivationRate = float64(result.TotalCrits) / float64(totalHits)
		impact.AvgDamageIncrease = 0.5 // Crits do 1.5x damage, so 50% increase
	}

	return impact
}

// AnalyzeDodgeImpact quantifies dodge system impact
func AnalyzeDodgeImpact(result *SimulationResult) *MechanicImpactData {
	impact := &MechanicImpactData{
		MechanicName: "Dodge",
	}

	totalAttempts := result.TotalHits + result.TotalMisses + result.TotalDodges + result.TotalCrits
	if totalAttempts > 0 {
		impact.TotalOpportunities = totalAttempts
		impact.ActivationCount = result.TotalDodges
		impact.ActivationRate = float64(result.TotalDodges) / float64(totalAttempts)
		impact.AvgDamageReduction = 1.0 // Dodge negates all damage
	}

	return impact
}

// AnalyzeMechanicsImpact groups all mechanic impact analysis
func AnalyzeMechanicsImpact(result *SimulationResult) *MechanicsAnalysis {
	return &MechanicsAnalysis{
		Cover:      AnalyzeCoverImpact(result),
		Crit:       AnalyzeCritImpact(result),
		Dodge:      AnalyzeDodgeImpact(result),
		Range:      make(map[int]*MechanicImpactData),
		AttackType: make(map[string]*MechanicImpactData),
	}
}

// =============================================================================
// WIN RATE ANALYSIS
// =============================================================================

// AnalyzeWinRate creates detailed win rate analysis
func AnalyzeWinRate(result *SimulationResult) WinRateSection {
	section := WinRateSection{}

	if result.Iterations == 0 {
		return section
	}

	section.AttackerWinRate = float64(result.AttackerWins) / float64(result.Iterations)
	section.DefenderWinRate = float64(result.DefenderWins) / float64(result.Iterations)
	section.DrawRate = float64(result.Draws) / float64(result.Iterations)

	// Calculate confidence intervals using Wilson score
	section.AttackerCI95[0], section.AttackerCI95[1] = CalculateProportionCI95(result.AttackerWins, result.Iterations)
	section.DefenderCI95[0], section.DefenderCI95[1] = CalculateProportionCI95(result.DefenderWins, result.Iterations)

	// Balance assessment
	section.IsBalanced = section.AttackerWinRate >= 0.45 && section.AttackerWinRate <= 0.55
	section.ImbalanceAmount = absFloat(section.AttackerWinRate - 0.5)

	if section.AttackerWinRate > section.DefenderWinRate+0.05 {
		section.FavoredSide = "Attacker"
	} else if section.DefenderWinRate > section.AttackerWinRate+0.05 {
		section.FavoredSide = "Defender"
	} else {
		section.FavoredSide = "Neither"
	}

	return section
}

// =============================================================================
// DURATION ANALYSIS
// =============================================================================

// AnalyzeDuration creates detailed duration analysis
func AnalyzeDuration(result *SimulationResult, timelines []TimelineData) DurationSection {
	section := DurationSection{
		DistributionBuckets: make(map[string]int),
	}

	if result.Iterations == 0 {
		return section
	}

	// Collect all durations
	durations := make([]int, 0, len(timelines))
	for _, tl := range timelines {
		durations = append(durations, tl.CombatDuration)
	}

	if len(durations) == 0 {
		return section
	}

	section.Average = CalculateMeanInt(durations)
	section.Median = CalculateMedianInt(durations)
	section.StdDev = CalculateStdDevInt(durations, section.Average)

	// Find min/max
	section.Min = durations[0]
	section.Max = durations[0]
	for _, d := range durations {
		if d < section.Min {
			section.Min = d
		}
		if d > section.Max {
			section.Max = d
		}
	}

	// Create distribution buckets
	for _, d := range durations {
		var bucket string
		switch {
		case d <= 2:
			bucket = "1-2"
		case d <= 4:
			bucket = "3-4"
		case d <= 6:
			bucket = "5-6"
		case d <= 8:
			bucket = "7-8"
		default:
			bucket = "9+"
		}
		section.DistributionBuckets[bucket]++
	}

	// Healthy duration is typically 3-8 rounds
	section.IsHealthyDuration = section.Average >= 3 && section.Average <= 8

	return section
}

// =============================================================================
// UNIT PERFORMANCE SECTION
// =============================================================================

// CreateUnitPerformanceSection creates the unit performance report section
func CreateUnitPerformanceSection(unitAggs map[string]*UnitPerformanceAggregated, result *SimulationResult) *UnitPerformanceSection {
	section := &UnitPerformanceSection{
		ByUnit:          make(map[string]*UnitReportData),
		TopPerformers:   make([]string, 0),
		UnderPerformers: make([]string, 0),
	}

	// Create per-unit data
	type efficiencyEntry struct {
		name       string
		efficiency float64
	}
	efficiencies := make([]efficiencyEntry, 0)

	for name, agg := range unitAggs {
		urd := &UnitReportData{
			TemplateName:   name,
			Role:           agg.Role,
			AvgDamageDealt: CalculateAvgDamageDealt(agg),
			AvgDamageTaken: CalculateAvgDamageReceived(agg),
			SurvivalRate:   CalculateSurvivalRate(agg),
			Efficiency:     CalculateUnitEfficiency(agg),
			AvgDeathTurn:   CalculateAvgDeathTurn(agg),
			CoverProvided:  agg.TotalCoverReduction,
		}

		section.ByUnit[name] = urd
		efficiencies = append(efficiencies, efficiencyEntry{name: name, efficiency: urd.Efficiency})
	}

	// Sort by efficiency to find top/under performers
	for i := 0; i < len(efficiencies); i++ {
		for j := i + 1; j < len(efficiencies); j++ {
			if efficiencies[j].efficiency > efficiencies[i].efficiency {
				efficiencies[i], efficiencies[j] = efficiencies[j], efficiencies[i]
			}
		}
	}

	// Top 3 performers
	for i := 0; i < len(efficiencies) && i < 3; i++ {
		section.TopPerformers = append(section.TopPerformers, efficiencies[i].name)
	}

	// Bottom 3 under-performers
	for i := len(efficiencies) - 1; i >= 0 && len(section.UnderPerformers) < 3; i-- {
		section.UnderPerformers = append(section.UnderPerformers, efficiencies[i].name)
	}

	// Role effectiveness
	section.ByRole = AnalyzeRoleEffectiveness(unitAggs)

	return section
}

// =============================================================================
// TIMELINE SECTION
// =============================================================================

// CreateTimelineSection creates the timeline analysis section
func CreateTimelineSection(timelines []TimelineData) *TimelineSection {
	section := &TimelineSection{}

	if len(timelines) == 0 {
		return section
	}

	agg := AggregateTimelines(timelines)

	section.AvgFirstBlood = agg.AvgFirstBlood
	section.AvgTurningPoint = agg.AvgTurningPoint
	section.AvgDuration = agg.AvgDuration

	// Calculate snowball factor across all timelines
	snowballSum := 0.0
	for _, tl := range timelines {
		snowballSum += CalculateSnowballFactor(tl)
	}
	section.SnowballFactor = snowballSum / float64(len(timelines))

	// Generate damage curve
	section.DamageCurve = GenerateDamageCurve(agg)

	return section
}

// =============================================================================
// MECHANICS SECTION
// =============================================================================

// CreateMechanicsSection creates the mechanics analysis section
func CreateMechanicsSection(result *SimulationResult) *MechanicsSection {
	section := &MechanicsSection{}

	analysis := AnalyzeMechanicsImpact(result)

	if analysis.Cover != nil {
		section.CoverEffectiveness = analysis.Cover.AvgDamageReduction
		section.CoverActivation = analysis.Cover.ActivationRate
	}

	if analysis.Crit != nil {
		section.CritRate = analysis.Crit.ActivationRate
		section.CritImpact = analysis.Crit.AvgDamageIncrease
	}

	if analysis.Dodge != nil {
		section.DodgeRate = analysis.Dodge.ActivationRate
		section.DodgeImpact = analysis.Dodge.AvgDamageReduction
	}

	return section
}

// =============================================================================
// CONFIDENCE SECTION
// =============================================================================

// CreateConfidenceSection creates the statistical confidence section
func CreateConfidenceSection(result *SimulationResult) *ConfidenceSection {
	section := &ConfidenceSection{
		SampleSize: result.Iterations,
	}

	// Calculate margin of error for win rate
	winRate := float64(result.AttackerWins) / float64(result.Iterations)
	section.MarginOfError = CalculateProportionMarginOfError(winRate, result.Iterations, ZScore95)

	section.IsStatisticallySound = IsStatisticallySound(section.MarginOfError)

	// Recommend samples for 1.5% margin of error
	if !section.IsStatisticallySound {
		stdDev := winRate * (1 - winRate) // Variance of binomial
		section.RecommendedSamples = RecommendSampleSize(stdDev, 0.015, ZScore95)
	}

	return section
}

// =============================================================================
// INSIGHT GENERATION
// =============================================================================

// GenerateInsights creates actionable balance recommendations
func GenerateInsights(report *BalanceReport) []BalanceInsight {
	insights := make([]BalanceInsight, 0)

	// Win rate imbalance
	if !report.WinRate.IsBalanced {
		severity := "Warning"
		if report.WinRate.ImbalanceAmount > 0.15 {
			severity = "Critical"
		}

		insights = append(insights, BalanceInsight{
			Severity:   severity,
			Category:   "WinRate",
			Issue:      report.WinRate.FavoredSide + " is favored",
			Evidence:   formatPercent(report.WinRate.ImbalanceAmount*2) + " win rate difference",
			Suggestion: "Adjust " + report.WinRate.FavoredSide + " squad stats",
		})
	}

	// Duration issues
	if report.Duration.Average < 2 {
		insights = append(insights, BalanceInsight{
			Severity:   "Warning",
			Category:   "Duration",
			Issue:      "Combat ends too quickly",
			Evidence:   formatFloat(report.Duration.Average, 1) + " avg rounds",
			Suggestion: "Increase HP or reduce damage output",
		})
	} else if report.Duration.Average > 10 {
		insights = append(insights, BalanceInsight{
			Severity:   "Warning",
			Category:   "Duration",
			Issue:      "Combat drags on too long",
			Evidence:   formatFloat(report.Duration.Average, 1) + " avg rounds",
			Suggestion: "Increase damage output or reduce HP",
		})
	}

	// Unit efficiency issues
	if report.UnitPerformance != nil {
		for name, data := range report.UnitPerformance.ByUnit {
			if data.Efficiency > 3.0 {
				insights = append(insights, BalanceInsight{
					Severity:   "Warning",
					Category:   "Unit",
					Issue:      name + " is overperforming",
					Evidence:   formatFloat(data.Efficiency, 2) + " efficiency ratio",
					Suggestion: "Reduce " + name + " damage or increase fragility",
				})
			} else if data.Efficiency < 0.3 && data.SurvivalRate < 0.3 {
				insights = append(insights, BalanceInsight{
					Severity:   "Info",
					Category:   "Unit",
					Issue:      name + " is underperforming",
					Evidence:   formatFloat(data.Efficiency, 2) + " efficiency, " + formatPercent(data.SurvivalRate) + " survival",
					Suggestion: "Increase " + name + " damage or survivability",
				})
			}
		}
	}

	// Cover effectiveness
	if report.Mechanics != nil && report.Mechanics.CoverActivation > 0.5 && report.Mechanics.CoverEffectiveness < 0.1 {
		insights = append(insights, BalanceInsight{
			Severity:   "Info",
			Category:   "Mechanic",
			Issue:      "Cover activates often but has low impact",
			Evidence:   formatPercent(report.Mechanics.CoverActivation) + " activation, " + formatPercent(report.Mechanics.CoverEffectiveness) + " reduction",
			Suggestion: "Increase cover values or reduce activation frequency",
		})
	}

	// Sample size warning
	if report.Confidence != nil && !report.Confidence.IsStatisticallySound {
		insights = append(insights, BalanceInsight{
			Severity:   "Info",
			Category:   "Statistical",
			Issue:      "Sample size may be insufficient",
			Evidence:   formatPercent(report.Confidence.MarginOfError) + " margin of error",
			Suggestion: "Increase iterations to " + formatInt(report.Confidence.RecommendedSamples),
		})
	}

	return insights
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func formatPercent(f float64) string {
	return formatFloat(f*100, 1) + "%"
}

func formatFloat(f float64, decimals int) string {
	format := "%." + formatInt(decimals) + "f"
	return sprintf(format, f)
}

func formatInt(i int) string {
	return sprintf("%d", i)
}

// sprintf is a simple format function without importing fmt
func sprintf(format string, a ...interface{}) string {
	// Simple implementation for common cases
	result := format
	for _, v := range a {
		switch val := v.(type) {
		case int:
			result = replaceFirst(result, "%d", intToString(val))
		case float64:
			result = replaceFirst(result, "%.1f", floatToString(val, 1))
			result = replaceFirst(result, "%.2f", floatToString(val, 2))
		}
	}
	return result
}

func replaceFirst(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	result := ""
	for n > 0 {
		digit := n % 10
		result = string(rune('0'+digit)) + result
		n /= 10
	}

	if negative {
		result = "-" + result
	}

	return result
}

func floatToString(f float64, decimals int) string {
	// Simple implementation
	negative := f < 0
	if negative {
		f = -f
	}

	intPart := int(f)
	fracPart := f - float64(intPart)

	result := intToString(intPart) + "."

	for i := 0; i < decimals; i++ {
		fracPart *= 10
		digit := int(fracPart)
		result += string(rune('0' + digit))
		fracPart -= float64(digit)
	}

	if negative {
		result = "-" + result
	}

	return result
}
