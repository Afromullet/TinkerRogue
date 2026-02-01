// Package overworld provides the strategic layer for the overworld map.
//
// This file provides backward-compatible re-exports from the subpackages.
// New code should import the specific subpackages directly:
//   - world/overworld/core - Shared components, types, events, tick system
//   - world/overworld/threat - Threat node system
//   - world/overworld/faction - NPC faction AI
//   - world/overworld/victory - Win/loss conditions
//   - world/overworld/travel - Player movement
//   - world/overworld/encounter - Combat encounter generation
package overworld

import (
	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/encounter"

	"github.com/bytearena/ecs"
)

// =============================================================================
// Type Aliases - for backward compatibility
// =============================================================================

// Core types
type ThreatType = core.ThreatType
type FactionType = core.FactionType
type FactionIntent = core.FactionIntent
type InfluenceEffect = core.InfluenceEffect
type VictoryCondition = core.VictoryCondition
type DefeatReasonType = core.DefeatReasonType
type EventType = core.EventType
type ThreatTypeParams = core.ThreatTypeParams

// Component data types
type ThreatNodeData = core.ThreatNodeData
type OverworldFactionData = core.OverworldFactionData
type TickStateData = core.TickStateData
type InfluenceData = core.InfluenceData
type TerritoryData = core.TerritoryData
type StrategicIntentData = core.StrategicIntentData
type TravelStateData = core.TravelStateData
type VictoryStateData = core.VictoryStateData
type DefeatCheckResult = core.DefeatCheckResult
type OverworldEncounterData = core.OverworldEncounterData
type OverworldEvent = core.OverworldEvent
type EventLog = core.EventLog
type OverworldContext = core.OverworldContext
type SquadChecker = core.SquadChecker

// Encounter types
type EncounterParams = encounter.EncounterParams
type UnitTemplate = encounter.UnitTemplate
type RewardTable = encounter.RewardTable

// =============================================================================
// Constants - re-export from core
// =============================================================================

// Threat types
const (
	ThreatNecromancer = core.ThreatNecromancer
	ThreatBanditCamp  = core.ThreatBanditCamp
	ThreatCorruption  = core.ThreatCorruption
	ThreatBeastNest   = core.ThreatBeastNest
	ThreatOrcWarband  = core.ThreatOrcWarband
)

// Faction types
const (
	FactionNecromancers = core.FactionNecromancers
	FactionBandits      = core.FactionBandits
	FactionBeasts       = core.FactionBeasts
	FactionOrcs         = core.FactionOrcs
	FactionCultists     = core.FactionCultists
)

// Faction intents
const (
	IntentExpand  = core.IntentExpand
	IntentFortify = core.IntentFortify
	IntentRaid    = core.IntentRaid
	IntentRetreat = core.IntentRetreat
	IntentIdle    = core.IntentIdle
)

// Victory conditions
const (
	VictoryNone          = core.VictoryNone
	VictoryPlayerWins    = core.VictoryPlayerWins
	VictoryPlayerLoses   = core.VictoryPlayerLoses
	VictoryTimeLimit     = core.VictoryTimeLimit
	VictoryFactionDefeat = core.VictoryFactionDefeat
)

// Event types
const (
	EventThreatSpawned   = core.EventThreatSpawned
	EventThreatEvolved   = core.EventThreatEvolved
	EventThreatDestroyed = core.EventThreatDestroyed
	EventFactionExpanded = core.EventFactionExpanded
	EventFactionRaid     = core.EventFactionRaid
	EventFactionDefeated = core.EventFactionDefeated
	EventVictory         = core.EventVictory
	EventDefeat          = core.EventDefeat
	EventCombatResolved  = core.EventCombatResolved
)

// =============================================================================
// Component/Tag Variables - synced from core package
// =============================================================================

// Component variables - synced from core package after ECS initialization
// via the init() subsystem registration below.
var (
	ThreatNodeComponent         *ecs.Component
	OverworldFactionComponent   *ecs.Component
	TickStateComponent          *ecs.Component
	InfluenceComponent          *ecs.Component
	TerritoryComponent          *ecs.Component
	StrategicIntentComponent    *ecs.Component
	VictoryStateComponent       *ecs.Component
	TravelStateComponent        *ecs.Component
	OverworldEncounterComponent *ecs.Component

	ThreatNodeTag         ecs.Tag
	OverworldFactionTag   ecs.Tag
	TickStateTag          ecs.Tag
	VictoryStateTag       ecs.Tag
	TravelStateTag        ecs.Tag
	OverworldEncounterTag ecs.Tag
)

// init registers a subsystem that syncs our local variables from the core package.
// This runs after core.init() because Go initializes packages in dependency order.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		// Sync component pointers from core
		ThreatNodeComponent = core.ThreatNodeComponent
		OverworldFactionComponent = core.OverworldFactionComponent
		TickStateComponent = core.TickStateComponent
		InfluenceComponent = core.InfluenceComponent
		TerritoryComponent = core.TerritoryComponent
		StrategicIntentComponent = core.StrategicIntentComponent
		VictoryStateComponent = core.VictoryStateComponent
		TravelStateComponent = core.TravelStateComponent
		OverworldEncounterComponent = core.OverworldEncounterComponent

		// Sync tags from core
		ThreatNodeTag = core.ThreatNodeTag
		OverworldFactionTag = core.OverworldFactionTag
		TickStateTag = core.TickStateTag
		VictoryStateTag = core.VictoryStateTag
		TravelStateTag = core.TravelStateTag
		OverworldEncounterTag = core.OverworldEncounterTag
	})
}
