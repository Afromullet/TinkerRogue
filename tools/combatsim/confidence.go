package combatsim

import (
	"math"
	"sort"
)

// =============================================================================
// CORE STATISTICS
// =============================================================================

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

// CalculateStdDevInt computes sample standard deviation for integers
func CalculateStdDevInt(values []int, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	variance := CalculateVarianceInt(values, mean)
	return math.Sqrt(variance)
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
// HELPER FUNCTIONS
// =============================================================================

// IsStatisticallySound checks if sample size is adequate
func IsStatisticallySound(marginOfError float64) bool {
	// Consider margin of error < 3% as statistically sound
	return marginOfError <= 0.03
}
