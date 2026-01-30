package overworld

// ThreatType represents different categories of threats
type ThreatType int

const (
	ThreatNecromancer ThreatType = iota
	ThreatBanditCamp
	ThreatCorruption
	ThreatBeastNest
	ThreatOrcWarband
)

// String returns human-readable threat name
func (t ThreatType) String() string {
	switch t {
	case ThreatNecromancer:
		return "Necromancer"
	case ThreatBanditCamp:
		return "Bandit Camp"
	case ThreatCorruption:
		return "Corruption"
	case ThreatBeastNest:
		return "Beast Nest"
	case ThreatOrcWarband:
		return "Orc Warband"
	default:
		return "Unknown Threat"
	}
}

// FactionType represents different faction types
type FactionType int

const (
	FactionNecromancers FactionType = iota // Undead necromancer faction
	FactionBandits                         // Bandit faction
	FactionBeasts                          // Beast faction
	FactionOrcs                            // Orc faction
	FactionCultists                        // Corruption/cultist faction

)

// FactionIntent represents current strategic objective
type FactionIntent int

const (
	IntentExpand  FactionIntent = iota // Claim new territory
	IntentFortify                      // Increase strength, spawn threats
	IntentRaid                         // Attack player or rival faction
	IntentRetreat                      // Abandon weak positions
	IntentIdle                         // No action (weak factions)
)

// String returns human-readable intent name
func (i FactionIntent) String() string {
	switch i {
	case IntentExpand:
		return "Expand"
	case IntentFortify:
		return "Fortify"
	case IntentRaid:
		return "Raid"
	case IntentRetreat:
		return "Retreat"
	case IntentIdle:
		return "Idle"
	default:
		return "Unknown"
	}
}

// String returns human-readable faction type name
func (f FactionType) String() string {
	switch f {
	case FactionNecromancers:
		return "Necromancers"
	case FactionBandits:
		return "Bandits"
	case FactionOrcs:
		return "Orcs"
	case FactionBeasts:
		return "Beasts"
	case FactionCultists:
		return "Cultists"
	default:
		return "Unknown"
	}
}

// InfluenceEffect represents type of influence
type InfluenceEffect int

const (
	InfluenceSpawnBoost InfluenceEffect = iota
	InfluenceResourceDrain
	InfluenceTerrainCorruption
	InfluenceCombatDebuff
)

// Tuning parameters
const (
	// Threat growth rates
	DefaultGrowthRate       = 0.05 // 20 ticks to evolve
	ContainmentSlowdown     = 0.5  // 50% slower when player nearby
	MaxThreatIntensity      = 10   // Hard cap
	ChildNodeSpawnThreshold = 3    // Spawn child at tier 3, 6, 9
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

// ThreatTypeParams defines behavior per threat type
type ThreatTypeParams struct {
	BaseGrowthRate   float64
	BaseRadius       int
	PrimaryEffect    InfluenceEffect
	CanSpawnChildren bool
	MaxIntensity     int
}

// GetThreatTypeParams returns parameters for each threat type
func GetThreatTypeParams(threatType ThreatType) ThreatTypeParams {
	switch threatType {
	case ThreatNecromancer:
		return ThreatTypeParams{
			BaseGrowthRate:   0.05,
			BaseRadius:       3,
			PrimaryEffect:    InfluenceSpawnBoost,
			CanSpawnChildren: true,
			MaxIntensity:     10,
		}
	case ThreatBanditCamp:
		return ThreatTypeParams{
			BaseGrowthRate:   0.08,
			BaseRadius:       2,
			PrimaryEffect:    InfluenceResourceDrain,
			CanSpawnChildren: false,
			MaxIntensity:     7,
		}
	case ThreatCorruption:
		return ThreatTypeParams{
			BaseGrowthRate:   0.03,
			BaseRadius:       5,
			PrimaryEffect:    InfluenceTerrainCorruption,
			CanSpawnChildren: true,
			MaxIntensity:     10,
		}
	case ThreatBeastNest:
		return ThreatTypeParams{
			BaseGrowthRate:   0.06,
			BaseRadius:       2,
			PrimaryEffect:    InfluenceSpawnBoost,
			CanSpawnChildren: false,
			MaxIntensity:     6,
		}
	case ThreatOrcWarband:
		return ThreatTypeParams{
			BaseGrowthRate:   0.07,
			BaseRadius:       3,
			PrimaryEffect:    InfluenceCombatDebuff,
			CanSpawnChildren: false,
			MaxIntensity:     8,
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
