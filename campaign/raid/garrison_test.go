package raid

import (
	"testing"

	"game_main/core/common"
	"game_main/core/coords"
	"game_main/mind/encounter"
	"game_main/tactical/squads/squadcore"
	testfx "game_main/testing"

	"github.com/bytearena/ecs"
)

// newTestSquad creates a squad entity plus unitCount member units, all positioned
// at pos and registered with the spatial grid. It mirrors the shape of what
// squadcore.CreateSquadFromTemplate produces for the fields disposal cares about,
// without needing monster templates loaded from gamedata.
func newTestSquad(manager *common.EntityManager, name string, pos coords.LogicalPosition, unitCount int) (ecs.EntityID, []ecs.EntityID) {
	squadEntity := manager.World.NewEntity()
	squadID := squadEntity.GetID()
	squadEntity.AddComponent(squadcore.SquadComponent, &squadcore.SquadData{
		SquadID:  squadID,
		Name:     name,
		MaxUnits: 9,
	})
	squadPos := pos
	squadEntity.AddComponent(common.PositionComponent, &squadPos)
	common.GlobalPositionSystem.AddEntity(squadID, squadPos)

	unitIDs := make([]ecs.EntityID, 0, unitCount)
	for i := 0; i < unitCount; i++ {
		unitEntity := manager.World.NewEntity()
		unitID := unitEntity.GetID()
		unitEntity.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: squadID})
		unitPos := pos
		unitEntity.AddComponent(common.PositionComponent, &unitPos)
		common.GlobalPositionSystem.AddEntity(unitID, unitPos)
		unitIDs = append(unitIDs, unitID)
	}

	return squadID, unitIDs
}

// markAsGarrison tags an existing squad as part of the garrison defense.
func markAsGarrison(manager *common.EntityManager, squadID ecs.EntityID, floorNumber, roomNodeID int, isReserve bool) {
	manager.FindEntityByID(squadID).AddComponent(GarrisonSquadComponent, &GarrisonSquadData{
		ArchetypeName: "test_archetype",
		FloorNumber:   floorNumber,
		RoomNodeID:    roomNodeID,
		IsReserve:     isReserve,
	})
}

// testRaid is the entity graph a generated garrison leaves in the world.
type testRaid struct {
	playerSquadID   ecs.EntityID
	playerUnitIDs   []ecs.EntityID
	garrisonSquadID ecs.EntityID
	garrisonUnitIDs []ecs.EntityID
	reserveSquadID  ecs.EntityID
	reserveUnitIDs  []ecs.EntityID
}

// buildTestRaid stands up a one-floor raid: a raid state, a floor state, a room,
// an alert, a deployment, one room garrison squad, one reserve squad, and a
// player squad that must survive disposal.
func buildTestRaid(manager *common.EntityManager) *testRaid {
	playerSquadID, playerUnitIDs := newTestSquad(manager, "Player Squad", coords.LogicalPosition{X: 1, Y: 1}, 2)
	garrisonSquadID, garrisonUnitIDs := newTestSquad(manager, "Garrison Squad", coords.LogicalPosition{X: 5, Y: 5}, 3)
	reserveSquadID, reserveUnitIDs := newTestSquad(manager, "Reserve Squad", coords.LogicalPosition{X: 0, Y: 0}, 2)

	markAsGarrison(manager, garrisonSquadID, 1, 2, false)
	markAsGarrison(manager, reserveSquadID, 1, -1, true)

	raidEntity := manager.World.NewEntity()
	raidEntity.AddComponent(RaidStateComponent, &RaidStateData{
		CurrentFloor:   1,
		TotalFloors:    1,
		Status:         RaidActive,
		PlayerSquadIDs: []ecs.EntityID{playerSquadID},
	})

	floorEntity := manager.World.NewEntity()
	floorEntity.AddComponent(FloorStateComponent, &FloorStateData{
		FloorNumber:      1,
		RoomsTotal:       1,
		GarrisonSquadIDs: []ecs.EntityID{garrisonSquadID},
		ReserveSquadIDs:  []ecs.EntityID{reserveSquadID},
	})

	roomEntity := manager.World.NewEntity()
	roomEntity.AddComponent(RoomDataComponent, &RoomData{
		NodeID:           2,
		RoomType:         "combat",
		FloorNumber:      1,
		IsAccessible:     true,
		GarrisonSquadIDs: []ecs.EntityID{garrisonSquadID},
	})

	alertEntity := manager.World.NewEntity()
	alertEntity.AddComponent(AlertDataComponent, &AlertData{FloorNumber: 1, CurrentLevel: 2})

	deployEntity := manager.World.NewEntity()
	deployEntity.AddComponent(DeploymentComponent, &DeploymentData{
		DeployedSquadIDs: []ecs.EntityID{playerSquadID},
	})

	return &testRaid{
		playerSquadID:   playerSquadID,
		playerUnitIDs:   playerUnitIDs,
		garrisonSquadID: garrisonSquadID,
		garrisonUnitIDs: garrisonUnitIDs,
		reserveSquadID:  reserveSquadID,
		reserveUnitIDs:  reserveUnitIDs,
	}
}

// assertDisposed fails if any of the entities still exist in the world.
func assertDisposed(t *testing.T, manager *common.EntityManager, label string, ids []ecs.EntityID) {
	t.Helper()
	for _, id := range ids {
		if manager.FindEntityByID(id) != nil {
			t.Errorf("%s: entity %d still in the world after disposal", label, id)
		}
	}
}

// assertAlive fails if any of the entities have been removed from the world.
func assertAlive(t *testing.T, manager *common.EntityManager, label string, ids []ecs.EntityID) {
	t.Helper()
	for _, id := range ids {
		if manager.FindEntityByID(id) == nil {
			t.Errorf("%s: entity %d was disposed but should have survived", label, id)
		}
	}
}

// assertNoRaidTagsRemain fails if any raid-owned tag still matches an entity.
func assertNoRaidTagsRemain(t *testing.T, manager *common.EntityManager) {
	t.Helper()
	tags := map[string]ecs.Tag{
		"raid state":     RaidStateTag,
		"floor state":    FloorStateTag,
		"room data":      RoomDataTag,
		"alert data":     AlertDataTag,
		"deployment":     DeploymentTag,
		"garrison squad": GarrisonSquadTag,
	}
	for label, tag := range tags {
		if got := len(manager.World.Query(tag)); got != 0 {
			t.Errorf("want 0 %s entities after disposal, got %d", label, got)
		}
	}
}

func TestDisposeRaidEntities_RemovesGarrisonLeavesPlayerSquads(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	tr := buildTestRaid(manager)

	DisposeRaidEntities(manager)

	assertNoRaidTagsRemain(t, manager)
	assertDisposed(t, manager, "garrison squad", []ecs.EntityID{tr.garrisonSquadID})
	assertDisposed(t, manager, "garrison units", tr.garrisonUnitIDs)
	assertDisposed(t, manager, "reserve squad", []ecs.EntityID{tr.reserveSquadID})
	assertDisposed(t, manager, "reserve units", tr.reserveUnitIDs)

	// Player squads belong to the commander's roster and outlive the raid.
	assertAlive(t, manager, "player squad", []ecs.EntityID{tr.playerSquadID})
	assertAlive(t, manager, "player units", tr.playerUnitIDs)
	if got := len(manager.World.Query(squadcore.SquadTag)); got != 1 {
		t.Errorf("want only the player squad left, got %d squads", got)
	}
}

func TestDisposeRaidEntities_UnregistersFromPositionSystem(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	tr := buildTestRaid(manager)

	garrisonPos := coords.LogicalPosition{X: 5, Y: 5}
	if got := len(common.GlobalPositionSystem.GetAllEntityIDsAt(garrisonPos)); got == 0 {
		t.Fatalf("test setup: garrison entities were never registered at %v", garrisonPos)
	}

	DisposeRaidEntities(manager)

	if got := common.GlobalPositionSystem.GetAllEntityIDsAt(garrisonPos); len(got) != 0 {
		t.Errorf("garrison entities still in the spatial grid at %v: %v", garrisonPos, got)
	}

	// The player squad stays registered — disposal must not touch the grid entries
	// of entities it does not own.
	playerPos := coords.LogicalPosition{X: 1, Y: 1}
	if got := len(common.GlobalPositionSystem.GetAllEntityIDsAt(playerPos)); got != len(tr.playerUnitIDs)+1 {
		t.Errorf("want player squad + %d units at %v, got %d entries", len(tr.playerUnitIDs), playerPos, got)
	}
}

// Combat teardown disposes the garrison squads of the room that was just fought
// before the post-combat callback reaches finishRaid, so disposal has to tolerate
// squads and units that are already gone.
func TestDisposeRaidEntities_ToleratesSquadsDisposedByCombatTeardown(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	tr := buildTestRaid(manager)

	squadcore.DisposeSquadAndUnits(tr.garrisonSquadID, manager)

	DisposeRaidEntities(manager)

	assertNoRaidTagsRemain(t, manager)
	assertDisposed(t, manager, "reserve squad", []ecs.EntityID{tr.reserveSquadID})
	assertDisposed(t, manager, "reserve units", tr.reserveUnitIDs)
	assertAlive(t, manager, "player squad", []ecs.EntityID{tr.playerSquadID})
}

func TestDisposeRaidEntities_NoRaidInProgressIsANoop(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	playerSquadID, playerUnitIDs := newTestSquad(manager, "Player Squad", coords.LogicalPosition{X: 1, Y: 1}, 2)

	DisposeRaidEntities(manager)

	assertAlive(t, manager, "player squad", []ecs.EntityID{playerSquadID})
	assertAlive(t, manager, "player units", playerUnitIDs)
}

func TestFinishRaid_DisposesEntitiesAndResetsRunner(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	tr := buildTestRaid(manager)

	runner := NewRaidRunner(manager, encounter.NewEncounterService(manager, nil))
	runner.raidEntityID = manager.World.Query(RaidStateTag)[0].Entity.GetID()
	if !runner.IsActive() {
		t.Fatal("test setup: runner should report an active raid")
	}

	runner.finishRaid(RaidVictory)

	if runner.IsActive() {
		t.Error("runner still reports an active raid after finishRaid")
	}
	assertNoRaidTagsRemain(t, manager)
	assertDisposed(t, manager, "garrison units", tr.garrisonUnitIDs)
	assertAlive(t, manager, "player squad", []ecs.EntityID{tr.playerSquadID})
}
