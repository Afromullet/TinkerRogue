package overworld

import "fmt"

// formatEventString is a helper for formatting event descriptions
// Centralizes string formatting for event logging
func formatEventString(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// MapFactionToThreatType converts faction type to corresponding threat type
// Used when factions spawn new threat nodes
func MapFactionToThreatType(factionType FactionType) ThreatType {
	switch factionType {
	case FactionNecromancers:
		return ThreatNecromancer
	case FactionBandits:
		return ThreatBanditCamp
	case FactionOrcs:
		return ThreatOrcWarband
	case FactionBeasts:
		return ThreatBeastNest
	case FactionCultists:
		return ThreatCorruption
	default:
		return ThreatBanditCamp
	}
}

// getThreatTypeName converts threat type enum to display string
// Used for event logging and UI display
func getThreatTypeName(threatType ThreatType) string {
	switch threatType {
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
