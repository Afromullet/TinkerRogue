package guioverworld

import (
	"fmt"

	"game_main/common"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/overworld/travel"
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
	// ESC - context switch back to battle map
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		if ih.deps.ModeCoordinator != nil {
			if err := ih.deps.ModeCoordinator.EnterBattleMap("exploration"); err != nil {
				fmt.Printf("ERROR: Failed to return to battle map: %v\n", err)
			}
			return true
		}
	}

	// N - enter node placement mode
	if inputState.KeysJustPressed[ebiten.KeyN] {
		ih.deps.ModeManager.SetMode("node_placement")
		return true
	}

	// Space - advance tick
	if inputState.KeysJustPressed[ebiten.KeySpace] {
		ih.actionHandler.AdvanceTick()
		return true
	}

	// A - toggle auto-travel
	if inputState.KeysJustPressed[ebiten.KeyA] {
		ih.actionHandler.ToggleAutoTravel()
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

	// E - engage selected threat
	if inputState.KeysJustPressed[ebiten.KeyE] {
		if ih.deps.State.HasSelection() {
			ih.actionHandler.EngageThreat(ih.deps.State.SelectedNodeID)
			return true
		}
	}

	// C - cancel travel
	if inputState.KeysJustPressed[ebiten.KeyC] {
		if travel.IsTraveling(ih.deps.Manager) {
			ih.actionHandler.CancelTravel()
			return true
		}
	}

	// Movement keys advance time (W/S/D/Q/Z)
	if inputState.KeysJustPressed[ebiten.KeyW] ||
		inputState.KeysJustPressed[ebiten.KeyS] ||
		inputState.KeysJustPressed[ebiten.KeyD] ||
		inputState.KeysJustPressed[ebiten.KeyQ] ||
		inputState.KeysJustPressed[ebiten.KeyZ] {
		ih.actionHandler.AdvanceTick()
		return true
	}

	// Mouse click - node selection
	if inputState.MousePressed && inputState.MouseButton == ebiten.MouseButtonLeft {
		return ih.handleMouseClick(inputState)
	}

	return false
}

// handleMouseClick handles node selection via mouse.
func (ih *OverworldInputHandler) handleMouseClick(inputState *framework.InputState) bool {
	// Check for threat first
	threatID := ih.deps.Renderer.GetThreatAtPosition(inputState.MouseX, inputState.MouseY)
	if threatID != 0 {
		ih.deps.State.SelectedNodeID = threatID
		ih.deps.RefreshPanels()
		ih.deps.LogEvent(fmt.Sprintf("Selected threat %d (Press E to engage)", threatID))
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

	playerEntityID := ecs.EntityID(0)
	if ih.deps.PlayerData != nil {
		playerEntityID = ih.deps.PlayerData.PlayerEntityID
	}

	ih.showGarrisonDialog(nodeID, playerEntityID)
}

// showGarrisonDialog builds and displays the garrison management dialog.
func (ih *OverworldInputHandler) showGarrisonDialog(nodeID ecs.EntityID, playerEntityID ecs.EntityID) {
	manager := ih.deps.Manager

	garrisonData := garrison.GetGarrisonAtNode(manager, nodeID)
	availableSquads := garrison.GetAvailableSquadsForGarrison(manager, playerEntityID)

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
