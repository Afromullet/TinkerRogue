package overworld

import (
	"game_main/common"
)

// VictoryCondition represents win/loss state
type VictoryCondition int

const (
	VictoryNone          VictoryCondition = iota // Game in progress
	VictoryPlayerWins                            // Player eliminated all threats/factions
	VictoryPlayerLoses                           // Player overwhelmed
	VictoryTimeLimit                             // Survived N ticks
	VictoryFactionDefeat                         // Defeated specific faction
)

// DefeatReasonType categorizes why the player was defeated
type DefeatReasonType int

const (
	DefeatNone                  DefeatReasonType = iota // Not defeated
	DefeatByInfluence                                   // Overwhelmed by threat influence
	DefeatByHighIntensityThreats                        // Too many powerful threats
	DefeatBySquadLoss                                   // All squads destroyed
)

func (v VictoryCondition) String() string {
	switch v {
	case VictoryNone:
		return "In Progress"
	case VictoryPlayerWins:
		return "Victory!"
	case VictoryPlayerLoses:
		return "Defeat"
	case VictoryTimeLimit:
		return "Survival Victory"
	case VictoryFactionDefeat:
		return "Faction Defeated"
	default:
		return "Unknown"
	}
}

// VictoryStateData tracks victory condition progress
type VictoryStateData struct {
	Condition         VictoryCondition
	TicksToSurvive    int64 // For survival victory
	TargetFactionType FactionType
	VictoryAchieved   bool
	DefeatReason      string
}

// DefeatCheckResult contains the result of checking defeat conditions.
// Single source of truth for defeat determination - avoids duplicate checks.
type DefeatCheckResult struct {
	IsDefeated         bool
	DefeatReason       DefeatReasonType
	DefeatMessage      string
	TotalInfluence     float64 // Cached value to avoid recalculation
	HighIntensityCount int     // Cached value to avoid recalculation
}

// SquadChecker is an interface for checking squad status without circular dependency
// This interface allows the overworld package to query squad status without importing the squads package
type SquadChecker interface {
	// HasActiveSquads returns true if the player has any squads with living units
	HasActiveSquads(manager *common.EntityManager) bool
}

// squadChecker is the injected implementation for squad checking
// Set this in main package initialization via SetSquadChecker()
var squadChecker SquadChecker

// SetSquadChecker injects the squad checking implementation
// Call this from main package after squads package is initialized
func SetSquadChecker(checker SquadChecker) {
	squadChecker = checker
}
