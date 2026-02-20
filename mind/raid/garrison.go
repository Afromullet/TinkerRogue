package raid

import (
	"fmt"

	"game_main/common"
	"game_main/tactical/squads"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
)

// GenerateGarrison creates a complete garrison defense as ECS entities across multiple floors.
// It builds the DAG for each floor, assigns archetypes to rooms, and instantiates garrison squads.
// Returns the raid state entity ID.
func GenerateGarrison(manager *common.EntityManager, floorCount int, commanderID ecs.EntityID, playerSquadIDs []ecs.EntityID) ecs.EntityID {
	// Create raid state entity
	raidEntity := manager.World.NewEntity()
	raidEntityID := raidEntity.GetID()

	raidState := &RaidStateData{
		CurrentFloor:   1,
		TotalFloors:    floorCount,
		Status:         RaidActive,
		CommanderID:    commanderID,
		PlayerSquadIDs: append([]ecs.EntityID{}, playerSquadIDs...),
	}
	raidEntity.AddComponent(RaidStateComponent, raidState)

	// Generate each floor
	for floor := 1; floor <= floorCount; floor++ {
		generateFloor(manager, floor)
	}

	fmt.Printf("GenerateGarrison: Created %d-floor garrison (raid entity %d)\n", floorCount, raidEntityID)
	return raidEntityID
}

// generateFloor builds one garrison floor: DAG, rooms, alert, and garrison squads.
func generateFloor(manager *common.EntityManager, floorNumber int) {
	// Build abstract DAG from worldmap package
	dag := worldmap.BuildGarrisonDAG(floorNumber)

	// Create alert data entity for this floor
	alertEntity := manager.World.NewEntity()
	alertEntity.AddComponent(AlertDataComponent, &AlertData{
		FloorNumber:    floorNumber,
		CurrentLevel:   0,
		EncounterCount: 0,
	})

	// Create room data entities from DAG
	buildFloorGraph(manager, dag, floorNumber)
	roomCount := len(dag.Nodes)

	// Assign archetypes to rooms
	assignments := AssignArchetypesToFloor(dag, floorNumber)

	// Instantiate garrison squads and link to rooms
	var garrisonSquadIDs []ecs.EntityID
	var reserveSquadIDs []ecs.EntityID

	for nodeID, archetypeName := range assignments {
		squadID := InstantiateGarrisonSquad(manager, GetArchetype(archetypeName), floorNumber, nodeID, false)
		if squadID != 0 {
			garrisonSquadIDs = append(garrisonSquadIDs, squadID)

			// Link squad to room
			roomData := GetRoomData(manager, nodeID, floorNumber)
			if roomData != nil {
				roomData.GarrisonSquadIDs = append(roomData.GarrisonSquadIDs, squadID)
			}
		}
	}

	// Create reserve squads per floor (not assigned to rooms)
	reserveArchetypes := ReserveArchetypes()

	reserveCount := ReserveCountForFloor(floorNumber)
	for i := 0; i < reserveCount; i++ {
		archName := reserveArchetypes[common.RandomInt(len(reserveArchetypes))]
		archetype := GetArchetype(archName)
		if archetype != nil {
			squadID := InstantiateGarrisonSquad(manager, archetype, floorNumber, -1, true)
			if squadID != 0 {
				reserveSquadIDs = append(reserveSquadIDs, squadID)
			}
		}
	}

	// Create floor state entity
	floorEntity := manager.World.NewEntity()
	floorEntity.AddComponent(FloorStateComponent, &FloorStateData{
		FloorNumber:      floorNumber,
		RoomsCleared:     0,
		RoomsTotal:       roomCount,
		GarrisonSquadIDs: garrisonSquadIDs,
		ReserveSquadIDs:  reserveSquadIDs,
		IsComplete:       false,
	})

	fmt.Printf("  Floor %d: %d rooms, %d garrison squads, %d reserves\n",
		floorNumber, roomCount, len(garrisonSquadIDs), len(reserveSquadIDs))
}

// InstantiateGarrisonSquad creates a garrison squad from an archetype definition.
// Uses squads.CreateSquadFromTemplate with a dummy position (garrison squads are placed when combat starts).
// Returns the squad entity ID, or 0 if the archetype is nil or has no valid units.
func InstantiateGarrisonSquad(manager *common.EntityManager, archetype *SquadArchetype, floorNumber, roomNodeID int, isReserve bool) ecs.EntityID {
	if archetype == nil {
		return 0
	}

	// Build unit templates from archetype
	var unitTemplates []squads.UnitTemplate
	for _, au := range archetype.Units {
		template := squads.GetTemplateByUnitType(au.MonsterType)
		if template == nil {
			fmt.Printf("WARNING: Monster template '%s' not found for archetype '%s'\n", au.MonsterType, archetype.Name)
			continue
		}

		// Clone the template and override grid position
		ut := *template
		ut.GridRow = au.GridRow
		ut.GridCol = au.GridCol
		if au.GridWidth > 0 {
			ut.GridWidth = au.GridWidth
		}
		if au.GridHeight > 0 {
			ut.GridHeight = au.GridHeight
		}
		ut.IsLeader = au.IsLeader
		unitTemplates = append(unitTemplates, ut)
	}

	if len(unitTemplates) == 0 {
		fmt.Printf("WARNING: No valid units for archetype '%s'\n", archetype.Name)
		return 0
	}

	// Use dummy position — garrison squads get placed when combat starts
	dummyPos := coords.LogicalPosition{X: 0, Y: 0}
	displayName := fmt.Sprintf("%s (F%d)", archetype.DisplayName, floorNumber)

	squadID := squads.CreateSquadFromTemplate(manager, displayName, squads.FormationBalanced, dummyPos, unitTemplates)

	// Add garrison squad component
	squadEntity := manager.FindEntityByID(squadID)
	if squadEntity != nil {
		squadEntity.AddComponent(GarrisonSquadComponent, &GarrisonSquadData{
			ArchetypeName: archetype.Name,
			FloorNumber:   floorNumber,
			RoomNodeID:    roomNodeID,
			IsReserve:     isReserve,
			IsDestroyed:   false,
		})
	}

	return squadID
}
