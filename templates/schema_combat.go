package templates

// schema_combat.go holds JSON DTOs for combat-AI tuning configs:
//   - AI behavior weights (threat layers, role behaviors) from aiconfig.json
//   - Power evaluation profiles, role multipliers, composition bonuses from powerconfig.json

// --- AI config (aiconfig.json) ---

// JSONAIConfig is the root container for AI behavior configuration.
type JSONAIConfig struct {
	ThreatCalculation      ThreatCalculationConfig `json:"threatCalculation"`
	RoleBehaviors          []RoleBehaviorConfig    `json:"roleBehaviors"`
	SupportLayer           SupportLayerConfig      `json:"supportLayer"`
	SharedRangedWeight     float64                 `json:"sharedRangedWeight"`     // Shared ranged threat weight (all roles)
	SharedPositionalWeight float64                 `json:"sharedPositionalWeight"` // Shared positional awareness weight (all roles)
}

// ThreatCalculationConfig defines threat calculation parameters.
type ThreatCalculationConfig struct {
	FlankingThreatRangeBonus   int `json:"flankingThreatRangeBonus"`
	IsolationThreshold         int `json:"isolationThreshold"`
	RetreatSafeThreatThreshold int `json:"retreatSafeThreatThreshold"`
}

// RoleBehaviorConfig defines how a role weighs different threat layers.
// RangedWeight and PositionalWeight are hardcoded as shared constants.
type RoleBehaviorConfig struct {
	Role          string  `json:"role"`
	MeleeWeight   float64 `json:"meleeWeight"`
	SupportWeight float64 `json:"supportWeight"`
}

// SupportLayerConfig defines support layer parameters.
type SupportLayerConfig struct {
	HealRadius int `json:"healRadius"`
}

// --- Power config (powerconfig.json) ---

// JSONPowerConfig is the root container for power evaluation configuration.
type JSONPowerConfig struct {
	Profiles           []PowerProfileConfig     `json:"profiles"`
	RoleMultipliers    []RoleMultiplierConfig   `json:"roleMultipliers"`
	CompositionBonuses []CompositionBonusConfig `json:"compositionBonuses"`
	LeaderBonus        float64                  `json:"leaderBonus"`
}

// PowerProfileConfig defines a power calculation profile.
// Only top-level category weights are configurable; sub-calculations use fixed formulas.
type PowerProfileConfig struct {
	Name            string  `json:"name"`
	OffensiveWeight float64 `json:"offensiveWeight"` // Weight for offensive stats (damage output)
	DefensiveWeight float64 `json:"defensiveWeight"` // Weight for defensive stats (survivability)
	UtilityWeight   float64 `json:"utilityWeight"`   // Weight for utility (role, abilities, cover)
	HealthPenalty   float64 `json:"healthPenalty"`   // Exponent for health-based power scaling
}

// RoleMultiplierConfig defines role multiplier value.
type RoleMultiplierConfig struct {
	Role       string  `json:"role"`
	Multiplier float64 `json:"multiplier"`
}

// CompositionBonusConfig defines composition bonus.
type CompositionBonusConfig struct {
	UniqueTypes int     `json:"uniqueTypes"`
	Bonus       float64 `json:"bonus"`
}
