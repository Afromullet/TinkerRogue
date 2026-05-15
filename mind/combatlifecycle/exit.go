package combatlifecycle

import (
	"game_main/core/common"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// CombatExitReason describes why combat ended.
type CombatExitReason int

const (
	ExitVictory CombatExitReason = iota
	ExitDefeat
	ExitFlee
)

// DetermineExitReason maps combat-end state to the lifecycle exit enum.
// Flee takes precedence over victory state: a player who wins on the same tick
// they pressed flee is still treated as fleeing.
func DetermineExitReason(fleeRequested, playerVictory bool) CombatExitReason {
	switch {
	case fleeRequested:
		return ExitFlee
	case playerVictory:
		return ExitVictory
	default:
		return ExitDefeat
	}
}

// String returns a human-readable name for the exit reason.
func (r CombatExitReason) String() string {
	switch r {
	case ExitVictory:
		return "Victory"
	case ExitDefeat:
		return "Defeat"
	case ExitFlee:
		return "Fled"
	default:
		return "Unknown"
	}
}

// EncounterOutcome captures the combat outcome for the exit pipeline.
// Built by the GUI layer from CombatService.CheckVictoryCondition().
type EncounterOutcome struct {
	IsPlayerVictory  bool
	VictorFaction    ecs.EntityID
	VictorName       string
	RoundsCompleted  int
	DefeatedFactions []ecs.EntityID
}

// ResolutionContext is the single value-struct carrying everything the exit
// pipeline needs from the moment EncounterService.ExitCombat starts the
// orchestration. Built once by the caller and passed by value to ExecuteCombatExit,
// the resolver, and every ExitHooks method.
//
// Fields are role-phased rather than type-segregated; consumers should read
// only the fields appropriate to their role:
//
//   - Resolver-facing: Reason, Outcome, PlayerEntityID, PlayerSquadIDs.
//     Resolvers should treat Setup and OriginalPlayerPosition as opaque —
//     state captured into the resolver's own fields at Prepare time is the
//     correct channel for setup data.
//   - Orchestration-facing: Setup (for Type/EnemySquadIDs/Resolver routing),
//     OriginalPlayerPosition (for OnRestorePlayer), and Outcome (for hooks
//     that need VictorName, RoundsCompleted, etc.).
//
// Outcome must be non-nil; the GUI layer builds it in CombatMode.Exit and it
// is propagated through ExecuteCombatExit without copy-on-write.
type ResolutionContext struct {
	Setup CombatSetup

	Reason         CombatExitReason
	Outcome        *EncounterOutcome
	PlayerEntityID ecs.EntityID
	PlayerSquadIDs []ecs.EntityID

	OriginalPlayerPosition coords.LogicalPosition
}

// CombatResolver handles context-specific combat resolution.
// Each combat type implements this: overworld, raid, garrison defense.
// Resolve() applies domain-specific state changes (threat damage, room
// clearing, etc.) and returns a freshly-built ResolutionResult describing
// rewards to grant. Resolvers leave RewardText empty; ExecuteResolution
// fills it in after applying Grant. Return nil for no-resolution scenarios.
type CombatResolver interface {
	Resolve(manager *common.EntityManager, ctx ResolutionContext) *ResolutionResult
}

// ResolutionResult carries the outcome of a combat resolution from the
// resolver through the grant step and out to post-combat consumers. Fields
// are lifecycle-phased:
//
//   - Rewards, Target, Description are set by the resolver. Target identifies
//     who receives Rewards and is consumed only by ExecuteResolution; it is
//     left zero-valued by resolvers that grant nothing.
//   - RewardText is set by ExecuteResolution after Grant runs and is empty
//     when the resolver returns. It carries the human-readable summary that
//     post-combat listeners (e.g., RaidRunner) display.
type ResolutionResult struct {
	Rewards     Reward
	Target      GrantTarget // input to ExecuteResolution; zero-valued post-Grant
	RewardText  string      // output from ExecuteResolution; empty pre-Grant
	Description string      // resolver-supplied summary ("Threat 42 destroyed")
}

// ExecuteResolution is THE single entry point for all combat resolution.
// All combat types call this. All rewards flow through here.
// Mutates the resolver-returned pointer to fill in RewardText after Grant.
func ExecuteResolution(manager *common.EntityManager, resolver CombatResolver, ctx ResolutionContext) *ResolutionResult {
	result := resolver.Resolve(manager, ctx)
	if result == nil {
		return &ResolutionResult{}
	}
	if result.Rewards.Gold > 0 || result.Rewards.Experience > 0 || result.Rewards.Mana > 0 {
		result.RewardText = Grant(manager, result.Rewards, result.Target)
	}
	return result
}

// ExitHooks is the contract for the mode-specific side effects that fire
// at fixed points during the exit orchestration. The encounter package
// supplies the concrete implementation; raid relies on the post-combat
// callback fired by the caller after ExecuteCombatExit returns.
//
//   - OnAfterResolution: fires after the resolver runs. Used for sprite restore
//     on flee and marking the encounter defeated on victory.
//   - OnRestorePlayer:   fires next. Restores the player to their pre-encounter
//     position.
//   - OnBeforeTeardown:  fires immediately before CombatTeardown.TeardownCombat,
//     allowing mode-specific entity preservation (e.g., garrison-defense return
//     to node).
type ExitHooks interface {
	OnAfterResolution(ctx ResolutionContext)
	OnRestorePlayer(ctx ResolutionContext)
	OnBeforeTeardown(ctx ResolutionContext)
}

// ExecuteCombatExit is THE single entry point for all combat exit orchestration.
// Mirrors ExecuteCombatStart for the resolution/teardown half of the lifecycle.
// All combat types flow through here.
//
// Sequencing (any nil dependency is skipped):
//  1. ExecuteResolution dispatches the resolver.
//  2. hooks.OnAfterResolution applies mode-specific post-resolution effects.
//  3. hooks.OnRestorePlayer restores player position.
//  4. onHistory lets the caller record the completed encounter to its history
//     and clear its activeEncounter snapshot before teardown disposes entities.
//  5. hooks.OnBeforeTeardown applies mode-specific pre-teardown effects.
//  6. teardown.TeardownCombat disposes tactical entities.
//
// The caller is responsible for firing any post-combat listeners (e.g.,
// RaidRunner) after this function returns; doing it inside the orchestration
// would tie the listener lifecycle to combatlifecycle, which is intentionally
// kept domain-free.
func ExecuteCombatExit(
	manager *common.EntityManager,
	ctx ResolutionContext,
	hooks ExitHooks,
	teardown CombatTeardown,
	onHistory func(*ResolutionResult),
) *ResolutionResult {
	var resolution *ResolutionResult
	if ctx.Setup.Resolver != nil {
		resolution = ExecuteResolution(manager, ctx.Setup.Resolver, ctx)
	} else {
		resolution = &ResolutionResult{}
	}

	if hooks != nil {
		hooks.OnAfterResolution(ctx)
		hooks.OnRestorePlayer(ctx)
	}

	if onHistory != nil {
		onHistory(resolution)
	}

	if hooks != nil {
		hooks.OnBeforeTeardown(ctx)
	}

	if teardown != nil {
		teardown.TeardownCombat(ctx.Setup.EnemySquadIDs)
	}

	return resolution
}
