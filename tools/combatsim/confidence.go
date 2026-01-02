package combatsim

import (
	"math"
	"sort"
)

// =============================================================================
// CORE STATISTICS
// =============================================================================

// CalculateMean computes arithmetic mean of values
func CalculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// CalculateMeanInt computes arithmetic mean of integer values
func CalculateMeanInt(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

// CalculateStdDev computes sample standard deviation
func CalculateStdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	variance := CalculateVariance(values, mean)
	return math.Sqrt(variance)
}

// CalculateStdDevInt computes sample standard deviation for integers
func CalculateStdDevInt(values []int, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	variance := CalculateVarianceInt(values, mean)
	return math.Sqrt(variance)
}

// CalculateVariance computes sample variance
func CalculateVariance(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	// Sample variance uses n-1 (Bessel's correction)
	return sumSquares / float64(len(values)-1)
}

// CalculateVarianceInt computes sample variance for integers
func CalculateVarianceInt(values []int, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := float64(v) - mean
		sumSquares += diff * diff
	}
	return sumSquares / float64(len(values)-1)
}

// CalculateMedian computes median value
func CalculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Copy to avoid modifying original
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// CalculateMedianInt computes median of integer values
func CalculateMedianInt(values []int) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return float64(sorted[n/2-1]+sorted[n/2]) / 2
	}
	return float64(sorted[n/2])
}

// =============================================================================
// CONFIDENCE INTERVALS
// =============================================================================

// Z-scores for common confidence levels
const (
	ZScore90 = 1.645
	ZScore95 = 1.960
	ZScore99 = 2.576
)

// T-distribution critical values for small samples (approximate)
// Key: degrees of freedom -> t-value at 95% confidence (two-tailed)
var tTable95 = map[int]float64{
	1:  12.706,
	2:  4.303,
	3:  3.182,
	4:  2.776,
	5:  2.571,
	6:  2.447,
	7:  2.365,
	8:  2.306,
	9:  2.262,
	10: 2.228,
	15: 2.131,
	20: 2.086,
	25: 2.060,
	30: 2.042,
}

// getTValue returns the t-value for given degrees of freedom at 95% confidence
func getTValue(df int) float64 {
	if df >= 30 {
		return ZScore95 // Use z-score for large samples
	}
	if t, ok := tTable95[df]; ok {
		return t
	}
	// Interpolate for missing values
	for i := df; i >= 1; i-- {
		if t, ok := tTable95[i]; ok {
			return t
		}
	}
	return ZScore95
}

// CalculateCI95 computes 95% confidence interval
// Uses t-distribution for n < 30, z-score otherwise
func CalculateCI95(mean, stdDev float64, n int) (low, high float64) {
	if n <= 1 {
		return mean, mean
	}

	standardError := stdDev / math.Sqrt(float64(n))

	var multiplier float64
	if n < 30 {
		multiplier = getTValue(n - 1) // degrees of freedom = n - 1
	} else {
		multiplier = ZScore95
	}

	margin := multiplier * standardError
	return mean - margin, mean + margin
}

// CalculateCI99 computes 99% confidence interval
func CalculateCI99(mean, stdDev float64, n int) (low, high float64) {
	if n <= 1 {
		return mean, mean
	}

	standardError := stdDev / math.Sqrt(float64(n))
	margin := ZScore99 * standardError

	return mean - margin, mean + margin
}

// CalculateMarginOfError computes margin of error for given confidence
func CalculateMarginOfError(stdDev float64, n int, zScore float64) float64 {
	if n <= 0 {
		return 0
	}
	standardError := stdDev / math.Sqrt(float64(n))
	return zScore * standardError
}

// RecommendSampleSize calculates needed samples for desired precision
// desiredMargin is the desired margin of error (e.g., 0.02 for +/- 2%)
// confidence is the z-score (use ZScore95 for 95% confidence)
func RecommendSampleSize(currentStdDev, desiredMargin, confidence float64) int {
	if desiredMargin <= 0 {
		return 1000 // Default recommendation
	}

	// Formula: n = (z * stdDev / margin)^2
	n := math.Pow(confidence*currentStdDev/desiredMargin, 2)
	return int(math.Ceil(n))
}

// =============================================================================
// PROPORTION CONFIDENCE (for win rates)
// =============================================================================

// CalculateProportionCI95 calculates 95% CI for a proportion (like win rate)
// Uses Wilson score interval for better accuracy with proportions
func CalculateProportionCI95(successes, total int) (low, high float64) {
	if total == 0 {
		return 0, 0
	}

	p := float64(successes) / float64(total)
	n := float64(total)
	z := ZScore95

	// Wilson score interval
	denominator := 1 + z*z/n
	centre := p + z*z/(2*n)
	adjustment := z * math.Sqrt((p*(1-p)+z*z/(4*n))/n)

	low = (centre - adjustment) / denominator
	high = (centre + adjustment) / denominator

	// Clamp to [0, 1]
	if low < 0 {
		low = 0
	}
	if high > 1 {
		high = 1
	}

	return low, high
}

// CalculateProportionMarginOfError calculates margin of error for a proportion
func CalculateProportionMarginOfError(p float64, n int, zScore float64) float64 {
	if n <= 0 || p < 0 || p > 1 {
		return 0
	}
	// Standard error of proportion
	se := math.Sqrt(p * (1 - p) / float64(n))
	return zScore * se
}

// =============================================================================
// SIGNIFICANCE TESTING
// =============================================================================

// PerformTTest performs independent samples t-test
func PerformTTest(groupA, groupB []float64) *SignificanceTest {
	result := &SignificanceTest{}

	nA := len(groupA)
	nB := len(groupB)

	if nA < 2 || nB < 2 {
		return result
	}

	meanA := CalculateMean(groupA)
	meanB := CalculateMean(groupB)
	varA := CalculateVariance(groupA, meanA)
	varB := CalculateVariance(groupB, meanB)

	result.GroupAMean = meanA
	result.GroupBMean = meanB
	result.MeanDifference = meanA - meanB

	// Pooled standard error (assuming equal variances)
	pooledVar := ((float64(nA-1)*varA + float64(nB-1)*varB) / float64(nA+nB-2))
	pooledSE := math.Sqrt(pooledVar * (1.0/float64(nA) + 1.0/float64(nB)))

	if pooledSE == 0 {
		return result
	}

	// T-statistic
	result.TStatistic = result.MeanDifference / pooledSE

	// Degrees of freedom
	df := nA + nB - 2

	// Calculate p-value (two-tailed) using approximation
	result.PValue = calculatePValue(result.TStatistic, df)

	// Significance at 95%
	result.IsSignificant = result.PValue < 0.05

	// Effect size (Cohen's d)
	pooledStdDev := math.Sqrt(pooledVar)
	if pooledStdDev > 0 {
		result.EffectSize = math.Abs(result.MeanDifference) / pooledStdDev
	}

	return result
}

// calculatePValue approximates p-value from t-statistic
// Uses normal approximation for simplicity (accurate for large df)
func calculatePValue(tStat float64, df int) float64 {
	// For large df, t-distribution approaches normal
	// This is an approximation
	abst := math.Abs(tStat)

	// Use error function approximation for normal CDF
	// P(|Z| > z) = 2 * (1 - Phi(z))
	z := abst
	if df < 30 {
		// Rough adjustment for small samples
		z = abst * math.Sqrt(float64(df)/float64(df+3))
	}

	// Normal CDF approximation using error function
	cdf := 0.5 * (1 + erf(z/math.Sqrt2))

	// Two-tailed p-value
	return 2 * (1 - cdf)
}

// erf approximates the error function
func erf(x float64) float64 {
	// Horner form coefficients for approximation
	a1 := 0.254829592
	a2 := -0.284496736
	a3 := 1.421413741
	a4 := -1.453152027
	a5 := 1.061405429
	p := 0.3275911

	sign := 1.0
	if x < 0 {
		sign = -1.0
	}
	x = math.Abs(x)

	t := 1.0 / (1.0 + p*x)
	y := 1.0 - (((((a5*t+a4)*t)+a3)*t+a2)*t+a1)*t*math.Exp(-x*x)

	return sign * y
}

// CalculateEffectSize computes Cohen's d
func CalculateEffectSize(meanA, meanB, pooledStdDev float64) float64 {
	if pooledStdDev == 0 {
		return 0
	}
	return math.Abs(meanA-meanB) / pooledStdDev
}

// InterpretEffectSize returns interpretation of Cohen's d
func InterpretEffectSize(d float64) string {
	d = math.Abs(d)
	switch {
	case d < 0.2:
		return "negligible"
	case d < 0.5:
		return "small"
	case d < 0.8:
		return "medium"
	default:
		return "large"
	}
}

// =============================================================================
// SUMMARY GENERATION
// =============================================================================

// GenerateStatisticalSummary creates full statistical summary for a dataset
func GenerateStatisticalSummary(values []float64) *StatisticalSummary {
	summary := &StatisticalSummary{
		SampleSize: len(values),
	}

	if len(values) == 0 {
		return summary
	}

	summary.Mean = CalculateMean(values)
	summary.Median = CalculateMedian(values)
	summary.Variance = CalculateVariance(values, summary.Mean)
	summary.StdDev = math.Sqrt(summary.Variance)

	summary.CI95Low, summary.CI95High = CalculateCI95(summary.Mean, summary.StdDev, summary.SampleSize)
	summary.CI99Low, summary.CI99High = CalculateCI99(summary.Mean, summary.StdDev, summary.SampleSize)

	summary.MarginOfError95 = CalculateMarginOfError(summary.StdDev, summary.SampleSize, ZScore95)

	// Recommend sample size for +/- 1% margin at 95% confidence
	summary.RecommendedSampleSize = RecommendSampleSize(summary.StdDev, 0.01, ZScore95)

	return summary
}

// GenerateStatisticalSummaryInt creates summary for integer values
func GenerateStatisticalSummaryInt(values []int) *StatisticalSummary {
	floats := make([]float64, len(values))
	for i, v := range values {
		floats[i] = float64(v)
	}
	return GenerateStatisticalSummary(floats)
}

// SummarizeWinRate creates summary specifically for win rate data
func SummarizeWinRate(wins, total int) *StatisticalSummary {
	if total == 0 {
		return &StatisticalSummary{}
	}

	p := float64(wins) / float64(total)

	summary := &StatisticalSummary{
		Mean:       p,
		SampleSize: total,
	}

	// Standard deviation for binomial proportion
	summary.StdDev = math.Sqrt(p * (1 - p))
	summary.Variance = summary.StdDev * summary.StdDev

	// Wilson score interval for proportions
	summary.CI95Low, summary.CI95High = CalculateProportionCI95(wins, total)

	// Margin of error
	summary.MarginOfError95 = CalculateProportionMarginOfError(p, total, ZScore95)

	// Recommend samples for +/- 1.5% margin
	if summary.StdDev > 0 {
		summary.RecommendedSampleSize = RecommendSampleSize(summary.StdDev, 0.015, ZScore95)
	}

	return summary
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// IsStatisticallySound checks if sample size is adequate
func IsStatisticallySound(marginOfError float64) bool {
	// Consider margin of error < 3% as statistically sound
	return marginOfError <= 0.03
}

// CalculateCorrelation computes Pearson correlation coefficient between two datasets
func CalculateCorrelation(x, y []float64) float64 {
	n := len(x)
	if n != len(y) || n < 2 {
		return 0
	}

	meanX := CalculateMean(x)
	meanY := CalculateMean(y)

	var sumXY, sumX2, sumY2 float64
	for i := 0; i < n; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		sumXY += dx * dy
		sumX2 += dx * dx
		sumY2 += dy * dy
	}

	denominator := math.Sqrt(sumX2 * sumY2)
	if denominator == 0 {
		return 0
	}

	return sumXY / denominator
}
