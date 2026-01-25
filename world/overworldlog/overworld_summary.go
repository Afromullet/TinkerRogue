package overworldlog

import (
	"github.com/bytearena/ecs"
)

// ThreatActivitySummary aggregates threat-related events.
type ThreatActivitySummary struct {
	TotalSpawned      int                        `json:"total_spawned"`
	TotalDestroyed    int                        `json:"total_destroyed"`
	TotalEvolved      int                        `json:"total_evolved"`
	MaxIntensity      int                        `json:"max_intensity"`
	ThreatTypeStats   map[string]*ThreatTypeStats `json:"threat_type_stats"` // Stats per threat type
}

// ThreatTypeStats tracks statistics for a specific threat type.
type ThreatTypeStats struct {
	Spawned      int `json:"spawned"`
	Destroyed    int `json:"destroyed"`
	Evolved      int `json:"evolved"`
}

// FactionActivitySummary aggregates faction-related events.
type FactionActivitySummary struct {
	TotalExpansions     int                          `json:"total_expansions"`
	TotalRaids          int                          `json:"total_raids"`
	TotalDefeated       int                          `json:"total_defeated"`
	FactionStats        map[ecs.EntityID]*FactionStats `json:"faction_stats"` // Stats per faction
}

// FactionStats tracks statistics for a specific faction.
type FactionStats struct {
	FactionID      ecs.EntityID `json:"faction_id"`
	Expansions     int          `json:"expansions"`
	Raids          int          `json:"raids"`
	TerritoryGains int          `json:"territory_gains"`
	Defeated       bool         `json:"defeated"`
}

// CombatActivitySummary aggregates combat-related events.
type CombatActivitySummary struct {
	TotalCombats      int `json:"total_combats"`
	Victories         int `json:"victories"`
	Defeats           int `json:"defeats"`
	IntensityReduced  int `json:"intensity_reduced"`
}

// GenerateThreatSummary creates threat activity statistics from event records.
func GenerateThreatSummary(events []EventRecord) *ThreatActivitySummary {
	summary := &ThreatActivitySummary{
		ThreatTypeStats: make(map[string]*ThreatTypeStats),
	}

	for _, event := range events {
		switch event.Type {
		case "Threat Spawned":
			summary.TotalSpawned++

			// Track by type if available in data
			if threatType, ok := event.Data["threat_type"].(string); ok {
				if _, exists := summary.ThreatTypeStats[threatType]; !exists {
					summary.ThreatTypeStats[threatType] = &ThreatTypeStats{}
				}
				summary.ThreatTypeStats[threatType].Spawned++
			}

			// Track max intensity
			if intensity, ok := event.Data["intensity"].(int); ok {
				if intensity > summary.MaxIntensity {
					summary.MaxIntensity = intensity
				}
			}
			// Handle float64 from JSON unmarshaling
			if intensity, ok := event.Data["intensity"].(float64); ok {
				intensityInt := int(intensity)
				if intensityInt > summary.MaxIntensity {
					summary.MaxIntensity = intensityInt
				}
			}

		case "Threat Destroyed":
			summary.TotalDestroyed++

			if threatType, ok := event.Data["threat_type"].(string); ok {
				if _, exists := summary.ThreatTypeStats[threatType]; !exists {
					summary.ThreatTypeStats[threatType] = &ThreatTypeStats{}
				}
				summary.ThreatTypeStats[threatType].Destroyed++
			}

		case "Threat Evolved":
			summary.TotalEvolved++

			if threatType, ok := event.Data["threat_type"].(string); ok {
				if _, exists := summary.ThreatTypeStats[threatType]; !exists {
					summary.ThreatTypeStats[threatType] = &ThreatTypeStats{}
				}
				summary.ThreatTypeStats[threatType].Evolved++
			}

			// Track intensity from evolved threats
			if intensity, ok := event.Data["new_intensity"].(int); ok {
				if intensity > summary.MaxIntensity {
					summary.MaxIntensity = intensity
				}
			}
			if intensity, ok := event.Data["new_intensity"].(float64); ok {
				intensityInt := int(intensity)
				if intensityInt > summary.MaxIntensity {
					summary.MaxIntensity = intensityInt
				}
			}
		}
	}

	return summary
}

// GenerateFactionSummary creates faction activity statistics from event records.
func GenerateFactionSummary(events []EventRecord) *FactionActivitySummary {
	summary := &FactionActivitySummary{
		FactionStats: make(map[ecs.EntityID]*FactionStats),
	}

	for _, event := range events {
		// Ensure faction exists in stats map
		if event.EntityID != 0 {
			if _, exists := summary.FactionStats[event.EntityID]; !exists {
				summary.FactionStats[event.EntityID] = &FactionStats{
					FactionID: event.EntityID,
				}
			}
		}

		switch event.Type {
		case "Faction Expanded":
			summary.TotalExpansions++
			if event.EntityID != 0 {
				summary.FactionStats[event.EntityID].Expansions++

				// Track territory gains if available
				if tiles, ok := event.Data["tiles_gained"].(int); ok {
					summary.FactionStats[event.EntityID].TerritoryGains += tiles
				}
				if tiles, ok := event.Data["tiles_gained"].(float64); ok {
					summary.FactionStats[event.EntityID].TerritoryGains += int(tiles)
				}
			}

		case "Faction Raid":
			summary.TotalRaids++
			if event.EntityID != 0 {
				summary.FactionStats[event.EntityID].Raids++
			}

		case "Faction Defeated":
			summary.TotalDefeated++
			if event.EntityID != 0 {
				summary.FactionStats[event.EntityID].Defeated = true
			}
		}
	}

	return summary
}

// GenerateCombatSummary creates combat activity statistics from event records.
func GenerateCombatSummary(events []EventRecord) *CombatActivitySummary {
	summary := &CombatActivitySummary{}

	for _, event := range events {
		switch event.Type {
		case "Combat Resolved":
			summary.TotalCombats++

			// Track victory/defeat
			if victory, ok := event.Data["victory"].(bool); ok && victory {
				summary.Victories++
			} else if victory, ok := event.Data["victory"].(bool); ok && !victory {
				summary.Defeats++
			}

			// Track intensity reduced
			if intensityReduced, ok := event.Data["intensity_reduced"].(int); ok {
				summary.IntensityReduced += intensityReduced
			}
			if intensityReduced, ok := event.Data["intensity_reduced"].(float64); ok {
				summary.IntensityReduced += int(intensityReduced)
			}
		}
	}

	return summary
}
