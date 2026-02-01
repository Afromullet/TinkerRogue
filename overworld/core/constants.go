package core

// ThreatTypeParams defines behavior per threat type
type ThreatTypeParams struct {
	BaseGrowthRate   float64
	BaseRadius       int
	PrimaryEffect    InfluenceEffect
	CanSpawnChildren bool
	MaxIntensity     int
}

// DEPRECATED: These constants are now loaded from overworldconfig.json.
// Use getter functions from config.go instead (e.g., GetThreatGrowthRate()).
// Kept for fallback purposes only.
const (
	// Threat growth rates
	DefaultGrowthRate       = 0.05 // 20 ticks to evolve
	ContainmentSlowdown     = 0.5  // 50% slower when player nearby
	MaxThreatIntensity      = 5    // Hard cap (matches difficulty levels 1-5)
	ChildNodeSpawnThreshold = 3    // Spawn child at tier 3
	PlayerContainmentRadius = 5    // Player presence slows threats in radius

	// Threat spawn attempt limits
	MaxChildNodeSpawnAttempts = 10 // Maximum tries to find valid spawn position for child nodes

	// Faction AI parameters
	DefaultIntentTickDuration  = 10 // Re-evaluate every 10 ticks
	ExpansionStrengthThreshold = 5  // Minimum strength to expand
	ExpansionTerritoryLimit    = 20 // Max tiles before expansion slows
	FortificationWeakThreshold = 3  // Fortify if strength below this
	FortificationStrengthGain  = 1  // Strength gained per fortify action
	RaidStrengthThreshold      = 7  // Minimum strength to raid
	RaidProximityRange         = 5  // Max distance to raid target
	RetreatCriticalStrength    = 2  // Retreat if strength below this
	MaxTerritorySize           = 30 // Cap on faction territory

	// Faction spawn probabilities (percentage 0-100)
	ExpansionThreatSpawnChance = 20 // Chance to spawn threat when faction expands territory
	FortifyThreatSpawnChance   = 30 // Chance to spawn threat when faction fortifies

	// Item drop probabilities (percentage 0-100)
	BonusItemDropChance = 30 // Chance for bonus item drop from encounters

	// Map dimensions (TODO: Should come from worldmap config)
	DefaultMapWidth  = 100 // Default overworld map width
	DefaultMapHeight = 80  // Default overworld map height
)

// DEPRECATED: Use GetThreatTypeParamsFromConfig() from config.go instead.
// This function is kept for fallback purposes only.
// Threat type parameters are now loaded from overworldconfig.json for designer-friendly tuning.
func GetThreatTypeParams(threatType ThreatType) ThreatTypeParams {
	switch threatType {
	case ThreatNecromancer:
		return ThreatTypeParams{
			BaseGrowthRate:   0.05,
			BaseRadius:       3,
			PrimaryEffect:    InfluenceSpawnBoost,
			CanSpawnChildren: true,
			MaxIntensity:     5,
		}
	case ThreatBanditCamp:
		return ThreatTypeParams{
			BaseGrowthRate:   0.08,
			BaseRadius:       2,
			PrimaryEffect:    InfluenceResourceDrain,
			CanSpawnChildren: false,
			MaxIntensity:     5,
		}
	case ThreatCorruption:
		return ThreatTypeParams{
			BaseGrowthRate:   0.03,
			BaseRadius:       5,
			PrimaryEffect:    InfluenceTerrainCorruption,
			CanSpawnChildren: true,
			MaxIntensity:     5,
		}
	case ThreatBeastNest:
		return ThreatTypeParams{
			BaseGrowthRate:   0.06,
			BaseRadius:       2,
			PrimaryEffect:    InfluenceSpawnBoost,
			CanSpawnChildren: false,
			MaxIntensity:     5,
		}
	case ThreatOrcWarband:
		return ThreatTypeParams{
			BaseGrowthRate:   0.07,
			BaseRadius:       3,
			PrimaryEffect:    InfluenceCombatDebuff,
			CanSpawnChildren: false,
			MaxIntensity:     5,
		}
	default:
		return ThreatTypeParams{
			BaseGrowthRate:   0.05,
			BaseRadius:       2,
			PrimaryEffect:    InfluenceSpawnBoost,
			CanSpawnChildren: false,
			MaxIntensity:     5,
		}
	}
}
