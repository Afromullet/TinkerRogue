package core

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
	DefeatNone                   DefeatReasonType = iota // Not defeated
	DefeatByInfluence                                    // Overwhelmed by threat influence
	DefeatByHighIntensityThreats                         // Too many powerful threats
	DefeatBySquadLoss                                    // All squads destroyed
)

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

// EventType categorizes overworld events
type EventType int

const (
	EventThreatSpawned   EventType = iota // New threat appeared
	EventThreatEvolved                    // Threat gained intensity
	EventThreatDestroyed                  // Threat eliminated
	EventFactionExpanded                  // Faction claimed territory
	EventFactionRaid                      // Faction launched raid
	EventFactionDefeated                  // Faction eliminated
	EventVictory                          // Player won
	EventDefeat                           // Player lost
	EventCombatResolved                   // Combat outcome applied
)

func (e EventType) String() string {
	switch e {
	case EventThreatSpawned:
		return "Threat Spawned"
	case EventThreatEvolved:
		return "Threat Evolved"
	case EventThreatDestroyed:
		return "Threat Destroyed"
	case EventFactionExpanded:
		return "Faction Expanded"
	case EventFactionRaid:
		return "Faction Raid"
	case EventFactionDefeated:
		return "Faction Defeated"
	case EventVictory:
		return "Victory"
	case EventDefeat:
		return "Defeat"
	case EventCombatResolved:
		return "Combat Resolved"
	default:
		return "Unknown Event"
	}
}
