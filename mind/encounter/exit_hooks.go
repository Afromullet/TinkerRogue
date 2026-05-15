package encounter

import (
	"game_main/campaign/overworld/core"
	"game_main/campaign/overworld/garrison"
	"game_main/core/common"
	"game_main/mind/combatlifecycle"

	"github.com/bytearena/ecs"
)

// EncounterExitHooks implements combatlifecycle.ExitHooks for overworld and
// garrison-defense encounters managed by EncounterService. It holds the
// dependencies the hook methods need (ECS manager and the mode coordinator
// used for player-position restoration) so the orchestration in
// combatlifecycle.ExecuteCombatExit can fire them at the right points without
// reaching back into encounter state.
type EncounterExitHooks struct {
	manager         *common.EntityManager
	modeCoordinator ModeCoordinator
}

// NewEncounterExitHooks builds the hook set used by EncounterService.ExitCombat.
func NewEncounterExitHooks(manager *common.EntityManager, mc ModeCoordinator) *EncounterExitHooks {
	return &EncounterExitHooks{manager: manager, modeCoordinator: mc}
}

// OnAfterResolution restores the encounter sprite on flee and marks the
// encounter defeated on overworld victory. Raid encounters skip both since
// they have no OverworldEncounterData.
func (h *EncounterExitHooks) OnAfterResolution(ctx combatlifecycle.ResolutionContext) {
	entity, data := h.getEncounterData(ctx.Setup.EncounterID)
	if ctx.Reason == combatlifecycle.ExitFlee {
		h.restoreEncounterSprite(entity, data)
	}
	if ctx.Outcome.IsPlayerVictory && ctx.Setup.Type != combatlifecycle.CombatTypeRaid {
		h.markEncounterDefeated(entity, data)
	}
}

// OnRestorePlayer puts the player camera back where it was before the encounter
// teleport. No-op when the mode coordinator is unset (test setups).
func (h *EncounterExitHooks) OnRestorePlayer(ctx combatlifecycle.ResolutionContext) {
	if h.modeCoordinator == nil {
		return
	}
	if pos := h.modeCoordinator.GetPlayerPosition(); pos != nil {
		*pos = ctx.OriginalPlayerPosition
	}
}

// OnBeforeTeardown returns garrison squads to their node after a successful
// defense so they survive the teardown's enemy-squad disposal. Other combat
// types are a no-op.
func (h *EncounterExitHooks) OnBeforeTeardown(ctx combatlifecycle.ResolutionContext) {
	if ctx.Setup.Type != combatlifecycle.CombatTypeGarrisonDefense || !ctx.Outcome.IsPlayerVictory {
		return
	}
	garrisonData := garrison.GetGarrisonAtNode(h.manager, ctx.Setup.DefendedNodeID)
	if garrisonData == nil {
		return
	}
	combatlifecycle.StripCombatComponents(h.manager, garrisonData.SquadIDs)
}

// markEncounterDefeated marks the encounter as defeated and hides its sprite permanently.
func (h *EncounterExitHooks) markEncounterDefeated(entity *ecs.Entity, encounterData *core.OverworldEncounterData) {
	if entity == nil || encounterData == nil {
		return
	}
	encounterData.IsDefeated = true
	if r := common.GetComponentType[*common.Renderable](entity, common.RenderableComponent); r != nil {
		r.Visible = false
	}
}

// restoreEncounterSprite restores the encounter sprite visibility when fleeing combat.
// No-op when the encounter has already been marked defeated by a prior victory.
func (h *EncounterExitHooks) restoreEncounterSprite(entity *ecs.Entity, encounterData *core.OverworldEncounterData) {
	if entity == nil || encounterData == nil || encounterData.IsDefeated {
		return
	}
	if r := common.GetComponentType[*common.Renderable](entity, common.RenderableComponent); r != nil {
		r.Visible = true
	}
}

// getEncounterData looks up an encounter entity and its OverworldEncounterData.
// Returns (nil, nil) if either the entity or the component is missing.
func (h *EncounterExitHooks) getEncounterData(encounterID ecs.EntityID) (*ecs.Entity, *core.OverworldEncounterData) {
	entity := h.manager.FindEntityByID(encounterID)
	if entity == nil {
		return nil, nil
	}
	data := common.GetComponentType[*core.OverworldEncounterData](entity, core.OverworldEncounterComponent)
	return entity, data
}
