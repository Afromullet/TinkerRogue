package guiartifacts

import (
	"fmt"
	"game_main/core/coords"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/powers/artifacts"
	"game_main/tactical/powers/powercore"

	"github.com/bytearena/ecs"
)

// ArtifactOption represents a single activatable artifact shown in the panel.
type ArtifactOption struct {
	BehaviorKey string
	Name        string
	Description string
	Available   bool // false if charge is spent
}

// ArtifactActivationHandler manages the full artifact activation workflow:
// artifact list display, targeting mode, and execution.
type ArtifactActivationHandler struct {
	deps      *ArtifactActivationDeps
	playerPos *coords.LogicalPosition
}

// NewArtifactActivationHandler creates a new artifact activation handler.
func NewArtifactActivationHandler(deps *ArtifactActivationDeps) *ArtifactActivationHandler {
	return &ArtifactActivationHandler{
		deps: deps,
	}
}

// SetPlayerPosition sets the player position for viewport calculations.
func (h *ArtifactActivationHandler) SetPlayerPosition(pos *coords.LogicalPosition) {
	h.playerPos = pos
}

// SelectArtifact stores the selected artifact and enters targeting mode
// (or activates immediately for no-target artifacts).
func (h *ArtifactActivationHandler) SelectArtifact(behaviorKey string) {
	if !artifacts.CanActivateArtifact(behaviorKey, h.deps.CombatService.GetChargeTracker()) {
		return
	}

	targetType := artifacts.GetTargetType(behaviorKey)

	if targetType == artifacts.TargetNone {
		// Execute immediately (e.g. Saboteur's Hourglass)
		h.executeArtifact(behaviorKey, 0)
		return
	}

	h.deps.BattleState.SelectedArtifactBehavior = behaviorKey
}

// HandleTargetClick resolves a click to a squad and validates the target.
func (h *ArtifactActivationHandler) HandleTargetClick(mouseX, mouseY int) {
	behaviorKey := h.deps.BattleState.SelectedArtifactBehavior
	if behaviorKey == "" || h.playerPos == nil {
		return
	}

	clickedPos := coords.MouseToLogicalPosition(mouseX, mouseY, *h.playerPos)
	clickedSquadID := combatstate.GetSquadAtPosition(clickedPos, h.deps.CombatService.EntityManager)

	if clickedSquadID == 0 {
		return
	}

	targetType := artifacts.GetTargetType(behaviorKey)
	encounterID := h.deps.Encounter.GetCurrentEncounterID()
	isEnemy := h.deps.Queries.IsEnemySquadInEncounter(clickedSquadID, encounterID)

	switch targetType {
	case artifacts.TargetFriendly:
		if isEnemy {
			return
		}
	case artifacts.TargetEnemy:
		if !isEnemy {
			return
		}
	}

	h.executeArtifact(behaviorKey, clickedSquadID)
}

// CancelArtifactMode clears all artifact state.
func (h *ArtifactActivationHandler) CancelArtifactMode() {
	h.deps.BattleState.InArtifactMode = false
	h.deps.BattleState.SelectedArtifactBehavior = ""
}

// GetAvailableArtifacts returns options for all equipped major artifacts in the player's faction.
func (h *ArtifactActivationHandler) GetAvailableArtifacts() []ArtifactOption {
	encounterID := h.deps.Encounter.GetCurrentEncounterID()
	playerFactionID := h.deps.Queries.GetPlayerFactionForEncounter(encounterID)
	if playerFactionID == 0 {
		return nil
	}

	squadIDs := combatstate.GetSquadsForFaction(playerFactionID, h.deps.CombatService.EntityManager)
	chargeTracker := h.deps.CombatService.GetChargeTracker()

	seen := make(map[string]bool)
	var options []ArtifactOption

	for _, squadID := range squadIDs {
		defs := artifacts.GetArtifactDefinitions(squadID, h.deps.CombatService.EntityManager)
		for _, def := range defs {
			if def.Tier != "major" || def.Behavior == "" {
				continue
			}
			// Only include player-activated behaviors
			b := artifacts.GetBehavior(def.Behavior)
			if b == nil || !b.IsPlayerActivated() {
				continue
			}
			if seen[def.Behavior] {
				continue
			}
			seen[def.Behavior] = true

			options = append(options, ArtifactOption{
				BehaviorKey: def.Behavior,
				Name:        def.Name,
				Description: def.Description,
				Available:   artifacts.CanActivateArtifact(def.Behavior, chargeTracker),
			})
		}
	}

	return options
}

// IsInArtifactMode returns true if artifact mode is active.
func (h *ArtifactActivationHandler) IsInArtifactMode() bool {
	return h.deps.BattleState.InArtifactMode
}

// HasSelectedArtifact returns true if an artifact has been selected for targeting.
func (h *ArtifactActivationHandler) HasSelectedArtifact() bool {
	return h.deps.BattleState.SelectedArtifactBehavior != ""
}

// --- Internal helpers ---

func (h *ArtifactActivationHandler) executeArtifact(behaviorKey string, targetSquadID ecs.EntityID) {
	ctx := artifacts.NewBehaviorContext(
		powercore.NewPowerContext(
			h.deps.CombatService.EntityManager,
			h.deps.CombatService.CombatCache,
			0,
			nil,
		),
		h.deps.CombatService.GetChargeTracker(),
	)

	if err := artifacts.ActivateArtifact(behaviorKey, targetSquadID, ctx); err != nil {
		fmt.Printf("[ARTIFACT] activation failed: %v\n", err)
	}

	// Clear artifact state regardless of success
	h.deps.BattleState.InArtifactMode = false
	h.deps.BattleState.SelectedArtifactBehavior = ""

	// Invalidate caches since artifact effects may have changed squad stats
	h.deps.Queries.MarkAllSquadsDirty()
}
