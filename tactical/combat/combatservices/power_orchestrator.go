package combatservices

import (
	"game_main/core/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/powers/artifacts"
	"game_main/tactical/powers/perks"
	"game_main/tactical/powers/powercore"

	"github.com/bytearena/ecs"
)

// GUIHooks captures the optional GUI callbacks the power pipeline fans out to
// after artifacts and perks have run. Any field may be nil; the pipeline
// closures nil-check at call time so callbacks can be set later via
// CombatService.SetOn*GUI.
type GUIHooks struct {
	OnAttackComplete func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)
	OnMoveComplete   func(squadID ecs.EntityID)
	OnTurnEnd        func(round int)
}

// PowerOrchestrator owns the per-battle artifact/perk dispatchers, the shared
// charge tracker, and the PowerPipeline that fans lifecycle events out to them
// in declared order. It also holds the canonical PowerLogger reused by ability,
// gear, and service log emitters.
//
// CombatService composes PowerOrchestrator instead of owning these fields
// directly — adding a new power system is a single subscriber registration in
// WirePipeline, not a surgical edit to NewCombatService.
type PowerOrchestrator struct {
	chargeTracker      *artifacts.ArtifactChargeTracker
	artifactDispatcher *artifacts.ArtifactDispatcher
	perkDispatcher     *perks.SquadPerkDispatcher
	pipeline           *powercore.PowerPipeline
	logger             powercore.PowerLogger
}

// NewPowerOrchestrator constructs the orchestrator with its artifact pieces
// already wired (charge tracker shared with the dispatcher) but without a
// logger or perk dispatcher — those are set by InstallLogger so the same
// logger instance flows through every subsystem.
func NewPowerOrchestrator(manager *common.EntityManager, cache *combatstate.CombatQueryCache) *PowerOrchestrator {
	chargeTracker := artifacts.NewArtifactChargeTracker()
	return &PowerOrchestrator{
		chargeTracker:      chargeTracker,
		artifactDispatcher: artifacts.NewArtifactDispatcher(manager, cache, chargeTracker),
		pipeline:           &powercore.PowerPipeline{},
	}
}

// InstallLogger sets the shared PowerLogger across artifacts, perks, abilities,
// and gear messages, and creates the perk dispatcher bound to this logger.
// Must be called before WirePipeline so the perk dispatcher exists when
// subscribers register.
func (o *PowerOrchestrator) InstallLogger(logger powercore.PowerLogger) {
	o.logger = logger
	o.artifactDispatcher.SetLogger(logger)
	combatstate.SetGearLogger(func(source string, squadID ecs.EntityID, message string) {
		logger.Log(source, squadID, message)
	})

	perkDispatcher := &perks.SquadPerkDispatcher{}
	perkDispatcher.SetLogger(logger)
	o.perkDispatcher = perkDispatcher
}

// WirePipeline registers all pipeline subscribers in their required order and
// forwards subsystem-fired events (CombatActionSystem.OnAttackComplete,
// MovementSystem.OnMoveComplete, TurnManager.OnTurnEnd/PostResetHook) into the
// pipeline. The hooks pointer is dereferenced at event-fire time, so callers
// can mutate its fields after this returns.
//
// Order rationale per event:
//   - PostReset:        artifacts.OnPostReset → perks.TurnStart
//   - OnAttackComplete: artifacts.OnAttackComplete → perks state tracking → GUI
//   - OnTurnEnd:        artifacts charge refresh + OnTurnEnd → perks round reset → GUI
//   - OnMoveComplete:   perks movement tracking → GUI (no artifact hook)
func (o *PowerOrchestrator) WirePipeline(
	em *common.EntityManager,
	tm *combatcore.TurnManager,
	cas *combatcore.CombatActionSystem,
	ms *combatcore.CombatMovementSystem,
	hooks *GUIHooks,
) {
	o.pipeline.OnPostReset(o.artifactDispatcher.DispatchPostReset)
	o.pipeline.OnPostReset(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
		if o.perkDispatcher != nil {
			o.perkDispatcher.DispatchTurnStart(squadIDs, tm.GetCurrentRound(), em)
		}
	})

	o.pipeline.OnAttackComplete(o.artifactDispatcher.DispatchOnAttackComplete)
	o.pipeline.OnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
		if o.perkDispatcher != nil {
			o.perkDispatcher.DispatchAttackTracking(attackerID, defenderID, em)
		}
	})
	o.pipeline.OnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
		if hooks != nil && hooks.OnAttackComplete != nil {
			hooks.OnAttackComplete(attackerID, defenderID, result)
		}
	})

	o.pipeline.OnTurnEnd(o.artifactDispatcher.DispatchOnTurnEnd)
	o.pipeline.OnTurnEnd(func(round int) {
		if o.perkDispatcher != nil {
			o.perkDispatcher.DispatchRoundEnd(em)
		}
	})
	o.pipeline.OnTurnEnd(func(round int) {
		if hooks != nil && hooks.OnTurnEnd != nil {
			hooks.OnTurnEnd(round)
		}
	})

	o.pipeline.OnMoveComplete(func(squadID ecs.EntityID) {
		if o.perkDispatcher != nil {
			o.perkDispatcher.DispatchMoveTracking(squadID, em)
		}
	})
	o.pipeline.OnMoveComplete(func(squadID ecs.EntityID) {
		if hooks != nil && hooks.OnMoveComplete != nil {
			hooks.OnMoveComplete(squadID)
		}
	})

	cas.SetOnAttackComplete(o.pipeline.FireAttackComplete)
	ms.SetOnMoveComplete(o.pipeline.FireMoveComplete)
	tm.SetOnTurnEnd(o.pipeline.FireTurnEnd)
	tm.SetPostResetHook(o.pipeline.FirePostReset)
	cas.SetPerkDispatcher(o.perkDispatcher)
}

// Logger returns the canonical PowerLogger installed via InstallLogger.
func (o *PowerOrchestrator) Logger() powercore.PowerLogger { return o.logger }

// ChargeTracker returns the per-battle artifact charge tracker. Callers reset
// it via ChargeTracker().Reset() at the start of each combat.
func (o *PowerOrchestrator) ChargeTracker() *artifacts.ArtifactChargeTracker {
	return o.chargeTracker
}
