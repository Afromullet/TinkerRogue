package guioverworld

import (
	"fmt"

	"game_main/common"
	"game_main/mind/encounter"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/overworld/tick"
	"game_main/overworld/travel"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// OverworldActionHandler handles all game-state-changing logic for the overworld.
// Pattern from gui/guicombat/combat_action_handler.go.
type OverworldActionHandler struct {
	deps *OverworldModeDeps
}

func NewOverworldActionHandler(deps *OverworldModeDeps) *OverworldActionHandler {
	return &OverworldActionHandler{deps: deps}
}

// AdvanceTick advances the overworld tick and handles travel completion / raids.
func (ah *OverworldActionHandler) AdvanceTick() {
	tickResult, err := tick.AdvanceTick(ah.deps.Manager, ah.deps.PlayerData)
	if err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: %v", err))
		return
	}

	tickState := core.GetTickState(ah.deps.Manager)
	travelState := travel.GetTravelState(ah.deps.Manager)

	if travelState != nil && travelState.IsTraveling {
		ah.deps.LogEvent(fmt.Sprintf("Tick %d - %d ticks remaining",
			tickState.CurrentTick, travelState.TicksRemaining))
	} else {
		ah.deps.LogEvent(fmt.Sprintf("Tick advanced to %d", tickState.CurrentTick))
	}

	ah.deps.RefreshPanels()

	// Handle pending raid on a garrisoned player node
	if tickResult.PendingRaid != nil {
		ah.HandleRaid(tickResult.PendingRaid)
		return
	}

	// If travel completed, stop auto-travel and start combat
	if tickResult.TravelCompleted {
		ah.deps.State.IsAutoTraveling = false
		ah.StartCombatAfterTravel()
	}
}

// EngageThreat validates the selected node and starts travel toward it.
func (ah *OverworldActionHandler) EngageThreat(nodeID ecs.EntityID) {
	if nodeID == 0 {
		ah.deps.LogEvent("No threat selected")
		return
	}

	threatEntity := ah.deps.Manager.FindEntityByID(nodeID)
	if threatEntity == nil {
		ah.deps.LogEvent("ERROR: Threat entity not found")
		return
	}

	threatData := common.GetComponentType[*core.OverworldNodeData](threatEntity, core.OverworldNodeComponent)
	if threatData == nil {
		ah.deps.LogEvent("ERROR: Invalid threat entity")
		return
	}

	posData := common.GetComponentType[*coords.LogicalPosition](threatEntity, common.PositionComponent)
	if posData == nil {
		ah.deps.LogEvent("ERROR: Threat has no position")
		return
	}

	encounterID, err := encounter.TriggerCombatFromThreat(ah.deps.Manager, threatEntity)
	if err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: Failed to create encounter: %v", err))
		return
	}

	if err := travel.StartTravel(
		ah.deps.Manager,
		ah.deps.PlayerData,
		*posData,
		nodeID,
		encounterID,
	); err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: %v", err))
		return
	}

	travelNodeDef := core.GetNodeRegistry().GetNodeByID(threatData.NodeTypeID)
	travelDisplayName := threatData.NodeTypeID
	if travelNodeDef != nil {
		travelDisplayName = travelNodeDef.DisplayName
	}
	ah.deps.LogEvent(fmt.Sprintf("Traveling to %s (Press C to cancel)...", travelDisplayName))
	ah.deps.State.ClearSelection()
}

// CancelTravel cancels active travel and resets auto-travel.
func (ah *OverworldActionHandler) CancelTravel() {
	if !travel.IsTraveling(ah.deps.Manager) {
		ah.deps.LogEvent("Not currently traveling")
		return
	}

	if err := travel.CancelTravel(ah.deps.Manager, ah.deps.PlayerData); err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: %v", err))
		return
	}

	ah.deps.State.IsAutoTraveling = false
	ah.deps.LogEvent("Travel cancelled - returned to origin")
}

// StartCombatAfterTravel initiates combat once travel completes.
func (ah *OverworldActionHandler) StartCombatAfterTravel() {
	travelState := travel.GetTravelState(ah.deps.Manager)
	if travelState == nil {
		return
	}

	threatEntity := ah.deps.Manager.FindEntityByID(travelState.TargetThreatID)
	if threatEntity == nil {
		ah.deps.LogEvent("ERROR: Threat not found")
		return
	}

	threatData := common.GetComponentType[*core.OverworldNodeData](
		threatEntity, core.OverworldNodeComponent)
	posData := common.GetComponentType[*coords.LogicalPosition](
		threatEntity, common.PositionComponent)

	if threatData == nil || posData == nil {
		ah.deps.LogEvent("ERROR: Invalid threat entity")
		return
	}

	nodeDef := core.GetNodeRegistry().GetNodeByID(threatData.NodeTypeID)
	displayName := threatData.NodeTypeID
	if nodeDef != nil {
		displayName = nodeDef.DisplayName
	}
	threatName := fmt.Sprintf("%s (Level %d)", displayName, threatData.Intensity)

	playerEntityID := ecs.EntityID(0)
	if ah.deps.PlayerData != nil {
		playerEntityID = ah.deps.PlayerData.PlayerEntityID
	}

	if err := ah.deps.EncounterService.StartEncounter(
		travelState.TargetEncounterID,
		travelState.TargetThreatID,
		threatName,
		*posData,
		playerEntityID,
	); err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: %v", err))
	}
}

// HandleRaid creates and starts a garrison defense encounter.
func (ah *OverworldActionHandler) HandleRaid(raid *core.PendingRaid) {
	ah.deps.LogEvent(fmt.Sprintf("%s faction raiding garrisoned node %d!", raid.AttackingFactionType.String(), raid.TargetNodeID))

	playerEntityID := ecs.EntityID(0)
	if ah.deps.PlayerData != nil {
		playerEntityID = ah.deps.PlayerData.PlayerEntityID
	}

	encounterID, err := encounter.TriggerGarrisonDefense(
		ah.deps.Manager,
		raid.TargetNodeID,
		raid.AttackingFactionType,
		raid.AttackingStrength,
	)
	if err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: Failed to create garrison defense: %v", err))
		return
	}

	if err := ah.deps.EncounterService.StartGarrisonDefense(
		encounterID,
		raid.TargetNodeID,
		playerEntityID,
	); err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: Failed to start garrison defense: %v", err))
		return
	}
}

// ToggleInfluence toggles the influence zone display.
func (ah *OverworldActionHandler) ToggleInfluence() {
	ah.deps.State.ShowInfluence = !ah.deps.State.ShowInfluence

	if ah.deps.State.ShowInfluence {
		ah.deps.LogEvent("Influence zones visible")
	} else {
		ah.deps.LogEvent("Influence zones hidden")
	}
}

// ToggleAutoTravel toggles automatic tick advancement during travel.
func (ah *OverworldActionHandler) ToggleAutoTravel() {
	if !travel.IsTraveling(ah.deps.Manager) {
		ah.deps.LogEvent("Auto-travel only available during travel")
		return
	}

	ah.deps.State.IsAutoTraveling = !ah.deps.State.IsAutoTraveling

	if ah.deps.State.IsAutoTraveling {
		ah.deps.LogEvent("Auto-travel enabled - automatically advancing ticks")
	} else {
		ah.deps.LogEvent("Auto-travel disabled")
	}
}

// AssignSquadToGarrison assigns a squad to a node's garrison.
func (ah *OverworldActionHandler) AssignSquadToGarrison(squadID, nodeID ecs.EntityID) error {
	if err := garrison.AssignSquadToNode(ah.deps.Manager, squadID, nodeID); err != nil {
		return err
	}
	squadName := squads.GetSquadName(squadID, ah.deps.Manager)
	ah.deps.LogEvent(fmt.Sprintf("Assigned %s to garrison", squadName))
	ah.deps.RefreshPanels()
	return nil
}

// RemoveSquadFromGarrison removes a squad from a node's garrison.
func (ah *OverworldActionHandler) RemoveSquadFromGarrison(squadID, nodeID ecs.EntityID) error {
	if err := garrison.RemoveSquadFromNode(ah.deps.Manager, squadID, nodeID); err != nil {
		return err
	}
	squadName := squads.GetSquadName(squadID, ah.deps.Manager)
	ah.deps.LogEvent(fmt.Sprintf("Removed %s from garrison", squadName))
	ah.deps.RefreshPanels()
	return nil
}
