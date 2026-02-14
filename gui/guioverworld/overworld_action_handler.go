package guioverworld

import (
	"fmt"

	"game_main/common"
	"game_main/config"
	"game_main/mind/encounter"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/tactical/commander"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// OverworldActionHandler handles all game-state-changing logic for the overworld.
// Pattern from gui/guicombat/combat_action_handler.go.
type OverworldActionHandler struct {
	deps *OverworldModeDeps
}

func NewOverworldActionHandler(deps *OverworldModeDeps) *OverworldActionHandler {
	return &OverworldActionHandler{deps: deps}
}

// EndTurn ends the overworld turn: advances tick simulation and resets all commanders.
func (ah *OverworldActionHandler) EndTurn() {
	tickResult, err := commander.EndTurn(ah.deps.Manager, ah.deps.PlayerData)
	if err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: %v", err))
		return
	}

	tickState := core.GetTickState(ah.deps.Manager)
	turnState := commander.GetOverworldTurnState(ah.deps.Manager)

	turnStr := ""
	if turnState != nil {
		turnStr = fmt.Sprintf("Turn %d | ", turnState.CurrentTurn)
	}
	ah.deps.LogEvent(fmt.Sprintf("%sTick advanced to %d", turnStr, tickState.CurrentTick))

	// Clear move mode on turn end
	ah.deps.State.ExitMoveMode()

	ah.deps.RefreshPanels()

	// Handle pending raid on a garrisoned player node
	if tickResult.PendingRaid != nil {
		ah.HandleRaid(tickResult.PendingRaid)
	}
}

// MoveSelectedCommander moves the selected commander to the target position.
func (ah *OverworldActionHandler) MoveSelectedCommander(targetPos coords.LogicalPosition) {
	cmdID := ah.deps.State.SelectedCommanderID
	if cmdID == 0 {
		ah.deps.LogEvent("No commander selected")
		return
	}

	if ah.deps.CommanderMovement == nil {
		ah.deps.LogEvent("ERROR: Commander movement system not initialized")
		return
	}

	if err := ah.deps.CommanderMovement.MoveCommander(cmdID, targetPos); err != nil {
		ah.deps.LogEvent(fmt.Sprintf("Move failed: %v", err))
		return
	}

	// Update valid tiles after move
	tiles := ah.deps.CommanderMovement.GetValidMovementTiles(cmdID)
	if len(tiles) == 0 {
		// No movement left - exit move mode
		ah.deps.State.ExitMoveMode()
		ah.deps.LogEvent("Moved - no movement remaining")
	} else {
		ah.deps.State.ValidMoveTiles = tiles
		actionState := commander.GetCommanderActionState(cmdID, ah.deps.Manager)
		remaining := 0
		if actionState != nil {
			remaining = actionState.MovementRemaining
		}
		ah.deps.LogEvent(fmt.Sprintf("Moved to (%d,%d) - %d movement remaining", targetPos.X, targetPos.Y, remaining))
	}

	ah.deps.RefreshPanels()
}

// EngageThreat validates the selected threat and starts combat.
// Commander must be on the same tile as the threat.
func (ah *OverworldActionHandler) EngageThreat(nodeID ecs.EntityID) {
	if nodeID == 0 {
		ah.deps.LogEvent("No threat selected")
		return
	}

	cmdID := ah.deps.State.SelectedCommanderID
	if cmdID == 0 {
		ah.deps.LogEvent("No commander selected - select a commander first")
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

	threatPos := common.GetComponentType[*coords.LogicalPosition](threatEntity, common.PositionComponent)
	if threatPos == nil {
		ah.deps.LogEvent("ERROR: Threat has no position")
		return
	}

	// Commander must be on the same tile as the threat
	cmdEntity := ah.deps.Manager.FindEntityByID(cmdID)
	if cmdEntity == nil {
		ah.deps.LogEvent("ERROR: Commander entity not found")
		return
	}
	cmdPos := common.GetComponentType[*coords.LogicalPosition](cmdEntity, common.PositionComponent)
	if cmdPos == nil || cmdPos.X != threatPos.X || cmdPos.Y != threatPos.Y {
		ah.deps.LogEvent("Commander must be on the same tile as the threat to engage")
		return
	}

	encounterID, err := encounter.TriggerCombatFromThreat(ah.deps.Manager, threatEntity)
	if err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: Failed to create encounter: %v", err))
		return
	}

	nodeDef := core.GetNodeRegistry().GetNodeByID(threatData.NodeTypeID)
	displayName := threatData.NodeTypeID
	if nodeDef != nil {
		displayName = nodeDef.DisplayName
	}
	threatName := fmt.Sprintf("%s (Level %d)", displayName, threatData.Intensity)

	// Pass commander ID for roster access (commander owns the squads)
	if err := ah.deps.EncounterService.StartEncounter(
		encounterID,
		nodeID,
		threatName,
		*threatPos,
		cmdID, // Commander instead of player - squads are on the commander
	); err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: %v", err))
		return
	}

	// Clear move mode on combat start
	ah.deps.State.ExitMoveMode()
	ah.deps.State.ClearSelection()
}

// HandleRaid creates and starts a garrison defense encounter.
func (ah *OverworldActionHandler) HandleRaid(raid *core.PendingRaid) {
	ah.deps.LogEvent(fmt.Sprintf("%s faction raiding garrisoned node %d!", raid.AttackingFactionType.String(), raid.TargetNodeID))

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

// RecruitCommander creates a new commander at the selected commander's position.
// Requires the selected commander to be on a friendly settlement or fortress node.
// Costs gold from the player's resource stockpile.
func (ah *OverworldActionHandler) RecruitCommander() {
	cmdID := ah.deps.State.SelectedCommanderID
	if cmdID == 0 {
		ah.deps.LogEvent("No commander selected - select a commander first")
		return
	}

	playerID := ah.deps.PlayerData.PlayerEntityID

	// Check commander roster capacity
	roster := commander.GetPlayerCommanderRoster(playerID, ah.deps.Manager)
	if roster == nil {
		ah.deps.LogEvent("ERROR: No commander roster found")
		return
	}
	current, max := roster.GetCommanderCount()
	if current >= max {
		ah.deps.LogEvent(fmt.Sprintf("Commander limit reached (%d/%d)", current, max))
		return
	}

	// Get selected commander's position
	cmdEntity := ah.deps.Manager.FindEntityByID(cmdID)
	if cmdEntity == nil {
		ah.deps.LogEvent("ERROR: Commander entity not found")
		return
	}
	cmdPos := common.GetComponentType[*coords.LogicalPosition](cmdEntity, common.PositionComponent)
	if cmdPos == nil {
		ah.deps.LogEvent("ERROR: Commander has no position")
		return
	}

	// Check for friendly settlement/fortress at commander's position
	nodeID := core.GetNodeAtPosition(ah.deps.Manager, *cmdPos)
	if nodeID == 0 {
		ah.deps.LogEvent("Must be at a settlement or fortress to recruit")
		return
	}
	nodeEntity := ah.deps.Manager.FindEntityByID(nodeID)
	if nodeEntity == nil {
		ah.deps.LogEvent("ERROR: Node entity not found")
		return
	}
	nodeData := common.GetComponentType[*core.OverworldNodeData](nodeEntity, core.OverworldNodeComponent)
	if nodeData == nil || !core.IsFriendlyOwner(nodeData.OwnerID) {
		ah.deps.LogEvent("Must be at a player-owned settlement or fortress to recruit")
		return
	}
	if nodeData.Category != core.NodeCategorySettlement && nodeData.Category != core.NodeCategoryFortress {
		ah.deps.LogEvent("Must be at a settlement or fortress to recruit")
		return
	}

	// Check gold cost
	stockpile := common.GetResourceStockpile(playerID, ah.deps.Manager)
	if stockpile == nil {
		ah.deps.LogEvent("ERROR: No resource stockpile found")
		return
	}
	cost := config.DefaultCommanderCost
	if !common.CanAffordGold(stockpile, cost) {
		ah.deps.LogEvent(fmt.Sprintf("Not enough gold (need %d, have %d)", cost, stockpile.Gold))
		return
	}

	// Spend gold
	if err := common.SpendGold(stockpile, cost); err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: %v", err))
		return
	}

	// Load commander image
	commanderImage, _, err := ebitenutil.NewImageFromFile(config.PlayerImagePath)
	if err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: Failed to load commander image: %v", err))
		// Refund gold on failure
		common.AddGold(stockpile, cost)
		return
	}

	// Create new commander
	name := fmt.Sprintf("Commander %d", current+1)
	newCmdID := commander.CreateCommander(
		ah.deps.Manager,
		name,
		*cmdPos,
		config.DefaultCommanderMovementSpeed,
		config.DefaultCommanderMaxSquads,
		commanderImage,
	)

	// Add to roster
	if err := roster.AddCommander(newCmdID); err != nil {
		ah.deps.LogEvent(fmt.Sprintf("ERROR: %v", err))
		common.AddGold(stockpile, cost)
		return
	}

	// Select the new commander
	ah.deps.State.SelectedCommanderID = newCmdID
	ah.deps.State.ExitMoveMode()

	ah.deps.LogEvent(fmt.Sprintf("Recruited %s (cost: %d gold)", name, cost))
	ah.deps.RefreshPanels()
}
