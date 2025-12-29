package combatsim

import "github.com/bytearena/ecs"

// =============================================================================
// PER-UNIT PERFORMANCE TRACKING
// =============================================================================

// UnitPerformanceData tracks individual unit combat performance for a single simulation
type UnitPerformanceData struct {
	UnitID       ecs.EntityID
	TemplateName string
	Role         string // Tank, DPS, Support
	GridRow      int
	GridCol      int

	// Damage metrics
	DamageDealt    int
	DamageReceived int
	DamageBlocked  int // Via resistance/cover

	// Combat actions
	AttacksAttempted int
	AttacksHit       int
	AttacksCrit      int
	AttacksMissed    int
	AttacksDodged    int // Attacks against this unit that were dodged

	// Survivability
	StartingHP  int
	EndingHP    int
	TurnOfDeath int // -1 if survived

	// Cover provided to allies
	CoverInstancesProvided  int
	DamageReductionProvided float64
}

// UnitPerformanceAggregated aggregates unit stats across multiple simulations
type UnitPerformanceAggregated struct {
	TemplateName string
	Role         string
	SampleCount  int

	// Aggregated damage
	TotalDamageDealt    int64
	TotalDamageReceived int64
	TotalDamageBlocked  int64

	// Aggregated combat actions
	TotalAttacksAttempted int
	TotalAttacksHit       int
	TotalAttacksCrit      int
	TotalAttacksMissed    int
	TotalAttacksDodged    int

	// Survivability aggregated
	SurvivalCount int
	DeathCount    int

	// For variance calculations
	DamageDealtValues    []int
	DamageReceivedValues []int
	DeathTurns           []int // Turns when deaths occurred

	// Cover aggregated
	TotalCoverInstances   int
	TotalCoverReduction   float64
}

// =============================================================================
// TIMELINE STRUCTURES
// =============================================================================

// RoundSnapshot captures combat state at a specific round
type RoundSnapshot struct {
	RoundNumber int

	// Squad states
	AttackerUnitsAlive int
	AttackerTotalHP    int
	AttackerMaxHP      int
	DefenderUnitsAlive int
	DefenderTotalHP    int
	DefenderMaxHP      int

	// Round events
	DamageDealtThisRound  int
	DamageTakenThisRound  int
	UnitsKilledThisRound  int
	CritsThisRound        int
	DodgesThisRound       int

	// Momentum indicator (positive = attacker advantage)
	Momentum float64
}

// TimelineData stores full combat timeline for a single simulation
type TimelineData struct {
	Rounds          []RoundSnapshot
	FirstBloodRound int // Round when first unit died (0 if none)
	TurningPoint    int // Round when momentum shifted permanently
	CombatDuration  int // Total rounds
	Winner          string
}

// RoundStatistics holds aggregated data for a specific round number across simulations
type RoundStatistics struct {
	RoundNumber int
	SampleCount int // How many simulations reached this round

	// Averages
	AvgAttackerHP      float64
	AvgDefenderHP      float64
	AvgDamageDealt     float64
	AvgUnitsKilled     float64
	AvgMomentum        float64

	// For variance calculations
	AttackerHPValues []int
	DefenderHPValues []int
	DamageValues     []int
}

// TimelineAggregated aggregates timelines across simulations
type TimelineAggregated struct {
	RoundStats      []RoundStatistics
	AvgFirstBlood   float64
	AvgTurningPoint float64
	AvgDuration     float64

	// For variance
	FirstBloodValues   []int
	TurningPointValues []int
	DurationValues     []int
}

// =============================================================================
// STATISTICAL SUMMARY
// =============================================================================

// StatisticalSummary provides confidence intervals and significance
type StatisticalSummary struct {
	Mean       float64
	Median     float64
	StdDev     float64
	Variance   float64
	SampleSize int

	// Confidence intervals
	CI95Low  float64
	CI95High float64
	CI99Low  float64
	CI99High float64

	// Margins
	MarginOfError95       float64
	RecommendedSampleSize int // For desired precision
}

// SignificanceTest results for A/B comparisons
type SignificanceTest struct {
	GroupAName     string
	GroupBName     string
	GroupAMean     float64
	GroupBMean     float64
	MeanDifference float64
	TStatistic     float64
	PValue         float64
	IsSignificant  bool // At 95% confidence
	EffectSize     float64 // Cohen's d
}

// =============================================================================
// PARAMETER SWEEP
// =============================================================================

// SweepConfig defines a parameter sweep analysis
type SweepConfig struct {
	Name        string
	Description string

	// Target unit/squad to modify
	TargetSquad string // "Attacker" or "Defender"
	TargetUnit  string // Unit template name, or "*" for all

	// Attribute to sweep
	Attribute string // Strength, Dexterity, Armor, Weapon, etc.
	BaseValue int

	// Range
	MinValue int
	MaxValue int
	StepSize int

	// Iterations per step
	IterationsPerStep int
}

// SweepResult stores sweep analysis results
type SweepResult struct {
	Config       SweepConfig
	TestedValues []int
	WinRates     []float64
	AvgDamage    []float64
	AvgSurvival  []float64

	// Derived analysis
	BreakPoints  []BreakPoint
	OptimalValue int // Closest to 50% win rate

	// Scaling curve analysis
	LinearRegionEnd  int     // Where linear scaling ends
	DiminishingStart int     // Where diminishing returns begin
	MarginalAtBase   float64 // Win% per attribute point at base
	MarginalAtCap    float64 // Win% per attribute point at cap
}

// BreakPoint identifies where a metric threshold is crossed
type BreakPoint struct {
	AttributeValue int
	MetricName     string  // WinRate, Survival, etc.
	CrossedValue   float64 // The threshold crossed
	Direction      string  // "above" or "below"
}

// AttributeComparisonResult compares value of different attributes
type AttributeComparisonResult struct {
	Attribute1 string
	Attribute2 string

	// Exchange rate: how many points of Attr2 equal 1 point of Attr1
	ExchangeRate    float64
	ConfidenceRange [2]float64 // 95% CI

	// Context
	BaselineWinRate float64
	TestScenario    string
}

// =============================================================================
// ROLE AND MECHANICS ANALYSIS
// =============================================================================

// RoleEffectivenessData quantifies role contributions
type RoleEffectivenessData struct {
	Role string

	// Damage contribution
	AvgDamageDealt float64
	DamageShare    float64 // Percentage of squad's total damage

	// Survival contribution
	AvgSurvivalRate float64
	AvgDeathTurn    float64

	// Tactical value
	CoverProvided  float64
	TargetingValue float64 // How often targeted vs expected

	// Sample size
	UnitCount int
}

// MechanicImpactData quantifies combat mechanic effects
type MechanicImpactData struct {
	MechanicName string // Cover, Crit, Dodge, Range, AttackType

	// Usage frequency
	ActivationRate float64
	ActivationCount int
	TotalOpportunities int

	// Impact when activated
	AvgDamageReduction float64 // For defensive mechanics
	AvgDamageIncrease  float64 // For offensive mechanics

	// Win correlation
	WinRateWithMechanic    float64
	WinRateWithoutMechanic float64
}

// MechanicsAnalysis groups all mechanic impact data
type MechanicsAnalysis struct {
	Cover    *MechanicImpactData
	Crit     *MechanicImpactData
	Dodge    *MechanicImpactData
	Range    map[int]*MechanicImpactData // By distance
	AttackType map[string]*MechanicImpactData // By attack type
}

// =============================================================================
// BALANCE REPORT
// =============================================================================

// BalanceInsight represents an actionable balance recommendation
type BalanceInsight struct {
	Severity   string // "Critical", "Warning", "Info"
	Category   string // "Attribute", "Mechanic", "Composition", "Duration"
	Issue      string // What the problem is
	Evidence   string // Data supporting the finding
	Suggestion string // Recommended fix
	Impact     string // Expected effect of fix
}

// WinRateSection contains win rate analysis
type WinRateSection struct {
	AttackerWinRate float64
	DefenderWinRate float64
	DrawRate        float64

	AttackerCI95 [2]float64
	DefenderCI95 [2]float64

	IsBalanced      bool    // Within 45-55% range
	ImbalanceAmount float64 // How far from 50%
	FavoredSide     string  // "Attacker", "Defender", or "Neither"
}

// DurationSection contains combat duration analysis
type DurationSection struct {
	Average float64
	Median  float64
	StdDev  float64
	Min     int
	Max     int

	DistributionBuckets map[string]int // "1-3", "4-6", etc.
	IsHealthyDuration   bool           // Between 3-8 rounds typical
}

// UnitReportData contains per-unit report data
type UnitReportData struct {
	TemplateName   string
	Role           string
	AvgDamageDealt float64
	AvgDamageTaken float64
	SurvivalRate   float64
	Efficiency     float64 // Damage dealt / damage taken
	AvgDeathTurn   float64
	CoverProvided  float64
}

// UnitPerformanceSection contains all unit performance data
type UnitPerformanceSection struct {
	ByUnit          map[string]*UnitReportData
	ByRole          map[string]*RoleEffectivenessData
	TopPerformers   []string // Unit names with highest efficiency
	UnderPerformers []string // Unit names with lowest efficiency
}

// TimelineSection contains timeline analysis
type TimelineSection struct {
	AvgFirstBlood   float64
	AvgTurningPoint float64
	AvgDuration     float64
	SnowballFactor  float64 // How much early leads compound
	DamageCurve     []float64 // Avg damage per round
}

// MechanicsSection contains mechanics analysis summary
type MechanicsSection struct {
	CoverEffectiveness float64 // Avg damage reduction when cover applies
	CoverActivation    float64 // How often cover is used
	CritRate           float64
	CritImpact         float64 // Avg damage increase from crits
	DodgeRate          float64
	DodgeImpact        float64 // Avg damage avoided from dodges
	RangeAdvantage     float64 // Win rate difference for ranged units
}

// ConfidenceSection contains statistical confidence information
type ConfidenceSection struct {
	SampleSize           int
	MarginOfError        float64
	IsStatisticallySound bool
	RecommendedSamples   int
}

// BalanceReport is the main output structure for balance decisions
type BalanceReport struct {
	ScenarioName  string
	Iterations    int
	AnalysisMode  string // "quick", "standard", "comprehensive"

	// Core results
	WinRate  WinRateSection
	Duration DurationSection

	// Detailed analysis (standard+)
	UnitPerformance *UnitPerformanceSection
	Mechanics       *MechanicsSection

	// Comprehensive analysis
	Timeline   *TimelineSection
	Confidence *ConfidenceSection

	// Recommendations
	Insights []BalanceInsight
}
