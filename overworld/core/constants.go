package core

// ThreatTypeParams defines behavior per threat type
type ThreatTypeParams struct {
	BaseGrowthRate   float64
	BaseRadius       int
	PrimaryEffect    InfluenceEffect
	CanSpawnChildren bool
	MaxIntensity     int
}

const (

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
