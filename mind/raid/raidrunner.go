package raid

import (
	"fmt"

	"game_main/common"
	"game_main/mind/encounter"
	"game_main/mind/combatpipeline"
	"game_main/world/worldmap"

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
// encounter triggering, and post-combat combatpipeline.
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

	// Register as post-combat listener
	encounterService.PostCombatCallback = func(reason encounter.CombatExitReason, result *encounter.CombatResult) {
		if rr.raidEntityID != 0 {
			rr.ResolveEncounter(reason, result)
		}
	}

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

	fmt.Printf("RaidRunner: Raid started with %d squads across %d floors\n", len(playerSquadIDs), floorCount)
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

	fmt.Printf("RaidRunner: Entering floor %d (%d rooms)\n", floorNumber, floorState.RoomsTotal)
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
	case worldmap.GarrisonRoomRestRoom:
		rr.processRestRoom(raidState, room)
		return nil
	case worldmap.GarrisonRoomStairs:
		rr.processStairsRoom(raidState, room)
		return nil
	}

	fmt.Printf("RaidRunner: Selected room %d (%s) on floor %d\n",
		nodeID, room.RoomType, raidState.CurrentFloor)

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
		rr.preCombatAliveCounts[squadID] = combatpipeline.CountLivingUnitsInSquad(rr.manager, squadID)
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
	_, err := combatpipeline.ExecuteCombatStart(rr.encounterService, rr.manager, starter)
	return err
}

// processRestRoom applies rest room recovery and marks the room cleared.
func (rr *RaidRunner) processRestRoom(raidState *RaidStateData, room *RoomData) {
	// Apply rest room HP recovery from config
	if RaidConfig != nil {
		for _, squadID := range raidState.PlayerSquadIDs {
			combatpipeline.ApplyHPRecovery(rr.manager, squadID, RaidConfig.Recovery.RestRoomHPPercent)
		}
	}

	MarkRoomCleared(rr.manager, room.NodeID, room.FloorNumber)
	fmt.Printf("RaidRunner: Rest room %d cleared, recovery applied\n", room.NodeID)
}

// processStairsRoom marks stairs cleared and advances floor.
func (rr *RaidRunner) processStairsRoom(raidState *RaidStateData, room *RoomData) {
	MarkRoomCleared(rr.manager, room.NodeID, room.FloorNumber)

	floorState := GetFloorState(rr.manager, room.FloorNumber)
	if floorState != nil {
		floorState.IsComplete = true
	}

	fmt.Printf("RaidRunner: Stairs room cleared on floor %d\n", room.FloorNumber)
}

// ResolveEncounter processes the result of a completed combat encounter.
// Called via PostCombatCallback from EncounterService.
func (rr *RaidRunner) ResolveEncounter(reason encounter.CombatExitReason, result *encounter.CombatResult) {
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
				post := combatpipeline.CountLivingUnitsInSquad(rr.manager, squadID)
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
	case encounter.ExitVictory:
		resolver := &RaidRoomResolver{RaidState: raidState, RoomNodeID: rr.currentRoomNodeID}
		result := combatpipeline.ExecuteResolution(rr.manager, resolver)
		if result != nil {
			rewardText = result.RewardText
		}
	case encounter.ExitDefeat, encounter.ExitFlee:
		resolver := &RaidDefeatResolver{}
		combatpipeline.ExecuteResolution(rr.manager, resolver)
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
		IsVictory:  reason == encounter.ExitVictory,
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
	fmt.Println("RaidRunner: Player retreated from raid (state preserved)")
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
	fmt.Printf("RaidRunner: Raid finished with status: %s\n", status)

	// Clear callback to avoid stale references
	rr.encounterService.PostCombatCallback = nil
	rr.raidEntityID = 0
}
