package raid

import (
	"fmt"

	"game_main/core/common"
	"game_main/mind/combatlifecycle"
	"game_main/mind/encounter"
	"game_main/tactical/squads/squadcore"
	"game_main/world/garrisongen"

	"github.com/bytearena/ecs"
)

// RaidEncounterResult holds the outcome of a raid encounter for GUI display.
type RaidEncounterResult struct {
	RoomName   string
	RoomType   string
	UnitsLost  int
	AlertLevel int
	RewardText string
	IsVictory  bool
}

// RaidRunner coordinates the raid loop: floor progression, room selection,
// encounter triggering, and post-combat combatlifecycle.
// It is NOT an ECS system — it's a service/controller that orchestrates ECS state.
type RaidRunner struct {
	manager          *common.EntityManager
	encounterService *encounter.EncounterService
	raidEntityID     ecs.EntityID
	// preCombatAliveCounts stores living unit counts per player squad before an encounter.
	// Used to calculate units lost for the post-encounter summary display.
	preCombatAliveCounts map[ecs.EntityID]int

	// currentRoomNodeID tracks which room is being fought in (set by TriggerRaidEncounter).
	currentRoomNodeID int

	// LastEncounterResult holds the outcome of the most recent encounter for GUI display.
	// Read by RaidMode when entering after combat to build the summary panel.
	LastEncounterResult *RaidEncounterResult
}

// NewRaidRunner creates a new RaidRunner.
func NewRaidRunner(manager *common.EntityManager, encounterService *encounter.EncounterService) *RaidRunner {
	rr := &RaidRunner{
		manager:          manager,
		encounterService: encounterService,
	}

	// Register as post-combat listener.
	// Guard: only process results when raid is active AND not retreated.
	// Without this guard, retreating from a raid and then triggering an overworld
	// encounter would cause the listener to process the overworld result as a raid.
	encounterService.SetPostCombatCallback(func(reason combatlifecycle.CombatExitReason, result *combatlifecycle.EncounterOutcome) {
		raidState := GetRaidState(rr.manager)
		if rr.raidEntityID != 0 && raidState != nil && raidState.Status == RaidActive {
			rr.ResolveEncounter(reason, result)
		}
	})

	return rr
}

// StartRaid generates the garrison and initializes the raid state.
func (rr *RaidRunner) StartRaid(commanderID ecs.EntityID, playerEntityID ecs.EntityID, playerSquadIDs []ecs.EntityID, floorCount int) error {
	if rr.raidEntityID != 0 {
		return fmt.Errorf("raid already in progress")
	}

	if len(playerSquadIDs) == 0 {
		return fmt.Errorf("no player squads provided")
	}

	maxSquads := MaxPlayerSquads()
	if len(playerSquadIDs) > maxSquads {
		return fmt.Errorf("too many squads: %d (max %d)", len(playerSquadIDs), maxSquads)
	}

	rr.raidEntityID = GenerateGarrison(rr.manager, floorCount, commanderID, playerEntityID, playerSquadIDs)

	return nil
}

// EnterFloor transitions to a new floor and resets floor-specific state.
func (rr *RaidRunner) EnterFloor(floorNumber int) error {
	raidState := GetRaidState(rr.manager)
	if raidState == nil {
		return fmt.Errorf("no active raid")
	}

	if floorNumber < 1 || floorNumber > raidState.TotalFloors {
		return fmt.Errorf("invalid floor number: %d (total: %d)", floorNumber, raidState.TotalFloors)
	}

	raidState.CurrentFloor = floorNumber

	floorState := GetFloorState(rr.manager, floorNumber)
	if floorState == nil {
		return fmt.Errorf("floor state not found for floor %d", floorNumber)
	}

	return nil
}

// SelectRoom validates a room selection and handles non-combat rooms directly.
// For combat rooms, call TriggerRaidEncounter after deployment is confirmed.
func (rr *RaidRunner) SelectRoom(nodeID int) error {
	raidState := GetRaidState(rr.manager)
	if raidState == nil {
		return fmt.Errorf("no active raid")
	}

	room := GetRoomData(rr.manager, nodeID, raidState.CurrentFloor)
	if room == nil {
		return fmt.Errorf("room %d not found on floor %d", nodeID, raidState.CurrentFloor)
	}

	if !room.IsAccessible {
		return fmt.Errorf("room %d is not accessible", nodeID)
	}

	if room.IsCleared {
		return fmt.Errorf("room %d is already cleared", nodeID)
	}

	// Handle non-combat rooms
	switch room.RoomType {
	case garrisongen.GarrisonRoomRestRoom:
		rr.processRestRoom(raidState, room)
		return nil
	case garrisongen.GarrisonRoomStairs:
		rr.processStairsRoom(raidState, room)
		return nil
	}

	return nil
}

// TriggerRaidEncounter sets up combat for a garrison room and transitions to combat mode.
// Call this after deployment is confirmed (via OnDeployConfirmed).
func (rr *RaidRunner) TriggerRaidEncounter(nodeID int) error {
	raidState := GetRaidState(rr.manager)
	if raidState == nil {
		return fmt.Errorf("no active raid")
	}

	room := GetRoomData(rr.manager, nodeID, raidState.CurrentFloor)
	if room == nil {
		return fmt.Errorf("room %d not found", nodeID)
	}

	if len(room.GarrisonSquadIDs) == 0 {
		return fmt.Errorf("room %d has no garrison squads", nodeID)
	}

	// Snapshot alive counts for post-encounter summary
	rr.preCombatAliveCounts = make(map[ecs.EntityID]int)
	for _, squadID := range raidState.PlayerSquadIDs {
		rr.preCombatAliveCounts[squadID] = squadcore.CountLivingUnitsInSquad(rr.manager, squadID)
	}

	// Get deployed squads (use deployment if available, otherwise all player squads)
	deployedIDs := raidState.PlayerSquadIDs
	deployment := GetDeployment(rr.manager)
	if deployment != nil && len(deployment.DeployedSquadIDs) > 0 {
		deployedIDs = deployment.DeployedSquadIDs
	}

	// Combat position from config
	combatPos := CombatPosition()

	// Store current room for post-combat resolution
	rr.currentRoomNodeID = nodeID
	rr.LastEncounterResult = nil

	// Use unified combat start pipeline
	starter := &RaidCombatStarter{
		RaidEntityID:     rr.raidEntityID,
		GarrisonSquadIDs: room.GarrisonSquadIDs,
		DeployedSquadIDs: deployedIDs,
		CombatPos:        combatPos,
		CommanderID:      raidState.CommanderID,
	}
	return combatlifecycle.ExecuteCombatStart(rr.encounterService, rr.manager, starter)
}

// processRestRoom applies rest room recovery and marks the room cleared.
func (rr *RaidRunner) processRestRoom(raidState *RaidStateData, room *RoomData) {
	// Apply rest room HP recovery from config
	if RaidConfig != nil {
		for _, squadID := range raidState.PlayerSquadIDs {
			combatlifecycle.ApplyHPRecovery(rr.manager, squadID, RaidConfig.Recovery.RestRoomHPPercent)
		}
	}

	MarkRoomCleared(rr.manager, room.NodeID, room.FloorNumber)
}

// processStairsRoom marks stairs cleared and advances floor.
func (rr *RaidRunner) processStairsRoom(raidState *RaidStateData, room *RoomData) {
	MarkRoomCleared(rr.manager, room.NodeID, room.FloorNumber)

	floorState := GetFloorState(rr.manager, room.FloorNumber)
	if floorState != nil {
		floorState.IsComplete = true
	}

}

// ResolveEncounter processes the result of a completed combat encounter.
// Called via PostCombatCallback from EncounterService.
func (rr *RaidRunner) ResolveEncounter(reason combatlifecycle.CombatExitReason, result *combatlifecycle.EncounterOutcome) {
	raidState := GetRaidState(rr.manager)
	if raidState == nil {
		return
	}

	// Calculate units lost before processing (for summary)
	unitsLostTotal := 0
	if rr.preCombatAliveCounts != nil {
		for _, squadID := range raidState.PlayerSquadIDs {
			pre, ok := rr.preCombatAliveCounts[squadID]
			if ok {
				post := squadcore.CountLivingUnitsInSquad(rr.manager, squadID)
				if pre > post {
					unitsLostTotal += pre - post
				}
			}
		}
	}

	// Get room info for summary
	room := GetRoomData(rr.manager, rr.currentRoomNodeID, raidState.CurrentFloor)
	roomName := fmt.Sprintf("Room %d", rr.currentRoomNodeID)
	roomType := "unknown"
	if room != nil {
		roomType = room.RoomType
	}

	var rewardText string
	switch reason {
	case combatlifecycle.ExitVictory:
		resolver := &RaidRoomResolver{RaidState: raidState, RoomNodeID: rr.currentRoomNodeID}
		result := combatlifecycle.ExecuteResolution(rr.manager, resolver)
		if result != nil {
			rewardText = result.RewardText
		}
	case combatlifecycle.ExitDefeat, combatlifecycle.ExitFlee:
		resolver := &RaidDefeatResolver{}
		combatlifecycle.ExecuteResolution(rr.manager, resolver)
	}

	rr.PostEncounterProcessing()

	// Build encounter result for GUI display
	alertLevel := 0
	alertData := GetAlertData(rr.manager, raidState.CurrentFloor)
	if alertData != nil {
		alertLevel = alertData.CurrentLevel
	}

	rr.LastEncounterResult = &RaidEncounterResult{
		RoomName:   roomName,
		RoomType:   roomType,
		UnitsLost:  unitsLostTotal,
		AlertLevel: alertLevel,
		RewardText: rewardText,
		IsVictory:  reason == combatlifecycle.ExitVictory,
	}
}

// PostEncounterProcessing runs after encounter resolution:
// increments alert, checks end conditions.
func (rr *RaidRunner) PostEncounterProcessing() {
	raidState := GetRaidState(rr.manager)
	if raidState == nil {
		rr.finishRaid(RaidDefeat)
		return
	}
	if raidState.Status != RaidActive {
		rr.finishRaid(raidState.Status)
		return
	}

	// Post-encounter recovery (deployed vs reserve differentiation)
	ApplyPostEncounterRecovery(rr.manager, raidState)

	// Increment alert level and potentially activate reserves
	IncrementAlert(rr.manager, raidState.CurrentFloor)

	// Check end conditions
	status := CheckRaidEndConditions(rr.manager)
	if status != RaidActive {
		raidState.Status = status
		rr.finishRaid(status)
	}
}

// AdvanceFloor moves to the next floor and applies between-floor recovery.
func (rr *RaidRunner) AdvanceFloor() error {
	raidState := GetRaidState(rr.manager)
	if raidState == nil {
		return fmt.Errorf("no active raid")
	}

	nextFloor := raidState.CurrentFloor + 1
	if nextFloor > raidState.TotalFloors {
		// All floors cleared — victory
		raidState.Status = RaidVictory
		rr.finishRaid(RaidVictory)
		return nil
	}

	// Apply between-floor recovery
	ApplyBetweenFloorRecovery(rr.manager, raidState)

	return rr.EnterFloor(nextFloor)
}

// Retreat ends the raid with Retreated status.
// State is preserved so the player can resume the raid later.
func (rr *RaidRunner) Retreat() error {
	raidState := GetRaidState(rr.manager)
	if raidState == nil {
		return fmt.Errorf("no active raid")
	}

	raidState.Status = RaidRetreated
	return nil
}

// IsActive returns true if a raid is currently in progress.
func (rr *RaidRunner) IsActive() bool {
	return rr.raidEntityID != 0
}

// RestoreFromSave sets the raid entity ID from a loaded save,
// allowing the runner to resume an in-progress raid without generating a new garrison.
func (rr *RaidRunner) RestoreFromSave(raidEntityID ecs.EntityID) {
	rr.raidEntityID = raidEntityID
}

// finishRaid clears the runner state after the raid ends.
func (rr *RaidRunner) finishRaid(status RaidStatus) {
	// Clear callback to avoid stale references
	rr.encounterService.ClearPostCombatCallback()
	rr.raidEntityID = 0
}
