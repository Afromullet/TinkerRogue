package core

// ThreatType represents different categories of threats.
// Now string-based for flexibility - can add new types via JSON without code changes.
type ThreatType string

// Threat type constants (optional - for IDE autocomplete and type safety)
// The actual types are defined in nodeDefinitions.json
const (
	ThreatNecromancer ThreatType = "necromancer"
	ThreatBanditCamp  ThreatType = "banditcamp"
	ThreatCorruption  ThreatType = "corruption"
	ThreatBeastNest   ThreatType = "beastnest"
	ThreatOrcWarband  ThreatType = "orcwarband"
)

// String returns human-readable threat name.
// Uses NodeRegistry for data-driven lookup.
func (t ThreatType) String() string {
	return GetNodeRegistry().GetDisplayName(t)
}

// EncounterTypeID returns the JSON encounter type ID for this threat.
// Uses NodeRegistry for data-driven lookup.
// These IDs match the "encounter.typeId" field in assets/gamedata/encounterdata.json.
func (t ThreatType) EncounterTypeID() string {
	return GetNodeRegistry().GetEncounterTypeID(t)
}

// NodeTypeID identifies a placeable node type from nodeDefinitions.json
type NodeTypeID string

const (
	NodeTypeTown       NodeTypeID = "town"
	NodeTypeGuildHall  NodeTypeID = "guild_hall"
	NodeTypeTemple     NodeTypeID = "temple"
	NodeTypeWatchtower NodeTypeID = "watchtower"
)

// String returns the display name for this node type.
func (n NodeTypeID) String() string {
	node := GetNodeRegistry().GetNodeByID(string(n))
	if node != nil {
		return node.DisplayName
	}
	return string(n)
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

// --- Owner Constants ---

const (
	OwnerPlayer  = "player"
	OwnerNeutral = "Neutral"
)

// IsHostileOwner returns true if the owner is neither player nor neutral.
func IsHostileOwner(ownerID string) bool {
	return ownerID != OwnerPlayer && ownerID != OwnerNeutral
}

// IsFriendlyOwner returns true if the owner is the player.
func IsFriendlyOwner(ownerID string) bool {
	return ownerID == OwnerPlayer
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

// VictoryStateData tracks victory condition progress
type VictoryStateData struct {
	Condition         VictoryCondition
	TicksToSurvive    int64 // For survival victory
	TargetFactionType FactionType
	VictoryAchieved   bool
	DefeatReason      string
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
	EventCombatResolved                    // Combat outcome applied
	EventPlayerNodePlaced                  // Player placed a node
	EventInfluenceSynergy                  // Synergy cluster formed
	EventInfluenceCompetition              // Faction rivalry detected
	EventInfluenceSuppression              // Player node suppressing threat
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
	case EventPlayerNodePlaced:
		return "Player Node Placed"
	case EventInfluenceSynergy:
		return "Influence Synergy"
	case EventInfluenceCompetition:
		return "Influence Competition"
	case EventInfluenceSuppression:
		return "Influence Suppression"
	default:
		return "Unknown Event"
	}
}
