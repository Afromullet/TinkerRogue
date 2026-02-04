package core

// ThreatTypeParams defines behavior per threat type.
// Note: MaxIntensity has been removed - use GetMaxThreatIntensity() for the global cap.
type ThreatTypeParams struct {
	BaseGrowthRate   float64
	BaseRadius       int
	PrimaryEffect    InfluenceEffect
	CanSpawnChildren bool
}

const (
	// Map dimensions (TODO: Should come from worldmap config)
	DefaultMapWidth  = 100 // Default overworld map width
	DefaultMapHeight = 80  // Default overworld map height
)

// DEPRECATED: GetThreatTypeParams has been removed.
// All threat parameters are now loaded from encounterdata.json via ThreatRegistry.
// Use GetThreatTypeParamsFromConfig() or GetThreatRegistry().GetOverworldParams() instead.
