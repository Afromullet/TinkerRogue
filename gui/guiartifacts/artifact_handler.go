package guiartifacts

import (
	"fmt"
	"game_main/gear"
	"game_main/tactical/combat"
	"game_main/templates"
	"game_main/visual/graphics"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// TargetType describes what kind of target an artifact requires.
type TargetType int

const (
	TargetFriendlySquad TargetType = iota
	TargetEnemySquad
	TargetNoTarget
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

// ToggleArtifactMode opens the artifact list or cancels artifact mode.
func (h *ArtifactActivationHandler) ToggleArtifactMode() {
	if h.deps.BattleState.InArtifactMode {
		h.CancelArtifactMode()
		return
	}

	available := h.GetAvailableArtifacts()
	if len(available) == 0 {
		h.deps.AddCombatLog("No activatable artifacts equipped")
		return
	}

	h.deps.BattleState.InArtifactMode = true
	h.deps.AddCombatLog("ARTIFACT MODE - Select an artifact to activate, ESC to cancel")
}

// SelectArtifact stores the selected artifact and enters targeting mode
// (or activates immediately for no-target artifacts).
func (h *ArtifactActivationHandler) SelectArtifact(behaviorKey string) {
	if !gear.CanActivateArtifact(behaviorKey, h.deps.CombatService.GetChargeTracker()) {
		h.deps.AddCombatLog("Artifact charge not available")
		return
	}

	targetType := GetTargetType(behaviorKey)

	if targetType == TargetNoTarget {
		// Execute immediately (e.g. Saboteur's Hourglass)
		h.executeArtifact(behaviorKey, 0)
		return
	}

	h.deps.BattleState.SelectedArtifactBehavior = behaviorKey

	switch targetType {
	case TargetFriendlySquad:
		h.deps.AddCombatLog("Click a friendly squad to apply artifact effect")
	case TargetEnemySquad:
		h.deps.AddCombatLog("Click an enemy squad to apply artifact effect")
	}
}

// HandleTargetClick resolves a click to a squad and validates the target.
func (h *ArtifactActivationHandler) HandleTargetClick(mouseX, mouseY int) {
	behaviorKey := h.deps.BattleState.SelectedArtifactBehavior
	if behaviorKey == "" || h.playerPos == nil {
		return
	}

	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *h.playerPos)
	clickedSquadID := combat.GetSquadAtPosition(clickedPos, h.deps.CombatService.EntityManager)

	if clickedSquadID == 0 {
		h.deps.AddCombatLog("No squad at that position")
		return
	}

	targetType := GetTargetType(behaviorKey)
	isEnemy := h.isEnemySquad(clickedSquadID)

	switch targetType {
	case TargetFriendlySquad:
		if isEnemy {
			h.deps.AddCombatLog("Must target a friendly squad")
			return
		}
	case TargetEnemySquad:
		if !isEnemy {
			h.deps.AddCombatLog("Must target an enemy squad")
			return
		}
	}

	h.executeArtifact(behaviorKey, clickedSquadID)
}

// CancelArtifactMode clears all artifact state.
func (h *ArtifactActivationHandler) CancelArtifactMode() {
	h.deps.BattleState.InArtifactMode = false
	h.deps.BattleState.SelectedArtifactBehavior = ""
	h.deps.AddCombatLog("Artifact activation cancelled")
}

// GetAvailableArtifacts returns options for all equipped major artifacts in the player's faction.
func (h *ArtifactActivationHandler) GetAvailableArtifacts() []ArtifactOption {
	playerFactionID := h.getPlayerFactionID()
	if playerFactionID == 0 {
		return nil
	}

	squadIDs := combat.GetSquadsForFaction(playerFactionID, h.deps.CombatService.EntityManager)
	chargeTracker := h.deps.CombatService.GetChargeTracker()

	seen := make(map[string]bool)
	var options []ArtifactOption

	for _, squadID := range squadIDs {
		defs := gear.GetArtifactDefinitions(squadID, h.deps.CombatService.EntityManager)
		for _, def := range defs {
			if def.Tier != "major" || def.Behavior == "" {
				continue
			}
			// Only include player-activated behaviors
			b := gear.GetBehavior(def.Behavior)
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
				Available:   gear.CanActivateArtifact(def.Behavior, chargeTracker),
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

// GetTargetType returns the targeting type for a given behavior key.
func GetTargetType(behaviorKey string) TargetType {
	switch behaviorKey {
	case gear.BehaviorDoubleTime, gear.BehaviorAnthemPerseverance, gear.BehaviorChainOfCommand:
		return TargetFriendlySquad
	case gear.BehaviorStandDown, gear.BehaviorDeadlockShackles:
		return TargetEnemySquad
	case gear.BehaviorSaboteurWsHourglass:
		return TargetNoTarget
	default:
		return TargetNoTarget
	}
}

// --- Internal helpers ---

func (h *ArtifactActivationHandler) executeArtifact(behaviorKey string, targetSquadID ecs.EntityID) {
	ctx := &gear.BehaviorContext{
		Manager:       h.deps.CombatService.EntityManager,
		Cache:         h.deps.CombatService.CombatCache,
		ChargeTracker: h.deps.CombatService.GetChargeTracker(),
	}

	err := gear.ActivateArtifact(behaviorKey, targetSquadID, ctx)
	if err != nil {
		h.deps.AddCombatLog(fmt.Sprintf("Artifact failed: %s", err))
	} else {
		name := behaviorKey
		if def := h.findDefinitionByBehavior(behaviorKey); def != nil {
			name = def.Name
		}
		h.deps.AddCombatLog(fmt.Sprintf("Activated %s!", name))
	}

	// Clear artifact state regardless of success
	h.deps.BattleState.InArtifactMode = false
	h.deps.BattleState.SelectedArtifactBehavior = ""

	// Invalidate caches since artifact effects may have changed squad stats
	h.deps.Queries.MarkAllSquadsDirty()
}

func (h *ArtifactActivationHandler) findDefinitionByBehavior(behaviorKey string) *templates.ArtifactDefinition {
	playerFactionID := h.getPlayerFactionID()
	if playerFactionID == 0 {
		return nil
	}
	squadIDs := combat.GetSquadsForFaction(playerFactionID, h.deps.CombatService.EntityManager)
	for _, squadID := range squadIDs {
		defs := gear.GetArtifactDefinitions(squadID, h.deps.CombatService.EntityManager)
		for _, def := range defs {
			if def.Behavior == behaviorKey {
				return def
			}
		}
	}
	return nil
}

func (h *ArtifactActivationHandler) isEnemySquad(squadID ecs.EntityID) bool {
	squadInfo := h.deps.Queries.GetSquadInfo(squadID)
	if squadInfo == nil {
		return false
	}

	encounterID := h.deps.EncounterService.GetCurrentEncounterID()
	if encounterID == 0 {
		return false
	}

	factions := h.deps.Queries.GetFactionsForEncounter(encounterID)
	for _, factionID := range factions {
		factionData := h.deps.Queries.CombatCache.FindFactionDataByID(factionID)
		if factionData != nil && factionData.IsPlayerControlled {
			return squadInfo.FactionID != factionID
		}
	}
	return false
}

func (h *ArtifactActivationHandler) getPlayerFactionID() ecs.EntityID {
	encounterID := h.deps.EncounterService.GetCurrentEncounterID()
	if encounterID == 0 {
		return 0
	}
	factions := h.deps.Queries.GetFactionsForEncounter(encounterID)
	for _, factionID := range factions {
		factionData := h.deps.Queries.CombatCache.FindFactionDataByID(factionID)
		if factionData != nil && factionData.IsPlayerControlled {
			return factionID
		}
	}
	return 0
}
