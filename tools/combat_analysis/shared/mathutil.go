package shared

// SafeDiv returns num/denom, or 0.0 when denom is zero.
func SafeDiv(num, denom float64) float64 {
	if denom == 0 {
		return 0.0
	}
	return num / denom
}
