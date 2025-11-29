package combat

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// Component and tag variables
var (
	FactionComponent       *ecs.Component
	TurnStateComponent     *ecs.Component
	ActionStateComponent   *ecs.Component
	CombatFactionComponent *ecs.Component

	FactionTag       ecs.Tag
	TurnStateTag     ecs.Tag
	ActionStateTag   ecs.Tag
	CombatFactionTag ecs.Tag
)

type FactionData struct {
	FactionID          ecs.EntityID // Unique faction identifier
	Name               string       // Display name (e.g., "Player", "Goblins")
	Mana               int          // Current mana for magic abilities
	MaxMana            int          // Maximum mana capacity
	IsPlayerControlled bool         // True if controlled by player
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
}

// CombatFactionData links a squad to its faction during combat.
// This component is added to squad entities when they enter combat.
// Squad gains this component when entering combat, loses it when exiting.
type CombatFactionData struct {
	FactionID ecs.EntityID // Faction that owns this squad
}

// InitCombatComponents registers all combat-related components with the ECS manager.
// Call this during game initialization, similar to InitSquadComponents.
func InitCombatComponents(manager *common.EntityManager) {
	FactionComponent = manager.World.NewComponent()
	TurnStateComponent = manager.World.NewComponent()
	ActionStateComponent = manager.World.NewComponent()
	CombatFactionComponent = manager.World.NewComponent()
}

// InitCombatTags creates tags for querying combat-related entities.
// Call this after InitCombatComponents.
func InitCombatTags(manager *common.EntityManager) {
	FactionTag = ecs.BuildTag(FactionComponent)
	TurnStateTag = ecs.BuildTag(TurnStateComponent)
	ActionStateTag = ecs.BuildTag(ActionStateComponent)
	CombatFactionTag = ecs.BuildTag(CombatFactionComponent)

	manager.WorldTags["faction"] = FactionTag
	manager.WorldTags["turnstate"] = TurnStateTag
	manager.WorldTags["actionstate"] = ActionStateTag
	manager.WorldTags["combatfaction"] = CombatFactionTag
}

// InitializeCombatSystem initializes combat components and tags in the provided EntityManager.
// This should be called during game initialization, similar to InitializeSquadData.
func InitializeCombatSystem(manager *common.EntityManager) {
	InitCombatComponents(manager)
	InitCombatTags(manager)
}

// Component data structures defined above
