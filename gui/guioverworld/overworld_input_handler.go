package guioverworld

import (
	"fmt"

	"game_main/common"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/tactical/commander"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
	ebitenui "github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
)

// OverworldInputHandler dispatches keyboard and mouse input for the overworld.
// Pattern from gui/guicombat/combat_input_handler.go.
type OverworldInputHandler struct {
	actionHandler *OverworldActionHandler
	deps          *OverworldModeDeps
	ui            *ebitenui.UI
}

func NewOverworldInputHandler(actionHandler *OverworldActionHandler, deps *OverworldModeDeps, ui *ebitenui.UI) *OverworldInputHandler {
	return &OverworldInputHandler{
		actionHandler: actionHandler,
		deps:          deps,
		ui:            ui,
	}
}

// HandleInput processes all keyboard and mouse input. Returns true if consumed.
func (ih *OverworldInputHandler) HandleInput(inputState *framework.InputState) bool {
	// ESC - exit move mode first, otherwise context switch back to tactical
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		if ih.deps.State.InMoveMode {
			ih.deps.State.ExitMoveMode()
			ih.deps.LogEvent("Movement mode cancelled")
			return true
		}
		if ih.deps.ModeCoordinator != nil {
			if err := ih.deps.ModeCoordinator.EnterTactical("exploration"); err != nil {
				fmt.Printf("ERROR: Failed to return to tactical context: %v\n", err)
			}
			return true
		}
	}

	// N - enter node placement mode
	if inputState.KeysJustPressed[ebiten.KeyN] {
		ih.deps.ModeManager.SetMode("node_placement")
		return true
	}

	// Space or Enter - end turn
	if inputState.KeysJustPressed[ebiten.KeySpace] || inputState.KeysJustPressed[ebiten.KeyEnter] {
		ih.actionHandler.EndTurn()
		return true
	}

	// M - toggle movement mode for selected commander
	if inputState.KeysJustPressed[ebiten.KeyM] {
		ih.toggleMoveMode()
		return true
	}

	// Tab - cycle to next commander
	if inputState.KeysJustPressed[ebiten.KeyTab] {
		ih.cycleCommander()
		return true
	}

	// I - toggle influence
	if inputState.KeysJustPressed[ebiten.KeyI] {
		ih.actionHandler.ToggleInfluence()
		return true
	}

	// G - garrison management
	if inputState.KeysJustPressed[ebiten.KeyG] {
		ih.handleGarrison()
		return true
	}

	// R - recruit new commander
	if inputState.KeysJustPressed[ebiten.KeyR] {
		ih.actionHandler.RecruitCommander()
		return true
	}

	// S - open squad management for selected commander
	if inputState.KeysJustPressed[ebiten.KeyS] {
		if ih.deps.State.SelectedCommanderID != 0 {
			ih.deps.ModeManager.SetMode("squad_editor")
			return true
		}
		ih.deps.LogEvent("No commander selected - click a commander first")
		return true
	}

	// E - engage selected threat (commander must be on same tile)
	if inputState.KeysJustPressed[ebiten.KeyE] {
		if ih.deps.State.HasSelection() {
			ih.actionHandler.EngageThreat(ih.deps.State.SelectedNodeID)
			return true
		}
	}

	// Mouse click
	if inputState.MousePressed && inputState.MouseButton == ebiten.MouseButtonLeft {
		return ih.handleMouseClick(inputState)
	}

	return false
}

// handleMouseClick handles commander selection, movement, and node selection.
func (ih *OverworldInputHandler) handleMouseClick(inputState *framework.InputState) bool {
	// In move mode: click on valid tile to move
	if ih.deps.State.InMoveMode && ih.deps.State.SelectedCommanderID != 0 {
		logicalPos := ih.deps.Renderer.ScreenToLogical(inputState.MouseX, inputState.MouseY)

		// Check if clicked position is a valid move tile
		for _, validPos := range ih.deps.State.ValidMoveTiles {
			if validPos.X == logicalPos.X && validPos.Y == logicalPos.Y {
				ih.actionHandler.MoveSelectedCommander(logicalPos)
				return true
			}
		}

		// Click outside valid tiles exits move mode
		ih.deps.State.ExitMoveMode()
		ih.deps.LogEvent("Movement mode cancelled")
		return true
	}

	// Check for commander at clicked position
	commanderID := ih.deps.Renderer.GetCommanderAtPosition(inputState.MouseX, inputState.MouseY)
	if commanderID != 0 {
		ih.deps.State.SelectedCommanderID = commanderID
		cmdData := commander.GetCommanderData(commanderID, ih.deps.Manager)
		name := "Commander"
		if cmdData != nil {
			name = cmdData.Name
		}

		// Also select any threat/node at the same tile so E/G work immediately
		threatID := ih.deps.Renderer.GetThreatAtPosition(inputState.MouseX, inputState.MouseY)
		if threatID != 0 {
			ih.deps.State.SelectedNodeID = threatID
			ih.deps.LogEvent(fmt.Sprintf("Selected %s on threat (Press E to engage)", name))
		} else {
			nodeID := ih.deps.Renderer.GetNodeAtPosition(inputState.MouseX, inputState.MouseY)
			if nodeID != 0 {
				ih.deps.State.SelectedNodeID = nodeID
				ih.deps.LogEvent(fmt.Sprintf("Selected %s at node (Press G to garrison)", name))
			} else {
				ih.deps.LogEvent(fmt.Sprintf("Selected %s (M=Move, S=Squads)", name))
			}
		}

		ih.deps.RefreshPanels()
		return true
	}

	// Check for threat (no commander on tile)
	threatID := ih.deps.Renderer.GetThreatAtPosition(inputState.MouseX, inputState.MouseY)
	if threatID != 0 {
		ih.deps.State.SelectedNodeID = threatID
		ih.deps.RefreshPanels()
		ih.deps.LogEvent(fmt.Sprintf("Selected threat %d (Move a commander here to engage)", threatID))
		return true
	}

	// Check for any node (settlements, fortresses, etc.)
	nodeID := ih.deps.Renderer.GetNodeAtPosition(inputState.MouseX, inputState.MouseY)
	if nodeID != 0 {
		ih.deps.State.SelectedNodeID = nodeID
		ih.deps.RefreshPanels()
		ih.deps.LogEvent(fmt.Sprintf("Selected node %d (Press G to garrison)", nodeID))
		return true
	}

	// Click on empty space clears selection
	ih.deps.State.ClearSelection()
	ih.deps.RefreshPanels()
	return false
}

// toggleMoveMode toggles movement overlay for the selected commander.
func (ih *OverworldInputHandler) toggleMoveMode() {
	if ih.deps.State.SelectedCommanderID == 0 {
		ih.deps.LogEvent("No commander selected - click a commander first")
		return
	}

	if ih.deps.State.InMoveMode {
		ih.deps.State.ExitMoveMode()
		ih.deps.LogEvent("Movement mode cancelled")
		return
	}

	// Calculate valid tiles
	if ih.deps.CommanderMovement != nil {
		tiles := ih.deps.CommanderMovement.GetValidMovementTiles(ih.deps.State.SelectedCommanderID)
		if len(tiles) == 0 {
			ih.deps.LogEvent("No movement remaining this turn")
			return
		}
		ih.deps.State.InMoveMode = true
		ih.deps.State.ValidMoveTiles = tiles
		ih.deps.LogEvent("Movement mode active - click a blue tile to move")
	}
}

// cycleCommander cycles SelectedCommanderID to the next commander in the roster.
func (ih *OverworldInputHandler) cycleCommander() {
	if ih.deps.PlayerData == nil {
		return
	}

	commanders := commander.GetAllCommanders(ih.deps.PlayerData.PlayerEntityID, ih.deps.Manager)
	if len(commanders) == 0 {
		return
	}

	// Exit move mode when switching
	ih.deps.State.ExitMoveMode()

	// Find current index
	currentIdx := -1
	for i, id := range commanders {
		if id == ih.deps.State.SelectedCommanderID {
			currentIdx = i
			break
		}
	}

	// Cycle to next
	nextIdx := (currentIdx + 1) % len(commanders)
	ih.deps.State.SelectedCommanderID = commanders[nextIdx]

	cmdData := commander.GetCommanderData(ih.deps.State.SelectedCommanderID, ih.deps.Manager)
	name := "Commander"
	if cmdData != nil {
		name = cmdData.Name
	}
	ih.deps.LogEvent(fmt.Sprintf("Selected %s (%d/%d)", name, nextIdx+1, len(commanders)))
	ih.deps.RefreshPanels()
}

// handleGarrison validates the selected node and opens the garrison dialog.
func (ih *OverworldInputHandler) handleGarrison() {
	if ih.deps.State.SelectedNodeID == 0 {
		ih.deps.LogEvent("No node selected - click a player node first")
		return
	}

	nodeID := ih.deps.State.SelectedNodeID
	nodeEntity := ih.deps.Manager.FindEntityByID(nodeID)
	if nodeEntity == nil {
		ih.deps.LogEvent("ERROR: Node entity not found")
		return
	}

	nodeData := common.GetComponentType[*core.OverworldNodeData](nodeEntity, core.OverworldNodeComponent)
	if nodeData == nil {
		ih.deps.LogEvent("ERROR: Selected entity is not an overworld node")
		return
	}

	if !core.IsFriendlyOwner(nodeData.OwnerID) {
		ih.deps.LogEvent(fmt.Sprintf("Can only garrison player-owned nodes (this node owned by: %s)", nodeData.OwnerID))
		return
	}

	commanderID := ih.deps.State.SelectedCommanderID
	if commanderID == 0 {
		ih.deps.LogEvent("No commander selected - select a commander first")
		return
	}

	ih.showGarrisonDialog(nodeID, commanderID)
}

// showGarrisonDialog builds and displays the garrison management dialog.
func (ih *OverworldInputHandler) showGarrisonDialog(nodeID ecs.EntityID, rosterOwnerID ecs.EntityID) {
	manager := ih.deps.Manager

	garrisonData := garrison.GetGarrisonAtNode(manager, nodeID)
	availableSquads := garrison.GetAvailableSquadsForGarrison(manager, rosterOwnerID)

	type garrisonEntry struct {
		SquadID      ecs.EntityID
		IsGarrisoned bool
	}
	var entries []garrisonEntry
	var entryLabels []string

	if garrisonData != nil {
		for _, squadID := range garrisonData.SquadIDs {
			squadName := squads.GetSquadName(squadID, manager)
			unitCount := len(squads.GetUnitIDsInSquad(squadID, manager))
			entries = append(entries, garrisonEntry{SquadID: squadID, IsGarrisoned: true})
			entryLabels = append(entryLabels, fmt.Sprintf("[Garrisoned] %s (%d units) - Click to REMOVE", squadName, unitCount))
		}
	}

	for _, squadID := range availableSquads {
		squadName := squads.GetSquadName(squadID, manager)
		unitCount := len(squads.GetUnitIDsInSquad(squadID, manager))
		entries = append(entries, garrisonEntry{SquadID: squadID, IsGarrisoned: false})
		entryLabels = append(entryLabels, fmt.Sprintf("[Available] %s (%d units) - Click to ASSIGN", squadName, unitCount))
	}

	if len(entries) == 0 {
		ih.deps.LogEvent("No squads available and no squads garrisoned")
		return
	}

	dialog := builders.CreateSelectionDialog(builders.SelectionDialogConfig{
		Title:            "Manage Garrison",
		Message:          "Select a squad to assign or remove from garrison:",
		SelectionEntries: entryLabels,
		MinWidth:         600,
		MinHeight:        400,
		OnSelect: func(selected string) {
			for i, label := range entryLabels {
				if label == selected {
					entry := entries[i]
					if entry.IsGarrisoned {
						if err := ih.actionHandler.RemoveSquadFromGarrison(entry.SquadID, nodeID); err != nil {
							ih.deps.LogEvent(fmt.Sprintf("ERROR: %v", err))
						}
					} else {
						if err := ih.actionHandler.AssignSquadToGarrison(entry.SquadID, nodeID); err != nil {
							ih.deps.LogEvent(fmt.Sprintf("ERROR: %v", err))
						}
					}
					return
				}
			}
		},
		OnCancel: func() {
			ih.deps.LogEvent("Garrison management cancelled")
		},
	})

	ih.ui.AddWindow(dialog)
}
