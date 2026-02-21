package combat

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// DefaultMovementSpeed is the fallback movement speed for squads with no valid units.
const DefaultMovementSpeed = 3

// Component and tag variables
var (
	CombatFactionComponent     *ecs.Component
	TurnStateComponent         *ecs.Component
	ActionStateComponent       *ecs.Component
	FactionMembershipComponent *ecs.Component

	FactionTag       ecs.Tag
	TurnStateTag     ecs.Tag
	ActionStateTag   ecs.Tag
	CombatFactionTag ecs.Tag
)

// Package-level Views for zero-allocation queries.
// Initialized in init(); automatically maintained by the ECS library.
var (
	factionView     *ecs.View
	combatSquadView *ecs.View // View on squads.SquadTag, used by GetSquadsForFaction
)

type FactionData struct {
	FactionID          ecs.EntityID // Unique faction identifier
	Name               string       // Display name (e.g., "Player", "Goblins")
	Mana               int          // Current mana for magic abilities
	MaxMana            int          // Maximum mana capacity
	IsPlayerControlled bool         // True if controlled by player
	PlayerID           int          // Player identifier (0 = AI, 1 = Player 1, 2 = Player 2, etc.)
	PlayerName         string       // Display name for player ("Player 1", "Player 2", or custom)
	EncounterID        ecs.EntityID // Encounter this faction belongs to (0 if not from encounter)
}

type TurnStateData struct {
	CurrentRound     int            // Current round number (starts at 1)
	TurnOrder        []ecs.EntityID // Faction IDs in order (randomized at start)
	CurrentTurnIndex int            // Index into TurnOrder (0 to len-1)
	CombatActive     bool           // True if combat is ongoing
}

type ActionStateData struct {
	SquadID           ecs.EntityID // Squad this action state belongs to
	HasMoved          bool         // True if squad moved this turn
	HasActed          bool         // True if squad attacked/used skill this turn
	MovementRemaining int          // Tiles left to move (starts at squad speed)
	BonusAttackActive bool         // When true, next markSquadAsActed is consumed without setting HasActed
}

// CombatFactionData links a squad to its faction during combat.
// This component is added to squad entities when they enter combat.
// Squad gains this component when entering combat, loses it when exiting.
type CombatFactionData struct {
	FactionID ecs.EntityID // Faction that owns this squad
}

// init registers the combat subsystem with the ECS component registry.
// This allows the combat package to self-register its components without
// game_main needing to know about combat internals.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		InitCombatComponents(em)
		InitCombatTags(em)
		factionView = em.World.CreateView(FactionTag)
		combatSquadView = em.World.CreateView(squads.SquadTag)
	})
}

// InitCombatComponents registers all combat-related components with the ECS manager.
// Call this during game initialization, similar to InitSquadComponents.
func InitCombatComponents(manager *common.EntityManager) {
	CombatFactionComponent = manager.World.NewComponent()
	TurnStateComponent = manager.World.NewComponent()
	ActionStateComponent = manager.World.NewComponent()
	FactionMembershipComponent = manager.World.NewComponent()
}

// InitCombatTags creates tags for querying combat-related entities.
// Call this after InitCombatComponents.
func InitCombatTags(manager *common.EntityManager) {
	FactionTag = ecs.BuildTag(CombatFactionComponent)
	TurnStateTag = ecs.BuildTag(TurnStateComponent)
	ActionStateTag = ecs.BuildTag(ActionStateComponent)
	CombatFactionTag = ecs.BuildTag(FactionMembershipComponent)

	manager.WorldTags["faction"] = FactionTag
	manager.WorldTags["turnstate"] = TurnStateTag
	manager.WorldTags["actionstate"] = ActionStateTag
	manager.WorldTags["combatfaction"] = CombatFactionTag
}
